package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

// DefaultMenuConfig returns the built-in menu configuration used when no config file is found.
func DefaultMenuConfig() MenuConfig {
	return MenuConfig{
		MenuItems: []MenuItem{
			{
				Title:               "/r/soccer match-threads",
				Type:                "soccer_match",
				Subreddit:           "soccer",
				Flair:               []string{"Match Thread", "match thread"},
				MaxAgeHours:         6,
				Limit:               50,
				TitleMustContain:    []string{"Match Thread"},
				TitleMustNotContain: []string{"Post Match Thread", "Post-Match Thread"},
			},
			{
				Title:            "/r/soccer post-match-threads",
				Type:             "soccer_post_match",
				Subreddit:        "soccer",
				Flair:            []string{"Post Match Thread", "post match thread"},
				MaxAgeHours:      12,
				Limit:            50,
				TitleMustContain: []string{"Post Match Thread"},
			},
			{
				Title:            "/r/fantasypl",
				Type:             "fpl_rant",
				Subreddit:        "FantasyPL",
				Flair:            []string{"GW Rant & Info", "gw rant & info"},
				MaxAgeHours:      168,
				Limit:            50,
				TitleMustContain: []string{"Rant"},
			},
			{
				Title:               "/r/nfl game-threads",
				Type:                "nfl_game",
				Subreddit:           "nfl",
				Flair:               []string{"Game Thread", "game thread"},
				MaxAgeHours:         12,
				Limit:               100,
				TitleMustContain:    []string{"Game Thread"},
				TitleMustNotContain: []string{"Post Game Thread", "Post-Game Thread"},
			},
			{
				Title:            "/r/nfl post-game-threads",
				Type:             "nfl_post_game",
				Subreddit:        "nfl",
				Flair:            []string{"Game Thread", "game thread"},
				MaxAgeHours:      12,
				Limit:            100,
				TitleMustContain: []string{"Post Game Thread"},
			},
			{
				Type:  "separator",
				Title: " ",
			},
			{
				Title:       "Enter Reddit URL",
				Type:        "url_input",
				Description: "View any Reddit thread by URL",
			},
		},
	}
}

// LoadMenuConfig loads menu configuration from file, or returns defaults if not found.
func LoadMenuConfig(path string) (MenuConfig, error) {
	data, err := readConfigFile(path)
	if err != nil {
		// Config file not found - use defaults
		return DefaultMenuConfig(), nil
	}
	var cfg MenuConfig
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

// configSearchPaths returns the list of directories to search for config files.
// Order: home dir, next to exe, 1 up from exe, 2 up from exe
func configSearchPaths() []string {
	var paths []string

	// Home directory: ~/.reddit-stream-console/
	if home := getHomeDir(); home != "" {
		paths = append(paths, filepath.Join(home, ".reddit-stream-console"))
	}

	// Relative to executable
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		paths = append(paths,
			exeDir,                            // next to exe
			filepath.Join(exeDir, ".."),       // 1 up from exe
			filepath.Join(exeDir, "..", ".."), // 2 up from exe
		)
	}

	return paths
}

func getHomeDir() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("USERPROFILE")
	}
	return os.Getenv("HOME")
}

func readConfigFile(path string) ([]byte, error) {
	if filepath.IsAbs(path) {
		return os.ReadFile(path)
	}

	// Search through all candidate directories
	for _, dir := range configSearchPaths() {
		candidate := filepath.Join(dir, path)
		if data, err := os.ReadFile(candidate); err == nil {
			return data, nil
		}
	}

	return nil, os.ErrNotExist
}
