package commands

import (
	"context"
	"fmt"
	"html"
	"strings"
)

func (h *Handler) cmdAnsweredHistory(ctx context.Context, chatID int64, args []string) error {
	limit := 10
	if len(args) > 0 {
		parsed, err := parsePositiveLimit(args[0], 50)
		if err != nil {
			return h.deps.SendMessage(ctx, chatID, "Usage: /answered [limit], e.g. /answered 10")
		}
		limit = parsed
	}

	items, err := h.deps.ListAnsweredQuestions(ctx, chatID, limit)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return h.deps.SendMessage(ctx, chatID, "No answered questions yet. Use /lc and either solve correctly or /done.")
	}

	lines := make([]string, 0, len(items)+4)
	lines = append(lines, "<b>📚 Answered Questions</b>", "")
	for i, item := range items {
		last := "unknown"
		if !item.LastAnsweredAt.IsZero() {
			last = item.LastAnsweredAt.UTC().Format("2006-01-02")
		}
		lines = append(lines,
			fmt.Sprintf("<b>%d. %s</b>", i+1, escapeHTML(item.Title)),
			fmt.Sprintf("• Difficulty: %s", escapeHTML(item.Difficulty)),
			fmt.Sprintf("• Slug: <code>%s</code>", escapeHTML(item.Slug)),
			fmt.Sprintf("• Attempts: %d", item.Attempts),
			fmt.Sprintf("• Last answered: %s", escapeHTML(last)),
			"",
		)
	}
	lines = append(lines, "Use /revise <slug> to revisit a specific question, or /revise for a random one.")

	return h.deps.SendRichMessage(ctx, chatID, strings.Join(lines, "\n"))
}

func escapeHTML(text string) string {
	return html.EscapeString(text)
}
