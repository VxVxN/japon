package web

import (
	"testing"

	"test/internal/app"
)

type fakeConn struct {
	sent []wsMessage
}

func (f *fakeConn) read() (wsMessage, bool) { return wsMessage{}, false }
func (f *fakeConn) send(m wsMessage) error  { f.sent = append(f.sent, m); return nil }
func (f *fakeConn) closeWrite() error       { return nil }

func lastQuiz(t *testing.T, f *fakeConn) *Quiz {
	t.Helper()
	m := f.sent[len(f.sent)-1]
	if m.State == nil || m.State.Quiz == nil {
		t.Fatalf("no quiz state in last frame: %+v", m)
	}
	return m.State.Quiz
}

func TestPhraseShowKeepsButtonsAndMovesOnNext(t *testing.T) {
	f := &fakeConn{}
	s := NewSession(f, nil)

	if !s.handleSelect(app.DestPhrases) {
		t.Fatal("start phrases failed")
	}
	first := lastQuiz(t, f)
	if first.Kind != "phrase" || !first.Input || first.Translation != "" {
		t.Fatalf("initial phrase state wrong: %+v", first)
	}
	firstPrompt := first.Prompt

	// "show" reveals the translation but keeps the SAME phrase with buttons.
	if !s.handleAnswer("show") {
		t.Fatal("show action failed")
	}
	q := lastQuiz(t, f)
	if q.Prompt != firstPrompt {
		t.Fatalf("phrase changed after show: %q -> %q", firstPrompt, q.Prompt)
	}
	if q.Translation == "" {
		t.Fatal("translation not revealed")
	}
	if !q.Input {
		t.Fatal("action buttons must stay visible after showing translation")
	}

	// Then the user can still decide on this same phrase and moves to the next.
	if !s.handleAnswer("know") {
		t.Fatal("know action failed")
	}
	q2 := lastQuiz(t, f)
	if q2.Prompt == firstPrompt {
		t.Fatal("expected to move to the next phrase after deciding")
	}
	if q2.Kind != "phrase" || !q2.Input {
		t.Fatalf("next phrase state wrong: %+v", q2)
	}
}
