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
	// First phrase is あ (sorted order); "2" shows its translation.
	a, buf, _, exits, _ := newTestApp(t, kbd.DigitKey('2'), kbd.DigitKey('1'), kbd.DigitKey('1'))
	a.RunLearningPhrases(testPhrases)

	if !strings.Contains(buf.String(), "あ - A") {
		t.Errorf("translation line missing:\n%s", buf.String())
	}
	if len(*exits) != 0 {
		t.Errorf("unexpected exit: %v", *exits)
	}
}

func TestRunLearningPhrasesBackToMain(t *testing.T) {
	a, _, _, exits, navs := newTestApp(t, kbd.DigitKey('3'))
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

	if len(listener.Consumed) != 4 {
		t.Fatalf("unknown key must not consume the selection; consumed=%d", len(listener.Consumed))
	}
	// Menu drawn once per phrase plus one redraw after the unknown key.
	if got := strings.Count(buf.String(), "1: I can translate"); got != 4 {
		t.Errorf("menu draws = %d, want 4:\n%s", got, buf.String())
	}
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
