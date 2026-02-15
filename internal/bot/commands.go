package bot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (s *Service) handleCommand(ctx context.Context, chatID int64, text string) error {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return nil
	}

	cmd := normalizeCommand(parts[0])
	args := parts[1:]

	switch cmd {
	case "/start", "/help":
		return s.tgClient.SendMessage(ctx, chatID, helpText())
	case "/lc":
		return s.cmdLC(ctx, chatID)
	case "/done":
		return s.cmdDone(ctx, chatID)
	case "/skip":
		return s.cmdSkip(ctx, chatID)
	case "/exit":
		return s.cmdExit(ctx, chatID)
	case "/delete", "/unrevise":
		return s.cmdDeleteRevisedQuestion(ctx, chatID, args)
	case "/answered", "/history":
		return s.cmdAnsweredHistory(ctx, chatID, args)
	case "/revise":
		return s.cmdRevise(ctx, chatID, args)
	case "/daily_on":
		return s.cmdDailyOn(ctx, chatID, args)
	case "/daily_off":
		return s.cmdDailyOff(ctx, chatID)
	case "/daily_time":
		return s.cmdDailyTime(ctx, chatID, args)
	case "/daily_status":
		return s.cmdDailyStatus(ctx, chatID)
	default:
		return s.tgClient.SendMessage(ctx, chatID, "Unknown command. Use /help to see available commands.")
	}
}

func (s *Service) cmdLC(ctx context.Context, chatID int64) error {
	settings, err := s.store.GetChatSettings(ctx, chatID)
	if err != nil {
		return err
	}
	if settings.CurrentQuestion != nil {
		return s.sendUniqueQuestion(ctx, chatID, "Here is your random LeetCode question:", settings.CurrentQuestion.Slug)
	}
	return s.sendUniqueQuestion(ctx, chatID, "Here is your random LeetCode question:")
}

func (s *Service) cmdDone(ctx context.Context, chatID int64) error {
	settings, err := s.store.GetChatSettings(ctx, chatID)
	if err != nil {
		return err
	}
	if settings.CurrentQuestion == nil {
		return s.tgClient.SendMessage(ctx, chatID, "No active question. Use /lc first.")
	}

	if err := s.persistCompletedQuestion(ctx, chatID, *settings.CurrentQuestion); err != nil {
		return err
	}

	return s.tgClient.SendMessage(ctx, chatID, "Marked as done and saved to your seen/revision history. Send /lc for another question.")
}

func (s *Service) cmdSkip(ctx context.Context, chatID int64) error {
	settings, err := s.store.GetChatSettings(ctx, chatID)
	if err != nil {
		return err
	}
	if settings.CurrentQuestion == nil {
		return s.tgClient.SendMessage(ctx, chatID, "No active question to skip. Use /lc first.")
	}

	return s.sendUniqueQuestion(ctx, chatID, "Skipped. Here is another LeetCode question:", settings.CurrentQuestion.Slug)
}

func (s *Service) cmdExit(ctx context.Context, chatID int64) error {
	settings, err := s.store.GetChatSettings(ctx, chatID)
	if err != nil {
		return err
	}
	if settings.CurrentQuestion == nil {
		return s.tgClient.SendMessage(ctx, chatID, "No active practice mode. Use /lc when you want a question.")
	}

	if err := s.store.ClearCurrentQuestion(ctx, chatID); err != nil {
		return err
	}

	return s.tgClient.SendMessage(ctx, chatID, "Exited practice mode. Send /lc when you want another question.")
}

func (s *Service) cmdDeleteRevisedQuestion(ctx context.Context, chatID int64, args []string) error {
	if len(args) == 0 {
		return s.tgClient.SendMessage(ctx, chatID, "Usage: /delete <slug>, e.g. /delete two-sum")
	}

	slug := normalizeSlug(args[0])
	if slug == "" {
		return s.tgClient.SendMessage(ctx, chatID, "Usage: /delete <slug>, e.g. /delete two-sum")
	}

	if err := s.store.DeleteAnsweredQuestion(ctx, chatID, slug); err != nil {
		if errors.Is(err, ErrAnsweredQuestionNotFound) {
			return s.tgClient.SendMessage(ctx, chatID, "I couldn't find that slug in your revised list. Use /answered to see available slugs.")
		}
		return err
	}
	if err := s.store.RemoveServedQuestion(ctx, chatID, slug); err != nil {
		return err
	}

	settings, err := s.store.GetChatSettings(ctx, chatID)
	if err != nil {
		return err
	}
	if settings.CurrentQuestion != nil && settings.CurrentQuestion.Slug == slug {
		if err := s.store.ClearCurrentQuestion(ctx, chatID); err != nil {
			return err
		}
	}

	return s.tgClient.SendMessage(ctx, chatID, fmt.Sprintf("Deleted %q from revised history and seen set.", slug))
}

