package calculation

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

func TestFERSMonteCarloEngine_NewFERSMonteCarloEngine(t *testing.T) {
	config := createTestConfig()

	// Test engine creation
	engine := NewFERSMonteCarloEngine(config, nil)

	if engine == nil {
		t.Fatal("Expected engine to be created, got nil")
	}

	if engine.baseConfig != config {
		t.Error("Expected baseConfig to match input config")
	}

	if engine.historicalData != nil {
		t.Error("Expected historicalData to be nil when not provided")
	}

	if engine.calculationEngine == nil {
		t.Error("Expected calculationEngine to be initialized")
	}

	// Test default configuration values
	if engine.config.NumSimulations != 1000 {
		t.Errorf("Expected NumSimulations to be 1000, got %d", engine.config.NumSimulations)
	}

	if engine.config.ProjectionYears != 30 {
		t.Errorf("Expected ProjectionYears to be 30, got %d", engine.config.ProjectionYears)
	}

	if !engine.config.UseHistorical {
		t.Error("Expected UseHistorical to be true by default")
	}

	// Test variability settings
	expectedTSPVariability := decimal.NewFromFloat(0.05)
	if !engine.config.TSPReturnVariability.Equal(expectedTSPVariability) {
		t.Errorf("Expected TSPReturnVariability to be %v, got %v", expectedTSPVariability, engine.config.TSPReturnVariability)
	}

	expectedInflationVariability := decimal.NewFromFloat(0.01)
	if !engine.config.InflationVariability.Equal(expectedInflationVariability) {
		t.Errorf("Expected InflationVariability to be %v, got %v", expectedInflationVariability, engine.config.InflationVariability)
	}
}

func TestFERSMonteCarloEngine_generateMarketConditions(t *testing.T) {
	config := createTestConfig()
	engine := NewFERSMonteCarloEngine(config, nil)

	// Test market condition generation
	rng := rand.New(rand.NewSource(12345))
	condition := engine.generateMarketConditions(rng)

	// Test TSP returns
	expectedFunds := []string{"C", "S", "I", "F", "G"}
	for _, fund := range expectedFunds {
		if _, exists := condition.TSPReturns[fund]; !exists {
			t.Errorf("Expected TSP return for fund %s to be generated", fund)
		}
	}

	// Test inflation rate
	if condition.InflationRate.IsZero() {
		t.Error("Expected inflation rate to be generated")
	}

	// Test COLA rate
	if condition.COLARate.IsZero() {
		t.Error("Expected COLA rate to be generated")
	}

	// Test FEHB inflation
	if condition.FEHBInflation.IsZero() {
		t.Error("Expected FEHB inflation to be generated")
	}
}

func TestFERSMonteCarloEngine_generateStatisticalTSPReturn(t *testing.T) {
	config := createTestConfig()
	engine := NewFERSMonteCarloEngine(config, nil)

	tests := []struct {
		fund     string
		expected decimal.Decimal
	}{
		{"C", decimal.NewFromFloat(0.10)}, // 10% average
		{"S", decimal.NewFromFloat(0.12)}, // 12% average
		{"I", decimal.NewFromFloat(0.08)}, // 8% average
		{"F", decimal.NewFromFloat(0.05)}, // 5% average
		{"G", decimal.NewFromFloat(0.03)}, // 3% average
	}

	for _, test := range tests {
		t.Run(test.fund, func(t *testing.T) {
			// Run multiple times to test variability
			results := make([]decimal.Decimal, 100)
			rng := rand.New(rand.NewSource(12345))
			for i := 0; i < 100; i++ {
				results[i] = engine.generateStatisticalTSPReturn(test.fund, rng)
			}

			// Check that results have variability (not all the same)
			allSame := true
			for i := 1; i < len(results); i++ {
				if !results[i].Equal(results[0]) {
					allSame = false
					break
				}
			}

			if allSame {
				t.Error("Expected statistical TSP returns to have variability")
			}

			// Check that results are within reasonable bounds (not too extreme)
			for _, result := range results {
				if result.LessThan(decimal.NewFromFloat(-0.5)) {
					t.Errorf("Expected TSP return to be >= -50%%, got %v", result)
				}
				if result.GreaterThan(decimal.NewFromFloat(1.0)) {
					t.Errorf("Expected TSP return to be <= 100%%, got %v", result)
				}
			}
		})
	}
}

