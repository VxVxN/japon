package app

import (
	"fmt"
	"sort"

	"atomicgo.dev/keyboard/keys"
)

// printLearningPhrasesMenu prints the per-phrase menu.
func (a *App) printLearningPhrasesMenu() {
	fmt.Fprint(a.Out, "1: I can translate\n2: I can't translate\n3: Go back to the main menu\n0: Exit\n")
}

// RunLearningPhrases walks through the given phrase set. For each phrase the
// user either confirms they can translate it ("1"), asks to see the
// translation ("2"), returns to the main menu ("3") or exits ("0"/Esc).
func (a *App) RunLearningPhrases(phrases map[string]string) {
	for _, k := range SortedPhraseKeys(phrases) {
		value := phrases[k]
		fmt.Fprintln(a.Out, k)
		fmt.Fprintln(a.Out)
		a.printLearningPhrasesMenu()

		stillListening := true
		_ = a.Listener.Listen(func(key keys.Key) bool {
			if !stillListening {
				// The user chose to leave this flow; stop consuming further keys.
				return true
			}
			switch key.String() {
			case "0":
				a.Exit(0)
				stillListening = false
				return true
			case "1":
				a.Clean()
				return true
			case "2":
				a.Clean()
				fmt.Fprintf(a.Out, "%s - %s\n", k, value)
				return true
			case "3":
				a.NavigateTo(DestMainMenu)
				stillListening = false
				return true
			default:
				if key.Code == keys.Escape {
					a.Exit(0)
					stillListening = false
					return true
				}
				a.Clean()
				a.printLearningPhrasesMenu()
				return false
			}
		})
		if !stillListening {
			return
		}
		fmt.Fprintln(a.Out)
	}
}

// SortedPhraseKeys returns phrase keys in deterministic order.
func SortedPhraseKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
