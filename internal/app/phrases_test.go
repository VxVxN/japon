package app

import (
	"fmt"
	"strings"
	"testing"
	"test/internal/data"
	"test/internal/kbd"

	"atomicgo.dev/keyboard/keys"
)

// flattenKeys concatenates a list of key sequences and single keys into one
// flat sequence, e.g. flattenKeys(kbd.TextKeys("a"), kbd.EnterKey()).
func flattenKeys(items ...interface{}) []keys.Key {
	var out []keys.Key
	for _, it := range items {
		switch v := it.(type) {
		case keys.Key:
			out = append(out, v)
		case []keys.Key:
			out = append(out, v...)
		case string:
			out = append(out, kbd.TextKeys(v)...)
		case []interface{}:
			out = append(out, flattenKeys(v...)...)
		default:
			panic(fmt.Sprintf("flattenKeys: unsupported argument type %T (%v)", it, it))
		}
	}
	return out
}

var testPhrases = map[string]string{
	"あ": "A",
	"い": "I",
	"う": "U",
}

func TestRunLearningPhrasesCanTranslate(t *testing.T) {
	// "1" for every phrase: flow finishes without exiting or navigating.
	a, buf, listener, exits, navs := newTestApp(t, kbd.DigitKey('1'), kbd.DigitKey('1'), kbd.DigitKey('1'))
	a.RunLearningPhrases(testPhrases)

	if len(*exits) != 0 {
		t.Errorf("unexpected exit: %v", *exits)
	}
	if len(*navs) != 0 {
		t.Errorf("unexpected navigation: %v", *navs)
	}
	if len(listener.Consumed) != 3 {
		t.Errorf("expected 3 keys consumed, got %d", len(listener.Consumed))
	}
	for k := range testPhrases {
		if !strings.Contains(buf.String(), k) {
			t.Errorf("phrase %q was not shown:\n%s", k, buf.String())
		}
	}
}

func TestRunLearningPhrasesShowTranslation(t *testing.T) {
	// First phrase is あ (sorted order); "2" shows its translation while the
	// menu stays in place, then "1" moves on to the next phrase.
	a, buf, listener, exits, _ := newTestApp(t, kbd.DigitKey('2'), kbd.DigitKey('1'), kbd.DigitKey('1'), kbd.DigitKey('1'))
	a.RunLearningPhrases(testPhrases)

	if !strings.Contains(buf.String(), "あ - A") {
		t.Errorf("translation line missing:\n%s", buf.String())
	}
	if len(*exits) != 0 {
		t.Errorf("unexpected exit: %v", *exits)
	}
	// All three phrases are completed with "1"; "2" only reveals the
	// translation and keeps listening on the same menu. Phrases appear in
	// random order, so we only assert at least the four selections happened.
	if len(listener.Consumed) < 4 {
		t.Errorf("expected at least 4 keys consumed, got %d", len(listener.Consumed))
	}
	// The menu must not be redrawn by "2": one draw per phrase only.
	if got := strings.Count(buf.String(), "1: I can translate"); got != 3 {
		t.Errorf("menu draws = %d, want 3 (menu stays in place):\n%s", got, buf.String())
	}
}

func TestRunLearningPhrasesCantTranslate(t *testing.T) {
	// "3" means "I can't translate": the phrase is skipped for now but stays
	// in the rotation, so after the fake keys run out the flow keeps going
	// and finally returns to the main menu without exiting.
	a, _, listener, exits, navs := newTestApp(t, kbd.DigitKey('3'), kbd.DigitKey('3'), kbd.DigitKey('3'))
	a.RunLearningPhrases(testPhrases)

	if len(listener.Consumed) < 3 {
		t.Errorf("expected at least 3 keys consumed, got %d", len(listener.Consumed))
	}
	if len(*exits) != 0 {
		t.Errorf("unexpected exit: %v", *exits)
	}
	if len(*navs) == 0 || (*navs)[len(*navs)-1] != DestMainMenu {
		t.Errorf("expected return to main menu, got %v", *navs)
	}
}

func TestRunLearningPhrasesBackToMain(t *testing.T) {
	a, _, _, exits, navs := newTestApp(t, kbd.DigitKey('4'))
	a.RunLearningPhrases(testPhrases)

	if len(*navs) != 1 || (*navs)[0] != DestMainMenu {
		t.Fatalf("expected navigation to main menu, got %v", *navs)
	}
	if len(*exits) != 0 {
		t.Errorf("going back must not exit: %v", *exits)
	}
}

func TestRunLearningPhrasesExitOnZero(t *testing.T) {
	a, _, listener, exits, _ := newTestApp(t, kbd.DigitKey('0'), kbd.DigitKey('1'))
	a.RunLearningPhrases(testPhrases)

	if len(*exits) != 1 || (*exits)[0] != 0 {
		t.Fatalf("expected exit(0), got %v", *exits)
	}
	if len(listener.Consumed) != 1 {
		t.Errorf("flow must stop after exit; consumed=%d", len(listener.Consumed))
	}
}

