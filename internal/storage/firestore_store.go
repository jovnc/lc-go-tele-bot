package storage

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	chatsCollectionName    = "chats"
	servedSubcollName      = "served_questions"
	answeredSubcollName    = "answered_questions"
	resetBatchCommitSize   = 450
	maxAnsweredListResults = 50
)

var ErrAnsweredQuestionNotFound = errors.New("answered question not found")

type Store struct {
	client           *firestore.Client
	defaultDailyTime string
	defaultDailyTZ   string
}

func NewStore(client *firestore.Client, defaultDailyTime, defaultDailyTZ string) *Store {
	return &Store{
		client:           client,
		defaultDailyTime: defaultDailyTime,
		defaultDailyTZ:   defaultDailyTZ,
	}
}

type QuestionRef struct {
	Slug       string `firestore:"slug"`
	Title      string `firestore:"title"`
	Difficulty string `firestore:"difficulty"`
	URL        string `firestore:"url"`
}

type AnsweredQuestion struct {
	Slug            string    `firestore:"slug"`
	Title           string    `firestore:"title"`
	Difficulty      string    `firestore:"difficulty"`
	URL             string    `firestore:"url"`
	FirstAnsweredAt time.Time `firestore:"first_answered_at"`
	LastAnsweredAt  time.Time `firestore:"last_answered_at"`
	Attempts        int       `firestore:"attempts"`
}

type ChatSettings struct {
	ChatID          int64        `firestore:"chat_id"`
	DailyEnabled    bool         `firestore:"daily_enabled"`
	DailyTime       string       `firestore:"daily_time"`
	Timezone        string       `firestore:"timezone"`
	CurrentQuestion *QuestionRef `firestore:"current_question,omitempty"`
	LastDailySentOn string       `firestore:"last_daily_sent_on"`
	UpdatedAt       time.Time    `firestore:"updated_at"`
}

func (s *Store) GetChatSettings(ctx context.Context, chatID int64) (ChatSettings, error) {
	snap, err := s.chatDoc(chatID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return s.defaultSettings(chatID), nil
		}
		return ChatSettings{}, fmt.Errorf("get chat settings: %w", err)
	}

	var settings ChatSettings
	if err := snap.DataTo(&settings); err != nil {
		return ChatSettings{}, fmt.Errorf("decode chat settings: %w", err)
	}

	if settings.ChatID == 0 {
		settings.ChatID = chatID
	}
	if settings.DailyTime == "" {
		settings.DailyTime = s.defaultDailyTime
	}
	if settings.Timezone == "" {
		settings.Timezone = s.defaultDailyTZ
	}

	return settings, nil
}

func (s *Store) UpsertDailySettings(ctx context.Context, chatID int64, enabled bool, hhmm, tz string) error {
	if hhmm == "" {
		hhmm = s.defaultDailyTime
	}
	if tz == "" {
		tz = s.defaultDailyTZ
	}

	_, err := s.chatDoc(chatID).Set(ctx, map[string]any{
		"chat_id":       chatID,
		"daily_enabled": enabled,
		"daily_time":    hhmm,
		"timezone":      tz,
		"updated_at":    firestore.ServerTimestamp,
	}, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("upsert daily settings: %w", err)
	}
	return nil
}

func (s *Store) SetCurrentQuestion(ctx context.Context, chatID int64, q QuestionRef) error {
	_, err := s.chatDoc(chatID).Set(ctx, map[string]any{
		"chat_id":          chatID,
		"current_question": q,
		"updated_at":       firestore.ServerTimestamp,
	}, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("set current question: %w", err)
	}
	return nil
}

func (s *Store) ClearCurrentQuestion(ctx context.Context, chatID int64) error {
	_, err := s.chatDoc(chatID).Set(ctx, map[string]any{
		"chat_id":          chatID,
		"current_question": firestore.Delete,
		"updated_at":       firestore.ServerTimestamp,
	}, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("clear current question: %w", err)
	}
	return nil
}

func (s *Store) MarkDailySent(ctx context.Context, chatID int64, day string) error {
	_, err := s.chatDoc(chatID).Set(ctx, map[string]any{
		"chat_id":            chatID,
		"last_daily_sent_on": day,
		"updated_at":         firestore.ServerTimestamp,
	}, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("mark daily sent: %w", err)
	}
	return nil
}

func (s *Store) MarkQuestionAnswered(ctx context.Context, chatID int64, q QuestionRef) error {
	if q.Slug == "" {
		return fmt.Errorf("mark question answered: slug is empty")
	}

	ref := s.chatDoc(chatID).Collection(answeredSubcollName).Doc(q.Slug)
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		_, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			return tx.Set(ref, map[string]any{
				"slug":              q.Slug,
				"title":             q.Title,
				"difficulty":        q.Difficulty,
				"url":               q.URL,
				"attempts":          1,
				"first_answered_at": firestore.ServerTimestamp,
				"last_answered_at":  firestore.ServerTimestamp,
			})
		}
		if err != nil {
			return err
		}

		return tx.Set(ref, map[string]any{
			"slug":             q.Slug,
			"title":            q.Title,
			"difficulty":       q.Difficulty,
			"url":              q.URL,
			"attempts":         firestore.Increment(int64(1)),
			"last_answered_at": firestore.ServerTimestamp,
		}, firestore.MergeAll)
	})
	if err != nil {
		return fmt.Errorf("mark question answered: %w", err)
	}
	return nil
}

