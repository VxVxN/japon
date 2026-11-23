package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"atomicgo.dev/keyboard/keys"
)

// app bundles all external dependencies of the interactive flows so they can
// be swapped out in unit tests.
type app struct {
	out        io.Writer         // where all program output goes
	listener   keyListener       // source of key events
	clean      func()            // clears the display
	exit       func(code int)    // called instead of os.Exit, so tests can intercept it
	navigateTo func(name string) // how menu navigation is performed
}

// newApp creates an app wired to the real terminal.
func newApp() *app {
	return &app{
		out:        os.Stdout,
		listener:   realKeyListener{},
		clean:      cleanDisplay,
		exit:       os.Exit,
		navigateTo: navigateTo,
	}
}

// navigateTo dispatches a named destination to its flow. It breaks the
// mainMenu <-> learningPhrases recursion by going through a function variable.
func navigateTo(name string) {
	switch name {
	case destPhrases:
		currentApp.runLearningPhrases(allPhrases())
	case destHiragana:
		currentApp.runLearningHiragana(hiraganaMap)
	default: // destMainMenu
		currentApp.runMainMenu()
	}
}

// currentApp is the app instance used by navigateTo. It is set once in main
// and replaced with a fake one inside tests. It cannot be initialized at
// package level because newApp refers to navigateTo which refers back to it.
var currentApp *app

// Navigation destination names used by the menus.
const (
	destMainMenu  = "main"
	destPhrases   = "phrases"
	destHiragana  = "hiragana"
	destQuit      = "quit"
	destUnchanged = ""
)

// classifyMenuKey maps a main-menu key press to a navigation destination,
// matching the printed menu: "1: Learning phrases", "2: Learning hiragana",
// "0: Exit". Escape also exits; everything else returns destUnchanged.
func classifyMenuKey(k keys.Key) string {
	if k.Code == keys.Escape {
		return destQuit
	}
	s := k.String()
	switch s {
	case "0":
		return destQuit
	case "1":
		return destPhrases
	case "2":
		return destHiragana
	}
	if len(s) != 1 || s[0] < '0' || s[0] > '9' {
		return destUnchanged
	}
	return destUnchanged
}

// formatStats renders the hiragana learning statistics for the given state.
// knownKeys are answered correctly, unknownKeys were explicitly marked as not
// known, restKeys were never attempted. Keys within each section are sorted
// for deterministic output. All three slices must contain distinct keys that
// exist in data.
func formatStats(data map[string]string, knownKeys, unknownKeys, restKeys []string) string {
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
