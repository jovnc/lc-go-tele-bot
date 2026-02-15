package bot

import (
	"context"
	"errors"
	"time"
)

var ErrNoUnseenQuestions = errors.New("no unseen questions available")
var ErrAnsweredQuestionNotFound = errors.New("answered question not found")

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

type AnswerReview struct {
	Score    int
	Feedback string
	Guidance string
}

type AnsweredQuestion struct {
	Question
	FirstAnsweredAt time.Time
	LastAnsweredAt  time.Time
	Attempts        int
}

type TelegramSender interface {
	SendMessage(ctx context.Context, chatID int64, text string) error
	SendMarkdownMessage(ctx context.Context, chatID int64, text string) error
}

type QuestionProvider interface {
	RandomQuestion(ctx context.Context, seen map[string]struct{}) (Question, error)
	AllQuestions(ctx context.Context) ([]Question, error)
	QuestionPrompt(ctx context.Context, slug string) (string, error)
}

type Coach interface {
	ReviewAnswer(ctx context.Context, question Question, answer string) (AnswerReview, error)
}

type StateStore interface {
	GetChatSettings(ctx context.Context, chatID int64) (ChatSettings, error)
	UpsertDailySettings(ctx context.Context, chatID int64, enabled bool, hhmm, tz string) error
	SetCurrentQuestion(ctx context.Context, chatID int64, q Question) error
	ClearCurrentQuestion(ctx context.Context, chatID int64) error
	MarkDailySent(ctx context.Context, chatID int64, day string) error
	MarkQuestionAnswered(ctx context.Context, chatID int64, q Question) error
	AddServedQuestion(ctx context.Context, chatID int64, q Question) error
	RemoveServedQuestion(ctx context.Context, chatID int64, slug string) error
	SeenQuestionSet(ctx context.Context, chatID int64) (map[string]struct{}, error)
	ResetServedQuestions(ctx context.Context, chatID int64) error
	ListDailyEnabledChats(ctx context.Context) ([]ChatSettings, error)
	ListAnsweredQuestions(ctx context.Context, chatID int64, limit int) ([]AnsweredQuestion, error)
	GetAnsweredQuestion(ctx context.Context, chatID int64, slug string) (Question, error)
}
