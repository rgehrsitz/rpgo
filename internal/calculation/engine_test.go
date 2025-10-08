package calculation

import (
	"context"
	"testing"
	"time"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestNewCalculationEngine(t *testing.T) {
	engine := NewCalculationEngine()

	assert.NotNil(t, engine, "Should create engine")
	assert.NotNil(t, engine.TaxCalc, "Should initialize tax calculator")
	assert.NotNil(t, engine.MedicareCalc, "Should initialize medicare calculator")
	assert.NotNil(t, engine.NetIncomeCalc, "Should initialize net income calculator")
	assert.NotNil(t, engine.Logger, "Should initialize logger")
}

func TestCalculationEngine_SetLogger(t *testing.T) {
	engine := NewCalculationEngine()

	// Test setting a custom logger
	customLogger := &TestLogger{}
	engine.SetLogger(customLogger)

	assert.Equal(t, customLogger, engine.Logger, "Should set custom logger")

	// Test setting nil logger (should use no-op logger)
	engine.SetLogger(nil)

	assert.NotNil(t, engine.Logger, "Should not be nil")
	assert.IsType(t, NopLogger{}, engine.Logger, "Should be no-op logger")
}

func TestCalculationEngine_RunScenarioAuto_InvalidIndex(t *testing.T) {
	engine := NewCalculationEngine()

	config := &domain.Configuration{
		Scenarios: []domain.GenericScenario{
			{Name: "scenario1"},
		},
	}

	// Test with invalid index
	result, err := engine.RunScenarioAuto(context.Background(), config, 5)

	assert.Error(t, err, "Should error for invalid index")
	assert.Nil(t, result, "Should return nil result")
	assert.Contains(t, err.Error(), "scenario index 5 out of range", "Should have specific error message")
}

func TestCalculationEngine_RunScenarioAuto_ValidIndex(t *testing.T) {
	engine := NewCalculationEngine()

	config := &domain.Configuration{
		Scenarios: []domain.GenericScenario{
			{
				Name: "test-scenario",
				ParticipantScenarios: map[string]domain.ParticipantScenario{
					"participant1": {
						RetirementDate: timePtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
					},
				},
			},
		},
		Household: &domain.Household{
			Participants: []domain.Participant{
				{
					Name:                   "participant1",
					BirthDate:              time.Date(1965, 1, 1, 0, 0, 0, 0, time.UTC),
					HireDate:               timePtr(time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)),
					CurrentSalary:          decimalPtr(decimal.NewFromInt(100000)),
					TSPBalanceTraditional:  decimalPtr(decimal.NewFromInt(500000)),
					TSPBalanceRoth:         decimalPtr(decimal.NewFromInt(100000)),
					TSPContributionPercent: decimalPtr(decimal.NewFromFloat(0.05)),
					IsFederal:              true,
				},
			},
			FilingStatus: "married_filing_jointly",
		},
		GlobalAssumptions: domain.GlobalAssumptions{
			InflationRate:           decimal.NewFromFloat(0.025),
			TSPReturnPreRetirement:  decimal.NewFromFloat(0.06),
			TSPReturnPostRetirement: decimal.NewFromFloat(0.04),
			COLAGeneralRate:         decimal.NewFromFloat(0.02),
			FEHBPremiumInflation:    decimal.NewFromFloat(0.06),
			ProjectionYears:         30,
			FederalRules: domain.FederalRules{
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
	}

	result, err := engine.RunScenarioAuto(context.Background(), config, 0)

	assert.NoError(t, err, "Should not error for valid index")
	assert.NotNil(t, result, "Should return result")
	assert.Equal(t, "test-scenario", result.Name, "Should have correct scenario name")
}

// Helper functions for creating pointers
func timePtr(t time.Time) *time.Time {
	return &t
}

// TestLogger is a simple logger for testing
type TestLogger struct {
	messages []string
}

func (tl *TestLogger) Debugf(format string, args ...interface{}) {
	tl.messages = append(tl.messages, "DEBUG: "+format)
}

func (tl *TestLogger) Infof(format string, args ...interface{}) {
	tl.messages = append(tl.messages, "INFO: "+format)
}

func (tl *TestLogger) Warnf(format string, args ...interface{}) {
	tl.messages = append(tl.messages, "WARN: "+format)
}

func (tl *TestLogger) Errorf(format string, args ...interface{}) {
	tl.messages = append(tl.messages, "ERROR: "+format)
}
