package commands

import (
	"context"
	"strings"
)

func (h *Handler) cmdLC(ctx context.Context, chatID int64, args []string) error {
	if len(args) == 0 {
		h.deps.SetPendingTopicSelection(chatID, true)
		return h.deps.SendMessage(ctx, chatID, "Which topic do you want to practice? Reply with a topic like array, graph, dp, tree, or enter \"random\".")
	}

	topic := strings.TrimSpace(strings.Join(args, " "))
	if strings.EqualFold(topic, "random") || topic == "" {
		topic = ""
	}

	settings, err := h.deps.GetChatSettings(ctx, chatID)
	if err != nil {
		return err
	}
	if settings.CurrentQuestion != nil {
		return h.deps.SendUniqueQuestionByTopic(ctx, chatID, "Here is your random LeetCode question:", topic, settings.CurrentQuestion.Slug)
	}
	return h.deps.SendUniqueQuestionByTopic(ctx, chatID, "Here is your random LeetCode question:", topic)
}
