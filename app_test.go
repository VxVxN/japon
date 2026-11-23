package main

import (
	"bytes"
	"strings"
	"testing"

	"atomicgo.dev/keyboard/keys"
)

// newTestApp builds an app wired to fakes. All state changes are recorded so
// tests can assert on them without touching a real terminal.
func newTestApp(t *testing.T, ks ...keys.Key) (*app, *bytes.Buffer, *fakeKeyListener, *[]int, *[]string) {
	t.Helper()
	buf := &bytes.Buffer{}
	listener := newFakeKeyListener(ks...)
	var exits []int
	var navs []string
	a := &app{
		out:      buf,
		listener: listener,
		clean:    func() {},
		exit:     func(code int) { exits = append(exits, code) },
		navigateTo: func(name string) {
			navs = append(navs, name)
			// stop the flow that navigated away, like the real one-shot menus do
			listener.Keys = nil
		},
	}
	return a, buf, listener, &exits, &navs
}

func TestClassifyMenuKey(t *testing.T) {
	tests := []struct {
		name string
		key  keys.Key
		want string
	}{
		{"escape quits", escKey(), destQuit},
		{"zero quits", digitKey('0'), destQuit},
		{"one opens phrases", digitKey('1'), destPhrases},
		{"two opens hiragana", digitKey('2'), destHiragana},
		{"three is not in menu", digitKey('3'), destUnchanged},
		{"nine is not in menu", digitKey('9'), destUnchanged},
		{"letter unchanged", runeKey('a'), destUnchanged},
		{"enter unchanged", enterKey(), destUnchanged},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyMenuKey(tt.key); got != tt.want {
				t.Errorf("classifyMenuKey(%v) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

// The mapping must match the printed main menu text.
func TestClassifyMenuKeyMatchesPrintedMenu(t *testing.T) {
	a, buf, _, _, _ := newTestApp(t)
	a.printMainMenu()
	out := buf.String()

	wants := map[string]string{
		"1: Learning phrases":  destPhrases,
		"2: Learning hiragana": destHiragana,
		"0: Exit":              destQuit,
	}
	for line, dest := range wants {
		if !strings.Contains(out, line) {
			t.Fatalf("menu output missing line %q; got:\n%s", line, out)
		}
		digit := line[0] - '0' // '1', '2', '0'
		if got := classifyMenuKey(digitKey(digit)); got != dest {
			t.Errorf("digit %q classified as %q, but menu says %q -> %q", digit, got, line, dest)
		}
	}
}

func TestFormatStats(t *testing.T) {
	data := map[string]string{"あ": "a", "い": "i", "う": "u"}
	got := formatStats(data,
		[]string{"い", "あ"}, // deliberately unsorted input
		[]string{"う"},
		nil,
	)
	want := "Statistics (2/3 answered correctly):\n\n" +
		"Known hiragana:\n" +
		"  あ - a\n" +
		"  い - i\n" +
		"\nNot known / not answered hiragana:\n" +
		"  う - u\n"
	if got != want {
		t.Errorf("formatStats() =\n%q\nwant\n%q", got, want)
	}
}

func TestFormatStatsEmptyKnown(t *testing.T) {
	data := map[string]string{"あ": "a"}
	got := formatStats(data, nil, nil, []string{"あ"})
	if !strings.Contains(got, "Statistics (0/1 answered correctly)") {
		t.Errorf("header wrong:\n%s", got)
	}
	if !strings.Contains(got, "Known hiragana:\n  none") {
		t.Errorf("empty known section should print \"none\":\n%s", got)
	}
	if !strings.Contains(got, "  あ - a") {
		t.Errorf("rest key missing from not-known section:\n%s", got)
	}
}

func TestWriteKeyListNone(t *testing.T) {
	w := &strings.Builder{}
	writeKeyList(w, map[string]string{}, nil)
	if got := w.String(); got != "  none\n" {
		t.Errorf("writeKeyList(empty) = %q, want %q", got, "  none\n")
	}
}

func TestSortedHelpers(t *testing.T) {
	m := map[string]bool{"b": true, "a": true, "c": true}
	if got := strings.Join(sortedKeys(m), ","); got != "a,b,c" {
		t.Errorf("sortedKeys = %s", got)
	}
	s := map[string]string{"z": "1", "y": "2"}
	if got := strings.Join(sortedBoolMap(s), ","); got != "y,z" {
		t.Errorf("sortedBoolMap = %s", got)
	}
	if got := strings.Join(sortedPhraseKeys(s), ","); got != "y,z" {
		t.Errorf("sortedPhraseKeys = %s", got)
	}
}

func TestMustNoErr(t *testing.T) {
	mustNoErr(nil) // must not panic
	mustNoErr(errFake)
}

type fakeErr struct{}

func (fakeErr) Error() string { return "fake" }

var errFake = fakeErr{}
