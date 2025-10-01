package breakeven

import (
	"context"
	"fmt"
	"time"

	"github.com/rgehrsitz/rpgo/internal/calculation"
	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/rgehrsitz/rpgo/internal/transform"
	"github.com/shopspring/decimal"
)

// Solver provides multi-dimensional break-even optimization
type Solver struct {
	CalcEngine *calculation.CalculationEngine
	Options    SolverOptions
}

// NewSolver creates a new break-even solver
func NewSolver(calcEngine *calculation.CalculationEngine, options SolverOptions) *Solver {
	return &Solver{
		CalcEngine: calcEngine,
		Options:    options,
	}
}

// NewDefaultSolver creates a solver with default options
func NewDefaultSolver(calcEngine *calculation.CalculationEngine) *Solver {
	return NewSolver(calcEngine, DefaultSolverOptions())
}

// Optimize performs optimization based on the request
func (s *Solver) Optimize(ctx context.Context, req OptimizationRequest) (*OptimizationResult, error) {
	// Validate constraints
	if err := req.Constraints.Validate(); err != nil {
		return nil, err
	}

	// Apply defaults
	if req.MaxIterations == 0 {
		req.MaxIterations = s.Options.MaxIterations
	}
	if req.Tolerance.IsZero() {
		req.Tolerance = s.Options.Tolerance
	}

	// Route to appropriate solver based on target
	switch req.Target {
	case OptimizeTSPRate:
		return s.optimizeTSPRate(ctx, req)
	case OptimizeRetirementDate:
		return s.optimizeRetirementDate(ctx, req)
	case OptimizeSSAge:
		return s.optimizeSSAge(ctx, req)
	case OptimizeTSPBalance:
		return s.optimizeTSPBalance(ctx, req)
	default:
		return nil, &BreakEvenError{
			Operation: "optimize",
			Message:   fmt.Sprintf("unsupported optimization target: %s", req.Target),
		}
	}
}

// optimizeTSPRate finds the optimal TSP withdrawal rate
func (s *Solver) optimizeTSPRate(ctx context.Context, req OptimizationRequest) (*OptimizationResult, error) {
	// Get rate bounds from constraints
	minRate := decimal.NewFromFloat(0.01)
	maxRate := decimal.NewFromFloat(0.15)
	if req.Constraints.MinTSPRate != nil {
		minRate = *req.Constraints.MinTSPRate
	}
	if req.Constraints.MaxTSPRate != nil {
		maxRate = *req.Constraints.MaxTSPRate
	}

	var bestResult *OptimizationResult
	iterations := 0

	// Binary search for optimal rate
	for iterations < req.MaxIterations {
		iterations++

		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		testRate := minRate.Add(maxRate).Div(decimal.NewFromInt(2))

		// Apply rate transform to scenario
		rateTransform := &transform.AdjustTSPRate{
			Participant: req.Constraints.Participant,
			NewRate:     testRate,
		}

		// First, ensure the strategy supports rates
		strategyTransform := &transform.ModifyTSPStrategy{
			Participant: req.Constraints.Participant,
			NewStrategy: "variable_percentage",
		}

		modifiedScenario, err := transform.ApplyTransforms(req.BaseScenario, []transform.ScenarioTransform{
			strategyTransform,
			rateTransform,
		})
		if err != nil {
			return nil, &BreakEvenError{
				Operation: "optimize_tsp_rate",
				Message:   "failed to apply rate transform",
				Cause:     err,
			}
		}

		// Calculate scenario
		summary, err := s.CalcEngine.RunGenericScenario(ctx, req.Config, modifiedScenario)
		if err != nil {
			return nil, &BreakEvenError{
				Operation: "optimize_tsp_rate",
				Message:   "failed to calculate scenario",
				Cause:     err,
			}
		}

		// Evaluate based on goal
		result := s.evaluateResult(req, summary, &testRate, nil, nil, iterations)

		// Check if goal is met
		if req.Goal == GoalMatchIncome && req.Constraints.TargetIncome != nil {
			diff := summary.FirstYearNetIncome.Sub(*req.Constraints.TargetIncome)
			if diff.Abs().LessThan(req.Tolerance) {
				result.Success = true
				result.ConvergenceInfo = fmt.Sprintf("Converged to target income within $%s", req.Tolerance.StringFixed(0))
				return result, nil
			}

			// Adjust bounds
			if diff.LessThan(decimal.Zero) {
				// Need more income, increase rate
				minRate = testRate
			} else {
				// Too much income, decrease rate
				maxRate = testRate
			}
		} else {
			// For other goals, track best result
			if bestResult == nil || s.isBetter(result, bestResult, req.Goal) {
				bestResult = result
			}
		}

		// Check convergence
		if maxRate.Sub(minRate).LessThan(decimal.NewFromFloat(0.0001)) {
			if bestResult != nil {
				bestResult.Success = true
				bestResult.ConvergenceInfo = "Binary search converged"
				return bestResult, nil
			}
			result.Success = true
			result.ConvergenceInfo = "Binary search converged"
			return result, nil
		}
	}

	if bestResult != nil {
		bestResult.ConvergenceInfo = fmt.Sprintf("Max iterations (%d) reached", req.MaxIterations)
		return bestResult, nil
	}

	return nil, &BreakEvenError{
		Operation: "optimize_tsp_rate",
		Message:   fmt.Sprintf("optimization did not converge after %d iterations", req.MaxIterations),
	}
}

