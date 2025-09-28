package calculation

import (
	"fmt"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// CalculateBreakEvenTSPWithdrawalRate calculates the TSP withdrawal percentage needed to match current net income
// Gather all participants

// Find the first year when all are fully retired

// Binary search for the correct TSP withdrawal rate

// Return the best rate found
// CalculateBreakEvenTSPWithdrawalRate calculates the TSP withdrawal percentage needed to match current net income (generic)
func cloneGenericScenario(s *domain.GenericScenario) *domain.GenericScenario {
	if s == nil {
		return nil
	}
	newScenario := *s
	newScenario.ParticipantScenarios = make(map[string]domain.ParticipantScenario)
	for k, v := range s.ParticipantScenarios {
		newScenario.ParticipantScenarios[k] = v
	}
	return &newScenario
}

func (ce *CalculationEngine) CalculateBreakEvenTSPWithdrawalRate(config *domain.Configuration, scenario *domain.GenericScenario, targetNetIncome decimal.Decimal) (decimal.Decimal, *domain.AnnualCashFlow, error) {
	// Gather participant names from scenario
	participantNames := make([]string, 0, len(scenario.ParticipantScenarios))
	for name := range scenario.ParticipantScenarios {
		participantNames = append(participantNames, name)
	}

	projectionStartYear := ProjectionBaseYear
	retirementYears := make(map[string]int)
	for _, name := range participantNames {
		retirementDate := scenario.ParticipantScenarios[name].RetirementDate
		retirementYears[name] = retirementDate.Year() - projectionStartYear
	}
	firstFullRetirementYear := 0
	for _, year := range retirementYears {
		if year > firstFullRetirementYear {
			firstFullRetirementYear = year
		}
	}
	firstFullRetirementYear++

	minRate := decimal.NewFromFloat(0.001)
	maxRate := decimal.NewFromFloat(0.15)
	tolerance := decimal.NewFromFloat(1000)
	maxIterations := 50

	for i := 0; i < maxIterations; i++ {
		testRate := minRate.Add(maxRate).Div(decimal.NewFromInt(2))
		testScenario := cloneGenericScenario(scenario)
		for _, name := range participantNames {
			ps := testScenario.ParticipantScenarios[name]
			ps.TSPWithdrawalStrategy = "variable_percentage"
			ps.TSPWithdrawalRate = &testRate
			testScenario.ParticipantScenarios[name] = ps
		}
		// Use generic household participants
		household := config.Household
		projection := ce.GenerateAnnualProjectionGeneric(household, testScenario, &config.GlobalAssumptions, config.GlobalAssumptions.FederalRules)
		if firstFullRetirementYear >= len(projection) {
			return decimal.Zero, nil, fmt.Errorf("first full retirement year (%d) exceeds projection length (%d)", firstFullRetirementYear, len(projection))
		}
		testYear := projection[firstFullRetirementYear]
		netIncomeDiff := testYear.NetIncome.Sub(targetNetIncome)
		if netIncomeDiff.Abs().LessThan(tolerance) {
			return testRate, &testYear, nil
		}
		if netIncomeDiff.LessThan(decimal.Zero) {
			minRate = testRate
		} else {
			maxRate = testRate
		}
		if maxRate.Sub(minRate).LessThan(decimal.NewFromFloat(0.0001)) {
			break
		}
	}
	finalRate := minRate.Add(maxRate).Div(decimal.NewFromInt(2))
	testScenario := cloneGenericScenario(scenario)
	for _, name := range participantNames {
		ps := testScenario.ParticipantScenarios[name]
		ps.TSPWithdrawalStrategy = "variable_percentage"
		ps.TSPWithdrawalRate = &finalRate
		testScenario.ParticipantScenarios[name] = ps
	}
	// Use generic household participants for final projection
	household := config.Household
	projection := ce.GenerateAnnualProjectionGeneric(household, testScenario, &config.GlobalAssumptions, config.GlobalAssumptions.FederalRules)
	finalYear := projection[firstFullRetirementYear]
	return finalRate, &finalYear, nil
}

// Only one definition of CalculateBreakEvenAnalysis should exist. If duplicate, remove the extra.

// CalculateBreakEvenAnalysis calculates break-even TSP withdrawal rates for all scenarios

func (ce *CalculationEngine) CalculateBreakEvenAnalysis(config *domain.Configuration) (*BreakEvenAnalysis, error) {
	// Calculate current net income as the target (sum for all participants)
	household := config.Household
	targetNetIncome := ce.calculateCurrentNetIncomeGeneric(household)

	// Iterate over generic scenarios
	results := make([]BreakEvenResult, len(config.Scenarios))
	for i := range config.Scenarios {
		scenario := &config.Scenarios[i]
		rate, yearData, err := ce.CalculateBreakEvenTSPWithdrawalRate(config, scenario, targetNetIncome)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate break-even rate for scenario %s: %v", scenario.Name, err)
		}

		// Sum all participant TSP withdrawals
		withdrawalTotal := decimal.Zero
		for _, w := range yearData.TSPWithdrawals {
			withdrawalTotal = withdrawalTotal.Add(w)
		}
		results[i] = BreakEvenResult{
			ScenarioName:            scenario.Name,
			BreakEvenWithdrawalRate: rate,
			ProjectedNetIncome:      yearData.NetIncome,
			ProjectedYear:           yearData.Year + (ProjectionBaseYear - 1),
			TSPWithdrawalAmount:     withdrawalTotal,
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
