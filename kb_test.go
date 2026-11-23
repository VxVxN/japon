package main

import (
	"fmt"
	"strings"
	"testing"

	"atomicgo.dev/keyboard/keys"
)

// flattenKeys concatenates a list of key sequences and single keys into one
// flat sequence, e.g. flattenKeys(runeKeys("a"), enterKey()).
func flattenKeys(items ...interface{}) []keys.Key {
	var out []keys.Key
	for _, it := range items {
		switch v := it.(type) {
		case keys.Key:
			out = append(out, v)
		case []keys.Key:
			out = append(out, v...)
		case string:
			out = append(out, textKeys(v)...)
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
	a, buf, listener, exits, navs := newTestApp(t, digitKey('1'), digitKey('1'), digitKey('1'))
	a.runLearningPhrases(testPhrases)

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
	a, buf, _, exits, _ := newTestApp(t, digitKey('2'), digitKey('1'), digitKey('1'))
	a.runLearningPhrases(testPhrases)

	if !strings.Contains(buf.String(), "あ - A") {
		t.Errorf("translation line missing:\n%s", buf.String())
	}
	if len(*exits) != 0 {
		t.Errorf("unexpected exit: %v", *exits)
	}
}

func TestRunLearningPhrasesBackToMain(t *testing.T) {
	a, _, _, exits, navs := newTestApp(t, digitKey('3'))
	a.runLearningPhrases(testPhrases)

	if len(*navs) != 1 || (*navs)[0] != destMainMenu {
		t.Fatalf("expected navigation to main menu, got %v", *navs)
	}
	if len(*exits) != 0 {
		t.Errorf("going back must not exit: %v", *exits)
	}
}

func TestRunLearningPhrasesExitOnZero(t *testing.T) {
	a, _, listener, exits, _ := newTestApp(t, digitKey('0'), digitKey('1'))
	a.runLearningPhrases(testPhrases)

	if len(*exits) != 1 || (*exits)[0] != 0 {
		t.Fatalf("expected exit(0), got %v", *exits)
	}
	if len(listener.Consumed) != 1 {
		t.Errorf("flow must stop after exit; consumed=%d", len(listener.Consumed))
	}
}

func TestRunLearningPhrasesEscapeExits(t *testing.T) {
	a, _, _, exits, navs := newTestApp(t, escKey())
	a.runLearningPhrases(testPhrases)

	if len(*exits) != 1 || (*exits)[0] != 0 {
		t.Fatalf("Escape should exit(0), got %v", *exits)
	}
	if len(*navs) != 0 {
		t.Errorf("Escape must not navigate; got %v", *navs)
	}
}

func TestRunLearningPhrasesUnknownKeyRedrawsMenu(t *testing.T) {
	a, buf, listener, _, _ := newTestApp(t, runeKey('z'), digitKey('1'), digitKey('1'), digitKey('1'))
	a.runLearningPhrases(testPhrases)

	if len(listener.Consumed) != 4 {
		t.Fatalf("unknown key must not consume the selection; consumed=%d", len(listener.Consumed))
	}
	// Menu drawn once per phrase plus one redraw after the unknown key.
	if got := strings.Count(buf.String(), "1: I can translate"); got != 4 {
		t.Errorf("menu draws = %d, want 4:\n%s", got, buf.String())
	}
}

func TestNavigateToDispatches(t *testing.T) {
	defer func(saved *app) { currentApp = saved }(currentApp)

	var ran string
	listener := newFakeKeyListener(digitKey('0')) // any key ends the fake flows quickly
	currentApp = &app{
		out:      &strings.Builder{},
		listener: listener,
		clean:    func() {},
		exit:     func(int) {},
		navigateTo: func(name string) {
			// Intercept navigation so we can observe dispatch without recursion.
			ran = name
			listener.Keys = nil
		},
	}

	// destPhrases -> runLearningPhrases sees digit '1'... but our app's
	// navigateTo is what the flows call; instead verify dispatch by checking
	// that each destination runs a flow that consumes at least one key.
	for _, dest := range []string{destPhrases, destHiragana, destMainMenu} {
		listener.Consumed = nil
		listener.Keys = []keys.Key{escKey()}
		navigateTo(dest)
		if len(listener.Consumed) == 0 {
			t.Errorf("navigateTo(%q) did not run any flow", dest)
		}
	}
	_ = ran
}
