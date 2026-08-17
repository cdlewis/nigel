package main

import "testing"

func TestClaudeProcessLineSurfacesErrorAssistantMessage(t *testing.T) {
	b := &ClaudeBackend{}
	line := `{"type":"assistant","message":{"content":[{"type":"text","text":"Failed to authenticate. API Error: 401 OAuth access token has expired."}]},"error":"authentication_failed"}`

	text, done := b.ProcessLine(line)
	if text != "Failed to authenticate. API Error: 401 OAuth access token has expired.\n" {
		t.Fatalf("ProcessLine() text = %q", text)
	}
	if done {
		t.Fatal("ProcessLine() unexpectedly marked session done")
	}
}

func TestClaudeProcessLineIgnoresOrdinaryAssistantMessage(t *testing.T) {
	b := &ClaudeBackend{}
	line := `{"type":"assistant","message":{"content":[{"type":"text","text":"already streamed"}]}}`

	text, done := b.ProcessLine(line)
	if text != "" || done {
		t.Fatalf("ProcessLine() = (%q, %v), want (\"\", false)", text, done)
	}
}

func TestClaudeProcessLineSurfacesErrorResult(t *testing.T) {
	b := &ClaudeBackend{}
	line := `{"type":"result","subtype":"error_during_execution","is_error":true,"result":"Permission denied"}`

	text, done := b.ProcessLine(line)
	if text != "Permission denied\n" {
		t.Fatalf("ProcessLine() text = %q", text)
	}
	if !done {
		t.Fatal("ProcessLine() did not mark result as done")
	}
}

func TestClaudeProcessLineShowsToolDescription(t *testing.T) {
	b := &ClaudeBackend{}
	line := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"secret command","description":"Run statement-order sweep"}}]}}`

	text, done := b.ProcessLine(line)
	if text != "[Claude: Run statement-order sweep]\n" {
		t.Fatalf("ProcessLine() text = %q", text)
	}
	if done {
		t.Fatal("ProcessLine() unexpectedly marked session done")
	}
}

func TestClaudeProcessLineShowsSparseToolHeartbeats(t *testing.T) {
	b := &ClaudeBackend{}

	text, _ := b.ProcessLine(`{"type":"tool_progress","tool_name":"Bash","elapsed_time_seconds":30,"heartbeat":true}`)
	if text != "[Claude: Bash still running (30s)]\n" {
		t.Fatalf("30-second heartbeat text = %q", text)
	}

	text, _ = b.ProcessLine(`{"type":"tool_progress","tool_name":"Bash","elapsed_time_seconds":60,"heartbeat":true}`)
	if text != "" {
		t.Fatalf("60-second heartbeat text = %q, want suppressed", text)
	}

	text, _ = b.ProcessLine(`{"type":"tool_progress","tool_name":"Bash","elapsed_time_seconds":300,"heartbeat":true}`)
	if text != "[Claude: Bash still running (5m0s)]\n" {
		t.Fatalf("five-minute heartbeat text = %q", text)
	}
}

func TestClaudeProcessLineShowsBackgroundTaskNotification(t *testing.T) {
	b := &ClaudeBackend{}
	line := `{"type":"system","subtype":"task_notification","status":"failed","summary":"Background command \"Sweep declarations\" failed with exit code 1"}`

	text, done := b.ProcessLine(line)
	if text != "[Claude: Background command \"Sweep declarations\" failed with exit code 1]\n" {
		t.Fatalf("ProcessLine() text = %q", text)
	}
	if done {
		t.Fatal("ProcessLine() unexpectedly marked session done")
	}
}
