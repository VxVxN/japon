package main

import (
	"fmt"

	"atomicgo.dev/keyboard/keys"
)

// printMainMenu prints the top-level menu to the app's output.
func (a *app) printMainMenu() {
	fmt.Fprint(a.out, "1: Learning phrases\n2: Learning hiragana\n0: Exit\n")
}

// runMainMenu shows the main menu and waits for a selection. Escape or "0"
// exits the program, "1" opens the phrases flow, "2" opens the hiragana flow.
// Any other key simply redraws the menu.
func (a *app) runMainMenu() {
	a.clean()
	a.printMainMenu()
	_ = a.listener.Listen(func(k keys.Key) bool {
		switch dest := classifyMenuKey(k); dest {
		case destQuit:
			a.exit(0)
			return true
		case destPhrases, destHiragana:
			a.clean()
			a.navigateTo(dest)
			return true
		default:
			a.clean()
			a.printMainMenu()
			return false
		}
	})
}

// mainMenu is the entry point used by main().
func mainMenu() {
	currentApp.runMainMenu()
}
