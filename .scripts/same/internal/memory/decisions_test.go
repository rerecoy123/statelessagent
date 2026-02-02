package memory

import (
	"encoding/json"
	"os"
	"testing"
)

type testCase struct {
	Text                 string   `json:"text"`
	ExpectedDecisions    []string `json:"expected_decisions"`
	ExpectedNonDecisions []string `json:"expected_non_decisions"`
}

type groundTruth struct {
	TestCases []testCase `json:"test_cases"`
}

func TestDecisionExtraction(t *testing.T) {
	// Try to load ground truth from eval data
	gtPath := "../../.scripts/eval/data/decision_ground_truth.json"
	// Also try relative to project root
	if _, err := os.Stat(gtPath); err != nil {
		gtPath = "../../../eval/data/decision_ground_truth.json"
	}

	data, err := os.ReadFile(gtPath)
	if err != nil {
		t.Log("Ground truth file not found, using inline test cases")
		testInlineDecisions(t)
		return
	}

	var gt groundTruth
	if err := json.Unmarshal(data, &gt); err != nil {
		t.Fatalf("parse ground truth: %v", err)
	}

	for i, tc := range gt.TestCases {
		decisions := ExtractDecisions(tc.Text, true)

		// Check expected decisions are found
		for _, expected := range tc.ExpectedDecisions {
			found := false
			for _, d := range decisions {
				if containsNormalized(d.Text, expected) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("case %d: expected to find decision containing %q, got %d decisions: %v",
					i, expected, len(decisions), decisionTexts(decisions))
			}
		}

		// Check non-decisions are NOT found
		for _, nonExpected := range tc.ExpectedNonDecisions {
			for _, d := range decisions {
				if containsNormalized(d.Text, nonExpected) {
					t.Errorf("case %d: should NOT have matched %q but got decision: %q",
						i, nonExpected, d.Text)
				}
			}
		}
	}
}

func testInlineDecisions(t *testing.T) {
	tests := []struct {
		text     string
		wantAny  bool
		wantText string
	}{
		{
			text:     "**Decision:** Use nomic-embed-text for all vault embeddings.",
			wantAny:  true,
			wantText: "Decision",
		},
		{
			text:     "We decided to use LanceDB over ChromaDB because it has better support.",
			wantAny:  true,
			wantText: "decided to",
		},
		{
			text:     "Let's go with the hook-based architecture for session handoffs.",
			wantAny:  true,
			wantText: "go with",
		},
		{
			text:    "If we decide to add a graph database later, we should consider Neo4j.",
			wantAny: false,
		},
		{
			text:    "We haven't decided on the final deployment model yet.",
			wantAny: false,
		},
		{
			text:    "We need to decide about whether to use structured frontmatter.",
			wantAny: false,
		},
	}

	for i, tt := range tests {
		decisions := ExtractDecisions(tt.text, true)
		if tt.wantAny && len(decisions) == 0 {
			t.Errorf("case %d: expected decisions, got none for: %s", i, tt.text)
		}
		if !tt.wantAny && len(decisions) > 0 {
			t.Errorf("case %d: expected no decisions, got %d for: %s", i, len(decisions), tt.text)
		}
	}
}

func containsNormalized(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 &&
		(haystack == needle ||
			len(haystack) >= len(needle) && (haystack[:len(needle)] == needle ||
				findSubstring(haystack, needle)))
}

func findSubstring(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func decisionTexts(decisions []Decision) []string {
	var texts []string
	for _, d := range decisions {
		texts = append(texts, d.Text)
	}
	return texts
}
