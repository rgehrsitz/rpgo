package compare

import (
	"testing"
	"time"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

func TestMetricsCalculator_CalculateMetrics(t *testing.T) {
	calc := NewMetricsCalculator()

	summary := &domain.ScenarioSummary{
		Name:                "Test Scenario",
		FirstYearNetIncome:  decimal.NewFromInt(100000),
		TotalLifetimeIncome: decimal.NewFromInt(3000000),
		TSPLongevity:        25,
		FinalTSPBalance:     decimal.NewFromInt(500000),
		Projection: []domain.AnnualCashFlow{
			{
				Year:       0,
				Date:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				FederalTax: decimal.NewFromInt(10000),
				StateTax:   decimal.NewFromInt(3000),
				LocalTax:   decimal.NewFromInt(500),
				FICATax:    decimal.NewFromInt(7000),
			},
			{
				Year:       1,
				Date:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				FederalTax: decimal.NewFromInt(12000),
				StateTax:   decimal.NewFromInt(3500),
				LocalTax:   decimal.NewFromInt(600),
				FICATax:    decimal.NewFromInt(0),
			},
		},
	}

	result := calc.CalculateMetrics(summary)

	if result.ScenarioName != "Test Scenario" {
		t.Errorf("Expected scenario name 'Test Scenario', got %s", result.ScenarioName)
	}

	if !result.FirstYearNetIncome.Equal(decimal.NewFromInt(100000)) {
		t.Errorf("Expected first year income 100000, got %s", result.FirstYearNetIncome.String())
	}

	if !result.LifetimeIncome.Equal(decimal.NewFromInt(3000000)) {
		t.Errorf("Expected lifetime income 3000000, got %s", result.LifetimeIncome.String())
	}

	if result.TSPLongevity != 25 {
		t.Errorf("Expected TSP longevity 25, got %d", result.TSPLongevity)
	}

	// Check lifetime taxes calculation: 10000 + 3000 + 500 + 7000 + 12000 + 3500 + 600 = 36600
	expectedTaxes := decimal.NewFromInt(36600)
	if !result.LifetimeTaxes.Equal(expectedTaxes) {
		t.Errorf("Expected lifetime taxes %s, got %s", expectedTaxes.String(), result.LifetimeTaxes.String())
	}
}

func TestMetricsCalculator_CalculateComparison(t *testing.T) {
	calc := NewMetricsCalculator()

	base := ComparisonResult{
		ScenarioName:    "Base",
		LifetimeIncome:  decimal.NewFromInt(3000000),
		TSPLongevity:    25,
		LifetimeTaxes:   decimal.NewFromInt(500000),
	}

	scenario := ComparisonResult{
		ScenarioName:    "Alternative",
		LifetimeIncome:  decimal.NewFromInt(3200000),
		TSPLongevity:    28,
		LifetimeTaxes:   decimal.NewFromInt(550000),
	}

	result := calc.CalculateComparison(scenario, base)

	// Check income difference: 3200000 - 3000000 = 200000
	expectedIncomeDiff := decimal.NewFromInt(200000)
	if !result.IncomeDiffFromBase.Equal(expectedIncomeDiff) {
		t.Errorf("Expected income diff %s, got %s", expectedIncomeDiff.String(), result.IncomeDiffFromBase.String())
	}

	// Check percentage: 200000 / 3000000 * 100 = 6.67%
	expectedPct := decimal.NewFromFloat(6.666666666666667)
	if result.IncomePctFromBase.Sub(expectedPct).Abs().GreaterThan(decimal.NewFromFloat(0.01)) {
		t.Errorf("Expected income pct ~6.67, got %s", result.IncomePctFromBase.String())
	}

	// Check TSP longevity difference: 28 - 25 = 3
	if result.TSPLongevityDiff != 3 {
		t.Errorf("Expected TSP longevity diff 3, got %d", result.TSPLongevityDiff)
	}

	// Check tax difference: 550000 - 500000 = 50000
	expectedTaxDiff := decimal.NewFromInt(50000)
	if !result.TaxDiffFromBase.Equal(expectedTaxDiff) {
		t.Errorf("Expected tax diff %s, got %s", expectedTaxDiff.String(), result.TaxDiffFromBase.String())
	}
}

func TestGenerateRecommendations(t *testing.T) {
	baseResult := &ComparisonResult{
		ScenarioName:    "Base",
		LifetimeIncome:  decimal.NewFromInt(3000000),
		TSPLongevity:    25,
		LifetimeTaxes:   decimal.NewFromInt(500000),
	}

	alt1 := ComparisonResult{
		ScenarioName:       "Alternative 1",
		LifetimeIncome:     decimal.NewFromInt(3200000),
		IncomeDiffFromBase: decimal.NewFromInt(200000),
		TSPLongevity:       25,
		TSPLongevityDiff:   0,
		LifetimeTaxes:      decimal.NewFromInt(500000),
		TaxDiffFromBase:    decimal.Zero,
	}

	alt2 := ComparisonResult{
		ScenarioName:       "Alternative 2",
		LifetimeIncome:     decimal.NewFromInt(3100000),
		IncomeDiffFromBase: decimal.NewFromInt(100000),
		TSPLongevity:       28,
		TSPLongevityDiff:   3,
		LifetimeTaxes:      decimal.NewFromInt(450000),
		TaxDiffFromBase:    decimal.NewFromInt(-50000),
	}

	compSet := &ComparisonSet{
		BaseScenarioName:   "Base",
		BaseResult:         baseResult,
		AlternativeResults: []ComparisonResult{alt1, alt2},
	}

	recommendations := GenerateRecommendations(compSet)

	if len(recommendations) == 0 {
		t.Fatal("Expected recommendations, got none")
	}

	// Should recommend alt1 for best income
	foundIncomeRec := false
	for _, rec := range recommendations {
		if contains(rec, "Alternative 1") && contains(rec, "Best Income") {
			foundIncomeRec = true
		}
	}

	if !foundIncomeRec {
		t.Error("Expected recommendation for best income (Alternative 1)")
	}

	// Should recommend alt2 for best longevity
	foundLongevityRec := false
	for _, rec := range recommendations {
		if contains(rec, "Alternative 2") && contains(rec, "Best Longevity") {
			foundLongevityRec = true
		}
	}

	if !foundLongevityRec {
		t.Error("Expected recommendation for best longevity (Alternative 2)")
	}

	// Should recommend alt2 for lowest taxes
	foundTaxRec := false
	for _, rec := range recommendations {
		if contains(rec, "Alternative 2") && contains(rec, "Lowest Taxes") {
			foundTaxRec = true
		}
	}

	if !foundTaxRec {
		t.Error("Expected recommendation for lowest taxes (Alternative 2)")
	}
}

func TestGenerateRecommendations_EmptyAlternatives(t *testing.T) {
	baseResult := &ComparisonResult{
		ScenarioName:   "Base",
		LifetimeIncome: decimal.NewFromInt(3000000),
	}

	compSet := &ComparisonSet{
		BaseScenarioName:   "Base",
		BaseResult:         baseResult,
		AlternativeResults: []ComparisonResult{},
	}

	recommendations := GenerateRecommendations(compSet)

	if len(recommendations) != 0 {
		t.Errorf("Expected no recommendations, got %d", len(recommendations))
	}
}

func TestGenerateRecommendations_NoBetterThanBase(t *testing.T) {
	baseResult := &ComparisonResult{
		ScenarioName:    "Base",
		LifetimeIncome:  decimal.NewFromInt(3000000),
		TSPLongevity:    30,
		LifetimeTaxes:   decimal.NewFromInt(400000),
	}

	alt1 := ComparisonResult{
		ScenarioName:       "Alternative 1",
		LifetimeIncome:     decimal.NewFromInt(2900000),
		IncomeDiffFromBase: decimal.NewFromInt(-100000),
		TSPLongevity:       28,
		TSPLongevityDiff:   -2,
		LifetimeTaxes:      decimal.NewFromInt(450000),
		TaxDiffFromBase:    decimal.NewFromInt(50000),
	}

	compSet := &ComparisonSet{
		BaseScenarioName:   "Base",
		BaseResult:         baseResult,
		AlternativeResults: []ComparisonResult{alt1},
	}

	recommendations := GenerateRecommendations(compSet)

	// Should not recommend alternatives that are worse than base
	if len(recommendations) > 0 {
		t.Logf("Recommendations: %v", recommendations)
		t.Error("Expected no recommendations when alternatives are worse than base")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInMiddle(s, substr)))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
