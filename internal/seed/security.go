// Package seed provides seed vault download, extraction, and installation.
package seed

import (
	"fmt"
	"path/filepath"
	"strings"
)

// allowedExtensions is the set of file extensions permitted in seed archives.
var allowedExtensions = map[string]bool{
	".md":      true,
	".toml":    true,
	".json":    true,
	".txt":     true,
	".yml":     true,
	".yaml":    true,
	".example": true,
	".gitkeep": true,
}

// validateExtractPath validates that a tar entry path is safe to extract into destDir.
// Returns the cleaned absolute destination path, or an error if the path is dangerous.
func validateExtractPath(entryPath, destDir string) (string, error) {
	// SECURITY: reject null bytes
	if strings.ContainsRune(entryPath, 0) {
		return "", fmt.Errorf("path contains null byte")
	}

	// Normalize the path
	clean := filepath.Clean(filepath.FromSlash(entryPath))

	// SECURITY: reject absolute paths
	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("absolute path rejected: %s", entryPath)
	}

	// SECURITY: reject traversal via ..
	for _, part := range strings.Split(filepath.ToSlash(clean), "/") {
		if part == ".." {
			return "", fmt.Errorf("path traversal rejected: %s", entryPath)
		}
	}

	// SECURITY: reject dot-prefixed components (hidden files/dirs)
	for _, part := range strings.Split(filepath.ToSlash(clean), "/") {
		if strings.HasPrefix(part, ".") && part != "." {
			return "", fmt.Errorf("hidden path component rejected: %s", entryPath)
		}
	}

	// SECURITY: check file extension
	ext := filepath.Ext(clean)
	// Allow directories (no extension) and .gitkeep
	if ext != "" && !allowedExtensions[strings.ToLower(ext)] {
		return "", fmt.Errorf("disallowed extension %q in: %s", ext, entryPath)
	}

	// Build the final destination path
	absResult := filepath.Join(destDir, clean)

	// SECURITY: final containment check — result must be under destDir
	absDest, err := filepath.Abs(destDir)
	if err != nil {
		return "", fmt.Errorf("cannot resolve destination: %w", err)
	}
	absResultClean, err := filepath.Abs(absResult)
	if err != nil {
		return "", fmt.Errorf("cannot resolve result path: %w", err)
	}
	if !pathWithinBase(absDest, absResultClean) {
		return "", fmt.Errorf("path escapes destination: %s", entryPath)
	}

	return absResultClean, nil
}

func pathWithinBase(base, candidate string) bool {
	rel, err := filepath.Rel(base, candidate)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, "../"))
}
