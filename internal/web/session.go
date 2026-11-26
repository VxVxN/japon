// Package web serves the learning application as a graphical web UI. Each
// browser connection gets its own Session. The session owns the whole
// interaction loop: it loads the data sets and the persisted progress,
// renders structured JSON view models to the browser (menu cards, quiz
// questions, statistics tables) and applies the semantic actions coming back
// from it ("select", "answer", "confirm"). No terminal text or emulated
// keystrokes are involved — the interface is a real web app, not a mirror of
// a console screen.
package web

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strings"
	"sync"

	"test/internal/app"
	"test/internal/data"
)

// wsMessage is one JSON frame exchanged with the browser.
//
// Server -> client: {"type":"state","state":{...}} | {"type":"closed"}.
// Client -> server: {"type":"action","action":"select|answer|confirm","value":...}.
type wsMessage struct {
	Type    string          `json:"type"`
	State   *State          `json:"state,omitempty"`
	Action  string          `json:"action,omitempty"`
	Value   string          `json:"value,omitempty"`
	Raw     json.RawMessage `json:"raw,omitempty"` // typed payload (bool for confirm)
	Message string          `json:"message,omitempty"`
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

// scriptSpec describes one kana data set tracked by the session.
type scriptSpec struct {
	dest       string // app.DestHiragana / app.DestKatakana
	name       string // display name
	title      string // view title
	table      map[string]string
	knownField func(*app.ProgressData) *[]string
}

// resetTarget identifies which persisted list a pending reset clears.
type resetTarget struct {
	label string
	field func(*app.ProgressData) *[]string
}

// Session is one browser tab running the application.
type Session struct {
	c   conn
	hub *hub
	app *app.App

	quit      chan struct{}
	closeOnce sync.Once
	mu        sync.Mutex
	closed    bool

	// Interaction state. All fields below are guarded by actMu and only
	// touched from DeliverAction (the websocket read-loop goroutine).
	actMu        sync.Mutex
	awaiting     string // one of awaitNone / awaitAnswer / awaitFeedback / awaitConfirm
	prompt       string // current question prompt (kana symbol / phrase key)
	pendingRetry bool   // the pending re-ask is a retry of a previously wrong kana
	pendingReset *resetTarget

	hiraScr, kataScr scriptSpec
	phraseTable      map[string]string
	// retryPool tracks the kana answered incorrectly during the current quiz
	// run; they are asked again before new ones (see askNextKana).
	retryPool map[string]bool
}

// NewSession wires an app.App around the given connection. It is separated
// from the HTTP handler so tests can create sessions over a fake transport.
func NewSession(c conn, h *hub) *Session {
	s := &Session{
		c:    c,
		hub:  h,
		quit: make(chan struct{}),
	}

	a := app.New()
	// The web UI never writes terminal output and never exits the process:
	// Out swallows any legacy flow text, Clean/Exit/NavigateTo are inert
	// because the session drives its own state machine instead of the
	// console flows.
	a.Out = discardWriter{}
	a.Clean = func() {}
	a.Exit = func(int) {}
	a.NavigateTo = func(string) {}
	s.app = a

	s.hiraScr = scriptSpec{
		dest:       app.DestHiragana,
		name:       "hiragana",
		title:      "Хирагана",
		table:      data.Hiragana,
		knownField: func(p *app.ProgressData) *[]string { return &p.KnownHiragana },
	}
	s.kataScr = scriptSpec{
		dest:       app.DestKatakana,
		name:       "katakana",
		title:      "Катакана",
		table:      data.Katakana,
		knownField: func(p *app.ProgressData) *[]string { return &p.KnownKatakana },
	}
	s.phraseTable = data.AllPhrases()
	s.retryPool = map[string]bool{}

	if h != nil {
		h.register(s)
	}
	return s
}

// discardWriter absorbs the text the shared app flows print; the web UI
// renders structured state instead.
type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

// Serve sends the initial menu state and then waits until the session is
// closed. All further rendering happens synchronously inside DeliverAction.
func (s *Session) Serve() {
	s.sendState(s.menuState())
	<-s.quit
}

// Close terminates the session and closes the transport. Safe to call twice.
func (s *Session) Close() {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()
		close(s.quit)
		_ = s.c.send(wsMessage{Type: "closed"})
		_ = s.c.closeWrite()
		if s.hub != nil {
			s.hub.unregister(s)
		}
	})
}

