package bot

import (
	"strings"
	"testing"
)

func TestFormatQuestionMarkdownIncludesSectionsAndEscapes(t *testing.T) {
	msg := formatQuestionMarkdown(
		"Here is your random LeetCode question:",
		"",
		Question{Title: "Two Sum", Difficulty: "Easy"},
		"Given nums[i] and target (int), find answer_1.",
	)

	for _, marker := range []string{"*🧩 Here is your random LeetCode question:*", "*Problem Statement*", "*Next Step*"} {
		if !strings.Contains(msg, marker) {
			t.Fatalf("expected question markdown to include %q: %s", marker, msg)
		}
	}
	if !strings.Contains(msg, "nums\\[i\\]") {
		t.Fatalf("expected markdown escaping for brackets: %s", msg)
	}
	if !strings.Contains(msg, "answer\\_1") {
		t.Fatalf("expected markdown escaping for underscore: %s", msg)
	}
}

func TestFormatEvaluationMarkdownIncludesStatusSection(t *testing.T) {
	msg := formatEvaluationMarkdown(
		Question{Title: "Two Sum", Difficulty: "Easy"},
		8,
		"AI",
		"Good approach.",
		"State complexity.",
		"Correct. Saved to your seen/revision history.",
	)

	for _, marker := range []string{"*🧠 Evaluation*", "*Feedback*", "*Guided Next Steps*", "*Status*", "Correct\\. Saved"} {
		if !strings.Contains(msg, marker) {
			t.Fatalf("expected evaluation markdown to include %q: %s", marker, msg)
		}
	}
}

func TestRenderMarkdownForTelegramHeadingsListsAndCodeBlocks(t *testing.T) {
	input := "# Main\n## Sub\n- item\n1. first\n```go\nfmt.Println(`ok`)\n```"
	got := renderMarkdownForTelegram(input)

	checks := []string{
		"*Main*",
		"*▸ Sub*",
		"• item",
		"1\\. first",
		"```go",
		"fmt.Println(\\`ok\\`)",
	}
	for _, c := range checks {
		if !strings.Contains(got, c) {
			t.Fatalf("expected rendered markdown to include %q: %s", c, got)
		}
	}
}
