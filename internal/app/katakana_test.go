package app

import (
	"strings"
	"testing"
	"test/internal/data"
	"test/internal/kbd"
)

// smallKanaData is a tiny katakana set so tests do not depend on the full
// syllabary.
var smallKanaData = map[string]string{"ア": "a", "イ": "i"}

func TestRunLearningKatakanaAllCorrect(t *testing.T) {
	// Answer ア->a, Enter, イ->i, Enter -> statistics + exit(0).
	seq := []interface{}{"a", kbd.EnterKey(), "i", kbd.EnterKey()}
	a, buf, _, exits, _ := newTestApp(t, flattenKeys(seq)...)
	a.RunLearningKatakana(smallKanaData)

	out := buf.String()
	if !strings.Contains(out, `Correct! You answered "a".`) {
		t.Errorf("missing correct-answer feedback:\n%s", out)
	}
	if !strings.Contains(out, "Statistics (2/2 answered correctly)") {
		t.Errorf("final stats should report 2/2:\n%s", out)
	}
	if !strings.Contains(out, "Known katakana:") {
		t.Errorf("stats should use the katakana wording:\n%s", out)
	}
	if strings.Contains(out, "Known hiragana:") {
		t.Errorf("katakana stats must not say hiragana:\n%s", out)
	}
	if len(*exits) != 1 || (*exits)[0] != 0 {
		t.Fatalf("expected exit(0) after finishing, got %v", *exits)
	}
}

