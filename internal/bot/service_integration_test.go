package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"
)

type fakeTelegramClient struct {
	messages  map[int64][]string
	richCount map[int64]int
}

func newFakeTelegramClient() *fakeTelegramClient {
	return &fakeTelegramClient{
		messages:  make(map[int64][]string),
		richCount: make(map[int64]int),
	}
}

func (f *fakeTelegramClient) SendMessage(_ context.Context, chatID int64, text string) error {
	f.messages[chatID] = append(f.messages[chatID], text)
	return nil
}

func (f *fakeTelegramClient) SendRichMessage(_ context.Context, chatID int64, text string) error {
	f.messages[chatID] = append(f.messages[chatID], text)
	f.richCount[chatID]++
	return nil
}

type fakeQuestionProvider struct {
	questions []Question
}

func (f *fakeQuestionProvider) RandomQuestion(_ context.Context, seen map[string]struct{}) (Question, error) {
	for _, q := range f.questions {
		if _, ok := seen[q.Slug]; ok {
			continue
		}
		return q, nil
	}
	return Question{}, ErrNoUnseenQuestions
}

func (f *fakeQuestionProvider) AllQuestions(_ context.Context) ([]Question, error) {
	out := make([]Question, len(f.questions))
	copy(out, f.questions)
	return out, nil
}

func (f *fakeQuestionProvider) QuestionPrompt(_ context.Context, slug string) (string, error) {
	for _, q := range f.questions {
		if q.Slug == slug {
			return q.Title + " statement", nil
		}
	}
	return "Question statement unavailable.", nil
}

type fakeCoach struct {
	review            AnswerReview
	reviewErr         error
	hint              string
	hintErr           error
	questionPrompt    string
	questionPromptErr error
}

func (f *fakeCoach) ReviewAnswer(_ context.Context, _ Question, _ string) (AnswerReview, error) {
	if f.reviewErr != nil {
		return AnswerReview{}, f.reviewErr
	}
	return f.review, nil
}

func (f *fakeCoach) GenerateHint(_ context.Context, _ Question, _ string) (string, error) {
	if f.hintErr != nil {
		return "", f.hintErr
	}
	return f.hint, nil
}

func (f *fakeCoach) FormatQuestion(_ context.Context, _ Question, prompt string) (string, error) {
	if f.questionPromptErr != nil {
		return "", f.questionPromptErr
	}
	if strings.TrimSpace(f.questionPrompt) == "" {
		return prompt, nil
	}
	return f.questionPrompt, nil
}

type memoryStore struct {
	chats    map[int64]ChatSettings
	served   map[int64]map[string]Question
	answered map[int64]map[string]AnsweredQuestion
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		chats:    make(map[int64]ChatSettings),
		served:   make(map[int64]map[string]Question),
		answered: make(map[int64]map[string]AnsweredQuestion),
	}
}

func (m *memoryStore) GetChatSettings(_ context.Context, chatID int64) (ChatSettings, error) {
	if item, ok := m.chats[chatID]; ok {
		return cloneChat(item), nil
	}
	item := ChatSettings{
		ChatID:       chatID,
		DailyEnabled: false,
		DailyTime:    "20:00",
		Timezone:     "Asia/Singapore",
	}
	m.chats[chatID] = item
	return cloneChat(item), nil
}

func (m *memoryStore) UpsertDailySettings(_ context.Context, chatID int64, enabled bool, hhmm, tz string) error {
	item, _ := m.GetChatSettings(context.Background(), chatID)
	item.DailyEnabled = enabled
	item.DailyTime = hhmm
	item.Timezone = tz
	m.chats[chatID] = item
	return nil
}

func (m *memoryStore) SetCurrentQuestion(_ context.Context, chatID int64, q Question) error {
	item, _ := m.GetChatSettings(context.Background(), chatID)
	qCopy := q
	item.CurrentQuestion = &qCopy
	m.chats[chatID] = item
	return nil
}

func (m *memoryStore) ClearCurrentQuestion(_ context.Context, chatID int64) error {
	item, _ := m.GetChatSettings(context.Background(), chatID)
	item.CurrentQuestion = nil
	m.chats[chatID] = item
	return nil
}

