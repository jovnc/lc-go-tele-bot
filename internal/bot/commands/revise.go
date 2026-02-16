package commands

import (
	"context"
	"strings"
)

func (h *Handler) cmdRevise(ctx context.Context, chatID int64, args []string) error {
	var (
		q   Question
		err error
	)

	if len(args) > 0 {
		slug := normalizeSlug(strings.Join(args, " "))
		if slug == "" {
			return h.deps.SendMessage(ctx, chatID, "Usage: /revise <slug>, e.g. /revise two-sum")
		}
		q, err = h.deps.GetAnsweredQuestion(ctx, chatID, slug)
		if err != nil {
			if h.deps.IsAnsweredQuestionNotFound(err) {
				return h.deps.SendMessage(ctx, chatID, "I couldn't find that slug in your answered history. Use /answered to see available slugs.")
			}
			return err
		}
	} else {
		items, listErr := h.deps.ListAnsweredQuestions(ctx, chatID, 50)
		if listErr != nil {
			return listErr
		}
		if len(items) == 0 {
			return h.deps.SendMessage(ctx, chatID, "No answered questions to revise yet. Complete one first with /lc and /done (or a correct attempt).")
		}
		idx := int(h.deps.Now().UnixNano() % int64(len(items)))
		if idx < 0 {
			idx = -idx
		}
		q = items[idx].Question
	}

	if err := h.deps.SetCurrentQuestion(ctx, chatID, q); err != nil {
		return err
	}

	prompt, err := h.deps.QuestionPrompt(ctx, q.Slug)
	if err != nil {
		h.deps.Logf("revision question prompt lookup failed for slug=%s: %v", q.Slug, err)
		prompt = ""
	}

	msg := h.deps.FormatQuestionMessage("Revision question from your history:", "", q, prompt)
	return h.deps.SendRichMessage(ctx, chatID, msg)
}
