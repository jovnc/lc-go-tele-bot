package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/firestore"

	"telegram-leetcode-bot/internal/adapters"
	"telegram-leetcode-bot/internal/ai"
	"telegram-leetcode-bot/internal/bot"
	"telegram-leetcode-bot/internal/config"
	"telegram-leetcode-bot/internal/leetcode"
	"telegram-leetcode-bot/internal/storage"
	"telegram-leetcode-bot/internal/telegram"
)

func Main() {
	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)
	if err := run(logger); err != nil {
		logger.Fatalf("%v", err)
	}
}

func run(logger *log.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	ctx := context.Background()
	fireClient, err := firestore.NewClient(ctx, cfg.FirestoreProject)
	if err != nil {
		return fmt.Errorf("create firestore client: %w", err)
	}
	defer func() {
		if err := fireClient.Close(); err != nil {
			logger.Printf("close firestore client: %v", err)
		}
	}()

	tgClient := telegram.NewClient(cfg.TelegramBotToken)
	lcClient := leetcode.NewClient(time.Duration(cfg.QuestionCacheSec) * time.Second)
	store := storage.NewStore(fireClient, cfg.DefaultDailyTime, cfg.DefaultTimezone)
	var coach bot.Coach
	if cfg.AIEnabled && cfg.OpenAIAPIKey != "" {
		c, err := ai.NewOpenAICoach(
			cfg.OpenAIAPIKey,
			cfg.OpenAIModel,
			time.Duration(cfg.AITimeoutSec)*time.Second,
		)
		if err != nil {
			logger.Printf("AI coach disabled due to initialization error: %v", err)
		} else {
			coach = c
			logger.Printf("AI coach enabled with model %s", cfg.OpenAIModel)
		}
	} else {
		logger.Printf("AI coach disabled (AI_ENABLED=%t, key_present=%t)", cfg.AIEnabled, cfg.OpenAIAPIKey != "")
	}

	service := bot.NewService(
		logger,
		tgClient,
		adapters.NewLeetCodeProvider(lcClient),
		coach,
		adapters.NewFirestoreStateStore(store),
		cfg.WebhookSecret,
		cfg.CronSecret,
		cfg.DefaultDailyTime,
		cfg.DefaultTimezone,
		cfg.AllowedUsernames,
	)

	if cfg.AutoSetWebhook {
		autoSetWebhook(ctx, logger, tgClient, cfg.BotBaseURL, cfg.WebhookSecret)
	}

	go service.Warmup(context.Background())

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/webhook/", service.WebhookHandler)
	mux.HandleFunc("/cron/daily", service.CronHandler)

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	shutdownDone := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigCh)

		<-sigCh

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Printf("shutdown error: %v", err)
		}
		close(shutdownDone)
	}()

	logger.Printf("bot server listening on %s", httpServer.Addr)
	err = httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	<-shutdownDone
	logger.Printf("shutdown complete")
	return nil
}

func autoSetWebhook(ctx context.Context, logger *log.Logger, client *telegram.Client, baseURL, secret string) {
	if baseURL == "" {
		logger.Printf("AUTO_SET_WEBHOOK=true but BOT_BASE_URL is empty; skipping")
		return
	}

	webhookURL, err := telegram.BuildWebhookURL(baseURL, secret)
	if err != nil {
		logger.Printf("build webhook URL failed: %v", err)
		return
	}

	setCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	if err := client.SetWebhook(setCtx, webhookURL); err != nil {
		logger.Printf("set webhook failed: %v", err)
		return
	}
	logger.Printf("webhook set to %s", webhookURL)
}
