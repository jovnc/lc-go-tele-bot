package leetcode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

const problemsEndpoint = "https://leetcode.com/api/problems/all/"
const graphqlEndpoint = "https://leetcode.com/graphql"

const questionPromptQuery = `
query questionPrompt($titleSlug: String!) {
  question(titleSlug: $titleSlug) {
    content
  }
}`

var ErrNoUnseenQuestions = errors.New("no unseen questions available")

type Client struct {
	httpClient *http.Client
	cacheTTL   time.Duration

	mu       sync.RWMutex
	cachedAt time.Time
	cached   []Question
}

func NewClient(cacheTTL time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 20 * time.Second},
		cacheTTL:   cacheTTL,
	}
}

type Question struct {
	Slug       string
	Title      string
	Difficulty string
	URL        string
}

type problemResponse struct {
	StatStatusPairs []struct {
		PaidOnly bool `json:"paid_only"`
		Stat     struct {
			Title string `json:"question__title"`
			Slug  string `json:"question__title_slug"`
		} `json:"stat"`
		Difficulty struct {
			Level int `json:"level"`
		} `json:"difficulty"`
	} `json:"stat_status_pairs"`
}

func (c *Client) RandomQuestion(ctx context.Context, seen map[string]struct{}) (Question, error) {
	all, err := c.AllQuestions(ctx)
	if err != nil {
		return Question{}, err
	}

	candidates := make([]Question, 0, len(all))
	for _, q := range all {
		if _, exists := seen[q.Slug]; exists {
			continue
		}
		candidates = append(candidates, q)
	}

	if len(candidates) == 0 {
		return Question{}, ErrNoUnseenQuestions
	}

	return candidates[rand.Intn(len(candidates))], nil
}

func (c *Client) AllQuestions(ctx context.Context) ([]Question, error) {
	c.mu.RLock()
	if len(c.cached) > 0 && time.Since(c.cachedAt) < c.cacheTTL {
		out := slices.Clone(c.cached)
		c.mu.RUnlock()
		return out, nil
	}
	c.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, problemsEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create leetcode request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch leetcode questions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("leetcode status %d", resp.StatusCode)
	}

	var parsed problemResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode leetcode response: %w", err)
	}

	questions := make([]Question, 0, len(parsed.StatStatusPairs))
	for _, p := range parsed.StatStatusPairs {
		if p.PaidOnly {
			continue
		}
		slug := strings.TrimSpace(p.Stat.Slug)
		title := strings.TrimSpace(p.Stat.Title)
		if slug == "" || title == "" {
			continue
		}
		questions = append(questions, Question{
			Slug:       slug,
			Title:      title,
			Difficulty: difficultyLabel(p.Difficulty.Level),
			URL:        "https://leetcode.com/problems/" + slug + "/",
		})
	}

	if len(questions) == 0 {
		return nil, fmt.Errorf("leetcode returned no free questions")
	}

	c.mu.Lock()
	c.cached = questions
	c.cachedAt = time.Now()
	c.mu.Unlock()

	return slices.Clone(questions), nil
}

func (c *Client) QuestionPrompt(ctx context.Context, slug string) (string, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return "", fmt.Errorf("slug is empty")
	}

	payload := map[string]any{
		"operationName": "questionPrompt",
		"query":         questionPromptQuery,
		"variables": map[string]string{
			"titleSlug": slug,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal graphql payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create graphql request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", "https://leetcode.com/problems/"+slug+"/")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch question prompt: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("leetcode graphql status %d", resp.StatusCode)
	}

	var parsed struct {
		Data struct {
			Question struct {
				Content string `json:"content"`
			} `json:"question"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode question prompt response: %w", err)
	}
	if len(parsed.Errors) > 0 {
		return "", fmt.Errorf("leetcode graphql error: %s", parsed.Errors[0].Message)
	}

	prompt := htmlToText(parsed.Data.Question.Content)
	if prompt == "" {
		return "", fmt.Errorf("question prompt is empty")
	}
	return prompt, nil
}

func difficultyLabel(level int) string {
	switch level {
	case 1:
		return "Easy"
	case 2:
		return "Medium"
	case 3:
		return "Hard"
	default:
		return "Unknown"
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
