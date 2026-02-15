package commands

import (
	"context"
	"fmt"
)

func (h *Handler) cmdDailyOn(ctx context.Context, chatID int64, args []string) error {
	settings, err := h.deps.GetChatSettings(ctx, chatID)
	if err != nil {
		return err
	}

	hhmm := settings.DailyTime
	if hhmm == "" {
		hhmm = h.deps.DefaultDailyHH()
	}
	if len(args) > 0 {
		hhmm, err = normalizeHHMM(args[0])
		if err != nil {
			return h.deps.SendMessage(ctx, chatID, "Invalid time. Use 24h HH:MM, e.g. /daily_on 20:30")
		}
	}

	if err := h.deps.UpsertDailySettings(ctx, chatID, true, hhmm, h.deps.DefaultTZ()); err != nil {
		return err
	}

	return h.deps.SendMessage(ctx, chatID, fmt.Sprintf("Daily question is ON at %s %s. Use /daily_off to stop.", hhmm, tzLabel(h.deps.DefaultTZ())))
}
