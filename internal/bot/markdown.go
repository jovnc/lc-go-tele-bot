package bot

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const maxQuestionPromptRunes = 1700
const maxFeedbackRunes = 1200
const maxGuidanceRunes = 1400

var markdownV2Escaper = strings.NewReplacer(
	"\\", "\\\\",
	"_", "\\_",
	"*", "\\*",
	"[", "\\[",
	"]", "\\]",
	"(", "\\(",
	")", "\\)",
	"~", "\\~",
	"`", "\\`",
	">", "\\>",
	"#", "\\#",
	"+", "\\+",
	"-", "\\-",
	"=", "\\=",
	"|", "\\|",
	"{", "\\{",
	"}", "\\}",
	".", "\\.",
	"!", "\\!",
)

func formatQuestionMarkdown(intro, note string, q Question, prompt string) string {
	intro = strings.TrimSpace(intro)
	note = strings.TrimSpace(note)
	prompt = strings.TrimSpace(prompt)

	if prompt == "" {
		prompt = fmt.Sprintf("Could not load the full question statement right now.\nLeetCode URL: %s", q.URL)
	}

	prompt = truncateRunes(prompt, maxQuestionPromptRunes)

	lines := make([]string, 0, 8)
	if intro != "" {
		lines = append(lines, "*🧩 "+escapeMarkdownV2(intro)+"*")
	}
	if note != "" {
		lines = append(lines, "_"+escapeMarkdownV2(note)+"_")
	}

	lines = append(lines,
		fmt.Sprintf("*%s* \\(%s\\)", escapeMarkdownV2(q.Title), escapeMarkdownV2(q.Difficulty)),
		"",
		"*Problem Statement*",
		"",
		renderMarkdownForTelegram(prompt),
		"",
		"*Next Step*",
		"Reply with your approach and I will evaluate it\\. Use /skip for another question or /exit to leave practice mode\\.",
	)

	return strings.Join(lines, "\n")
}

func formatEvaluationMarkdown(q Question, score int, source, feedback, guidance, status string) string {
	feedback = truncateRunes(strings.TrimSpace(feedback), maxFeedbackRunes)
	guidance = truncateRunes(strings.TrimSpace(guidance), maxGuidanceRunes)
	status = strings.TrimSpace(status)

	lines := []string{
		"*🧠 Evaluation*",
		fmt.Sprintf("*Question:* %s \\(%s\\)", escapeMarkdownV2(q.Title), escapeMarkdownV2(q.Difficulty)),
		fmt.Sprintf("*Score:* %d/10", score),
		fmt.Sprintf("*Source:* %s", escapeMarkdownV2(source)),
		"",
		"*Feedback*",
		"",
		renderMarkdownForTelegram(feedback),
		"",
		"*Guided Next Steps*",
		"",
		renderMarkdownForTelegram(guidance),
		"",
		"*Status*",
		"",
		escapeMarkdownV2(status),
		"",
		"Send another attempt, /skip for another question, /exit to leave practice mode, or /lc for a new random question\\.",
	}

	return strings.Join(lines, "\n")
}

func escapeMarkdownV2(text string) string {
	return markdownV2Escaper.Replace(text)
}

var numberedListPattern = regexp.MustCompile(`^(\d+)[\.)]\s+(.*)$`)

func renderMarkdownForTelegram(text string) string {
	text = strings.ReplaceAll(strings.TrimSpace(text), "\r\n", "\n")
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			if inCodeBlock {
				out = append(out, "```")
				inCodeBlock = false
				continue
			}

			lang := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
			if lang != "" {
				out = append(out, "```"+escapeMarkdownV2(lang))
			} else {
				out = append(out, "```")
			}
			inCodeBlock = true
			continue
		}

		if inCodeBlock {
			out = append(out, escapeCodeBlockLine(line))
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
		out = append(out, "```")
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
			return "*" + strings.Repeat("▸ ", i-1) + escapeMarkdownV2(label) + "*", true
		}
	}
	return "", false
}

func escapeCodeBlockLine(line string) string {
	line = strings.ReplaceAll(line, "\\", "\\\\")
	return strings.ReplaceAll(line, "`", "\\`")
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
