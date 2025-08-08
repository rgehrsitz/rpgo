package calculation

import (
	"fmt"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// CalculateBreakEvenTSPWithdrawalRate calculates the TSP withdrawal percentage needed to match current net income
func (ce *CalculationEngine) CalculateBreakEvenTSPWithdrawalRate(config *domain.Configuration, scenario *domain.Scenario, targetNetIncome decimal.Decimal) (decimal.Decimal, *domain.AnnualCashFlow, error) {
	robertEmployee := config.PersonalDetails["robert"]
	dawnEmployee := config.PersonalDetails["dawn"]

	// Find the first year when both are fully retired
	projectionStartYear := ProjectionBaseYear
	robertRetirementYear := scenario.Robert.RetirementDate.Year() - projectionStartYear
	dawnRetirementYear := scenario.Dawn.RetirementDate.Year() - projectionStartYear
	firstFullRetirementYear := robertRetirementYear
	if dawnRetirementYear > robertRetirementYear {
		firstFullRetirementYear = dawnRetirementYear
	}
	// Add 1 to get the first FULL year after both are retired
	firstFullRetirementYear++

	// Binary search for the correct TSP withdrawal rate
	minRate := decimal.NewFromFloat(0.001)  // 0.1%
	maxRate := decimal.NewFromFloat(0.15)   // 15%
	tolerance := decimal.NewFromFloat(1000) // Within $1,000
	maxIterations := 50

	for i := 0; i < maxIterations; i++ {
		// Calculate midpoint withdrawal rate
		testRate := minRate.Add(maxRate).Div(decimal.NewFromInt(2))

		// Create a test scenario with this withdrawal rate
		testScenario := *scenario
		testScenario.Robert.TSPWithdrawalStrategy = "variable_percentage"
		testScenario.Robert.TSPWithdrawalRate = &testRate
		testScenario.Dawn.TSPWithdrawalStrategy = "variable_percentage"
		testScenario.Dawn.TSPWithdrawalRate = &testRate

		// Run projection to get the first full retirement year
		projection := ce.GenerateAnnualProjection(&robertEmployee, &dawnEmployee, &testScenario, &config.GlobalAssumptions, config.GlobalAssumptions.FederalRules)

		// Check if we have enough projection years
		if firstFullRetirementYear >= len(projection) {
			return decimal.Zero, nil, fmt.Errorf("first full retirement year (%d) exceeds projection length (%d)", firstFullRetirementYear, len(projection))
		}

		testYear := projection[firstFullRetirementYear]
		netIncomeDiff := testYear.NetIncome.Sub(targetNetIncome)

		// Check if we're within tolerance
		if netIncomeDiff.Abs().LessThan(tolerance) {
			return testRate, &testYear, nil
		}

		// Adjust search range
		if netIncomeDiff.LessThan(decimal.Zero) {
			// Net income is too low, need higher withdrawal rate
			minRate = testRate
		} else {
			// Net income is too high, need lower withdrawal rate
			maxRate = testRate
		}

		// Check if search range is too narrow
		if maxRate.Sub(minRate).LessThan(decimal.NewFromFloat(0.0001)) {
			break
		}
	}

	// Return the best rate found
	finalRate := minRate.Add(maxRate).Div(decimal.NewFromInt(2))
	testScenario := *scenario
	testScenario.Robert.TSPWithdrawalStrategy = "variable_percentage"
	testScenario.Robert.TSPWithdrawalRate = &finalRate
	testScenario.Dawn.TSPWithdrawalStrategy = "variable_percentage"
	testScenario.Dawn.TSPWithdrawalRate = &finalRate

	projection := ce.GenerateAnnualProjection(&robertEmployee, &dawnEmployee, &testScenario, &config.GlobalAssumptions, config.GlobalAssumptions.FederalRules)
	finalYear := projection[firstFullRetirementYear]

	return finalRate, &finalYear, nil
}

// CalculateBreakEvenAnalysis calculates break-even TSP withdrawal rates for all scenarios
func (ce *CalculationEngine) CalculateBreakEvenAnalysis(config *domain.Configuration) (*BreakEvenAnalysis, error) {
	// Calculate current net income as the target
	robertEmployee := config.PersonalDetails["robert"]
	dawnEmployee := config.PersonalDetails["dawn"]
	targetNetIncome := ce.calculateCurrentNetIncome(&robertEmployee, &dawnEmployee, &config.GlobalAssumptions)

	results := make([]BreakEvenResult, len(config.Scenarios))

	for i, scenario := range config.Scenarios {
		rate, yearData, err := ce.CalculateBreakEvenTSPWithdrawalRate(config, &scenario, targetNetIncome)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate break-even rate for scenario %s: %v", scenario.Name, err)
		}

		results[i] = BreakEvenResult{
			ScenarioName:            scenario.Name,
			BreakEvenWithdrawalRate: rate,
			ProjectedNetIncome:      yearData.NetIncome,
			ProjectedYear:           yearData.Year + (ProjectionBaseYear - 1),
			TSPWithdrawalAmount:     yearData.TSPWithdrawalRobert.Add(yearData.TSPWithdrawalDawn),
			TotalTSPBalance:         yearData.TotalTSPBalance(),
			CurrentVsBreakEvenDiff:  yearData.NetIncome.Sub(targetNetIncome),
		}
	}

	return &BreakEvenAnalysis{
		TargetNetIncome: targetNetIncome,
		Results:         results,
	}, nil
}

// BreakEvenAnalysis contains the results of break-even TSP withdrawal rate analysis
type BreakEvenAnalysis struct {
	TargetNetIncome decimal.Decimal   `json:"target_net_income"`
	Results         []BreakEvenResult `json:"results"`
}

// BreakEvenResult contains break-even calculation results for a single scenario
type BreakEvenResult struct {
	ScenarioName            string          `json:"scenario_name"`
	BreakEvenWithdrawalRate decimal.Decimal `json:"break_even_withdrawal_rate"`
	ProjectedNetIncome      decimal.Decimal `json:"projected_net_income"`
	ProjectedYear           int             `json:"projected_year"`
	TSPWithdrawalAmount     decimal.Decimal `json:"tsp_withdrawal_amount"`
	TotalTSPBalance         decimal.Decimal `json:"total_tsp_balance"`
	CurrentVsBreakEvenDiff  decimal.Decimal `json:"current_vs_break_even_diff"`
}
