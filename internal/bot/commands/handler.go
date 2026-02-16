package commands

import (
	"context"
	"strings"
)

type Handler struct {
	deps Dependencies
}

func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

func (h *Handler) Handle(ctx context.Context, chatID int64, text string) error {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return nil
	}

	cmd := normalizeCommand(parts[0])
	args := parts[1:]

	switch cmd {
	case "/start", "/help":
		return h.cmdHelp(ctx, chatID)
	case "/lc":
		return h.cmdLC(ctx, chatID)
	case "/hint":
		return h.cmdHint(ctx, chatID, args)
	case "/done":
		return h.cmdDone(ctx, chatID)
	case "/skip":
		return h.cmdSkip(ctx, chatID)
	case "/exit":
		return h.cmdExit(ctx, chatID)
	case "/delete":
		return h.cmdDeleteRevisedQuestion(ctx, chatID, args)
	case "/answered":
		return h.cmdAnsweredHistory(ctx, chatID, args)
	case "/revise":
		return h.cmdRevise(ctx, chatID, args)
	case "/daily_on":
		return h.cmdDailyOn(ctx, chatID, args)
	case "/daily_off":
		return h.cmdDailyOff(ctx, chatID)
	case "/daily_time":
		return h.cmdDailyTime(ctx, chatID, args)
	case "/daily_status":
		return h.cmdDailyStatus(ctx, chatID)
	default:
		return h.deps.SendMessage(ctx, chatID, "Unknown command. Use /help to see available commands.")
	}
}
