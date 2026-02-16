package guard

import (
	"testing"
)

func TestBuiltinPatterns_GitHubToken(t *testing.T) {
	patterns := builtinPatterns()
	var ghPattern Pattern
	for _, p := range patterns {
		if p.Name == "github_token" {
			ghPattern = p
			break
		}
	}
	if ghPattern.Regex == nil {
		t.Fatal("github_token pattern not found in builtinPatterns")
	}

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"ghp personal access token", "token: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijkl", true},
		{"ghs server token", "ghs_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijkl", true},
		{"too short", "ghp_ABC", false},
		{"not a token", "ghp is not a token", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ghPattern.Regex.MatchString(tt.input)
			if got != tt.want {
				t.Errorf("github_token match(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuiltinPatterns_SlackToken(t *testing.T) {
	patterns := builtinPatterns()
	var slackPattern Pattern
	for _, p := range patterns {
		if p.Name == "slack_token" {
			slackPattern = p
			break
		}
	}
	if slackPattern.Regex == nil {
		t.Fatal("slack_token pattern not found in builtinPatterns")
	}

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"xoxb bot token", "xoxb-1234567890-abcdefghij", true},
		{"xoxp user token", "xoxp-1234567890-abcdefghij", true},
		{"xoxs session token", "xoxs-1234567890-abcdefghij", true},
		{"too short", "xoxb-abc", false},
		{"not a token", "xoxb is not a token", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slackPattern.Regex.MatchString(tt.input)
			if got != tt.want {
				t.Errorf("slack_token match(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuiltinPatterns_ExistingPatterns(t *testing.T) {
	patterns := builtinPatterns()

	expectedNames := []string{
		"email", "us_phone", "ssn", "local_path_unix", "local_path_windows",
		"api_key_assignment", "sk_key", "aws_key", "private_key_header",
		"github_token", "slack_token",
	}

	nameSet := make(map[string]bool, len(patterns))
	for _, p := range patterns {
		nameSet[p.Name] = true
	}

	for _, name := range expectedNames {
		if !nameSet[name] {
			t.Errorf("expected pattern %q not found in builtinPatterns", name)
		}
	}
}

func TestRedact_Security(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"short string fully masked", "abc", "***"},
		{"6 chars fully masked", "abcdef", "******"},
		{"7+ chars partial", "abcdefg", "abc*efg"},
		{"email redacted", "user@example.com", "use**********com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redact(tt.input)
			if got != tt.want {
				t.Errorf("redact(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsExcluded_TestFiles(t *testing.T) {
	if !isExcluded("anything", "internal/guard/guard_test.go") {
		t.Error("expected test files to be excluded")
	}
	if !isExcluded("anything", "internal/tests/integration/foo.go") {
		t.Error("expected /tests/ directory to be excluded")
	}
	if isExcluded("real content here", "internal/guard/guard.go") {
		t.Error("non-test file should not be excluded without indicators")
	}
}

func TestIsExcluded_Indicators(t *testing.T) {
	indicators := []string{
		"this is an example",
		"test@example.com",
		"placeholder value",
		"fake data here",
		"dummy entry",
		"mock service",
		"user@example.com",
		"noreply@company.com",
		"xxx-xx-xxxx",
		"000-00-0000",
	}

	for _, line := range indicators {
		if !isExcluded(line, "real_file.go") {
			t.Errorf("expected line %q to be excluded", line)
		}
	}
}