func (m *memoryStore) MarkDailySent(_ context.Context, chatID int64, day string) error {
	item, _ := m.GetChatSettings(context.Background(), chatID)
	item.LastDailySentOn = day
	m.chats[chatID] = item
	return nil
}

func (m *memoryStore) MarkQuestionAnswered(_ context.Context, chatID int64, q Question) error {
	if _, ok := m.answered[chatID]; !ok {
		m.answered[chatID] = make(map[string]AnsweredQuestion)
	}
	now := time.Now().UTC()
	entry, exists := m.answered[chatID][q.Slug]
	if !exists {
		entry = AnsweredQuestion{
			Question:        q,
			FirstAnsweredAt: now,
			LastAnsweredAt:  now,
			Attempts:        1,
		}
	} else {
		entry.Question = q
		entry.Attempts++
		entry.LastAnsweredAt = now
	}
	m.answered[chatID][q.Slug] = entry
	return nil
}

func (m *memoryStore) DeleteAnsweredQuestion(_ context.Context, chatID int64, slug string) error {
	if _, ok := m.answered[chatID][slug]; !ok {
		return ErrAnsweredQuestionNotFound
	}
	delete(m.answered[chatID], slug)
	return nil
}

func (m *memoryStore) AddServedQuestion(_ context.Context, chatID int64, q Question) error {
	if _, ok := m.served[chatID]; !ok {
		m.served[chatID] = make(map[string]Question)
	}
	m.served[chatID][q.Slug] = q
	return nil
}

func (m *memoryStore) RemoveServedQuestion(_ context.Context, chatID int64, slug string) error {
	delete(m.served[chatID], slug)
	return nil
}

func (m *memoryStore) SeenQuestionSet(_ context.Context, chatID int64) (map[string]struct{}, error) {
	seen := make(map[string]struct{})
	for slug := range m.served[chatID] {
		seen[slug] = struct{}{}
	}
	return seen, nil
}

func (m *memoryStore) ResetServedQuestions(_ context.Context, chatID int64) error {
	m.served[chatID] = make(map[string]Question)
	return nil
}

func (m *memoryStore) ListDailyEnabledChats(_ context.Context) ([]ChatSettings, error) {
	out := make([]ChatSettings, 0)
	for _, item := range m.chats {
		if item.DailyEnabled {
			out = append(out, cloneChat(item))
		}
	}
	return out, nil
}

func (m *memoryStore) ListAnsweredQuestions(_ context.Context, chatID int64, limit int) ([]AnsweredQuestion, error) {
	if limit <= 0 {
		limit = 10
	}
	items := make([]AnsweredQuestion, 0, len(m.answered[chatID]))
	for _, item := range m.answered[chatID] {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].LastAnsweredAt.After(items[j].LastAnsweredAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (m *memoryStore) GetAnsweredQuestion(_ context.Context, chatID int64, slug string) (Question, error) {
	item, ok := m.answered[chatID][slug]
	if !ok {
		return Question{}, ErrAnsweredQuestionNotFound
	}
	return item.Question, nil
}

func cloneChat(in ChatSettings) ChatSettings {
	out := in
	if in.CurrentQuestion != nil {
		q := *in.CurrentQuestion
		out.CurrentQuestion = &q
	}
	return out
}

func TestWebhookLCUniquenessAndGrading(t *testing.T) {
	tg := newFakeTelegramClient()
	store := newMemoryStore()
	provider := &fakeQuestionProvider{questions: []Question{
		{Slug: "two-sum", Title: "Two Sum", Difficulty: "Easy", URL: "https://leetcode.com/problems/two-sum/"},
		{Slug: "merge-intervals", Title: "Merge Intervals", Difficulty: "Medium", URL: "https://leetcode.com/problems/merge-intervals/"},
	}}

	svc := NewService(
		log.New(bytes.NewBuffer(nil), "", 0),
		tg,
		provider,
		nil,
		store,
		"webhook-secret",
		"cron-secret",
		"20:00",
		"Asia/Singapore",
		nil,
	)

	chatID := int64(42)

	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/lc random"}})
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/lc random"}})
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "Use a hash map, O(n) time, O(n) space. Handle duplicates and empty list."}})

	messages := tg.messages[chatID]
	if len(messages) != 3 {
		t.Fatalf("expected 3 outgoing messages, got %d", len(messages))
	}

	if !strings.Contains(messages[0], "Two Sum") {
		t.Fatalf("first question should be Two Sum, got: %s", messages[0])
	}
	if !strings.Contains(messages[1], "Merge Intervals") {
		t.Fatalf("second question should be Merge Intervals, got: %s", messages[1])
	}
	if !strings.Contains(messages[2], "Score: *") {
		t.Fatalf("grading response missing score: %s", messages[2])
	}
	if !strings.Contains(messages[2], "Source: Heuristic") {
		t.Fatalf("expected heuristic fallback, got: %s", messages[2])
	}
	if !strings.Contains(messages[2], "Not saved yet") {
		t.Fatalf("expected not-saved status for non-correct attempt, got: %s", messages[2])
	}

	settings, _ := store.GetChatSettings(context.Background(), chatID)
	if settings.CurrentQuestion == nil || settings.CurrentQuestion.Title != "Merge Intervals" {
		t.Fatalf("expected current question to stay active for iterative coaching")
	}

	answered, _ := store.ListAnsweredQuestions(context.Background(), chatID, 10)
	if len(answered) != 0 {
		t.Fatalf("expected unanswered attempt to stay out of revised history")
	}

	seen, _ := store.SeenQuestionSet(context.Background(), chatID)
	if len(seen) != 0 {
		t.Fatalf("expected unanswered attempt to stay out of seen history")
	}

	if tg.richCount[chatID] != 3 {
		t.Fatalf("expected all /lc + evaluation responses to use rich formatting; richCount=%d", tg.richCount[chatID])
	}
}

