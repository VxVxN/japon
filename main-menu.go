package main

import (
	"fmt"
	"os"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
)

func printMainMenu() {
	fmt.Println("1: Learning phrases\n2: Learning\nhiragana\n0: Exit")
}

func mainMenu() {
	cleanDisplay()
	printMainMenu()
	keyboard.Listen(func(key keys.Key) (stop bool, err error) {
		switch key.Code {
		case keys.Escape:
			os.Exit(0)
		default:
		}
		switch key.String() {
		case "0":
			os.Exit(0)
		case "1":
			cleanDisplay()
			learningPhrases()
		case "2":
			cleanDisplay()
			learningHiragana()
		default:
			cleanDisplay()
			printMainMenu()
			return false, nil
		}
		return true, nil
	})
}
