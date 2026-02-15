package commands

import "context"

func (h *Handler) cmdDone(ctx context.Context, chatID int64) error {
	settings, err := h.deps.GetChatSettings(ctx, chatID)
	if err != nil {
		return err
	}
	if settings.CurrentQuestion == nil {
		return h.deps.SendMessage(ctx, chatID, "No active question. Use /lc first.")
	}

	if err := h.deps.PersistCompletedQuestion(ctx, chatID, *settings.CurrentQuestion); err != nil {
		return err
	}

	return h.deps.SendMessage(ctx, chatID, "Marked as done and saved to your seen/revision history. Send /lc for another question.")
}
