package bot

import (
	"context"
	"strings"
)

func (s *Service) sendHintForChat(ctx context.Context, chatID int64, learnerContext string) error {
	settings, err := s.store.GetChatSettings(ctx, chatID)
	if err != nil {
		return err
	}
	if settings.CurrentQuestion == nil {
		return s.tgClient.SendMessage(ctx, chatID, "No active question. Use /lc first.")
	}

	return s.sendHint(ctx, chatID, *settings.CurrentQuestion, learnerContext)
}

func (s *Service) sendHint(ctx context.Context, chatID int64, q Question, learnerContext string) error {
	hint, source := s.generateHint(ctx, q, learnerContext)
	msg := formatHintMessage(q, source, hint)
	return s.tgClient.SendRichMessage(ctx, chatID, msg)
}

func (s *Service) generateHint(ctx context.Context, q Question, learnerContext string) (string, string) {
	if s.coach != nil {
		hint, err := s.coach.GenerateHint(ctx, q, learnerContext)
		if err == nil {
			hint = strings.TrimSpace(hint)
			if hint != "" {
				return hint, "AI"
			}
		} else {
			s.logger.Printf("AI hint failed, falling back to heuristic hint: %v", err)
		}
	}

	return fallbackHint(q, learnerContext), "Heuristic"
}

func fallbackHint(q Question, learnerContext string) string {
	lines := []string{
		"## Direction",
		"- Track the minimum state needed to make each next decision.",
		"- Choose the simplest structure that supports O(1) updates/lookups when possible.",
		"- Validate with one normal case and one edge case before finalizing.",
		"",
		"## Pseudocode",
		"```text",
		"initialize state",
		"for each item in input:",
		"  update state",
		"  if success condition: return result",
		"return fallback",
		"```",
	}

	if strings.EqualFold(q.Difficulty, "Hard") {
		lines = append(lines,
			"",
			"## Hard Focus",
			"- Compare one baseline and one optimized approach before committing.",
		)
	}

	if ctx := strings.TrimSpace(learnerContext); ctx != "" {
		lines = append(lines,
			"",
			"## Based On Your Context",
			"- You are likely close. Tighten your invariant and termination condition.",
		)
	}

	return strings.Join(lines, "\n")
}

func parseHintRequest(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}

	lower := strings.ToLower(trimmed)
	switch lower {
	case "hint", "hint please", "need hint", "need a hint", "give me a hint", "can i get a hint", "another hint":
		return "", true
	}

	for _, prefix := range []string{"hint:", "hint ", "give me a hint ", "can i get a hint "} {
		if strings.HasPrefix(lower, prefix) {
			context := strings.TrimSpace(trimmed[len(prefix):])
			context = strings.Trim(context, "-:,. ")
			return context, true
		}
	}

	return "", false
}
