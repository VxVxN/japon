package main

import (
	"fmt"
)

func main() {
	currentApp = newApp()
	mainMenu()
}

func cleanDisplay() {
	fmt.Print("\033[2J\033[H")
}
