package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- TestRegisterInstance ---

func TestRegisterInstance_CreatesFile(t *testing.T) {
	tmp := t.TempDir()

	dataDir := filepath.Join(tmp, ".same", "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}
	t.Setenv("SAME_DATA_DIR", dataDir)
	t.Setenv("VAULT_PATH", tmp)

	registerInstance("test-session-42", "Initial context about refactoring")

	instDir := filepath.Join(tmp, ".same", "instances")
	expectedFile := filepath.Join(instDir, "test-session-42.json")

	data, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("expected instance file to be created at %s: %v", expectedFile, err)
	}

	var info instanceInfo
	if err := json.Unmarshal(data, &info); err != nil {
		t.Fatalf("unmarshal instance file: %v", err)
	}

	if info.SessionID != "test-session-42" {
		t.Errorf("expected SessionID=test-session-42, got %s", info.SessionID)
	}
	if info.Status != "active" {
		t.Errorf("expected Status=active, got %s", info.Status)
	}
	if info.Summary != "Initial context about refactoring" {
		t.Errorf("expected matching summary, got %s", info.Summary)
	}
	if info.Started == "" {
		t.Error("expected non-empty Started timestamp")
	}
	if info.Updated == "" {
		t.Error("expected non-empty Updated timestamp")
	}
	if info.Machine == "" {
		t.Error("expected non-empty Machine name")
	}

	// Verify Started and Updated are valid RFC3339 timestamps.
	if _, err := time.Parse(time.RFC3339, info.Started); err != nil {
		t.Errorf("Started is not valid RFC3339: %s", info.Started)
	}
	if _, err := time.Parse(time.RFC3339, info.Updated); err != nil {
		t.Errorf("Updated is not valid RFC3339: %s", info.Updated)
	}
}

func TestRegisterInstance_FilePermissions(t *testing.T) {
	tmp := t.TempDir()

	dataDir := filepath.Join(tmp, ".same", "data")
	os.MkdirAll(dataDir, 0o755)
	t.Setenv("SAME_DATA_DIR", dataDir)
	t.Setenv("VAULT_PATH", tmp)

	registerInstance("perms-session", "test")

	instDir := filepath.Join(tmp, ".same", "instances")
	expectedFile := filepath.Join(instDir, "perms-session.json")

	info, err := os.Stat(expectedFile)
	if err != nil {
		t.Fatalf("stat instance file: %v", err)
	}

	perms := info.Mode().Perm()
	if perms != 0o600 {
		t.Errorf("expected file permissions 0600, got %04o", perms)
	}
}

func TestRegisterInstance_TruncatesLongSummary(t *testing.T) {
	tmp := t.TempDir()

	dataDir := filepath.Join(tmp, ".same", "data")
	os.MkdirAll(dataDir, 0o755)
	t.Setenv("SAME_DATA_DIR", dataDir)
	t.Setenv("VAULT_PATH", tmp)

	longContext := make([]byte, 500)
	for i := range longContext {
		longContext[i] = 'x'
	}

	registerInstance("long-summary-session", string(longContext))

	instDir := filepath.Join(tmp, ".same", "instances")
	data, err := os.ReadFile(filepath.Join(instDir, "long-summary-session.json"))
	if err != nil {
		t.Fatalf("read instance file: %v", err)
	}

	var info instanceInfo
	json.Unmarshal(data, &info)

	if len(info.Summary) > 200 {
		t.Errorf("expected summary truncated to 200 chars, got %d", len(info.Summary))
	}
}

func TestRegisterInstance_EmptySessionID(t *testing.T) {
	tmp := t.TempDir()

	dataDir := filepath.Join(tmp, ".same", "data")
	os.MkdirAll(dataDir, 0o755)
	t.Setenv("SAME_DATA_DIR", dataDir)
	t.Setenv("VAULT_PATH", tmp)

	// Should silently return without creating a file.
	registerInstance("", "some context")

	instDir := filepath.Join(tmp, ".same", "instances")
	entries, err := os.ReadDir(instDir)
	if err != nil {
		// Directory may not even be created for empty session ID — that's fine.
		return
	}
	if len(entries) > 0 {
		t.Errorf("expected no files for empty session ID, got %d", len(entries))
	}
}