func TestGradingUsesCoachWhenConfigured(t *testing.T) {
	tg := newFakeTelegramClient()
	store := newMemoryStore()
	provider := &fakeQuestionProvider{questions: []Question{
		{Slug: "two-sum", Title: "Two Sum", Difficulty: "Easy", URL: "https://leetcode.com/problems/two-sum/"},
	}}
	coach := &fakeCoach{
		review: AnswerReview{
			Score:    9,
			Feedback: "Great structure and complexity reasoning.",
			Guidance: "Try proving correctness with an invariant on each iteration.",
		},
	}

	svc := NewService(
		log.New(bytes.NewBuffer(nil), "", 0),
		tg,
		provider,
		coach,
		store,
		"webhook-secret",
		"cron-secret",
		"20:00",
		"Asia/Singapore",
		nil,
	)

	chatID := int64(77)

	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/lc random"}})
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "I would use nested loops then optimize"}})

	messages := tg.messages[chatID]
	if len(messages) != 2 {
		t.Fatalf("expected 2 outgoing messages, got %d", len(messages))
	}
	if !strings.Contains(messages[1], "Source: AI") {
		t.Fatalf("grading should come from AI coach, got: %s", messages[1])
	}
	if !strings.Contains(messages[1], "invariant") {
		t.Fatalf("AI guidance missing from grade response: %s", messages[1])
	}
	if !strings.Contains(messages[1], "Correct\\. Saved") {
		t.Fatalf("expected correct status in evaluation output, got: %s", messages[1])
	}

	settings, _ := store.GetChatSettings(context.Background(), chatID)
	if settings.CurrentQuestion != nil {
		t.Fatalf("expected correct answer to clear current question")
	}
	answered, _ := store.ListAnsweredQuestions(context.Background(), chatID, 10)
	if len(answered) != 1 || answered[0].Slug != "two-sum" {
		t.Fatalf("expected correct answer to save question in revised history")
	}
	seen, _ := store.SeenQuestionSet(context.Background(), chatID)
	if len(seen) != 1 {
		t.Fatalf("expected correct answer to save question in seen history")
	}
}