func (s *Service) cmdAnsweredHistory(ctx context.Context, chatID int64, args []string) error {
	limit := 10
	if len(args) > 0 {
		parsed, err := parsePositiveLimit(args[0], 50)
		if err != nil {
			return s.tgClient.SendMessage(ctx, chatID, "Usage: /answered [limit], e.g. /answered 10")
		}
		limit = parsed
	}

	items, err := s.store.ListAnsweredQuestions(ctx, chatID, limit)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return s.tgClient.SendMessage(ctx, chatID, "No answered questions yet. Use /lc and either solve correctly or /done.")
	}

	lines := make([]string, 0, len(items)+2)
	lines = append(lines, "Answered questions (latest first):")
	for i, item := range items {
		last := "unknown"
		if !item.LastAnsweredAt.IsZero() {
			last = item.LastAnsweredAt.UTC().Format("2006-01-02")
		}
		lines = append(lines, fmt.Sprintf("%d. %s (%s) | slug: %s | attempts: %d | last: %s",
			i+1, item.Title, item.Difficulty, item.Slug, item.Attempts, last))
	}
	lines = append(lines, "Use /revise <slug> to revisit a specific question, or /revise for a random one.")

	return s.tgClient.SendMessage(ctx, chatID, strings.Join(lines, "\n"))
}

func (s *Service) cmdRevise(ctx context.Context, chatID int64, args []string) error {
	var (
		q   Question
		err error
	)

	if len(args) > 0 {
		slug := normalizeSlug(args[0])
		if slug == "" {
			return s.tgClient.SendMessage(ctx, chatID, "Usage: /revise <slug>, e.g. /revise two-sum")
		}
		q, err = s.store.GetAnsweredQuestion(ctx, chatID, slug)
		if err != nil {
			if errors.Is(err, ErrAnsweredQuestionNotFound) {
				return s.tgClient.SendMessage(ctx, chatID, "I couldn't find that slug in your answered history. Use /answered to see available slugs.")
			}
			return err
		}
	} else {
		items, listErr := s.store.ListAnsweredQuestions(ctx, chatID, 50)
		if listErr != nil {
			return listErr
		}
		if len(items) == 0 {
			return s.tgClient.SendMessage(ctx, chatID, "No answered questions to revise yet. Complete one first with /lc and /done (or a correct attempt).")
		}
		idx := int(s.nowFn().UnixNano() % int64(len(items)))
		if idx < 0 {
			idx = -idx
		}
		q = items[idx].Question
	}

	if err := s.store.SetCurrentQuestion(ctx, chatID, q); err != nil {
		return err
	}

	prompt, err := s.questions.QuestionPrompt(ctx, q.Slug)
	if err != nil {
		s.logger.Printf("revision question prompt lookup failed for slug=%s: %v", q.Slug, err)
		prompt = ""
	}

	msg := formatQuestionMarkdown("Revision question from your history:", "", q, prompt)
	return s.tgClient.SendMarkdownMessage(ctx, chatID, msg)
}

func (s *Service) cmdDailyOn(ctx context.Context, chatID int64, args []string) error {
	settings, err := s.store.GetChatSettings(ctx, chatID)
	if err != nil {
		return err
	}

	hhmm := settings.DailyTime
	if hhmm == "" {
		hhmm = s.defaultDailyHH
	}
	if len(args) > 0 {
		hhmm, err = normalizeHHMM(args[0])
		if err != nil {
			return s.tgClient.SendMessage(ctx, chatID, "Invalid time. Use 24h HH:MM, e.g. /daily_on 20:30")
		}
	}

	if err := s.store.UpsertDailySettings(ctx, chatID, true, hhmm, s.defaultTZ); err != nil {
		return err
	}

	return s.tgClient.SendMessage(ctx, chatID, fmt.Sprintf("Daily question is ON at %s %s. Use /daily_off to stop.", hhmm, tzLabel(s.defaultTZ)))
}

