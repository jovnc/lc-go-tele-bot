package commands

import "context"

func (h *Handler) cmdHelp(ctx context.Context, chatID int64) error {
	return h.deps.SendMessage(ctx, chatID, helpText())
}

func helpText() string {
	return `Commands:
/lc - Get a random LeetCode question
/hint [context] - Get a hint for the active question
/done - Mark current question complete and save it to seen/revision history
/skip - Skip the current question without adding it to seen history
/exit - Exit active /lc practice mode
/delete <slug> - Remove a question from revised history and seen set
/answered [limit] - List previously answered questions
/revise [slug] - Revisit an answered question (random if slug omitted)
/daily_on [HH:MM] - Enable daily question in SGT (default 20:00)
/daily_off - Disable daily question
/daily_time HH:MM - Set daily time in SGT and enable
/daily_status - Show current daily schedule

After /lc, send your approach for evaluation.
Use /hint anytime in active practice mode.`
}
