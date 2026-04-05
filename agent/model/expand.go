package model

import "strings"

// singularize strips common plural suffixes.
// Only applies to words longer than 3 chars to avoid mangling short words like "ls".
func singularize(word string) string {
	if len(word) <= 3 {
		return word
	}
	if strings.HasSuffix(word, "ies") && len(word) > 4 {
		return word[:len(word)-3] + "y"
	}
	if strings.HasSuffix(word, "ses") && len(word) > 4 {
		return word[:len(word)-2]
	}
	if strings.HasSuffix(word, "s") {
		return word[:len(word)-1]
	}
	return word
}

// ExpandKeywords returns the original keywords plus singularized variants.
func ExpandKeywords(keywords []string) []string {
	return expandKeywords(keywords)
}

func expandKeywords(keywords []string) []string {
	expanded := make([]string, 0, len(keywords)*2)
	seen := make(map[string]bool)
	for _, kw := range keywords {
		if !seen[kw] {
			expanded = append(expanded, kw)
			seen[kw] = true
		}
		s := singularize(kw)
		if s != kw && !seen[s] {
			expanded = append(expanded, s)
			seen[s] = true
		}
	}
	return expanded
}
