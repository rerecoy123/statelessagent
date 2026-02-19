package main

import (
	"strings"
	"testing"
	"time"

	"github.com/sgx-labs/statelessagent/internal/config"
	"github.com/sgx-labs/statelessagent/internal/store"
)

func TestRunAsk_NoChatProviderConfigured(t *testing.T) {
	vault := t.TempDir()
	origVault := config.VaultOverride
	config.VaultOverride = vault
	t.Cleanup(func() { config.VaultOverride = origVault })

	t.Setenv("SAME_EMBED_PROVIDER", "none")
	t.Setenv("SAME_CHAT_PROVIDER", "none")

	db, err := store.Open()
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer db.Close()

	if _, err := db.BulkInsertNotesLite([]store.NoteRecord{
		{
			Path:         "notes/arch.md",
			Title:        "Architecture",
			Tags:         "[]",
			ChunkID:      0,
			ChunkHeading: "",
			Text:         "We chose sqlite for portability.",
			Modified:     float64(time.Now().Unix()),
			ContentHash:  "test-hash-1",
			ContentType:  "note",
			Confidence:   0.8,
		},
	}); err != nil {
		t.Fatalf("BulkInsertNotesLite: %v", err)
	}

	err = runAsk("sqlite portability", "", 5)
	if err == nil {
		t.Fatal("expected error when no chat provider is configured")
	}
	if !strings.Contains(err.Error(), "No chat provider available") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunAsk_OllamaUnavailableReturnsActionableHint(t *testing.T) {
	vault := t.TempDir()
	origVault := config.VaultOverride
	config.VaultOverride = vault
	t.Cleanup(func() { config.VaultOverride = origVault })

	t.Setenv("SAME_EMBED_PROVIDER", "none")
	t.Setenv("SAME_CHAT_PROVIDER", "ollama")
	t.Setenv("OLLAMA_URL", "http://127.0.0.1:1")

	db, err := store.Open()
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer db.Close()

	if _, err := db.BulkInsertNotesLite([]store.NoteRecord{
		{
			Path:         "notes/ops.md",
			Title:        "Ops",
			Tags:         "[]",
			ChunkID:      0,
			ChunkHeading: "",
			Text:         "Operational checklist includes sqlite maintenance.",
			Modified:     float64(time.Now().Unix()),
			ContentHash:  "test-hash-2",
			ContentType:  "note",
			Confidence:   0.8,
		},
	}); err != nil {
		t.Fatalf("BulkInsertNotesLite: %v", err)
	}

	err = runAsk("sqlite maintenance", "", 5)
	if err == nil {
		t.Fatal("expected error when ollama is unavailable")
	}
	if !strings.Contains(err.Error(), "No chat provider available") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "SAME_CHAT_PROVIDER") {
		t.Fatalf("expected actionable hint in error, got: %v", err)
	}
}
