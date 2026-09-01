package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateConfig(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)

	if err := createConfig(); err != nil {
		t.Fatalf("createConfig() error = %v", err)
	}

	configPath := filepath.Join(configRoot, "kilo-agent", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if config.Name != "Kilo Agent" {
		t.Errorf("config name = %q, want %q", config.Name, "Kilo Agent")
	}
	if config.TelegramBotToken != "" {
		t.Errorf("Telegram bot token = %q, want an empty value", config.TelegramBotToken)
	}

	dirInfo, err := os.Stat(filepath.Dir(configPath))
	if err != nil {
		t.Fatalf("stat config directory: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0700 {
		t.Errorf("config directory permissions = %o, want 700", got)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Errorf("config permissions = %o, want 600", got)
	}
}

func TestCreateConfigPreservesExistingFile(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)

	configDir := filepath.Join(configRoot, "kilo-agent")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("create config directory: %v", err)
	}

	configPath := filepath.Join(configDir, "config.json")
	want := []byte(`{"name":"Existing configuration"}`)
	if err := os.WriteFile(configPath, want, 0600); err != nil {
		t.Fatalf("write existing config: %v", err)
	}
	if err := createConfig(); err != nil {
		t.Fatalf("createConfig() error = %v", err)
	}

	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read existing config: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("existing config was changed: got %q, want %q", got, want)
	}
}

func TestLoadConfig(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)

	configDir := filepath.Join(configRoot, "kilo-agent")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("create config directory: %v", err)
	}

	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{
		"name": "Configured Agent",
		"telegram_bot_token": "test-token",
		"telegram_chat_id": 765046979
	}`), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if config.Name != "Configured Agent" {
		t.Errorf("config name = %q, want %q", config.Name, "Configured Agent")
	}
	if config.TelegramBotToken != "test-token" {
		t.Errorf("Telegram bot token = %q, want %q", config.TelegramBotToken, "test-token")
	}
	if config.TelegramChatID != 765046979 {
		t.Errorf("Telegram chat ID = %d, want 765046979", config.TelegramChatID)
	}
}

func TestSaveConfig(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)

	want := Config{
		Name:             "Configured Agent",
		TelegramBotToken: "test-token",
		TelegramChatID:   765046979,
	}
	if err := saveConfig(want); err != nil {
		t.Fatalf("saveConfig() error = %v", err)
	}

	got, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if got != want {
		t.Errorf("loaded config = %#v, want %#v", got, want)
	}

	configPath, err := getConfigPath()
	if err != nil {
		t.Fatalf("getConfigPath() error = %v", err)
	}
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Errorf("config permissions = %o, want 600", got)
	}
}

func TestReadSystemPromptUsesConfigOverride(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)

	configDir := filepath.Join(configRoot, "kilo-agent")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("create config directory: %v", err)
	}

	want := "You are the configured system prompt.\n"
	promptPath := filepath.Join(configDir, "SYSTEM.md")
	if err := os.WriteFile(promptPath, []byte(want), 0644); err != nil {
		t.Fatalf("write system prompt: %v", err)
	}

	got, err := readSystemPrompt()
	if err != nil {
		t.Fatalf("readSystemPrompt() error = %v", err)
	}
	if got != want {
		t.Errorf("system prompt = %q, want %q", got, want)
	}
}

func TestReadSystemPromptUsesEmbeddedDefault(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)

	got, err := readSystemPrompt()
	if err != nil {
		t.Fatalf("readSystemPrompt() error = %v", err)
	}
	if got != defaultSystemPrompt {
		t.Error("system prompt does not match the embedded default")
	}
	if got == "" {
		t.Error("embedded system prompt is empty")
	}
}
