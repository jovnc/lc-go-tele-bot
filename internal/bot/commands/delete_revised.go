package commands

import (
	"context"
	"fmt"
	"strings"
)

func (h *Handler) cmdDeleteRevisedQuestion(ctx context.Context, chatID int64, args []string) error {
	if len(args) == 0 {
		return h.deps.SendMessage(ctx, chatID, "Usage: /delete <slug>, e.g. /delete two-sum")
	}

	slug := normalizeSlug(strings.Join(args, " "))
	if slug == "" {
		return h.deps.SendMessage(ctx, chatID, "Usage: /delete <slug>, e.g. /delete two-sum")
	}

	if err := h.deps.DeleteAnsweredQuestion(ctx, chatID, slug); err != nil {
		if h.deps.IsAnsweredQuestionNotFound(err) {
			return h.deps.SendMessage(ctx, chatID, "I couldn't find that slug in your revised list. Use /answered to see available slugs.")
		}
		return err
	}
	if err := h.deps.RemoveServedQuestion(ctx, chatID, slug); err != nil {
		return err
	}

	settings, err := h.deps.GetChatSettings(ctx, chatID)
	if err != nil {
		return err
	}
	if settings.CurrentQuestion != nil && settings.CurrentQuestion.Slug == slug {
		if err := h.deps.ClearCurrentQuestion(ctx, chatID); err != nil {
			return err
		}
	}

	return h.deps.SendMessage(ctx, chatID, fmt.Sprintf("Deleted %q from revised history and seen set.", slug))
}
