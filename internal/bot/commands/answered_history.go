package commands

import (
	"context"
	"fmt"
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

	lines := make([]string, 0, len(items)+2)
	lines = append(lines, "Answered questions (latest first):")
	for i, item := range items {
		last := "unknown"
		if !item.LastAnsweredAt.IsZero() {
			last = item.LastAnsweredAt.UTC().Format("2006-01-02")
		}
		lines = append(lines, fmt.Sprintf("%d. %s (%s) | slug: %s | attempts: %d | last: %s",
			i+1, item.Title, item.Difficulty, item.Slug, item.Attempts, last))
	}
	lines = append(lines, "Use /revise <slug> to revisit a specific question, or /revise for a random one.")

	return h.deps.SendMessage(ctx, chatID, strings.Join(lines, "\n"))
}
