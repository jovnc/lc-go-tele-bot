package adapters

import (
	"context"
	"errors"

	"telegram-leetcode-bot/internal/bot"
	"telegram-leetcode-bot/internal/leetcode"
	"telegram-leetcode-bot/internal/storage"
)

func NewLeetCodeProvider(client *leetcode.Client) bot.QuestionProvider {
	return &leetCodeProvider{client: client}
}

type leetCodeProvider struct {
	client *leetcode.Client
}

func (p *leetCodeProvider) RandomQuestion(ctx context.Context, seen map[string]struct{}) (bot.Question, error) {
	q, err := p.client.RandomQuestion(ctx, seen)
	if err != nil {
		if errors.Is(err, leetcode.ErrNoUnseenQuestions) {
			return bot.Question{}, bot.ErrNoUnseenQuestions
		}
		return bot.Question{}, err
	}
	return bot.Question{
		Slug:       q.Slug,
		Title:      q.Title,
		Difficulty: q.Difficulty,
		URL:        q.URL,
	}, nil
}

func (p *leetCodeProvider) AllQuestions(ctx context.Context) ([]bot.Question, error) {
	all, err := p.client.AllQuestions(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]bot.Question, 0, len(all))
	for _, q := range all {
		out = append(out, bot.Question{
			Slug:       q.Slug,
			Title:      q.Title,
			Difficulty: q.Difficulty,
			URL:        q.URL,
		})
	}
	return out, nil
}

func (p *leetCodeProvider) QuestionPrompt(ctx context.Context, slug string) (string, error) {
	return p.client.QuestionPrompt(ctx, slug)
}

func NewFirestoreStateStore(store *storage.Store) bot.StateStore {
	return &firestoreStateStore{store: store}
}

type firestoreStateStore struct {
	store *storage.Store
}

func (s *firestoreStateStore) GetChatSettings(ctx context.Context, chatID int64) (bot.ChatSettings, error) {
	item, err := s.store.GetChatSettings(ctx, chatID)
	if err != nil {
		return bot.ChatSettings{}, err
	}
	return mapChatSettings(item), nil
}

func (s *firestoreStateStore) UpsertDailySettings(ctx context.Context, chatID int64, enabled bool, hhmm, tz string) error {
	return s.store.UpsertDailySettings(ctx, chatID, enabled, hhmm, tz)
}

func (s *firestoreStateStore) SetCurrentQuestion(ctx context.Context, chatID int64, q bot.Question) error {
	return s.store.SetCurrentQuestion(ctx, chatID, mapQuestionOut(q))
}

func (s *firestoreStateStore) ClearCurrentQuestion(ctx context.Context, chatID int64) error {
	return s.store.ClearCurrentQuestion(ctx, chatID)
}

func (s *firestoreStateStore) MarkDailySent(ctx context.Context, chatID int64, day string) error {
	return s.store.MarkDailySent(ctx, chatID, day)
}

func (s *firestoreStateStore) MarkQuestionAnswered(ctx context.Context, chatID int64, q bot.Question) error {
	return s.store.MarkQuestionAnswered(ctx, chatID, mapQuestionOut(q))
}

func (s *firestoreStateStore) DeleteAnsweredQuestion(ctx context.Context, chatID int64, slug string) error {
	if err := s.store.DeleteAnsweredQuestion(ctx, chatID, slug); err != nil {
		if errors.Is(err, storage.ErrAnsweredQuestionNotFound) {
			return bot.ErrAnsweredQuestionNotFound
		}
		return err
	}
	return nil
}

func (s *firestoreStateStore) AddServedQuestion(ctx context.Context, chatID int64, q bot.Question) error {
	return s.store.AddServedQuestion(ctx, chatID, mapQuestionOut(q))
}

func (s *firestoreStateStore) RemoveServedQuestion(ctx context.Context, chatID int64, slug string) error {
	return s.store.RemoveServedQuestion(ctx, chatID, slug)
}

func (s *firestoreStateStore) SeenQuestionSet(ctx context.Context, chatID int64) (map[string]struct{}, error) {
	return s.store.SeenQuestionSet(ctx, chatID)
}

func (s *firestoreStateStore) ResetServedQuestions(ctx context.Context, chatID int64) error {
	return s.store.ResetServedQuestions(ctx, chatID)
}

func (s *firestoreStateStore) ListDailyEnabledChats(ctx context.Context) ([]bot.ChatSettings, error) {
	items, err := s.store.ListDailyEnabledChats(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]bot.ChatSettings, 0, len(items))
	for _, item := range items {
		out = append(out, mapChatSettings(item))
	}
	return out, nil
}

func (s *firestoreStateStore) ListAnsweredQuestions(ctx context.Context, chatID int64, limit int) ([]bot.AnsweredQuestion, error) {
	items, err := s.store.ListAnsweredQuestions(ctx, chatID, limit)
	if err != nil {
		return nil, err
	}

	out := make([]bot.AnsweredQuestion, 0, len(items))
	for _, item := range items {
		out = append(out, bot.AnsweredQuestion{
			Question: bot.Question{
				Slug:       item.Slug,
				Title:      item.Title,
				Difficulty: item.Difficulty,
				URL:        item.URL,
			},
			FirstAnsweredAt: item.FirstAnsweredAt,
			LastAnsweredAt:  item.LastAnsweredAt,
			Attempts:        item.Attempts,
		})
	}

	return out, nil
}

func (s *firestoreStateStore) GetAnsweredQuestion(ctx context.Context, chatID int64, slug string) (bot.Question, error) {
	item, err := s.store.GetAnsweredQuestion(ctx, chatID, slug)
	if err != nil {
		if errors.Is(err, storage.ErrAnsweredQuestionNotFound) {
			return bot.Question{}, bot.ErrAnsweredQuestionNotFound
		}
		return bot.Question{}, err
	}

	return mapQuestionIn(item), nil
}

func mapChatSettings(item storage.ChatSettings) bot.ChatSettings {
	mapped := bot.ChatSettings{
		ChatID:          item.ChatID,
		DailyEnabled:    item.DailyEnabled,
		DailyTime:       item.DailyTime,
		Timezone:        item.Timezone,
		LastDailySentOn: item.LastDailySentOn,
	}
	if item.CurrentQuestion != nil {
		q := mapQuestionIn(*item.CurrentQuestion)
		mapped.CurrentQuestion = &q
	}
	return mapped
}

func mapQuestionIn(in storage.QuestionRef) bot.Question {
	return bot.Question{
		Slug:       in.Slug,
		Title:      in.Title,
		Difficulty: in.Difficulty,
		URL:        in.URL,
	}
}

func mapQuestionOut(in bot.Question) storage.QuestionRef {
	return storage.QuestionRef{
		Slug:       in.Slug,
		Title:      in.Title,
		Difficulty: in.Difficulty,
		URL:        in.URL,
	}
}
