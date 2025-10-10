package transform

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestNewTransformRegistry(t *testing.T) {
	registry := NewTransformRegistry()

	assert.NotNil(t, registry, "Should create registry")
	assert.NotNil(t, registry.factories, "Should initialize factories map")
	assert.Greater(t, len(registry.factories), 0, "Should have built-in transforms registered")
}

func TestTransformRegistry_Register(t *testing.T) {
	registry := &TransformRegistry{
		factories: make(map[string]TransformFactory),
	}

	// Register a test factory
	factory := func(params map[string]string) (ScenarioTransform, error) {
		return &PostponeRetirement{Participant: "test", Months: 12}, nil
	}

	registry.Register("test_transform", factory)

	assert.Contains(t, registry.factories, "test_transform", "Should register transform")
	assert.NotNil(t, registry.factories["test_transform"], "Should store factory function")
}

func TestTransformRegistry_Create_UnknownTransform(t *testing.T) {
	registry := NewTransformRegistry()

	transform, err := registry.Create("unknown_transform", map[string]string{})

	assert.Error(t, err, "Should error for unknown transform")
	assert.Nil(t, transform, "Should return nil transform")
	assert.Contains(t, err.Error(), "unknown transform", "Should have specific error message")
}

func TestTransformRegistry_Create_ValidTransform(t *testing.T) {
	registry := NewTransformRegistry()

	params := map[string]string{
		"participant": "Alice",
		"months":      "12",
	}

	transform, err := registry.Create("postpone_retirement", params)

	assert.NoError(t, err, "Should not error for valid transform")
	assert.NotNil(t, transform, "Should return transform")
	assert.Equal(t, "postpone_retirement", transform.Name(), "Should have correct name")
}

func TestTransformRegistry_List(t *testing.T) {
	registry := NewTransformRegistry()

	transforms := registry.List()

	assert.NotEmpty(t, transforms, "Should list transforms")
	assert.Contains(t, transforms, "postpone_retirement", "Should include postpone_retirement")
	assert.Contains(t, transforms, "delay_ss", "Should include delay_ss")
	assert.Contains(t, transforms, "modify_tsp_strategy", "Should include modify_tsp_strategy")
}

func TestApplyTransforms_NilTransforms(t *testing.T) {
	base := createTestScenario()

	result, err := ApplyTransforms(base, nil)

	assert.NoError(t, err, "Should not error for nil transforms")
	assert.NotNil(t, result, "Should return result")
	assert.Equal(t, base.Name, result.Name, "Should return deep copy of base")
}

func TestApplyTransforms_NilTransformInList(t *testing.T) {
	base := createTestScenario()

	transforms := []ScenarioTransform{
		&PostponeRetirement{Participant: "Alice", Months: 12},
		nil,
	}

	result, err := ApplyTransforms(base, transforms)

	assert.Error(t, err, "Should error for nil transform in list")
	assert.Nil(t, result, "Should return nil result")
	assert.Contains(t, err.Error(), "transform at index 1 is nil", "Should have specific error message")
}

func TestApplyTransforms_TransformValidationFails(t *testing.T) {
	base := createTestScenario()

	// Create a transform that will fail validation
	transform := &PostponeRetirement{
		Participant: "UnknownParticipant", // This participant doesn't exist
		Months:      12,
	}

	transforms := []ScenarioTransform{transform}

	result, err := ApplyTransforms(base, transforms)

	assert.Error(t, err, "Should error when transform validation fails")
	assert.Nil(t, result, "Should return nil result")
}

func TestApplyTransforms_Success(t *testing.T) {
	base := createTestScenario()

	transforms := []ScenarioTransform{
		&PostponeRetirement{Participant: "Alice", Months: 12},
		&DelaySSClaim{Participant: "Alice", NewAge: 65},
	}

	result, err := ApplyTransforms(base, transforms)

	assert.NoError(t, err, "Should not error for valid transforms")
	assert.NotNil(t, result, "Should return result")
	assert.Equal(t, base.Name, result.Name, "Should preserve scenario name")

	// Check that transforms were applied
	aliceScenario, exists := result.ParticipantScenarios["Alice"]
	assert.True(t, exists, "Should have Alice scenario")

	// Check retirement date was postponed (original was 2027-06-30, should be 2028-06-30)
	expectedDate := time.Date(2028, 6, 30, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expectedDate, *aliceScenario.RetirementDate, "Should postpone retirement date")

	// Check SS age was changed
	assert.Equal(t, 65, aliceScenario.SSStartAge, "Should change SS start age")
}

