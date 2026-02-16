package bot

import (
	"context"
	"errors"
	"time"

	"telegram-leetcode-bot/internal/bot/commands"
)

type commandDeps struct {
	service *Service
}

func newCommandHandler(service *Service) *commands.Handler {
	return commands.NewHandler(&commandDeps{service: service})
}

func (d *commandDeps) SendMessage(ctx context.Context, chatID int64, text string) error {
	return d.service.tgClient.SendMessage(ctx, chatID, text)
}

func (d *commandDeps) SendRichMessage(ctx context.Context, chatID int64, text string) error {
	return d.service.tgClient.SendRichMessage(ctx, chatID, text)
}

func (d *commandDeps) GetChatSettings(ctx context.Context, chatID int64) (commands.ChatSettings, error) {
	settings, err := d.service.store.GetChatSettings(ctx, chatID)
	if err != nil {
		return commands.ChatSettings{}, err
	}
	return toCommandChatSettings(settings), nil
}

func (d *commandDeps) UpsertDailySettings(ctx context.Context, chatID int64, enabled bool, hhmm, tz string) error {
	return d.service.store.UpsertDailySettings(ctx, chatID, enabled, hhmm, tz)
}

func (d *commandDeps) SetCurrentQuestion(ctx context.Context, chatID int64, q commands.Question) error {
	return d.service.store.SetCurrentQuestion(ctx, chatID, fromCommandQuestion(q))
}

func (d *commandDeps) ClearCurrentQuestion(ctx context.Context, chatID int64) error {
	return d.service.store.ClearCurrentQuestion(ctx, chatID)
}

func (d *commandDeps) DeleteAnsweredQuestion(ctx context.Context, chatID int64, slug string) error {
	return d.service.store.DeleteAnsweredQuestion(ctx, chatID, slug)
}

func (d *commandDeps) RemoveServedQuestion(ctx context.Context, chatID int64, slug string) error {
	return d.service.store.RemoveServedQuestion(ctx, chatID, slug)
}

func (d *commandDeps) ListAnsweredQuestions(ctx context.Context, chatID int64, limit int) ([]commands.AnsweredQuestion, error) {
	items, err := d.service.store.ListAnsweredQuestions(ctx, chatID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]commands.AnsweredQuestion, 0, len(items))
	for _, item := range items {
		out = append(out, commands.AnsweredQuestion{
			Question:        toCommandQuestion(item.Question),
			FirstAnsweredAt: item.FirstAnsweredAt,
			LastAnsweredAt:  item.LastAnsweredAt,
			Attempts:        item.Attempts,
		})
	}
	return out, nil
}

func (d *commandDeps) GetAnsweredQuestion(ctx context.Context, chatID int64, slug string) (commands.Question, error) {
	q, err := d.service.store.GetAnsweredQuestion(ctx, chatID, slug)
	if err != nil {
		return commands.Question{}, err
	}
	return toCommandQuestion(q), nil
}

func (d *commandDeps) QuestionPrompt(ctx context.Context, slug string) (string, error) {
	return d.service.questions.QuestionPrompt(ctx, slug)
}

func (d *commandDeps) SendUniqueQuestion(ctx context.Context, chatID int64, intro string, transientExclude ...string) error {
	return d.service.sendUniqueQuestion(ctx, chatID, intro, transientExclude...)
}

func (d *commandDeps) PersistCompletedQuestion(ctx context.Context, chatID int64, q commands.Question) error {
	return d.service.persistCompletedQuestion(ctx, chatID, fromCommandQuestion(q))
}

func (d *commandDeps) SendHint(ctx context.Context, chatID int64, learnerContext string) error {
	return d.service.sendHintForChat(ctx, chatID, learnerContext)
}

func (d *commandDeps) Now() time.Time {
	return d.service.nowFn()
}

func (d *commandDeps) DefaultDailyHH() string {
	return d.service.defaultDailyHH
}

func (d *commandDeps) DefaultTZ() string {
	return d.service.defaultTZ
}

func (d *commandDeps) Logf(format string, args ...any) {
	d.service.logger.Printf(format, args...)
}

func (d *commandDeps) IsAnsweredQuestionNotFound(err error) bool {
	return errors.Is(err, ErrAnsweredQuestionNotFound)
}

func (d *commandDeps) FormatQuestionMessage(ctx context.Context, intro, note string, q commands.Question, prompt string) string {
	prompt = d.service.formatQuestionPrompt(ctx, fromCommandQuestion(q), prompt)
	return formatQuestionMessage(intro, note, fromCommandQuestion(q), prompt)
}

func toCommandQuestion(q Question) commands.Question {
	return commands.Question{
		Slug:       q.Slug,
		Title:      q.Title,
		Difficulty: q.Difficulty,
		URL:        q.URL,
	}
}

func fromCommandQuestion(q commands.Question) Question {
	return Question{
		Slug:       q.Slug,
		Title:      q.Title,
		Difficulty: q.Difficulty,
		URL:        q.URL,
	}
}

func toCommandChatSettings(in ChatSettings) commands.ChatSettings {
	out := commands.ChatSettings{
		ChatID:          in.ChatID,
		DailyEnabled:    in.DailyEnabled,
		DailyTime:       in.DailyTime,
		Timezone:        in.Timezone,
		LastDailySentOn: in.LastDailySentOn,
	}
	if in.CurrentQuestion != nil {
		q := toCommandQuestion(*in.CurrentQuestion)
		out.CurrentQuestion = &q
	}
	return out
}
