package main

import (
	"fmt"
	"maps"

	"atomicgo.dev/keyboard/keys"
)

// Phrase categories shown by the phrases learning flow.
var (
	foodPhrases = map[string]string{
		"おちゃとごはんをください。":  "Tea and rice, please.",
		"みずとラーメンをください。":  "Water and ramen, please.",
		"ピザはおいしいです。":     "Pizza is tasty.",
		"サラダはおいしいですか。":   "Is the salad tasty?",
		"はい、ケーキはおいしいです。": "Yes, cake is tasty.",
	}
	commonPhrases = map[string]string{
		"こんばんは。":  "Good evening!",
		"こんにちは。":  "Hello!",
		"おはよう。":   "Good morning!",
		"またあした。":  "See you tomorrow.",
		"はじめまして。": "Nice to meet you.",
	}
	clothesPhrases = map[string]string{
		"あかいかばん。":    "The red bag.",
		"しろいくつ。":     "The white shoes.",
		"コートはどこですか。": "Where is the coat?",
	}
	etcPhrases = map[string]string{
		"アメリカはおおきいです。": "America is big.",
		"にほんはちいさいですか。": "Is Japan small?",
	}
)

// allPhrases merges every category into a single map.
func allPhrases() map[string]string {
	all := map[string]string{}
	for _, m := range []map[string]string{foodPhrases, commonPhrases, clothesPhrases, etcPhrases} {
		maps.Copy(all, m)
	}
	return all
}

// printLearningPhrasesMenu prints the per-phrase menu.
func (a *app) printLearningPhrasesMenu() {
	fmt.Fprint(a.out, "1: I can translate\n2: I can't translate\n3: Go back to the main menu\n0: Exit\n")
}

// runLearningPhrases walks through the given phrase set. For each phrase the
// user either confirms they can translate it ("1"), asks to see the
// translation ("2"), returns to the main menu ("3") or exits ("0"/Esc).
func (a *app) runLearningPhrases(phrases map[string]string) {
	for _, k := range sortedPhraseKeys(phrases) {
		value := phrases[k]
		fmt.Fprintln(a.out, k)
		fmt.Fprintln(a.out)
		a.printLearningPhrasesMenu()

		_ = a.listener.Listen(func(key keys.Key) bool {
			switch key.String() {
			case "0":
				a.exit(0)
				return true
			case "1":
				a.clean()
				return true
			case "2":
				a.clean()
				fmt.Fprintf(a.out, "%s - %s\n", k, value)
				return true
			case "3":
				a.navigateTo(destMainMenu)
				return true
			default:
				if key.Code == keys.Escape {
					a.exit(0)
					return true
				}
				a.clean()
				a.printLearningPhrasesMenu()
				return false
			}
		})
		fmt.Fprintln(a.out)
	}
}

// sortedPhraseKeys returns phrase keys in deterministic order.
func sortedPhraseKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sortStrings(out)
	return out
}

// learningPhrases is the entry point used by the main menu.
func learningPhrases() {
	currentApp.runLearningPhrases(allPhrases())
}
