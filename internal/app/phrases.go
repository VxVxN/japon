package app

import (
	"fmt"
	"math/rand"
	"sort"

	"atomicgo.dev/keyboard/keys"
)

// printLearningPhrasesMenu prints the per-phrase menu.
func (a *App) printLearningPhrasesMenu() {
	fmt.Fprint(a.Out, "1: I can translate\n2: Show translation\n3: I can't translate\n4: Go back to the main menu\n0: Exit\n")
}

// RunLearningPhrases walks through the given phrase set. Phrases are shown in
// random order and progress is persisted: phrases the user already confirmed
// they can translate ("I can translate") are skipped in later sessions, while
// phrases marked as not translatable stay in the rotation. When every phrase
// is known, a completion message is printed instead of asking anything.
//
// For each phrase the user either confirms they can translate it ("1"), asks
// to see the translation ("2" — the translation is shown and the menu stays
// in place), marks the phrase as not translatable ("3"), returns to the main
// menu ("4") or exits ("0"/Esc).
func (a *App) RunLearningPhrases(phrases map[string]string) {
	// Restore previously learned phrases so they are not shown again. Only
	// keys that exist in the current data set are trusted.
	known := map[string]bool{}
	for _, k := range a.LoadProgress().KnownPhrases {
		if _, ok := phrases[k]; ok {
			known[k] = true
		}
	}

	markKnown := func(k string) {
		if known[k] {
			return
		}
		known[k] = true
		list := make([]string, 0, len(known))
		for kk := range known {
			list = append(list, kk)
		}
		sort.Strings(list)
		p := a.LoadProgress()
		p.KnownPhrases = list
		_ = a.SaveProgress(p)
	}

	// Ask only about phrases that are still unknown, in random order.
	remaining := func() []string {
		out := make([]string, 0, len(phrases))
		for k := range phrases {
			if !known[k] {
				out = append(out, k)
			}
		}
		rand.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
		return out
	}

	rounds := 0
	for {
		queue := remaining()
		if len(queue) == 0 {
			if rounds == 0 {
				a.Clean()
				fmt.Fprintln(a.Out, "You already know all the phrases!")
				fmt.Fprintln(a.Out, "\nPress Enter to return to the menu...")
				a.waitForEnter()
			} else {
				fmt.Fprintln(a.Out, "\nAll phrases learned! Well done!")
			}
			a.NavigateTo(DestMainMenu)
			return
		}

		for _, k := range queue {
			value := phrases[k]
			printedTranslation := false
			fmt.Fprintln(a.Out, k)
			fmt.Fprintln(a.Out)
			a.printLearningPhrasesMenu()

			stillListening := true
			skipPhrase := false
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
					markKnown(k)
					a.Clean()
					skipPhrase = true
					return true
				case "2":
					// Show the translation but keep the menu in place.
					if !printedTranslation {
						fmt.Fprintf(a.Out, "%s - %s\n", k, value)
						printedTranslation = true
					}
					return false
				case "3":
					a.Clean()
					skipPhrase = true
					return true
				case "4":
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
					printedTranslation = false
					a.printLearningPhrasesMenu()
					return false
				}
			})
			if !stillListening {
				return
			}
			if skipPhrase {
				continue
			}
			fmt.Fprintln(a.Out)
		}
		rounds++
		// Unknown phrases are re-asked (reshuffled) until everything is known.
	}
}

// SortedPhraseKeys returns phrase keys in deterministic (sorted) order.
func SortedPhraseKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
