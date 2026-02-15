package commands

import "context"

func (h *Handler) cmdDailyOff(ctx context.Context, chatID int64) error {
	settings, err := h.deps.GetChatSettings(ctx, chatID)
	if err != nil {
		return err
	}

	hhmm := settings.DailyTime
	if hhmm == "" {
		hhmm = h.deps.DefaultDailyHH()
	}

	if err := h.deps.UpsertDailySettings(ctx, chatID, false, hhmm, h.deps.DefaultTZ()); err != nil {
		return err
	}

	return h.deps.SendMessage(ctx, chatID, "Daily question is OFF. Use /daily_on to re-enable.")
}
