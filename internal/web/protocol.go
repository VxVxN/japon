package web

import (
	"encoding/json"
	"strings"
)

// The web UI is a structured, graphical interface: the server owns all
// application state and sends the browser JSON "view models" describing what
// to render (menu cards, quiz questions, stats tables, confirmations). The
// browser never receives raw terminal text and replies with semantic actions
// ("select", "answer", "confirm") instead of emulated keystrokes.
//
// Server -> client frames: {"type":"state","state":State} | {"type":"closed"}
// Client -> server frames: {"type":"action","action":"select","value":"hiragana"}|
// {"type":"action","action":"answer","value":"ka"}|
// {"type":"action","action":"confirm","raw":true}

// Action names sent by the browser.
const (
	ActionSelect  = "select"  // value: destination name (see app.Dest*) or phrase choice
	ActionAnswer  = "answer"  // value: romaji answer for the current kana
	ActionConfirm = "confirm" // raw: true/false for a confirmation prompt
)

// View kinds rendered by the client.
const (
	ViewMenu   = "menu"
	ViewQuiz   = "quiz"
	ViewStats  = "stats"
	ViewDone   = "done"
	ViewNotice = "notice"
)

// State is the full snapshot of the session shown in the browser. Every
// interaction round ends with a fresh state frame, so the client simply
// renders whatever it receives.
type State struct {
	View     string        `json:"view"`
	Title    string        `json:"title,omitempty"`
	Subtitle string        `json:"subtitle,omitempty"`
	Menu     []MenuItem    `json:"menu,omitempty"`
	Quiz     *Quiz         `json:"quiz,omitempty"`
	Stats    *Stats        `json:"stats,omitempty"`
	Done     *DoneInfo     `json:"done,omitempty"`
	Notice   *NoticeInfo   `json:"notice,omitempty"`
	Progress *ProgressInfo `json:"progress,omitempty"`
}

// MenuItem is one card of the main menu.
type MenuItem struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Icon  string `json:"icon"`
	Style string `json:"style,omitempty"` // "" | "danger"
}

// Quiz is the current question of a learning flow.
type Quiz struct {
	Kind        string `json:"kind"` // "kana" | "phrase"
	Prompt      string `json:"prompt"`
	Translation string `json:"translation,omitempty"` // revealed on request / after answer
	Input       bool   `json:"input"`                 // show the answer form
	Hint        string `json:"hint,omitempty"`
	Retry       bool   `json:"retry,omitempty"`   // previously wrong / unknown
	Correct     *bool  `json:"correct,omitempty"` // set on the kana feedback screen
	Section     string `json:"section,omitempty"` // learning section id (for the reset button)
}

// StatItem is one "symbol — reading" row of a statistics group.
type StatItem struct {
	Symbol  string `json:"symbol"`
	Reading string `json:"reading"`
}

// Stats is the result screen of a kana quiz.
type Stats struct {
	Correct int        `json:"correct"`
	Total   int        `json:"total"`
	Known   []StatItem `json:"known,omitempty"`
	Rest    []StatItem `json:"rest,omitempty"`
}

// DoneInfo reports that the whole data set has been learned (or the session
// ended).
type DoneInfo struct {
	Message string `json:"message"`
}

// NoticeInfo is an informational screen (reset confirmation / error).
type NoticeInfo struct {
	Message string `json:"message"`
	Confirm bool   `json:"confirm"` // ask yes/no instead of just showing text
}

// ProgressInfo carries the persisted learning totals for the sidebar.
type ProgressInfo struct {
	HiraganaKnown int `json:"hiraganaKnown"`
	HiraganaTotal int `json:"hiraganaTotal"`
	KatakanaKnown int `json:"katakanaKnown"`
	KatakanaTotal int `json:"katakanaTotal"`
	PhrasesKnown  int `json:"phrasesKnown"`
	PhrasesTotal  int `json:"phrasesTotal"`
}

// action is one decoded browser action.
type action struct {
	kind      string
	value     string
	confirmed bool // ActionConfirm payload
}

// decodeAction parses an incoming message; ok is false for malformed input.
func decodeAction(m wsMessage) (action, bool) {
	if m.Type != "action" || m.Action == "" {
		return action{}, false
	}
	switch m.Action {
	case ActionSelect:
		return action{kind: ActionSelect, value: strings.TrimSpace(m.Value)}, true
	case ActionAnswer:
		return action{kind: ActionAnswer, value: m.Value}, true
	case ActionConfirm:
		var b bool
		if len(m.Raw) > 0 {
			_ = json.Unmarshal(m.Raw, &b)
		}
		return action{kind: ActionConfirm, confirmed: b}, true
	}
	return action{}, false
}
