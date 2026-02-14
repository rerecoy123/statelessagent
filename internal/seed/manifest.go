package seed

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// ManifestURL is the default location of the seed manifest.
	ManifestURL = "https://raw.githubusercontent.com/sgx-labs/seed-vaults/main/seeds.json"

	// ManifestCacheTTL is how long a cached manifest is considered fresh.
	ManifestCacheTTL = 1 * time.Hour

	// MaxManifestSize is the maximum manifest download size.
	MaxManifestSize = 1 * 1024 * 1024 // 1 MB
)

// Manifest is the top-level seed registry structure.
type Manifest struct {
	SchemaVersion int    `json:"schema_version"`
	Seeds         []Seed `json:"seeds"`
}

// Seed describes a single installable seed vault.
type Seed struct {
	Name           string   `json:"name"`
	DisplayName    string   `json:"display_name"`
	Description    string   `json:"description"`
	Audience       string   `json:"audience"`
	NoteCount      int      `json:"note_count"`
	SizeKB         int      `json:"size_kb"`
	Tags           []string `json:"tags"`
	MinSameVersion string   `json:"min_same_version"`
	Path           string   `json:"path"`
	Featured       bool     `json:"featured"`
}

// manifestCache wraps a manifest with a timestamp for TTL checking.
type manifestCache struct {
	FetchedAt time.Time `json:"fetched_at"`
	Manifest  Manifest  `json:"manifest"`
}

// manifestCachePath returns the path to the cached manifest file.
func manifestCachePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "same", "seed-manifest.json")
}

// FetchManifest retrieves the seed manifest, using a local cache when fresh.
// Set forceRefresh to bypass the cache.
func FetchManifest(forceRefresh bool) (*Manifest, error) {
	cachePath := manifestCachePath()

	// Try cache first (unless forced)
	if !forceRefresh {
		if m, err := loadCachedManifest(cachePath); err == nil {
			return m, nil
		}
	}

	// Fetch from remote
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(ManifestURL)
	if err != nil {
		// If network fails, try stale cache
		if m, err := loadCachedManifest(cachePath); err == nil {
			return m, nil
		}
		return nil, fmt.Errorf("fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Try stale cache on HTTP errors
		if m, err := loadCachedManifest(cachePath); err == nil {
			return m, nil
		}
		return nil, fmt.Errorf("fetch manifest: HTTP %d", resp.StatusCode)
	}

	// SECURITY: limit response size
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxManifestSize))
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	if manifest.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupported manifest schema version: %d", manifest.SchemaVersion)
	}

	// Validate seed names
	for _, s := range manifest.Seeds {
		if err := validateSeedName(s.Name); err != nil {
			return nil, fmt.Errorf("invalid seed in manifest: %w", err)
		}
	}

	// Write cache
	saveManifestCache(cachePath, &manifest)

	return &manifest, nil
}

// FindSeed looks up a seed by name in the manifest.
func FindSeed(manifest *Manifest, name string) *Seed {
	lower := strings.ToLower(name)
	for i := range manifest.Seeds {
		if strings.ToLower(manifest.Seeds[i].Name) == lower {
			return &manifest.Seeds[i]
		}
	}
	return nil
}

// validateSeedName checks that a seed name is safe for use as a directory name.
func validateSeedName(name string) error {
	if name == "" {
		return fmt.Errorf("seed name cannot be empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("seed name too long (%d chars, max 64)", len(name))
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return fmt.Errorf("seed name must be lowercase alphanumeric with hyphens (got %q)", name)
		}
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return fmt.Errorf("seed name cannot start or end with a hyphen")
	}
	return nil
}

// loadCachedManifest reads and validates the cached manifest.
// Returns nil if the cache is missing, stale, or corrupt.
// Note: for network-fallback usage, we accept stale caches.
func loadCachedManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cache manifestCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	if time.Since(cache.FetchedAt) > ManifestCacheTTL {
		return nil, fmt.Errorf("cache expired")
	}
	if cache.Manifest.SchemaVersion != 1 {
		return nil, fmt.Errorf("stale schema version")
	}
	return &cache.Manifest, nil
}

// saveManifestCache writes the manifest to the cache file.
func saveManifestCache(path string, m *Manifest) {
	cache := manifestCache{
		FetchedAt: time.Now(),
		Manifest:  *m,
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return
	}
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, data, 0o600)
}
