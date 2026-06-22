package main

import "strings"

// Backend abstracts an AI command backend (Claude, Codex, etc.).
type Backend interface {
	// BuildCommand constructs the shell command string to execute.
	BuildCommand(baseCmd, extraFlags, prompt string) string
	// ProcessLine parses one line of JSON output from the backend.
	// Returns text to stream to the terminal/log, and whether the session is complete.
	ProcessLine(line string) (streamText string, sessionDone bool)
	// RateLimitPhrases returns substrings that indicate rate limiting.
	RateLimitPhrases() []string
	// DisplayName returns the backend name for UI messages.
	DisplayName() string
}

// NewBackend auto-detects the backend from the command name.
// If baseCmd starts with "codex", returns the Codex backend; otherwise Claude.
func NewBackend(baseCmd string) Backend {
	cmd := strings.Fields(baseCmd)
	if len(cmd) > 0 && cmd[0] == "codex" {
		return &CodexBackend{}
	}
	return &ClaudeBackend{}
}
