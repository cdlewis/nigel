package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigRejectsUnknownFields(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "valid config with canonical agent fields",
			yaml: `
agent: "/path/to/codex"
agent_flags: "--model gpt-5"
verify_command: "cargo check"
success_command: "git commit -m 'Fix: $CANDIDATE'"
reset_command: "git reset --hard"
`,
			wantErr: false,
		},
		{
			name: "legacy claude_command alias",
			yaml: `
claude_command: "claude"
`,
			wantErr: false,
		},
		{
			name: "unknown field",
			yaml: `
agent: "claude"
rofnkjsnfke3: "bad"
`,
			wantErr: true,
		},
		{
			name: "typo in claude_command",
			yaml: `
cluade_command: "claude"
`,
			wantErr: true,
		},
		{
			name: "typo in verify_command",
			yaml: `
agent: "claude"
verify_comand: "cargo check"
`,
			wantErr: true,
		},
		{
			name: "minimal valid config",
			yaml: `
agent: "claude"
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.yaml), 0644); err != nil {
				t.Fatal(err)
			}

			_, err := loadConfig(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfigNormalizesAgentAliases(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	yaml := `
agent: "codex"
agent_flags: "--model gpt-5"
claude_command: "claude"
claude_flags: "--legacy"
`
	if err := os.WriteFile(configPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}
	if config.Agent != "codex" {
		t.Fatalf("Agent = %q, want codex", config.Agent)
	}
	if config.AgentFlags != "--model gpt-5" {
		t.Fatalf("AgentFlags = %q, want --model gpt-5", config.AgentFlags)
	}
}

func TestLoadConfigUsesLegacyAliasesWhenCanonicalMissing(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	yaml := `
claude_command: "claude"
claude_flags: "--legacy"
`
	if err := os.WriteFile(configPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}
	if config.Agent != "claude" {
		t.Fatalf("Agent = %q, want claude", config.Agent)
	}
	if config.AgentFlags != "--legacy" {
		t.Fatalf("AgentFlags = %q, want --legacy", config.AgentFlags)
	}
}

func TestLoadTaskRejectsUnknownFields(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "valid task with prompt",
			yaml: `
candidate_source: "cargo check 2>&1 | grep error"
prompt: "Fix this issue: $INPUT"
`,
			wantErr: false,
		},
		{
			name: "valid task with template",
			yaml: `
candidate_source: "cargo check 2>&1 | grep error"
template: "template.txt"
`,
			wantErr: false,
		},
		{
			name: "valid task with all fields",
			yaml: `
candidate_source: "cargo check 2>&1 | grep error"
prompt: "Fix this issue: $INPUT"
agent_flags: "--fast"
agent: "/custom/codex"
accept_best_effort: true
`,
			wantErr: false,
		},
		{
			name: "unknown field",
			yaml: `
candidate_source: "cargo check"
prompt: "fix it"
unknown_field: "value"
`,
			wantErr: true,
		},
		{
			name: "typo in candidate_source",
			yaml: `
candiate_source: "cargo check"
prompt: "fix it"
`,
			wantErr: true,
		},
		{
			name: "typo in accept_best_effort",
			yaml: `
candidate_source: "cargo check"
prompt: "fix it"
accept_best_effort: truee
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			taskPath := filepath.Join(tmpDir, "task.yaml")
			if err := os.WriteFile(taskPath, []byte(tt.yaml), 0644); err != nil {
				t.Fatal(err)
			}

			_, err := loadTask(taskPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadTask() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadTaskNormalizesAgentAliases(t *testing.T) {
	tmpDir := t.TempDir()
	taskPath := filepath.Join(tmpDir, "task.yaml")
	yaml := `
candidate_source: "cargo check"
prompt: "fix it"
agent: "codex"
agent_flags: "--model gpt-5"
claude_command: "claude"
claude_flags: "--legacy"
`
	if err := os.WriteFile(taskPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	task, err := loadTask(taskPath)
	if err != nil {
		t.Fatalf("loadTask failed: %v", err)
	}
	if task.Agent != "codex" {
		t.Fatalf("Agent = %q, want codex", task.Agent)
	}
	if task.AgentFlags != "--model gpt-5" {
		t.Fatalf("AgentFlags = %q, want --model gpt-5", task.AgentFlags)
	}
}

func TestLoadTaskUsesLegacyAliasesWhenCanonicalMissing(t *testing.T) {
	tmpDir := t.TempDir()
	taskPath := filepath.Join(tmpDir, "task.yaml")
	yaml := `
candidate_source: "cargo check"
prompt: "fix it"
claude_command: "claude"
claude_flags: "--legacy"
`
	if err := os.WriteFile(taskPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	task, err := loadTask(taskPath)
	if err != nil {
		t.Fatalf("loadTask failed: %v", err)
	}
	if task.Agent != "claude" {
		t.Fatalf("Agent = %q, want claude", task.Agent)
	}
	if task.AgentFlags != "--legacy" {
		t.Fatalf("AgentFlags = %q, want --legacy", task.AgentFlags)
	}
}
