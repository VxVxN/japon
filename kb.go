package main

import (
	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
)

// keyHandler processes a single key event. It returns true when the listener
// should stop (i.e. the current interaction is finished).
type keyHandler func(k keys.Key) bool

// keyListener abstracts the keyboard listening mechanism so that interactive
// code can be unit tested with a fake key source.
type keyListener interface {
	Listen(h keyHandler) error
}

// realKeyListener uses atomicgo.dev/keyboard to read actual terminal input.
type realKeyListener struct{}

func (realKeyListener) Listen(h keyHandler) error {
	return keyboard.Listen(func(key keys.Key) (bool, error) {
		return h(key), nil
	})
}

// fakeKeyListener feeds a predefined sequence of keys to the handler. It is
// intended for tests: every consumed key is recorded in Consumed. If the
// sequence ends before the handler stops, Listen returns normally and
// Exhausted is set to true.
type fakeKeyListener struct {
	Keys      []keys.Key
	Consumed  []keys.Key
	Exhausted bool
}

func newFakeKeyListener(k ...keys.Key) *fakeKeyListener {
	return &fakeKeyListener{Keys: k}
}

// Listen feeds the predefined keys to the handler until it stops or the
// sequence runs out.
func (f *fakeKeyListener) Listen(h keyHandler) error {
	for len(f.Keys) > 0 {
		k := f.Keys[0]
		f.Keys = f.Keys[1:]
		f.Consumed = append(f.Consumed, k)
		if h(k) {
			return nil
		}
	}
	f.Exhausted = true
	return nil
}

// runeKey builds a key event for a typed character exactly like the real
// keyboard listener does (KeyCode == keys.RuneKey plus the produced rune).
func runeKey(r rune) keys.Key {
	return keys.Key{Code: keys.RuneKey, Runes: []rune{r}}
}

// digitKey builds a key event for a typed digit. Note: digits arrive as
// RuneKey events with the digit's rune, NOT with Code '0'..'9'.
func digitKey(d byte) keys.Key {
	return runeKey(rune('0' + d))
}

// textKeys builds the key events produced by typing the given text.
func textKeys(text string) []keys.Key {
	ks := make([]keys.Key, 0, len(text))
	for _, r := range text {
		ks = append(ks, runeKey(r))
	}
	return ks
}

// runeKeys builds a key sequence that types the given text and presses Enter.
func runeKeys(text string) []keys.Key {
	return append(textKeys(text), enterKey())
}

// key constructors ------------------------------------------------------------

func enterKey() keys.Key     { return keys.Key{Code: keys.Enter} }
func escKey() keys.Key       { return keys.Key{Code: keys.Escape} }
func backspaceKey() keys.Key { return keys.Key{Code: keys.Backspace} }
