package transform

import (
	"fmt"
	"testing"
	"time"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// Helper function to create a basic test scenario
func createTestScenario() *domain.GenericScenario {
	retireDate := time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC)
	withdrawalRate := decimal.NewFromFloat(0.04)

	return &domain.GenericScenario{
		Name: "Test Scenario",
		ParticipantScenarios: map[string]domain.ParticipantScenario{
			"Alice": {
				ParticipantName:       "Alice",
				RetirementDate:        &retireDate,
				SSStartAge:            62,
				TSPWithdrawalStrategy: "4_percent_rule",
				TSPWithdrawalRate:     &withdrawalRate,
			},
			"Bob": {
				ParticipantName:       "Bob",
				RetirementDate:        &retireDate,
				SSStartAge:            65,
				TSPWithdrawalStrategy: "variable_percentage",
				TSPWithdrawalRate:     &withdrawalRate,
			},
		},
	}
}

func TestApplyTransforms_NilScenario(t *testing.T) {
	transforms := []ScenarioTransform{
		&PostponeRetirement{Participant: "Alice", Months: 12},
	}

	_, err := ApplyTransforms(nil, transforms)
	if err == nil {
		t.Error("Expected error for nil scenario, got nil")
	}
}

func TestApplyTransforms_EmptyTransforms(t *testing.T) {
	base := createTestScenario()
	transforms := []ScenarioTransform{}

	result, err := ApplyTransforms(base, transforms)
	if err != nil {
		t.Fatalf("Expected no error for empty transforms, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Should return a copy, not the same instance
	if result == base {
		t.Error("Expected a copy, got same instance")
	}

	// But content should be the same
	if result.Name != base.Name {
		t.Errorf("Expected name %s, got %s", base.Name, result.Name)
	}
}

func TestApplyTransforms_NilTransform(t *testing.T) {
	base := createTestScenario()
	transforms := []ScenarioTransform{
		&PostponeRetirement{Participant: "Alice", Months: 12},
		nil, // Nil transform should cause error
	}

	_, err := ApplyTransforms(base, transforms)
	if err == nil {
		t.Error("Expected error for nil transform in list, got nil")
	}
}

func TestApplyTransforms_ValidationFailure(t *testing.T) {
	base := createTestScenario()
	transforms := []ScenarioTransform{
		&PostponeRetirement{Participant: "NonExistent", Months: 12},
	}

	_, err := ApplyTransforms(base, transforms)
	if err == nil {
		t.Error("Expected validation error for non-existent participant, got nil")
	}
}

func TestApplyTransforms_SingleTransform(t *testing.T) {
	base := createTestScenario()
	originalDate := *base.ParticipantScenarios["Alice"].RetirementDate

	transforms := []ScenarioTransform{
		&PostponeRetirement{Participant: "Alice", Months: 12},
	}

	result, err := ApplyTransforms(base, transforms)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	newDate := *result.ParticipantScenarios["Alice"].RetirementDate
	expectedDate := originalDate.AddDate(0, 12, 0)

	if !newDate.Equal(expectedDate) {
		t.Errorf("Expected date %v, got %v", expectedDate, newDate)
	}

	// Original should be unchanged
	if !base.ParticipantScenarios["Alice"].RetirementDate.Equal(originalDate) {
		t.Error("Original scenario was modified")
	}
}

func TestApplyTransforms_MultipleTransforms(t *testing.T) {
	base := createTestScenario()

	transforms := []ScenarioTransform{
		&PostponeRetirement{Participant: "Alice", Months: 12},
		&DelaySSClaim{Participant: "Alice", NewAge: 67},
		&ModifyTSPStrategy{Participant: "Alice", NewStrategy: "need_based"},
	}

	result, err := ApplyTransforms(base, transforms)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check all transforms were applied
	aliceScenario := result.ParticipantScenarios["Alice"]

	// Check retirement date was postponed
	expectedDate := base.ParticipantScenarios["Alice"].RetirementDate.AddDate(0, 12, 0)
	if !aliceScenario.RetirementDate.Equal(expectedDate) {
		t.Errorf("Expected retirement date %v, got %v", expectedDate, *aliceScenario.RetirementDate)
	}

	// Check SS age was changed
	if aliceScenario.SSStartAge != 67 {
		t.Errorf("Expected SS start age 67, got %d", aliceScenario.SSStartAge)
	}

	// Check TSP strategy was changed
	if aliceScenario.TSPWithdrawalStrategy != "need_based" {
		t.Errorf("Expected TSP strategy need_based, got %s", aliceScenario.TSPWithdrawalStrategy)
	}

	// Original should be unchanged
	if base.ParticipantScenarios["Alice"].SSStartAge != 62 {
		t.Error("Original scenario was modified")
	}
}

func TestApplyTransforms_TransformChaining(t *testing.T) {
	base := createTestScenario()

	// Each transform receives the output of the previous one
	transforms := []ScenarioTransform{
		&PostponeRetirement{Participant: "Alice", Months: 6},
		&PostponeRetirement{Participant: "Alice", Months: 6}, // Should add another 6 months
	}

	result, err := ApplyTransforms(base, transforms)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Total postponement should be 12 months
	expectedDate := base.ParticipantScenarios["Alice"].RetirementDate.AddDate(0, 12, 0)
	actualDate := *result.ParticipantScenarios["Alice"].RetirementDate

	if !actualDate.Equal(expectedDate) {
		t.Errorf("Expected date %v (12 months later), got %v", expectedDate, actualDate)
	}
}

func TestTransformError(t *testing.T) {
	err := NewTransformError("test_transform", "apply", "test reason", nil)

	if err == nil {
		t.Fatal("Expected non-nil error")
	}

	expectedMsg := "transform test_transform (apply): test reason"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestTransformError_WithWrappedError(t *testing.T) {
	innerErr := fmt.Errorf("inner error")
	err := NewTransformError("test_transform", "validate", "validation failed", innerErr)

	if err == nil {
		t.Fatal("Expected non-nil error")
	}

	expectedMsg := "transform test_transform (validate): validation failed: inner error"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}
