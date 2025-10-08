package compare

import (
	"testing"
	"time"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

func TestTableFormatter_Format(t *testing.T) {
	formatter := &TableFormatter{}

	compSet := &ComparisonSet{
		BaseScenarioName: "Base Scenario",
		ConfigPath:       "/path/to/config.yaml",
		BaseResult: &ComparisonResult{
			ScenarioName:       "Base Scenario",
			FirstYearNetIncome: decimal.NewFromInt(100000),
			LifetimeIncome:     decimal.NewFromInt(3000000),
			TSPLongevity:       25,
			FinalTSPBalance:    decimal.NewFromInt(500000),
			LifetimeTaxes:      decimal.NewFromInt(500000),
		},
		AlternativeResults: []ComparisonResult{
			{
				ScenarioName:       "Alternative 1",
				FirstYearNetIncome: decimal.NewFromInt(110000),
				LifetimeIncome:     decimal.NewFromInt(3200000),
				TSPLongevity:       28,
				FinalTSPBalance:    decimal.NewFromInt(600000),
				LifetimeTaxes:      decimal.NewFromInt(450000),
				IncomeDiffFromBase: decimal.NewFromInt(200000),
				IncomePctFromBase:  decimal.NewFromFloat(6.67),
				TSPLongevityDiff:   3,
				TaxDiffFromBase:    decimal.NewFromInt(-50000),
			},
		},
		Recommendations: []string{
			"Best Income: Alternative 1 provides $200000 more lifetime income than base scenario",
			"Best Longevity: Alternative 1 extends TSP by 3 years",
			"Lowest Taxes: Alternative 1 saves $50000 in lifetime taxes",
		},
	}

	result := formatter.Format(compSet)

	if result == "" {
		t.Fatal("Expected formatted output, got empty string")
	}

	// Check that key elements are present
	if !contains(result, "RETIREMENT SCENARIO COMPARISON") {
		t.Error("Expected header in output")
	}

	if !contains(result, "Base Scenario: Base Scenario") {
		t.Error("Expected base scenario name in output")
	}

	if !contains(result, "Configuration: /path/to/config.yaml") {
		t.Error("Expected config path in output")
	}

	if !contains(result, "Base Scenario") {
		t.Error("Expected base scenario in table")
	}

	if !contains(result, "Alternative 1") {
		t.Error("Expected alternative scenario in table")
	}

	if !contains(result, "RECOMMENDATIONS") {
		t.Error("Expected recommendations section")
	}
}

func TestTableFormatter_Format_EmptyAlternatives(t *testing.T) {
	formatter := &TableFormatter{}

	compSet := &ComparisonSet{
		BaseScenarioName: "Base Scenario",
		ConfigPath:       "/path/to/config.yaml",
		BaseResult: &ComparisonResult{
			ScenarioName:       "Base Scenario",
			FirstYearNetIncome: decimal.NewFromInt(100000),
			LifetimeIncome:     decimal.NewFromInt(3000000),
			TSPLongevity:       25,
			FinalTSPBalance:    decimal.NewFromInt(500000),
			LifetimeTaxes:      decimal.NewFromInt(500000),
		},
		AlternativeResults: []ComparisonResult{},
		Recommendations:    []string{},
	}

	result := formatter.Format(compSet)

	if result == "" {
		t.Fatal("Expected formatted output, got empty string")
	}

	// Should still have header and base scenario
	if !contains(result, "RETIREMENT SCENARIO COMPARISON") {
		t.Error("Expected header in output")
	}

	if !contains(result, "Base Scenario") {
		t.Error("Expected base scenario in table")
	}

	// Should not have alternative scenarios
	if contains(result, "Alternative") {
		t.Error("Should not have alternative scenarios in output")
	}
}

func TestTableFormatter_formatRow(t *testing.T) {
	formatter := &TableFormatter{}

	result := &ComparisonResult{
		ScenarioName:       "Test Scenario",
		FirstYearNetIncome: decimal.NewFromInt(100000),
		LifetimeIncome:     decimal.NewFromInt(3000000),
		TSPLongevity:       25,
		FinalTSPBalance:    decimal.NewFromInt(500000),
		IncomeDiffFromBase: decimal.NewFromInt(200000),
		IncomePctFromBase:  decimal.NewFromFloat(6.67),
		TSPLongevityDiff:   3,
		TaxDiffFromBase:    decimal.NewFromInt(-50000),
	}

	// Test base scenario row
	baseRow := formatter.formatRow(result, 25, 15, true)
	if baseRow == "" {
		t.Fatal("Expected formatted row, got empty string")
	}

	if !contains(baseRow, "Test Scenario") {
		t.Error("Expected scenario name in row")
	}

	// Test alternative scenario row
	altRow := formatter.formatRow(result, 25, 15, false)
	if altRow == "" {
		t.Fatal("Expected formatted row, got empty string")
	}

	if !contains(altRow, "Test Scenario") {
		t.Error("Expected scenario name in row")
	}

	// Debug: print the actual output to see what it contains
	t.Logf("Base row output: %q", baseRow)
	t.Logf("Alt row output: %q", altRow)
}

func TestCSVFormatter_Format(t *testing.T) {
	formatter := &CSVFormatter{}

	compSet := &ComparisonSet{
		BaseScenarioName: "Base Scenario",
		ConfigPath:       "/path/to/config.yaml",
		BaseResult: &ComparisonResult{
			ScenarioName:       "Base Scenario",
			FirstYearNetIncome: decimal.NewFromInt(100000),
			LifetimeIncome:     decimal.NewFromInt(3000000),
			TSPLongevity:       25,
			FinalTSPBalance:    decimal.NewFromInt(500000),
			LifetimeTaxes:      decimal.NewFromInt(500000),
		},
		AlternativeResults: []ComparisonResult{
			{
				ScenarioName:       "Alternative 1",
				FirstYearNetIncome: decimal.NewFromInt(110000),
				LifetimeIncome:     decimal.NewFromInt(3200000),
				TSPLongevity:       28,
				FinalTSPBalance:    decimal.NewFromInt(600000),
				LifetimeTaxes:      decimal.NewFromInt(450000),
				IncomeDiffFromBase: decimal.NewFromInt(200000),
				IncomePctFromBase:  decimal.NewFromFloat(6.67),
				TSPLongevityDiff:   3,
				TaxDiffFromBase:    decimal.NewFromInt(-50000),
			},
		},
	}

	result, err := formatter.Format(compSet)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == "" {
		t.Fatal("Expected CSV output, got empty string")
	}

	// Check that CSV structure is present
	if !contains(result, "Scenario") {
		t.Error("Expected CSV header")
	}

	if !contains(result, "Base Scenario") {
		t.Error("Expected base scenario in CSV")
	}

	if !contains(result, "Alternative 1") {
		t.Error("Expected alternative scenario in CSV")
	}

	// Check that values are properly formatted
	if !contains(result, "100000") {
		t.Error("Expected first year income value in CSV")
	}

	if !contains(result, "3000000") {
		t.Error("Expected lifetime income value in CSV")
	}
}

func TestJSONFormatter_Format(t *testing.T) {
	formatter := &JSONFormatter{}

	compSet := &ComparisonSet{
		BaseScenarioName: "Base Scenario",
		ConfigPath:       "/path/to/config.yaml",
		BaseResult: &ComparisonResult{
			ScenarioName:       "Base Scenario",
			FirstYearNetIncome: decimal.NewFromInt(100000),
			LifetimeIncome:     decimal.NewFromInt(3000000),
			TSPLongevity:       25,
			FinalTSPBalance:    decimal.NewFromInt(500000),
			LifetimeTaxes:      decimal.NewFromInt(500000),
		},
		AlternativeResults: []ComparisonResult{
			{
				ScenarioName:       "Alternative 1",
				FirstYearNetIncome: decimal.NewFromInt(110000),
				LifetimeIncome:     decimal.NewFromInt(3200000),
				TSPLongevity:       28,
				FinalTSPBalance:    decimal.NewFromInt(600000),
				LifetimeTaxes:      decimal.NewFromInt(450000),
				IncomeDiffFromBase: decimal.NewFromInt(200000),
				IncomePctFromBase:  decimal.NewFromFloat(6.67),
				TSPLongevityDiff:   3,
				TaxDiffFromBase:    decimal.NewFromInt(-50000),
			},
		},
		Recommendations: []string{
			"Best Income: Alternative 1 provides $200000 more lifetime income than base scenario",
		},
	}

	result, err := formatter.Format(compSet)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == "" {
		t.Fatal("Expected JSON output, got empty string")
	}

	// Check that JSON structure is present
	if !contains(result, "\"baseScenarioName\"") {
		t.Error("Expected baseScenarioName field in JSON")
	}

	if !contains(result, "\"Base Scenario\"") {
		t.Error("Expected base scenario name in JSON")
	}

	if !contains(result, "\"alternativeResults\"") {
		t.Error("Expected alternativeResults field in JSON")
	}

	if !contains(result, "\"recommendations\"") {
		t.Error("Expected recommendations field in JSON")
	}
}

func TestComparisonSet_ToScenarioComparison(t *testing.T) {
	compSet := &ComparisonSet{
		BaseScenarioName: "Base Scenario",
		BaseResult: &ComparisonResult{
			ScenarioName:       "Base Scenario",
			FirstYearNetIncome: decimal.NewFromInt(100000),
			Summary: &domain.ScenarioSummary{
				Name: "Base Scenario",
				Projection: []domain.AnnualCashFlow{
					{
						Year: 0,
						Date: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
					},
				},
			},
		},
		AlternativeResults: []ComparisonResult{
			{
				ScenarioName: "Alternative 1",
				Summary: &domain.ScenarioSummary{
					Name: "Alternative 1",
					Projection: []domain.AnnualCashFlow{
						{
							Year: 0,
							Date: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
						},
					},
				},
			},
		},
	}

	result := compSet.ToScenarioComparison()

	if result == nil {
		t.Fatal("Expected ScenarioComparison, got nil")
	}

	if !result.BaselineNetIncome.Equal(decimal.NewFromInt(100000)) {
		t.Errorf("Expected baseline net income 100000, got %s", result.BaselineNetIncome.String())
	}

	if len(result.Scenarios) != 2 {
		t.Errorf("Expected 2 scenarios, got %d", len(result.Scenarios))
	}

	if result.Scenarios[0].Name != "Base Scenario" {
		t.Errorf("Expected first scenario 'Base Scenario', got %s", result.Scenarios[0].Name)
	}

	if result.Scenarios[1].Name != "Alternative 1" {
		t.Errorf("Expected second scenario 'Alternative 1', got %s", result.Scenarios[1].Name)
	}

	if len(result.Assumptions) == 0 {
		t.Error("Expected assumptions to be populated")
	}
}

func TestComparisonSet_ToScenarioComparison_NilBaseResult(t *testing.T) {
	compSet := &ComparisonSet{
		BaseScenarioName: "Base Scenario",
		BaseResult:       nil,
		AlternativeResults: []ComparisonResult{
			{
				ScenarioName: "Alternative 1",
				Summary: &domain.ScenarioSummary{
					Name: "Alternative 1",
				},
			},
		},
	}

	result := compSet.ToScenarioComparison()

	if result == nil {
		t.Fatal("Expected ScenarioComparison, got nil")
	}

	if !result.BaselineNetIncome.IsZero() {
		t.Error("Expected zero baseline net income when base result is nil")
	}

	if len(result.Scenarios) != 1 {
		t.Errorf("Expected 1 scenario, got %d", len(result.Scenarios))
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
