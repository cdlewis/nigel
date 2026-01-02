package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const separator = "================================================================================"

// ClaudeLogger handles logging of Claude interactions.
type ClaudeLogger struct {
	file *os.File
}

// NewClaudeLogger creates a new logger for Claude interactions.
func NewClaudeLogger(taskDir string) (*ClaudeLogger, error) {
	path := filepath.Join(taskDir, "claude.log")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open claude log: %w", err)
	}

	return &ClaudeLogger{file: file}, nil
}

// StartEntry begins a new log entry with timestamp and prompt.
func (l *ClaudeLogger) StartEntry(prompt string) error {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	_, err := fmt.Fprintf(l.file, "\n%s\nTimestamp: %s\nPrompt: %s\n%s\n",
		separator, timestamp, prompt, separator)
	return err
}

// EndEntry closes the current log entry.
func (l *ClaudeLogger) EndEntry() error {
	_, err := fmt.Fprintf(l.file, "%s\n", separator)
	return err
}

// Write implements io.Writer for streaming Claude output to the log.
func (l *ClaudeLogger) Write(p []byte) (n int, err error) {
	return l.file.Write(p)
}

// Close closes the log file.
func (l *ClaudeLogger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Path returns the path to the log file.
func (l *ClaudeLogger) Path() string {
	return l.file.Name()
}