// IsClosed reports whether the session has been terminated.
func (s *Session) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// DeliverAction applies one browser action and pushes the resulting state
// back to the client. It returns false when the session is dead (the read
// loop then tears the connection down).
func (s *Session) DeliverAction(m wsMessage) bool {
	if s.IsClosed() {
		return false
	}
	a, ok := decodeAction(m)
	if !ok {
		return true
	}
	s.actMu.Lock()
	defer s.actMu.Unlock()

	switch a.kind {
	case ActionSelect:
		return s.handleSelect(a.value)
	case ActionAnswer:
		return s.handleAnswer(a.value)
	case ActionConfirm:
		return s.handleConfirm(a.confirmed)
	}
	return true
}

// ---------- rendering helpers ----------

func (s *Session) sendState(st *State) bool {
	if s.IsClosed() {
		return false
	}
	st.Progress = s.progressInfo()
	return s.c.send(wsMessage{Type: "state", State: st}) == nil
}

func (s *Session) progressInfo() *ProgressInfo {
	p := s.app.LoadProgress()
	count := func(list []string, table map[string]string) int {
		n := 0
		for _, k := range list {
			if _, ok := table[k]; ok {
				n++
			}
		}
		return n
	}
	return &ProgressInfo{
		HiraganaKnown: count(p.KnownHiragana, s.hiraScr.table),
		HiraganaTotal: len(s.hiraScr.table),
		KatakanaKnown: count(p.KnownKatakana, s.kataScr.table),
		KatakanaTotal: len(s.kataScr.table),
		PhrasesKnown:  count(p.KnownPhrases, s.phraseTable),
		PhrasesTotal:  len(s.phraseTable),
	}
}

func (s *Session) menuState() *State {
	return &State{
		View:     ViewMenu,
		Title:    "Главное меню",
		Subtitle: "Выберите раздел",
		Menu: []MenuItem{
			{ID: app.DestPhrases, Label: "Учим фразы", Icon: "book"},
			{ID: app.DestHiragana, Label: "Учим хирагану", Icon: "pencil"},
			{ID: app.DestKatakana, Label: "Учим катакану", Icon: "kata"},
		},
		// Progress resets are secondary actions: the client renders them as a
		// compact row of small buttons under the menu instead of full cards.
		Resets: []MenuItem{
			{ID: app.DestResetPhrases, Label: "Фразы", Icon: "reset", Style: "danger"},
			{ID: app.DestResetHiragana, Label: "Хирагана", Icon: "reset", Style: "danger"},
			{ID: app.DestResetKatakana, Label: "Катакана", Icon: "reset", Style: "danger"},
		},
	}
}

// Awaiting states of the session interaction state machine.
const (
	awaitNone     = ""         // idle: no pending interaction
	awaitAnswer   = "answer"   // waiting for a quiz answer
	awaitFeedback = "feedback" // showing the answer result, waiting for continue
	awaitConfirm  = "confirm"  // waiting for a yes/no confirmation
)

// stopQuiz clears any pending interaction state before switching views.
func (s *Session) stopQuiz() {
	s.awaiting = awaitNone
	s.prompt = ""
	s.pendingRetry = false
	s.pendingReset = nil
}

// ---------- select handling ----------

func (s *Session) handleSelect(dest string) bool {
	switch dest {
	case app.DestMainMenu:
		s.stopQuiz()
		return s.sendState(s.menuState())
	case app.DestQuit:
		return s.closeWithGoodbye()
	case app.DestPhrases:
		return s.startPhrases()
	case app.DestHiragana:
		return s.startKanaQuiz(&s.hiraScr)
	case app.DestKatakana:
		return s.startKanaQuiz(&s.kataScr)
	case app.DestResetHiragana:
		return s.startReset("Хирагана", "все выученные знаки будут снова заданы.", &s.hiraScr)
	case app.DestResetKatakana:
		return s.startReset("Катакана", "все выученные знаки будут снова заданы.", &s.kataScr)
	case app.DestResetPhrases:
		return s.startReset("Фразы", "все выученные фразы будут снова показаны.", nil)
	default:
		return s.sendState(s.menuState())
	}
}

