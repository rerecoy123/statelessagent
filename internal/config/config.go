// Package config provides configuration for the SAME binary.
// Reads from environment variables with sensible defaults.
package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Embedding model settings.
const (
	EmbeddingModel = "nomic-embed-text"
	EmbeddingDim   = 768
)

// Indexing settings.
const (
	ChunkTokenThreshold = 6000 // chunk notes longer than ~6K chars by H2 headings
	MaxEmbedChars       = 7500 // nomic-embed-text context limit ~8192 tokens
	MaxSnippetLength    = 500
)

// Memory engine settings.
const (
	SessionLogTable           = "session_log"
	ContextUsageTable         = "context_usage"
	MaxContextInjectionTokens = 1000
	ContextSurfacingMinChars  = 20
)

// HandoffDirectory returns the directory for session handoff notes.
// Reads SAME_HANDOFF_DIR env var, defaults to "sessions".
func HandoffDirectory() string {
	if v := os.Getenv("SAME_HANDOFF_DIR"); v != "" {
		return v
	}
	return "sessions"
}

// DecisionLogPath returns the path (relative to vault root) for the decision log.
// Reads SAME_DECISION_LOG env var, defaults to "decisions.md".
func DecisionLogPath() string {
	if v := os.Getenv("SAME_DECISION_LOG"); v != "" {
		return v
	}
	return "decisions.md"
}

// defaultSkipDirs are directories to skip during vault walks.
// SECURITY: _PRIVATE contains client-sensitive content and must never be indexed
// or auto-surfaced. Access to _PRIVATE requires explicit MCP tool calls.
var defaultSkipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	".smart-env":   true,
	".obsidian":    true,
	".logseq":      true,
	".same":        true,
	".claude":      true,
	".trash":       true,
	"_PRIVATE":     true,
}

// SkipDirs returns the set of directories to skip during vault walks.
// Reads SAME_SKIP_DIRS env var (comma-separated) to add extra dirs.
var SkipDirs = buildSkipDirs()

func buildSkipDirs() map[string]bool {
	dirs := make(map[string]bool)
	for k, v := range defaultSkipDirs {
		dirs[k] = v
	}
	if extra := os.Getenv("SAME_SKIP_DIRS"); extra != "" {
		for _, d := range strings.Split(extra, ",") {
			d = strings.TrimSpace(d)
			if d != "" {
				dirs[d] = true
			}
		}
	}
	return dirs
}

// VaultPath returns the vault root directory.
func VaultPath() string {
	if v := os.Getenv("VAULT_PATH"); v != "" {
		return v
	}
	return defaultVaultPath()
}

// OllamaURL returns the validated Ollama API URL.
// Panics if the URL does not point to localhost.
func OllamaURL() string {
	raw := os.Getenv("OLLAMA_URL")
	if raw == "" {
		raw = "http://localhost:11434"
	}
	u, err := url.Parse(raw)
	if err != nil {
		panic(fmt.Sprintf("invalid OLLAMA_URL: %v", err))
	}
	host := u.Hostname()
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		panic(fmt.Sprintf("OLLAMA_URL must point to localhost for security. Got: %s", host))
	}
	return raw
}

// DBPath returns the path to the SQLite database file.
func DBPath() string {
	return filepath.Join(DataDir(), "vault.db")
}

// DataDir returns the data directory for the same binary.
// Reads SAME_DATA_DIR env var, defaults to <vault>/.same/data.
func DataDir() string {
	if v := os.Getenv("SAME_DATA_DIR"); v != "" {
		return v
	}
	return filepath.Join(VaultPath(), ".same", "data")
}

// VaultRegistry holds registered vault paths with aliases.
type VaultRegistry struct {
	Vaults  map[string]string `json:"vaults"`  // alias -> path
	Default string            `json:"default"`  // alias of default vault
}

// RegistryPath returns the path to the vault registry file.
func RegistryPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "same", "vaults.json")
}

// LoadRegistry loads or creates the vault registry.
func LoadRegistry() *VaultRegistry {
	data, err := os.ReadFile(RegistryPath())
	if err != nil {
		return &VaultRegistry{Vaults: make(map[string]string)}
	}
	var reg VaultRegistry
	if err := json.Unmarshal(data, &reg); err != nil {
		return &VaultRegistry{Vaults: make(map[string]string)}
	}
	if reg.Vaults == nil {
		reg.Vaults = make(map[string]string)
	}
	return &reg
}

// Save writes the registry to disk.
func (r *VaultRegistry) Save() error {
	path := RegistryPath()
	os.MkdirAll(filepath.Dir(path), 0o755)
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ResolveVault resolves a vault alias to a path. Returns empty string if not found.
func (r *VaultRegistry) ResolveVault(alias string) string {
	if p, ok := r.Vaults[alias]; ok {
		return p
	}
	// Maybe it's already a path
	if info, err := os.Stat(alias); err == nil && info.IsDir() {
		return alias
	}
	return ""
}

// VaultOverride is set by the --vault global flag.
var VaultOverride string

// vaultMarkers are dotfiles/directories that indicate a knowledge base root.
// Checked in priority order: SAME's own marker first, then common tools.
var vaultMarkers = []string{".same", ".obsidian", ".logseq", ".foam", ".dendron"}

func defaultVaultPath() string {
	// Check --vault flag override first
	if VaultOverride != "" {
		reg := LoadRegistry()
		if resolved := reg.ResolveVault(VaultOverride); resolved != "" {
			return resolved
		}
		// Treat as direct path
		return VaultOverride
	}

	// Check registry default
	reg := LoadRegistry()
	if reg.Default != "" {
		if p, ok := reg.Vaults[reg.Default]; ok {
			return p
		}
	}

	// Auto-detect: check CWD for any known marker
	if cwd, err := os.Getwd(); err == nil {
		for _, marker := range vaultMarkers {
			if _, err := os.Stat(filepath.Join(cwd, marker)); err == nil {
				return cwd
			}
		}
	}

	// Walk up from binary location looking for any marker
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for i := 0; i < 5; i++ {
			for _, marker := range vaultMarkers {
				if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
					return dir
				}
			}
			dir = filepath.Dir(dir)
		}
	}

	// No vault found — return empty string (caller should show helpful error)
	return ""
}
