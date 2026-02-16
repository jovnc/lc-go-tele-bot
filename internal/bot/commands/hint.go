package commands

import (
	"context"
	"strings"
)

func (h *Handler) cmdHint(ctx context.Context, chatID int64, args []string) error {
	learnerContext := strings.TrimSpace(strings.Join(args, " "))
	return h.deps.SendHint(ctx, chatID, learnerContext)
}
