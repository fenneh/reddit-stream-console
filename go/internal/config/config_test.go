package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/fenneh/reddit-stream-console/internal/config"
)

func TestStringOrSliceArray(t *testing.T) {
	var s config.StringOrSlice
	if err := json.Unmarshal([]byte(`["a","b","c"]`), &s); err != nil {
		t.Fatal(err)
	}
	if len(s) != 3 || s[0] != "a" || s[2] != "c" {
		t.Errorf("got %v", []string(s))
	}
}

func TestStringOrSliceSingle(t *testing.T) {
	var s config.StringOrSlice
	if err := json.Unmarshal([]byte(`"hello"`), &s); err != nil {
		t.Fatal(err)
	}
	if len(s) != 1 || s[0] != "hello" {
		t.Errorf("got %v", []string(s))
	}
}

func TestStringOrSliceNull(t *testing.T) {
	var s config.StringOrSlice
	if err := json.Unmarshal([]byte(`null`), &s); err != nil {
		t.Fatal(err)
	}
	if s != nil {
		t.Errorf("expected nil, got %v", []string(s))
	}
}

func TestStringOrSliceEmpty(t *testing.T) {
	var s config.StringOrSlice
	if err := json.Unmarshal([]byte(`""`), &s); err != nil {
		t.Fatal(err)
	}
	if len(s) != 1 || s[0] != "" {
		t.Errorf("got %v", []string(s))
	}
}

func TestDefaultMenuConfigHasItems(t *testing.T) {
	cfg := config.DefaultMenuConfig()
	if len(cfg.MenuItems) == 0 {
		t.Fatal("DefaultMenuConfig returned empty items")
	}
}

func TestDefaultMenuConfigHasSeparatorAndURLInput(t *testing.T) {
	cfg := config.DefaultMenuConfig()
	var hasSep, hasURL bool
	for _, item := range cfg.MenuItems {
		switch item.Type {
		case "separator":
			hasSep = true
		case "url_input":
			hasURL = true
		}
	}
	if !hasSep {
		t.Error("DefaultMenuConfig missing separator item")
	}
	if !hasURL {
		t.Error("DefaultMenuConfig missing url_input item")
	}
}

func TestLoadMenuConfigNoFile(t *testing.T) {
	cfg, err := config.LoadMenuConfig("/nonexistent/path/menu_config.json")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if len(cfg.MenuItems) == 0 {
		t.Error("expected default items when file missing")
	}
}

func TestLoadMenuConfigInvalidJSON(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "menu*.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`{not valid json`); err != nil {
		t.Fatal(err)
	}
	f.Close()

	_, err = config.LoadMenuConfig(f.Name())
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadMenuConfigValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "menu_config.json")
	content := `{"menu_items":[{"title":"Test","type":"url_input"}]}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadMenuConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.MenuItems) != 1 || cfg.MenuItems[0].Title != "Test" {
		t.Errorf("got %+v", cfg.MenuItems)
	}
}

func TestLoadAppConfigValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app_config.json")
	content := `{"debug_logging":true,"theme":"dracula"}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadAppConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.DebugLogging {
		t.Error("expected debug_logging=true")
	}
	if cfg.Theme != "dracula" {
		t.Errorf("got theme %q, want dracula", cfg.Theme)
	}
}

func TestLoadAppConfigMissingFile(t *testing.T) {
	_, err := config.LoadAppConfig("/nonexistent/app_config.json")
	if err == nil {
		t.Error("expected error for missing app config file")
	}
}
