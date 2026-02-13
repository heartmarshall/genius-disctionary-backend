package domain

import (
	"strings"
)

// NormalizeText prepares text for storage and comparison:
//   - trims leading/trailing whitespace
//   - converts to lowercase
//   - compresses multiple spaces into one
//
// Diacritics, hyphens, and apostrophes are preserved.
func NormalizeText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	text = strings.ToLower(text)

	// Compress multiple spaces into one.
	var b strings.Builder
	b.Grow(len(text))
	prevSpace := false
	for _, r := range text {
		if r == ' ' {
			if prevSpace {
				continue
			}
			prevSpace = true
		} else {
			prevSpace = false
		}
		b.WriteRune(r)
	}
	return b.String()
}
