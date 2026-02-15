package leetcode

import (
	"html"
	"regexp"
	"strings"
)

var (
	reLineBreak     = regexp.MustCompile(`(?i)<\s*br\s*/?\s*>`)
	reBlockClose    = regexp.MustCompile(`(?i)</\s*(p|div|section|article|pre|blockquote|h[1-6]|tr)\s*>`)
	reListItemOpen  = regexp.MustCompile(`(?i)<\s*li\s*>`)
	reListContainer = regexp.MustCompile(`(?i)</?\s*(ul|ol)\s*>`)
	reAllTags       = regexp.MustCompile(`(?s)<[^>]+>`)
	reMultiSpace    = regexp.MustCompile(`[ \t]{2,}`)
	reManyNewlines  = regexp.MustCompile(`\n{3,}`)
)

func htmlToText(content string) string {
	text := strings.TrimSpace(content)
	if text == "" {
		return ""
	}

	text = reLineBreak.ReplaceAllString(text, "\n")
	text = reBlockClose.ReplaceAllString(text, "\n\n")
	text = reListItemOpen.ReplaceAllString(text, "\n- ")
	text = reListContainer.ReplaceAllString(text, "\n")
	text = reAllTags.ReplaceAllString(text, "")
	text = html.UnescapeString(text)

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		line = reMultiSpace.ReplaceAllString(line, " ")
		lines[i] = line
	}

	text = strings.Join(lines, "\n")
	text = reManyNewlines.ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}
