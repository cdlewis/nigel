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
