package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafeVaultPath_RejectsTraversalNullAbsoluteAndPrivate(t *testing.T) {
	setupTestVault(t)

	tests := []string{
		"../secret.md",
		"notes/../../secret.md",
		"notes/..\\..\\secret.md",
		"notes/evil\x00.md",
		"/etc/passwd",
		"C:/Windows/System32/drivers/etc/hosts",
		"_PRIVATE/secret.md",
		"_private/secret.md",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if got := safeVaultPath(input); got != "" {
				t.Fatalf("expected unsafe path %q to be rejected, got %q", input, got)
			}
		})
	}
}

func TestSafeVaultPath_SymlinkEscapeBlocked(t *testing.T) {
	vault := setupTestVault(t)
	notesDir := filepath.Join(vault, "notes")
	if err := os.MkdirAll(notesDir, 0o755); err != nil {
		t.Fatalf("mkdir notes: %v", err)
	}

	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.md"), []byte("secret"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}

	if err := os.Symlink(outside, filepath.Join(notesDir, "escape")); err != nil {
		t.Skipf("symlink not available on this platform: %v", err)
	}

	if got := safeVaultPath("notes/escape/secret.md"); got != "" {
		t.Fatalf("expected symlink escape to be blocked, got %q", got)
	}
}

func TestSafeVaultPath_SymlinkWithinVaultAllowed(t *testing.T) {
	vault := setupTestVault(t)
	notesDir := filepath.Join(vault, "notes")
	if err := os.MkdirAll(notesDir, 0o755); err != nil {
		t.Fatalf("mkdir notes: %v", err)
	}
	if err := os.WriteFile(filepath.Join(notesDir, "ok.md"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("write note: %v", err)
	}
	if err := os.Symlink(notesDir, filepath.Join(vault, "alias")); err != nil {
		t.Skipf("symlink not available on this platform: %v", err)
	}

	if got := safeVaultPath("alias/ok.md"); got == "" {
		t.Fatal("expected symlink path within vault to be allowed")
	}
}
