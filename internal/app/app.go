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
}

// New creates an App wired to the real terminal. navigateTo should be set by
// the caller (see cmd/japan/main.go) to dispatch named destinations.
func New() *App {
	return &App{
		Out:      os.Stdout,
		Listener: kbd.Real{},
		Clean:    CleanDisplay,
		Exit:     os.Exit,
	}
}

// CleanDisplay clears the terminal screen.
func CleanDisplay() {
	fmt.Print("\033[2J\033[H")
}

// Navigation destination names used by the menus.
const (
	DestMainMenu  = "main"
	DestPhrases   = "phrases"
	DestHiragana  = "hiragana"
	DestQuit      = "quit"
	DestUnchanged = ""
)

// ClassifyMenuKey maps a main-menu key press to a navigation destination,
// matching the printed menu: "1: Learning phrases", "2: Learning hiragana",
// "0: Exit". Escape also exits; everything else returns DestUnchanged.
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
	}
	if len(s) != 1 || s[0] < '0' || s[0] > '9' {
		return DestUnchanged
	}
	return DestUnchanged
}

// FormatStats renders the hiragana learning statistics for the given state.
// knownKeys are answered correctly, unknownKeys were explicitly marked as not
// known, restKeys were never attempted. Keys within each section are sorted
// for deterministic output. All three slices must contain distinct keys that
// exist in data.
func FormatStats(data map[string]string, knownKeys, unknownKeys, restKeys []string) string {
	w := &strings.Builder{}
	fmt.Fprintf(w, "Statistics (%d/%d answered correctly):\n\n", len(knownKeys), len(data))

	fmt.Fprintln(w, "Known hiragana:")
	writeKeyList(w, data, knownKeys)

	fmt.Fprintln(w, "\nNot known / not answered hiragana:")
	writeKeyList(w, data, append(append([]string{}, restKeys...), unknownKeys...))

	return w.String()
}

// writeKeyList prints the "kana - reading" lines for the given keys in sorted
// order, or "none" when there is nothing to show.
func writeKeyList(w io.Writer, data map[string]string, list []string) {
	if len(list) == 0 {
		fmt.Fprintln(w, "  none")
		return
	}
	sorted := append([]string(nil), list...)
	sort.Strings(sorted)
	for _, k := range sorted {
		fmt.Fprintf(w, "  %s - %s\n", k, data[k])
	}
}