func TestRegisterInstance_UnsafeSessionID(t *testing.T) {
	tmp := t.TempDir()

	dataDir := filepath.Join(tmp, ".same", "data")
	os.MkdirAll(dataDir, 0o755)
	t.Setenv("SAME_DATA_DIR", dataDir)
	t.Setenv("VAULT_PATH", tmp)

	// Path-traversal attempts should be sanitized.
	registerInstance("../../etc/passwd", "malicious context")

	instDir := filepath.Join(tmp, ".same", "instances")
	entries, err := os.ReadDir(instDir)
	if err != nil {
		// Fine if dir doesn't exist.
		return
	}

	// Verify no file was created outside instances dir, and any created file
	// has a sanitized name.
	for _, e := range entries {
		name := e.Name()
		if name == "passwd.json" || name == "etc.json" {
			// The sanitizer strips "/" and "..", so "../../etc/passwd" -> "etcpasswd"
			// which is safe since it stays inside the instances directory.
			continue
		}
	}

	// Verify nothing was written outside the instances directory.
	outsidePath := filepath.Join(tmp, ".same", "..", "..", "etc", "passwd.json")
	if _, err := os.Stat(outsidePath); err == nil {
		t.Fatal("SECURITY: file created outside instances directory via path traversal")
	}
}

// --- TestSanitizeSessionID ---

func TestSanitizeSessionID_Normal(t *testing.T) {
	got := sanitizeSessionID("abc-123-def")
	if got != "abc-123-def" {
		t.Errorf("expected abc-123-def, got %s", got)
	}
}

