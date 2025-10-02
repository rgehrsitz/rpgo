package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rgehrsitz/rpgo/internal/tui"
)

func main() {
	// Get config file path from arguments
	configPath := ""
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	} else {
		fmt.Println("Usage: rpgo-tui <config-file>")
		os.Exit(1)
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("Error: Config file not found: %s\n", configPath)
		os.Exit(1)
	}

	// Create the application model
	model := tui.NewModel(configPath)

	// Create the Bubble Tea program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
