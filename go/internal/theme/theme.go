// Package theme defines colour palettes for the TUI.
//
// Each Theme maps semantic UI roles (Border, Accent, Primary, ...) to a Colour
// that exposes both a tcell.Color (for tview's Set*Color APIs) and a hex
// string (for inline tview markup like "[#RRGGBB]...[-]").
package theme

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
)

type Color struct {
	TCell tcell.Color
	Hex   string
}

type Theme struct {
	Name string

	Border         Color
	InactiveBorder Color

	Primary   Color
	Accent    Color
	Secondary Color
	Muted     Color
	Subtle    Color

	InputBg     Color
	Placeholder Color
}

func hex(s string) Color {
	var r, g, b int32
	if _, err := fmt.Sscanf(strings.TrimPrefix(s, "#"), "%02x%02x%02x", &r, &g, &b); err != nil {
		return Color{TCell: tcell.ColorDefault, Hex: "#000000"}
	}
	return Color{
		TCell: tcell.NewRGBColor(r, g, b),
		Hex:   fmt.Sprintf("#%02X%02X%02X", r, g, b),
	}
}

var themes = map[string]Theme{
	"default":              defaultTheme(),
	"catppuccin-mocha":     catppuccinMocha(),
	"catppuccin-macchiato": catppuccinMacchiato(),
	"catppuccin-frappe":    catppuccinFrappe(),
	"catppuccin-latte":     catppuccinLatte(),
	"dracula":              dracula(),
	"nord":                 nord(),
	"gruvbox-dark":         gruvboxDark(),
	"tokyo-night":          tokyoNight(),
}

// Get returns the named theme, or Default() if name is empty or unknown.
func Get(name string) Theme {
	if name == "" {
		return Default()
	}
	if t, ok := themes[strings.ToLower(strings.TrimSpace(name))]; ok {
		return t
	}
	return Default()
}

func Default() Theme { return themes["default"] }

// Names returns the sorted list of built-in theme names.
func Names() []string {
	out := make([]string, 0, len(themes))
	for name := range themes {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func defaultTheme() Theme {
	return Theme{
		Name:           "default",
		Border:         hex("#659287"),
		InactiveBorder: hex("#505050"),
		Primary:        hex("#FFE6A9"),
		Accent:         hex("#DEAA79"),
		Secondary:      hex("#B1C29E"),
		Muted:          hex("#888888"),
		Subtle:         hex("#666666"),
		InputBg:        hex("#282828"),
		Placeholder:    hex("#646464"),
	}
}

func catppuccinMocha() Theme {
	return Theme{
		Name:           "catppuccin-mocha",
		Border:         hex("#cba6f7"), // Mauve
		InactiveBorder: hex("#313244"), // Surface0
		Primary:        hex("#cdd6f4"), // Text
		Accent:         hex("#fab387"), // Peach
		Secondary:      hex("#a6e3a1"), // Green
		Muted:          hex("#a6adc8"), // Subtext0
		Subtle:         hex("#6c7086"), // Overlay0
		InputBg:        hex("#313244"), // Surface0
		Placeholder:    hex("#6c7086"), // Overlay0
	}
}

func catppuccinMacchiato() Theme {
	return Theme{
		Name:           "catppuccin-macchiato",
		Border:         hex("#c6a0f6"),
		InactiveBorder: hex("#363a4f"),
		Primary:        hex("#cad3f5"),
		Accent:         hex("#f5a97f"),
		Secondary:      hex("#a6da95"),
		Muted:          hex("#a5adcb"),
		Subtle:         hex("#6e738d"),
		InputBg:        hex("#363a4f"),
		Placeholder:    hex("#6e738d"),
	}
}

func catppuccinFrappe() Theme {
	return Theme{
		Name:           "catppuccin-frappe",
		Border:         hex("#ca9ee6"),
		InactiveBorder: hex("#414559"),
		Primary:        hex("#c6d0f5"),
		Accent:         hex("#ef9f76"),
		Secondary:      hex("#a6d189"),
		Muted:          hex("#a5adce"),
		Subtle:         hex("#737994"),
		InputBg:        hex("#414559"),
		Placeholder:    hex("#737994"),
	}
}

func catppuccinLatte() Theme {
	return Theme{
		Name:           "catppuccin-latte",
		Border:         hex("#8839ef"),
		InactiveBorder: hex("#ccd0da"),
		Primary:        hex("#4c4f69"),
		Accent:         hex("#fe640b"),
		Secondary:      hex("#40a02b"),
		Muted:          hex("#6c6f85"),
		Subtle:         hex("#9ca0b0"),
		InputBg:        hex("#ccd0da"),
		Placeholder:    hex("#9ca0b0"),
	}
}

func dracula() Theme {
	return Theme{
		Name:           "dracula",
		Border:         hex("#bd93f9"), // Purple
		InactiveBorder: hex("#44475a"), // Current Line
		Primary:        hex("#f8f8f2"), // Foreground
		Accent:         hex("#ffb86c"), // Orange
		Secondary:      hex("#50fa7b"), // Green
		Muted:          hex("#6272a4"), // Comment
		Subtle:         hex("#44475a"), // Current Line
		InputBg:        hex("#44475a"),
		Placeholder:    hex("#6272a4"),
	}
}

func nord() Theme {
	return Theme{
		Name:           "nord",
		Border:         hex("#88c0d0"), // Frost
		InactiveBorder: hex("#3b4252"), // Polar Night 1
		Primary:        hex("#eceff4"), // Snow Storm 2
		Accent:         hex("#d08770"), // Aurora orange
		Secondary:      hex("#a3be8c"), // Aurora green
		Muted:          hex("#d8dee9"), // Snow Storm 0
		Subtle:         hex("#4c566a"), // Polar Night 3
		InputBg:        hex("#3b4252"),
		Placeholder:    hex("#4c566a"),
	}
}

func gruvboxDark() Theme {
	return Theme{
		Name:           "gruvbox-dark",
		Border:         hex("#83a598"), // bright aqua
		InactiveBorder: hex("#3c3836"),
		Primary:        hex("#ebdbb2"), // fg
		Accent:         hex("#fe8019"), // bright orange
		Secondary:      hex("#b8bb26"), // bright green
		Muted:          hex("#a89984"),
		Subtle:         hex("#7c6f64"),
		InputBg:        hex("#3c3836"),
		Placeholder:    hex("#7c6f64"),
	}
}

func tokyoNight() Theme {
	return Theme{
		Name:           "tokyo-night",
		Border:         hex("#7aa2f7"), // blue
		InactiveBorder: hex("#292e42"),
		Primary:        hex("#c0caf5"), // fg
		Accent:         hex("#ff9e64"), // orange
		Secondary:      hex("#9ece6a"), // green
		Muted:          hex("#a9b1d6"),
		Subtle:         hex("#565f89"),
		InputBg:        hex("#292e42"),
		Placeholder:    hex("#565f89"),
	}
}
