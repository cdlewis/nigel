package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAgentLogPathDefaultsToAgentLog(t *testing.T) {
	taskDir := t.TempDir()

	got := AgentLogPath(taskDir)
	want := filepath.Join(taskDir, "agent.log")
	if got != want {
		t.Fatalf("AgentLogPath() = %q, want %q", got, want)
	}
}

func TestAgentLogPathReusesExistingClaudeLog(t *testing.T) {
	taskDir := t.TempDir()
	legacyPath := filepath.Join(taskDir, "claude.log")
	if err := os.WriteFile(legacyPath, []byte("legacy"), 0644); err != nil {
		t.Fatal(err)
	}

	got := AgentLogPath(taskDir)
	if got != legacyPath {
		t.Fatalf("AgentLogPath() = %q, want %q", got, legacyPath)
	}
}

func TestNewAgentLoggerCreatesAgentLog(t *testing.T) {
	taskDir := t.TempDir()

	logger, err := NewAgentLogger(taskDir)
	if err != nil {
		t.Fatalf("NewAgentLogger failed: %v", err)
	}
	defer logger.Close()

	want := filepath.Join(taskDir, "agent.log")
	if logger.Path() != want {
		t.Fatalf("logger.Path() = %q, want %q", logger.Path(), want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("agent log was not created: %v", err)
	}
}
