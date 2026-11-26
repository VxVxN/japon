package kbd

import (
	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
)

// Handler processes a single key event. It returns true when the listener
// should stop (i.e. the current interaction is finished).
type Handler func(k keys.Key) bool

// Listener abstracts the keyboard listening mechanism so that interactive
// code can be unit tested with a fake key source.
type Listener interface {
	Listen(h Handler) error
}

// Real uses atomicgo.dev/keyboard to read actual terminal input.
type Real struct{}

func (Real) Listen(h Handler) error {
	return keyboard.Listen(func(key keys.Key) (bool, error) {
		return h(key), nil
	})
}

// Fake feeds a predefined sequence of keys to the handler. It is intended
// for tests: every consumed key is recorded in Consumed. If the sequence
// ends before the handler stops, Listen returns normally and Exhausted is
// set to true. When the optional OnDrained hook is set, it runs at that
// moment; interactive flows (see cmd/web) use it to deliver one final
// synthetic Enter so an in-progress line is still answered when the source
// runs dry - exactly like a user pressing Enter right before closing the
// session. The hook must not feed new keys into Keys while Listen is
// iterating (that would loop forever); it only signals waiting readers.
type Fake struct {
	Keys      []keys.Key
	Consumed  []keys.Key
	Exhausted bool
	OnDrained func()
}

// NewFake builds a fake listener for the given key sequence.
func NewFake(k ...keys.Key) *Fake {
	return &Fake{Keys: k}
}

// Listen feeds the predefined keys to the handler until it stops or the
// sequence runs out. Once the sequence is exhausted, every subsequent call
// returns immediately (after setting Exhausted) instead of spinning forever;
// interactive flows must treat this like the real terminal going quiet and
// finish their current interaction.
func (f *Fake) Listen(h Handler) error {
	if len(f.Keys) == 0 {
		// The source already ran dry in an earlier call: stay exhausted so
		// repeated calls return immediately instead of spinning forever.
		f.Exhausted = true
		return nil
	}
	for len(f.Keys) > 0 {
		k := f.Keys[0]
		f.Keys = f.Keys[1:]
		f.Consumed = append(f.Consumed, k)
		if h(k) {
			return nil
		}
	}
	f.Exhausted = true
	if f.OnDrained != nil {
		f.OnDrained()
	}
	return nil
}

// IsExhausted reports whether a previous Listen call consumed the whole key
// sequence without the handler stopping on its own. It is intentionally NOT
// set when the handler terminates on the very last key of the sequence: that
// means the interaction received a real answer (Enter/Escape), and flows use
// this method to distinguish "the user answered" from "the key source ran
// dry" (see Watch and the quiet() helpers in the app package). Callers that
// need to detect exhaustion right after a ReadLine - including the case
// where the terminating Enter was the final key of the sequence - check
// HasPendingKeys themselves once Listen returned (see cmd/web and the
// exhaustion handling in the app package).
func (f *Fake) IsExhausted() bool { return f.Exhausted }

// HasPendingKeys reports whether the fake can still deliver at least one key
// event to the next Listen call.
func (f *Fake) HasPendingKeys() bool { return len(f.Keys) > 0 }

// Watch wraps the given Listener and sets *seen to true whenever a key event
// actually reaches the handler. Flows use it together with the optional
// IsExhausted method to distinguish "the user answered" from "the key source
// ran dry", which would otherwise cause infinite loops when no further keys
// can arrive (exhausted fake in tests, closed web session).
func Watch(l Listener, seen *bool) Listener {
	return &watcher{inner: l, seen: seen}
}

type watcher struct {
	inner Listener
	seen  *bool
}

// Inner exposes the wrapped listener so that callers can probe it for the
// optional IsExhausted method (see the quiet() helpers in the app package).
func (w *watcher) Inner() Listener { return w.inner }

func (w *watcher) Listen(h Handler) error {
	return w.inner.Listen(func(k keys.Key) bool {
		if w.seen != nil {
			*w.seen = true
		}
		return h(k)
	})
}

func (w *watcher) IsExhausted() bool {
	if q, ok := w.inner.(interface{ IsExhausted() bool }); ok {
		return q.IsExhausted()
	}
	return false
}

// RuneKey builds a key event for a typed character exactly like the real
// keyboard listener does (KeyCode == keys.RuneKey plus the produced rune).
func RuneKey(r rune) keys.Key {
	return keys.Key{Code: keys.RuneKey, Runes: []rune{r}}
}

// DigitKey builds a key event for a typed digit. Note: digits arrive as
// RuneKey events with the digit's rune, NOT with Code '0'..'9'. The argument
// may be given either as the numeric value (0..9) or as the ASCII byte ('0'..'9').
func DigitKey(d byte) keys.Key {
	if d >= '0' && d <= '9' {
		return RuneKey(rune(d))
	}
	return RuneKey(rune('0' + d))
}

// TextKeys builds the key events produced by typing the given text.
func TextKeys(text string) []keys.Key {
	ks := make([]keys.Key, 0, len(text))
	for _, r := range text {
		ks = append(ks, RuneKey(r))
	}
	return ks
}

// KeysWithEnter builds a key sequence that types the given text and presses Enter.
func KeysWithEnter(text string) []keys.Key {
	return append(TextKeys(text), EnterKey())
}

// Key constructors ------------------------------------------------------------

func EnterKey() keys.Key     { return keys.Key{Code: keys.Enter} }
func EscKey() keys.Key       { return keys.Key{Code: keys.Escape} }
func BackspaceKey() keys.Key { return keys.Key{Code: keys.Backspace} }
