package bot

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode/utf8"
)

const maxQuestionPromptRunes = 1500
const maxFeedbackRunes = 650
const maxGuidanceRunes = 700
const maxHintRunes = 700

func formatQuestionMessage(intro, note string, q Question, prompt string) string {
	intro = strings.TrimSpace(intro)
	note = strings.TrimSpace(note)
	prompt = strings.TrimSpace(prompt)

	if prompt == "" {
		prompt = fmt.Sprintf("Could not load the full question statement right now.\nLeetCode URL: %s", q.URL)
	}

	prompt = truncateRunes(prompt, maxQuestionPromptRunes)

	lines := make([]string, 0, 10)
	if intro != "" {
		lines = append(lines, "<b>🧩 "+escapeHTML(intro)+"</b>")
	}
	if note != "" {
		lines = append(lines, "<i>"+escapeHTML(note)+"</i>")
	}

	lines = append(lines,
		fmt.Sprintf("<b>%s</b> (%s)", escapeHTML(q.Title), escapeHTML(q.Difficulty)),
		fmt.Sprintf("Link: <a href=\"%s\">%s</a>", escapeHTML(q.URL), escapeHTML(q.URL)),
		"",
		"<b>Problem</b>",
		"",
		renderStructuredTextForTelegram(prompt),
		"",
		"<b>Next</b>",
		"Reply with your approach. Use /hint for guidance, /skip for another question, or /exit.",
	)

	return strings.Join(lines, "\n")
}

func formatEvaluationMessage(q Question, score int, source, feedback, guidance, status string) string {
	feedback = truncateRunes(strings.TrimSpace(feedback), maxFeedbackRunes)
	guidance = truncateRunes(strings.TrimSpace(guidance), maxGuidanceRunes)
	status = strings.TrimSpace(status)

	lines := []string{
		"<b>🧠 Evaluation</b>",
		fmt.Sprintf("<b>%s</b> (%s)", escapeHTML(q.Title), escapeHTML(q.Difficulty)),
		fmt.Sprintf("Score: <b>%d/10</b> • Source: %s", score, escapeHTML(source)),
		"",
		"<b>Feedback</b>",
		"",
		renderStructuredTextForTelegram(feedback),
		"",
		"<b>Next Steps</b>",
		"",
		renderStructuredTextForTelegram(guidance),
		"",
		"<b>Status</b>",
		"",
		escapeHTML(status),
		"",
		"Send another attempt, /hint, /skip, /done, /exit, or /lc.",
	}

	return strings.Join(lines, "\n")
}

func formatHintMessage(q Question, source, hint string) string {
	hint = truncateRunes(strings.TrimSpace(hint), maxHintRunes)

	lines := []string{
		"<b>💡 Hint</b>",
		fmt.Sprintf("<b>%s</b> (%s)", escapeHTML(q.Title), escapeHTML(q.Difficulty)),
		fmt.Sprintf("Source: %s", escapeHTML(source)),
		"",
		renderStructuredTextForTelegram(hint),
		"",
		"Try updating your approach, then send it for evaluation.",
	}

	return strings.Join(lines, "\n")
}

var numberedListPattern = regexp.MustCompile(`^(\d+)[\.)]\s+(.*)$`)
var fencedCodeLangPattern = regexp.MustCompile(`[^a-zA-Z0-9_+\-]`)

func renderStructuredTextForTelegram(text string) string {
	text = strings.ReplaceAll(strings.TrimSpace(text), "\r\n", "\n")
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines)+4)
	inCodeBlock := false
	codeLang := ""
	codeBlock := make([]string, 0, 16)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			if inCodeBlock {
				out = append(out, renderCodeBlock(codeLang, codeBlock))
				inCodeBlock = false
				codeLang = ""
				codeBlock = codeBlock[:0]
				continue
			}

			codeLang = sanitizeCodeLanguage(strings.TrimSpace(strings.TrimPrefix(trimmed, "```")))
			inCodeBlock = true
			continue
		}

		if inCodeBlock {
			codeBlock = append(codeBlock, line)
			continue
		}

		if trimmed == "" {
			out = append(out, "")
			continue
		}

		if heading, ok := formatHeadingLine(trimmed); ok {
			out = append(out, heading)
			continue
		}

		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			out = append(out, "• "+escapeHTML(strings.TrimSpace(trimmed[2:])))
			continue
		}

		if matched := numberedListPattern.FindStringSubmatch(trimmed); len(matched) == 3 {
			out = append(out, fmt.Sprintf("%s. %s", matched[1], escapeHTML(strings.TrimSpace(matched[2]))))
			continue
		}

		out = append(out, escapeHTML(line))
	}

	if inCodeBlock {
		out = append(out, renderCodeBlock(codeLang, codeBlock))
	}

	return strings.Join(out, "\n")
}

func formatHeadingLine(line string) (string, bool) {
	for i := 0; i < len(line); i++ {
		if line[i] != '#' {
			if i == 0 || i > 3 || line[i] != ' ' {
				return "", false
			}
			label := strings.TrimSpace(line[i+1:])
			if label == "" {
				return "", false
			}
			return "<b>" + strings.Repeat("▸ ", i-1) + escapeHTML(label) + "</b>", true
		}
	}
	return "", false
}

func renderCodeBlock(lang string, codeLines []string) string {
	code := escapeHTML(strings.Join(codeLines, "\n"))
	if lang == "" {
		return "<pre>" + code + "</pre>"
	}
	return fmt.Sprintf("<pre><code class=\"language-%s\">%s</code></pre>", escapeHTML(lang), code)
}

func sanitizeCodeLanguage(lang string) string {
	lang = strings.TrimSpace(lang)
	if lang == "" {
		return ""
	}
	return strings.Trim(fencedCodeLangPattern.ReplaceAllString(lang, ""), "-_+")
}

func escapeHTML(text string) string {
	return html.EscapeString(text)
}

func truncateRunes(in string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(in) <= max {
		return in
	}

	out := make([]rune, 0, max+1)
	for _, r := range in {
		if len(out) >= max {
			break
		}
		out = append(out, r)
	}
	return strings.TrimSpace(string(out)) + "\n\n[truncated]"
}
