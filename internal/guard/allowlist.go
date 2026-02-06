package guard

import (
	"path/filepath"
	"strings"
)

// defaultAllowedPaths are directories and files that may be committed.
// Everything else is rejected by omission.
var defaultAllowedPaths = []string{
	".scripts/",
	".claude/",
	".config/",
	".gitignore",
	".gitattributes",
	".prettierignore",
	"package.json",
	"package-lock.json",
	"SECURITY.md",
	"PRIVACY.md",
	"README.md",
	"SeedClaude.md",
	"LICENSE",
	"CHANGELOG.md",
}

// IsPathAllowed checks if a file path is in the allowlist.
// Paths are matched against the default allowlist plus any custom entries.
func IsPathAllowed(filePath string, customPaths []string) bool {
	all := append(defaultAllowedPaths, customPaths...)
	// Normalize to forward slashes for cross-platform matching
	normalized := filepath.ToSlash(filePath)

	for _, allowed := range all {
		allowed = filepath.ToSlash(allowed)

		if strings.HasSuffix(allowed, "/") {
			// Directory prefix match
			if strings.HasPrefix(normalized, allowed) || normalized == strings.TrimSuffix(allowed, "/") {
				return true
			}
		} else {
			// Exact file match (compare basename or full path)
			if normalized == allowed || filepath.Base(normalized) == allowed {
				return true
			}
		}
	}
	return false
}
