package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexModelDisplay(t *testing.T) {
	for _, test := range []struct {
		name, config, want string
	}{
		{"current model", "model = 'gpt-6.1-sol'\nmodel_reasoning_effort = 'medium'", "Sol 6.1 Medium"},
		{"comments and nested settings", "model = 'gpt-6-astra' # selected\nmodel_reasoning_effort = 'high'\n[profiles.other]\nmodel = 'gpt-6-luna'", "Astra 6 High"},
		{"selected profile", "model = 'gpt-6.1-sol'\nmodel_reasoning_effort = 'medium'\nprofile = 'work'\n[profiles.work]\nmodel = 'gpt-6-terra'\nmodel_reasoning_effort = 'xhigh'", "Terra 6 Xhigh"},
		{"profile inherits effort", "model = 'gpt-6.1-sol'\nmodel_reasoning_effort = 'medium'\nprofile = 'work'\n[profiles.work]\nmodel = 'gpt-6-luna'", "Luna 6 Medium"},
		{"unknown model", "model = 'custom-model'\nmodel_reasoning_effort = 'low'", "custom-model Low"},
		{"no effort", "model = 'gpt-6.1-sol'", "Sol 6.1"},
		{"no model", "model_reasoning_effort = 'medium'", ""},
		{"invalid config", "model = [", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("CODEX_HOME", dir)
			if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(test.config), 0600); err != nil {
				t.Fatal(err)
			}
			if got := codexModelDisplay(); got != test.want {
				t.Errorf("codexModelDisplay() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCodexModelDisplayUsesHomeDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "")
	if got := codexModelDisplay(); got != "" {
		t.Fatalf("missing config display = %q, want empty", got)
	}
	dir := filepath.Join(home, ".codex")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("model = 'gpt-6.1-sol'\nmodel_reasoning_effort = 'medium'"), 0600); err != nil {
		t.Fatal(err)
	}
	banner := StartupBanner("test", "", "standard", codexModelDisplay())
	if !strings.Contains(banner, "Mode: standard\n") || !strings.Contains(banner, "Model: Sol 6.1 Medium\n") {
		t.Fatalf("banner missing mode or model: %s", banner)
	}
	if strings.Index(banner, "Model:") < strings.Index(banner, "Mode:") {
		t.Fatal("model should appear below mode")
	}
}
