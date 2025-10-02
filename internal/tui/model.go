package tui

import (
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
	parametersModel interface{} // ParametersModel (to be created)
	compareModel    interface{} // CompareModel (to be created)
	optimizeModel   interface{} // OptimizeModel (to be created)
	resultsModel    interface{} // ResultsModel (to be created)
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