func TestRunLearningKatakanaWrongThenRepeated(t *testing.T) {
	// Wrong answer for ア, then correct on the repeat pass; イ correct right away.
	seq := []interface{}{
		"x", kbd.EnterKey(), // wrong for ア
		"i", kbd.EnterKey(), // correct for イ
		"a", kbd.EnterKey(), // retry ア -> correct
	}
	a, buf, _, exits, _ := newTestApp(t, flattenKeys(seq)...)
	a.RunLearningKatakana(smallKanaData)

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

func TestRunLearningKatakanaEmptyAnswer(t *testing.T) {
	// Empty input counts as wrong ("didn't type anything").
	seq := []interface{}{
		kbd.EnterKey(), // empty answer for ア
		"a", kbd.EnterKey(), // retry correct
		"i", kbd.EnterKey(), // イ correct
	}
	a, buf, _, _, _ := newTestApp(t, flattenKeys(seq)...)
	a.RunLearningKatakana(smallKanaData)
	if !strings.Contains(buf.String(), "Wrong. You didn't type anything.") {
		t.Errorf("empty answer should say \"didn't type anything\":\n%s", buf.String())
	}
}

func TestRunLearningKatakanaEscapeShowsStats(t *testing.T) {
	// Escape during the first question shows statistics and exits.
	a, buf, _, exits, _ := newTestApp(t, kbd.EscKey())
	a.RunLearningKatakana(smallKanaData)

	out := buf.String()
	if !strings.Contains(out, "Statistics (0/2 answered correctly)") {
		t.Errorf("stats missing after quit:\n%s", out)
	}
	if !strings.Contains(out, "none") {
		t.Errorf("known section should show none:\n%s", out)
	}
	if !strings.Contains(out, "Not known / not answered katakana:") {
		t.Errorf("stats should use the katakana wording:\n%s", out)
	}
	if len(*exits) != 1 || (*exits)[0] != 0 {
		t.Fatalf("expected exit(0), got %v", *exits)
	}
}

func TestRunLearningKatakanaCaseInsensitive(t *testing.T) {
	seq := []interface{}{"A", kbd.EnterKey(), "I", kbd.EnterKey()}
	a, buf, _, _, _ := newTestApp(t, flattenKeys(seq)...)
	a.RunLearningKatakana(smallKanaData)
	if strings.Contains(buf.String(), "Wrong.") {
		t.Errorf("uppercase answers should be accepted:\n%s", buf.String())
	}
}

func TestRunLearningKatakanaSkipsKnownAndKeepsHiraganaProgress(t *testing.T) {
	// ア was already learned in a previous session; only イ is asked. The
	// katakana save must not clobber the persisted hiragana progress.
	var stored ProgressData
	stored = ProgressData{KnownKatakana: []string{"ア"}, KnownHiragana: []string{"あ"}}
	a, buf, _, exits, _ := newTestAppWithProgress(t, &stored, flattenKeys([]interface{}{"i", kbd.EnterKey()})...)

	a.RunLearningKatakana(smallKanaData)

	out := buf.String()
	if strings.Contains(out, "\nア\n") {
		t.Errorf("already-known katakana should be skipped:\n%s", out)
	}
	if !strings.Contains(out, "Statistics (2/2 answered correctly)") {
		t.Errorf("final stats should report 2/2:\n%s", out)
	}
	if len(stored.KnownHiragana) != 1 || stored.KnownHiragana[0] != "あ" {
		t.Errorf("katakana progress save clobbered hiragana progress: %+v", stored)
	}
	if len(stored.KnownKatakana) != 2 {
		t.Errorf("expected both kana known after run, got %v", stored.KnownKatakana)
	}
	if len(*exits) != 1 || (*exits)[0] != 0 {
		t.Fatalf("expected exit(0), got %v", *exits)
	}
}

func TestRunResetKatakanaProgress(t *testing.T) {
	var stored ProgressData
	stored = ProgressData{KnownKatakana: []string{"ア"}, KnownHiragana: []string{"あ"}, KnownPhrases: []string{"p"}}
	a, buf, _, _, navs := newTestAppWithProgress(t, &stored, kbd.EnterKey())

	a.RunResetKatakanaProgress()

	if len(stored.KnownKatakana) != 0 {
		t.Errorf("katakana progress should be cleared, got %v", stored.KnownKatakana)
	}
	if len(stored.KnownHiragana) != 1 || len(stored.KnownPhrases) != 1 {
		t.Errorf("other progress sections must stay intact: %+v", stored)
	}
	if !strings.Contains(buf.String(), "katakana progress has been reset") {
		t.Errorf("missing confirmation message:\n%s", buf.String())
	}
	if len(*navs) == 0 || (*navs)[len(*navs)-1] != DestMainMenu {
		t.Errorf("should return to the main menu, navs=%v", *navs)
	}
}

func TestRunResetKatakanaProgressCancel(t *testing.T) {
	var stored ProgressData
	stored = ProgressData{KnownKatakana: []string{"ア"}}
	a, _, _, _, navs := newTestAppWithProgress(t, &stored, kbd.EscKey())

	a.RunResetKatakanaProgress()

	if len(stored.KnownKatakana) != 1 {
		t.Errorf("cancelled reset must not touch progress: %+v", stored)
	}
	if len(*navs) != 1 || (*navs)[0] != DestMainMenu {
		t.Errorf("cancel should navigate back to the main menu, got %v", *navs)
	}
}

func TestKatakanaMapSanity(t *testing.T) {
	// Basic syllabary spot checks and no empty readings.
	for _, k := range []string{"ア", "カ", "サ", "タ", "ナ", "ハ", "マ", "ヤ", "ラ", "ワ", "ン", "ガ", "パ"} {
		if data.Katakana[k] == "" {
			t.Errorf("data.Katakana[%q] has no reading", k)
		}
	}
	if len(data.Katakana) < 70 {
		t.Errorf("data.Katakana unexpectedly small: %d", len(data.Katakana))
	}
	// Katakana and hiragana sets must have the same size and readings.
	if len(data.Katakana) != len(data.Hiragana) {
		t.Errorf("katakana (%d) and hiragana (%d) sets differ in size", len(data.Katakana), len(data.Hiragana))
	}
	hiraReadings := map[string]int{}
	for _, v := range data.Hiragana {
		hiraReadings[v]++
	}
	for _, v := range data.Katakana {
		if hiraReadings[v] == 0 {
			t.Errorf("katakana reading %q has no hiragana counterpart", v)
		}
	}
}
