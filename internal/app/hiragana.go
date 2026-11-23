package app

import (
	"fmt"
	"sort"
	"strings"

	"atomicgo.dev/keyboard/keys"
)

// printLearningHiraganaMenu prints the help text of the hiragana exercise.
func (a *App) printLearningHiraganaMenu() {
	fmt.Fprintln(a.Out, "Type the reading in English letters and press Enter to check.")
	fmt.Fprintln(a.Out, "Press Ctrl+C or Esc to exit and see your statistics.")
}

// ReadLine collects typed characters until Enter is pressed. Backspace removes
// the last character and Escape aborts the input. It returns the raw text and
// whether the user aborted with Escape.
func (a *App) ReadLine() (text string, aborted bool) {
	_ = a.Listener.Listen(func(k keys.Key) bool {
		switch k.Code {
		case keys.Escape:
			aborted = true
			return true
		case keys.Enter:
			return true
		case keys.Backspace:
			if len(text) > 0 {
				runes := []rune(text)
				text = string(runes[:len(runes)-1])
				fmt.Fprint(a.Out, "\b \b")
			}
			return false
		default:
			for _, r := range k.Runes {
				text += string(r)
				fmt.Fprint(a.Out, string(r))
			}
			return false
		}
	})
	return text, aborted
}

// waitForEnter waits for Enter; returns true if the user pressed Escape.
func (a *App) waitForEnter() bool {
	_, aborted := a.ReadLine()
	return aborted
}

// sortedKeys returns the map keys in deterministic (sorted) order.
func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// showStats prints the final statistics and exits the program.
func (a *App) showStats(data map[string]string, known, unknown map[string]bool) {
	rest := make(map[string]bool)
	for k := range data {
		if !known[k] && !unknown[k] {
			rest[k] = true
		}
	}
	fmt.Fprint(a.Out, FormatStats(data, sortedKeys(known), sortedKeys(unknown), sortedKeys(rest)))
	a.Exit(0)
}

// RunLearningHiragana runs the hiragana quiz over the given data set. Kana are
// asked in sorted order; previously wrong kana are repeated at the end. The
// flow ends when everything is answered correctly or the user quits with
// Escape, which shows the statistics.
func (a *App) RunLearningHiragana(data map[string]string) {
	known := map[string]bool{}
	unknown := map[string]bool{}

	// Ask each kana once per pass; wrong answers are collected into unknown and
	// re-asked after the initial pass completes.
	order := sortedStringMap(data)

	ask := func(k string) bool { // returns true if the caller should quit
		a.Clean()
		if unknown[k] {
			fmt.Fprintln(a.Out, "You didn't know this one before. Try again!")
			fmt.Fprintln(a.Out)
		}
		fmt.Fprintln(a.Out, k)
		fmt.Fprintln(a.Out)
		a.printLearningHiraganaMenu()
		fmt.Fprint(a.Out, "> ")

		input, aborted := a.ReadLine()
		if aborted {
			return true
		}

		answer := strings.ToLower(strings.TrimSpace(input))
		a.Clean()
		if answer == data[k] {
			known[k] = true
			delete(unknown, k)
			fmt.Fprintf(a.Out, "Correct! You answered \"%s\".\n", answer)
		} else {
			unknown[k] = true
			if answer == "" {
				fmt.Fprintln(a.Out, "Wrong. You didn't type anything.")
			} else {
				fmt.Fprintf(a.Out, "Wrong. You answered \"%s\", but that is incorrect.\n", answer)
			}
		}
		fmt.Fprintf(a.Out, "Correct answer: %s - %s\n", k, data[k])
		fmt.Fprintln(a.Out, "\nPress Enter to continue... (Esc shows statistics)")
		return a.waitForEnter()
	}

	for _, k := range order {
		if known[k] {
			continue
		}
		if ask(k) {
			a.showStats(data, known, unknown)
			return
		}
	}
	// Repeatedly re-ask everything that is still unknown.
	for len(unknown) > 0 {
		for _, k := range order {
			if known[k] {
				continue
			}
			if ask(k) {
				a.showStats(data, known, unknown)
				return
			}
		}
	}
	a.showStats(data, known, unknown)
}

// sortedStringMap returns the keys of m sorted.
func sortedStringMap(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
