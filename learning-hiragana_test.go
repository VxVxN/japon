package main

import (
	"strings"
	"testing"
)

// smallData is a tiny kana set so tests do not depend on the full syllabary.
var smallData = map[string]string{"あ": "a", "い": "i"}

func TestReadLineCollectsText(t *testing.T) {
	a, buf, _, _, _ := newTestApp(t, runeKeys("ka")...)
	text, aborted := a.readLine()
	if aborted {
		t.Fatal("unexpected abort")
	}
	if text != "ka" {
		t.Errorf("readLine() = %q, want %q", text, "ka")
	}
	if !strings.Contains(buf.String(), "ka") {
		t.Errorf("typed characters should be echoed, got %q", buf.String())
	}
}

func TestReadLineBackspace(t *testing.T) {
	// type "kax", backspace once, then Enter -> "ka"
	keys := append(runeKeys("kax")[:3], backspaceKey())
	keys = append(keys, enterKey())
	a, buf, _, _, _ := newTestApp(t, keys...)
	text, _ := a.readLine()
	if text != "ka" {
		t.Errorf("readLine() with backspace = %q, want %q", text, "ka")
	}
	if !strings.Contains(buf.String(), "\b \b") {
		t.Errorf("backspace should erase on screen")
	}
}

func TestReadLineBackspaceOnEmpty(t *testing.T) {
	a, _, listener, _, _ := newTestApp(t, backspaceKey(), enterKey())
	text, aborted := a.readLine()
	if text != "" || aborted {
		t.Errorf("got (%q, %v), want empty non-aborted", text, aborted)
	}
	if len(listener.Consumed) != 2 {
		t.Errorf("backspace on empty input must not stop the listener; consumed=%d", len(listener.Consumed))
	}
}

func TestReadLineEscapeAborts(t *testing.T) {
	a, _, listener, _, _ := newTestApp(t, runeKey('a'), escKey(), enterKey())
	_, aborted := a.readLine()
	if !aborted {
		t.Fatal("expected abort on Escape")
	}
	if len(listener.Consumed) != 2 {
		t.Errorf("listener must stop at Escape, consumed=%d", len(listener.Consumed))
	}
}

func TestRunLearningHiraganaAllCorrect(t *testing.T) {
	// Answer あ->a, Enter, い->i, Enter -> statistics + exit(0).
	seq := []interface{}{"a", enterKey(), "i", enterKey()}
	a, buf, _, exits, _ := newTestApp(t, flattenKeys(seq)...)
	a.runLearningHiragana(smallData)

	out := buf.String()
	if !strings.Contains(out, `Correct! You answered "a".`) {
		t.Errorf("missing correct-answer feedback:\n%s", out)
	}
	if !strings.Contains(out, "Statistics (2/2 answered correctly)") {
		t.Errorf("final stats should report 2/2:\n%s", out)
	}
	if len(*exits) != 1 || (*exits)[0] != 0 {
		t.Fatalf("expected exit(0) after finishing, got %v", *exits)
	}
}

func TestRunLearningHiraganaWrongThenRepeated(t *testing.T) {
	// Wrong answer for あ, then correct on the repeat pass; い correct right away.
	seq := []interface{}{
		"x", enterKey(), // wrong for あ
		"i", enterKey(), // correct for い
		"a", enterKey(), // retry あ -> correct
	}
	a, buf, _, exits, _ := newTestApp(t, flattenKeys(seq)...)
	a.runLearningHiragana(smallData)

	out := buf.String()
	if !strings.Contains(out, `Wrong. You answered "x", but that is incorrect.`) {
		t.Errorf("missing wrong-answer feedback:\n%s", out)
	}
	if !strings.Contains(out, "You didn't know this one before.") {
		t.Errorf("unknown kana should be re-asked with a hint:\n%s", out)
	}
	if !strings.Contains(out, "Statistics (2/2 answered correctly)") {
		t.Errorf("after retry everything is known:\n%s", out)
	}
	if len(*exits) != 1 {
		t.Fatalf("expected exactly one exit, got %v", *exits)
	}
}

func TestRunLearningHiraganaEmptyAnswer(t *testing.T) {
	// Empty input counts as wrong ("didn't type anything").
	seq := []interface{}{
		enterKey(),      // empty answer for あ
		"a", enterKey(), // retry correct
		"i", enterKey(), // い correct
	}
	a, buf, _, _, _ := newTestApp(t, flattenKeys(seq)...)
	a.runLearningHiragana(smallData)
	if !strings.Contains(buf.String(), "Wrong. You didn't type anything.") {
		t.Errorf("empty answer should say \"didn't type anything\":\n%s", buf.String())
	}
}

func TestRunLearningHiraganaEscapeShowsStats(t *testing.T) {
	// Escape during the first question shows statistics and exits.
	a, buf, _, exits, _ := newTestApp(t, escKey())
	a.runLearningHiragana(smallData)

	out := buf.String()
	if !strings.Contains(out, "Statistics (0/2 answered correctly)") {
		t.Errorf("stats missing after quit:\n%s", out)
	}
	if !strings.Contains(out, "none") {
		t.Errorf("known section should show none:\n%s", out)
	}
	if len(*exits) != 1 || (*exits)[0] != 0 {
		t.Fatalf("expected exit(0), got %v", *exits)
	}
}

func TestRunLearningHiraganaCaseInsensitive(t *testing.T) {
	seq := []interface{}{"A", enterKey(), "I", enterKey()}
	a, buf, _, _, _ := newTestApp(t, flattenKeys(seq)...)
	a.runLearningHiragana(smallData)
	if strings.Contains(buf.String(), "Wrong.") {
		t.Errorf("uppercase answers should be accepted:\n%s", buf.String())
	}
}
