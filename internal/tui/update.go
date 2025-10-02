package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/rgehrsitz/rpgo/internal/tui/scenes"
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

		// Propagate size changes to all scene models
		if m.scenariosModel != nil {
			m.scenariosModel.SetSize(msg.Width, msg.Height)
		}
		if m.parametersModel != nil {
			m.parametersModel.SetSize(msg.Width, msg.Height)
		}
		if m.compareModel != nil {
			m.compareModel.SetSize(msg.Width, msg.Height)
		}
		if m.optimizeModel != nil {
			m.optimizeModel.SetSize(msg.Width, msg.Height)
		}
		if m.resultsModel != nil {
			m.resultsModel.SetSize(msg.Width, msg.Height)
		}

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
		if msg.Config != nil {
			if m.scenariosModel != nil {
				m.scenariosModel.SetScenarios(msg.Config.Scenarios)
				m.scenariosModel.SetSize(m.width, m.height)
			}
			if m.compareModel != nil {
				m.compareModel.SetScenarios(msg.Config.Scenarios)
				m.compareModel.SetSize(m.width, m.height)
			}
			if m.optimizeModel != nil {
				m.optimizeModel.SetScenarios(msg.Config.Scenarios)
				m.optimizeModel.SetSize(m.width, m.height)
			}
		}
		return m, nil

	case ScenarioSelectedMsg:
		m.selectedScenario = msg.ScenarioName

		// Find and load the selected scenario into parameters model
		if m.config != nil && m.parametersModel != nil {
			for _, scenario := range m.config.Scenarios {
				if scenario.Name == msg.ScenarioName {
					// Make a copy to avoid modifying the original
					scenarioCopy := scenario.DeepCopy()
					m.parametersModel.SetScenario(scenarioCopy)
					m.parametersModel.SetSize(m.width, m.height)
					break
				}
			}
		}

		// Navigate to parameters scene
		return m, func() tea.Msg {
			return NavigateMsg{Scene: SceneParameters}
		}

	case ParameterChangedMsg:
		// Handle parameter changes
		// Will trigger recalculation in later phases
		return m, nil

	case CalculationStartedMsg:
		m.loading = true
		m.loadingMessage = "Calculating scenario..."

		// Get the scenario from parameters model
		if m.parametersModel != nil && m.parametersModel.GetScenario() != nil && m.config != nil {
			return m, calculateScenarioCmd(m.parametersModel.GetScenario(), m.config)
		}
		return m, nil

	case CalculationCompleteMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.selectedResults = msg.Results
			// Update results model and navigate to results scene
			if m.resultsModel != nil {
				m.resultsModel.SetResults(msg.ScenarioName, msg.Results)
				m.resultsModel.SetSize(m.width, m.height)
			}
			return m, func() tea.Msg {
				return NavigateMsg{Scene: SceneResults}
			}
		}
		return m, nil

	case ComparisonStartedMsg:
		m.comparisonScenarios = msg.ScenarioNames
		// Start calculating multiple scenarios
		if m.config != nil {
			return m, calculateMultipleScenariosCmd(msg.ScenarioNames, m.config)
		}
		return m, nil

	case ComparisonCompleteMsg:
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.comparisonResults = msg.Comparisons
			// Update compare model with results
			if m.compareModel != nil {
				m.compareModel.SetResults(msg.Comparisons)
			}
		}
		return m, nil

	case OptimizationStartedMsg:
		m.optimizationInProgress = true
		// Start the break-even optimization
		if m.config != nil {
			return m, optimizeBreakEvenCmd(msg.ScenarioName, msg.TargetIncome, m.config)
		}
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
			// Update optimize model with results
			if m.optimizeModel != nil {
				if result, ok := msg.Results.(*scenes.OptimizeResult); ok {
					m.optimizeModel.SetResult(result)
				}
			}
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
		if m.parametersModel != nil {
			updatedModel, cmd := m.parametersModel.Update(msg)
			m.parametersModel = updatedModel
			return m, cmd
		}
		return m, nil
	case SceneCompare:
		if m.compareModel != nil {
			updatedModel, cmd := m.compareModel.Update(msg)
			m.compareModel = updatedModel
			return m, cmd
		}
		return m, nil
	case SceneOptimize:
		if m.optimizeModel != nil {
			updatedModel, cmd := m.optimizeModel.Update(msg)
			m.optimizeModel = updatedModel
			return m, cmd
		}
		return m, nil
	case SceneResults:
		if m.resultsModel != nil {
			updatedModel, cmd := m.resultsModel.Update(msg)
			m.resultsModel = updatedModel
			return m, cmd
		}
		return m, nil
	case SceneHelp:
		// TODO: Update help model
		return m, nil
	}

	return m, nil
}
