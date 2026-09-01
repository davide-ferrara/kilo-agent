package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed SYSTEM.md
var defaultSystemPrompt string

type Config struct {
	Name             string `json:"name"`
	TelegramBotToken string `json:"telegram_bot_token"`
	TelegramChatID   int64  `json:"telegram_chat_id,omitempty"`
}

func getConfigPath() (string, error) {
	configRoot, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(configRoot, "kilo-agent")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}

	return filepath.Join(dir, "config.json"), nil
}

func createConfig() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	fp, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if errors.Is(err, fs.ErrExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer fp.Close()

	config := Config{
		Name:             "Kilo Agent",
		TelegramBotToken: "",
	}

	return json.NewEncoder(fp).Encode(config)
}

func loadConfig() (Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return Config{}, err
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, err
	}

	var config Config
	if err := json.Unmarshal(raw, &config); err != nil {
		return Config{}, err
	}
	return config, nil
}

func saveConfig(config Config) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	temporary, err := os.CreateTemp(filepath.Dir(configPath), ".config-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(0600); err != nil {
		temporary.Close()
		return err
	}
	if err := json.NewEncoder(temporary).Encode(config); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}

	return os.Rename(temporaryPath, configPath)
}

func readSystemPrompt() (string, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return "", err
	}

	systemPromptPath := filepath.Join(filepath.Dir(configPath), "SYSTEM.md")
	data, err := os.ReadFile(systemPromptPath)
	if errors.Is(err, fs.ErrNotExist) {
		return defaultSystemPrompt, nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}
