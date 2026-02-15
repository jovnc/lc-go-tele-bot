package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func normalizeCommand(token string) string {
	if idx := strings.Index(token, "@"); idx >= 0 {
		token = token[:idx]
	}
	return strings.ToLower(token)
}

func normalizeHHMM(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("time is empty")
	}

	for _, layout := range []string{"15:04", "15"} {
		t, err := time.Parse(layout, raw)
		if err == nil {
			return t.Format("15:04"), nil
		}
	}

	return "", fmt.Errorf("invalid time %q", raw)
}

func parsePositiveLimit(raw string, max int) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 || value > max {
		return 0, fmt.Errorf("invalid limit")
	}
	return value, nil
}

func normalizeSlug(raw string) string {
	v := strings.TrimSpace(strings.ToLower(raw))
	v = strings.Trim(v, "/")
	if v == "" {
		return ""
	}

	if idx := strings.Index(v, "leetcode.com/problems/"); idx >= 0 {
		v = v[idx+len("leetcode.com/problems/"):]
	}
	if idx := strings.IndexAny(v, "/?#"); idx >= 0 {
		v = v[:idx]
	}
	return strings.Trim(v, "/")
}

func tzLabel(tz string) string {
	if tz == "Asia/Singapore" {
		return "SGT"
	}
	return tz
}