func TestLCUsesAIQuestionFormattingWhenAvailable(t *testing.T) {
	tg := newFakeTelegramClient()
	store := newMemoryStore()
	provider := &fakeQuestionProvider{questions: []Question{
		{Slug: "asteroid-collision", Title: "Asteroid Collision", Difficulty: "Medium", URL: "https://leetcode.com/problems/asteroid-collision/"},
	}}
	coach := &fakeCoach{
		questionPrompt: "## Given\n- Asteroids move at same speed.\n- Sign is direction, absolute value is size.\n\n## Example 1\n```text\nInput: [5,10,-5]\nOutput: [5,10]\n```",
	}

	svc := NewService(
		log.New(bytes.NewBuffer(nil), "", 0),
		tg,
		provider,
		coach,
		store,
		"webhook-secret",
		"cron-secret",
		"20:00",
		"Asia/Singapore",
		nil,
	)

	chatID := int64(178)
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/lc random"}})

	messages := tg.messages[chatID]
	if len(messages) != 1 {
		t.Fatalf("expected 1 outgoing message, got %d", len(messages))
	}
	if !strings.Contains(messages[0], "__*Given*__") {
		t.Fatalf("expected AI heading formatting in /lc output: %s", messages[0])
	}
	if !strings.Contains(messages[0], "• Asteroids move at same speed") {
		t.Fatalf("expected AI bullet formatting in /lc output: %s", messages[0])
	}
	if !strings.Contains(messages[0], "```text") {
		t.Fatalf("expected AI code block formatting in /lc output: %s", messages[0])
	}
	if strings.Contains(messages[0], "Link: ") {
		t.Fatalf("expected no explicit link line in /lc output: %s", messages[0])
	}
}

func TestHintCommandUsesAIWhenAvailable(t *testing.T) {
	tg := newFakeTelegramClient()
	store := newMemoryStore()
	provider := &fakeQuestionProvider{questions: []Question{
		{Slug: "two-sum", Title: "Two Sum", Difficulty: "Easy", URL: "https://leetcode.com/problems/two-sum/"},
	}}
	coach := &fakeCoach{
		hint: "## Direction\n- Track complements in a map.\n\n## Pseudocode\n```go\nfor i, x := range nums {\n    if j, ok := seen[target-x]; ok { return []int{j, i} }\n    seen[x] = i\n}\n```",
	}

	svc := NewService(
		log.New(bytes.NewBuffer(nil), "", 0),
		tg,
		provider,
		coach,
		store,
		"webhook-secret",
		"cron-secret",
		"20:00",
		"Asia/Singapore",
		nil,
	)

	chatID := int64(78)

	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/lc random"}})
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/hint"}})

	messages := tg.messages[chatID]
	if len(messages) != 2 {
		t.Fatalf("expected 2 outgoing messages, got %d", len(messages))
	}
	if !strings.Contains(messages[1], "*💡 Hint*") {
		t.Fatalf("expected hint response, got: %s", messages[1])
	}
	if !strings.Contains(messages[1], "Source: AI") {
		t.Fatalf("expected AI hint source, got: %s", messages[1])
	}
}

func TestHintRequestFromFreeTextUsesHintFlow(t *testing.T) {
	tg := newFakeTelegramClient()
	store := newMemoryStore()
	provider := &fakeQuestionProvider{questions: []Question{
		{Slug: "merge-intervals", Title: "Merge Intervals", Difficulty: "Medium", URL: "https://leetcode.com/problems/merge-intervals/"},
	}}

	svc := NewService(
		log.New(bytes.NewBuffer(nil), "", 0),
		tg,
		provider,
		nil,
		store,
		"webhook-secret",
		"cron-secret",
		"20:00",
		"Asia/Singapore",
		nil,
	)

	chatID := int64(79)

	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/lc random"}})
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "hint please"}})

	messages := tg.messages[chatID]
	if len(messages) != 2 {
		t.Fatalf("expected 2 outgoing messages, got %d", len(messages))
	}
	if !strings.Contains(messages[1], "*💡 Hint*") {
		t.Fatalf("expected hint response, got: %s", messages[1])
	}
	if strings.Contains(messages[1], "*🧠 Evaluation*") {
		t.Fatalf("hint request should not trigger evaluation flow: %s", messages[1])
	}
}