// optimizeRetirementDate finds the optimal retirement date
func (s *Solver) optimizeRetirementDate(ctx context.Context, req OptimizationRequest) (*OptimizationResult, error) {
	// Get participant's current retirement date
	ps, exists := req.BaseScenario.ParticipantScenarios[req.Constraints.Participant]
	if !exists {
		return nil, &BreakEvenError{
			Operation: "optimize_retirement_date",
			Message:   fmt.Sprintf("participant %s not found", req.Constraints.Participant),
		}
	}

	baseDate := ps.RetirementDate
	if baseDate == nil {
		return nil, &BreakEvenError{
			Operation: "optimize_retirement_date",
			Message:   "participant has no retirement date set",
		}
	}

	// Determine search range (default: -24 to +36 months from base)
	minDate := baseDate.AddDate(0, -24, 0)
	maxDate := baseDate.AddDate(0, 36, 0)

	if req.Constraints.MinRetirementDate != nil {
		minDate = *req.Constraints.MinRetirementDate
	}
	if req.Constraints.MaxRetirementDate != nil {
		maxDate = *req.Constraints.MaxRetirementDate
	}

	// Grid search over retirement dates (monthly granularity)
	var bestResult *OptimizationResult
	currentDate := minDate
	iterations := 0

	for !currentDate.After(maxDate) && iterations < req.MaxIterations {
		iterations++

		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Calculate months difference from base
		months := monthsBetween(*baseDate, currentDate)

		// Apply retirement date transform
		var dateTransform transform.ScenarioTransform
		if months == 0 {
			// No change needed
			dateTransform = nil
		} else {
			dateTransform = &transform.PostponeRetirement{
				Participant: req.Constraints.Participant,
				Months:      months,
			}
		}

		var modifiedScenario *domain.GenericScenario
		var err error

		if dateTransform != nil {
			modifiedScenario, err = transform.ApplyTransforms(req.BaseScenario, []transform.ScenarioTransform{dateTransform})
			if err != nil {
				return nil, &BreakEvenError{
					Operation: "optimize_retirement_date",
					Message:   "failed to apply date transform",
					Cause:     err,
				}
			}
		} else {
			modifiedScenario = req.BaseScenario.DeepCopy()
		}

		// Calculate scenario
		summary, err := s.CalcEngine.RunGenericScenario(ctx, req.Config, modifiedScenario)
		if err != nil {
			// Skip this date if calculation fails
			currentDate = currentDate.AddDate(0, 1, 0)
			continue
		}

		// Evaluate result
		result := s.evaluateResult(req, summary, nil, &currentDate, nil, iterations)

		// Track best result
		if bestResult == nil || s.isBetter(result, bestResult, req.Goal) {
			bestResult = result
		}

		// Move to next month
		currentDate = currentDate.AddDate(0, 1, 0)
	}

	if bestResult != nil {
		bestResult.Success = true
		bestResult.ConvergenceInfo = fmt.Sprintf("Evaluated %d retirement dates", iterations)
		return bestResult, nil
	}

	return nil, &BreakEvenError{
		Operation: "optimize_retirement_date",
		Message:   "no valid retirement dates found",
	}
}

