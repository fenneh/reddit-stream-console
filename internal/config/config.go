package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type AppConfig struct {
	DebugLogging bool `json:"debug_logging"`
}

type MenuConfig struct {
	MenuItems []MenuItem `json:"menu_items"`
}

type MenuItem struct {
	Title               string        `json:"title"`
	Type                string        `json:"type"`
	Subreddit           string        `json:"subreddit"`
	Flair               StringOrSlice `json:"flair"`
	MaxAgeHours         int           `json:"max_age_hours"`
	Limit               int           `json:"limit"`
	TitleMustContain    []string      `json:"title_must_contain"`
	TitleMustNotContain []string      `json:"title_must_not_contain"`
	Description         string        `json:"description"`
}

type StringOrSlice []string

func (s *StringOrSlice) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		*s = nil
		return nil
	}
	if data[0] == '[' {
		var v []string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*s = v
		return nil
	}
	var single string
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	*s = []string{single}
	return nil
}

func LoadMenuConfig(path string) (MenuConfig, error) {
	var cfg MenuConfig
	data, err := readConfigFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read menu config: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse menu config: %w", err)
	}
	return cfg, nil
}

func LoadAppConfig(path string) (AppConfig, error) {
	var cfg AppConfig
	data, err := readConfigFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read app config: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse app config: %w", err)
	}
	return cfg, nil
}

func readConfigFile(path string) ([]byte, error) {
	if filepath.IsAbs(path) {
		return os.ReadFile(path)
	}

	// Try current working directory first.
	if data, err := os.ReadFile(path); err == nil {
		return data, nil
	}

	exePath, err := os.Executable()
	if err != nil {
		return nil, err
	}
	exeDir := filepath.Dir(exePath)

	candidates := []string{
		filepath.Join(exeDir, path),
		filepath.Join(exeDir, "..", path),
	}

	for _, candidate := range candidates {
		if data, err := os.ReadFile(candidate); err == nil {
			return data, nil
		}
	}

	return nil, os.ErrNotExist
}
