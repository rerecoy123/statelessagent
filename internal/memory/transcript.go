package memory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Message represents a parsed conversation message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ToolCall represents a parsed tool invocation.
type ToolCall struct {
	Tool  string                 `json:"tool"`
	Input map[string]interface{} `json:"input"`
}

// TranscriptData holds parsed transcript data.
type TranscriptData struct {
	Messages     []Message  `json:"messages"`
	ToolCalls    []ToolCall `json:"tool_calls"`
	FilesChanged []string   `json:"files_changed"`
}

// ParseTranscript parses a JSONL transcript file.
func ParseTranscript(path string) TranscriptData {
	result := TranscriptData{}

	f, err := os.Open(path)
	if err != nil {
		return result
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB max line

	filesChanged := make(map[string]bool)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		processEntry(entry, &result.Messages, &result.ToolCalls, filesChanged)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "same: warning: transcript read error (partial data): %v\n", err)
	}

	for f := range filesChanged {
		result.FilesChanged = append(result.FilesChanged, f)
	}
	return result
}

func processEntry(entry map[string]interface{}, messages *[]Message, toolCalls *[]ToolCall, filesChanged map[string]bool) {
	// Claude Code wraps messages in a "message" envelope:
	//   {"type": "user", "message": {"role": "user", "content": "..."}}
	// Unwrap to get the actual message object for role and content extraction.
	role, _ := entry["role"].(string)
	if role == "" {
		if msg, ok := entry["message"].(map[string]interface{}); ok {
			role, _ = msg["role"].(string)
			entry = msg // use inner message for content extraction below
		}
	}
	// Also accept the top-level "type" field as a role fallback
	if role == "" {
		role, _ = entry["type"].(string)
	}

	switch {
	case role == "user" || role == "human":
		content := extractTextContent(entry)
		if content != "" {
			*messages = append(*messages, Message{Role: "user", Content: content})
		}

	case role == "assistant":
		content := extractTextContent(entry)
		if content != "" {
			*messages = append(*messages, Message{Role: "assistant", Content: content})
		}

		// Check for tool use in content blocks
		contentBlocks, ok := entry["content"].([]interface{})
		if !ok {
			return
		}
		for _, block := range contentBlocks {
			blockMap, ok := block.(map[string]interface{})
			if !ok {
				continue
			}
			if blockMap["type"] != "tool_use" {
				continue
			}
			toolName, _ := blockMap["name"].(string)
			toolInput, _ := blockMap["input"].(map[string]interface{})
			*toolCalls = append(*toolCalls, ToolCall{Tool: toolName, Input: toolInput})
			extractFilesFromTool(toolName, toolInput, filesChanged)
		}
	}
}

func extractTextContent(entry map[string]interface{}) string {
	content := entry["content"]
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		var parts []string
		for _, block := range v {
			switch b := block.(type) {
			case map[string]interface{}:
				if b["type"] == "text" {
					text, _ := b["text"].(string)
					parts = append(parts, text)
				}
			case string:
				parts = append(parts, b)
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

func extractFilesFromTool(toolName string, input map[string]interface{}, filesChanged map[string]bool) {
	toolLower := strings.ToLower(toolName)

	switch toolLower {
	case "write", "create", "edit", "replace":
		path := getStr(input, "file_path")
		if path == "" {
			path = getStr(input, "path")
		}
		if path != "" {
			filesChanged[path] = true
		}
	case "bash":
		cmd := getStr(input, "command")
		extractBashFilePaths(cmd, filesChanged)
	}
}

var bashFilePatterns = []*regexp.Regexp{
	regexp.MustCompile(`>\s*([^\s;|&]+)`),
	regexp.MustCompile(`>>\s*([^\s;|&]+)`),
	regexp.MustCompile(`tee\s+([^\s;|&]+)`),
	regexp.MustCompile(`mv\s+\S+\s+([^\s;|&]+)`),
	regexp.MustCompile(`cp\s+\S+\s+([^\s;|&]+)`),
}

func extractBashFilePaths(command string, filesChanged map[string]bool) {
	for _, pattern := range bashFilePatterns {
		for _, match := range pattern.FindAllStringSubmatch(command, -1) {
			if len(match) > 1 {
				path := strings.Trim(match[1], "'\"")
				if path != "" && !strings.HasPrefix(path, "-") {
					filesChanged[path] = true
				}
			}
		}
	}
}

// GetLastNMessages returns the last N messages from a transcript.
func GetLastNMessages(path string, n int, role string) []Message {
	parsed := ParseTranscript(path)
	messages := parsed.Messages

	if role != "" {
		var filtered []Message
		for _, m := range messages {
			if m.Role == role {
				filtered = append(filtered, m)
			}
		}
		messages = filtered
	}

	if len(messages) > n {
		messages = messages[len(messages)-n:]
	}
	return messages
}

// GetSessionSummaryInputs extracts key inputs needed for handoff generation.
func GetSessionSummaryInputs(path string) map[string]interface{} {
	parsed := ParseTranscript(path)

	var userMsgs, assistantMsgs []string
	for _, m := range parsed.Messages {
		switch m.Role {
		case "user":
			userMsgs = append(userMsgs, m.Content)
		case "assistant":
			assistantMsgs = append(assistantMsgs, m.Content)
		}
	}

	return map[string]interface{}{
		"user_messages":      userMsgs,
		"assistant_messages": assistantMsgs,
		"files_changed":      parsed.FilesChanged,
		"tool_calls":         parsed.ToolCalls,
		"message_count":      len(parsed.Messages),
	}
}

func getStr(m map[string]interface{}, key string) string {
	v, _ := m[key].(string)
	return v
}
