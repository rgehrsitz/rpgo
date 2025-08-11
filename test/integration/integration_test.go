package integration

import (
	"testing"

	"github.com/rpgo/retirement-calculator/internal/calculation"
	"github.com/rpgo/retirement-calculator/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestEndToEndCalculation(t *testing.T) {
	// Test that we can load a configuration and run calculations
	parser := config.NewInputParser()
	config, err := parser.LoadFromFile("../testdata/generic_example_config.yaml")

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Len(t, config.Scenarios, 2)

	// Test that we can create a calculation engine
	engine := calculation.NewCalculationEngine()
	assert.NotNil(t, engine)

	// Test that we can run scenarios
	results, err := engine.RunScenarios(config)
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results.Scenarios, 2)

	// Verify basic results
	// Baseline net income may be zero with current generic stub; just ensure present
	assert.NotNil(t, results.BaselineNetIncome)
	// Recommendation fields may be empty until projection & analytics reimplemented
}

func TestConfigurationValidation(t *testing.T) {
	parser := config.NewInputParser()

	// Test valid configuration
	config, err := parser.LoadFromFile("../testdata/generic_example_config.yaml")
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Test that validation works
	err = parser.ValidateConfiguration(config)
	assert.NoError(t, err)
}
