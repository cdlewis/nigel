package main

import (
	"encoding/json"
	"fmt"
)

// Claude stream event types
type streamEvent struct {
	Type  string                 `json:"type"`
	Event map[string]interface{} `json:"event,omitempty"`
}

// contentBlockDelta represents the delta content in a Claude stream event
type contentBlockDelta struct {
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
}

// resultEvent represents the final result event from Claude
type resultEvent struct {
	Type   string `json:"type"`
	Result string `json:"result,omitempty"`
}

// ClaudeBackend implements Backend for the Claude CLI.
type ClaudeBackend struct {
	messageHasContent bool
}

func (b *ClaudeBackend) BuildCommand(baseCmd, extraFlags string) string {
	jsonFlags := "--print --output-format stream-json --include-partial-messages --verbose"

	// "-p" with no argument makes claude read the prompt from stdin, which
	// RunAICommand supplies directly - the prompt never passes through the shell.
	if extraFlags != "" {
		return fmt.Sprintf("%s %s %s -p", baseCmd, jsonFlags, extraFlags)
	}
	return fmt.Sprintf("%s %s -p", baseCmd, jsonFlags)
}

func (b *ClaudeBackend) ProcessLine(line string) (string, bool) {
	var se streamEvent
	if json.Unmarshal([]byte(line), &se) != nil {
		return "", false
	}

	switch se.Type {
	case "stream_event":
		if eventType, ok := se.Event["type"].(string); ok {
			if eventType == "content_block_delta" {
				eventJSON, _ := json.Marshal(se.Event)
				var delta contentBlockDelta
				if json.Unmarshal(eventJSON, &delta) == nil && delta.Delta.Type == "text_delta" && delta.Delta.Text != "" {
					b.messageHasContent = true
					return delta.Delta.Text, false
				}
			}
			if eventType == "message_stop" {
				if b.messageHasContent {
					b.messageHasContent = false
					return "\n", false
				}
				b.messageHasContent = false
			}
		}
	case "result":
		return "", true
	}

	return "", false
}

func (b *ClaudeBackend) RateLimitPhrases() []string {
	return []string{"You've hit your limit"}
}

func (b *ClaudeBackend) DisplayName() string {
	return "Claude"
}
