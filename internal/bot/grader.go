package bot

import (
	"fmt"
	"strings"
)

func gradeAnswer(answer string, difficulty string) (int, string) {
	text := strings.TrimSpace(answer)
	lower := strings.ToLower(text)
	wordCount := len(strings.Fields(text))

	score := 2
	feedback := make([]string, 0, 6)

	if wordCount >= 35 {
		score += 2
		feedback = append(feedback, "Coverage is good and detailed.")
	} else if wordCount >= 15 {
		score += 1
		feedback = append(feedback, "Approach is partially explained.")
	} else {
		feedback = append(feedback, "Explanation is short; add more detail on steps.")
	}

	if hasAny(lower, "for ", "while ", "if ", "return", "function", "def ", "map[", "[]", "stack", "queue") {
		score += 2
		feedback = append(feedback, "Pseudocode or algorithm structure is present.")
	} else {
		feedback = append(feedback, "Add pseudocode structure (loops, conditions, return).")
	}

	if hasAny(lower, "o(", "time complexity", "space complexity") {
		score += 2
		feedback = append(feedback, "Complexity considerations are included.")
	} else {
		feedback = append(feedback, "Include time and space complexity to strengthen your answer.")
	}

	if hasAny(lower, "empty", "null", "boundary", "single", "duplicate", "overflow", "underflow") {
		score += 1
		feedback = append(feedback, "Edge cases were considered.")
	} else {
		feedback = append(feedback, "Mention edge cases to improve robustness.")
	}

	if difficulty == "Hard" && !hasAny(lower, "o(", "time complexity") {
		score--
		feedback = append(feedback, "For Hard questions, complexity analysis is essential.")
	}

	if score < 1 {
		score = 1
	}
	if score > 10 {
		score = 10
	}

	verdict := "Needs work"
	if score >= 8 {
		verdict = "Strong"
	} else if score >= 5 {
		verdict = "Decent"
	}

	return score, fmt.Sprintf("Verdict: %s\n%s\nNote: This is heuristic grading from text/pseudocode, not code execution.", verdict, strings.Join(feedback, "\n"))
}

func hasAny(s string, patterns ...string) bool {
	for _, p := range patterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}
