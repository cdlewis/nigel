package main

import "strings"
import "testing"

func TestCodexBuildCommandUsesExecForBareCodex(t *testing.T) {
	cmd := (&CodexBackend{}).BuildCommand("codex", "", "hello")
	if !strings.HasPrefix(cmd, "codex exec --json - <<") {
		t.Fatalf("BuildCommand() = %q, want codex exec invocation", cmd)
	}
}

func TestCodexBuildCommandDoesNotDoubleExec(t *testing.T) {
	cmd := (&CodexBackend{}).BuildCommand("codex exec", "--sandbox read-only", "hello")
	if !strings.HasPrefix(cmd, "codex exec --json --sandbox read-only - <<") {
		t.Fatalf("BuildCommand() = %q, want existing codex exec invocation preserved", cmd)
	}
}
