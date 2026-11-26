// Package web serves the learning application as a web UI. Each browser
// connection gets its own Session, which drives the very same interactive
// flows (app.RunLearningHiragana, RunLearningPhrases, ...) that the console
// build uses: the session plugs itself into app.App as the key Listener and
// the output writer, so no flow logic is duplicated for the web.
package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"atomicgo.dev/keyboard/keys"

	"test/internal/app"
	"test/internal/data"
	"test/internal/kbd"
)

// wsMessage is one JSON frame exchanged with the browser.
//
// Server -> client: {"type":"render","html":...} or {"type":"closed"}.
// Client -> server: {"type":"key","code":"Enter","rune":"a"} — exactly one
// keystroke per message; "code" carries special keys ("Enter", "Escape",
// "Backspace") and "rune" carries typed characters.
type wsMessage struct {
	Type    string `json:"type"`
	HTML    string `json:"html,omitempty"`
	Code    string `json:"code,omitempty"`
	Rune    string `json:"rune,omitempty"`
	Message string `json:"message,omitempty"`
}

// conn abstracts the transport used to talk to the browser. The real
// implementation wraps a websocket connection; tests inject a fake one.
type conn interface {
	read() (wsMessage, bool)
	send(m wsMessage) error
	closeWrite() error
}

// hub tracks all live sessions so their goroutines can be released when the
// server shuts down.
type hub struct {
	mu       sync.Mutex
	sessions map[*Session]struct{}
	closed   bool
}

func newHub() *hub {
	return &hub{sessions: map[*Session]struct{}{}}
}

func (h *hub) register(s *Session) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		go s.Close()
		return
	}
	h.sessions[s] = struct{}{}
}

func (h *hub) unregister(s *Session) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.sessions, s)
}

func (h *hub) closeAll() {
	h.mu.Lock()
	list := make([]*Session, 0, len(h.sessions))
	for s := range h.sessions {
		list = append(list, s)
	}
	h.sessions = map[*Session]struct{}{}
	h.closed = true
	h.mu.Unlock()
	for _, s := range list {
		s.Close()
	}
}

// Session is one browser tab running the application.
type Session struct {
	c   conn
	hub *hub
	app *app.App

	quit     chan struct{}
	quitOnce sync.Once
	waiter   kbd.Waitable // current inner listener while a flow runs
	pumpDone chan struct{} // closed when the render pump goroutine exits

	mu        sync.Mutex
	buf       bytes.Buffer // pending display output (accumulates until rendered)
	lastFrame string       // last HTML sent to the browser (for dedup)
	dirty     bool         // buffer changed since the last render
	keys      chan keys.Key
	keyClosed bool
	exhausted bool // key source ran dry (browser gone / session closing)
	closed    bool
}

// NewSession wires an app.App around the given connection. It is separated
// from the HTTP handler so tests can create sessions over a fake transport.
func NewSession(c conn, h *hub) *Session {
	s := &Session{
		c:        c,
		hub:      h,
		quit:     make(chan struct{}),
		pumpDone: make(chan struct{}),
		keys:     make(chan keys.Key),
	}

	a := app.New()
	a.Out = s
	a.Listener = s
	a.NavigateTo = navigateTo(a)
	// The web UI never exits the process and does not clear the real
	// terminal: Clean only resets the display buffer (handled inside the
	// session), Exit just ends the current flow.
	a.Exit = func(int) {}

	s.app = a
	if h != nil {
		h.register(s)
	}
	return s
}

// navigateTo dispatches a named destination to its flow, mirroring the
// console dispatcher in cmd/japan/main.go. DestQuit simply ends the current
// flow; the main-menu loop then notices the closed session and returns.
func navigateTo(a *app.App) func(name string) {
	return func(name string) {
		switch name {
		case app.DestPhrases:
			a.RunLearningPhrases(data.AllPhrases())
		case app.DestHiragana:
			a.RunLearningHiragana(data.Hiragana)
		case app.DestKatakana:
			a.RunLearningKatakana(data.Katakana)
		case app.DestResetHiragana:
			a.RunResetHiraganaProgress()
		case app.DestResetKatakana:
			a.RunResetKatakanaProgress()
		case app.DestResetPhrases:
			a.RunResetPhrasesProgress()
		case app.DestQuit:
			// Ending the current flow returns control to the menu loop,
			// which exits once the session is closed.
		default: // app.DestMainMenu and anything unknown
			fmt.Fprint(a.Out, menuText)
		}
	}
}

// menuText is shown at the top of every fresh session; the buttons in the
// page sidebar deliver the same digits the console menu expects.
const menuText = "Choose a section:\n" +
	"1: Learning phrases\n2: Learning hiragana\n3: Learning katakana\n" +
	"4: Reset hiragana progress\n5: Reset katakana progress\n6: Reset phrases progress\n0: Exit\n"

// Serve runs the session's main-menu loop until the browser disconnects or
// the session is closed. It must run in its own goroutine.
func (s *Session) Serve() {
	defer s.Close()

	go s.pump()

	// Wrap the session listener so flows can report completion: every
	// interactive flow listens through kbd.Watch, whose Done channel we use
	// to detect "the flow returned" vs "still waiting for input".
	inner := s.app.Listener
	seen := false
	w := kbd.Watch(inner, &seen)
	s.app.Listener = w
	defer func() { s.app.Listener = inner }()

	s.render()

	for !s.IsClosed() {
		s.writeMenu()

		done := w.Done()
		s.setWaiter(w)
		<-done
		s.setWaiter(nil)

		// The flow finished on its own (e.g. all kana learned, reset
		// confirmed): redraw the menu and keep going. If it was cancelled
		// because the session died, the loop condition below ends Serve.
		if s.IsClosed() {
			return
		}
	}
}

