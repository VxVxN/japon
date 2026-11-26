package app

import (
"fmt"
"testing"
"test/internal/kbd"

"atomicgo.dev/keyboard/keys"
)

type tracingFake struct {
*kbd.Fake
calls int
}

func (tf *tracingFake) Listen(h kbd.Handler) error {
tf.calls++
start := len(tf.Consumed)
err := tf.Fake.Listen(func(k keys.Key) bool { return h(k) })
fmt.Printf("LISTEN#%d consumed=%d..%d exhaustedAfter=%v\n", tf.calls, start, len(tf.Consumed), tf.Exhausted)
return err
}

func TestZZRepro(t *testing.T) {
seq := []interface{}{"a", kbd.EnterKey(), "i", kbd.EnterKey()}
a, buf, listener, exits, _ := newTestApp(t, flattenKeys(seq)...)
tr := &tracingFake{Fake: listener}
a.Listener = tr
a.RunLearningHiragana(smallData)
fmt.Printf("DONE exits=%v buflen=%d\n", *exits, buf.Len())
}