func (s *Service) cmdDailyOff(ctx context.Context, chatID int64) error {
	settings, err := s.store.GetChatSettings(ctx, chatID)
	if err != nil {
		return err
	}

	hhmm := settings.DailyTime
	if hhmm == "" {
		hhmm = s.defaultDailyHH
	}

	if err := s.store.UpsertDailySettings(ctx, chatID, false, hhmm, s.defaultTZ); err != nil {
		return err
	}

	return s.tgClient.SendMessage(ctx, chatID, "Daily question is OFF. Use /daily_on to re-enable.")
}

func (s *Service) cmdDailyTime(ctx context.Context, chatID int64, args []string) error {
	if len(args) == 0 {
		return s.tgClient.SendMessage(ctx, chatID, "Usage: /daily_time HH:MM (24h), e.g. /daily_time 21:00")
	}

	hhmm, err := normalizeHHMM(args[0])
	if err != nil {
		return s.tgClient.SendMessage(ctx, chatID, "Invalid time. Use 24h HH:MM, e.g. /daily_time 21:00")
	}

	if err := s.store.UpsertDailySettings(ctx, chatID, true, hhmm, s.defaultTZ); err != nil {
		return err
	}

	return s.tgClient.SendMessage(ctx, chatID, fmt.Sprintf("Daily time set to %s %s and notifications are ON.", hhmm, tzLabel(s.defaultTZ)))
}

func (s *Service) cmdDailyStatus(ctx context.Context, chatID int64) error {
	settings, err := s.store.GetChatSettings(ctx, chatID)
	if err != nil {
		return err
	}

	status := "OFF"
	if settings.DailyEnabled {
		status = "ON"
	}

	hhmm := settings.DailyTime
	if hhmm == "" {
		hhmm = s.defaultDailyHH
	}
	zone := settings.Timezone
	if zone == "" {
		zone = s.defaultTZ
	}

	msg := fmt.Sprintf("Daily status: %s\nTime: %s\nTimezone: %s", status, hhmm, tzLabel(zone))
	return s.tgClient.SendMessage(ctx, chatID, msg)
}

func normalizeCommand(token string) string {
	if idx := strings.Index(token, "@"); idx >= 0 {
		token = token[:idx]
	}
	return strings.ToLower(token)
}

func normalizeHHMM(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("time is empty")
	}

	for _, layout := range []string{"15:04", "15"} {
		t, err := time.Parse(layout, raw)
		if err == nil {
			return t.Format("15:04"), nil
		}
	}

	return "", fmt.Errorf("invalid time %q", raw)
}

func parsePositiveLimit(raw string, max int) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 || value > max {
		return 0, fmt.Errorf("invalid limit")
	}
	return value, nil
}

func normalizeSlug(raw string) string {
	v := strings.TrimSpace(strings.ToLower(raw))
	v = strings.Trim(v, "/")
	if v == "" {
		return ""
	}

	if idx := strings.Index(v, "leetcode.com/problems/"); idx >= 0 {
		v = v[idx+len("leetcode.com/problems/"):]
	}
	if idx := strings.IndexAny(v, "/?#"); idx >= 0 {
		v = v[:idx]
	}
	return strings.Trim(v, "/")
}

func tzLabel(tz string) string {
	if tz == "Asia/Singapore" {
		return "SGT"
	}
	return tz
}

func helpText() string {
	return strings.TrimSpace(`Commands:
/lc - Get a random LeetCode question
/done - Mark current question complete and save it to seen/revision history
/skip - Skip the current question without adding it to seen history
/exit - Exit active /lc practice mode
/delete <slug> - Remove a question from revised history and seen set
/answered [limit] - List previously answered questions
/history [limit] - Alias of /answered
/revise [slug] - Revisit an answered question (random if slug omitted)
/daily_on [HH:MM] - Enable daily question in SGT (default 20:00)
/daily_off - Disable daily question
/daily_time HH:MM - Set daily time in SGT and enable
/daily_status - Show current daily schedule

After /lc, send your solution idea in words/pseudocode and I will evaluate it with AI (fallback: heuristic).`)
}
