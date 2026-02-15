package commands

import (
	"context"
	"fmt"
)

func (h *Handler) cmdDailyStatus(ctx context.Context, chatID int64) error {
	settings, err := h.deps.GetChatSettings(ctx, chatID)
	if err != nil {
		return err
	}

	status := "OFF"
	if settings.DailyEnabled {
		status = "ON"
	}

	hhmm := settings.DailyTime
	if hhmm == "" {
		hhmm = h.deps.DefaultDailyHH()
	}
	zone := settings.Timezone
	if zone == "" {
		zone = h.deps.DefaultTZ()
	}

	msg := fmt.Sprintf("Daily status: %s\nTime: %s\nTimezone: %s", status, hhmm, tzLabel(zone))
	return h.deps.SendMessage(ctx, chatID, msg)
}
