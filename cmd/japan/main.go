package main

import (
	"test/internal/app"
	"test/internal/data"
	"test/internal/ui"
)

func main() {
	a := app.New()
	a.NavigateTo = navigateTo(a)
	ui.NewMainMenu(a).Run()
}

// navigateTo dispatches a named destination to its flow. It breaks the
// mainMenu <-> learning flows recursion by returning a closure bound to the
// app instance.
func navigateTo(a *app.App) func(name string) {
	return func(name string) {
		switch name {
		case app.DestPhrases:
			a.RunLearningPhrases(data.AllPhrases())
		case app.DestHiragana:
			a.RunLearningHiragana(data.Hiragana)
		case app.DestResetHiragana:
			a.RunResetHiraganaProgress()
		case app.DestResetPhrases:
			a.RunResetPhrasesProgress()
		default: // app.DestMainMenu
			ui.NewMainMenu(a).Run()
		}
	}
}
