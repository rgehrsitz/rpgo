package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shopspring/decimal"

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
	compareModel    *scenes.CompareModel
	optimizeModel   *scenes.OptimizeModel
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
		compareModel:        scenes.NewCompareModel(),
		optimizeModel:       scenes.NewOptimizeModel(),
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

// calculateMultipleScenariosCmd returns a command that calculates multiple scenarios
func calculateMultipleScenariosCmd(scenarioNames []string, cfg *domain.Configuration) tea.Cmd {
	return func() tea.Msg {
		results := make(map[string]*domain.ScenarioSummary)

		// Find and calculate each scenario
		for _, name := range scenarioNames {
			// Find scenario by name
			var scenario *domain.GenericScenario
			for i := range cfg.Scenarios {
				if cfg.Scenarios[i].Name == name {
					scenario = &cfg.Scenarios[i]
					break
				}
			}

			if scenario == nil {
				continue
			}

			// Create temporary config with just this scenario
			tempConfig := &domain.Configuration{
				GlobalAssumptions: cfg.GlobalAssumptions,
				Household:         cfg.Household,
				Scenarios:         []domain.GenericScenario{*scenario},
			}

			// Create calculation engine
			engine := calculation.NewCalculationEngineWithConfig(cfg.GlobalAssumptions.FederalRules)

			// Run calculation
			projResults, err := engine.RunScenarios(tempConfig)
			if err == nil && len(projResults.Scenarios) > 0 {
				results[name] = &projResults.Scenarios[0]
			}
		}

		return ComparisonCompleteMsg{
			Comparisons: results,
			Err:         nil,
		}
	}
}

// optimizeBreakEvenCmd returns a command that runs break-even optimization
func optimizeBreakEvenCmd(scenarioName string, targetIncome interface{}, cfg *domain.Configuration) tea.Cmd {
	return func() tea.Msg {
		// Find scenario by name
		var scenario *domain.GenericScenario
		for i := range cfg.Scenarios {
			if cfg.Scenarios[i].Name == scenarioName {
				scenario = &cfg.Scenarios[i]
				break
			}
		}

		if scenario == nil {
			return OptimizationCompleteMsg{
				Results: nil,
				Err:     fmt.Errorf("scenario not found: %s", scenarioName),
			}
		}

		// Create calculation engine
		engine := calculation.NewCalculationEngineWithConfig(cfg.GlobalAssumptions.FederalRules)

		// Convert target income to decimal
		target, ok := targetIncome.(decimal.Decimal)
		if !ok {
			return OptimizationCompleteMsg{
				Results: nil,
				Err:     fmt.Errorf("invalid target income type"),
			}
		}

		// Run break-even optimization
		optimalRate, cashFlow, err := engine.CalculateBreakEvenTSPWithdrawalRate(cfg, scenario, target)
		if err != nil {
			return OptimizationCompleteMsg{
				Results: nil,
				Err:     err,
			}
		}

		// Create result
		result := &scenes.OptimizeResult{
			OptimalRate:  optimalRate,
			TargetIncome: target,
			CashFlow:     cashFlow,
			ScenarioName: scenarioName,
		}

		if cashFlow != nil {
			result.ActualIncome = cashFlow.NetIncome
		}

		return OptimizationCompleteMsg{
			Results: result,
			Err:     nil,
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
