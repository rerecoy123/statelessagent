package memory

import (
	"testing"
	"time"
)

func TestComputeRecencyScore(t *testing.T) {
	now := float64(time.Now().Unix())

	// Recent note should have high score
	score := ComputeRecencyScore(now, "note")
	if score < 0.95 {
		t.Errorf("recent note score should be ~1.0, got %.3f", score)
	}

	// 60-day-old note (half-life for "note" type) should be ~0.5
	sixtyDaysAgo := now - 60*86400
	score = ComputeRecencyScore(sixtyDaysAgo, "note")
	if score < 0.4 || score > 0.6 {
		t.Errorf("60-day-old note should be ~0.5, got %.3f", score)
	}

	// Decisions never decay
	score = ComputeRecencyScore(now-365*86400, "decision")
	if score != 1.0 {
		t.Errorf("decision should never decay, got %.3f", score)
	}

	// Hubs never decay
	score = ComputeRecencyScore(now-365*86400, "hub")
	if score != 1.0 {
		t.Errorf("hub should never decay, got %.3f", score)
	}
}

func TestComputeConfidence(t *testing.T) {
	now := float64(time.Now().Unix())

	// Decision should have high confidence
	score := ComputeConfidence("decision", now, 0, false)
	if score < 0.7 {
		t.Errorf("decision confidence should be high, got %.3f", score)
	}

	// Note should have moderate confidence
	score = ComputeConfidence("note", now, 0, false)
	if score < 0.4 || score > 0.7 {
		t.Errorf("note confidence should be moderate, got %.3f", score)
	}

	// Access boost should increase confidence
	scoreNoAccess := ComputeConfidence("note", now, 0, false)
	scoreWithAccess := ComputeConfidence("note", now, 100, false)
	if scoreWithAccess <= scoreNoAccess {
		t.Errorf("access boost should increase confidence: %f vs %f", scoreNoAccess, scoreWithAccess)
	}

	// Review-by boost
	scoreNoReview := ComputeConfidence("note", now, 0, false)
	scoreWithReview := ComputeConfidence("note", now, 0, true)
	if scoreWithReview <= scoreNoReview {
		t.Errorf("review boost should increase confidence: %f vs %f", scoreNoReview, scoreWithReview)
	}
}

func TestCompositeScore(t *testing.T) {
	now := float64(time.Now().Unix())

	score := CompositeScore(1.0, now, 0.8, "note", 0.5, 0.4, 0.1)
	if score < 0.8 || score > 1.0 {
		t.Errorf("composite score for high semantic + recent should be high, got %.3f", score)
	}

	// Low semantic score should reduce composite
	lowScore := CompositeScore(0.1, now, 0.5, "note", 0.5, 0.4, 0.1)
	if lowScore >= score {
		t.Errorf("low semantic score should reduce composite: %f vs %f", lowScore, score)
	}
}

func TestHasRecencyIntent(t *testing.T) {
	positives := []string{
		"what did I work on recently",
		"show me my latest notes",
		"what changed this week",
		"what was I working on yesterday",
		"notes I updated today",
		"what happened last session",
	}
	for _, q := range positives {
		if !HasRecencyIntent(q) {
			t.Errorf("expected recency intent for %q", q)
		}
	}

	negatives := []string{
		"how does the confidence scoring work",
		"explain the decision extraction pipeline",
		"what is the architecture of SAME",
		"tell me about docker containers",
	}
	for _, q := range negatives {
		if HasRecencyIntent(q) {
			t.Errorf("unexpected recency intent for %q", q)
		}
	}
}

func TestInferContentType(t *testing.T) {
	tests := []struct {
		path         string
		explicitType string
		tags         []string
		want         string
	}{
		{"07_Journal/Sessions/handoff.md", "", nil, "handoff"},
		{"decisions_and_conclusions.md", "", nil, "decision"},
		{"_PRIVATE/Research/foo.md", "", nil, "research"},
		{"01_Projects/bar.md", "", nil, "project"},
		{"03_Resources/hub.md", "", nil, "hub"},
		{"random.md", "", nil, "note"},
		{"random.md", "decision", nil, "decision"},
		{"random.md", "", []string{"research"}, "research"},
	}

	for _, tt := range tests {
		got := InferContentType(tt.path, tt.explicitType, tt.tags)
		if got != tt.want {
			t.Errorf("InferContentType(%q, %q, %v) = %q, want %q",
				tt.path, tt.explicitType, tt.tags, got, tt.want)
		}
	}
}
