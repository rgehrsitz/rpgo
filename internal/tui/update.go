package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles all messages and updates the model state
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Standard tea.Msg types
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	// Custom messages
	case NavigateMsg:
		m.previousScene = m.currentScene
		m.currentScene = msg.Scene
		return m, nil

	case QuitMsg:
		return m, tea.Quit

	case ErrorMsg:
		m.err = msg.Err
		return m, nil

	case ConfigLoadedMsg:
		m.config = msg.Config
		m.loading = false
		// Initialize calculation engine with config
		// m.calcEngine = calculation.NewCalculationEngine(msg.Config)

		// Populate scenarios model if config is loaded
		if msg.Config != nil && m.scenariosModel != nil {
			m.scenariosModel.SetScenarios(msg.Config.Scenarios)
			m.scenariosModel.SetSize(m.width, m.height)
		}
		return m, nil

	case ScenarioSelectedMsg:
		m.selectedScenario = msg.ScenarioName
		return m, nil

	case ParameterChangedMsg:
		// Handle parameter changes
		// Will trigger recalculation in later phases
		return m, nil

	case CalculationStartedMsg:
		m.loading = true
		m.loadingMessage = "Calculating scenario..."
		return m, nil

	case CalculationCompleteMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.selectedResults = msg.Results
		}
		return m, nil

	case ComparisonStartedMsg:
		m.loading = true
		m.loadingMessage = "Comparing scenarios..."
		m.comparisonScenarios = msg.ScenarioNames
		return m, nil

	case ComparisonCompleteMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.comparisonResults = msg.Comparisons
		}
		return m, nil

	case OptimizationStartedMsg:
		m.optimizationInProgress = true
		m.optimizationProgress = 0
		m.optimizationTotal = 100 // Default, will be updated by progress messages
		return m, nil

	case OptimizationProgressMsg:
		m.optimizationProgress = msg.Iteration
		m.optimizationTotal = msg.Total
		m.optimizationStatus = msg.Status
		return m, nil

	case OptimizationCompleteMsg:
		m.optimizationInProgress = false
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.optimizationResults = msg.Results
		}
		return m, nil

	case TickMsg:
		// Handle animation ticks if needed
		return m, nil
	}

	// Delegate to scene-specific update handlers
	return m.updateCurrentScene(msg)
}

// handleKeyPress processes keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keyboard shortcuts
	switch msg.String() {
	case "ctrl+c", "q":
		// Quit from any scene (except when editing)
		return m, tea.Quit

	case "?":
		// Show help from any scene
		return m, func() tea.Msg {
			return NavigateMsg{Scene: SceneHelp}
		}

	case "esc":
		// Go back to previous scene or home
		if m.currentScene != SceneHome {
			return m, func() tea.Msg {
				if m.previousScene != SceneHome && m.previousScene != m.currentScene {
					return NavigateMsg{Scene: m.previousScene}
				}
				return NavigateMsg{Scene: SceneHome}
			}
		}

	case "h":
		// Navigate to home (if not already there)
		if m.currentScene != SceneHome {
			return m, func() tea.Msg {
				return NavigateMsg{Scene: SceneHome}
			}
		}

	case "s":
		// Navigate to scenarios
		if m.currentScene != SceneScenarios {
			return m, func() tea.Msg {
				return NavigateMsg{Scene: SceneScenarios}
			}
		}

	case "p":
		// Navigate to parameters
		if m.currentScene != SceneParameters {
			return m, func() tea.Msg {
				return NavigateMsg{Scene: SceneParameters}
			}
		}

	case "c":
		// Navigate to compare
		if m.currentScene != SceneCompare {
			return m, func() tea.Msg {
				return NavigateMsg{Scene: SceneCompare}
			}
		}

	case "o":
		// Navigate to optimize
		if m.currentScene != SceneOptimize {
			return m, func() tea.Msg {
				return NavigateMsg{Scene: SceneOptimize}
			}
		}

	case "r":
		// Navigate to results
		if m.currentScene != SceneResults {
			return m, func() tea.Msg {
				return NavigateMsg{Scene: SceneResults}
			}
		}
	}

	// Let the current scene handle other keys
	return m.updateCurrentScene(msg)
}

// updateCurrentScene delegates updates to the current scene's model
func (m Model) updateCurrentScene(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Scene-specific update handling will be added as we build each scene
	// For now, just return the model unchanged
	switch m.currentScene {
	case SceneHome:
		// TODO: Update home model
		return m, nil
	case SceneScenarios:
		if m.scenariosModel != nil {
			updatedModel, cmd := m.scenariosModel.Update(msg)
			m.scenariosModel = updatedModel
			return m, cmd
		}
		return m, nil
	case SceneParameters:
		// TODO: Update parameters model
		return m, nil
	case SceneCompare:
		// TODO: Update compare model
		return m, nil
	case SceneOptimize:
		// TODO: Update optimize model
		return m, nil
	case SceneResults:
		// TODO: Update results model
		return m, nil
	case SceneHelp:
		// TODO: Update help model
		return m, nil
	}

	return m, nil
}
