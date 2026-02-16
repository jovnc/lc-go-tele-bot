package commands

import (
	"context"
	"time"
)

type Question struct {
	Slug       string
	Title      string
	Difficulty string
	URL        string
}

type ChatSettings struct {
	ChatID          int64
	DailyEnabled    bool
	DailyTime       string
	Timezone        string
	CurrentQuestion *Question
	LastDailySentOn string
}

type AnsweredQuestion struct {
	Question
	FirstAnsweredAt time.Time
	LastAnsweredAt  time.Time
	Attempts        int
}

type Dependencies interface {
	SendMessage(ctx context.Context, chatID int64, text string) error
	SendRichMessage(ctx context.Context, chatID int64, text string) error

	GetChatSettings(ctx context.Context, chatID int64) (ChatSettings, error)
	UpsertDailySettings(ctx context.Context, chatID int64, enabled bool, hhmm, tz string) error
	SetCurrentQuestion(ctx context.Context, chatID int64, q Question) error
	ClearCurrentQuestion(ctx context.Context, chatID int64) error
	DeleteAnsweredQuestion(ctx context.Context, chatID int64, slug string) error
	RemoveServedQuestion(ctx context.Context, chatID int64, slug string) error
	ListAnsweredQuestions(ctx context.Context, chatID int64, limit int) ([]AnsweredQuestion, error)
	GetAnsweredQuestion(ctx context.Context, chatID int64, slug string) (Question, error)

	QuestionPrompt(ctx context.Context, slug string) (string, error)
	SendUniqueQuestion(ctx context.Context, chatID int64, intro string, transientExclude ...string) error
	PersistCompletedQuestion(ctx context.Context, chatID int64, q Question) error
	SendHint(ctx context.Context, chatID int64, learnerContext string) error

	Now() time.Time
	DefaultDailyHH() string
	DefaultTZ() string
	Logf(format string, args ...any)
	IsAnsweredQuestionNotFound(err error) bool
	FormatQuestionMessage(ctx context.Context, intro, note string, q Question, prompt string) string
}
