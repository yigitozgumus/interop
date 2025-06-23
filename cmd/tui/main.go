package main

import (
	"fmt"
	"interop/internal/logging"
	"interop/internal/settings"
	"interop/internal/tui"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Load configuration
	cfg, err := settings.Load()
	if err != nil {
		logging.ErrorAndExit("Failed to load configuration: %v", err)
	}

	// Create the TUI model
	model := tui.NewCommandsModel(cfg)

	// Create the Bubble Tea program
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