func TestHistoryAndReviseCommands(t *testing.T) {
	tg := newFakeTelegramClient()
	store := newMemoryStore()
	provider := &fakeQuestionProvider{questions: []Question{
		{Slug: "two-sum", Title: "Two Sum", Difficulty: "Easy", URL: "https://leetcode.com/problems/two-sum/"},
		{Slug: "merge-intervals", Title: "Merge Intervals", Difficulty: "Medium", URL: "https://leetcode.com/problems/merge-intervals/"},
	}}

	svc := NewService(
		log.New(bytes.NewBuffer(nil), "", 0),
		tg,
		provider,
		nil,
		store,
		"webhook-secret",
		"cron-secret",
		"20:00",
		"Asia/Singapore",
		nil,
	)

	chatID := int64(88)

	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/lc random"}})
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/done"}})
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/answered"}})
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/revise two-sum"}})

	messages := tg.messages[chatID]
	if len(messages) != 4 {
		t.Fatalf("expected 4 outgoing messages, got %d", len(messages))
	}
	if !strings.Contains(messages[2], "*📚 Answered Questions*") || !strings.Contains(messages[2], "Slug: `two-sum`") {
		t.Fatalf("history output missing expected slug: %s", messages[2])
	}
	if !strings.Contains(messages[3], "Two Sum") || !strings.Contains(messages[3], "__*Problem*__") {
		t.Fatalf("revise output unexpected: %s", messages[3])
	}
}

func TestSkipDoesNotPersistSeenQuestion(t *testing.T) {
	tg := newFakeTelegramClient()
	store := newMemoryStore()
	provider := &fakeQuestionProvider{questions: []Question{
		{Slug: "two-sum", Title: "Two Sum", Difficulty: "Easy", URL: "https://leetcode.com/problems/two-sum/"},
		{Slug: "merge-intervals", Title: "Merge Intervals", Difficulty: "Medium", URL: "https://leetcode.com/problems/merge-intervals/"},
	}}

	svc := NewService(
		log.New(bytes.NewBuffer(nil), "", 0),
		tg,
		provider,
		nil,
		store,
		"webhook-secret",
		"cron-secret",
		"20:00",
		"Asia/Singapore",
		nil,
	)

	chatID := int64(120)
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/lc random"}})
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/skip"}})

	seenAfterSkip, err := store.SeenQuestionSet(context.Background(), chatID)
	if err != nil {
		t.Fatalf("load seen set: %v", err)
	}
	if len(seenAfterSkip) != 0 {
		t.Fatalf("expected skipped question flow to keep seen set empty until /done or correct answer")
	}

	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/lc random"}})

	messages := tg.messages[chatID]
	if len(messages) != 3 {
		t.Fatalf("expected 3 outgoing messages, got %d", len(messages))
	}
	if !strings.Contains(messages[1], "Merge Intervals") {
		t.Fatalf("skip should send next question, got: %s", messages[1])
	}
	if !strings.Contains(messages[2], "Two Sum") {
		t.Fatalf("expected skipped question to remain eligible, got: %s", messages[2])
	}
}

func TestExitClearsActivePracticeMode(t *testing.T) {
	tg := newFakeTelegramClient()
	store := newMemoryStore()
	provider := &fakeQuestionProvider{questions: []Question{
		{Slug: "two-sum", Title: "Two Sum", Difficulty: "Easy", URL: "https://leetcode.com/problems/two-sum/"},
	}}

	svc := NewService(
		log.New(bytes.NewBuffer(nil), "", 0),
		tg,
		provider,
		nil,
		store,
		"webhook-secret",
		"cron-secret",
		"20:00",
		"Asia/Singapore",
		nil,
	)

	chatID := int64(121)
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/lc random"}})
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/exit"}})
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "my approach"}})

	settings, err := store.GetChatSettings(context.Background(), chatID)
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if settings.CurrentQuestion != nil {
		t.Fatalf("expected /exit to clear current question")
	}

	messages := tg.messages[chatID]
	if len(messages) != 3 {
		t.Fatalf("expected 3 outgoing messages, got %d", len(messages))
	}
	if !strings.Contains(messages[1], "Exited practice mode") {
		t.Fatalf("unexpected /exit response: %s", messages[1])
	}
	if !strings.Contains(messages[2], "No active question. Use /lc first.") {
		t.Fatalf("expected no-active-question response after exit, got: %s", messages[2])
	}

	seen, _ := store.SeenQuestionSet(context.Background(), chatID)
	if len(seen) != 0 {
		t.Fatalf("expected /exit flow not to persist question in seen history")
	}
	answered, _ := store.ListAnsweredQuestions(context.Background(), chatID, 10)
	if len(answered) != 0 {
		t.Fatalf("expected /exit flow not to persist question in revised history")
	}
}

