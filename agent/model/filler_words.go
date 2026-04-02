package model

import "strings"

// FillerWords to strip from natural language input.
var FillerWords = []string{
	"give", "me", "the", "a", "an", "please", "can", "you",
	"i", "want", "need", "get", "show", "run", "execute",
	"do", "make", "let", "lets", "let's", "my", "some",
	"what", "is", "are", "how", "about", "whats", "what's",
	"your", "its", "it's", "tell", "whats",
}

func StripFillerWords(input string) string {
	fillerSet := make(map[string]bool, len(FillerWords))
	for _, f := range FillerWords {
		fillerSet[f] = true
	}
	words := strings.Fields(input)
	var result []string
	for _, w := range words {
		if !fillerSet[w] {
			result = append(result, w)
		}
	}
	return strings.Join(result, " ")
}
