package bot

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const maxQuestionPromptRunes = 3400
const maxFeedbackRunes = 1400
const maxGuidanceRunes = 1600
const maxHintRunes = 3400

func formatQuestionMessage(intro, note string, q Question, prompt string) string {
	_ = strings.TrimSpace(intro)
	note = strings.TrimSpace(note)
	prompt = strings.TrimSpace(prompt)

	if prompt == "" {
		prompt = "Could not load the full question statement right now."
	}

	prompt = truncateRunes(prompt, maxQuestionPromptRunes)
	prompt = stripQuestionLinkLines(prompt, q.URL)

	titleLine := fmt.Sprintf("*%s* \\(%s\\)", escapeMarkdownV2(q.Title), escapeMarkdownV2(q.Difficulty))
	if strings.TrimSpace(q.URL) != "" {
		titleLine = fmt.Sprintf("*[%s](%s)* \\(%s\\)", escapeMarkdownV2(q.Title), escapeMarkdownV2URL(q.URL), escapeMarkdownV2(q.Difficulty))
	}

	lines := make([]string, 0, 10)
	if note != "" {
		lines = append(lines, "_"+escapeMarkdownV2(note)+"_")
	}

	lines = append(lines,
		titleLine,
		"",
		"__*Problem*__",
		"",
		renderStructuredTextForTelegram(prompt),
		"",
		"__*Next*__",
		escapeMarkdownV2("Reply with your approach. Use /hint for guidance, /skip for another question, or /exit."),
	)

	return strings.Join(lines, "\n")
}

func formatEvaluationMessage(q Question, score int, source, feedback, guidance, status string) string {
	feedback = truncateRunes(strings.TrimSpace(feedback), maxFeedbackRunes)
	guidance = truncateRunes(strings.TrimSpace(guidance), maxGuidanceRunes)
	status = strings.TrimSpace(status)

	lines := []string{
		"*🧠 Evaluation*",
		fmt.Sprintf("*%s* \\(%s\\)", escapeMarkdownV2(q.Title), escapeMarkdownV2(q.Difficulty)),
		fmt.Sprintf("Score: *%d/10* • Source: %s", score, escapeMarkdownV2(source)),
		"",
		"__*Feedback*__",
		"",
		renderStructuredTextForTelegram(feedback),
		"",
		"__*Next Steps*__",
		"",
		renderStructuredTextForTelegram(guidance),
		"",
		"__*Status*__",
		"",
		escapeMarkdownV2(status),
		"",
		escapeMarkdownV2("Send another attempt, /hint, /skip, /done, /exit, or /lc."),
	}

	return strings.Join(lines, "\n")
}

func formatHintMessage(q Question, source, hint string) string {
	hint = strings.TrimSpace(hint)
	if !strings.Contains(hint, "```") {
		hint = truncateRunes(hint, maxHintRunes)
	}

	lines := []string{
		"*💡 Hint*",
		fmt.Sprintf("*%s* \\(%s\\)", escapeMarkdownV2(q.Title), escapeMarkdownV2(q.Difficulty)),
		fmt.Sprintf("Source: %s", escapeMarkdownV2(source)),
		"",
		renderStructuredTextForTelegram(hint),
		"",
		escapeMarkdownV2("Try updating your approach, then send it for evaluation."),
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
			out = append(out, "• "+escapeMarkdownV2(strings.TrimSpace(trimmed[2:])))
			continue
		}

		if matched := numberedListPattern.FindStringSubmatch(trimmed); len(matched) == 3 {
			out = append(out, fmt.Sprintf("%s\\. %s", matched[1], escapeMarkdownV2(strings.TrimSpace(matched[2]))))
			continue
		}

		out = append(out, escapeMarkdownV2(line))
	}

	if inCodeBlock {
		out = append(out, renderCodeBlock(codeLang, codeBlock))
	}

	return strings.Join(out, "\n")
}

func formatHeadingLine(line string) (string, bool) {
	for i := 0; i < len(line); i++ {
		if line[i] != '#' {
			if i == 0 || i > 6 || line[i] != ' ' {
				return "", false
			}
			label := strings.TrimSpace(line[i+1:])
			if label == "" {
				return "", false
			}
			return "__*" + escapeMarkdownV2(label) + "*__", true
		}
	}
	return "", false
}

func stripQuestionLinkLines(prompt, questionURL string) string {
	lines := strings.Split(strings.TrimSpace(prompt), "\n")
	if len(lines) == 0 {
		return ""
	}

	url := strings.ToLower(strings.TrimSpace(questionURL))
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "link:") || strings.HasPrefix(lower, "url:") {
			continue
		}
		if url != "" && strings.Contains(lower, url) {
			continue
		}
		out = append(out, line)
	}

	return strings.TrimSpace(strings.Join(out, "\n"))
}

func renderCodeBlock(lang string, codeLines []string) string {
	code := escapeMarkdownV2Code(strings.Join(codeLines, "\n"))
	if lang == "" {
		return "```\n" + code + "\n```"
	}
	return fmt.Sprintf("```%s\n%s\n```", sanitizeCodeLanguage(lang), code)
}

func sanitizeCodeLanguage(lang string) string {
	lang = strings.TrimSpace(lang)
	if lang == "" {
		return ""
	}
	return strings.Trim(fencedCodeLangPattern.ReplaceAllString(lang, ""), "-_+")
}

func escapeMarkdownV2(text string) string {
	if text == "" {
		return ""
	}

	var out strings.Builder
	out.Grow(len(text) + 8)
	for _, r := range text {
		switch r {
		case '\\', '_', '*', '[', ']', '(', ')', '~', '`', '>', '#', '+', '-', '=', '|', '{', '}', '.', '!':
			out.WriteByte('\\')
		}
		out.WriteRune(r)
	}
	return out.String()
}

func escapeMarkdownV2URL(url string) string {
	if url == "" {
		return ""
	}

	var out strings.Builder
	out.Grow(len(url) + 4)
	for _, r := range url {
		switch r {
		case '\\', ')':
			out.WriteByte('\\')
		}
		out.WriteRune(r)
	}
	return out.String()
}

func escapeMarkdownV2Code(code string) string {
	if code == "" {
		return ""
	}
	code = strings.ReplaceAll(code, "\\", "\\\\")
	code = strings.ReplaceAll(code, "`", "\\`")
	return code
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
	return strings.TrimSpace(string(out)) + "\n\n\\[truncated\\]"
}
