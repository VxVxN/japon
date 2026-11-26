package app

import (
	"time"
"bytes"
"testing"
"test/internal/kbd"

"atomicgo.dev/keyboard/keys"
)

func TestZZProbe(t *testing.T) {
buf := &bytes.Buffer{}
listener := kbd.NewFake(append(kbd.TextKeys("a"), kbd.EnterKey(), kbd.EnterKey(), kbd.EnterKey())...)
var exits []int
progress := ProgressData{}
a := &App{
Out:        writerFunc(func(p []byte) (int, error) { n := buf.Write(p); t.Logf("write %q len=%d", string(p), buf.Len()); return n, nil }),
Listener:   listener,
Clean:      func() {},
Exit:       func(code int) { exits = append(exits, code) },
NavigateTo: func(string) {},
LoadProgress: func() ProgressData { return progress },
SaveProgress: func(p ProgressData) error { progress = p; return nil },
}
go func() {
for i := 0; i < 20; i++ {
keysX := len(listener.Consumed)
t.Logf("tick %d consumed=%d buflen=%d", i, keysX, buf.Len())
sleepOneSec()
}
}()
_ = keys.Key{}
a.RunLearningHiragana(map[string]string{"あ": "a"})
t.Logf("done exits=%v", exits)
}

type writerFunc func(p []byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

func sleepOneSec() { time.Sleep(time.Second) }