func TestDoneSavesQuestionAndClearsCurrent(t *testing.T) {
	tg := newFakeTelegramClient()
	store := newMemoryStore()
	provider := &fakeQuestionProvider{questions: []Question{
		{Slug: "two-sum", Title: "Two Sum", Difficulty: "Easy", URL: "https://leetcode.com/problems/two-sum/"},
	}}

	svc := NewService(
		log.New(bytes.NewBuffer(nil), "", 0),
		tg,
		provider,
		nil,
		store,
		"webhook-secret",
		"cron-secret",
		"20:00",
		"Asia/Singapore",
		nil,
	)

	chatID := int64(123)
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/lc random"}})
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/done"}})

	settings, _ := store.GetChatSettings(context.Background(), chatID)
	if settings.CurrentQuestion != nil {
		t.Fatalf("expected /done to clear current question")
	}
	answered, _ := store.ListAnsweredQuestions(context.Background(), chatID, 10)
	if len(answered) != 1 || answered[0].Slug != "two-sum" {
		t.Fatalf("expected /done to save question in revised history")
	}
	seen, _ := store.SeenQuestionSet(context.Background(), chatID)
	if len(seen) != 1 {
		t.Fatalf("expected /done to save question in seen history")
	}
}

func TestDeleteRemovesQuestionFromRevisedAndSeen(t *testing.T) {
	tg := newFakeTelegramClient()
	store := newMemoryStore()
	provider := &fakeQuestionProvider{questions: []Question{
		{Slug: "two-sum", Title: "Two Sum", Difficulty: "Easy", URL: "https://leetcode.com/problems/two-sum/"},
	}}

	svc := NewService(
		log.New(bytes.NewBuffer(nil), "", 0),
		tg,
		provider,
		nil,
		store,
		"webhook-secret",
		"cron-secret",
		"20:00",
		"Asia/Singapore",
		nil,
	)

	chatID := int64(124)
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/lc random"}})
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/done"}})
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/delete two-sum"}})
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/answered"}})

	seen, _ := store.SeenQuestionSet(context.Background(), chatID)
	if len(seen) != 0 {
		t.Fatalf("expected /delete to remove question from seen history")
	}
	answered, _ := store.ListAnsweredQuestions(context.Background(), chatID, 10)
	if len(answered) != 0 {
		t.Fatalf("expected /delete to remove question from revised history")
	}

	messages := tg.messages[chatID]
	if len(messages) != 4 {
		t.Fatalf("expected 4 outgoing messages, got %d", len(messages))
	}
	if !strings.Contains(messages[2], `Deleted "two-sum"`) {
		t.Fatalf("unexpected delete response: %s", messages[2])
	}
	if !strings.Contains(messages[3], "No answered questions yet") {
		t.Fatalf("expected empty /answered after delete, got: %s", messages[3])
	}
}

func TestUsernameAllowListBlocksUnauthorizedUsers(t *testing.T) {
	tg := newFakeTelegramClient()
	store := newMemoryStore()
	provider := &fakeQuestionProvider{questions: []Question{
		{Slug: "two-sum", Title: "Two Sum", Difficulty: "Easy", URL: "https://leetcode.com/problems/two-sum/"},
	}}

	svc := NewService(
		log.New(bytes.NewBuffer(nil), "", 0),
		tg,
		provider,
		nil,
		store,
		"webhook-secret",
		"cron-secret",
		"20:00",
		"Asia/Singapore",
		[]string{"allowed_user"},
	)

	chatID := int64(122)
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{
		Message: webhookMessage{
			Chat: webhookChat{ID: chatID},
			From: webhookUser{Username: "intruder"},
			Text: "/lc random",
		},
	})
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{
		Message: webhookMessage{
			Chat: webhookChat{ID: chatID},
			From: webhookUser{Username: "allowed_user"},
			Text: "/lc random",
		},
	})

	messages := tg.messages[chatID]
	if len(messages) != 2 {
		t.Fatalf("expected 2 outgoing messages, got %d", len(messages))
	}
	if !strings.Contains(messages[0], "not allowed") {
		t.Fatalf("expected unauthorized response, got: %s", messages[0])
	}
	if !strings.Contains(messages[1], "Two Sum") {
		t.Fatalf("expected allowed user to receive question, got: %s", messages[1])
	}
}

