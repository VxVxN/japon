package main

import (
	"strings"
	"testing"
)

func TestRunMainMenuPhrasesDigit(t *testing.T) {
	a, _, _, exits, navs := newTestApp(t, digitKey('1'))
	a.runMainMenu()
	if len(*navs) != 1 || (*navs)[0] != destPhrases {
		t.Fatalf("expected navigation to %q, got %v", destPhrases, *navs)
	}
	if len(*exits) != 0 {
		t.Fatalf("unexpected exit calls: %v", *exits)
	}
}

func TestRunMainMenuHiraganaDigit(t *testing.T) {
	a, _, _, exits, navs := newTestApp(t, digitKey('2'))
	a.runMainMenu()
	if len(*navs) != 1 || (*navs)[0] != destHiragana {
		t.Fatalf("expected navigation to %q, got %v", destHiragana, *navs)
	}
	if len(*exits) != 0 {
		t.Fatalf("unexpected exit calls: %v", *exits)
	}
}

func TestRunMainMenuExitOnZero(t *testing.T) {
	a, buf, listener, exits, navs := newTestApp(t, digitKey('0'))
	a.runMainMenu()
	if len(*exits) != 1 || (*exits)[0] != 0 {
		t.Fatalf("expected a single exit(0), got %v", *exits)
	}
	if len(*navs) != 0 {
		t.Fatalf("unexpected navigation: %v", *navs)
	}
	if !strings.Contains(buf.String(), "0: Exit") {
		t.Errorf("menu text missing:\n%s", buf.String())
	}
	if len(listener.Consumed) != 1 {
		t.Errorf("expected exactly one key consumed, got %d", len(listener.Consumed))
	}
}

func TestRunMainMenuEscapeQuits(t *testing.T) {
	a, _, _, exits, _ := newTestApp(t, escKey())
	a.runMainMenu()
	if len(*exits) != 1 || (*exits)[0] != 0 {
		t.Fatalf("Escape should exit(0), got %v", *exits)
	}
}

func TestRunMainMenuUnknownKeyRedraws(t *testing.T) {
	// Unknown key redraws the menu and keeps listening; then '1' navigates.
	a, buf, listener, _, navs := newTestApp(t, runeKey('x'), digitKey('1'))
	a.runMainMenu()
	if len(*navs) != 1 || (*navs)[0] != destPhrases {
		t.Fatalf("expected navigation to phrases after unknown key, got %v", *navs)
	}
	if len(listener.Consumed) != 2 {
		t.Fatalf("expected both keys consumed, got %d", len(listener.Consumed))
	}
	// Menu printed once initially and once again after the unknown key.
	if got := strings.Count(buf.String(), "1: Learning phrases"); got != 2 {
		t.Errorf("menu expected to be drawn twice, got %d times:\n%s", got, buf.String())
	}
}

func TestAllPhrasesMerged(t *testing.T) {
	all := allPhrases()
	if len(all) != len(foodPhrases)+len(commonPhrases)+len(clothesPhrases)+len(etcPhrases) {
		t.Errorf("allPhrases lost entries: got %d, want %d",
			len(all), len(foodPhrases)+len(commonPhrases)+len(clothesPhrases)+len(etcPhrases))
	}
	for _, m := range []map[string]string{foodPhrases, commonPhrases, clothesPhrases, etcPhrases} {
		for k, v := range m {
			if all[k] != v {
				t.Errorf("phrase %q missing or wrong in allPhrases", k)
			}
		}
	}
}

func TestHiraganaMapSanity(t *testing.T) {
	// Basic syllabary spot checks and no empty readings.
	for _, k := range []string{"あ", "か", "さ", "た", "な", "は", "ま", "や", "ら", "わ", "ん", "が", "ぱ"} {
		if hiraganaMap[k] == "" {
			t.Errorf("hiraganaMap[%q] has no reading", k)
		}
	}
	if len(hiraganaMap) < 70 {
		t.Errorf("hiraganaMap unexpectedly small: %d", len(hiraganaMap))
	}
}
