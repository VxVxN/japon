package app

import (
	"fmt"
	"bytes"
	"strings"
	"testing"
	"test/internal/kbd"

	"atomicgo.dev/keyboard/keys"
)

// newTestApp builds an App wired to fakes. All state changes are recorded so
// tests can assert on them without touching a real terminal.
func newTestApp(t *testing.T, ks ...keys.Key) (*App, *bytes.Buffer, *kbd.Fake, *[]int, *[]string) {
	t.Helper()
	buf := &bytes.Buffer{}
	listener := kbd.NewFake(ks...)
	var exits []int
	var navs []string
	a := &App{
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

func TestClassifyMenuKey(t *testing.T) {
	tests := []struct {
		name string
		key  keys.Key
		want string
	}{
		{"escape quits", kbd.EscKey(), DestQuit},
		{"zero quits", kbd.DigitKey('0'), DestQuit},
		{"one opens phrases", kbd.DigitKey('1'), DestPhrases},
		{"two opens hiragana", kbd.DigitKey('2'), DestHiragana},
		{"three is not in menu", kbd.DigitKey('3'), DestUnchanged},
		{"nine is not in menu", kbd.DigitKey('9'), DestUnchanged},
		{"letter unchanged", kbd.RuneKey('a'), DestUnchanged},
		{"enter unchanged", kbd.EnterKey(), DestUnchanged},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyMenuKey(tt.key); got != tt.want {
				t.Errorf("ClassifyMenuKey(%v) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestFormatStats(t *testing.T) {
	data := map[string]string{"あ": "a", "い": "i", "う": "u"}
	got := FormatStats(data,
		[]string{"い", "あ"}, // deliberately unsorted input
		[]string{"う"},
		nil,
	)
	want := "Statistics (2/3 answered correctly):\n\n" +
		"Known hiragana:\n" +
		"  あ - a | い - i\n" +
		"\nNot known / not answered hiragana:\n" +
		"  う - u\n"
	if got != want {
		t.Errorf("FormatStats() =\n%q\nwant\n%q", got, want)
	}
}

func TestFormatStatsEmptyKnown(t *testing.T) {
	data := map[string]string{"あ": "a"}
	got := FormatStats(data, nil, nil, []string{"あ"})
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

func TestWriteKeyListCompactGrid(t *testing.T) {
	// More entries than fit in one row must wrap into a compact grid instead
	// of taking one line per kana.
	data := map[string]string{}
	keys := []string{"あ", "い", "う", "え", "お", "か", "き", "く"}
	for i, k := range keys {
		data[k] = fmt.Sprintf("r%d", i)
	}
	w := &strings.Builder{}
	writeKeyList(w, data, keys)
	lines := strings.Split(strings.TrimRight(w.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 grid lines for %d entries, got %d:\n%s", len(keys), len(lines), w.String())
	}
	if !strings.Contains(lines[0], "あ - r0") || !strings.Contains(lines[0], "か - r5") {
		t.Errorf("first row wrong: %q", lines[0])
	}
	if !strings.Contains(lines[1], "き - r6") || !strings.Contains(lines[1], "く - r7") {
		t.Errorf("second row wrong: %q", lines[1])
	}
	if strings.Contains(w.String(), "  あ - r0\n") {
		t.Errorf("entries should not be printed one per line:\n%s", w.String())
	}
}

func TestSortedHelpers(t *testing.T) {
	m := map[string]bool{"b": true, "a": true, "c": true}
	if got := strings.Join(sortedKeys(m), ","); got != "a,b,c" {
		t.Errorf("sortedKeys = %s", got)
	}
	s := map[string]string{"z": "1", "y": "2"}
	if got := strings.Join(sortedStringMap(s), ","); got != "y,z" {
		t.Errorf("sortedStringMap = %s", got)
	}
	if got := strings.Join(SortedPhraseKeys(s), ","); got != "y,z" {
		t.Errorf("SortedPhraseKeys = %s", got)
	}
}

func TestCleanDisplayDoesNotPanic(t *testing.T) {
	CleanDisplay() // must not panic outside a terminal
}
