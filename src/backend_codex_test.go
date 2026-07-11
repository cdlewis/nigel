package main

import "strings"
import "testing"

func TestCodexBuildCommandUsesExecForBareCodex(t *testing.T) {
	cmd := (&CodexBackend{}).BuildCommand("codex", "")
	if cmd != "codex exec --json -" {
		t.Fatalf("BuildCommand() = %q, want codex exec invocation", cmd)
	}
}

func TestCodexBuildCommandDoesNotDoubleExec(t *testing.T) {
	cmd := (&CodexBackend{}).BuildCommand("codex exec", "--sandbox read-only")
	if cmd != "codex exec --json --sandbox read-only -" {
		t.Fatalf("BuildCommand() = %q, want existing codex exec invocation preserved", cmd)
	}
}

func TestCodexBuildCommandExcludesPromptFromShellString(t *testing.T) {
	// The prompt must never be embedded in the returned shell string - it is
	// sent via stdin instead, so untrusted candidate content can't be parsed
	// by the shell (see the heredoc-injection issue this replaces).
	cmd := (&CodexBackend{}).BuildCommand("codex", "")
	if strings.Contains(cmd, "<<") {
		t.Fatalf("BuildCommand() = %q, should not contain a heredoc", cmd)
	}
}
