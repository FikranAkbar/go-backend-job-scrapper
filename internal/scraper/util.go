// Package scraper provides shared utility helpers for all scrapers.
package scraper

import (
	"html"
	"regexp"
	"strings"
)

var (
	reHTMLTag   = regexp.MustCompile(`<[^>]+>`)
	reMultiSpace = regexp.MustCompile(`\s{2,}`)
)

// stripHTMLTags removes HTML tags and unescapes HTML entities from s,
// returning clean plain text suitable for storage and AI prompts.
func stripHTMLTags(s string) string {
	s = reHTMLTag.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = reMultiSpace.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// truncate returns up to maxBytes UTF-8 characters of s.
func truncate(s string, maxChars int) string {
	runes := []rune(s)
	if len(runes) <= maxChars {
		return s
	}
	return string(runes[:maxChars])
}

