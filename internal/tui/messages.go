package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/rgehrsitz/rpgo/internal/domain"
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

// ErrorMsg displays an error to the user
type ErrorMsg struct {
	Err error
}

// ConfigLoadedMsg signals configuration has been loaded
type ConfigLoadedMsg struct {
	Config *domain.Configuration
}

// ScenarioSelectedMsg signals a scenario has been selected
type ScenarioSelectedMsg struct {
	ScenarioName string
}

// ParameterChangedMsg signals a parameter value has changed
type ParameterChangedMsg struct {
	Participant string
	Parameter   string
	Value       interface{}
}

// CalculationStartedMsg signals a calculation has begun
type CalculationStartedMsg struct {
	ScenarioName string
}

// CalculationCompleteMsg signals a calculation has finished
type CalculationCompleteMsg struct {
	ScenarioName string
	Results      *domain.ScenarioSummary
	Err          error
}

// ComparisonStartedMsg signals a comparison calculation has begun
type ComparisonStartedMsg struct {
	ScenarioNames []string
}

// ComparisonCompleteMsg signals a comparison has finished
type ComparisonCompleteMsg struct {
	Comparisons map[string]*domain.ScenarioSummary
	Err         error
}

// OptimizationStartedMsg signals an optimization has begun
type OptimizationStartedMsg struct {
	Target string
	Goal   string
}

// OptimizationProgressMsg provides progress updates during optimization
type OptimizationProgressMsg struct {
	Iteration int
	Total     int
	Status    string
}

// OptimizationCompleteMsg signals an optimization has finished
type OptimizationCompleteMsg struct {
	Target  string
	Goal    string
	Results interface{} // OptimizationResult or MultiDimensionalResult
	Err     error
}

// KeyMsg is a wrapper for tea.KeyMsg for easier handling
type KeyMsg tea.KeyMsg

// WindowSizeMsg signals the terminal window has been resized
type WindowSizeMsg struct {
	Width  int
	Height int
}

// TickMsg is sent at regular intervals for animations
type TickMsg struct{}
