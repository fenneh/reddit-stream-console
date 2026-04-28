package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fenneh/reddit-stream-console/internal/app"
	"github.com/fenneh/reddit-stream-console/internal/config"
	"github.com/fenneh/reddit-stream-console/internal/reddit"
	"github.com/fenneh/reddit-stream-console/internal/theme"
)

func main() {
	diag := false
	for _, arg := range os.Args[1:] {
		if arg == "--diag" || arg == "-diag" {
			diag = true
		}
	}

	_ = config.LoadDotEnv(".env")

	appConfig, appConfigErr := config.LoadAppConfig("config/app_config.json")
	if appConfig.DebugLogging {
		file, err := os.OpenFile("reddit_stream_debug.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			log.SetOutput(file)
		}
	}

	menuConfig, err := config.LoadMenuConfig("config/menu_config.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load menu config: %v\n", err)
		os.Exit(1)
	}

	userAgent := os.Getenv("REDDIT_USER_AGENT")
	if userAgent == "" {
		userAgent = "RedditStreamConsole/1.0"
	}

	resolvedTheme, themeOK := theme.Lookup(appConfig.Theme)
	var themeWarning string
	if !themeOK {
		themeWarning = fmt.Sprintf("Unknown theme %q — using %q. Available: %s",
			appConfig.Theme, resolvedTheme.Name, strings.Join(theme.Names(), ", "))
		fmt.Fprintln(os.Stderr, themeWarning)
	}

	if diag {
		printDiagnostics(appConfig, appConfigErr, resolvedTheme)
		return
	}

	client := reddit.NewClient(userAgent)
	tviewApp := app.NewTviewApp(menuConfig.MenuItems, client, resolvedTheme)
	if themeWarning != "" {
		tviewApp.SetStartupNotice(themeWarning)
	}

	if err := tviewApp.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start app: %v\n", err)
		os.Exit(1)
	}
}

func printDiagnostics(appConfig config.AppConfig, appConfigErr error, resolved theme.Theme) {
	exe, _ := os.Executable()
	fmt.Println("reddit-stream-console diagnostics")
	fmt.Println("=================================")
	fmt.Printf("executable        : %s\n", exe)
	fmt.Printf("working directory : %s\n", mustWD())
	fmt.Println()
	fmt.Println("config search paths (priority order):")
	for i, dir := range config.SearchPaths() {
		mark := "  "
		if path := filepath.Join(dir, "config", "app_config.json"); fileExists(path) {
			mark = "✓ "
		}
		fmt.Printf("  %d. %s%s\n", i+1, mark, dir)
	}
	fmt.Println()
	fmt.Printf("app_config.json resolved : %s\n", emptyAsDash(config.ResolveConfigPath("config/app_config.json")))
	fmt.Printf("app_config.json error    : %v\n", appConfigErr)
	fmt.Printf("menu_config.json resolved: %s\n", emptyAsDash(config.ResolveConfigPath("config/menu_config.json")))
	fmt.Println()
	fmt.Printf("theme requested : %q\n", appConfig.Theme)
	fmt.Printf("theme resolved  : %s\n", resolved.Name)
	fmt.Printf("available themes: %s\n", strings.Join(theme.Names(), ", "))
	fmt.Println()
	fmt.Println("environment:")
	for _, name := range []string{"TERM", "COLORTERM", "TERM_PROGRAM", "SSH_CONNECTION", "WT_SESSION"} {
		fmt.Printf("  %-15s = %q\n", name, os.Getenv(name))
	}
	if os.Getenv("COLORTERM") != "truecolor" && os.Getenv("COLORTERM") != "24bit" {
		fmt.Println()
		fmt.Println("WARNING: COLORTERM is not 'truecolor' or '24bit'. The terminal may snap")
		fmt.Println("         theme colours to a 256-colour palette, making themes look similar.")
		fmt.Println("         Try: COLORTERM=truecolor ./reddit-stream-console")
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func emptyAsDash(s string) string {
	if s == "" {
		return "(none found)"
	}
	return s
}

func mustWD() string {
	wd, _ := os.Getwd()
	return wd
}
