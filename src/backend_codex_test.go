package main

import "strings"
import "testing"

func TestCodexBuildCommandUsesExecForBareCodex(t *testing.T) {
	cmd := (&CodexBackend{}).BuildCommand("codex", "", "hello")
	if !strings.HasPrefix(cmd, "codex exec --json --yolo - <<") {
		t.Fatalf("BuildCommand() = %q, want codex exec invocation", cmd)
	}
}

func TestCodexBuildCommandDoesNotDoubleExec(t *testing.T) {
	cmd := (&CodexBackend{}).BuildCommand("codex exec", "--sandbox read-only", "hello")
	if !strings.HasPrefix(cmd, "codex exec --json --yolo --sandbox read-only - <<") {
		t.Fatalf("BuildCommand() = %q, want existing codex exec invocation preserved", cmd)
	}
}

func TestCodexBuildCommandDoesNotDuplicateYolo(t *testing.T) {
	cmd := (&CodexBackend{}).BuildCommand("codex exec --yolo", "--sandbox read-only", "hello")
	if !strings.HasPrefix(cmd, "codex exec --yolo --json --sandbox read-only - <<") {
		t.Fatalf("BuildCommand() = %q, want existing --yolo invocation preserved", cmd)
	}

	cmd = (&CodexBackend{}).BuildCommand("codex exec", "--yolo --sandbox read-only", "hello")
	if !strings.HasPrefix(cmd, "codex exec --json --yolo --sandbox read-only - <<") {
		t.Fatalf("BuildCommand() = %q, want extraFlags --yolo invocation preserved", cmd)
	}
}

func TestCodexRateLimitTextIgnoresSuccessfulEventPayloads(t *testing.T) {
	b := &CodexBackend{}
	lines := []string{
		`{"type":"item.completed","item":{"type":"command_execution","aggregated_output":"generate_rate_limit_variants.sh"}}`,
		`{"type":"item.completed","item":{"type":"agent_message","text":"You've hit your usage limit appeared in the test fixture"}}`,
		`{"type":"error","message":"Reconnecting after HTTP 429"}`,
		`{"type":"turn.completed","usage":{"input_tokens":429}}`,
	}
	for _, line := range lines {
		if got := b.RateLimitText(line); got != "" {
			t.Errorf("RateLimitText(%q) = %q, want empty", line, got)
		}
	}
}

func TestCodexRateLimitTextIncludesTerminalFailure(t *testing.T) {
	b := &CodexBackend{}
	line := `{"type":"turn.failed","error":{"message":"You've hit your usage limit. Try again later."}}`
	got := b.RateLimitText(line)
	if !strings.Contains(got, "usage limit") {
		t.Fatalf("RateLimitText(%q) = %q, want terminal failure record", line, got)
	}
	if _, ok := findRateLimitMatch(got, b.RateLimitPhrases()); !ok {
		t.Fatalf("terminal usage-limit failure was not detected: %q", got)
	}
}

func TestCodexProcessLineOnlyStopsOnTerminalTurnEvent(t *testing.T) {
	b := &CodexBackend{}
	if _, done := b.ProcessLine(`{"type":"error","message":"Reconnecting... 1/5"}`); done {
		t.Fatal("transient error event should not end the session")
	}
	if _, done := b.ProcessLine(`{"type":"turn.failed","error":{"message":"request failed"}}`); !done {
		t.Fatal("turn.failed should end the session")
	}
}
