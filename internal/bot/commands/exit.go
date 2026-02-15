package commands

import "context"

func (h *Handler) cmdExit(ctx context.Context, chatID int64) error {
	settings, err := h.deps.GetChatSettings(ctx, chatID)
	if err != nil {
		return err
	}
	if settings.CurrentQuestion == nil {
		return h.deps.SendMessage(ctx, chatID, "No active practice mode. Use /lc when you want a question.")
	}

	if err := h.deps.ClearCurrentQuestion(ctx, chatID); err != nil {
		return err
	}

	return h.deps.SendMessage(ctx, chatID, "Exited practice mode. Send /lc when you want another question.")
}
