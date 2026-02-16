package store

import (
	"strings"
	"testing"
)

func TestNormalizeClaimPath_SecurityEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		// Null byte injection
		{"null byte in middle", "notes/foo\x00bar.md", true, "null byte"},
		{"null byte at start", "\x00notes/foo.md", true, "null byte"},
		{"null byte at end", "notes/foo.md\x00", true, "null byte"},

		// Path traversal variants
		{"simple traversal", "../etc/passwd", true, "within the vault"},
		{"double traversal", "../../etc/passwd", true, "within the vault"},
		{"traversal after dir", "notes/../../etc/passwd", true, "within the vault"},
		{"dot-dot only", "..", true, "within the vault"},
		{"dot only resolves to root", ".", true, "within the vault"},

		// Absolute paths
		{"unix absolute", "/etc/passwd", true, "relative"},
		{"windows drive C", "C:/Users/foo/bar.md", true, "relative"},
		{"windows drive D", "D:\\notes\\secret.md", true, "relative"},

		// Backslash normalization
		{"backslash traversal", "notes\\..\\..\\etc\\passwd", true, "within the vault"},

		// Empty/whitespace
		{"empty string", "", true, "required"},
		{"whitespace only", "   ", true, "required"},

		// Valid paths
		{"simple file", "notes/foo.md", false, ""},
		{"nested path", "projects/backend/decisions/auth.md", false, ""},
		{"hyphenated", "my-project/notes.md", false, ""},
		{"underscored", "my_project/notes.md", false, ""},
		{"with dots in name", "notes/v2.0.1-release.md", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeClaimPath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil (result=%q)", tt.errMsg, result)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestNormalizeClaimPath_WindowsDriveVariants(t *testing.T) {
	drives := []string{"C:", "D:", "E:", "Z:"}
	separators := []string{"/", "\\"}

	for _, drive := range drives {
		for _, sep := range separators {
			path := drive + sep + "Users" + sep + "foo"
			_, err := NormalizeClaimPath(path)
			if err == nil {
				t.Errorf("expected error for Windows drive path %q", path)
			}
		}
	}
}
