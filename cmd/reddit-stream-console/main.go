package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/fenneh/reddit-stream-console/internal/app"
	"github.com/fenneh/reddit-stream-console/internal/config"
	"github.com/fenneh/reddit-stream-console/internal/reddit"
)

func main() {
	_ = config.LoadDotEnv(".env")

	appConfig, _ := config.LoadAppConfig("config/app_config.json")
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

	client := reddit.NewClient(userAgent)
	model := app.NewModel(menuConfig.MenuItems, client)

	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start app: %v\n", err)
		os.Exit(1)
	}
}
