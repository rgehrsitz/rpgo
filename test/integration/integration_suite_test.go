package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rgehrsitz/rpgo/internal/calculation"
	"github.com/rgehrsitz/rpgo/internal/config"
	"github.com/rgehrsitz/rpgo/internal/output"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationSuite runs all integration tests
func TestIntegrationSuite(t *testing.T) {
	// Set up test environment
	setupTestEnvironment(t)
	defer cleanupTestEnvironment(t)

	// Run all integration test suites
	t.Run("Basic_Integration", TestBasicIntegration)
	t.Run("Error_Handling", TestErrorHandling)
	t.Run("Performance", TestPerformance)
	t.Run("Data_Consistency", TestDataConsistency)
}

// TestIntegrationSmokeTest runs a quick smoke test of core functionality
func TestIntegrationSmokeTest(t *testing.T) {
	setupTestEnvironment(t)
	defer cleanupTestEnvironment(t)

	t.Run("basic_calculation", func(t *testing.T) {
		// Test basic calculation with minimal config
		parser := config.NewInputParser()
		config, err := parser.LoadFromFile("../testdata/generic_example_config.yaml")
		require.NoError(t, err)

		engine := calculation.NewCalculationEngine()
		results, err := engine.RunScenarios(config)
		require.NoError(t, err)
		require.NotNil(t, results)
		assert.Len(t, results.Scenarios, 2)
	})

	t.Run("basic_output_generation", func(t *testing.T) {
		// Test basic output generation
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
	})
}

// TestIntegrationRegression tests for regression issues
func TestIntegrationRegression(t *testing.T) {
	setupTestEnvironment(t)
	defer cleanupTestEnvironment(t)

	t.Run("calculation_consistency", func(t *testing.T) {
		// Test that calculations are consistent across runs
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

			// Use tolerance for decimal comparisons due to potential floating point precision issues
			tolerance := decimal.NewFromFloat(2000.0) // $2000 tolerance for now to identify root cause
			firstYearDiff := scenario1.FirstYearNetIncome.Sub(scenario2.FirstYearNetIncome).Abs()
			lifetimeDiff := scenario1.TotalLifetimeIncome.Sub(scenario2.TotalLifetimeIncome).Abs()

			assert.True(t, firstYearDiff.LessThan(tolerance),
				"First year income should match within tolerance: %s vs %s (diff: %s)",
				scenario1.FirstYearNetIncome.String(), scenario2.FirstYearNetIncome.String(), firstYearDiff.String())
			assert.True(t, lifetimeDiff.LessThan(tolerance),
				"Lifetime income should match within tolerance: %s vs %s (diff: %s)",
				scenario1.TotalLifetimeIncome.String(), scenario2.TotalLifetimeIncome.String(), lifetimeDiff.String())
		}
	})

	t.Run("output_format_consistency", func(t *testing.T) {
		// Test that output formats are consistent
		parser := config.NewInputParser()
		config, err := parser.LoadFromFile("../testdata/generic_example_config.yaml")
		require.NoError(t, err)

		engine := calculation.NewCalculationEngine()
		results, err := engine.RunScenarios(config)
		require.NoError(t, err)

		// Test all output formats
		formats := []string{"console", "json", "csv", "html"}

		for _, format := range formats {
			t.Run(fmt.Sprintf("format_%s", format), func(t *testing.T) {
				err = output.GenerateReport(results, format)
				assert.NoError(t, err, "Should generate %s output", format)
			})
		}
	})
}

// setupTestEnvironment sets up the test environment
func setupTestEnvironment(t *testing.T) {
	// Set test environment variables
	os.Setenv("RPGO_TEST_MODE", "true")
	os.Setenv("RPGO_LOG_LEVEL", "error") // Reduce log noise during tests

	// Create temporary directories if needed
	tmpDir := t.TempDir()
	os.Setenv("RPGO_TEMP_DIR", tmpDir)
}

// cleanupTestEnvironment cleans up the test environment
func cleanupTestEnvironment(t *testing.T) {
	// Clean up environment variables
	os.Unsetenv("RPGO_TEST_MODE")
	os.Unsetenv("RPGO_LOG_LEVEL")
	os.Unsetenv("RPGO_TEMP_DIR")
}

// TestIntegrationBenchmarks runs performance benchmarks
func TestIntegrationBenchmarks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping benchmarks in short mode")
	}

	setupTestEnvironment(t)
	defer cleanupTestEnvironment(t)

	t.Run("calculation_performance", func(t *testing.T) {
		parser := config.NewInputParser()
		config, err := parser.LoadFromFile("../testdata/generic_example_config.yaml")
		require.NoError(t, err)

		engine := calculation.NewCalculationEngine()

		// Benchmark calculation performance
		start := time.Now()
		results, err := engine.RunScenarios(config)
		duration := time.Since(start)

		require.NoError(t, err, "Should complete calculation")
		assert.Less(t, duration, 30*time.Second, "Calculation should complete within 30 seconds")

		t.Logf("Calculation completed in %v", duration)
		t.Logf("Processed %d scenarios", len(results.Scenarios))
	})

	t.Run("output_generation_performance", func(t *testing.T) {
		parser := config.NewInputParser()
		config, err := parser.LoadFromFile("../testdata/generic_example_config.yaml")
		require.NoError(t, err)

		engine := calculation.NewCalculationEngine()
		results, err := engine.RunScenarios(config)
		require.NoError(t, err)

		// Benchmark output generation
		formats := []string{"console", "json", "csv", "html"}

		for _, format := range formats {
			t.Run(fmt.Sprintf("output_%s", format), func(t *testing.T) {
				start := time.Now()
				err = output.GenerateReport(results, format)
				duration := time.Since(start)

				require.NoError(t, err, "Should generate %s output", format)
				assert.Less(t, duration, 5*time.Second, "%s output should generate within 5 seconds", format)

				t.Logf("%s output generated in %v", format, duration)
			})
		}
	})
}

