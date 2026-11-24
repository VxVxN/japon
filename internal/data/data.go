// Package data holds the static learning content: hiragana kana and phrase
// categories.
package data

import "maps"

// Hiragana maps every hiragana kana to its romaji reading. It includes the
// basic syllabary plus the voiced (dakuten/handakuten) series.
var Hiragana = map[string]string{
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

// Phrase categories shown by the phrases learning flow.
var (
	FoodPhrases = map[string]string{
		"おちゃとごはん、ください。":  "Tea and rice, please.",
		"みずとラーメン、ください。":  "Water and ramen, please.",
		"ピザはおいしいです。":     "Pizza is tasty.",
		"サラダはおいしいですか。":   "Is the salad tasty?",
		"はい、ケーキはおいしいです。": "Yes, cake is tasty.",
	}
	CommonPhrases = map[string]string{
		"こんばんは。":  "Good evening!",
		"こんにちは。":  "Hello!",
		"おはよう。":   "Good morning!",
		"またあした。":  "See you tomorrow.",
		"はじめまして。": "Nice to meet you.",
	}
	ClothesPhrases = map[string]string{
		"あかいかばん。":    "The red bag.",
		"しろいくつ。":     "The white shoes.",
		"コートはどこですか。": "Where is the coat?",
	}
	EtcPhrases = map[string]string{
		"アメリカはおおきいです。": "America is big.",
		"にほんはちいさいですか。": "Is Japan small?",
	}
)

// AllPhrases merges every category into a single map.
func AllPhrases() map[string]string {
	all := map[string]string{}
	for _, m := range []map[string]string{FoodPhrases, CommonPhrases, ClothesPhrases, EtcPhrases} {
		maps.Copy(all, m)
	}
	return all
}
