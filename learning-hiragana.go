package main

import (
	"fmt"
	"os"
	"strings"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
)

func printLearningHiraganaMenu() {
	fmt.Println("Type the reading in English letters and press Enter to check.")
	fmt.Println("Press Ctrl+C or Esc to exit and see your statistics.")
}

func waitForEnter() bool {
	quit := false

	keyboard.Listen(func(key keys.Key) (bool, error) {
		switch key.Code {
		case keys.Escape:
			quit = true
			return true, nil
		case keys.Enter:
			return true, nil
		}
		return false, nil
	})

	return quit
}

func learningHiragana() {
	hiragana := map[string]string{
		"あ": "a", "い": "i", "う": "u", "え": "e", "お": "o",
		"か": "ka", "き": "ki", "く": "ku", "け": "ke", "こ": "ko",
		"さ": "sa", "し": "shi", "す": "su", "せ": "se", "そ": "so",
		"た": "ta", "ち": "chi", "つ": "tsu", "て": "te", "と": "to",
		"な": "na", "に": "ni", "ぬ": "nu", "ね": "ne", "の": "no",
		"は": "ha", "ひ": "hi", "ふ": "fu", "へ": "he", "ほ": "ho",
		"ま": "ma", "み": "mi", "む": "mu", "め": "me", "も": "mo",
		"や": "ya", "ゆ": "yu", "よ": "yo",
		"ら": "ra", "り": "ri", "る": "ru", "れ": "re", "ろ": "ro",
		"わ": "wa", "を": "wo",
		"ん": "n",
		"が": "ga", "ぎ": "gi", "ぐ": "gu", "げ": "ge", "ご": "go",
		"ざ": "za", "じ": "ji", "ず": "zu", "ぜ": "ze", "ぞ": "zo",
		"だ": "da", "ぢ": "ji", "づ": "zu", "で": "de", "ど": "do",
		"ば": "ba", "び": "bi", "ぶ": "bu", "べ": "be", "ぼ": "bo",
		"ぱ": "pa", "ぴ": "pi", "ぷ": "pu", "ぺ": "pe", "ぽ": "po",
	}

	known := map[string]bool{}
	unknown := map[string]bool{}

	showStats := func() {
		cleanDisplay()
		total := len(hiragana)
		fmt.Printf("Statistics (%d/%d answered correctly):\n\n", len(known), total)

		fmt.Println("Known hiragana:")
		if len(known) == 0 {
			fmt.Println("  none")
		} else {
			for k := range known {
				fmt.Printf("  %s - %s\n", k, hiragana[k])
			}
		}

		fmt.Println("\nNot known / not answered hiragana:")
		left := 0
		for k := range hiragana {
			if !known[k] && !unknown[k] {
				fmt.Printf("  %s - %s\n", k, hiragana[k])
				left++
			}
		}
		for k := range unknown {
			fmt.Printf("  %s - %s\n", k, hiragana[k])
		}
		if len(unknown) == 0 && left == 0 {
			fmt.Println("  none")
		}

		os.Exit(0)
	}

	for k, value := range hiragana {
		if known[k] {
			continue
		}
		cleanDisplay()
		if unknown[k] {
			fmt.Println("You didn't know this one before. Try again!")
			fmt.Println()
		}
		fmt.Println(k)
		fmt.Println()
		printLearningHiraganaMenu()
		fmt.Print("> ")

		var input string
		quit := false

		keyboard.Listen(func(key keys.Key) (bool, error) {
			switch key.Code {
			case keys.Escape:
				quit = true
				return true, nil
			case keys.Enter:
				return true, nil
			case keys.Backspace:
				if len(input) > 0 {
					runes := []rune(input)
					input = string(runes[:len(runes)-1])
					fmt.Print("\b \b")
				}
				return false, nil
			default:
				if len(key.Runes) > 0 {
					for _, r := range key.Runes {
						input += string(r)
						fmt.Print(string(r))
					}
				}
				return false, nil
			}
		})

		if quit {
			showStats()
		}

		answer := strings.ToLower(strings.TrimSpace(input))
		cleanDisplay()
		if answer == value {
			known[k] = true
			delete(unknown, k)
			fmt.Printf("Correct! You answered \"%s\".\n", answer)
			fmt.Printf("Correct answer: %s - %s\n", k, value)
		} else {
			unknown[k] = true
			if answer == "" {
				fmt.Println("Wrong. You didn't type anything.")
			} else {
				fmt.Printf("Wrong. You answered \"%s\", but that is incorrect.\n", answer)
			}
			fmt.Printf("Correct answer: %s - %s\n", k, value)
		}
		fmt.Println("\nPress Enter to continue... (Esc shows statistics)")
		if waitForEnter() {
			showStats()
		}
	}

	showStats()
}
