package hooks

import (
	"fmt"
	"os"

	"github.com/sgxdev/same/internal/memory"
	"github.com/sgxdev/same/internal/store"
)

// runHandoffGenerator generates a handoff note from the transcript.
func runHandoffGenerator(_ *store.DB, input *HookInput) *HookOutput {
	transcriptPath := input.TranscriptPath
	if transcriptPath == "" {
		return nil
	}
	if _, err := os.Stat(transcriptPath); err != nil {
		return nil
	}

	hookEvent := input.HookEventName
	if hookEvent == "" {
		hookEvent = "Stop"
	}

	result := memory.AutoHandoffFromTranscript(transcriptPath, input.SessionID)
	if result == nil {
		return nil
	}

	return &HookOutput{
		HookSpecificOutput: &HookSpecific{
			HookEventName: hookEvent,
			AdditionalContext: fmt.Sprintf(
				"\n<vault-handoff>\nSession handoff written to: %s\nSession ID: %s\n</vault-handoff>\n",
				result.Path, result.SessionID,
			),
		},
	}
}
