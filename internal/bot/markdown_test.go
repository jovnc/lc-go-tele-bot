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

	if !strings.Contains(msg, "*Problem Statement*") {
		t.Fatalf("expected question markdown to include Problem Statement heading: %s", msg)
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

	for _, marker := range []string{"*Feedback*", "*Guided Next Steps*", "*Status*", "Correct\\. Saved"} {
		if !strings.Contains(msg, marker) {
			t.Fatalf("expected evaluation markdown to include %q: %s", marker, msg)
		}
	}
}