func TestFERSMonteCarloEngine_calculateFERSSummary(t *testing.T) {
	config := createTestConfig()
	engine := NewFERSMonteCarloEngine(config, nil)

	// Create test simulations
	simulations := []FERSMonteCarloSimulation{
		{
			SimulationID: 1,
			Success:      true,
			ScenarioSummary: domain.ScenarioSummary{
				TotalLifetimeIncome: decimal.NewFromFloat(4000000),
				TSPLongevity:        25,
				Year5NetIncome:      decimal.NewFromFloat(150000),
				Year10NetIncome:     decimal.NewFromFloat(160000),
			},
		},
		{
			SimulationID: 2,
			Success:      true,
			ScenarioSummary: domain.ScenarioSummary{
				TotalLifetimeIncome: decimal.NewFromFloat(4500000),
				TSPLongevity:        30,
				Year5NetIncome:      decimal.NewFromFloat(170000),
				Year10NetIncome:     decimal.NewFromFloat(180000),
			},
		},
		{
			SimulationID: 3,
			Success:      true,
			ScenarioSummary: domain.ScenarioSummary{
				TotalLifetimeIncome: decimal.NewFromFloat(5000000),
				TSPLongevity:        30,
				Year5NetIncome:      decimal.NewFromFloat(190000),
				Year10NetIncome:     decimal.NewFromFloat(200000),
			},
		},
	}

	marketConditions := []MarketCondition{
		{TSPReturns: map[string]decimal.Decimal{"C": decimal.NewFromFloat(0.1)}},
		{TSPReturns: map[string]decimal.Decimal{"C": decimal.NewFromFloat(0.12)}},
		{TSPReturns: map[string]decimal.Decimal{"C": decimal.NewFromFloat(0.08)}},
	}

	result := engine.calculateFERSSummary(simulations, marketConditions, "Test Scenario")

	if result == nil {
		t.Fatal("Expected summary to be calculated, got nil")
	}

	if result.BaseScenarioName != "Test Scenario" {
		t.Errorf("Expected BaseScenarioName to be 'Test Scenario', got %s", result.BaseScenarioName)
	}

	if result.NumSimulations != 1000 {
		t.Errorf("Expected NumSimulations to be 1000 (from engine config), got %d", result.NumSimulations)
	}

	// Test success rate (should be 100% since all simulations succeeded)
	expectedSuccessRate := decimal.NewFromInt(1)
	if !result.SuccessRate.Equal(expectedSuccessRate) {
		t.Errorf("Expected SuccessRate to be %v, got %v", expectedSuccessRate, result.SuccessRate)
	}

	// Test median lifetime income (should be 4500000 - middle value)
	expectedMedianIncome := decimal.NewFromFloat(4500000)
	if !result.MedianLifetimeIncome.Equal(expectedMedianIncome) {
		t.Errorf("Expected MedianLifetimeIncome to be %v, got %v", expectedMedianIncome, result.MedianLifetimeIncome)
	}

	// Test median TSP longevity (should be 30 - middle value)
	expectedMedianLongevity := 30
	if result.MedianTSPLongevity != expectedMedianLongevity {
		t.Errorf("Expected MedianTSPLongevity to be %d, got %d", expectedMedianLongevity, result.MedianTSPLongevity)
	}

	// Test percentile ranges
	if len(result.PercentileRanges.LifetimeIncome) != 5 {
		t.Errorf("Expected 5 percentile ranges for lifetime income, got %d", len(result.PercentileRanges.LifetimeIncome))
	}

	if len(result.PercentileRanges.TSPLongevity) != 5 {
		t.Errorf("Expected 5 percentile ranges for TSP longevity, got %d", len(result.PercentileRanges.TSPLongevity))
	}
}

func TestFERSMonteCarloEngine_RunFERSMonteCarlo_InvalidScenario(t *testing.T) {
	config := createTestConfig()
	engine := NewFERSMonteCarloEngine(config, nil)

	ctx := context.Background()
	result, err := engine.RunFERSMonteCarlo(ctx, "Non-existent Scenario")

	if err == nil {
		t.Error("Expected error for non-existent scenario, got nil")
	}

	if result != nil {
		t.Error("Expected result to be nil for non-existent scenario")
	}
}

// Helper function to create test configuration
func createTestConfig() *domain.Configuration {
	return &domain.Configuration{
		Household: &domain.Household{
			Participants: []domain.Participant{
				{
					Name:                   "Test Participant",
					BirthDate:              time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
					IsFederal:              true,
					HireDate:               &[]time.Time{time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}[0],
					CurrentSalary:          &[]decimal.Decimal{decimal.NewFromFloat(100000)}[0],
					High3Salary:            &[]decimal.Decimal{decimal.NewFromFloat(100000)}[0],
					TSPBalanceTraditional:  &[]decimal.Decimal{decimal.NewFromFloat(500000)}[0],
					TSPBalanceRoth:         &[]decimal.Decimal{decimal.NewFromFloat(0)}[0],
					TSPContributionPercent: &[]decimal.Decimal{decimal.NewFromFloat(0.05)}[0],
					TSPAllocation: &domain.TSPAllocation{
						CFund: decimal.NewFromFloat(0.6),
						SFund: decimal.NewFromFloat(0.2),
						IFund: decimal.NewFromFloat(0.1),
						FFund: decimal.NewFromFloat(0.1),
						GFund: decimal.Zero,
					},
				},
			},
			FilingStatus: "married_filing_jointly",
		},
		GlobalAssumptions: domain.GlobalAssumptions{
			InflationRate:           decimal.NewFromFloat(0.025),
			FEHBPremiumInflation:    decimal.NewFromFloat(0.06),
			TSPReturnPreRetirement:  decimal.NewFromFloat(0.06),
			TSPReturnPostRetirement: decimal.NewFromFloat(0.04),
			COLAGeneralRate:         decimal.NewFromFloat(0.02),
			ProjectionYears:         30,
			FederalRules: domain.FederalRules{
				// Use minimal required fields for testing
				SocialSecurityTaxThresholds: domain.SocialSecurityTaxThresholds{},
				SocialSecurityRules:         domain.SocialSecurityRules{},
				FERSRules:                   domain.FERSRules{},
				FederalTaxConfig:            domain.FederalTaxConfig{},
				StateLocalTaxConfig:         domain.StateLocalTaxConfig{},
				FICATaxConfig:               domain.FICATaxConfig{},
				MedicareConfig:              domain.MedicareConfig{},
				FEHBConfig:                  domain.FEHBConfig{},
			},
		},
		Scenarios: []domain.GenericScenario{
			{
				Name: "Test Scenario",
				ParticipantScenarios: map[string]domain.ParticipantScenario{
					"Test Participant": {
						ParticipantName: "Test Participant",
						SSStartAge:      62,
					},
				},
			},
		},
	}
}