func (s *Store) ListAnsweredQuestions(ctx context.Context, chatID int64, limit int) ([]AnsweredQuestion, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > maxAnsweredListResults {
		limit = maxAnsweredListResults
	}

	iter := s.chatDoc(chatID).Collection(answeredSubcollName).
		OrderBy("last_answered_at", firestore.Desc).
		Limit(limit).
		Documents(ctx)
	defer iter.Stop()

	out := make([]AnsweredQuestion, 0, limit)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list answered questions: %w", err)
		}

		var item AnsweredQuestion
		if err := doc.DataTo(&item); err != nil {
			return nil, fmt.Errorf("decode answered question: %w", err)
		}
		if item.Slug == "" {
			item.Slug = doc.Ref.ID
		}
		if item.Attempts == 0 {
			item.Attempts = 1
		}
		out = append(out, item)
	}

	return out, nil
}

func (s *Store) GetAnsweredQuestion(ctx context.Context, chatID int64, slug string) (QuestionRef, error) {
	snap, err := s.chatDoc(chatID).Collection(answeredSubcollName).Doc(slug).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return QuestionRef{}, ErrAnsweredQuestionNotFound
		}
		return QuestionRef{}, fmt.Errorf("get answered question: %w", err)
	}

	var q QuestionRef
	if err := snap.DataTo(&q); err != nil {
		return QuestionRef{}, fmt.Errorf("decode answered question: %w", err)
	}
	if q.Slug == "" {
		q.Slug = slug
	}
	return q, nil
}

func (s *Store) DeleteAnsweredQuestion(ctx context.Context, chatID int64, slug string) error {
	ref := s.chatDoc(chatID).Collection(answeredSubcollName).Doc(slug)
	if _, err := ref.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return ErrAnsweredQuestionNotFound
		}
		return fmt.Errorf("get answered question before delete: %w", err)
	}

	if _, err := ref.Delete(ctx); err != nil {
		return fmt.Errorf("delete answered question: %w", err)
	}
	return nil
}

func (s *Store) AddServedQuestion(ctx context.Context, chatID int64, q QuestionRef) error {
	_, err := s.chatDoc(chatID).Collection(servedSubcollName).Doc(q.Slug).Set(ctx, map[string]any{
		"slug":       q.Slug,
		"title":      q.Title,
		"difficulty": q.Difficulty,
		"url":        q.URL,
		"asked_at":   firestore.ServerTimestamp,
	})
	if err != nil {
		return fmt.Errorf("add served question: %w", err)
	}
	return nil
}

func (s *Store) RemoveServedQuestion(ctx context.Context, chatID int64, slug string) error {
	_, err := s.chatDoc(chatID).Collection(servedSubcollName).Doc(slug).Delete(ctx)
	if err != nil && status.Code(err) != codes.NotFound {
		return fmt.Errorf("remove served question: %w", err)
	}
	return nil
}

func (s *Store) SeenQuestionSet(ctx context.Context, chatID int64) (map[string]struct{}, error) {
	iter := s.chatDoc(chatID).Collection(servedSubcollName).Documents(ctx)
	defer iter.Stop()

	seen := make(map[string]struct{})
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("load served questions: %w", err)
		}
		seen[doc.Ref.ID] = struct{}{}
	}

	return seen, nil
}

func (s *Store) ResetServedQuestions(ctx context.Context, chatID int64) error {
	iter := s.chatDoc(chatID).Collection(servedSubcollName).Documents(ctx)
	defer iter.Stop()

	batch := s.client.Batch()
	ops := 0

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("list served questions: %w", err)
		}

		batch.Delete(doc.Ref)
		ops++

		if ops == resetBatchCommitSize {
			if _, err := batch.Commit(ctx); err != nil {
				return fmt.Errorf("commit reset batch: %w", err)
			}
			batch = s.client.Batch()
			ops = 0
		}
	}

	if ops > 0 {
		if _, err := batch.Commit(ctx); err != nil {
			return fmt.Errorf("commit final reset batch: %w", err)
		}
	}

	return nil
}

func (s *Store) ListDailyEnabledChats(ctx context.Context) ([]ChatSettings, error) {
	iter := s.client.Collection(chatsCollectionName).Where("daily_enabled", "==", true).Documents(ctx)
	defer iter.Stop()

	out := make([]ChatSettings, 0, 32)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query daily chats: %w", err)
		}

		var item ChatSettings
		if err := doc.DataTo(&item); err != nil {
			return nil, fmt.Errorf("decode daily chat: %w", err)
		}
		if item.ChatID == 0 {
			parsed, parseErr := strconv.ParseInt(doc.Ref.ID, 10, 64)
			if parseErr == nil {
				item.ChatID = parsed
			}
		}
		if item.ChatID == 0 {
			continue
		}
		if item.DailyTime == "" {
			item.DailyTime = s.defaultDailyTime
		}
		if item.Timezone == "" {
			item.Timezone = s.defaultDailyTZ
		}
		out = append(out, item)
	}

	return out, nil
}

func (s *Store) chatDoc(chatID int64) *firestore.DocumentRef {
	return s.client.Collection(chatsCollectionName).Doc(strconv.FormatInt(chatID, 10))
}

func (s *Store) defaultSettings(chatID int64) ChatSettings {
	return ChatSettings{
		ChatID:       chatID,
		DailyEnabled: false,
		DailyTime:    s.defaultDailyTime,
		Timezone:     s.defaultDailyTZ,
	}
}
