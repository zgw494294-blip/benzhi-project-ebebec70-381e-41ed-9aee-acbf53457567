package domain

import (
	"strings"
	"unicode"
)

func NormalizeText(text string) string {
	var b strings.Builder
	space := false
	for _, r := range text {
		if unicode.IsSpace(r) {
			if !space {
				b.WriteRune(' ')
			}
			space = true
			continue
		}
		space = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}
func NormalizeSpeaker(label string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(label), " "))
}
