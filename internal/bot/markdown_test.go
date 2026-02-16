package bot

import (
	"strings"
	"testing"
)

func TestFormatQuestionMessageIncludesSectionsAndEscapes(t *testing.T) {
	msg := formatQuestionMessage(
		"Here is your random LeetCode question:",
		"",
		Question{Title: "Two Sum", Difficulty: "Easy", URL: "https://leetcode.com/problems/two-sum/"},
		"Given nums[i] and target (int), find answer_1.",
	)

	for _, marker := range []string{"*[Two Sum](https://leetcode.com/problems/two-sum/)* \\(Easy\\)", "__*Problem*__", "__*Next*__"} {
		if !strings.Contains(msg, marker) {
			t.Fatalf("expected question message to include %q: %s", marker, msg)
		}
	}
	if strings.Contains(msg, "Here is your random LeetCode question:") {
		t.Fatalf("expected intro banner to be omitted: %s", msg)
	}
	if !strings.Contains(msg, "nums\\[i\\]") {
		t.Fatalf("expected bracket content to be escaped for MarkdownV2: %s", msg)
	}
	if !strings.Contains(msg, "answer\\_1") {
		t.Fatalf("expected underscore content to be escaped for MarkdownV2: %s", msg)
	}
}

func TestFormatEvaluationMessageIncludesStatusSection(t *testing.T) {
	msg := formatEvaluationMessage(
		Question{Title: "Two Sum", Difficulty: "Easy"},
		8,
		"AI",
		"Good approach.",
		"State complexity.",
		"Correct. Saved to history.",
	)

	for _, marker := range []string{"*🧠 Evaluation*", "__*Feedback*__", "__*Next Steps*__", "__*Status*__", "Correct\\. Saved"} {
		if !strings.Contains(msg, marker) {
			t.Fatalf("expected evaluation message to include %q: %s", marker, msg)
		}
	}
}

func TestRenderStructuredTextForTelegramHeadingsListsAndCodeBlocks(t *testing.T) {
	input := "# Main\n## Sub\n- item\n1. first\n```go\nfmt.Println(`ok`)\n```"
	got := renderStructuredTextForTelegram(input)

	checks := []string{
		"__*Main*__",
		"__*Sub*__",
		"• item",
		"1\\. first",
		"```go",
		"fmt.Println(\\`ok\\`)",
	}
	for _, c := range checks {
		if !strings.Contains(got, c) {
			t.Fatalf("expected rendered rich text to include %q: %s", c, got)
		}
	}
}
