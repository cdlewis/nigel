package main

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
)

// codexModelDisplay reads the user's configured model for the startup banner.
// Missing or invalid configuration should never prevent a task from running.
func codexModelDisplay() string {
	configDir := os.Getenv("CODEX_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		configDir = filepath.Join(home, ".codex")
	}

	type modelSettings struct {
		Model  string `toml:"model"`
		Effort string `toml:"model_reasoning_effort"`
	}
	var config struct {
		Model    string                   `toml:"model"`
		Effort   string                   `toml:"model_reasoning_effort"`
		Profile  string                   `toml:"profile"`
		Profiles map[string]modelSettings `toml:"profiles"`
	}
	if _, err := toml.DecodeFile(filepath.Join(configDir, "config.toml"), &config); err != nil {
		return ""
	}
	if profile, ok := config.Profiles[config.Profile]; ok {
		if profile.Model != "" {
			config.Model = profile.Model
		}
		if profile.Effort != "" {
			config.Effort = profile.Effort
		}
	}
	if config.Model == "" {
		return ""
	}

	model := config.Model
	parts := strings.Split(model, "-")
	if len(parts) == 3 && parts[0] == "gpt" {
		switch parts[2] {
		case "sol", "astra", "terra", "luna":
			model = capitalize(parts[2]) + " " + parts[1]
		}
	}
	if config.Effort != "" {
		model += " " + capitalize(config.Effort)
	}
	return model
}

func capitalize(value string) string {
	runes := []rune(value)
	if len(runes) > 0 {
		runes[0] = unicode.ToUpper(runes[0])
	}
	return string(runes)
}
