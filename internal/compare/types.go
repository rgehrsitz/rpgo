package compare

import (
	"fmt"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// ComparisonResult represents a single scenario comparison with calculated metrics
type ComparisonResult struct {
	ScenarioName string `json:"scenarioName"`
	Description  string `json:"description"`
	Summary      *domain.ScenarioSummary

	// Key Metrics
	FirstYearNetIncome decimal.Decimal `json:"firstYearNetIncome"`
	LifetimeIncome     decimal.Decimal `json:"lifetimeIncome"`
	TSPLongevity       int             `json:"tspLongevity"` // Years until TSP depletion
	FinalTSPBalance    decimal.Decimal `json:"finalTSPBalance"`
	LifetimeTaxes      decimal.Decimal `json:"lifetimeTaxes"`

	// Comparison to Base
	IncomeDiffFromBase decimal.Decimal `json:"incomeDiffFromBase"`
	IncomePctFromBase  decimal.Decimal `json:"incomePctFromBase"`
	TSPLongevityDiff   int             `json:"tspLongevityDiff"`
	TaxDiffFromBase    decimal.Decimal `json:"taxDiffFromBase"`

	// Scenario Specifics (extracted from scenario for display)
	RetirementDate        string `json:"retirementDate,omitempty"`
	SSClaimAge            int    `json:"ssClaimAge,omitempty"`
	TSPWithdrawalStrategy string `json:"tspWithdrawalStrategy,omitempty"`
	TSPWithdrawalRate     string `json:"tspWithdrawalRate,omitempty"`
}

// ComparisonSet represents a collection of scenario comparisons
type ComparisonSet struct {
	BaseScenarioName   string             `json:"baseScenarioName"`
	BaseResult         *ComparisonResult  `json:"baseResult"`
	AlternativeResults []ComparisonResult `json:"alternativeResults"`
	Recommendations    []string           `json:"recommendations"`
	ConfigPath         string             `json:"configPath"`
}

// ToScenarioComparison converts a ComparisonSet to a domain.ScenarioComparison for HTML output
func (cs *ComparisonSet) ToScenarioComparison() *domain.ScenarioComparison {
	scenarios := make([]domain.ScenarioSummary, 0, len(cs.AlternativeResults)+1)

	// Add base scenario
	if cs.BaseResult != nil && cs.BaseResult.Summary != nil {
		scenarios = append(scenarios, *cs.BaseResult.Summary)
	}

	// Add alternative scenarios
	for _, result := range cs.AlternativeResults {
		if result.Summary != nil {
			scenarios = append(scenarios, *result.Summary)
		}
	}

	// Create baseline net income from base scenario
	baselineNetIncome := decimal.Zero
	if cs.BaseResult != nil {
		baselineNetIncome = cs.BaseResult.FirstYearNetIncome
	}

	return &domain.ScenarioComparison{
		BaselineNetIncome: baselineNetIncome,
		Scenarios:         scenarios,
		Assumptions: []string{
			"General COLA (FERS pension & SS): 2.0% annually",
			"FEHB premium inflation: 6.0% annually",
			"TSP growth pre-retirement: 6.0% annually",
			"TSP growth post-retirement: 4.0% annually",
			"Social Security wage base indexing: ~5% annually (2025 est: $168,600)",
			"Tax brackets: 2025 levels held constant (no inflation indexing)",
		},
	}
}

// MetricsCalculator extracts key metrics from scenario summaries
type MetricsCalculator struct{}

// NewMetricsCalculator creates a new metrics calculator
func NewMetricsCalculator() *MetricsCalculator {
	return &MetricsCalculator{}
}

// CalculateMetrics computes all comparison metrics for a scenario summary
func (mc *MetricsCalculator) CalculateMetrics(summary *domain.ScenarioSummary) ComparisonResult {
	result := ComparisonResult{
		ScenarioName:       summary.Name,
		Summary:            summary,
		FirstYearNetIncome: summary.FirstYearNetIncome,
		LifetimeIncome:     summary.TotalLifetimeIncome,
		TSPLongevity:       summary.TSPLongevity,
		FinalTSPBalance:    summary.FinalTSPBalance,
		LifetimeTaxes:      mc.calculateLifetimeTaxes(summary),
	}

	// Extract scenario specifics from first projection year
	if len(summary.Projection) > 0 {
		firstYear := summary.Projection[0]
		result.RetirementDate = firstYear.Date.Format("2006-01-02")

		// Get participant info (for display purposes, use first participant)
		participantNames := firstYear.GetParticipantNames()
		if len(participantNames) > 0 {
			// This is just for display - actual scenario has all participant details
			result.SSClaimAge = 0 // Would need to track this from scenario
		}
	}

	return result
}

// CalculateComparison computes comparison metrics between a scenario and a base
func (mc *MetricsCalculator) CalculateComparison(scenario, base ComparisonResult) ComparisonResult {
	scenario.IncomeDiffFromBase = scenario.LifetimeIncome.Sub(base.LifetimeIncome)

	if !base.LifetimeIncome.IsZero() {
		scenario.IncomePctFromBase = scenario.IncomeDiffFromBase.
			Div(base.LifetimeIncome).
			Mul(decimal.NewFromInt(100))
	}

	scenario.TSPLongevityDiff = scenario.TSPLongevity - base.TSPLongevity
	scenario.TaxDiffFromBase = scenario.LifetimeTaxes.Sub(base.LifetimeTaxes)

	return scenario
}

// calculateLifetimeTaxes sums all taxes paid across the projection
func (mc *MetricsCalculator) calculateLifetimeTaxes(summary *domain.ScenarioSummary) decimal.Decimal {
	total := decimal.Zero
	for _, year := range summary.Projection {
		total = total.Add(year.FederalTax).
			Add(year.StateTax).
			Add(year.LocalTax).
			Add(year.FICATax)
	}
	return total
}

// GenerateRecommendations creates recommendations based on comparison results
func GenerateRecommendations(compSet *ComparisonSet) []string {
	recommendations := []string{}

	if len(compSet.AlternativeResults) == 0 {
		return recommendations
	}

	// Find best scenario by lifetime income
	bestIncome := compSet.BaseResult
	for _, alt := range compSet.AlternativeResults {
		if alt.LifetimeIncome.GreaterThan(bestIncome.LifetimeIncome) {
			bestIncome = &alt
		}
	}

	if bestIncome != compSet.BaseResult {
		incomeDiff := bestIncome.LifetimeIncome.Sub(compSet.BaseResult.LifetimeIncome)
		recommendations = append(recommendations,
			"Best Income: "+bestIncome.ScenarioName+" provides $"+incomeDiff.StringFixed(0)+
				" more lifetime income than base scenario")
	}

	// Find best TSP longevity
	bestLongevity := compSet.BaseResult
	for _, alt := range compSet.AlternativeResults {
		if alt.TSPLongevity > bestLongevity.TSPLongevity {
			bestLongevity = &alt
		}
	}

	if bestLongevity != compSet.BaseResult {
		yearsDiff := bestLongevity.TSPLongevity - compSet.BaseResult.TSPLongevity
		recommendations = append(recommendations,
			"Best Longevity: "+bestLongevity.ScenarioName+" extends TSP by "+
				fmt.Sprintf("%d years", yearsDiff))
	}

	// Find lowest tax burden
	lowestTax := compSet.BaseResult
	for _, alt := range compSet.AlternativeResults {
		if alt.LifetimeTaxes.LessThan(lowestTax.LifetimeTaxes) {
			lowestTax = &alt
		}
	}

	if lowestTax != compSet.BaseResult {
		taxSavings := compSet.BaseResult.LifetimeTaxes.Sub(lowestTax.LifetimeTaxes)
		recommendations = append(recommendations,
			"Lowest Taxes: "+lowestTax.ScenarioName+" saves $"+taxSavings.StringFixed(0)+
				" in lifetime taxes")
	}

	return recommendations
}
