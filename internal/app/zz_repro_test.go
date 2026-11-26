package app

import (
"testing"
"test/internal/kbd"
)

func TestZZReproLoop(t *testing.T) {
seq := []interface{}{"a", kbd.EnterKey(), "i", kbd.EnterKey()}
a, buf, listener, exits, _ := newTestApp(t, flattenKeys(seq)...)
done := make(chan struct{})
go func() { a.RunLearningHiragana(smallData); close(done) }()
for i := 0; i < 3000000; i++ {
select {
case <-done:
t.Logf("DONE consumed=%d exhausted=%v buflen=%d exits=%v", len(listener.Consumed), listener.Exhausted, buf.Len(), *exits)
return
default:
}
}
t.Logf("STILL RUNNING consumed=%d exhausted=%v buflen=%d exits=%v", len(listener.Consumed), listener.Exhausted, buf.Len(), *exits)
panic("stuck")
}
