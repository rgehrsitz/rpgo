package breakeven

import (
	"time"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// OptimizationTarget defines what parameter to optimize
type OptimizationTarget string

const (
	OptimizeRetirementDate OptimizationTarget = "retirement_date"
	OptimizeTSPRate        OptimizationTarget = "tsp_rate"
	OptimizeTSPBalance     OptimizationTarget = "tsp_balance"
	OptimizeSSAge          OptimizationTarget = "ss_age"
	OptimizeAll            OptimizationTarget = "all"
)

// OptimizationGoal defines what outcome to achieve
type OptimizationGoal string

const (
	GoalMatchIncome      OptimizationGoal = "match_income"       // Match specific target income
	GoalMaximizeIncome   OptimizationGoal = "maximize_income"    // Maximize lifetime income
	GoalMaximizeLongevity OptimizationGoal = "maximize_longevity" // Maximize TSP longevity
	GoalMinimizeTaxes    OptimizationGoal = "minimize_taxes"     // Minimize tax burden
)

// Constraints define bounds for optimization parameters
type Constraints struct {
	// Retirement date constraints
	MinRetirementDate *time.Time `json:"min_retirement_date,omitempty"`
	MaxRetirementDate *time.Time `json:"max_retirement_date,omitempty"`

	// TSP withdrawal rate constraints (as decimal, e.g., 0.04 for 4%)
	MinTSPRate *decimal.Decimal `json:"min_tsp_rate,omitempty"`
	MaxTSPRate *decimal.Decimal `json:"max_tsp_rate,omitempty"`

	// TSP balance constraints
	MinTSPBalance *decimal.Decimal `json:"min_tsp_balance,omitempty"`
	MaxTSPBalance *decimal.Decimal `json:"max_tsp_balance,omitempty"`

	// Social Security age constraints
	MinSSAge *int `json:"min_ss_age,omitempty"`
	MaxSSAge *int `json:"max_ss_age,omitempty"`

	// Income target for match_income goal
	TargetIncome *decimal.Decimal `json:"target_income,omitempty"`

	// Participant to optimize (required)
	Participant string `json:"participant"`
}

// DefaultConstraints returns sensible default constraints
func DefaultConstraints(participant string) Constraints {
	minRate := decimal.NewFromFloat(0.02)
	maxRate := decimal.NewFromFloat(0.10)
	minSSAge := 62
	maxSSAge := 70

	return Constraints{
		MinTSPRate:  &minRate,
		MaxTSPRate:  &maxRate,
		MinSSAge:    &minSSAge,
		MaxSSAge:    &maxSSAge,
		Participant: participant,
	}
}

// OptimizationRequest defines the parameters for an optimization run
type OptimizationRequest struct {
	BaseScenario   *domain.GenericScenario
	Config         *domain.Configuration
	Target         OptimizationTarget
	Goal           OptimizationGoal
	Constraints    Constraints
	MaxIterations  int // Maximum solver iterations
	Tolerance      decimal.Decimal // Convergence tolerance for binary search
}

// OptimizationResult contains the results of an optimization run
type OptimizationResult struct {
	// Optimization metadata
	Request         OptimizationRequest
	Success         bool
	Iterations      int
	ConvergenceInfo string

	// Optimized parameters
	OptimalRetirementDate *time.Time       `json:"optimal_retirement_date,omitempty"`
	OptimalTSPRate        *decimal.Decimal `json:"optimal_tsp_rate,omitempty"`
	OptimalTSPBalance     *decimal.Decimal `json:"optimal_tsp_balance,omitempty"`
	OptimalSSAge          *int             `json:"optimal_ss_age,omitempty"`

	// Results at optimal parameters
	ScenarioSummary     *domain.ScenarioSummary `json:"scenario_summary"`
	FirstYearNetIncome  decimal.Decimal         `json:"first_year_net_income"`
	LifetimeIncome      decimal.Decimal         `json:"lifetime_income"`
	TSPLongevity        int                     `json:"tsp_longevity"`
	LifetimeTaxes       decimal.Decimal         `json:"lifetime_taxes"`

	// Comparison to base (if applicable)
	BaseScenarioSummary *domain.ScenarioSummary `json:"base_scenario_summary,omitempty"`
	IncomeDiffFromBase  decimal.Decimal         `json:"income_diff_from_base,omitempty"`
	TaxDiffFromBase     decimal.Decimal         `json:"tax_diff_from_base,omitempty"`
}

// MultiDimensionalResult contains results when optimizing multiple parameters
type MultiDimensionalResult struct {
	Results        []OptimizationResult
	BestByIncome   *OptimizationResult
	BestByLongevity *OptimizationResult
	BestByTaxes    *OptimizationResult
	Recommendations []string
}

// SolverOptions configures the solver algorithm
type SolverOptions struct {
	Algorithm      string          // "binary_search", "grid_search", "gradient"
	GridResolution int             // For grid search: points per dimension
	Tolerance      decimal.Decimal // Convergence tolerance
	MaxIterations  int             // Maximum iterations
	Parallel       bool            // Use parallel evaluation (future)
}

// DefaultSolverOptions returns default solver configuration
func DefaultSolverOptions() SolverOptions {
	return SolverOptions{
		Algorithm:      "binary_search",
		GridResolution: 10,
		Tolerance:      decimal.NewFromInt(1000), // $1000 tolerance
		MaxIterations:  50,
		Parallel:       false,
	}
}

// Validate checks if constraints are internally consistent
func (c *Constraints) Validate() error {
	if c.Participant == "" {
		return &BreakEvenError{
			Operation: "validate_constraints",
			Message:   "participant name is required",
		}
	}

	// Check retirement date range
	if c.MinRetirementDate != nil && c.MaxRetirementDate != nil {
		if c.MinRetirementDate.After(*c.MaxRetirementDate) {
			return &BreakEvenError{
				Operation: "validate_constraints",
				Message:   "min_retirement_date cannot be after max_retirement_date",
			}
		}
	}

	// Check TSP rate range
	if c.MinTSPRate != nil && c.MaxTSPRate != nil {
		if c.MinTSPRate.GreaterThan(*c.MaxTSPRate) {
			return &BreakEvenError{
				Operation: "validate_constraints",
				Message:   "min_tsp_rate cannot be greater than max_tsp_rate",
			}
		}
	}

	// Check SS age range
	if c.MinSSAge != nil && c.MaxSSAge != nil {
		if *c.MinSSAge > *c.MaxSSAge {
			return &BreakEvenError{
				Operation: "validate_constraints",
				Message:   "min_ss_age cannot be greater than max_ss_age",
			}
		}
		if *c.MinSSAge < 62 || *c.MaxSSAge > 70 {
			return &BreakEvenError{
				Operation: "validate_constraints",
				Message:   "ss_age must be between 62 and 70",
			}
		}
	}

	return nil
}

// BreakEvenError represents errors from break-even solver
type BreakEvenError struct {
	Operation string
	Message   string
	Cause     error
}

func (e *BreakEvenError) Error() string {
	if e.Cause != nil {
		return e.Operation + ": " + e.Message + ": " + e.Cause.Error()
	}
	return e.Operation + ": " + e.Message
}

func (e *BreakEvenError) Unwrap() error {
	return e.Cause
}
