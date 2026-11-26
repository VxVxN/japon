package app

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"unicode"

	"atomicgo.dev/keyboard/keys"
	"test/internal/kbd"
)

// printLearningKanaMenu prints the help text of the kana exercise. It is
// shared by the hiragana and katakana flows.
func (a *App) printLearningKanaMenu() {
	fmt.Fprintln(a.Out, "Type the reading in English letters and press Enter to check.")
	fmt.Fprintln(a.Out, "Press Ctrl+C or Esc to exit and see your statistics.")
}

// printLearningHiraganaMenu prints the help text of the hiragana exercise.
func (a *App) printLearningHiraganaMenu() {
	a.printLearningKanaMenu()
}

// ReadLine collects typed characters until Enter is pressed. Backspace removes
// the last character and Escape aborts the input. It returns the raw text and
// whether the user aborted with Escape. When the key source runs out of events
// in the middle of the line - before any terminating key arrives (exhausted
// fake in tests, closed web session) - the input is treated as aborted too,
// so that callers can end the current interaction instead of looping forever
// on unanswered prompts. A line that was properly terminated by Enter stays
// non-aborted even when it consumed the very last key of the sequence; flows
// detect that situation themselves via sourceSpent after the read.
func (a *App) ReadLine() (text string, aborted bool) {
	stoppedOnKey := false
	var answered string
	_ = a.Listener.Listen(func(k keys.Key) bool {
		switch k.Code {
		case keys.Escape:
			aborted = true
			stoppedOnKey = true
			return true
		case keys.Enter:
			stoppedOnKey = true
			answered = text
			return true
		case keys.Backspace:
			if len(text) > 0 {
				runes := []rune(text)
				text = string(runes[:len(runes)-1])
				fmt.Fprint(a.Out, "\b \b")
			}
			return false
		default:
			// Only actually typed characters count as input. Control keys
			// (e.g. a lone Backspace on empty input) must not stop the
			// listener and must not be mistaken for an answer.
			if len(k.Runes) == 0 {
				return false
			}
			for _, r := range k.Runes {
				text += string(r)
				fmt.Fprint(a.Out, string(r))
			}
			return false
		}
	})
	if !stoppedOnKey && !aborted && sourceQuiet(a.Listener) {
		// Listen returned without Enter/Escape because the key source ran
		// out of events mid-line; report an abort so flows terminate
		// gracefully instead of re-asking the same question forever.
		aborted = true
	}
	if a.answerSink != nil {
		*a.answerSink = &answered
	}
	return text, aborted
}

// answerSink, when set by setAnswerSink, receives a pointer to the finished
// line (the text typed before the terminating Enter) after every ReadLine
// call, even when that read was reported as aborted because the key source
// ran dry right with the terminating key. Interactive flows use it to still
// process the last answer of such a session (see runKanaQuiz).
func (a *App) setAnswerSink(sink **string) func() {
	prev := a.answerSink
	a.answerSink = sink
	return func() { a.answerSink = prev }
}

// sourceQuiet reports whether l is a key source that has run dry: it either
// exposes IsExhausted directly (kbd.Fake, the web session listener) or wraps
// one (kbd.Watch). A listener without such an interface (the real terminal)
// is never considered quiet.
func sourceQuiet(l kbd.Listener) bool {
	if q, ok := l.(interface{ IsExhausted() bool }); ok {
		return q.IsExhausted()
	}
	if w, ok := l.(interface{ Inner() kbd.Listener }); ok {
		return sourceQuiet(w.Inner())
	}
	return false
}

// sourceSpent reports whether the key source can no longer deliver any key
// event to a subsequent Listen call. Unlike sourceQuiet this also covers the
// case where the previous interaction stopped on the very last key of the
// sequence (a real Enter/Escape answer that happened to be the final event):
// after that moment the source is spent as well and flows must end the
// current interaction instead of looping forever on unanswered prompts.
func sourceSpent(l kbd.Listener) bool {
	if p, ok := l.(interface{ HasPendingKeys() bool }); ok {
		return !p.HasPendingKeys()
	}
	if w, ok := l.(interface{ Inner() kbd.Listener }); ok {
		return sourceSpent(w.Inner())
	}
	// Listeners without a pending-keys probe (the real terminal, live web
	// sessions) are assumed to keep delivering events.
	return false
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
	a.showStatsScript("hiragana", data, known, unknown)
}

