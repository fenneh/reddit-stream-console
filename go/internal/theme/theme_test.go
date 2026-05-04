package theme_test

import (
	"strings"
	"testing"

	"github.com/fenneh/reddit-stream-console/internal/theme"
)

func TestLookupEmpty(t *testing.T) {
	th, ok := theme.Lookup("")
	if !ok {
		t.Error("empty name should return ok=true")
	}
	if th.Name != "default" {
		t.Errorf("got %q, want default", th.Name)
	}
}

func TestLookupKnown(t *testing.T) {
	for _, name := range theme.Names() {
		th, ok := theme.Lookup(name)
		if !ok {
			t.Errorf("Lookup(%q) returned ok=false", name)
		}
		if th.Name != name {
			t.Errorf("Lookup(%q).Name = %q", name, th.Name)
		}
	}
}

func TestLookupUnknown(t *testing.T) {
	th, ok := theme.Lookup("does-not-exist")
	if ok {
		t.Error("unknown name should return ok=false")
	}
	if th.Name != "default" {
		t.Errorf("unknown name should fall back to default, got %q", th.Name)
	}
}

func TestLookupCaseInsensitive(t *testing.T) {
	th, ok := theme.Lookup("DRACULA")
	if !ok {
		t.Error("lookup should be case-insensitive")
	}
	if th.Name != "dracula" {
		t.Errorf("got %q, want dracula", th.Name)
	}
}

func TestLookupTrimsSpace(t *testing.T) {
	th, ok := theme.Lookup("  nord  ")
	if !ok {
		t.Error("lookup should trim whitespace")
	}
	if th.Name != "nord" {
		t.Errorf("got %q, want nord", th.Name)
	}
}

func TestGetKnown(t *testing.T) {
	th := theme.Get("dracula")
	if th.Name != "dracula" {
		t.Errorf("got %q, want dracula", th.Name)
	}
}

func TestGetUnknownFallsBack(t *testing.T) {
	th := theme.Get("bogus")
	if th.Name != "default" {
		t.Errorf("unknown theme should fall back to default, got %q", th.Name)
	}
}

func TestNamesAreSorted(t *testing.T) {
	names := theme.Names()
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Errorf("names not sorted: %q before %q", names[i-1], names[i])
		}
	}
}

func TestNamesContainsExpected(t *testing.T) {
	expected := []string{"default", "dracula", "nord", "gruvbox-dark", "tokyo-night",
		"catppuccin-mocha", "catppuccin-macchiato", "catppuccin-frappe", "catppuccin-latte"}
	names := theme.Names()
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	for _, e := range expected {
		if !set[e] {
			t.Errorf("Names() missing %q", e)
		}
	}
}

func TestAllThemesHaveHexColors(t *testing.T) {
	for _, name := range theme.Names() {
		th := theme.Get(name)
		colors := []struct {
			role string
			hex  string
		}{
			{"HeaderBg", th.HeaderBg.Hex},
			{"HeaderFg", th.HeaderFg.Hex},
			{"Border", th.Border.Hex},
			{"Primary", th.Primary.Hex},
			{"Accent", th.Accent.Hex},
		}
		for _, c := range colors {
			if !strings.HasPrefix(c.hex, "#") || len(c.hex) != 7 {
				t.Errorf("theme %q: %s.Hex = %q, want #RRGGBB", name, c.role, c.hex)
			}
		}
	}
}

func TestDefaultIsConsistent(t *testing.T) {
	d1 := theme.Default()
	d2 := theme.Default()
	if d1.Name != d2.Name || d1.Primary.Hex != d2.Primary.Hex {
		t.Error("Default() should be deterministic")
	}
}
