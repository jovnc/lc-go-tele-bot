package commands

import (
	"context"
	"fmt"
)

func (h *Handler) cmdDailyTime(ctx context.Context, chatID int64, args []string) error {
	if len(args) == 0 {
		return h.deps.SendMessage(ctx, chatID, "Usage: /daily_time HH:MM (24h), e.g. /daily_time 21:00")
	}

	hhmm, err := normalizeHHMM(args[0])
	if err != nil {
		return h.deps.SendMessage(ctx, chatID, "Invalid time. Use 24h HH:MM, e.g. /daily_time 21:00")
	}

	if err := h.deps.UpsertDailySettings(ctx, chatID, true, hhmm, h.deps.DefaultTZ()); err != nil {
		return err
	}

	return h.deps.SendMessage(ctx, chatID, fmt.Sprintf("Daily time set to %s %s and notifications are ON.", hhmm, tzLabel(h.deps.DefaultTZ())))
}
