package integration

import (
	"testing"
	"time"

	"github.com/rgehrsitz/rpgo/internal/calculation"
	"github.com/rgehrsitz/rpgo/internal/config"
	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/rgehrsitz/rpgo/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBasicIntegration tests basic end-to-end functionality
func TestBasicIntegration(t *testing.T) {
	t.Run("configuration_loading", func(t *testing.T) {
		parser := config.NewInputParser()
		config, err := parser.LoadFromFile("../testdata/generic_example_config.yaml")
		require.NoError(t, err, "Should load configuration successfully")
		require.NotNil(t, config, "Configuration should not be nil")

		// Validate basic structure
		assert.NotEmpty(t, config.Household.Participants, "Should have participants")
		assert.NotEmpty(t, config.Scenarios, "Should have scenarios")
		assert.NotNil(t, config.GlobalAssumptions, "Should have global assumptions")
	})

	t.Run("calculation_engine", func(t *testing.T) {
		parser := config.NewInputParser()
		config, err := parser.LoadFromFile("../testdata/generic_example_config.yaml")
		require.NoError(t, err)

		engine := calculation.NewCalculationEngine()
		require.NotNil(t, engine, "Calculation engine should not be nil")

		results, err := engine.RunScenarios(config)
		require.NoError(t, err, "Should run scenarios successfully")
		require.NotNil(t, results, "Results should not be nil")

		// Validate results structure
		assert.Len(t, results.Scenarios, len(config.Scenarios), "Should have same number of scenarios")
		assert.NotNil(t, results.BaselineNetIncome, "Should have baseline net income")

		// Validate scenario results
		for i, scenario := range results.Scenarios {
			configScenario := config.Scenarios[i]
			assert.Equal(t, configScenario.Name, scenario.Name, "Scenario names should match")
			assert.NotNil(t, scenario.FirstYearNetIncome, "Should have first year net income")
			assert.NotNil(t, scenario.TotalLifetimeIncome, "Should have total lifetime income")
			assert.GreaterOrEqual(t, scenario.TSPLongevity, 0, "TSP longevity should be non-negative")
		}
	})

	t.Run("output_generation", func(t *testing.T) {
		parser := config.NewInputParser()
		config, err := parser.LoadFromFile("../testdata/generic_example_config.yaml")
		require.NoError(t, err)

		engine := calculation.NewCalculationEngine()
		results, err := engine.RunScenarios(config)
		require.NoError(t, err)

		// Test console output
		err = output.GenerateReport(results, "console")
		assert.NoError(t, err, "Should generate console output")

		// Test JSON output
		err = output.GenerateReport(results, "json")
		assert.NoError(t, err, "Should generate JSON output")

		// Test CSV output
		err = output.GenerateReport(results, "csv")
		assert.NoError(t, err, "Should generate CSV output")

		// Test HTML output
		err = output.GenerateReport(results, "html")
		assert.NoError(t, err, "Should generate HTML output")
	})

	t.Run("configuration_validation", func(t *testing.T) {
		parser := config.NewInputParser()
		config, err := parser.LoadFromFile("../testdata/generic_example_config.yaml")
		require.NoError(t, err)

		// Test validation
		err = parser.ValidateConfiguration(config)
		assert.NoError(t, err, "Valid configuration should pass validation")
	})
}

// TestErrorHandling tests error conditions
func TestErrorHandling(t *testing.T) {
	t.Run("missing_config_file", func(t *testing.T) {
		parser := config.NewInputParser()
		_, err := parser.LoadFromFile("nonexistent.yaml")
		assert.Error(t, err, "Should fail for missing config file")
	})

	t.Run("invalid_config_structure", func(t *testing.T) {
		parser := config.NewInputParser()

		// Create invalid config - use domain.Configuration instead
		invalidConfig := &domain.Configuration{
			// Missing required fields
		}

		err := parser.ValidateConfiguration(invalidConfig)
		assert.Error(t, err, "Should fail validation for invalid config")
	})
}

// TestPerformance tests basic performance requirements
func TestPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	t.Run("calculation_performance", func(t *testing.T) {
		parser := config.NewInputParser()
		config, err := parser.LoadFromFile("../testdata/generic_example_config.yaml")
		require.NoError(t, err)

		engine := calculation.NewCalculationEngine()

		start := time.Now()
		results, err := engine.RunScenarios(config)
		duration := time.Since(start)

		require.NoError(t, err, "Should complete calculation")
		assert.Less(t, duration, 30*time.Second, "Calculation should complete within 30 seconds")

		t.Logf("Calculation completed in %v", duration)
		t.Logf("Processed %d scenarios", len(results.Scenarios))
	})
}

// TestDataConsistency tests data consistency across operations
func TestDataConsistency(t *testing.T) {
	t.Run("calculation_consistency", func(t *testing.T) {
		parser := config.NewInputParser()
		config, err := parser.LoadFromFile("../testdata/generic_example_config.yaml")
		require.NoError(t, err)

		engine := calculation.NewCalculationEngine()

		// Run calculation twice
		results1, err := engine.RunScenarios(config)
		require.NoError(t, err)

		results2, err := engine.RunScenarios(config)
		require.NoError(t, err)

		// Results should be identical
		assert.Equal(t, len(results1.Scenarios), len(results2.Scenarios), "Should have same number of scenarios")

		for i, scenario1 := range results1.Scenarios {
			scenario2 := results2.Scenarios[i]
			assert.Equal(t, scenario1.Name, scenario2.Name, "Scenario names should match")
			assert.Equal(t, scenario1.FirstYearNetIncome, scenario2.FirstYearNetIncome, "First year income should match")
			assert.Equal(t, scenario1.TotalLifetimeIncome, scenario2.TotalLifetimeIncome, "Lifetime income should match")
		}
	})
}
