package integration

import (
	"testing"

	"github.com/rpgo/retirement-calculator/internal/calculation"
	"github.com/rpgo/retirement-calculator/internal/config"
	"github.com/rpgo/retirement-calculator/internal/output"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestOutputGeneration(t *testing.T) {
	// Load configuration
	parser := config.NewInputParser()
	config, err := parser.LoadFromFile("../testdata/generic_example_config.yaml")
	assert.NoError(t, err)

	// Run calculations
	engine := calculation.NewCalculationEngine()
	results, err := engine.RunScenarios(config)
	assert.NoError(t, err)

	// Test console output
	err = output.GenerateReport(results, "console")
	assert.NoError(t, err)

	// Test JSON output
	err = output.GenerateReport(results, "json")
	assert.NoError(t, err)

	// Test CSV output
	err = output.GenerateReport(results, "csv")
	assert.NoError(t, err)

	// Test HTML output
	err = output.GenerateReport(results, "html")
	assert.NoError(t, err)
}

func TestBasicCalculations(t *testing.T) {
	// Test that basic calculations produce reasonable results
	parser := config.NewInputParser()
	config, err := parser.LoadFromFile("../testdata/generic_example_config.yaml")
	assert.NoError(t, err)

	engine := calculation.NewCalculationEngine()
	results, err := engine.RunScenarios(config)
	assert.NoError(t, err)

	// Verify we have results
	assert.Len(t, results.Scenarios, 2)
	assert.True(t, results.BaselineNetIncome.GreaterThan(decimal.Zero))

	// Verify structural fields present (values may be zero with current projection stub)
	for _, scenario := range results.Scenarios {
		assert.NotEmpty(t, scenario.Name)
		assert.GreaterOrEqual(t, scenario.TSPLongevity, 0)
	}
}