// closeWithGoodbye ends the session after telling the client to show the
// goodbye screen (the read loop keeps the socket open long enough to flush).
func (s *Session) closeWithGoodbye() bool {
	ok := s.c.send(wsMessage{Type: "state", State: &State{
		View:  ViewDone,
		Title: "До встречи!",
		Done:  &DoneInfo{Message: "Сессия завершена. Обновите страницу, чтобы начать заново."},
	}})
	s.Close()
	return ok == nil
}

// ---------- kana quiz ----------

func (s *Session) knownSet(table map[string]string, list []string) map[string]bool {
	known := map[string]bool{}
	for _, k := range list {
		if _, ok := table[k]; ok {
			known[k] = true
		}
	}
	return known
}

func sortedKeyList(known map[string]bool) []string {
	out := make([]string, 0, len(known))
	for k := range known {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func knownIntersection(known map[string]bool, table map[string]string) int {
	n := 0
	for k := range known {
		if _, ok := table[k]; ok {
			n++
		}
	}
	return n
}

func (s *Session) startKanaQuiz(sc *scriptSpec) bool {
	p := s.app.LoadProgress()
	known := s.knownSet(sc.table, *sc.knownField(&p))
	s.retryPool = map[string]bool{}
	if len(known) >= len(sc.table) {
		s.stopQuiz()
		return s.sendState(&State{
			View:  ViewDone,
			Title: sc.title + " уже выучена",
			Done:  &DoneInfo{Message: fmt.Sprintf("Вы знаете все %d знаков. Сбросьте прогресс, чтобы пройти ещё раз.", len(sc.table))},
		})
	}
	return s.askNextKana(sc, known, false)
}

// askNextKana picks a random unknown kana, stores it as the pending prompt
// and renders the quiz card. When retry is true the pool is restricted to
// the kana that were answered incorrectly earlier in this session (the
// console-flow behaviour: difficult signs come back until they are known).
func (s *Session) askNextKana(sc *scriptSpec, known map[string]bool, retry bool) bool {
	pool := make([]string, 0, len(sc.table))
	for k := range sc.table {
		if !known[k] {
			pool = append(pool, k)
		}
	}
	if retry && len(s.retryPool) > 0 {
		filtered := make([]string, 0, len(s.retryPool))
		for k := range s.retryPool {
			if !known[k] {
				filtered = append(filtered, k)
			}
		}
		if len(filtered) > 0 {
			pool = filtered
		}
	}
	if len(pool) == 0 {
		return s.showKanaStats(sc, known)
	}
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	k := pool[0]

	s.prompt = k
	s.awaiting = awaitAnswer
	return s.sendState(&State{
		View:  ViewQuiz,
		Title: sc.title,
		Quiz: &Quiz{
			Kind:    "kana",
			Prompt:  k,
			Input:   true,
			Retry:   retry,
			Section: sc.name,
			Hint:    fmt.Sprintf("Уже выучено: %d из %d", knownIntersection(known, sc.table), len(sc.table)),
		},
	})
}

func (s *Session) handleAnswer(value string) bool {
	// The client acknowledged a feedback screen: show the next question.
	if s.awaiting == awaitFeedback {
		if s.prompt == "" {
			return s.handleSelect(app.DestMainMenu)
		}
		sc := &s.hiraScr
		if _, ok := sc.table[s.prompt]; !ok {
			sc = &s.kataScr
		}
		if _, ok := sc.table[s.prompt]; !ok {
			return s.handleSelect(app.DestMainMenu)
		}
		p := s.app.LoadProgress()
		known := s.knownSet(sc.table, *sc.knownField(&p))
		retry := s.pendingRetry
		s.pendingRetry = false
		if knownIntersection(known, sc.table) >= len(sc.table) {
			return s.showKanaStats(sc, known)
		}
		return s.askNextKana(sc, known, retry)
	}

	if s.awaiting != awaitAnswer || s.prompt == "" {
		return true // stale answer: ignore
	}

	// Phrases flow: the answer carries the choice made for the phrase.
	if strings.HasPrefix(s.prompt, phrasePromptPrefix) {
		return s.handlePhraseChoice(strings.TrimPrefix(s.prompt, phrasePromptPrefix), value)
	}

	// Kana quiz: determine which script the pending prompt belongs to.
	sc := &s.hiraScr
	if _, ok := sc.table[s.prompt]; !ok {
		sc = &s.kataScr
	}
	if _, ok := sc.table[s.prompt]; !ok {
		return true
	}
	p := s.app.LoadProgress()
	known := s.knownSet(sc.table, *sc.knownField(&p))
	answer := strings.ToLower(strings.TrimSpace(value))
	correct := answer == sc.table[s.prompt]

	var feedback *State
	if correct {
		known[s.prompt] = true
		delete(s.retryPool, s.prompt)
		*sc.knownField(&p) = sortedKeyList(known)
		if err := s.app.SaveProgress(p); err != nil {
			log.Printf("web: save progress: %v", err)
		}
		feedback = &State{View: ViewQuiz, Title: sc.title, Quiz: &Quiz{
			Kind: "kana", Prompt: s.prompt, Translation: sc.table[s.prompt],
			Hint: "Верно! Отлично!", Input: false, Correct: ptr(true), Section: sc.name,
		}}
		// Pause on the feedback screen so the user sees the result; the client
		// shows a "Продолжить" button that sends another "answer" action.
		s.awaiting = awaitFeedback
		s.pendingRetry = false
		return s.sendState(feedback)
	}

	if strings.TrimSpace(answer) != "" {
		s.retryPool[s.prompt] = true
	}
	feedback = &State{View: ViewQuiz, Title: sc.title, Quiz: &Quiz{
		Kind: "kana", Prompt: s.prompt, Translation: sc.table[s.prompt],
		Hint: wrongAnswerHint(answer), Input: false, Correct: ptr(false), Section: sc.name,
	}}
	// Pause on the feedback screen so the user sees the correct reading; the
	// client shows a "Продолжить" button that sends another "answer" action.
	s.awaiting = awaitFeedback
	s.pendingRetry = true
	return s.sendState(feedback)
}

// ptr returns a pointer to b, used for the optional *bool field of Quiz.
func ptr(b bool) *bool { return &b }

func wrongAnswerHint(answer string) string {
	if strings.TrimSpace(answer) == "" {
		return "Пустой ответ — попробуйте ещё раз."
	}
	return "Неверно. Этот знак вернётся в очереди ещё раз."
}

func (s *Session) showKanaStats(sc *scriptSpec, known map[string]bool) bool {
	s.stopQuiz()

	knownItems := make([]StatItem, 0)
	restItems := make([]StatItem, 0)
	for _, k := range sortedKeyList(known) {
		knownItems = append(knownItems, StatItem{Symbol: k, Reading: sc.table[k]})
	}
	var restKeys []string
	for k := range sc.table {
		if !known[k] {
			restKeys = append(restKeys, k)
		}
	}
	sort.Strings(restKeys)
	for _, k := range restKeys {
		restItems = append(restItems, StatItem{Symbol: k, Reading: sc.table[k]})
	}
	return s.sendState(&State{
		View:  ViewStats,
		Title: "Статистика · " + sc.title,
		Stats: &Stats{Correct: len(knownItems), Total: len(sc.table), Known: knownItems, Rest: restItems},
	})
}

// ---------- phrases flow ----------

const phrasePromptPrefix = "phrase:"

func (s *Session) startPhrases() bool { return s.askNextPhrase(nil, "") }

// askNextPhrase picks the next phrase to show. exclude is the phrase the user
// just decided about ("unknown"); it must never come back immediately — the
// rotation has to move on to a different phrase (falling back to exclude only
// when it is the last unknown one left).
func (s *Session) askNextPhrase(retry map[string]bool, exclude string) bool {
	p := s.app.LoadProgress()
	known := s.knownSet(s.phraseTable, p.KnownPhrases)
	if len(known) >= len(s.phraseTable) {
		s.stopQuiz()
		return s.sendState(&State{
			View:  ViewDone,
			Title: "Все фразы выучены!",
			Done:  &DoneInfo{Message: fmt.Sprintf("Вы можете перевести все %d фраз. Отличная работа!", len(s.phraseTable))},
		})
	}
	queue := make([]string, 0, len(s.phraseTable))
	retryLeft := make([]string, 0)
	for k := range s.phraseTable {
		if known[k] {
			continue
		}
		queue = append(queue, k)
		if retry[k] {
			retryLeft = append(retryLeft, k)
		}
	}
	pool := queue
	if len(retryLeft) > 0 {
		pool = retryLeft
	}
	// Never re-show the phrase the user just marked as unknown: pick another
	// one from the pool so the rotation visibly moves forward.
	if exclude != "" && len(pool) > 1 {
		filtered := make([]string, 0, len(pool))
		for _, k := range pool {
			if k != exclude {
				filtered = append(filtered, k)
			}
		}
		if len(filtered) > 0 {
			pool = filtered
		}
	}
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	k := pool[0]

	s.prompt = phrasePromptPrefix + k
	s.awaiting = awaitAnswer
	return s.sendState(&State{
		View:  ViewQuiz,
		Title: "Фразы",
		Quiz: &Quiz{
			Kind:    "phrase",
			Prompt:  k,
			Input:   true, // show the phrase action buttons (know / show / unknown / menu)
			Section: "phrases",
			Hint:    fmt.Sprintf("Уже выучено: %d из %d", knownIntersection(known, s.phraseTable), len(s.phraseTable)),
			Retry:   retry[k],
		},
	})
}

// handlePhraseChoice processes a button choice for the pending phrase:
// "know" marks it learned, "show" reveals the translation (staying on the
// same phrase), "unknown" puts it at the end of the rotation, "menu" leaves
// the flow.
func (s *Session) handlePhraseChoice(key, choice string) bool {
	value, ok := s.phraseTable[key]
	if !ok {
		return true
	}
	switch choice {
	case "show":
		// Reveal the translation but keep the phrase in front of the user:
		// the action buttons stay visible so they can still confirm ("know")
		// or skip ("unknown") this very phrase before moving on to the next
		// one. The session remains in awaitAnswer for this phrase.
		p := s.app.LoadProgress()
		known := s.knownSet(s.phraseTable, p.KnownPhrases)
		return s.sendState(&State{View: ViewQuiz, Title: "Фразы", Quiz: &Quiz{
			Kind:        "phrase",
			Prompt:      key,
			Translation: value,
			Section:     "phrases",
			Input:       true,
			Hint:        fmt.Sprintf("Уже выучено: %d из %d", knownIntersection(known, s.phraseTable), len(s.phraseTable)),
		}})
	case "menu":
		s.stopQuiz()
		return s.sendState(s.menuState())
	case "know", "unknown":
		p := s.app.LoadProgress()
		known := s.knownSet(s.phraseTable, p.KnownPhrases)
		retry := map[string]bool{}
		if choice == "know" {
			known[key] = true
			p.KnownPhrases = sortedKeyList(known)
			if err := s.app.SaveProgress(p); err != nil {
				log.Printf("web: save progress: %v", err)
			}
		} else {
			retry[key] = true
		}
		s.stopQuiz()
		return s.askNextPhrase(retry, key)
	}
	return true
}

// ---------- reset flows ----------

func (s *Session) startReset(label, detail string, sc *scriptSpec) bool {
	s.stopQuiz()
	t := &resetTarget{label: label}
	if sc != nil {
		t.field = sc.knownField
	} else {
		t.field = func(p *app.ProgressData) *[]string { return &p.KnownPhrases }
	}
	s.pendingReset = t
	s.awaiting = awaitConfirm
	return s.sendState(&State{
		View:   ViewNotice,
		Title:  "Сброс прогресса",
		Notice: &NoticeInfo{Message: "Прогресс по разделу «" + label + "» будет удалён: " + detail, Confirm: true},
	})
}

func (s *Session) handleConfirm(yes bool) bool {
	if s.awaiting != awaitConfirm || s.pendingReset == nil {
		return true
	}
	t := s.pendingReset
	s.stopQuiz()

	if yes {
		p := s.app.LoadProgress()
		*t.field(&p) = nil
		if err := s.app.SaveProgress(p); err != nil {
			return s.sendState(&State{View: ViewNotice, Title: "Ошибка",
				Notice: &NoticeInfo{Message: fmt.Sprintf("Не удалось сбросить прогресс: %v", err)}})
		}
		s.retryPool = map[string]bool{}
	}
	return s.sendState(s.menuState())
}
