package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/rgehrsitz/rpgo/internal/tui/tuimsg"
)

// Scene represents different screens in the TUI
type Scene int

const (
	SceneHome Scene = iota
	SceneScenarios
	SceneParameters
	SceneCompare
	SceneOptimize
	SceneResults
	SceneHelp
)

// Message types for the Bubble Tea update cycle

// NavigateMsg switches to a different scene
type NavigateMsg struct {
	Scene Scene
}

// QuitMsg signals the application should exit
type QuitMsg struct{}

// Re-export messages from tuimsg to avoid import cycles
type (
	ErrorMsg                = tuimsg.ErrorMsg
	ConfigLoadedMsg         = tuimsg.ConfigLoadedMsg
	ScenarioSelectedMsg     = tuimsg.ScenarioSelectedMsg
	ParameterChangedMsg     = tuimsg.ParameterChangedMsg
	CalculationStartedMsg   = tuimsg.CalculationStartedMsg
	CalculationCompleteMsg  = tuimsg.CalculationCompleteMsg
	ComparisonStartedMsg    = tuimsg.ComparisonStartedMsg
	ComparisonCompleteMsg   = tuimsg.ComparisonCompleteMsg
	OptimizationStartedMsg  = tuimsg.OptimizationStartedMsg
	OptimizationProgressMsg = tuimsg.OptimizationProgressMsg
	OptimizationCompleteMsg = tuimsg.OptimizationCompleteMsg
	SaveScenarioMsg         = tuimsg.SaveScenarioMsg
	SaveCompleteMsg         = tuimsg.SaveCompleteMsg
)

// KeyMsg is a wrapper for tea.KeyMsg for easier handling
type KeyMsg tea.KeyMsg

// WindowSizeMsg signals the terminal window has been resized
type WindowSizeMsg struct {
	Width  int
	Height int
}

// TickMsg is sent at regular intervals for animations
type TickMsg struct{}