func TestRunLearningPhrasesEscapeExits(t *testing.T) {
	a, _, _, exits, navs := newTestApp(t, kbd.EscKey())
	a.RunLearningPhrases(testPhrases)

	if len(*exits) != 1 || (*exits)[0] != 0 {
		t.Fatalf("Escape should exit(0), got %v", *exits)
	}
	if len(*navs) != 0 {
		t.Errorf("Escape must not navigate; got %v", *navs)
	}
}

func TestRunLearningPhrasesUnknownKeyRedrawsMenu(t *testing.T) {
	a, buf, listener, _, _ := newTestApp(t, kbd.RuneKey('z'), kbd.DigitKey('1'), kbd.DigitKey('1'), kbd.DigitKey('1'))
	a.RunLearningPhrases(testPhrases)

	if len(listener.Consumed) < 4 {
		t.Fatalf("expected all seeded keys consumed; consumed=%d", len(listener.Consumed))
	}
	// Menu drawn once per phrase plus one redraw after the unknown key.
	if got := strings.Count(buf.String(), "1: I can translate"); got < 4 {
		t.Errorf("menu draws = %d, want at least 4:\n%s", got, buf.String())
	}
}

func TestRunLearningPhrasesRandomOrder(t *testing.T) {
	// With a larger set, the phrases should not always be shown in sorted
	// order across repeated runs.
	phrases := map[string]string{}
	for i := 0; i < 15; i++ {
		k := string(rune('ぁ' + i))
		phrases[k] = fmt.Sprintf("t%d", i)
	}
	sorted := SortedPhraseKeys(phrases)
	differs := false
	for run := 0; run < 10 && !differs; run++ {
		a, buf, _, _, _ := newTestApp(t, flattenKeys(strings.Repeat("1", 15))...)
		a.RunLearningPhrases(phrases)
		out := buf.String()
		last := -1
		inOrder := true
		for _, k := range sorted {
			idx := strings.Index(out, k+"\n")
			if idx < 0 {
				inOrder = false
				break
			}
			if idx < last {
				inOrder = false
				break
			}
			last = idx
		}
		if !inOrder {
			differs = true
		}
	}
	if !differs {
		t.Errorf("phrases always shown in sorted order")
	}
}

func TestRunLearningPhrasesSkipsKnownAndPersists(t *testing.T) {
	// "あ" was already learned in a previous session: it must not be shown,
	// and confirming the remaining two phrases must persist them as known.
	progress := ProgressData{KnownPhrases: []string{"あ"}}
	a, buf, _, _, _ := newTestAppWithProgress(t, progress, kbd.DigitKey('1'), kbd.DigitKey('1'))
	a.RunLearningPhrases(testPhrases)

	if strings.Contains(buf.String(), "1: I can translate\n2: Show translation\n3: I can't translate\n4: Go back to the main menu\n0: Exit\nあ\n") {
		t.Errorf("known phrase should not be shown:\n%s", buf.String())
	}
	got := strings.Join(sortedStrings(a.LoadProgress().KnownPhrases), ",")
	if got != "い,う" {
		t.Errorf("persisted known phrases = %q, want \"い,う\"", got)
	}
}

func TestRunLearningPhrasesAllKnownReturnsToMenu(t *testing.T) {
	progress := ProgressData{KnownPhrases: []string{"あ", "い", "う"}}
	a, buf, _, _, navs := newTestAppWithProgress(t, progress, kbd.EnterKey())
	a.RunLearningPhrases(testPhrases)

	if !strings.Contains(buf.String(), "You already know all the phrases!") {
		t.Errorf("completion message missing:\n%s", buf.String())
	}
	if len(*navs) != 1 || (*navs)[0] != DestMainMenu {
		t.Errorf("expected navigation to main menu, got %v", *navs)
	}
}

func sortedStrings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func TestAllPhrasesMerged(t *testing.T) {
	all := data.AllPhrases()
	cats := []map[string]string{data.FoodPhrases, data.CommonPhrases, data.ClothesPhrases, data.EtcPhrases}
	total := 0
	for _, m := range cats {
		total += len(m)
	}
	if len(all) != total {
		t.Errorf("AllPhrases lost entries: got %d, want %d", len(all), total)
	}
	for _, m := range cats {
		for k, v := range m {
			if all[k] != v {
				t.Errorf("phrase %q missing or wrong in AllPhrases", k)
			}
		}
	}
}

func TestHiraganaMapSanity(t *testing.T) {
	// Basic syllabary spot checks and no empty readings.
	for _, k := range []string{"あ", "か", "さ", "た", "な", "は", "ま", "や", "ら", "わ", "ん", "が", "ぱ"} {
		if data.Hiragana[k] == "" {
			t.Errorf("data.Hiragana[%q] has no reading", k)
		}
	}
	if len(data.Hiragana) < 70 {
		t.Errorf("data.Hiragana unexpectedly small: %d", len(data.Hiragana))
	}
}
