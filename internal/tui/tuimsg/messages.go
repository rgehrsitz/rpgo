package tuimsg

import (
	"github.com/rgehrsitz/rpgo/internal/domain"
)

// ScenarioSelectedMsg signals a scenario has been selected
type ScenarioSelectedMsg struct {
	ScenarioName string
}

// ConfigLoadedMsg signals configuration has been loaded
type ConfigLoadedMsg struct {
	Config *domain.Configuration
}

// ErrorMsg displays an error to the user
type ErrorMsg struct {
	Err error
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
	ScenarioName string
	TargetIncome interface{} // decimal.Decimal
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

// SaveScenarioMsg signals a request to save the modified scenario
type SaveScenarioMsg struct {
	Scenario *GenericScenario
	Filename string
}

// SaveCompleteMsg signals a save operation has finished
type SaveCompleteMsg struct {
	Filename string
	Err      error
}

// GenericScenario is imported from domain but we need a type alias for messaging
type GenericScenario = domain.GenericScenario
