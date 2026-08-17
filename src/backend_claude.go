package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Claude stream event types
type streamEvent struct {
	Type               string                 `json:"type"`
	Subtype            string                 `json:"subtype,omitempty"`
	Error              string                 `json:"error,omitempty"`
	Event              map[string]interface{} `json:"event,omitempty"`
	Message            *claudeMessage         `json:"message,omitempty"`
	Summary            string                 `json:"summary,omitempty"`
	ToolName           string                 `json:"tool_name,omitempty"`
	ElapsedTimeSeconds int                    `json:"elapsed_time_seconds,omitempty"`
}

type claudeMessage struct {
	Content []claudeContent `json:"content,omitempty"`
}

type claudeContent struct {
	Type  string                 `json:"type"`
	Text  string                 `json:"text,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
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
	Type    string `json:"type"`
	Result  string `json:"result,omitempty"`
	IsError bool   `json:"is_error,omitempty"`
}

// ClaudeBackend implements Backend for the Claude CLI.
type ClaudeBackend struct {
	messageHasContent bool
}

func (b *ClaudeBackend) BuildCommand(baseCmd, extraFlags, prompt string) string {
	const delimiter = "__NIGEL_PROMPT_EOF__"
	jsonFlags := "--print --output-format stream-json --include-partial-messages --verbose"

	if extraFlags != "" {
		return fmt.Sprintf("%s %s %s -p <<'%s'\n%s\n%s",
			baseCmd, jsonFlags, extraFlags, delimiter, prompt, delimiter)
	}
	return fmt.Sprintf("%s %s -p <<'%s'\n%s\n%s",
		baseCmd, jsonFlags, delimiter, prompt, delimiter)
}

func (b *ClaudeBackend) ProcessLine(line string) (string, bool) {
	var se streamEvent
	if json.Unmarshal([]byte(line), &se) != nil {
		return "", false
	}

	switch se.Type {
	case "assistant":
		// Claude reports some startup/API failures as a synthetic assistant
		// message on stdout, then exits non-zero without a result event or
		// anything on stderr. Only surface error-marked assistant messages:
		// ordinary assistant records duplicate the stream_event deltas.
		if se.Error != "" && se.Message != nil {
			var text string
			for _, content := range se.Message.Content {
				if content.Type == "text" {
					text += content.Text
				}
			}
			if text != "" {
				return text + "\n", false
			}
		}

		// Complete tool-use inputs are emitted in top-level assistant records,
		// while stream_event contains their arguments as partial JSON deltas.
		// Show the human-readable description so tool-heavy sessions do not
		// look hung, without echoing commands or large tool inputs.
		if se.Message != nil {
			var progress []string
			for _, content := range se.Message.Content {
				if content.Type != "tool_use" {
					continue
				}
				description, _ := content.Input["description"].(string)
				description = strings.Join(strings.Fields(description), " ")
				if description == "" {
					description = content.Name
				}
				if description != "" {
					progress = append(progress, description)
				}
			}
			if len(progress) > 0 {
				return b.progressLine(strings.Join(progress, "; ")), false
			}
		}
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
		var result resultEvent
		if json.Unmarshal([]byte(line), &result) == nil && result.IsError && result.Result != "" {
			return result.Result + "\n", true
		}
		return "", true
	case "tool_progress":
		// Claude currently emits heartbeats about every 30 seconds. Print the
		// first one and then every five minutes to confirm liveness without
		// flooding long-running campaigns.
		if se.ElapsedTimeSeconds == 30 || (se.ElapsedTimeSeconds > 0 && se.ElapsedTimeSeconds%300 == 0) {
			toolName := se.ToolName
			if toolName == "" {
				toolName = "tool"
			}
			return b.progressLine(fmt.Sprintf("%s still running (%s)", toolName, (time.Duration(se.ElapsedTimeSeconds) * time.Second).String())), false
		}
	case "system":
		if se.Subtype == "task_notification" && se.Summary != "" {
			return b.progressLine(strings.Join(strings.Fields(se.Summary), " ")), false
		}
	}

	return "", false
}

func (b *ClaudeBackend) progressLine(message string) string {
	prefix := ""
	if b.messageHasContent {
		prefix = "\n"
		b.messageHasContent = false
	}
	return prefix + "[Claude: " + message + "]\n"
}

func (b *ClaudeBackend) RateLimitPhrases() []string {
	return []string{"You've hit your limit"}
}

func (b *ClaudeBackend) DisplayName() string {
	return "Claude"
}
