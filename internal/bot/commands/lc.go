package commands

import "context"

func (h *Handler) cmdLC(ctx context.Context, chatID int64) error {
	settings, err := h.deps.GetChatSettings(ctx, chatID)
	if err != nil {
		return err
	}
	if settings.CurrentQuestion != nil {
		return h.deps.SendUniqueQuestion(ctx, chatID, "Here is your random LeetCode question:", settings.CurrentQuestion.Slug)
	}
	return h.deps.SendUniqueQuestion(ctx, chatID, "Here is your random LeetCode question:")
}
