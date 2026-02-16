package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"telegram-leetcode-bot/internal/bot/commands"
	"telegram-leetcode-bot/internal/telegram"
)

const correctAnswerScoreThreshold = 8

type Service struct {
	logger         *log.Logger
	tgClient       TelegramSender
	questions      QuestionProvider
	coach          Coach
	store          StateStore
	commandHandler *commands.Handler
	allowedUsers   map[string]struct{}
	webhookSecret  string
	cronSecret     string
	defaultDailyHH string
	defaultTZ      string
	defaultLoc     *time.Location
	nowFn          func() time.Time
}

func NewService(
	logger *log.Logger,
	tgClient TelegramSender,
	questions QuestionProvider,
	coach Coach,
	store StateStore,
	webhookSecret string,
	cronSecret string,
	defaultDailyHH string,
	defaultTZ string,
	allowedUsernames []string,
) *Service {
	loc, err := time.LoadLocation(defaultTZ)
	if err != nil {
		loc = time.FixedZone("UTC+8", 8*3600)
	}

	svc := &Service{
		logger:         logger,
		tgClient:       tgClient,
		questions:      questions,
		coach:          coach,
		store:          store,
		allowedUsers:   buildAllowedUserSet(allowedUsernames),
		webhookSecret:  webhookSecret,
		cronSecret:     cronSecret,
		defaultDailyHH: defaultDailyHH,
		defaultTZ:      defaultTZ,
		defaultLoc:     loc,
		nowFn:          time.Now,
	}
	svc.commandHandler = newCommandHandler(svc)
	return svc
}

