package bot

import (
	"fmt"
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
		lines = append(lines, "*"+escapeMarkdownV2(intro)+"*")
	}
	if note != "" {
		lines = append(lines, escapeMarkdownV2(note))
	}

	lines = append(lines,
		fmt.Sprintf("*%s* \\(%s\\)", escapeMarkdownV2(q.Title), escapeMarkdownV2(q.Difficulty)),
		"",
		"*Problem Statement*",
		"",
		escapeMarkdownV2(prompt),
		"",
		"Reply with your approach and I will evaluate it\\. Use /skip for another question or /exit to leave practice mode\\.",
	)

	return strings.Join(lines, "\n")
}

func formatEvaluationMarkdown(q Question, score int, source, feedback, guidance, status string) string {
	feedback = truncateRunes(strings.TrimSpace(feedback), maxFeedbackRunes)
	guidance = truncateRunes(strings.TrimSpace(guidance), maxGuidanceRunes)
	status = strings.TrimSpace(status)

	lines := []string{
		fmt.Sprintf("*Evaluation for %s* \\(%s\\)", escapeMarkdownV2(q.Title), escapeMarkdownV2(q.Difficulty)),
		fmt.Sprintf("Score: %d/10", score),
		fmt.Sprintf("Source: %s", escapeMarkdownV2(source)),
		"",
		"*Feedback*",
		"",
		escapeMarkdownV2(feedback),
		"",
		"*Guided Next Steps*",
		"",
		escapeMarkdownV2(guidance),
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
