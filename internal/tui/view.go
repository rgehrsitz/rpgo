package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the current state of the application
func (m Model) View() string {
	if m.loading {
		return m.renderLoading()
	}

	if m.err != nil {
		return m.renderError()
	}

	// Render the current scene
	var content string
	switch m.currentScene {
	case SceneHome:
		content = m.renderHome()
	case SceneScenarios:
		content = m.renderScenarios()
	case SceneParameters:
		content = m.renderParameters()
	case SceneCompare:
		content = m.renderCompare()
	case SceneOptimize:
		content = m.renderOptimize()
	case SceneResults:
		content = m.renderResults()
	case SceneHelp:
		content = m.renderHelp()
	default:
		content = "Unknown scene"
	}

	// Wrap content with app styling and status bar
	return m.renderApp(content)
}

// renderApp wraps content with title bar, status bar, and main container
func (m Model) renderApp(content string) string {
	titleBar := m.renderTitleBar()
	statusBar := m.renderStatusBar()

	// Calculate available height for content
	contentHeight := m.height - 4 // Title (2) + status (1) + padding (1)

	// Wrap content in a viewport-style container
	contentContainer := lipgloss.NewStyle().
		Height(contentHeight).
		Render(content)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleBar,
		contentContainer,
		statusBar,
	)
}

// renderTitleBar renders the application title and breadcrumb
func (m Model) renderTitleBar() string {
	title := TitleStyle.Render("RPGO - FERS Retirement Planning")

	// Build breadcrumb
	breadcrumb := ""
	if m.config != nil && m.selectedScenario != "" {
		breadcrumb = SubtitleStyle.Render(
			fmt.Sprintf("%s / %s", m.currentScene.String(), m.selectedScenario),
		)
	} else {
		breadcrumb = SubtitleStyle.Render(m.currentScene.String())
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		breadcrumb,
	)
}

// renderStatusBar renders the bottom status bar with keyboard shortcuts
func (m Model) renderStatusBar() string {
	shortcuts := []string{
		formatShortcut("h", "home"),
		formatShortcut("s", "scenarios"),
		formatShortcut("p", "parameters"),
		formatShortcut("c", "compare"),
		formatShortcut("o", "optimize"),
		formatShortcut("r", "results"),
		formatShortcut("?", "help"),
		formatShortcut("q", "quit"),
	}

	statusText := strings.Join(shortcuts, " • ")

	// Right-align config name if loaded
	if m.config != nil {
		configName := SubtitleStyle.Render("Config loaded")
		width := m.width - lipgloss.Width(statusText) - 4
		spacer := strings.Repeat(" ", max(0, width))
		statusText = statusText + spacer + configName
	}

	return StatusBarStyle.Width(m.width).Render(statusText)
}

// formatShortcut formats a keyboard shortcut with key and description
func formatShortcut(key, desc string) string {
	return StatusKeyStyle.Render(key) + " " + desc
}

// renderLoading renders a loading spinner/message
func (m Model) renderLoading() string {
	message := m.loadingMessage
	if message == "" {
		message = "Loading..."
	}

	spinner := "⠋" // Simple spinner frame (can be animated with TickMsg)

	content := BorderStyle.Render(
		fmt.Sprintf("%s %s", spinner, message),
	)

	return m.renderApp(content)
}

// renderError renders an error message
func (m Model) renderError() string {
	errorMsg := "An error occurred"
	if m.err != nil {
		errorMsg = m.err.Error()
	}

	content := ErrorStyle.Render(
		fmt.Sprintf("Error: %s\n\nPress any key to continue...", errorMsg),
	)

	return m.renderApp(content)
}

// Scene-specific render functions

// renderHome renders the home dashboard
func (m Model) renderHome() string {
	// Placeholder for home scene
	// Will be implemented in next phase
	if m.config == nil {
		return BorderStyle.Render(
			"Welcome to RPGO!\n\n" +
			"Loading configuration...",
		)
	}

	return BorderStyle.Render(
		"Welcome to RPGO!\n\n" +
		"Configuration loaded successfully.\n" +
		"Use the keyboard shortcuts below to navigate.",
	)
}

// renderScenarios renders the scenarios list
func (m Model) renderScenarios() string {
	if m.scenariosModel != nil {
		return m.scenariosModel.View()
	}
	return BorderStyle.Render("Scenarios scene - Coming soon")
}

// renderParameters renders the parameter editing screen
func (m Model) renderParameters() string {
	return BorderStyle.Render("Parameters scene - Coming soon")
}

// renderCompare renders the comparison view
func (m Model) renderCompare() string {
	return BorderStyle.Render("Compare scene - Coming soon")
}

// renderOptimize renders the optimization interface
func (m Model) renderOptimize() string {
	if m.optimizationInProgress {
		progress := fmt.Sprintf("Progress: %d/%d - %s",
			m.optimizationProgress,
			m.optimizationTotal,
			m.optimizationStatus,
		)
		return BorderStyle.Render(
			"Optimization in progress...\n\n" + progress,
		)
	}

	return BorderStyle.Render("Optimize scene - Coming soon")
}

// renderResults renders detailed results
func (m Model) renderResults() string {
	return BorderStyle.Render("Results scene - Coming soon")
}

// renderHelp renders the help screen
func (m Model) renderHelp() string {
	helpText := `
RPGO - FERS Retirement Planning Calculator

KEYBOARD SHORTCUTS:
  h        Navigate to Home
  s        Navigate to Scenarios
  p        Navigate to Parameters
  c        Navigate to Compare
  o        Navigate to Optimize
  r        Navigate to Results
  ?        Show this help
  ESC      Go back
  q/Ctrl+C Quit

NAVIGATION:
  Use arrow keys to navigate lists and menus
  Enter to select items
  Tab to move between sections

EDITING:
  Type to modify values
  +/- to increment/decrement
  Arrow keys to adjust sliders
`

	return BorderStyle.Render(helpText)
}

// Helper function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
