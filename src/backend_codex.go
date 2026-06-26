package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Codex JSONL event types
type codexEvent struct {
	Type string          `json:"type"`
	Item json.RawMessage `json:"item,omitempty"`
}

type codexItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// CodexBackend implements Backend for the OpenAI Codex CLI.
type CodexBackend struct{}

func (b *CodexBackend) BuildCommand(baseCmd, extraFlags, prompt string) string {
	const delimiter = "__NIGEL_PROMPT_EOF__"
	cmd := strings.TrimSpace(baseCmd)
	if cmd == "codex" {
		cmd = "codex exec"
	}

	// codex exec --json reads the prompt from stdin when using "-"
	// Heredoc avoids shell quoting issues
	if extraFlags != "" {
		return fmt.Sprintf("%s --json %s - <<'%s'\n%s\n%s",
			cmd, extraFlags, delimiter, prompt, delimiter)
	}
	return fmt.Sprintf("%s --json - <<'%s'\n%s\n%s",
		cmd, delimiter, prompt, delimiter)
}

func (b *CodexBackend) ProcessLine(line string) (string, bool) {
	var ev codexEvent
	if json.Unmarshal([]byte(line), &ev) != nil {
		return "", false
	}

	switch ev.Type {
	case "item.completed":
		var item codexItem
		if json.Unmarshal(ev.Item, &item) == nil && item.Type == "agent_message" && item.Text != "" {
			return item.Text + "\n", false
		}
	case "turn.completed":
		return "", true
	case "turn.failed":
		return "", true
	case "error":
		return "", true
	}

	return "", false
}

func (b *CodexBackend) RateLimitPhrases() []string {
	return []string{"rate_limit", "429"}
}

func (b *CodexBackend) DisplayName() string {
	return "Codex"
}