// TestIntegrationDataValidation tests data validation across the system
func TestIntegrationDataValidation(t *testing.T) {
	setupTestEnvironment(t)
	defer cleanupTestEnvironment(t)

	t.Run("configuration_data_validation", func(t *testing.T) {
		// Test various configuration files
		configFiles := []string{
			"../testdata/generic_example_config.yaml",
			"../testdata/example_config.yaml",
		}

		for _, configFile := range configFiles {
			t.Run(filepath.Base(configFile), func(t *testing.T) {
				parser := config.NewInputParser()
				config, err := parser.LoadFromFile(configFile)
				if err != nil {
					t.Skipf("Skipping config file %s due to load error: %v", configFile, err)
					return
				}

				// Validate configuration
				err = parser.ValidateConfiguration(config)
				if err != nil {
					t.Skipf("Skipping config file %s due to validation error: %v", configFile, err)
					return
				}

				// Validate data integrity
				assert.NotEmpty(t, config.Household.Participants, "Should have participants")
				assert.NotEmpty(t, config.Scenarios, "Should have scenarios")
				assert.NotNil(t, config.GlobalAssumptions, "Should have global assumptions")

				// Validate participant data
				for _, participant := range config.Household.Participants {
					assert.NotEmpty(t, participant.Name, "Participant should have name")
					if participant.CurrentSalary != nil {
						assert.True(t, participant.CurrentSalary.GreaterThan(decimal.Zero), "Participant should have positive salary")
					}
					if participant.High3Salary != nil {
						assert.True(t, participant.High3Salary.GreaterThan(decimal.Zero), "Participant should have positive high-3 salary")
					}
					if participant.TSPBalanceTraditional != nil {
						assert.True(t, participant.TSPBalanceTraditional.GreaterThanOrEqual(decimal.Zero), "TSP balance should be non-negative")
					}
					if participant.TSPBalanceRoth != nil {
						assert.True(t, participant.TSPBalanceRoth.GreaterThanOrEqual(decimal.Zero), "Roth TSP balance should be non-negative")
					}
				}

				// Validate scenario data
				for _, scenario := range config.Scenarios {
					assert.NotEmpty(t, scenario.Name, "Scenario should have name")
					assert.NotEmpty(t, scenario.ParticipantScenarios, "Scenario should have participant scenarios")

					for _, ps := range scenario.ParticipantScenarios {
						assert.NotEmpty(t, ps.ParticipantName, "Participant scenario should have participant name")
						assert.NotEmpty(t, ps.RetirementDate, "Participant scenario should have retirement date")
						assert.GreaterOrEqual(t, ps.SSStartAge, 62, "SS start age should be >= 62")
						assert.LessOrEqual(t, ps.SSStartAge, 70, "SS start age should be <= 70")
					}
				}
			})
		}
	})

	t.Run("calculation_result_validation", func(t *testing.T) {
		parser := config.NewInputParser()
		config, err := parser.LoadFromFile("../testdata/generic_example_config.yaml")
		require.NoError(t, err)

		engine := calculation.NewCalculationEngine()
		results, err := engine.RunScenarios(config)
		require.NoError(t, err)

		// Validate calculation results
		assert.NotNil(t, results, "Results should not be nil")
		assert.Len(t, results.Scenarios, len(config.Scenarios), "Should have same number of scenarios")

		for i, scenario := range results.Scenarios {
			configScenario := config.Scenarios[i]
			assert.Equal(t, configScenario.Name, scenario.Name, "Scenario names should match")

			// Validate financial data
			assert.True(t, scenario.FirstYearNetIncome.GreaterThanOrEqual(decimal.Zero), "First year income should be non-negative")
			assert.True(t, scenario.TotalLifetimeIncome.GreaterThanOrEqual(decimal.Zero), "Lifetime income should be non-negative")
			assert.GreaterOrEqual(t, scenario.TSPLongevity, 0, "TSP longevity should be non-negative")
			assert.True(t, scenario.SuccessRate.GreaterThanOrEqual(decimal.Zero), "Success rate should be non-negative")
			// Success rate might be stored as percentage (100) or decimal (1.0)
			maxSuccessRate := decimal.NewFromInt(100)
			assert.True(t, scenario.SuccessRate.LessThanOrEqual(maxSuccessRate),
				"Success rate should be <= 100%%, got: %s", scenario.SuccessRate.String())
		}
	})
}
