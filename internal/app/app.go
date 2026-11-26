package app

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"test/internal/kbd"

	"atomicgo.dev/keyboard/keys"
)

// App bundles all external dependencies of the interactive flows so they can
// be swapped out in unit tests.
type App struct {
	Out        io.Writer         // where all program output goes
	Listener   kbd.Listener      // source of key events
	Clean      func()            // clears the display
	Exit       func(code int)    // called instead of os.Exit, so tests can intercept it
	NavigateTo func(name string) // how menu navigation is performed

	// Progress persistence hooks; tests replace them with in-memory fakes.
	LoadProgress         func() ProgressData
	SaveProgress         func(ProgressData) error
	ResetProgress        func() error
	ResetHiragana        func() error
	ResetKatakana        func() error
	ResetPhrasesProgress func() error
}

// New creates an App wired to the real terminal. navigateTo should be set by
// the caller (see cmd/japan/main.go) to dispatch named destinations.
func New() *App {
	return &App{
		Out:                os.Stdout,
		Listener:           kbd.Real{},
		Clean:              CleanDisplay,
		Exit:               os.Exit,
		LoadProgress:       LoadProgress,
		SaveProgress:       SaveProgress,
		ResetProgress:      ResetProgress,
		ResetHiragana:      ResetHiraganaProgress,
		ResetKatakana:      ResetKatakanaProgress,
		ResetPhrasesProgress: ResetPhrasesProgress,
	}
}

// CleanDisplay clears the terminal screen.
func CleanDisplay() {
	fmt.Print("\033[2J\033[H")
}

// Navigation destination names used by the menus.
const (
	DestMainMenu       = "main"
	DestPhrases        = "phrases"
	DestHiragana       = "hiragana"
	DestKatakana       = "katakana"
	DestReset          = "reset"
	DestResetHiragana  = "reset-hiragana"
	DestResetKatakana  = "reset-katakana"
	DestResetPhrases   = "reset-phrases"
	DestQuit           = "quit"
	DestUnchanged      = ""
)

// ClassifyMenuKey maps a main-menu key press to a navigation destination,
// matching the printed menu: "1: Learning phrases", "2: Learning hiragana",
// "3: Learning katakana", "4: Reset hiragana progress",
// "5: Reset katakana progress", "6: Reset phrases progress", "0: Exit".
// Escape also exits; everything else returns DestUnchanged.
func ClassifyMenuKey(k keys.Key) string {
	if k.Code == keys.Escape {
		return DestQuit
	}
	s := k.String()
	switch s {
	case "0":
		return DestQuit
	case "1":
		return DestPhrases
	case "2":
		return DestHiragana
	case "3":
		return DestKatakana
	case "4":
		return DestResetHiragana
	case "5":
		return DestResetKatakana
	case "6":
		return DestResetPhrases
	}
	if len(s) != 1 || s[0] < '0' || s[0] > '9' {
		return DestUnchanged
	}
	return DestUnchanged
}

// FormatStats renders the hiragana learning statistics for the given state.
// It defaults to the "hiragana" wording in the section headers.
func FormatStats(data map[string]string, knownKeys, unknownKeys, restKeys []string) string {
	return FormatStatsScript("hiragana", data, knownKeys, unknownKeys, restKeys)
}

// FormatStatsScript renders the kana learning statistics using scriptName
// ("hiragana" or "katakana") in the section headers. knownKeys are answered
// correctly, unknownKeys were explicitly marked as not known, restKeys were
// never attempted. Keys within each section are sorted for deterministic
// output. All three slices must contain distinct keys that exist in data.
func FormatStatsScript(scriptName string, data map[string]string, knownKeys, unknownKeys, restKeys []string) string {
	w := &strings.Builder{}
	fmt.Fprintf(w, "Statistics (%d/%d answered correctly):\n\n", len(knownKeys), len(data))

	fmt.Fprintf(w, "Known %s:\n", scriptName)
	writeKeyList(w, data, knownKeys)

	fmt.Fprintf(w, "\nNot known / not answered %s:\n", scriptName)
	writeKeyList(w, data, append(append([]string{}, restKeys...), unknownKeys...))

	return w.String()
}

// statsColumns is how many "kana - reading" entries are printed per line in
// the statistics sections, keeping the output compact enough to fit on one
// screen.
const statsColumns = 6

// writeKeyList prints the "kana - reading" entries for the given keys in
// sorted order as a compact multi-column grid (several entries per line), or
// "none" when there is nothing to show.
func writeKeyList(w io.Writer, data map[string]string, list []string) {
	if len(list) == 0 {
		fmt.Fprintln(w, "  none")
		return
	}
	sorted := append([]string(nil), list...)
	sort.Strings(sorted)

	entries := make([]string, 0, len(sorted))
	width := 0
	for _, k := range sorted {
		e := fmt.Sprintf("%s - %s", k, data[k])
		entries = append(entries, e)
		if r := len([]rune(e)); r > width {
			width = r
		}
	}

	pad := strings.Repeat(" ", width)
	for i := 0; i < len(entries); i += statsColumns {
		end := i + statsColumns
		if end > len(entries) {
			end = len(entries)
		}
		sb := &strings.Builder{}
		sb.WriteString("  ")
		for j, e := range entries[i:end] {
			if j > 0 {
				sb.WriteString(" | ")
			}
			sb.WriteString(e)
			// Pad every cell except the last one on the last row so that the
			// columns of the grid line up without trailing spaces.
			if i+statsColumns < len(entries) {
				sb.WriteString(pad[len([]rune(e)):])
			}
		}
		fmt.Fprintln(w, strings.TrimRight(sb.String(), " "))
	}
}
