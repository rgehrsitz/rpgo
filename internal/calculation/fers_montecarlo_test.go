//go:build legacy_removed
// +build legacy_removed

package calculation

import (
	"testing"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

func TestFERSMonteCarloEngine(t *testing.T) {
	// Create test configuration
	config := createFERSMonteCarloTestConfiguration()

	// Create historical data manager with test data
	hdm := createTestHistoricalDataManager(t)

	// Create FERS Monte Carlo engine
	engine := NewFERSMonteCarloEngine(config, hdm)

	// Test configuration
	mcConfig := FERSMonteCarloConfig{
		BaseConfig:     config,
		NumSimulations: 10, // Small number for testing
		UseHistorical:  true,
		Seed:           12345,
	}

	// Run simulation
	result, err := engine.RunFERSMonteCarlo(mcConfig)
	if err != nil {
		t.Fatalf("Failed to run FERS Monte Carlo simulation: %v", err)
	}

	// Verify basic results
	if result.NumSimulations != 10 {
		t.Errorf("Expected 10 simulations, got %d", result.NumSimulations)
	}

	if len(result.Simulations) != 10 {
		t.Errorf("Expected 10 simulation results, got %d", len(result.Simulations))
	}

	// Verify success rate is reasonable (between 0 and 1)
	if result.SuccessRate.LessThan(decimal.Zero) || result.SuccessRate.GreaterThan(decimal.NewFromInt(1)) {
		t.Errorf("Success rate should be between 0 and 1, got %s", result.SuccessRate.String())
	}

	// Verify median net income is positive
	if result.MedianNetIncome.LessThanOrEqual(decimal.Zero) {
		t.Errorf("Median net income should be positive, got %s", result.MedianNetIncome.String())
	}
}

func TestFERSMonteCarloStatisticalMode(t *testing.T) {
	// Create test configuration
	config := createFERSMonteCarloTestConfiguration()

	// Create historical data manager with test data
	hdm := createTestHistoricalDataManager(t)

	// Create FERS Monte Carlo engine
	engine := NewFERSMonteCarloEngine(config, hdm)

	// Test configuration with statistical mode
	mcConfig := FERSMonteCarloConfig{
		BaseConfig:     config,
		NumSimulations: 5,     // Small number for testing
		UseHistorical:  false, // Use statistical distributions
		Seed:           54321,
	}

	// Run simulation
	result, err := engine.RunFERSMonteCarlo(mcConfig)
	if err != nil {
		t.Fatalf("Failed to run FERS Monte Carlo simulation in statistical mode: %v", err)
	}

	// Verify results
	if result.NumSimulations != 5 {
		t.Errorf("Expected 5 simulations, got %d", result.NumSimulations)
	}

	// Verify that market conditions were generated
	for i, sim := range result.Simulations {
		if sim.MarketConditions.Year == 0 {
			t.Errorf("Simulation %d: Market conditions year should not be 0", i)
		}

		// Verify TSP returns were generated
		if len(sim.MarketConditions.TSPReturns) == 0 {
			t.Errorf("Simulation %d: TSP returns should be generated", i)
		}

		// Verify inflation rate is reasonable
		if sim.MarketConditions.InflationRate.LessThan(decimal.NewFromFloat(-0.1)) ||
			sim.MarketConditions.InflationRate.GreaterThan(decimal.NewFromFloat(0.2)) {
			t.Errorf("Simulation %d: Inflation rate should be reasonable, got %s",
				i, sim.MarketConditions.InflationRate.String())
		}
	}
}

func TestFERSMonteCarloMarketConditionGeneration(t *testing.T) {
	// Create test configuration
	config := createFERSMonteCarloTestConfiguration()

	// Create historical data manager with test data
	hdm := createTestHistoricalDataManager(t)

	// Create FERS Monte Carlo engine
	engine := NewFERSMonteCarloEngine(config, hdm)

	// Test historical market condition generation
	historicalMarket := engine.generateHistoricalMarketConditions()

	// Verify historical market conditions
	if historicalMarket.Year < 1990 || historicalMarket.Year > 2023 {
		t.Errorf("Historical year should be between 1990-2023, got %d", historicalMarket.Year)
	}

	if len(historicalMarket.TSPReturns) != 5 {
		t.Errorf("Expected 5 TSP fund returns, got %d", len(historicalMarket.TSPReturns))
	}

	// Test statistical market condition generation
	statisticalMarket := engine.generateStatisticalMarketConditions()

	// Verify statistical market conditions
	if statisticalMarket.Year < 2025 || statisticalMarket.Year > 2055 {
		t.Errorf("Statistical year should be between 2025-2055, got %d", statisticalMarket.Year)
	}

	if len(statisticalMarket.TSPReturns) != 5 {
		t.Errorf("Expected 5 TSP fund returns, got %d", len(statisticalMarket.TSPReturns))
	}
}

func TestFERSMonteCarloStatisticalDistributions(t *testing.T) {
	// Create test configuration
	config := createFERSMonteCarloTestConfiguration()

	// Create historical data manager with test data
	hdm := createTestHistoricalDataManager(t)

	// Create FERS Monte Carlo engine
	engine := NewFERSMonteCarloEngine(config, hdm)

	// Test TSP return generation
	funds := []string{"C", "S", "I", "F", "G"}
	for _, fund := range funds {
		returnRate := engine.generateStatisticalTSPReturn(fund)

		// Verify return rate is reasonable (not extreme)
		if returnRate.LessThan(decimal.NewFromFloat(-0.5)) ||
			returnRate.GreaterThan(decimal.NewFromFloat(1.0)) {
			t.Errorf("TSP return for %s fund should be reasonable, got %s", fund, returnRate.String())
		}
	}

	// Test inflation generation
	inflation := engine.generateStatisticalInflation()
	if inflation.LessThan(decimal.NewFromFloat(-0.1)) ||
		inflation.GreaterThan(decimal.NewFromFloat(0.2)) {
		t.Errorf("Inflation rate should be reasonable, got %s", inflation.String())
	}

	// Test COLA generation
	cola := engine.generateStatisticalCOLA()
	if cola.LessThan(decimal.NewFromFloat(-0.1)) ||
		cola.GreaterThan(decimal.NewFromFloat(0.2)) {
		t.Errorf("COLA rate should be reasonable, got %s", cola.String())
	}

	// Test FEHB increase generation
	fehb := engine.generateStatisticalFEHBIncrease()
	if fehb.LessThan(decimal.NewFromFloat(-0.1)) ||
		fehb.GreaterThan(decimal.NewFromFloat(0.3)) {
		t.Errorf("FEHB increase should be reasonable, got %s", fehb.String())
	}
}

func TestFERSMonteCarloMetricsCalculation(t *testing.T) {
	// Create test configuration
	config := createFERSMonteCarloTestConfiguration()

	// Create historical data manager with test data
	hdm := createTestHistoricalDataManager(t)

	// Create FERS Monte Carlo engine
	engine := NewFERSMonteCarloEngine(config, hdm)

	// Create test simulations
	simulations := []FERSMonteCarloSimulation{
		{
			SimulationID: 1,
			Success:      true,
			NetIncomeMetrics: NetIncomeMetrics{
				FirstYearNetIncome: decimal.NewFromFloat(80000),
				AverageNetIncome:   decimal.NewFromFloat(80000),
			},
			TSPMetrics: TSPMetrics{
				Longevity: 25,
				Depleted:  false,
			},
		},
		{
			SimulationID: 2,
			Success:      false,
			NetIncomeMetrics: NetIncomeMetrics{
				FirstYearNetIncome: decimal.NewFromFloat(60000),
				AverageNetIncome:   decimal.NewFromFloat(60000),
			},
			TSPMetrics: TSPMetrics{
				Longevity: 15,
				Depleted:  true,
			},
		},
		{
			SimulationID: 3,
			Success:      true,
			NetIncomeMetrics: NetIncomeMetrics{
				FirstYearNetIncome: decimal.NewFromFloat(90000),
				AverageNetIncome:   decimal.NewFromFloat(90000),
			},
			TSPMetrics: TSPMetrics{
				Longevity: 30,
				Depleted:  false,
			},
		},
	}

	// Calculate aggregate results
	result := engine.calculateAggregateResults(simulations)

	// Verify results (with tolerance for floating point precision)
	expectedSuccessRate := decimal.NewFromFloat(2.0 / 3.0) // 2 out of 3 successful
	successRateDiff := result.SuccessRate.Sub(expectedSuccessRate).Abs()
	if successRateDiff.GreaterThan(decimal.NewFromFloat(0.0001)) {
		t.Errorf("Expected success rate %s, got %s (diff: %s)", expectedSuccessRate.String(), result.SuccessRate.String(), successRateDiff.String())
	}

	expectedDepletionRate := decimal.NewFromFloat(1.0 / 3.0) // 1 out of 3 depleted
	depletionRateDiff := result.TSPDepletionRate.Sub(expectedDepletionRate).Abs()
	if depletionRateDiff.GreaterThan(decimal.NewFromFloat(0.0001)) {
		t.Errorf("Expected TSP depletion rate %s, got %s (diff: %s)", expectedDepletionRate.String(), result.TSPDepletionRate.String(), depletionRateDiff.String())
	}

	// Verify median net income
	expectedMedian := decimal.NewFromFloat(80000) // Middle value
	if !result.MedianNetIncome.Equal(expectedMedian) {
		t.Errorf("Expected median net income %s, got %s", expectedMedian.String(), result.MedianNetIncome.String())
	}
}

func TestFERSMonteCarloErrorHandling(t *testing.T) {
	// Create test configuration
	config := createFERSMonteCarloTestConfiguration()

	// Create FERS Monte Carlo engine without historical data
	engine := NewFERSMonteCarloEngine(config, nil)

	// Test configuration
	mcConfig := FERSMonteCarloConfig{
		BaseConfig:     config,
		NumSimulations: 5,
		UseHistorical:  true,
	}

	// Run simulation should fail
	_, err := engine.RunFERSMonteCarlo(mcConfig)
	if err == nil {
		t.Error("Expected error when historical data is not loaded")
	}
}

// Helper functions

func createFERSMonteCarloTestConfiguration() *domain.Configuration {
	return &domain.Configuration{
		PersonalDetails: map[string]domain.Employee{
			"robert": {
				Name:                    "Robert",
				CurrentSalary:           decimal.NewFromFloat(100000),
				High3Salary:             decimal.NewFromFloat(100000),
				TSPBalanceTraditional:   decimal.NewFromFloat(500000),
				TSPBalanceRoth:          decimal.NewFromFloat(100000),
				TSPContributionPercent:  decimal.NewFromFloat(0.05),
				SSBenefitFRA:            decimal.NewFromFloat(2500),
				SSBenefit62:             decimal.NewFromFloat(1800),
				SSBenefit70:             decimal.NewFromFloat(3100),
				FEHBPremiumPerPayPeriod: decimal.NewFromFloat(500),
			},
			"dawn": {
				Name:                    "Dawn",
				CurrentSalary:           decimal.NewFromFloat(80000),
				High3Salary:             decimal.NewFromFloat(80000),
				TSPBalanceTraditional:   decimal.NewFromFloat(300000),
				TSPBalanceRoth:          decimal.NewFromFloat(50000),
				TSPContributionPercent:  decimal.NewFromFloat(0.05),
				SSBenefitFRA:            decimal.NewFromFloat(2000),
				SSBenefit62:             decimal.NewFromFloat(1400),
				SSBenefit70:             decimal.NewFromFloat(2500),
				FEHBPremiumPerPayPeriod: decimal.NewFromFloat(400),
			},
		},
		GlobalAssumptions: domain.GlobalAssumptions{
			InflationRate:           decimal.NewFromFloat(0.025),
			FEHBPremiumInflation:    decimal.NewFromFloat(0.05),
			TSPReturnPreRetirement:  decimal.NewFromFloat(0.07),
			TSPReturnPostRetirement: decimal.NewFromFloat(0.05),
			COLAGeneralRate:         decimal.NewFromFloat(0.025),
			ProjectionYears:         25,
			CurrentLocation: domain.Location{
				State:  "PA",
				County: "Allegheny",
			},
		},
		Scenarios: []domain.Scenario{
			{
				Name: "Test Scenario",
				Robert: domain.RetirementScenario{
					EmployeeName:          "robert",
					SSStartAge:            62,
					TSPWithdrawalStrategy: "4_percent_rule",
				},
				Dawn: domain.RetirementScenario{
					EmployeeName:          "dawn",
					SSStartAge:            62,
					TSPWithdrawalStrategy: "4_percent_rule",
				},
			},
		},
	}
}

func createTestHistoricalDataManager(t *testing.T) *HistoricalDataManager {
	// Use existing data directory - try multiple paths
	paths := []string{"./data", "../data", "../../data"}
	var hdm *HistoricalDataManager
	var err error

	for _, path := range paths {
		hdm = NewHistoricalDataManager(path)
		err = hdm.LoadAllData()
		if err == nil {
			return hdm
		}
	}

	t.Fatalf("Failed to load test historical data from any path: %v", err)
	return nil
}
