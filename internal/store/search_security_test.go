package store

import (
	"strings"
	"testing"
)

func TestSanitizeFTS5Term_RemovesOperators(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"star operator", "auth*", "auth"},
		{"caret operator", "^auth", "auth"},
		{"negation", "-excluded", "excluded"},
		{"quoted phrase", `"exact match"`, "exact match"},
		{"braces", "{column:value}", "column:value"},
		{"parens", "(group)", "group"},
		{"clean term", "authentication", "authentication"},
		{"mixed operators", `*"test"^-{foo}`, "testfoo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFTS5Term(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFTS5Term(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEscapeLIKE_EscapesWildcards(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"percent", "100%", `100\%`},
		{"underscore", "foo_bar", `foo\_bar`},
		{"backslash", `foo\bar`, `foo\\bar`},
		{"all wildcards", `50%_test\path`, `50\%\_test\\path`},
		{"clean term", "authentication", "authentication"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeLIKE(tt.input)
			if got != tt.want {
				t.Errorf("escapeLIKE(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractSearchTerms_FiltersStopWords(t *testing.T) {
	terms := ExtractSearchTerms("what is the authentication approach for our project")
	// "what", "is", "the", "for", "our", "project" are stop words
	// "authentication" and "approach" should survive
	found := make(map[string]bool)
	for _, term := range terms {
		found[term] = true
	}

	if !found["authentication"] {
		t.Error("expected 'authentication' to survive stop word filtering")
	}
	if !found["approach"] {
		t.Error("expected 'approach' to survive stop word filtering")
	}
	if found["what"] || found["the"] || found["for"] {
		t.Error("stop words should be filtered out")
	}
}

func TestExtractSearchTerms_ShortTerms(t *testing.T) {
	terms := ExtractSearchTerms("AI and ML for UX design")
	found := make(map[string]bool)
	for _, term := range terms {
		found[term] = true
	}

	// "ai", "ml", "ux" are meaningful 2-char terms
	if !found["ai"] {
		t.Error("expected 'ai' to be kept as meaningful short term")
	}
	if !found["ml"] {
		t.Error("expected 'ml' to be kept as meaningful short term")
	}
	if !found["ux"] {
		t.Error("expected 'ux' to be kept as meaningful short term")
	}
	// "and", "for" are stop words
	if found["and"] || found["for"] {
		t.Error("stop words should be filtered")
	}
}

func TestExtractSearchTerms_Deduplication(t *testing.T) {
	terms := ExtractSearchTerms("auth auth AUTH Auth")
	if len(terms) != 1 {
		t.Errorf("expected 1 deduplicated term, got %d: %v", len(terms), terms)
	}
}

func TestExtractSearchTerms_PunctuationStripping(t *testing.T) {
	terms := ExtractSearchTerms(`"authentication," approach. design!`)
	found := make(map[string]bool)
	for _, term := range terms {
		found[term] = true
	}
	if !found["authentication"] {
		t.Error("expected punctuation to be stripped from 'authentication'")
	}
	if !found["approach"] {
		t.Error("expected punctuation to be stripped from 'approach'")
	}
	if !found["design"] {
		t.Error("expected punctuation to be stripped from 'design'")
	}
}

func TestParseTags_Variants(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"valid json array", `["tag1","tag2","tag3"]`, 3},
		{"empty array", `[]`, 0},
		{"invalid json", `not json`, 0},
		{"empty string", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := ParseTags(tt.input)
			if len(tags) != tt.want {
				t.Errorf("ParseTags(%q) returned %d tags, want %d", tt.input, len(tags), tt.want)
			}
		})
	}
}

func TestEditDistance1(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want bool
	}{
		{"substitution", "auth", "autn", true},
		{"insertion", "auth", "auths", true},
		{"deletion", "auths", "auth", true},
		{"identical", "auth", "auth", false},
		{"too different", "auth", "search", false},
		{"empty vs one", "", "a", true},
		{"both empty", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := editDistance1(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("editDistance1(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestMaxFederatedVaults_Constant(t *testing.T) {
	if MaxFederatedVaults != 50 {
		t.Errorf("MaxFederatedVaults = %d, want 50", MaxFederatedVaults)
	}
}

func TestSplitTitleWords_SecurityCoverage(t *testing.T) {
	words := splitTitleWords("auth-approach for api_design (v2)")
	found := strings.Join(words, "|")
	for _, expected := range []string{"auth", "approach", "for", "api", "design", "v2"} {
		if !strings.Contains(found, expected) {
			t.Errorf("expected word %q in splitTitleWords result: %s", expected, found)
		}
	}
}
