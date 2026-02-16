package commands

import "testing"

func TestNormalizeSlug(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain slug", input: "two-sum", want: "two-sum"},
		{name: "leetcode url", input: "https://leetcode.com/problems/two-sum/", want: "two-sum"},
		{name: "copy from answered line", input: "slug: two-sum | attempts: 2 | last: 2026-01-01", want: "two-sum"},
		{name: "copy with punctuation", input: "`two-sum`,", want: "two-sum"},
		{name: "empty", input: "   ", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeSlug(tc.input); got != tc.want {
				t.Fatalf("normalizeSlug(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