func TestPostponeRetirement_Validate_UnknownParticipant(t *testing.T) {
	base := createTestScenario()

	transform := &PostponeRetirement{
		Participant: "UnknownParticipant",
		Months:      12,
	}

	err := transform.Validate(base)

	assert.Error(t, err, "Should error for unknown participant")
	assert.Contains(t, err.Error(), "participant UnknownParticipant not found in scenario", "Should have specific error message")
}

func TestPostponeRetirement_Validate_NegativeMonths(t *testing.T) {
	base := createTestScenario()

	transform := &PostponeRetirement{
		Participant: "Alice",
		Months:      -1,
	}

	err := transform.Validate(base)

	assert.Error(t, err, "Should error for negative months")
	assert.Contains(t, err.Error(), "months must be non-negative", "Should have specific error message")
}

func TestDelaySSClaim_Apply(t *testing.T) {
	base := createTestScenario()

	transform := &DelaySSClaim{
		Participant: "Alice",
		NewAge:      65,
	}

	result, err := transform.Apply(base)

	assert.NoError(t, err, "Should not error")
	assert.NotNil(t, result, "Should return result")

	aliceScenario, exists := result.ParticipantScenarios["Alice"]
	assert.True(t, exists, "Should have Alice scenario")

	assert.Equal(t, 65, aliceScenario.SSStartAge, "Should change SS start age to 65")
}

func TestDelaySSClaim_Validate_InvalidAge(t *testing.T) {
	base := createTestScenario()

	transform := &DelaySSClaim{
		Participant: "Alice",
		NewAge:      60, // Invalid - must be 62-70
	}

	err := transform.Validate(base)

	assert.Error(t, err, "Should error for invalid age")
	assert.Contains(t, err.Error(), "SS start age must be between 62 and 70", "Should have specific error message")
}

func TestModifyTSPStrategy_Apply(t *testing.T) {
	base := createTestScenario()

	transform := &ModifyTSPStrategy{
		Participant: "Alice",
		NewStrategy: "need_based",
	}

	result, err := transform.Apply(base)

	assert.NoError(t, err, "Should not error")
	assert.NotNil(t, result, "Should return result")

	aliceScenario, exists := result.ParticipantScenarios["Alice"]
	assert.True(t, exists, "Should have Alice scenario")

	assert.Equal(t, "need_based", aliceScenario.TSPWithdrawalStrategy, "Should change TSP strategy")
}

func TestModifyTSPStrategy_Validate_InvalidStrategy(t *testing.T) {
	base := createTestScenario()

	transform := &ModifyTSPStrategy{
		Participant: "Alice",
		NewStrategy: "invalid_strategy",
	}

	err := transform.Validate(base)

	assert.Error(t, err, "Should error for invalid strategy")
	assert.Contains(t, err.Error(), "invalid TSP strategy", "Should have specific error message")
}

func TestAdjustTSPRate_Apply(t *testing.T) {
	base := createTestScenario()

	newRate := decimal.NewFromFloat(0.05)
	transform := &AdjustTSPRate{
		Participant: "Alice",
		NewRate:     newRate,
	}

	result, err := transform.Apply(base)

	assert.NoError(t, err, "Should not error")
	assert.NotNil(t, result, "Should return result")

	aliceScenario, exists := result.ParticipantScenarios["Alice"]
	assert.True(t, exists, "Should have Alice scenario")

	assert.True(t, newRate.Equal(*aliceScenario.TSPWithdrawalRate), "Should change TSP withdrawal rate")
}

func TestAdjustTSPRate_Validate_InvalidRate(t *testing.T) {
	base := createTestScenario()

	transform := &AdjustTSPRate{
		Participant: "Alice",
		NewRate:     decimal.NewFromFloat(0.25), // Invalid - too high
	}

	err := transform.Validate(base)

	assert.Error(t, err, "Should error for invalid rate")
	assert.Contains(t, err.Error(), "TSP rate must be between 0 and 0.20", "Should have specific error message")
}

