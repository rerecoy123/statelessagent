package hooks

import (
	"runtime"
	"strings"
	"time"
)

// sessionsIndex represents the top-level structure of Claude Code's
// sessions-index.json file.
type sessionsIndex struct {
	Version int            `json:"version"`
	Entries []sessionEntry `json:"entries"`
}

// sessionEntry represents a single session in the sessions index.
type sessionEntry struct {
	SessionID    string `json:"sessionId"`
	FullPath     string `json:"fullPath"`
	FileMtime    int64  `json:"fileMtime"`
	FirstPrompt  string `json:"firstPrompt"`
	Summary      string `json:"summary"`
	MessageCount int    `json:"messageCount"`
	Created      string `json:"created"`
	Modified     string `json:"modified"`
	GitBranch    string `json:"gitBranch"`
	ProjectPath  string `json:"projectPath"`
	IsSidechain  bool   `json:"isSidechain"`
}

const (
	currentSessionThreshold = 60 * time.Second
	// sessionsIndexMaxBytes is the maximum size we'll read for sessions-index.json.
	// Claude Code indexes are typically <100KB; 5MB guards against anomalies.
	sessionsIndexMaxBytes = 5 * 1024 * 1024
)

// claudeProjectHash converts a directory path to Claude Code's project hash
// format. All path separators are replaced with dashes, and the result starts
// with a leading dash.
func claudeProjectHash(cwd string) string {
	result := strings.ReplaceAll(cwd, "/", "-")

	if runtime.GOOS == "windows" {
		result = strings.ReplaceAll(result, `\`, "-")
		result = strings.ReplaceAll(result, ":", "-")
	}

	if !strings.HasPrefix(result, "-") {
		result = "-" + result
	}

	return result
}
