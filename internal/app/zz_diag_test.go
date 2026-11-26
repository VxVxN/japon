package app

import (
"fmt"
"testing"
"test/internal/kbd"

"atomicgo.dev/keyboard/keys"
)

type diagFake struct {
*kbd.Fake
n     int
lette map[byte]int
}

func (d *diagFake) Listen(h kbd.Handler) error {
d.n++
var consumed []string
mark := len(d.Consumed)
err := d.Fake.Listen(func(k keys.Key) bool {
s := fmt.Sprintf("code=%v str=%q runes=%q", k.Code, k.String(), string(k.Runes))
consumed = append(consumed, s)
return h(k)
})
if d.n <= 8 {
fmt.Printf("DIAG listen#%d consumed=%v\n", d.n, consumed)
}
for _, c := range consumed {
for b := range d.lette {
_ = b
}
}
return err
}

func TestZZDiag(t *testing.T) {
seq := []interface{}{"a", kbd.EnterKey(), "i", kbd.EnterKey()}
a, buf, listener, _, _ := newTestApp(t, flattenKeys(seq)...)
a.Listener = &diagFake{Fake: listener}
done := make(chan struct{})
go func() { a.RunLearningHiragana(smallData); close(done) }()
for i := 0; i < 50000000; i++ {
select {
case <-done:
fmt.Printf("DIAG DONE buflen=%d\n", buf.Len())
return
default:
}
}
panic("stuck")
}
