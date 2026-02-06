package guard

import (
	"os"
	"path/filepath"
	"testing"
)

// --- Pattern tests ---

func TestRedact(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hi", "**"},
		{"abc", "***"},
		{"abcdefgh", "abc**fgh"},
		{"user@example.com", "use**********com"},
	}
	for _, tt := range tests {
		got := redact(tt.input)
		if got != tt.want {
			t.Errorf("redact(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBuiltinPatterns_Email(t *testing.T) {
	patterns := builtinPatterns()
	line := "contact me at john.doe@company.com for details"
	results := scanLine(line, "README.md", patterns)

	found := false
	for _, r := range results {
		if r.Pattern.Category == CatEmail {
			found = true
			if r.Match != "john.doe@company.com" {
				t.Errorf("expected email match 'john.doe@company.com', got %q", r.Match)
			}
		}
	}
	if !found {
		t.Error("expected email pattern to match")
	}
}

func TestBuiltinPatterns_Phone(t *testing.T) {
	patterns := builtinPatterns()
	tests := []string{
		"call (512) 915-5500 now",
		"phone: 555-867-5309",
		"reach us at +1 555.867.5309",
	}
	for _, line := range tests {
		results := scanLine(line, "file.go", patterns)
		found := false
		for _, r := range results {
			if r.Pattern.Category == CatPhone {
				found = true
			}
		}
		if !found {
			t.Errorf("expected phone pattern to match in %q", line)
		}
	}
}

func TestBuiltinPatterns_SSN(t *testing.T) {
	patterns := builtinPatterns()
	results := scanLine("SSN: 123-45-6789", "notes.md", patterns)

	found := false
	for _, r := range results {
		if r.Pattern.Category == CatSSN {
			found = true
		}
	}
	if !found {
		t.Error("expected SSN pattern to match")
	}
}

func TestBuiltinPatterns_LocalPath(t *testing.T) {
	patterns := builtinPatterns()
	results := scanLine("path = /Users/jdoe/Documents/vault", "Makefile", patterns)

	found := false
	for _, r := range results {
		if r.Pattern.Category == CatLocalPath {
			found = true
		}
	}
	if !found {
		t.Error("expected local path pattern to match")
	}
}

func TestBuiltinPatterns_APIKey(t *testing.T) {
	patterns := builtinPatterns()
	results := scanLine(`api_key = "sk-abcdefghijklmnopqrst1234"`, "config.go", patterns)

	if len(results) == 0 {
		t.Error("expected API key pattern to match")
	}
}

func TestBuiltinPatterns_AWSKey(t *testing.T) {
	patterns := builtinPatterns()
	results := scanLine("aws_key = AKIAIOSFODNN7EXAMPL0", "deploy.sh", patterns)

	found := false
	for _, r := range results {
		if r.Pattern.Category == CatAWSKey {
			found = true
		}
	}
	if !found {
		t.Error("expected AWS key pattern to match")
	}
}

func TestBuiltinPatterns_PrivateKey(t *testing.T) {
	patterns := builtinPatterns()
	results := scanLine("-----BEGIN PRIVATE KEY-----", "cert.pem", patterns)

	found := false
	for _, r := range results {
		if r.Pattern.Category == CatPrivateKey {
			found = true
		}
	}
	if !found {
		t.Error("expected private key pattern to match")
	}
}

func TestExclusion_TestFile(t *testing.T) {
	patterns := builtinPatterns()
	// Lines in test files should be excluded
	results := scanLine("email: user@real.com", "parser_test.go", patterns)
	if len(results) != 0 {
		t.Errorf("expected test file to be excluded, got %d matches", len(results))
	}
}

func TestExclusion_ExampleContent(t *testing.T) {
	patterns := builtinPatterns()
	results := scanLine("example: user@example.com", "README.md", patterns)
	if len(results) != 0 {
		t.Errorf("expected example content to be excluded, got %d matches", len(results))
	}
}

func TestExclusion_RegexDefinition(t *testing.T) {
	patterns := builtinPatterns()
	// Lines containing regexp. should be excluded (pattern definitions)
	results := scanLine(`regexp.MustCompile("[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}")`, "patterns.go", patterns)
	if len(results) != 0 {
		t.Errorf("expected regex definition to be excluded, got %d matches", len(results))
	}
}

// --- Allowlist tests ---

func TestIsPathAllowed(t *testing.T) {
	tests := []struct {
		path    string
		allowed bool
	}{
		{".scripts/build/Makefile", true},
		{".claude/settings.json", true},
		{".gitignore", true},
		{"package.json", true},
		{"SECURITY.md", true},
		{"README.md", true},
		{"projects/notes.md", false},
		{"Transcript 2 (Alice call).md", false},
		{"_PRIVATE/secret.md", false},
		{"random-file.txt", false},
	}

	for _, tt := range tests {
		got := IsPathAllowed(tt.path, nil)
		if got != tt.allowed {
			t.Errorf("IsPathAllowed(%q) = %v, want %v", tt.path, got, tt.allowed)
		}
	}
}

func TestIsPathAllowed_CustomPaths(t *testing.T) {
	custom := []string{"docs/", "Makefile"}
	if !IsPathAllowed("docs/guide.md", custom) {
		t.Error("expected docs/ to be allowed with custom paths")
	}
	if !IsPathAllowed("Makefile", custom) {
		t.Error("expected Makefile to be allowed with custom paths")
	}
}

// --- Blocklist tests ---

func TestBlocklist_Parse(t *testing.T) {
	dir := t.TempDir()
	blPath := filepath.Join(dir, ".blocklist")

	content := `[hard]
terms = ["Jane Doe", "blocked@test.com"]

[soft]
terms = ["rate sheet", "commission"]
`
	if err := os.WriteFile(blPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	terms, err := LoadBlocklist(blPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(terms) != 4 {
		t.Errorf("expected 4 terms, got %d", len(terms))
	}

	// Count tiers
	hard, soft := 0, 0
	for _, term := range terms {
		if term.Tier == TierHard {
			hard++
		} else {
			soft++
		}
	}
	if hard != 2 {
		t.Errorf("expected 2 hard terms, got %d", hard)
	}
	if soft != 2 {
		t.Errorf("expected 2 soft terms, got %d", soft)
	}
}

func TestBlocklist_CaseInsensitive(t *testing.T) {
	ct, err := compileTerm("Jane Doe", TierHard)
	if err != nil {
		t.Fatal(err)
	}

	if !ct.Regex.MatchString("Contact Jane Doe about it") {
		t.Error("expected case-sensitive match")
	}
	if !ct.Regex.MatchString("contact jane doe about it") {
		t.Error("expected case-insensitive match")
	}
	if !ct.Regex.MatchString("JANE DOE") {
		t.Error("expected uppercase match")
	}
}

func TestBlocklist_NoFile(t *testing.T) {
	terms, err := LoadBlocklist("/nonexistent/.blocklist")
	if err != nil {
		t.Errorf("expected nil error for missing file, got %v", err)
	}
	if terms != nil {
		t.Errorf("expected nil terms for missing file, got %v", terms)
	}
}

// --- Reviewed terms tests ---

func TestReviewedTerms_CRUD(t *testing.T) {
	dir := t.TempDir()

	rt, err := LoadReviewedTerms(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(rt.Terms) != 0 {
		t.Errorf("expected 0 terms, got %d", len(rt.Terms))
	}

	// Add
	rt.Add("markdownParser", "soft_blocklist", "variable name", "claude-agent", []string{"internal/parser/*.go"})
	if len(rt.Terms) != 1 {
		t.Fatalf("expected 1 term after add, got %d", len(rt.Terms))
	}

	// Save and reload
	if err := rt.Save(dir); err != nil {
		t.Fatal(err)
	}
	rt2, err := LoadReviewedTerms(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(rt2.Terms) != 1 {
		t.Fatalf("expected 1 term after reload, got %d", len(rt2.Terms))
	}

	// IsReviewed
	if !rt2.IsReviewed("markdownParser", "internal/parser/md.go", "soft_blocklist") {
		t.Error("expected term to be reviewed for matching glob")
	}
	if rt2.IsReviewed("markdownParser", "cmd/main.go", "soft_blocklist") {
		t.Error("expected term to NOT be reviewed for non-matching path")
	}
	if rt2.IsReviewed("otherTerm", "internal/parser/md.go", "soft_blocklist") {
		t.Error("expected different term to NOT be reviewed")
	}

	// Merge files on re-add
	rt2.Add("markdownParser", "soft_blocklist", "updated reason", "claude-agent", []string{"cmd/*.go"})
	if len(rt2.Terms) != 1 {
		t.Errorf("expected merge, not duplicate: got %d terms", len(rt2.Terms))
	}
	if len(rt2.Terms[0].Files) != 2 {
		t.Errorf("expected 2 file patterns after merge, got %d", len(rt2.Terms[0].Files))
	}

	// Remove
	removed := rt2.Remove("markdownParser", "soft_blocklist")
	if !removed {
		t.Error("expected Remove to return true")
	}
	if len(rt2.Terms) != 0 {
		t.Errorf("expected 0 terms after remove, got %d", len(rt2.Terms))
	}
}

// --- Output tests ---

func TestScanResult_FormatHuman_Pass(t *testing.T) {
	r := &ScanResult{Passed: true, FilesScanned: 3}
	out := r.FormatHuman()
	if out == "" {
		t.Error("expected non-empty output")
	}
	if !contains(out, "PASSED") {
		t.Error("expected PASSED in output")
	}
}

func TestScanResult_FormatHuman_Blocked(t *testing.T) {
	r := &ScanResult{
		Passed:       false,
		FilesScanned: 2,
		Violations: []Violation{
			{File: "Makefile", Line: 3, Tier: TierSoft, Category: CatLocalPath, Rule: "local_path_unix", Redacted: "/Us***/..."},
		},
		PathViolations: []PathViolation{
			{File: "Transcript.md", Reason: "not in allowed directories"},
		},
	}
	out := r.FormatHuman()
	if !contains(out, "BLOCKED") {
		t.Error("expected BLOCKED in output")
	}
	if !contains(out, "Makefile:3") {
		t.Error("expected file:line in output")
	}
	if !contains(out, "Commit blocked") {
		t.Error("expected 'Commit blocked' in output")
	}
}

func TestScanResult_FormatJSON(t *testing.T) {
	r := &ScanResult{Passed: true, FilesScanned: 1}
	j := r.FormatJSON()
	if j == "" || j[0] != '{' {
		t.Error("expected JSON output starting with {")
	}
}

// --- Integration: full pipeline ---

func TestScanFiles_Integration(t *testing.T) {
	dir := t.TempDir()

	fileContent := map[string][]byte{
		".scripts/config.sh": []byte("API_URL=https://api.example.com\nSECRET_KEY = \"sk-abcdefghijklmnopqrstuvwxyz1234\"\n"),
	}

	s := &Scanner{
		VaultPath: dir,
		Config:    DefaultGuardConfig(),
		patterns:  builtinPatterns(),
		reviewed:  &ReviewedTerms{},
		ContentReader: func(file string) ([]byte, error) {
			if data, ok := fileContent[file]; ok {
				return data, nil
			}
			return nil, os.ErrNotExist
		},
	}

	result, err := s.ScanFiles([]string{".scripts/config.sh"})
	if err != nil {
		t.Fatal(err)
	}

	// Should find API key violation
	if result.Passed {
		t.Error("expected scan to fail due to API key")
	}
	if len(result.Violations) == 0 {
		t.Error("expected at least one violation")
	}
}

func TestScanFiles_PathRejection(t *testing.T) {
	dir := t.TempDir()
	s := &Scanner{
		VaultPath: dir,
		Config:    DefaultGuardConfig(),
		patterns:  builtinPatterns(),
		reviewed:  &ReviewedTerms{},
	}

	result, err := s.ScanFiles([]string{"projects/secret.md", "Transcript.md"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed {
		t.Error("expected path violations to block")
	}
	if len(result.PathViolations) != 2 {
		t.Errorf("expected 2 path violations, got %d", len(result.PathViolations))
	}
}

func TestScanFiles_ReviewedTermPassthrough(t *testing.T) {
	dir := t.TempDir()

	reviewed := &ReviewedTerms{}
	reviewed.Add("/Users/jdoe/", "pii_local_path", "build path", "claude-agent", []string{".scripts/Makefile"})

	s := &Scanner{
		VaultPath: dir,
		Config:    DefaultGuardConfig(),
		patterns:  builtinPatterns(),
		reviewed:  reviewed,
	}

	// The file content has a local path but it's reviewed
	result, err := s.ScanFiles([]string{".scripts/Makefile"})
	if err != nil {
		t.Fatal(err)
	}

	// Should pass (no content to scan since we're not reading from git)
	// but with a real file content it would produce a warning instead of violation
	if !result.Passed {
		t.Error("expected scan to pass for path-only check (no git content)")
	}
}

// --- Config-driven scanner tests ---

func TestScanFiles_GuardDisabled(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultGuardConfig()
	cfg.Enabled = false

	s, err := NewScannerWithConfig(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	s.ContentReader = func(file string) ([]byte, error) {
		return []byte("secret_key = \"sk-abcdefghijklmnopqrstuvwxyz1234\""), nil
	}

	result, err := s.ScanFiles([]string{".scripts/test.go"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed {
		t.Error("expected scan to pass when guard is disabled")
	}
}

func TestScanFiles_PatternToggle(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultGuardConfig()
	cfg.PII.Patterns.LocalPath = false

	s, err := NewScannerWithConfig(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	s.ContentReader = func(file string) ([]byte, error) {
		return []byte("path = /Users/jdoe/Documents/vault"), nil
	}

	result, err := s.ScanFiles([]string{".scripts/Makefile"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed {
		t.Error("expected scan to pass when local_path pattern is disabled")
	}
}

func TestScanFiles_SoftModeWarn(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultGuardConfig()
	cfg.SoftMode = "warn"

	s, err := NewScannerWithConfig(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	s.ContentReader = func(file string) ([]byte, error) {
		return []byte("path = /Users/jdoe/Documents/vault"), nil
	}

	result, err := s.ScanFiles([]string{".scripts/Makefile"})
	if err != nil {
		t.Fatal(err)
	}
	// Soft violations should be warnings, not blocking
	if !result.Passed {
		t.Error("expected scan to pass in warn mode for soft violations")
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warnings for soft violations in warn mode")
	}
}

func TestScanFiles_PathFilterDisabled(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultGuardConfig()
	cfg.PathFilter.Enabled = false

	s, err := NewScannerWithConfig(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	s.ContentReader = func(file string) ([]byte, error) {
		return []byte("just some text"), nil
	}

	result, err := s.ScanFiles([]string{"projects/notes.md"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed {
		t.Error("expected scan to pass when path filter is disabled")
	}
	if len(result.PathViolations) != 0 {
		t.Error("expected no path violations when path filter is disabled")
	}
}

// --- Friendly output tests ---

func TestFormatFriendly_Pass(t *testing.T) {
	r := &ScanResult{Passed: true, FilesScanned: 3}
	out := r.FormatFriendly()
	if !contains(out, "All clear") {
		t.Error("expected 'All clear' in friendly output for passing scan")
	}
}

func TestFormatFriendly_Blocked(t *testing.T) {
	r := &ScanResult{
		Passed:       false,
		FilesScanned: 2,
		Violations: []Violation{
			{File: "Makefile", Line: 3, Tier: TierSoft, Category: CatLocalPath, Rule: "local_path_unix", Redacted: "/Us***/Documents/..."},
		},
	}
	out := r.FormatFriendly()
	if !contains(out, "personal info") {
		t.Error("expected 'personal info' in friendly output")
	}
	if !contains(out, "same guard allow") {
		t.Error("expected 'same guard allow' command in friendly output")
	}
	if !contains(out, "notes are never touched") {
		t.Error("expected reassurance about notes")
	}
}

func TestCategoryLabel(t *testing.T) {
	tests := []struct {
		cat  Category
		want string
	}{
		{CatEmail, "An email address"},
		{CatPhone, "A phone number"},
		{CatSSN, "A social security number"},
		{CatLocalPath, "A local file path"},
		{CatAPIKey, "An API key"},
		{CatAWSKey, "An AWS access key"},
		{CatPrivateKey, "A private key"},
		{CatHardBlock, "A blocklisted term"},
		{CatSoftBlock, "A blocklisted term (soft)"},
		{CatPathBlock, "Not in allowed directories"},
	}
	for _, tt := range tests {
		got := CategoryLabel(tt.cat)
		if got != tt.want {
			t.Errorf("CategoryLabel(%q) = %q, want %q", tt.cat, got, tt.want)
		}
	}
}

// --- Last scan cache tests ---

func TestSaveLoadLastScan(t *testing.T) {
	dir := t.TempDir()
	result := &ScanResult{
		Violations: []Violation{
			{File: "test.go", Line: 5, Category: CatEmail, Redacted: "us***com"},
		},
		PathViolations: []PathViolation{
			{File: "notes.md", Reason: "not in allowed directories"},
		},
	}

	if err := SaveLastScan(dir, result); err != nil {
		t.Fatal(err)
	}

	ls, err := LoadLastScan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ls.Violations) != 1 {
		t.Errorf("expected 1 violation, got %d", len(ls.Violations))
	}
	if ls.Violations[0].File != "test.go" {
		t.Errorf("expected file 'test.go', got %q", ls.Violations[0].File)
	}
	if len(ls.PathViolations) != 1 {
		t.Errorf("expected 1 path violation, got %d", len(ls.PathViolations))
	}
}

func TestLoadLastScan_NotFound(t *testing.T) {
	_, err := LoadLastScan(t.TempDir())
	if err == nil {
		t.Error("expected error for missing last scan file")
	}
}

// helper
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && containsImpl(s, substr))
}

func containsImpl(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