// pump periodically flushes the display buffer to the browser so keystroke
// echo (typing, backspaces) appears live, not only when a flow finishes.
func (s *Session) pump() {
	defer close(s.pumpDone)
	ticker := time.NewTicker(60 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.quit:
			s.render() // final flush
			return
		case <-ticker.C:
			s.mu.Lock()
			dirty := s.dirty
			s.mu.Unlock()
			if dirty {
				s.render()
			}
		}
	}
}

func (s *Session) writeMenu() {
	s.Clean()
	fmt.Fprint(s.app.Out, menuText)
	s.render()
}

func (s *Session) setWaiter(w kbd.Waitable) {
	s.mu.Lock()
	s.waiter = w
	s.mu.Unlock()
}

// Close terminates the session: pending listeners wake up, the key source
// reports itself exhausted (so any blocked flow unwinds), the running flow
// is cancelled and the transport closes.
func (s *Session) Close() {
	s.quitOnce.Do(func() { close(s.quit) })

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
	} else {
		s.closed = true
		s.exhausted = true
		if !s.keyClosed {
			close(s.keys)
			s.keyClosed = true
		}
		w := s.waiter
		s.mu.Unlock()
		if w != nil {
			w.Cancel()
		}
		<-s.pumpDone // let the pump deliver its final frame
	}

	_ = s.c.send(wsMessage{Type: "closed"})
	_ = s.c.closeWrite()
	if s.hub != nil {
		s.hub.unregister(s)
	}
}

// IsClosed reports whether the session has been terminated.
func (s *Session) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// DeliverKey feeds one keystroke coming from the browser into the session.
// It returns false when the session is already dead. Called by the websocket
// read loop for every incoming "key" message.
func (s *Session) DeliverKey(m wsMessage) bool {
	k := decodeKey(m)
	s.mu.Lock()
	closed := s.closed || s.keyClosed
	ch := s.keys
	s.mu.Unlock()
	if closed {
		return false
	}
	select {
	case ch <- k:
		return true
	case <-s.quit:
		return false
	}
}

// Listen implements kbd.Listener. It delivers browser keystrokes one by one;
// once the session is closed it never blocks and reports the source as
// exhausted, which is exactly how the interactive flows detect "the user is
// gone" (see app.sourceQuiet / sourceSpent).
func (s *Session) Listen(h kbd.Handler) error {
	for {
		s.mu.Lock()
		spent := s.exhausted
		ch := s.keys
		s.mu.Unlock()
		if spent {
			return nil
		}
		select {
		case <-s.quit:
			return nil
		case k, ok := <-ch:
			if !ok {
				return nil
			}
			if h(k) {
				return nil
			}
		}
	}
}

// IsExhausted lets app.sourceQuiet/sourceSpent recognise a dead session.
func (s *Session) IsExhausted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exhausted
}

// HasPendingKeys conservatively reports true while the session is alive: a
// live browser can always type another key, so flows must keep waiting for
// real input instead of ending early.
func (s *Session) HasPendingKeys() bool { return !s.IsExhausted() }

// Write implements io.Writer: flow output accumulates in the display buffer
// and is flushed to the browser by the render pump / explicit renders.
func (s *Session) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return len(p), nil
	}
	n, err := s.buf.Write(p)
	if n > 0 {
		s.dirty = true
	}
	return n, err
}

// Clean clears the display buffer (the web equivalent of clearing the
// terminal screen).
func (s *Session) Clean() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf.Reset()
	s.dirty = true
}

// render converts the buffered display text and sends it to the browser.
// The client pretty-prints the plain text itself, so identical consecutive
// frames are skipped and no HTML wrapping happens here.
func (s *Session) render() {
	s.mu.Lock()
	text := RenderDisplay(s.buf.String())
	s.dirty = false
	if text == s.lastFrame {
		s.mu.Unlock()
		return
	}
	s.lastFrame = text
	s.mu.Unlock()
	_ = s.c.send(wsMessage{Type: "render", HTML: text})
}

// RenderDisplay normalises the plain-text display buffer for the browser:
// invalid UTF-8 is replaced and backspace echo sequences emitted by ReadLine
// are dropped (the client renders the final state of each line).
// Exposed for tests.
func RenderDisplay(text string) string {
	return strings.NewReplacer("\b \b", "", "\b", "").
		Replace(strings.ToValidUTF8(text, "\uFFFD"))
}

// decodeKey converts a browser key message into a keys.Key event shaped just
// like the ones the real terminal listener produces (see kbd.RuneKey).
func decodeKey(m wsMessage) keys.Key {
	switch m.Code {
	case "Enter":
		return keys.Key{Code: keys.Enter}
	case "Escape":
		return keys.Key{Code: keys.Escape}
	case "Backspace":
		return keys.Key{Code: keys.Backspace}
	case "Tab":
		return keys.Key{Code: keys.Tab}
	case "Space":
		return kbd.RuneKey(' ')
	}
	if r := firstRune(m.Rune); r != 0 {
		return kbd.RuneKey(r)
	}
	return keys.Key{Code: keys.RuneKey}
}

func firstRune(s string) rune {
	for _, r := range s {
		return r
	}
	return 0
}

// MustJSON marshals v or returns "{}"; used by the handlers.
func MustJSON(v any) []byte {
	raw, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return raw
}

var (
	_ io.Writer    = (*Session)(nil)
	_ kbd.Listener = (*Session)(nil)
)
