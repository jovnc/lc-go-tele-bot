package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"telegram-leetcode-bot/internal/bot"
)

const defaultOpenAIBaseURL = "https://api.openai.com/v1"

type OpenAICoach struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

func NewOpenAICoach(apiKey, model string, timeout time.Duration) (*OpenAICoach, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required")
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = "gpt-4o-mini"
	}
	if timeout <= 0 {
		timeout = 25 * time.Second
	}

	return &OpenAICoach{
		apiKey:  apiKey,
		model:   model,
		baseURL: defaultOpenAIBaseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (c *OpenAICoach) ReviewAnswer(ctx context.Context, question bot.Question, answer string) (bot.AnswerReview, error) {
	system := "You are a senior coding interview coach. Be precise and actionable."
	user := fmt.Sprintf(
		"Question: %s (%s)\nLink: %s\n\nCandidate answer:\n%s\n\nReturn valid JSON only with keys: score (integer 1-10), feedback (string), guidance (string). Guidance must be a concrete step-by-step plan to improve and solve correctly.",
		question.Title,
		question.Difficulty,
		question.URL,
		answer,
	)

	content, err := c.chatCompletion(ctx, system, user, true)
	if err != nil {
		return bot.AnswerReview{}, err
	}

	var parsed struct {
		Score    int    `json:"score"`
		Feedback string `json:"feedback"`
		Guidance string `json:"guidance"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return bot.AnswerReview{}, fmt.Errorf("parse AI review JSON: %w", err)
	}

	return bot.AnswerReview{
		Score:    parsed.Score,
		Feedback: strings.TrimSpace(parsed.Feedback),
		Guidance: strings.TrimSpace(parsed.Guidance),
	}, nil
}

func (c *OpenAICoach) chatCompletion(ctx context.Context, system, user string, forceJSON bool) (string, error) {
	type chatMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	payload := map[string]any{
		"model": c.model,
		"messages": []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		"temperature": 0.2,
	}
	if forceJSON {
		payload["response_format"] = map[string]any{"type": "json_object"}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal chat payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build AI request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("AI request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read AI response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("AI status %d: %s", resp.StatusCode, string(raw))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("decode AI response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("AI response has no choices")
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("AI response content is empty")
	}

	return content, nil
}