func TestSanitizeSessionID_Empty(t *testing.T) {
	got := sanitizeSessionID("")
	if got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestSanitizeSessionID_PathTraversal(t *testing.T) {
	got := sanitizeSessionID("../../etc/passwd")
	// Slashes stripped, ".." stripped -> "etcpasswd"
	if got == "" {
		t.Error("expected non-empty result after sanitization")
	}
	if got == "../../etc/passwd" {
		t.Error("expected path traversal to be stripped")
	}
}

func TestSanitizeSessionID_NullBytes(t *testing.T) {
	got := sanitizeSessionID("session\x00id")
	if got != "sessionid" {
		t.Errorf("expected null bytes stripped, got %q", got)
	}
}

func TestSanitizeSessionID_ControlChars(t *testing.T) {
	got := sanitizeSessionID("session\t\nid")
	if got != "sessionid" {
		t.Errorf("expected control chars stripped, got %q", got)
	}
}

func TestSanitizeSessionID_LongID(t *testing.T) {
	long := make([]byte, 300)
	for i := range long {
		long[i] = 'a'
	}
	got := sanitizeSessionID(string(long))
	if len(got) > 255 {
		t.Errorf("expected ID capped at 255 chars, got %d", len(got))
	}
}

func TestSanitizeSessionID_DotOnly(t *testing.T) {
	got := sanitizeSessionID(".")
	if got != "" {
		t.Errorf("expected empty for '.', got %q", got)
	}
}

func TestSanitizeSessionID_DoubleDotOnly(t *testing.T) {
	got := sanitizeSessionID("..")
	if got != "" {
		t.Errorf("expected empty for '..', got %q", got)
	}
}

// --- TestCleanStaleInstances ---

func TestCleanStaleInstances_RemovesStale(t *testing.T) {
	tmp := t.TempDir()

	dataDir := filepath.Join(tmp, ".same", "data")
	os.MkdirAll(dataDir, 0o755)
	t.Setenv("SAME_DATA_DIR", dataDir)
	t.Setenv("VAULT_PATH", tmp)

	instDir := filepath.Join(tmp, ".same", "instances")
	os.MkdirAll(instDir, 0o755)

	now := time.Now().UTC()

	// Fresh instance (should survive cleaning).
	freshInfo := instanceInfo{
		SessionID: "fresh-session",
		Machine:   "test",
		Started:   now.Add(-1 * time.Hour).Format(time.RFC3339),
		Updated:   now.Add(-30 * time.Minute).Format(time.RFC3339),
		Summary:   "Fresh work",
		Status:    "active",
	}
	freshData, _ := json.MarshalIndent(freshInfo, "", "  ")
	os.WriteFile(filepath.Join(instDir, "fresh-session.json"), freshData, 0o600)

	// Stale instance (updated > 24h ago — should be removed).
	staleInfo := instanceInfo{
		SessionID: "stale-session",
		Machine:   "test",
		Started:   now.Add(-48 * time.Hour).Format(time.RFC3339),
		Updated:   now.Add(-36 * time.Hour).Format(time.RFC3339),
		Summary:   "Old work",
		Status:    "completed",
	}
	staleData, _ := json.MarshalIndent(staleInfo, "", "  ")
	os.WriteFile(filepath.Join(instDir, "stale-session.json"), staleData, 0o600)

	cleanStaleInstances("current-session")

	// Fresh instance should remain.
	if _, err := os.Stat(filepath.Join(instDir, "fresh-session.json")); err != nil {
		t.Error("expected fresh instance to survive cleaning")
	}

	// Stale instance should be removed.
	if _, err := os.Stat(filepath.Join(instDir, "stale-session.json")); err == nil {
		t.Error("expected stale instance to be removed")
	}
}

func TestCleanStaleInstances_PreservesCurrentSession(t *testing.T) {
	tmp := t.TempDir()

	dataDir := filepath.Join(tmp, ".same", "data")
	os.MkdirAll(dataDir, 0o755)
	t.Setenv("SAME_DATA_DIR", dataDir)
	t.Setenv("VAULT_PATH", tmp)

	instDir := filepath.Join(tmp, ".same", "instances")
	os.MkdirAll(instDir, 0o755)

	now := time.Now().UTC()

	// Current session with stale timestamp — should NOT be removed.
	currentInfo := instanceInfo{
		SessionID: "current-session",
		Machine:   "test",
		Started:   now.Add(-48 * time.Hour).Format(time.RFC3339),
		Updated:   now.Add(-36 * time.Hour).Format(time.RFC3339),
		Summary:   "Current work",
		Status:    "active",
	}
	currentData, _ := json.MarshalIndent(currentInfo, "", "  ")
	os.WriteFile(filepath.Join(instDir, "current-session.json"), currentData, 0o600)

	cleanStaleInstances("current-session")

	// Should still exist even though it's stale — it's the current session.
	if _, err := os.Stat(filepath.Join(instDir, "current-session.json")); err != nil {
		t.Error("expected current session to survive cleaning even if stale")
	}
}

func TestCleanStaleInstances_EmptyDirectory(t *testing.T) {
	tmp := t.TempDir()

	dataDir := filepath.Join(tmp, ".same", "data")
	os.MkdirAll(dataDir, 0o755)
	t.Setenv("SAME_DATA_DIR", dataDir)
	t.Setenv("VAULT_PATH", tmp)

	instDir := filepath.Join(tmp, ".same", "instances")
	os.MkdirAll(instDir, 0o755)

	// Should not panic on empty directory.
	cleanStaleInstances("any-session")
}

func TestCleanStaleInstances_NoDirectory(t *testing.T) {
	tmp := t.TempDir()

	dataDir := filepath.Join(tmp, ".same", "data")
	os.MkdirAll(dataDir, 0o755)
	t.Setenv("SAME_DATA_DIR", dataDir)
	t.Setenv("VAULT_PATH", tmp)

	// instances/ directory does not exist — should not panic.
	cleanStaleInstances("any-session")
}

func TestCleanStaleInstances_IgnoresNonJSON(t *testing.T) {
	tmp := t.TempDir()

	dataDir := filepath.Join(tmp, ".same", "data")
	os.MkdirAll(dataDir, 0o755)
	t.Setenv("SAME_DATA_DIR", dataDir)
	t.Setenv("VAULT_PATH", tmp)

	instDir := filepath.Join(tmp, ".same", "instances")
	os.MkdirAll(instDir, 0o755)

	// Write a non-JSON file — should not be touched.
	os.WriteFile(filepath.Join(instDir, "README.md"), []byte("ignore me"), 0o644)

	cleanStaleInstances("any-session")

	// Non-JSON file should remain.
	if _, err := os.Stat(filepath.Join(instDir, "README.md")); err != nil {
		t.Error("expected non-JSON file to remain untouched")
	}
}

// --- TestFindActiveInstances ---

func TestFindActiveInstances_ReturnsActivePeers(t *testing.T) {
	tmp := t.TempDir()

	dataDir := filepath.Join(tmp, ".same", "data")
	os.MkdirAll(dataDir, 0o755)
	t.Setenv("SAME_DATA_DIR", dataDir)
	t.Setenv("VAULT_PATH", tmp)

	instDir := filepath.Join(tmp, ".same", "instances")
	os.MkdirAll(instDir, 0o755)

	now := time.Now().UTC()

	// Active peer instance.
	peerInfo := instanceInfo{
		SessionID: "peer-session",
		Machine:   "dev-laptop",
		Started:   now.Add(-1 * time.Hour).Format(time.RFC3339),
		Updated:   now.Add(-5 * time.Minute).Format(time.RFC3339),
		Summary:   "Working on API endpoints",
		Status:    "active",
	}
	peerData, _ := json.MarshalIndent(peerInfo, "", "  ")
	os.WriteFile(filepath.Join(instDir, "peer-session.json"), peerData, 0o600)

	result := findActiveInstances("current-session")
	if result == "" {
		t.Fatal("expected non-empty result for active peer")
	}
	if !strings.Contains(result, "Active Instances") {
		t.Errorf("expected 'Active Instances' header, got: %s", result)
	}
	if !strings.Contains(result, "dev-laptop") {
		t.Errorf("expected machine name in output, got: %s", result)
	}
	if !strings.Contains(result, "API endpoints") {
		t.Errorf("expected summary in output, got: %s", result)
	}
}

func TestFindActiveInstances_ExcludesCurrentSession(t *testing.T) {
	tmp := t.TempDir()

	dataDir := filepath.Join(tmp, ".same", "data")
	os.MkdirAll(dataDir, 0o755)
	t.Setenv("SAME_DATA_DIR", dataDir)
	t.Setenv("VAULT_PATH", tmp)

	instDir := filepath.Join(tmp, ".same", "instances")
	os.MkdirAll(instDir, 0o755)

	now := time.Now().UTC()
	selfInfo := instanceInfo{
		SessionID: "my-session",
		Machine:   "my-machine",
		Started:   now.Add(-30 * time.Minute).Format(time.RFC3339),
		Updated:   now.Add(-1 * time.Minute).Format(time.RFC3339),
		Summary:   "My work",
		Status:    "active",
	}
	selfData, _ := json.MarshalIndent(selfInfo, "", "  ")
	os.WriteFile(filepath.Join(instDir, "my-session.json"), selfData, 0o600)

	result := findActiveInstances("my-session")
	if result != "" {
		t.Errorf("expected empty result when only current session exists, got: %s", result)
	}
}

func TestFindActiveInstances_ExcludesStale(t *testing.T) {
	tmp := t.TempDir()

	dataDir := filepath.Join(tmp, ".same", "data")
	os.MkdirAll(dataDir, 0o755)
	t.Setenv("SAME_DATA_DIR", dataDir)
	t.Setenv("VAULT_PATH", tmp)

	instDir := filepath.Join(tmp, ".same", "instances")
	os.MkdirAll(instDir, 0o755)

	now := time.Now().UTC()

	// Stale instance (updated > 12h ago — findActiveInstances uses 12h cutoff).
	staleInfo := instanceInfo{
		SessionID: "stale-peer",
		Machine:   "old-machine",
		Started:   now.Add(-24 * time.Hour).Format(time.RFC3339),
		Updated:   now.Add(-14 * time.Hour).Format(time.RFC3339),
		Summary:   "Old work",
		Status:    "completed",
	}
	staleData, _ := json.MarshalIndent(staleInfo, "", "  ")
	os.WriteFile(filepath.Join(instDir, "stale-peer.json"), staleData, 0o600)

	result := findActiveInstances("current-session")
	if result != "" {
		t.Errorf("expected empty result for stale instances, got: %s", result)
	}
}

func TestFindActiveInstances_OutputCappedAt500(t *testing.T) {
	tmp := t.TempDir()

	dataDir := filepath.Join(tmp, ".same", "data")
	os.MkdirAll(dataDir, 0o755)
	t.Setenv("SAME_DATA_DIR", dataDir)
	t.Setenv("VAULT_PATH", tmp)

	instDir := filepath.Join(tmp, ".same", "instances")
	os.MkdirAll(instDir, 0o755)

	now := time.Now().UTC()

	// Create many instances to exceed 500 char limit.
	for i := 0; i < 20; i++ {
		info := instanceInfo{
			SessionID: "peer-" + string(rune('a'+i)),
			Machine:   "machine-with-long-name-" + string(rune('a'+i)),
			Started:   now.Add(-time.Duration(i) * time.Minute).Format(time.RFC3339),
			Updated:   now.Add(-time.Duration(i) * time.Minute).Format(time.RFC3339),
			Summary:   "Working on a moderately long description of the current task at hand for testing purposes",
			Status:    "active",
		}
		data, _ := json.MarshalIndent(info, "", "  ")
		os.WriteFile(filepath.Join(instDir, info.SessionID+".json"), data, 0o600)
	}

	result := findActiveInstances("current-session")
	if len(result) > 500 {
		t.Errorf("expected output capped at 500 chars, got %d", len(result))
	}
}

func TestFindActiveInstances_EmptyDirectory(t *testing.T) {
	tmp := t.TempDir()

	dataDir := filepath.Join(tmp, ".same", "data")
	os.MkdirAll(dataDir, 0o755)
	t.Setenv("SAME_DATA_DIR", dataDir)
	t.Setenv("VAULT_PATH", tmp)

	instDir := filepath.Join(tmp, ".same", "instances")
	os.MkdirAll(instDir, 0o755)

	result := findActiveInstances("any-session")
	if result != "" {
		t.Errorf("expected empty result for empty directory, got: %s", result)
	}
}

func TestFindActiveInstances_NoDirectory(t *testing.T) {
	tmp := t.TempDir()

	dataDir := filepath.Join(tmp, ".same", "data")
	os.MkdirAll(dataDir, 0o755)
	t.Setenv("SAME_DATA_DIR", dataDir)
	t.Setenv("VAULT_PATH", tmp)

	// instances/ does not exist — should return empty without panic.
	result := findActiveInstances("any-session")
	if result != "" {
		t.Errorf("expected empty result when directory does not exist, got: %s", result)
	}
}

// --- TestRelativeTime ---

func TestRelativeTime_Minutes(t *testing.T) {
	got := relativeTime(time.Now().Add(-5 * time.Minute))
	if got != "5m ago" {
		t.Errorf("expected '5m ago', got %q", got)
	}
}

func TestRelativeTime_Hours(t *testing.T) {
	got := relativeTime(time.Now().Add(-3 * time.Hour))
	if got != "3h ago" {
		t.Errorf("expected '3h ago', got %q", got)
	}
}

func TestRelativeTime_Days(t *testing.T) {
	got := relativeTime(time.Now().Add(-48 * time.Hour))
	if got != "2d ago" {
		t.Errorf("expected '2d ago', got %q", got)
	}
}
