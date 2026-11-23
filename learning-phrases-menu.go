package main

import (
	"fmt"
	"maps"
	"os"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
)

func printLearningPhrasesMenu() {
	fmt.Println("1: I can translate\n2: I can't translate\n3: Go back to the main menu\n0: Exit")
}

func learningPhrases() {
	all := map[string]string{}
	food := map[string]string{
		"おちゃとごはんをください。":  "Tea and rice, please.",
		"みずとラーメンをください。":  "Water and ramen, please.",
		"ピザはおいしいです。":     "Pizza is tasty.",
		"サラダはおいしいですか。":   "Is the salad tasty?",
		"はい、ケーキはおいしいです。": "Yes, cake is tasty.",
	}
	common := map[string]string{
		"こんばんは。":  "Good evening!",
		"こんにちは。":  "Hello!",
		"おはよう。":   "Good morning!",
		"またあした。":  "See you tomorrow.",
		"はじめまして。": "Nice to meet you.",
	}
	clothes := map[string]string{
		"あかいかばん。":    "The red bag.",
		"しろいくつ。":     "The white shoes.",
		"コートはどこですか。": "Where is the coat?",
	}
	etc := map[string]string{
		"アメリカはおおきいです。": "America is big.",
		"にほんはちいさいですか。": "Is Japan small?",
	}
	maps.Copy(all, food)
	maps.Copy(all, common)
	maps.Copy(all, clothes)
	maps.Copy(all, etc)

	for k, value := range all {
		fmt.Println(k, "\n")
		printLearningPhrasesMenu()
		keyboard.Listen(func(key keys.Key) (stop bool, err error) {
			switch key.Code {
			case keys.Escape:
				os.Exit(0)
			}
			switch key.String() {
			case "0":
				os.Exit(0)
			case "1":
				cleanDisplay()
			case "2":
				cleanDisplay()
				fmt.Println(k, "-", value)
			case "3":
				mainMenu()
			default:
				cleanDisplay()
				printLearningPhrasesMenu()
				return false, nil
			}
			return true, nil
		})
		fmt.Println()
	}
}
