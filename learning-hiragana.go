package main

import (
	"fmt"
	"sort"
	"strings"

	"atomicgo.dev/keyboard/keys"
)

// hiraganaMap maps every hiragana kana to its romaji reading. It includes the
// basic syllabary plus the voiced (dakuten/handakuten) series.
var hiraganaMap = map[string]string{
	"あ": "a", "い": "i", "う": "u", "え": "e", "お": "o",
	"か": "ka", "き": "ki", "く": "ku", "け": "ke", "こ": "ko",
	"さ": "sa", "し": "shi", "す": "su", "せ": "se", "そ": "so",
	"た": "ta", "ち": "chi", "つ": "tsu", "て": "te", "と": "to",
	"な": "na", "に": "ni", "ぬ": "nu", "ね": "ne", "の": "no",
	"は": "ha", "ひ": "hi", "ふ": "fu", "へ": "he", "ほ": "ho",
	"ま": "ma", "み": "mi", "む": "mu", "め": "me", "も": "mo",
	"や": "ya", "ゆ": "yu", "よ": "yo",
	"ら": "ra", "り": "ri", "る": "ru", "れ": "re", "ろ": "ro",
	"わ": "wa", "を": "wo",
	"ん": "n",
	"が": "ga", "ぎ": "gi", "ぐ": "gu", "げ": "ge", "ご": "go",
	"ざ": "za", "じ": "ji", "ず": "zu", "ぜ": "ze", "ぞ": "zo",
	"だ": "da", "ぢ": "ji", "づ": "zu", "で": "de", "ど": "do",
	"ば": "ba", "び": "bi", "ぶ": "bu", "べ": "be", "ぼ": "bo",
	"ぱ": "pa", "ぴ": "pi", "ぷ": "pu", "ぺ": "pe", "ぽ": "po",
}

// printLearningHiraganaMenu prints the help text of the hiragana exercise.
func (a *app) printLearningHiraganaMenu() {
	fmt.Fprintln(a.out, "Type the reading in English letters and press Enter to check.")
	fmt.Fprintln(a.out, "Press Ctrl+C or Esc to exit and see your statistics.")
}

// readLine collects typed characters until Enter is pressed. Backspace removes
// the last character and Escape aborts the input. It returns the raw text and
// whether the user aborted with Escape.
func (a *app) readLine() (text string, aborted bool) {
	_ = a.listener.Listen(func(k keys.Key) bool {
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
				fmt.Fprint(a.out, "\b \b")
			}
			return false
		default:
			for _, r := range k.Runes {
				text += string(r)
				fmt.Fprint(a.out, string(r))
			}
			return false
		}
	})
	return text, aborted
}

// waitForEnter waits for Enter; returns true if the user pressed Escape.
func (a *app) waitForEnter() bool {
	_, aborted := a.readLine()
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
func (a *app) showStats(data map[string]string, known, unknown map[string]bool) {
	rest := make(map[string]bool)
	for k := range data {
		if !known[k] && !unknown[k] {
			rest[k] = true
		}
	}
	fmt.Fprint(a.out, formatStats(data, sortedKeys(known), sortedKeys(unknown), sortedKeys(rest)))
	a.exit(0)
}

// runLearningHiragana runs the hiragana quiz over the given data set. Kana are
// asked in sorted order; previously wrong kana are repeated at the end. The
// flow ends when everything is answered correctly or the user quits with
// Escape, which shows the statistics.
func (a *app) runLearningHiragana(data map[string]string) {
	known := map[string]bool{}
	unknown := map[string]bool{}

	// Ask each kana once per pass; wrong answers are collected into unknown and
	// re-asked after the initial pass completes.
	order := sortedBoolMap(data)

	ask := func(k string) bool { // returns true if the caller should quit
		a.clean()
		if unknown[k] {
			fmt.Fprintln(a.out, "You didn't know this one before. Try again!")
			fmt.Fprintln(a.out)
		}
		fmt.Fprintln(a.out, k)
		fmt.Fprintln(a.out)
		a.printLearningHiraganaMenu()
		fmt.Fprint(a.out, "> ")

		input, aborted := a.readLine()
		if aborted {
			return true
		}

		answer := strings.ToLower(strings.TrimSpace(input))
		a.clean()
		if answer == data[k] {
			known[k] = true
			delete(unknown, k)
			fmt.Fprintf(a.out, "Correct! You answered \"%s\".\n", answer)
		} else {
			unknown[k] = true
			if answer == "" {
				fmt.Fprintln(a.out, "Wrong. You didn't type anything.")
			} else {
				fmt.Fprintf(a.out, "Wrong. You answered \"%s\", but that is incorrect.\n", answer)
			}
		}
		fmt.Fprintf(a.out, "Correct answer: %s - %s\n", k, data[k])
		fmt.Fprintln(a.out, "\nPress Enter to continue... (Esc shows statistics)")
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

// sortedBoolMap returns the keys of m sorted.
func sortedBoolMap(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// learningHiragana is the entry point used by the main menu.
func learningHiragana() {
	currentApp.runLearningHiragana(hiraganaMap)
}