func TestCronDailyDispatchRespectsTimeAndDedupesByDay(t *testing.T) {
	tg := newFakeTelegramClient()
	store := newMemoryStore()
	provider := &fakeQuestionProvider{questions: []Question{
		{Slug: "valid-parentheses", Title: "Valid Parentheses", Difficulty: "Easy", URL: "https://leetcode.com/problems/valid-parentheses/"},
	}}

	svc := NewService(
		log.New(bytes.NewBuffer(nil), "", 0),
		tg,
		provider,
		nil,
		store,
		"webhook-secret",
		"cron-secret",
		"20:00",
		"Asia/Singapore",
		nil,
	)
	// 2026-02-14 12:00 UTC == 20:00 SGT
	svc.nowFn = func() time.Time { return time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC) }

	chatID := int64(99)
	if err := store.UpsertDailySettings(context.Background(), chatID, true, "20:00", "Asia/Singapore"); err != nil {
		t.Fatalf("failed to configure chat: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/cron/daily", nil)
	req.Header.Set("X-Cron-Secret", "cron-secret")
	res := httptest.NewRecorder()
	svc.CronHandler(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	res = httptest.NewRecorder()
	svc.CronHandler(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 on second call, got %d", res.Code)
	}

	messages := tg.messages[chatID]
	if len(messages) != 1 {
		t.Fatalf("expected one daily message due to same-day dedupe, got %d", len(messages))
	}
	if !strings.Contains(messages[0], "Valid Parentheses") {
		t.Fatalf("unexpected daily message: %s", messages[0])
	}
}

func TestCronRejectsInvalidSecret(t *testing.T) {
	svc := NewService(
		log.New(bytes.NewBuffer(nil), "", 0),
		newFakeTelegramClient(),
		&fakeQuestionProvider{},
		nil,
		newMemoryStore(),
		"webhook-secret",
		"cron-secret",
		"20:00",
		"Asia/Singapore",
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/cron/daily", nil)
	req.Header.Set("X-Cron-Secret", "wrong")
	res := httptest.NewRecorder()
	svc.CronHandler(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}

type webhookPayload struct {
	Message webhookMessage `json:"message"`
}

type webhookMessage struct {
	Chat webhookChat `json:"chat"`
	From webhookUser `json:"from"`
	Text string      `json:"text"`
}

type webhookChat struct {
	ID int64 `json:"id"`
}

type webhookUser struct {
	Username string `json:"username"`
}

func callWebhook(t *testing.T, svc *Service, path string, payload webhookPayload) {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	res := httptest.NewRecorder()

	svc.WebhookHandler(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestLCPromptsForTopicThenServesMatchingQuestion(t *testing.T) {
	tg := newFakeTelegramClient()
	store := newMemoryStore()
	provider := &fakeQuestionProvider{questions: []Question{
		{Slug: "binary-tree-level-order-traversal", Title: "Binary Tree Level Order Traversal", Difficulty: "Medium", URL: "https://leetcode.com/problems/binary-tree-level-order-traversal/"},
		{Slug: "two-sum", Title: "Two Sum", Difficulty: "Easy", URL: "https://leetcode.com/problems/two-sum/"},
	}}

	svc := NewService(
		log.New(bytes.NewBuffer(nil), "", 0),
		tg,
		provider,
		nil,
		store,
		"webhook-secret",
		"cron-secret",
		"20:00",
		"Asia/Singapore",
		nil,
	)

	chatID := int64(155)
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "/lc"}})
	callWebhook(t, svc, "/webhook/webhook-secret", webhookPayload{Message: webhookMessage{Chat: webhookChat{ID: chatID}, Text: "tree"}})

	messages := tg.messages[chatID]
	if len(messages) != 2 {
		t.Fatalf("expected 2 outgoing messages, got %d", len(messages))
	}
	if !strings.Contains(messages[0], "Which topic") {
		t.Fatalf("expected topic prompt, got: %s", messages[0])
	}
	if !strings.Contains(messages[1], "Binary Tree Level Order Traversal") {
		t.Fatalf("expected topic-matching question, got: %s", messages[1])
	}
}
