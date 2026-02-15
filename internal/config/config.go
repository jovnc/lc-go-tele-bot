package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port             string
	TelegramBotToken string
	WebhookSecret    string
	CronSecret       string
	FirestoreProject string
	AllowedUsernames []string

	DefaultDailyTime string
	DefaultTimezone  string
	AutoSetWebhook   bool
	BotBaseURL       string
	QuestionCacheSec int

	AIEnabled    bool
	OpenAIAPIKey string
	OpenAIModel  string
	AITimeoutSec int
}

func Load() (Config, error) {
	autoSetWebhook, err := parseBoolEnv("AUTO_SET_WEBHOOK", false)
	if err != nil {
		return Config{}, err
	}

	cacheSec, err := parseIntEnv("QUESTION_CACHE_SEC", 3600)
	if err != nil {
		return Config{}, err
	}
	aiEnabled, err := parseBoolEnv("AI_ENABLED", true)
	if err != nil {
		return Config{}, err
	}
	aiTimeoutSec, err := parseIntEnv("AI_TIMEOUT_SEC", 25)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Port:             getEnv("PORT", "8080"),
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		WebhookSecret:    os.Getenv("WEBHOOK_SECRET"),
		CronSecret:       os.Getenv("CRON_SECRET"),
		FirestoreProject: os.Getenv("FIRESTORE_PROJECT_ID"),
		AllowedUsernames: parseAllowedUsernamesEnv("ALLOWED_TELEGRAM_USERNAMES"),
		DefaultDailyTime: getEnv("DAILY_DEFAULT_TIME", "20:00"),
		DefaultTimezone:  getEnv("DAILY_TIMEZONE", "Asia/Singapore"),
		AutoSetWebhook:   autoSetWebhook,
		BotBaseURL:       getEnv("BOT_BASE_URL", ""),
		QuestionCacheSec: cacheSec,
		AIEnabled:        aiEnabled,
		OpenAIAPIKey:     strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		OpenAIModel:      getEnv("OPENAI_MODEL", "gpt-4o-mini"),
		AITimeoutSec:     aiTimeoutSec,
	}

	if cfg.TelegramBotToken == "" {
		return Config{}, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}
	if cfg.WebhookSecret == "" {
		return Config{}, fmt.Errorf("WEBHOOK_SECRET is required")
	}
	if cfg.CronSecret == "" {
		return Config{}, fmt.Errorf("CRON_SECRET is required")
	}
	if cfg.FirestoreProject == "" {
		return Config{}, fmt.Errorf("FIRESTORE_PROJECT_ID is required")
	}
	if _, err := time.Parse("15:04", cfg.DefaultDailyTime); err != nil {
		return Config{}, fmt.Errorf("invalid DAILY_DEFAULT_TIME %q: expected HH:MM", cfg.DefaultDailyTime)
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return strings.TrimSpace(v)
	}
	return fallback
}

func parseBoolEnv(key string, fallback bool) (bool, error) {
	raw, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}
	v, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return false, fmt.Errorf("invalid %s: %q", key, raw)
	}
	return v, nil
}

func parseIntEnv(key string, fallback int) (int, error) {
	raw, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || v <= 0 {
		return 0, fmt.Errorf("invalid %s: %q", key, raw)
	}
	return v, nil
}

func parseAllowedUsernamesEnv(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}

	dedup := make(map[string]struct{})
	out := make([]string, 0)
	for _, token := range strings.Split(raw, ",") {
		username := normalizeUsername(token)
		if username == "" {
			continue
		}
		if _, exists := dedup[username]; exists {
			continue
		}
		dedup[username] = struct{}{}
		out = append(out, username)
	}

	return out
}

func normalizeUsername(raw string) string {
	username := strings.TrimSpace(strings.ToLower(raw))
	username = strings.TrimPrefix(username, "@")
	return username
}
