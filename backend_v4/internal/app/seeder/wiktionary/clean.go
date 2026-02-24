package wiktionary

import (
	"regexp"
	"strings"
)

var (
	htmlTagRe   = regexp.MustCompile(`<[^>]*>`)
	wikiLinkRe  = regexp.MustCompile(`\[\[([^|\]]*\|)?([^\]]*)\]\]`)
	multiSpaceRe = regexp.MustCompile(`\s{2,}`)
)

// StripMarkup removes HTML tags and wiki-style links from s,
// collapses multiple spaces, and trims whitespace.
func StripMarkup(s string) string {
	if s == "" {
		return ""
	}

	// Remove HTML tags.
	s = htmlTagRe.ReplaceAllString(s, "")

	// Replace wiki links [[link|display]] → display, [[word]] → word.
	s = wikiLinkRe.ReplaceAllString(s, "$2")

	// Collapse multiple whitespace into a single space.
	s = multiSpaceRe.ReplaceAllString(s, " ")

	// Trim leading/trailing whitespace.
	s = strings.TrimSpace(s)

	return s
}

// TruncateDefinition truncates s to maxLen runes. If truncation occurs,
// an ellipsis character is appended after the truncated text.
func TruncateDefinition(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\u2026"
}

// DeduplicateStrings returns a new slice with duplicate strings removed,
// preserving the order of first occurrence. Returns nil for nil input.
func DeduplicateStrings(ss []string) []string {
	if ss == nil {
		return nil
	}

	seen := make(map[string]struct{}, len(ss))
	result := make([]string, 0, len(ss))

	for _, s := range ss {
		if _, exists := seen[s]; exists {
			continue
		}
		seen[s] = struct{}{}
		result = append(result, s)
	}

	return result
}
