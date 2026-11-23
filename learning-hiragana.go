package main

import (
	"fmt"
	"os"

	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
)

func printLearningHiraganaMenu() {
	fmt.Println("1: I can translate\n2: I can't translate\n3: Go back to the main menu\n0: Exit")
}

func learningHiragana() {
	hiragana := map[string]string{
		// あ行
		"あ": "a", "い": "i", "う": "u", "え": "e", "お": "o",
		// か行
		"か": "ka", "き": "ki", "く": "ku", "け": "ke", "こ": "ko",
		// さ行
		"さ": "sa", "し": "shi", "す": "su", "せ": "se", "そ": "so",
		// た行
		"た": "ta", "ち": "chi", "つ": "tsu", "て": "te", "と": "to",
		// な行
		"な": "na", "に": "ni", "ぬ": "nu", "ね": "ne", "の": "no",
		// は行
		"は": "ha", "ひ": "hi", "ふ": "fu", "へ": "he", "ほ": "ho",
		// ま行
		"ま": "ma", "み": "mi", "む": "mu", "め": "me", "も": "mo",
		// や行
		"や": "ya", "ゆ": "yu", "よ": "yo",
		// ら行
		"ら": "ra", "り": "ri", "る": "ru", "れ": "re", "ろ": "ro",
		// わ行
		"わ": "wa", "を": "wo",
		// ん
		"ん": "n",
		// 濁点 (dakuten)
		"が": "ga", "ぎ": "gi", "ぐ": "gu", "げ": "ge", "ご": "go",
		"ざ": "za", "じ": "ji", "ず": "zu", "ぜ": "ze", "ぞ": "zo",
		"だ": "da", "ぢ": "ji", "づ": "zu", "で": "de", "ど": "do",
		"ば": "ba", "び": "bi", "ぶ": "bu", "べ": "be", "ぼ": "bo",
		// 半濁点 (handakuten)
		"ぱ": "pa", "ぴ": "pi", "ぷ": "pu", "ぺ": "pe", "ぽ": "po",
	}

	for k, value := range hiragana {
		fmt.Println(k, "\n")
		printLearningHiraganaMenu()
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
				printLearningHiraganaMenu()
				return false, nil
			}
			return true, nil
		})
		fmt.Println()
	}
}
