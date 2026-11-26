package kbd

import (
	"sync"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
)

// Handler processes a single key event. It returns true when the listener
// should stop (i.e. the current interaction is finished).
type Handler func(k keys.Key) bool

// Listener abstracts the keyboard listening mechanism so that interactive
// code can be unit tested with a fake key source.
type Listener interface {
	Listen(h Handler) error
}

// Real uses atomicgo.dev/keyboard to read actual terminal input.
type Real struct{}

func (Real) Listen(h Handler) error {
	return keyboard.Listen(func(key keys.Key) (bool, error) {
		return h(key), nil
	})
}

// Fake feeds a predefined sequence of keys to the handler. It is intended
// for tests: every consumed key is recorded in Consumed. If the sequence
// ends before the handler stops, Listen returns normally and Exhausted is
// set to true; DrainedMidRead additionally records that the sequence ran out
// *during* the current read (the interaction never received its terminating
// key).
type Fake struct {
	Keys      []keys.Key
	Consumed  []keys.Key
	Exhausted bool
	// DrainedMidRead reports whether the most recent Listen call ended
	// because the key sequence ran out mid-interaction (the handler never
	// returned true). A read stopped by a real Enter/Escape key leaves it
	// false even when that key happened to be the last of the sequence.
	DrainedMidRead bool
}

// NewFake builds a fake listener for the given key sequence.
func NewFake(k ...keys.Key) *Fake {
	return &Fake{Keys: k}
}

// Listen feeds the predefined keys to the handler until it stops or the
// sequence runs out. Once the sequence is exhausted, every subsequent call
// returns immediately (after setting Exhausted) instead of spinning forever;
// interactive flows must treat this like the real terminal going quiet and
// finish their current interaction.
func (f *Fake) Listen(h Handler) error {
	f.DrainedMidRead = false
	if len(f.Keys) == 0 {
		f.Exhausted = true
		f.DrainedMidRead = true
		return nil
	}
	for len(f.Keys) > 0 {
		k := f.Keys[0]
		f.Keys = f.Keys[1:]
		f.Consumed = append(f.Consumed, k)
		if h(k) {
			// The handler stopped on this real key: the interaction was
			// properly answered. Nothing is replayed afterwards.
			return nil
		}
	}
	f.Exhausted = true
	f.DrainedMidRead = true
	return nil
}

// IsExhausted reports whether a previous Listen call consumed the whole key
// sequence without the handler stopping on its own. Flows use it together
// with HasPendingKeys to distinguish "the user answered" from "the key
// source ran dry".
func (f *Fake) IsExhausted() bool { return f.Exhausted }

// HasPendingKeys reports whether the fake can still deliver at least one key
// event to the next Listen call. Once the queue is empty the source is spent
// and flows must end their interaction instead of looping forever on
// unanswered questions.
func (f *Fake) HasPendingKeys() bool { return len(f.Keys) > 0 }

// RuneKey builds a key event for a typed character exactly like the real
// keyboard listener does (KeyCode == keys.RuneKey plus the produced rune).
func RuneKey(r rune) keys.Key {
	return keys.Key{Code: keys.RuneKey, Runes: []rune{r}}
}

// DigitKey builds a key event for a typed digit. Note: digits arrive as
// RuneKey events with the digit's rune, NOT with Code '0'..'9'. The argument
// may be given either as the numeric value (0..9) or as the ASCII byte ('0'..'9').
func DigitKey(d byte) keys.Key {
	if d >= '0' && d <= '9' {
		return RuneKey(rune(d))
	}
	return RuneKey(rune('0' + d))
}

// TextKeys builds the key events produced by typing the given text.
func TextKeys(text string) []keys.Key {
	ks := make([]keys.Key, 0, len(text))
	for _, r := range text {
		ks = append(ks, RuneKey(r))
	}
	return ks
}

// KeysWithEnter builds a key sequence that types the given text and presses Enter.
func KeysWithEnter(text string) []keys.Key {
	return append(TextKeys(text), EnterKey())
}

// Key constructors ------------------------------------------------------------

func EnterKey() keys.Key     { return keys.Key{Code: keys.Enter} }
func EscKey() keys.Key       { return keys.Key{Code: keys.Escape} }
func BackspaceKey() keys.Key { return keys.Key{Code: keys.Backspace} }

// Watch wraps the given Listener and sets *seen to true whenever a key event
// actually reaches the handler. Flows use it together with the optional
// IsExhausted method to distinguish "the user answered" from "the key source
// ran dry", which would otherwise cause infinite loops when no further keys
// can arrive (exhausted fake in tests).
//
// The returned listener also implements the Waitable interface: web sessions
// use Done/Cancel to wait until an interactive flow finishes (or to abort a
// flow that is blocked waiting for input when the browser goes away).
func Watch(l Listener, seen *bool) Waitable {
	return &watcher{inner: l, seen: seen}
}

// Waitable is implemented by listeners created with Watch. Done returns a
// channel that is closed once the most recent Listen call completes; Cancel
// aborts a blocked Listen call (the wrapped source may still deliver the
// pending event later, but the caller stops waiting for it).
type Waitable interface {
	Listener
	Done() <-chan struct{}
	Cancel()
}

type watcher struct {
	inner Listener
	seen  *bool

	mu     sync.Mutex
	done   chan struct{} // nil means "already done"
	closed bool          // the current done channel was closed
}

func (w *watcher) Done() <-chan struct{} {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.done == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return w.done
}

func (w *watcher) Cancel() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.done != nil && !w.closed {
		close(w.done)
		w.closed = true
	}
}

func (w *watcher) beginListen() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.done = make(chan struct{})
	w.closed = false
}

func (w *watcher) endListen() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.done != nil && !w.closed {
		close(w.done)
		w.closed = true
	}
	w.done = nil
}

// Inner exposes the wrapped listener so callers can probe it for the
// optional IsExhausted / HasPendingKeys methods.
func (w *watcher) Inner() Listener { return w.inner }

func (w *watcher) Listen(h Handler) error {
	w.beginListen()
	defer w.endListen()
	return w.inner.Listen(func(k keys.Key) bool {
		if w.seen != nil {
			*w.seen = true
		}
		return h(k)
	})
}

func (w *watcher) IsExhausted() bool {
	if q, ok := w.inner.(interface{ IsExhausted() bool }); ok {
		return q.IsExhausted()
	}
	return false
}

// HasPendingKeys reports whether the wrapped listener can still deliver at
// least one key event to a subsequent Listen call. Listeners without such a
// probe (the real terminal) are assumed to keep delivering events.
func (w *watcher) HasPendingKeys() bool {
	if p, ok := w.inner.(interface{ HasPendingKeys() bool }); ok {
		return p.HasPendingKeys()
	}
	return true
}
