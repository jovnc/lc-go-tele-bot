package bot

import (
	"strings"
	"testing"
)

func TestFormatQuestionMessageIncludesSectionsAndEscapes(t *testing.T) {
	msg := formatQuestionMessage(
		"Here is your random LeetCode question:",
		"",
		Question{Title: "Two Sum", Difficulty: "Easy"},
		"Given nums[i] and target (int), find answer_1.",
	)

	for _, marker := range []string{"<b>🧩 Here is your random LeetCode question:</b>", "<b>Problem</b>", "<b>Next</b>"} {
		if !strings.Contains(msg, marker) {
			t.Fatalf("expected question message to include %q: %s", marker, msg)
		}
	}
	if !strings.Contains(msg, "nums[i]") {
		t.Fatalf("expected bracket content to remain readable: %s", msg)
	}
	if !strings.Contains(msg, "answer_1") {
		t.Fatalf("expected underscore content to remain readable: %s", msg)
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

	for _, marker := range []string{"<b>🧠 Evaluation</b>", "<b>Feedback</b>", "<b>Next Steps</b>", "<b>Status</b>", "Correct. Saved"} {
		if !strings.Contains(msg, marker) {
			t.Fatalf("expected evaluation message to include %q: %s", marker, msg)
		}
	}
}

func TestRenderStructuredTextForTelegramHeadingsListsAndCodeBlocks(t *testing.T) {
	input := "# Main\n## Sub\n- item\n1. first\n```go\nfmt.Println(`ok`)\n```"
	got := renderStructuredTextForTelegram(input)

	checks := []string{
		"<b>Main</b>",
		"<b>▸ Sub</b>",
		"• item",
		"1. first",
		"<pre><code class=\"language-go\">",
		"fmt.Println(`ok`)",
	}
	for _, c := range checks {
		if !strings.Contains(got, c) {
			t.Fatalf("expected rendered rich text to include %q: %s", c, got)
		}
	}
}
