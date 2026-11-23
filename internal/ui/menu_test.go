package ui

import (
	"bytes"
	"strings"
	"testing"
	"test/internal/app"
	"test/internal/kbd"

	"atomicgo.dev/keyboard/keys"
)

// newTestApp builds an app wired to fakes, mirroring the helper in the app
// package's tests.
func newTestApp(t *testing.T, ks ...keys.Key) (*app.App, *bytes.Buffer, *kbd.Fake, *[]int, *[]string) {
	t.Helper()
	buf := &bytes.Buffer{}
	listener := kbd.NewFake(ks...)
	var exits []int
	var navs []string
	a := &app.App{
		Out:      buf,
		Listener: listener,
		Clean:    func() {},
		Exit:     func(code int) { exits = append(exits, code) },
		NavigateTo: func(name string) {
			navs = append(navs, name)
			// stop the flow that navigated away, like the real one-shot menus do
			listener.Keys = nil
		},
	}
	return a, buf, listener, &exits, &navs
}

func TestRunMainMenuPhrasesDigit(t *testing.T) {
	a, _, _, exits, navs := newTestApp(t, kbd.DigitKey('1'))
	NewMainMenu(a).Run()
	if len(*navs) != 1 || (*navs)[0] != app.DestPhrases {
		t.Fatalf("expected navigation to %q, got %v", app.DestPhrases, *navs)
	}
	if len(*exits) != 0 {
		t.Fatalf("unexpected exit calls: %v", *exits)
	}
}

func TestRunMainMenuHiraganaDigit(t *testing.T) {
	a, _, _, exits, navs := newTestApp(t, kbd.DigitKey('2'))
	NewMainMenu(a).Run()
	if len(*navs) != 1 || (*navs)[0] != app.DestHiragana {
		t.Fatalf("expected navigation to %q, got %v", app.DestHiragana, *navs)
	}
	if len(*exits) != 0 {
		t.Fatalf("unexpected exit calls: %v", *exits)
	}
}

func TestRunMainMenuExitOnZero(t *testing.T) {
	a, buf, listener, exits, navs := newTestApp(t, kbd.DigitKey('0'))
	NewMainMenu(a).Run()
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
	a, _, _, exits, _ := newTestApp(t, kbd.EscKey())
	NewMainMenu(a).Run()
	if len(*exits) != 1 || (*exits)[0] != 0 {
		t.Fatalf("Escape should exit(0), got %v", *exits)
	}
}

func TestRunMainMenuUnknownKeyRedraws(t *testing.T) {
	// Unknown key redraws the menu and keeps listening; then '1' navigates.
	a, buf, listener, _, navs := newTestApp(t, kbd.RuneKey('x'), kbd.DigitKey('1'))
	NewMainMenu(a).Run()
	if len(*navs) != 1 || (*navs)[0] != app.DestPhrases {
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

// The mapping must match the printed main menu text.
func TestMenuKeyMappingMatchesPrintedMenu(t *testing.T) {
	a, buf, _, _, _ := newTestApp(t)
	m := NewMainMenu(a)
	m.printMenu()
	out := buf.String()

	wants := map[string]string{
		"1: Learning phrases":  app.DestPhrases,
		"2: Learning hiragana": app.DestHiragana,
		"0: Exit":              app.DestQuit,
	}
	for line, dest := range wants {
		if !strings.Contains(out, line) {
			t.Fatalf("menu output missing line %q; got:\n%s", line, out)
		}
		digit := line[0] - '0' // '1', '2', '0'
		if got := app.ClassifyMenuKey(kbd.DigitKey(digit)); got != dest {
			t.Errorf("digit %q classified as %q, but menu says %q -> %q", digit, got, line, dest)
		}
	}
}
