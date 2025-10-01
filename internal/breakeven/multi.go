package breakeven

import (
	"context"
	"fmt"

	"github.com/rgehrsitz/rpgo/internal/domain"
)

// OptimizeMultiDimensional runs optimization across multiple targets and compares results
func (s *Solver) OptimizeMultiDimensional(
	ctx context.Context,
	baseScenario *domain.GenericScenario,
	config *domain.Configuration,
	constraints Constraints,
	goals []OptimizationGoal,
) (*MultiDimensionalResult, error) {

	// Validate constraints
	if err := constraints.Validate(); err != nil {
		return nil, err
	}

	// Define optimization targets to test
	targets := []OptimizationTarget{
		OptimizeTSPRate,
		OptimizeRetirementDate,
		OptimizeSSAge,
	}

	var results []OptimizationResult

	// Run optimization for each target and goal combination
	for _, target := range targets {
		for _, goal := range goals {
			req := OptimizationRequest{
				BaseScenario:  baseScenario,
				Config:        config,
				Target:        target,
				Goal:          goal,
				Constraints:   constraints,
				MaxIterations: s.Options.MaxIterations,
				Tolerance:     s.Options.Tolerance,
			}

			result, err := s.Optimize(ctx, req)
			if err != nil {
				// Log error but continue with other optimizations
				continue
			}

			if result != nil && result.Success {
				results = append(results, *result)
			}
		}
	}

	if len(results) == 0 {
		return nil, &BreakEvenError{
			Operation: "optimize_multi_dimensional",
			Message:   "no successful optimizations found",
		}
	}

	// Find best results for each metric
	mdResult := &MultiDimensionalResult{
		Results: results,
	}

	// Find best by income
	for i := range results {
		if mdResult.BestByIncome == nil ||
			results[i].LifetimeIncome.GreaterThan(mdResult.BestByIncome.LifetimeIncome) {
			mdResult.BestByIncome = &results[i]
		}
	}

	// Find best by longevity
	for i := range results {
		if mdResult.BestByLongevity == nil ||
			results[i].TSPLongevity > mdResult.BestByLongevity.TSPLongevity {
			mdResult.BestByLongevity = &results[i]
		}
	}

	// Find best by taxes
	for i := range results {
		if mdResult.BestByTaxes == nil ||
			results[i].LifetimeTaxes.LessThan(mdResult.BestByTaxes.LifetimeTaxes) {
			mdResult.BestByTaxes = &results[i]
		}
	}

	// Generate recommendations
	mdResult.Recommendations = s.generateMultiDimensionalRecommendations(mdResult)

	return mdResult, nil
}

// generateMultiDimensionalRecommendations creates recommendations from multi-dimensional results
func (s *Solver) generateMultiDimensionalRecommendations(result *MultiDimensionalResult) []string {
	var recommendations []string

	// Income recommendation
	if result.BestByIncome != nil {
		rec := fmt.Sprintf("To maximize lifetime income: Optimize %s",
			result.BestByIncome.Request.Target)
		if result.BestByIncome.OptimalTSPRate != nil {
			rec += fmt.Sprintf(" (%.2f%% withdrawal rate)",
				result.BestByIncome.OptimalTSPRate.InexactFloat64()*100)
		}
		if result.BestByIncome.OptimalRetirementDate != nil {
			rec += fmt.Sprintf(" (retire %s)",
				result.BestByIncome.OptimalRetirementDate.Format("Jan 2006"))
		}
		if result.BestByIncome.OptimalSSAge != nil {
			rec += fmt.Sprintf(" (claim SS at %d)", *result.BestByIncome.OptimalSSAge)
		}
		recommendations = append(recommendations, rec)
	}

	// Longevity recommendation
	if result.BestByLongevity != nil {
		rec := fmt.Sprintf("To maximize TSP longevity (%d years): Optimize %s",
			result.BestByLongevity.TSPLongevity,
			result.BestByLongevity.Request.Target)
		recommendations = append(recommendations, rec)
	}

	// Tax recommendation
	if result.BestByTaxes != nil {
		rec := fmt.Sprintf("To minimize taxes: Optimize %s (saves $%s)",
			result.BestByTaxes.Request.Target,
			result.BestByTaxes.LifetimeTaxes.StringFixed(0))
		recommendations = append(recommendations, rec)
	}

	// Check if same strategy wins multiple categories
	if result.BestByIncome != nil && result.BestByLongevity != nil {
		if result.BestByIncome.Request.Target == result.BestByLongevity.Request.Target {
			recommendations = append(recommendations,
				fmt.Sprintf("⭐ Optimizing %s provides both high income AND longevity",
					result.BestByIncome.Request.Target))
		}
	}

	return recommendations
}

// OptimizeAllTargets is a convenience method to optimize all targets with a single goal
func (s *Solver) OptimizeAllTargets(
	ctx context.Context,
	baseScenario *domain.GenericScenario,
	config *domain.Configuration,
	constraints Constraints,
	goal OptimizationGoal,
) (*MultiDimensionalResult, error) {
	return s.OptimizeMultiDimensional(ctx, baseScenario, config, constraints, []OptimizationGoal{goal})
}
