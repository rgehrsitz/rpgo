package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rgehrsitz/rpgo/internal/calculation"
	"github.com/rgehrsitz/rpgo/internal/config"
	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/rgehrsitz/rpgo/internal/tui/scenes"
)

// Model represents the entire application state
type Model struct {
	// Navigation
	currentScene Scene
	previousScene Scene

	// Terminal dimensions
	width  int
	height int

	// Configuration and data
	configPath string
	config     *domain.Configuration

	// Calculation engine
	calcEngine *calculation.CalculationEngine

	// Current selections
	selectedScenario string
	selectedResults  *domain.ScenarioSummary

	// Comparison data
	comparisonScenarios []string
	comparisonResults   map[string]*domain.ScenarioSummary

	// Optimization data
	optimizationInProgress bool
	optimizationProgress   int
	optimizationTotal      int
	optimizationStatus     string
	optimizationResults    interface{}

	// Scene-specific models (will be added as we build each scene)
	homeModel       interface{} // HomeModel (to be created)
	scenariosModel  *scenes.ScenariosModel
	parametersModel *scenes.ParametersModel
	compareModel    interface{} // CompareModel (to be created)
	optimizeModel   interface{} // OptimizeModel (to be created)
	resultsModel    *scenes.ResultsModel
	helpModel       interface{} // HelpModel (to be created)

	// Error state
	err error

	// Loading state
	loading bool
	loadingMessage string
}

// NewModel creates a new application model
func NewModel(configPath string) Model {
	return Model{
		currentScene:        SceneHome,
		configPath:          configPath,
		comparisonResults:   make(map[string]*domain.ScenarioSummary),
		scenariosModel:      scenes.NewScenariosModel(),
		parametersModel:     scenes.NewParametersModel(),
		resultsModel:        scenes.NewResultsModel(),
		width:               80,
		height:              24,
	}
}

// Init initializes the model (required by tea.Model interface)
func (m Model) Init() tea.Cmd {
	// Return a command to load the configuration
	return loadConfigCmd(m.configPath)
}

// loadConfigCmd returns a command that loads the configuration file
func loadConfigCmd(path string) tea.Cmd {
	return func() tea.Msg {
		// Load configuration from YAML file
		parser := config.NewInputParser()
		cfg, err := parser.LoadFromFile(path)
		if err != nil {
			return ErrorMsg{Err: err}
		}

		return ConfigLoadedMsg{
			Config: cfg,
		}
	}
}

// calculateScenarioCmd returns a command that calculates a scenario
func calculateScenarioCmd(scenario *domain.GenericScenario, cfg *domain.Configuration) tea.Cmd {
	return func() tea.Msg {
		// Create a temporary config with just this scenario
		tempConfig := &domain.Configuration{
			GlobalAssumptions: cfg.GlobalAssumptions,
			Household:         cfg.Household,
			Scenarios:         []domain.GenericScenario{*scenario},
		}

		// Create calculation engine
		engine := calculation.NewCalculationEngineWithConfig(cfg.GlobalAssumptions.FederalRules)

		// Run calculation
		results, err := engine.RunScenarios(tempConfig)
		if err != nil {
			return CalculationCompleteMsg{
				ScenarioName: scenario.Name,
				Results:      nil,
				Err:          err,
			}
		}

		// Extract the single scenario result
		if len(results.Scenarios) > 0 {
			return CalculationCompleteMsg{
				ScenarioName: scenario.Name,
				Results:      &results.Scenarios[0],
				Err:          nil,
			}
		}

		return CalculationCompleteMsg{
			ScenarioName: scenario.Name,
			Results:      nil,
			Err:          fmt.Errorf("no results returned from calculation"),
		}
	}
}

// GetSceneName returns a human-readable name for a scene
func (s Scene) String() string {
	switch s {
	case SceneHome:
		return "Home"
	case SceneScenarios:
		return "Scenarios"
	case SceneParameters:
		return "Parameters"
	case SceneCompare:
		return "Compare"
	case SceneOptimize:
		return "Optimize"
	case SceneResults:
		return "Results"
	case SceneHelp:
		return "Help"
	default:
		return "Unknown"
	}
}
