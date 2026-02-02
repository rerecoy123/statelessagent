package memory

import "testing"

func TestNormalizeForMatching(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello_world", "hello world"},
		{"2026-01-15 my-note", "2026 01 15 my note"},
		{"em\u2014dash", "em dash"},
		{"  multiple   spaces  ", "multiple spaces"},
	}

	for _, tt := range tests {
		got := NormalizeForMatching(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeForMatching(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractTitleWords(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"2026-01-15-my-cool-note", "my cool note"},
		{"2026_01_15 some_note", "some note"},
		{"no-date-prefix", "no date prefix"},
	}

	for _, tt := range tests {
		got := ExtractTitleWords(tt.input)
		if got != tt.want {
			t.Errorf("ExtractTitleWords(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEstimateTokens(t *testing.T) {
	text := "This is a test string with about forty characters."
	tokens := EstimateTokens(text)
	// ~50 chars / 4 = ~12 tokens
	if tokens < 10 || tokens > 15 {
		t.Errorf("EstimateTokens: expected ~12, got %d", tokens)
	}
}
