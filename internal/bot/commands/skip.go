package commands

import "context"

func (h *Handler) cmdSkip(ctx context.Context, chatID int64) error {
	settings, err := h.deps.GetChatSettings(ctx, chatID)
	if err != nil {
		return err
	}
	if settings.CurrentQuestion == nil {
		return h.deps.SendMessage(ctx, chatID, "No active question to skip. Use /lc first.")
	}

	return h.deps.SendUniqueQuestion(ctx, chatID, "Skipped. Here is another LeetCode question:", settings.CurrentQuestion.Slug)
}
