package ui

import (
	"fmt"
	"test/internal/app"

	"atomicgo.dev/keyboard/keys"
)

// MainMenu is the top-level menu flow. It uses an app.App for all IO so it
// can be unit tested with fakes.
type MainMenu struct {
	a *app.App
}

// NewMainMenu builds a MainMenu around the given app instance. The app's
// NavigateTo hook must already be wired to dispatch destinations.
func NewMainMenu(a *app.App) *MainMenu {
	return &MainMenu{a: a}
}

// printMenu prints the top-level menu to the app's output.
func (m *MainMenu) printMenu() {
	fmt.Fprint(m.a.Out, "1: Learning phrases\n2: Learning hiragana\n0: Exit\n")
}

// Run shows the main menu and waits for a selection. Escape or "0" exits the
// program, "1" opens the phrases flow, "2" opens the hiragana flow. Any other
// key simply redraws the menu.
func (m *MainMenu) Run() {
	m.a.Clean()
	m.printMenu()
	_ = m.a.Listener.Listen(func(k keys.Key) bool {
		switch dest := app.ClassifyMenuKey(k); dest {
		case app.DestQuit:
			m.a.Exit(0)
			return true
		case app.DestPhrases, app.DestHiragana:
			m.a.Clean()
			m.a.NavigateTo(dest)
			return true
		default:
			m.a.Clean()
			m.printMenu()
			return false
		}
	})
}