func (s *Service) WebhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Path != "/webhook/"+s.webhookSecret {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	defer r.Body.Close()

	var update telegram.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if update.Message != nil {
		if err := s.handleMessage(r.Context(), *update.Message); err != nil {
			s.logger.Printf("handle message failed: %v", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Service) CronHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Header.Get("X-Cron-Secret") != s.cronSecret {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	chats, err := s.store.ListDailyEnabledChats(r.Context())
	if err != nil {
		http.Error(w, "failed to load chats", http.StatusInternalServerError)
		return
	}

	nowUTC := s.nowFn().UTC()
	processed := 0
	sent := 0

	for _, chat := range chats {
		now := nowUTC.In(s.resolveLocation(chat.Timezone))
		hhmm := chat.DailyTime
		if hhmm == "" {
			hhmm = s.defaultDailyHH
		}
		if now.Format("15:04") != hhmm {
			continue
		}

		today := now.Format("2006-01-02")
		if chat.LastDailySentOn == today {
			continue
		}

		processed++
		if err := s.sendUniqueQuestion(r.Context(), chat.ChatID, "Daily LeetCode challenge:"); err != nil {
			s.logger.Printf("daily send failed for chat %d: %v", chat.ChatID, err)
			continue
		}
		if err := s.store.MarkDailySent(r.Context(), chat.ChatID, today); err != nil {
			s.logger.Printf("mark daily sent failed for chat %d: %v", chat.ChatID, err)
		}
		sent++
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(fmt.Sprintf("processed=%d sent=%d", processed, sent)))
}

func (s *Service) Warmup(ctx context.Context) {
	_, err := s.questions.AllQuestions(ctx)
	if err != nil {
		s.logger.Printf("warmup questions failed: %v", err)
	}
}

func (s *Service) handleMessage(ctx context.Context, msg telegram.Message) error {
	if !s.isAllowedUsername(msg.From.Username) {
		s.logger.Printf("blocked message from unauthorized username=%q chat=%d", msg.From.Username, msg.Chat.ID)
		return s.tgClient.SendMessage(ctx, msg.Chat.ID, "You are not allowed to use this bot.")
	}

	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return nil
	}

	if strings.HasPrefix(text, "/") {
		return s.handleCommand(ctx, msg.Chat.ID, text)
	}

	return s.handleFreeTextAnswer(ctx, msg.Chat.ID, text)
}

func (s *Service) handleFreeTextAnswer(ctx context.Context, chatID int64, answer string) error {
	settings, err := s.store.GetChatSettings(ctx, chatID)
	if err != nil {
		return err
	}
	if settings.CurrentQuestion == nil {
		return s.tgClient.SendMessage(ctx, chatID, "No active question. Use /lc first.")
	}
	if learnerContext, isHint := parseHintRequest(answer); isHint {
		return s.sendHint(ctx, chatID, *settings.CurrentQuestion, learnerContext)
	}

	review, aiUsed := s.reviewAnswer(ctx, *settings.CurrentQuestion, answer)
	source := "Heuristic"
	if aiUsed {
		source = "AI"
	}

	feedback := strings.TrimSpace(review.Feedback)
	if feedback == "" {
		feedback = "No feedback provided."
	}
	guidance := strings.TrimSpace(review.Guidance)
	if guidance == "" {
		guidance = fallbackGuidance(*settings.CurrentQuestion, answer)
	}

	score := clampScore(review.Score)
	status := "Not saved yet. Improve and resubmit, or use /done."
	if score >= correctAnswerScoreThreshold {
		if err := s.persistCompletedQuestion(ctx, chatID, *settings.CurrentQuestion); err != nil {
			return err
		}
		status = "Correct. Saved to history."
	}

	reply := formatEvaluationMessage(*settings.CurrentQuestion, score, source, feedback, guidance, status)
	if err := s.tgClient.SendRichMessage(ctx, chatID, reply); err != nil {
		return err
	}

	return nil
}

func (s *Service) sendUniqueQuestion(ctx context.Context, chatID int64, intro string, transientExclude ...string) error {
	seen, err := s.store.SeenQuestionSet(ctx, chatID)
	if err != nil {
		return err
	}

	excludeSet := toSlugSet(transientExclude)
	effectiveSeen := mergeSlugSets(seen, excludeSet)

	q, err := s.questions.RandomQuestion(ctx, effectiveSeen)
	note := ""
	if errors.Is(err, ErrNoUnseenQuestions) {
		if err := s.store.ResetServedQuestions(ctx, chatID); err != nil {
			return err
		}
		q, err = s.questions.RandomQuestion(ctx, excludeSet)
		if errors.Is(err, ErrNoUnseenQuestions) && len(excludeSet) > 0 {
			q, err = s.questions.RandomQuestion(ctx, map[string]struct{}{})
		}
		note = "Question history exhausted and reset to allow new picks.\n\n"
	}
	if err != nil {
		return err
	}

	if err := s.store.SetCurrentQuestion(ctx, chatID, q); err != nil {
		return err
	}

	prompt, err := s.questions.QuestionPrompt(ctx, q.Slug)
	if err != nil {
		s.logger.Printf("question prompt lookup failed for slug=%s: %v", q.Slug, err)
		prompt = ""
	}

	msg := formatQuestionMessage(intro, note, q, prompt)
	return s.tgClient.SendRichMessage(ctx, chatID, msg)
}

func (s *Service) persistCompletedQuestion(ctx context.Context, chatID int64, q Question) error {
	if err := s.store.AddServedQuestion(ctx, chatID, q); err != nil {
		return err
	}
	if err := s.store.MarkQuestionAnswered(ctx, chatID, q); err != nil {
		return err
	}
	if err := s.store.ClearCurrentQuestion(ctx, chatID); err != nil {
		return err
	}
	return nil
}

func (s *Service) reviewAnswer(ctx context.Context, q Question, answer string) (AnswerReview, bool) {
	if s.coach != nil {
		review, err := s.coach.ReviewAnswer(ctx, q, answer)
		if err == nil {
			if review.Score == 0 {
				review.Score = 5
			}
			review.Score = clampScore(review.Score)
			return review, true
		}
		s.logger.Printf("AI review failed, falling back to heuristic grading: %v", err)
	}

	score, feedback := gradeAnswer(answer, q.Difficulty)
	return AnswerReview{
		Score:    score,
		Feedback: feedback,
		Guidance: fallbackGuidance(q, answer),
	}, false
}

func fallbackGuidance(q Question, learnerContext string) string {
	base := strings.Builder{}
	base.WriteString("## Plan\n")
	base.WriteString("1. Restate input/output and constraints.\n")
	base.WriteString("2. Pick the core pattern (map, stack, two-pointers, DP, graph).\n")
	base.WriteString("3. Validate with one normal case and one edge case.\n")
	base.WriteString("4. State time/space complexity.\n")

	if strings.EqualFold(q.Difficulty, "Hard") {
		base.WriteString("\n## Hard Focus\n- Compare two strategies and justify why yours is optimal.")
	}
	if strings.TrimSpace(learnerContext) != "" {
		base.WriteString("\n## Focus\n- Tighten your state definition and loop invariant.")
	}
	return base.String()
}

func clampScore(score int) int {
	if score < 1 {
		return 1
	}
	if score > 10 {
		return 10
	}
	return score
}

func buildAllowedUserSet(usernames []string) map[string]struct{} {
	if len(usernames) == 0 {
		return nil
	}

	out := make(map[string]struct{}, len(usernames))
	for _, username := range usernames {
		normalized := normalizeTelegramUsername(username)
		if normalized == "" {
			continue
		}
		out[normalized] = struct{}{}
	}
	return out
}

func (s *Service) isAllowedUsername(username string) bool {
	if len(s.allowedUsers) == 0 {
		return true
	}
	normalized := normalizeTelegramUsername(username)
	if normalized == "" {
		return false
	}
	_, ok := s.allowedUsers[normalized]
	return ok
}

func normalizeTelegramUsername(raw string) string {
	out := strings.TrimSpace(strings.ToLower(raw))
	return strings.TrimPrefix(out, "@")
}

func toSlugSet(slugs []string) map[string]struct{} {
	out := make(map[string]struct{}, len(slugs))
	for _, slug := range slugs {
		if slug = strings.TrimSpace(slug); slug != "" {
			out[slug] = struct{}{}
		}
	}
	return out
}

func mergeSlugSets(base map[string]struct{}, extra map[string]struct{}) map[string]struct{} {
	merged := make(map[string]struct{}, len(base)+len(extra))
	for slug := range base {
		merged[slug] = struct{}{}
	}
	for slug := range extra {
		merged[slug] = struct{}{}
	}
	return merged
}

func (s *Service) resolveLocation(name string) *time.Location {
	if name == "" {
		return s.defaultLoc
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return s.defaultLoc
	}
	return loc
}
