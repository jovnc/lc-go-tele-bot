package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    fmt.Sprintf("https://api.telegram.org/bot%s", token),
	}
}

type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	payload := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}
	return c.postJSON(ctx, "/sendMessage", payload)
}

func (c *Client) SendRichMessage(ctx context.Context, chatID int64, text string) error {
	payload := map[string]any{
		"chat_id":                  chatID,
		"text":                     text,
		"parse_mode":               "HTML",
		"disable_web_page_preview": true,
	}
	return c.postJSON(ctx, "/sendMessage", payload)
}

func (c *Client) SetWebhook(ctx context.Context, webhookURL string) error {
	payload := map[string]any{
		"url": webhookURL,
	}
	return c.postJSON(ctx, "/setWebhook", payload)
}

func (c *Client) postJSON(ctx context.Context, path string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("telegram request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read telegram response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram status %d: %s", resp.StatusCode, string(data))
	}

	var out apiResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return fmt.Errorf("unmarshal telegram response: %w", err)
	}
	if !out.OK {
		return fmt.Errorf("telegram api error: %s", out.Description)
	}

	return nil
}

func BuildWebhookURL(baseURL, secret string) (string, error) {
	if baseURL == "" {
		return "", fmt.Errorf("base URL is empty")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid BOT_BASE_URL: %w", err)
	}
	parsed.Path = "/webhook/" + secret
	return parsed.String(), nil
}

// Telegram update models.
type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message"`
}

type Message struct {
	MessageID int    `json:"message_id"`
	Text      string `json:"text"`
	From      User   `json:"from"`
	Chat      Chat   `json:"chat"`
}

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}