func TestSetTSPTargetIncome_Apply(t *testing.T) {
	base := createTestScenario()

	targetIncome := decimal.NewFromInt(5000)
	transform := &SetTSPTargetIncome{
		Participant:   "Alice",
		MonthlyTarget: targetIncome,
	}

	result, err := transform.Apply(base)

	assert.NoError(t, err, "Should not error")
	assert.NotNil(t, result, "Should return result")

	aliceScenario, exists := result.ParticipantScenarios["Alice"]
	assert.True(t, exists, "Should have Alice scenario")

	assert.True(t, targetIncome.Equal(*aliceScenario.TSPWithdrawalTargetMonthly), "Should set TSP target income")
}

func TestSetTSPTargetIncome_Validate_NegativeIncome(t *testing.T) {
	base := createTestScenario()

	transform := &SetTSPTargetIncome{
		Participant:   "Alice",
		MonthlyTarget: decimal.NewFromInt(-1000), // Invalid
	}

	err := transform.Validate(base)

	assert.Error(t, err, "Should error for negative income")
	assert.Contains(t, err.Error(), "monthly target must be positive", "Should have specific error message")
}

func TestSetMortalityDate_Apply(t *testing.T) {
	base := createTestScenario()

	deathDate := time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC)
	transform := &SetMortalityDate{
		Participant: "Alice",
		DeathDate:   deathDate,
	}

	result, err := transform.Apply(base)

	assert.NoError(t, err, "Should not error")
	assert.NotNil(t, result, "Should return result")

	// Check that mortality was added to the scenario
	assert.NotNil(t, result.Mortality, "Should add mortality to scenario")
	assert.NotNil(t, result.Mortality.Participants, "Should have participants in mortality")

	aliceMortality, exists := result.Mortality.Participants["Alice"]
	assert.True(t, exists, "Should have Alice mortality")
	assert.Equal(t, deathDate, *aliceMortality.DeathDate, "Should set death date")
}

func TestSetMortalityDate_Validate_ZeroDate(t *testing.T) {
	base := createTestScenario()

	// Set zero death date
	transform := &SetMortalityDate{
		Participant: "Alice",
		DeathDate:   time.Time{}, // Zero time
	}

	err := transform.Validate(base)

	assert.Error(t, err, "Should error for zero death date")
	assert.Contains(t, err.Error(), "death date cannot be zero", "Should have specific error message")
}

func TestSetSurvivorSpendingFactor_Apply(t *testing.T) {
	base := createTestScenario()

	spendingFactor := decimal.NewFromFloat(0.8)
	transform := &SetSurvivorSpendingFactor{
		Factor: spendingFactor,
	}

	result, err := transform.Apply(base)

	assert.NoError(t, err, "Should not error")
	assert.NotNil(t, result, "Should return result")

	// Check that mortality assumptions were added
	assert.NotNil(t, result.Mortality, "Should add mortality to scenario")
	assert.NotNil(t, result.Mortality.Assumptions, "Should have mortality assumptions")

	assert.True(t, spendingFactor.Equal(result.Mortality.Assumptions.SurvivorSpendingFactor), "Should set survivor spending factor")
}

func TestSetSurvivorSpendingFactor_Validate_InvalidFactor(t *testing.T) {
	base := createTestScenario()

	transform := &SetSurvivorSpendingFactor{
		Factor: decimal.NewFromFloat(1.5), // Invalid - too high
	}

	err := transform.Validate(base)

	assert.Error(t, err, "Should error for invalid spending factor")
	assert.Contains(t, err.Error(), "survivor spending factor must be between 0 and 1", "Should have specific error message")
}

func TestSetTSPTransferMode_Apply(t *testing.T) {
	base := createTestScenario()

	transform := &SetTSPTransferMode{
		Mode: "merge",
	}

	result, err := transform.Apply(base)

	assert.NoError(t, err, "Should not error")
	assert.NotNil(t, result, "Should return result")

	// Check that mortality assumptions were added
	assert.NotNil(t, result.Mortality, "Should add mortality to scenario")
	assert.NotNil(t, result.Mortality.Assumptions, "Should have mortality assumptions")

	assert.Equal(t, "merge", result.Mortality.Assumptions.TSPSpousalTransfer, "Should set TSP transfer mode")
}

func TestSetTSPTransferMode_Validate_InvalidMode(t *testing.T) {
	base := createTestScenario()

	transform := &SetTSPTransferMode{
		Mode: "invalid_mode",
	}

	err := transform.Validate(base)

	assert.Error(t, err, "Should error for invalid transfer mode")
	assert.Contains(t, err.Error(), "invalid TSP transfer mode", "Should have specific error message")
}