// optimizeSSAge finds the optimal Social Security claiming age
func (s *Solver) optimizeSSAge(ctx context.Context, req OptimizationRequest) (*OptimizationResult, error) {
	// Get age bounds
	minAge := 62
	maxAge := 70
	if req.Constraints.MinSSAge != nil {
		minAge = *req.Constraints.MinSSAge
	}
	if req.Constraints.MaxSSAge != nil {
		maxAge = *req.Constraints.MaxSSAge
	}

	var bestResult *OptimizationResult
	iterations := 0

	// Grid search over ages (yearly granularity)
	for age := minAge; age <= maxAge; age++ {
		iterations++

		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Apply SS age transform
		ageTransform := &transform.DelaySSClaim{
			Participant: req.Constraints.Participant,
			NewAge:      age,
		}

		modifiedScenario, err := transform.ApplyTransforms(req.BaseScenario, []transform.ScenarioTransform{ageTransform})
		if err != nil {
			return nil, &BreakEvenError{
				Operation: "optimize_ss_age",
				Message:   "failed to apply age transform",
				Cause:     err,
			}
		}

		// Calculate scenario
		summary, err := s.CalcEngine.RunGenericScenario(ctx, req.Config, modifiedScenario)
		if err != nil {
			continue // Skip this age if calculation fails
		}

		// Evaluate result
		result := s.evaluateResult(req, summary, nil, nil, &age, iterations)

		// Track best result
		if bestResult == nil || s.isBetter(result, bestResult, req.Goal) {
			bestResult = result
		}
	}

	if bestResult != nil {
		bestResult.Success = true
		bestResult.ConvergenceInfo = fmt.Sprintf("Evaluated %d Social Security ages", iterations)
		return bestResult, nil
	}

	return nil, &BreakEvenError{
		Operation: "optimize_ss_age",
		Message:   "no valid SS ages found",
	}
}

// optimizeTSPBalance finds required TSP balance for goal
func (s *Solver) optimizeTSPBalance(ctx context.Context, req OptimizationRequest) (*OptimizationResult, error) {
	// This would require modifying TSP balance in scenario
	// For now, return not implemented
	return nil, &BreakEvenError{
		Operation: "optimize_tsp_balance",
		Message:   "TSP balance optimization not yet implemented",
	}
}

// evaluateResult creates an optimization result from a scenario summary
func (s *Solver) evaluateResult(
	req OptimizationRequest,
	summary *domain.ScenarioSummary,
	tspRate *decimal.Decimal,
	retirementDate *time.Time,
	ssAge *int,
	iterations int,
) *OptimizationResult {
	result := &OptimizationResult{
		Request:            req,
		Iterations:         iterations,
		ScenarioSummary:    summary,
		FirstYearNetIncome: summary.FirstYearNetIncome,
		LifetimeIncome:     summary.TotalLifetimeIncome,
		TSPLongevity:       summary.TSPLongevity,
		LifetimeTaxes:      s.calculateLifetimeTaxes(summary),
	}

	// Set optimal parameters
	if tspRate != nil {
		rateCopy := *tspRate
		result.OptimalTSPRate = &rateCopy
	}
	if retirementDate != nil {
		dateCopy := *retirementDate
		result.OptimalRetirementDate = &dateCopy
	}
	if ssAge != nil {
		ageCopy := *ssAge
		result.OptimalSSAge = &ageCopy
	}

	return result
}

// isBetter compares two results based on optimization goal
func (s *Solver) isBetter(a, b *OptimizationResult, goal OptimizationGoal) bool {
	switch goal {
	case GoalMaximizeIncome:
		return a.LifetimeIncome.GreaterThan(b.LifetimeIncome)
	case GoalMaximizeLongevity:
		return a.TSPLongevity > b.TSPLongevity
	case GoalMinimizeTaxes:
		return a.LifetimeTaxes.LessThan(b.LifetimeTaxes)
	case GoalMatchIncome:
		// For match income, closer to target is better
		if a.Request.Constraints.TargetIncome == nil {
			return false
		}
		aDiff := a.FirstYearNetIncome.Sub(*a.Request.Constraints.TargetIncome).Abs()
		bDiff := b.FirstYearNetIncome.Sub(*b.Request.Constraints.TargetIncome).Abs()
		return aDiff.LessThan(bDiff)
	default:
		return false
	}
}

// calculateLifetimeTaxes sums all taxes over projection
func (s *Solver) calculateLifetimeTaxes(summary *domain.ScenarioSummary) decimal.Decimal {
	total := decimal.Zero
	for _, year := range summary.Projection {
		total = total.Add(year.FederalTax).
			Add(year.StateTax).
			Add(year.LocalTax).
			Add(year.FICATax)
	}
	return total
}

// monthsBetween calculates months between two dates (can be negative)
func monthsBetween(from, to time.Time) int {
	years := to.Year() - from.Year()
	months := int(to.Month()) - int(from.Month())
	return years*12 + months
}