// showStatsScript is like showStats but lets the caller choose the wording of
// the "Known ..." section header ("hiragana" or "katakana").
func (a *App) showStatsScript(scriptName string, data map[string]string, known, unknown map[string]bool) {
	rest := make(map[string]bool)
	for k := range data {
		if !known[k] && !unknown[k] {
			rest[k] = true
		}
	}
	fmt.Fprint(a.Out, FormatStatsScript(scriptName, data, sortedKeys(known), sortedKeys(unknown), sortedKeys(rest)))
	a.Exit(0)
}

// shuffledStringMap returns the keys of m in random order.
func shuffledStringMap(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	rand.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

// RunLearningHiragana runs the hiragana quiz over the given data set. Kana are
// asked in random order; previously wrong kana are repeated at the end. The
// flow ends when everything is answered correctly or the user quits with
// Escape, which shows the statistics.
//
// Progress is persisted: kana that were already known from previous sessions
// (loaded via LoadProgress) are skipped, and every newly known kana is saved
// back to the progress file as soon as it is answered correctly.
func (a *App) RunLearningHiragana(data map[string]string) {
	a.runKanaQuiz("hiragana", data,
		func() []string { return a.LoadProgress().KnownHiragana },
		func(list []string) error {
			return a.SaveProgress(ProgressData{KnownHiragana: list})
		})
}

// runKanaQuiz is the shared quiz engine behind the hiragana and katakana
// flows. scriptName is used for wording ("hiragana"/"katakana"), loadKnown
// restores previously learned kana keys and save persists the current set of
// known kana after every correct answer.
func (a *App) runKanaQuiz(scriptName string, data map[string]string, loadKnown func() []string, save func(list []string) error) {
	known := map[string]bool{}
	unknown := map[string]bool{}

	// Restore previously learned kana so they are not asked again. Only keys
	// that exist in the current data set are trusted.
	for _, k := range loadKnown() {
		if _, ok := data[k]; ok {
			known[k] = true
		}
	}

	persist := func() {
		_ = save(sortedKeys(known))
	}

	// Ask each kana once per pass; wrong answers are collected into unknown and
	// re-asked after the initial pass completes. The order is randomized so the
	// quiz does not simply follow the syllabary table.
	order := shuffledStringMap(data)

	// quiet reports whether the key source ran dry (e.g. an exhausted fake
	// listener in tests or a closed web session). When it is true the flows
	// terminate instead of spinning forever on unanswered questions. Note:
	// Fake.Exhausted only means "the sequence was consumed", not that the
	// current interaction received an answer, so ReadLine must have actually
	// returned from a key event for the source to count as quiet.
	keySeen := false
	inner := a.Listener
	a.Listener = kbd.Watch(inner, &keySeen)
	defer func() { a.Listener = inner }()
	quiet := func() bool {
		if !keySeen {
			return false
		}
		return sourceQuiet(inner)
	}

	// answered receives the raw input of every finished ReadLine line (Enter
	// terminated) while the flow is running. It lets ask() process the last
	// answer of a sequence that ends right with the terminating Enter, even
	// though the exhaustion check immediately after ReadLine marks such a
	// read as aborted.
	var answered *string
	restoreListener := a.setAnswerSink(&answered)
	defer restoreListener()

	ask := func(k string) bool { // returns true if the caller should quit
		a.Clean()
		if unknown[k] {
			fmt.Fprintln(a.Out, "You didn't know this one before. Try again!")
			fmt.Fprintln(a.Out)
		}
		fmt.Fprintln(a.Out, k)
		fmt.Fprintln(a.Out)
		a.printLearningKanaMenu()
		fmt.Fprint(a.Out, "> ")

		input, aborted := a.ReadLine()
		if aborted && answered != nil && *answered == "" {
			// The line ended at the very moment the source ran dry (the
			// terminating Enter was its last event): take over the finished
			// answer, process it below, and end the quiz afterwards instead
			// of looping forever on unanswered questions.
			input, aborted = *answered, false
		}
		if aborted || quiet() {
			return true
		}

		answer := strings.ToLower(strings.TrimSpace(input))
		a.Clean()
		if answer == data[k] {
			known[k] = true
			delete(unknown, k)
			persist()
			fmt.Fprintf(a.Out, "Correct! You answered \"%s\".\n", answer)
		} else {
			unknown[k] = true
			if answer == "" {
				fmt.Fprintln(a.Out, "Wrong. You didn't type anything.")
			} else {
				fmt.Fprintf(a.Out, "Wrong. You answered \"%s\", but that is incorrect.\n", answer)
				fmt.Fprintf(a.Out, "Correct answer: %s - %s\n", k, data[k])
			}
		}
		if sourceSpent(a.Listener) {
			// No further keys can arrive; stop after processing this answer.
			return true
		}
		fmt.Fprintln(a.Out, "\nPress Enter to continue... (Esc shows statistics)")
		return a.waitForEnter()
	}

	for _, k := range order {
		if known[k] {
			continue
		}
		if ask(k) {
			a.showStatsScript(scriptName, data, known, unknown)
			return
		}
		if quiet() {
			a.showStatsScript(scriptName, data, known, unknown)
			return
		}
	}
	// Repeatedly re-ask everything that is still unknown. The quiet() guard
	// makes the loop terminate when the key source runs dry (closed web
	// session or exhausted fake listener in tests).
	for len(unknown) > 0 {
		for _, k := range order {
			if known[k] {
				continue
			}
			if ask(k) {
				a.showStatsScript(scriptName, data, known, unknown)
				return
			}
			if quiet() {
				a.showStatsScript(scriptName, data, known, unknown)
				return
			}
		}
	}
	a.showStatsScript(scriptName, data, known, unknown)
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

// RunLearningKatakana runs the katakana quiz over the given data set. It is
// the katakana counterpart of RunLearningHiragana and shares the same quiz
// engine: random order, repeated wrong answers, persisted progress in
// KnownKatakana and statistics on Escape.
func (a *App) RunLearningKatakana(data map[string]string) {
	a.runKanaQuiz("katakana", data,
		func() []string { return a.LoadProgress().KnownKatakana },
		func(list []string) error {
			p := a.LoadProgress()
			p.KnownKatakana = list
			return a.SaveProgress(p)
		})
}

// RunResetHiraganaProgress asks for confirmation and then clears only the
// persisted hiragana learning progress, returning to the main menu afterwards.
func (a *App) RunResetHiraganaProgress() {
	a.runReset("hiragana", "all kana you have learned so far will be asked again.", func() error {
		if a.ResetHiragana != nil {
			return a.ResetHiragana()
		}
		return ResetHiraganaProgress()
	})
}

// RunResetKatakanaProgress asks for confirmation and then clears only the
// persisted katakana learning progress, returning to the main menu afterwards.
func (a *App) RunResetKatakanaProgress() {
	a.runReset("katakana", "all kana you have learned so far will be asked again.", func() error {
		if a.ResetKatakana != nil {
			return a.ResetKatakana()
		}
		return ResetKatakanaProgress()
	})
}

// RunResetPhrasesProgress asks for confirmation and then clears only the
// persisted phrases learning progress, returning to the main menu afterwards.
func (a *App) RunResetPhrasesProgress() {
	a.runReset("phrases", "all phrases you have learned so far will be shown again.", func() error {
		if a.ResetPhrasesProgress != nil {
			return a.ResetPhrasesProgress()
		}
		return ResetPhrasesProgress()
	})
}

// runReset shows the shared confirmation prompt for a scoped progress reset
// and invokes the given reset function once the user confirms.
func (a *App) runReset(what, detail string, reset func() error) {
	a.Clean()
	fmt.Fprintf(a.Out, "This will delete your saved %s progress:\n", what)
	fmt.Fprintln(a.Out, detail)
	fmt.Fprintln(a.Out)
	fmt.Fprintln(a.Out, "Press Enter to confirm the reset, Esc to cancel.")
	fmt.Fprint(a.Out, "> ")

	if a.waitForEnter() {
		// Cancelled: go back to the main menu without touching anything.
		a.NavigateTo(DestMainMenu)
		return
	}

	a.Clean()
	if err := reset(); err != nil {
		fmt.Fprintf(a.Out, "Failed to reset progress: %v\n", err)
	} else {
		fmt.Fprintf(a.Out, "%s progress has been reset.\n", capitalize(what))
	}
	fmt.Fprintln(a.Out, "\nPress Enter to return to the menu...")
	a.waitForEnter()
	a.NavigateTo(DestMainMenu)
}

// capitalize upper-cases the first rune of s.
func capitalize(s string) string {
if s == "" {
return s
}
r := []rune(s)
r[0] = unicode.ToUpper(r[0])
return string(r)
}
