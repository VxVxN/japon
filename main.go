package main

import (
	"fmt"
)

func main() {
	mainMenu()
}

func cleanDisplay() {
	fmt.Print("\033[2J\033[H")
}
