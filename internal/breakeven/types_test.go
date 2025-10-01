package breakeven

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestDefaultConstraints(t *testing.T) {
	c := DefaultConstraints("Alice")

	if c.Participant != "Alice" {
		t.Errorf("Expected participant 'Alice', got %s", c.Participant)
	}

	if c.MinTSPRate == nil {
		t.Fatal("Expected MinTSPRate to be set")
	}
	expectedMin := decimal.NewFromFloat(0.02)
	if !c.MinTSPRate.Equal(expectedMin) {
		t.Errorf("Expected MinTSPRate 0.02, got %s", c.MinTSPRate.String())
	}

	if c.MaxTSPRate == nil {
		t.Fatal("Expected MaxTSPRate to be set")
	}
	expectedMax := decimal.NewFromFloat(0.10)
	if !c.MaxTSPRate.Equal(expectedMax) {
		t.Errorf("Expected MaxTSPRate 0.10, got %s", c.MaxTSPRate.String())
	}

	if c.MinSSAge == nil || *c.MinSSAge != 62 {
		t.Errorf("Expected MinSSAge 62, got %v", c.MinSSAge)
	}

	if c.MaxSSAge == nil || *c.MaxSSAge != 70 {
		t.Errorf("Expected MaxSSAge 70, got %v", c.MaxSSAge)
	}
}

func TestConstraints_Validate_EmptyParticipant(t *testing.T) {
	c := Constraints{
		Participant: "",
	}

	err := c.Validate()
	if err == nil {
		t.Error("Expected error for empty participant")
	}

	if _, ok := err.(*BreakEvenError); !ok {
		t.Errorf("Expected BreakEvenError, got %T", err)
	}
}

func TestConstraints_Validate_RetirementDateRange(t *testing.T) {
	minDate := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	maxDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) // Before min

	c := Constraints{
		Participant:       "Alice",
		MinRetirementDate: &minDate,
		MaxRetirementDate: &maxDate,
	}

	err := c.Validate()
	if err == nil {
		t.Error("Expected error for invalid retirement date range")
	}
}

func TestConstraints_Validate_TSPRateRange(t *testing.T) {
	minRate := decimal.NewFromFloat(0.10)
	maxRate := decimal.NewFromFloat(0.02) // Less than min

	c := Constraints{
		Participant: "Alice",
		MinTSPRate:  &minRate,
		MaxTSPRate:  &maxRate,
	}

	err := c.Validate()
	if err == nil {
		t.Error("Expected error for invalid TSP rate range")
	}
}

func TestConstraints_Validate_SSAgeRange(t *testing.T) {
	minAge := 70
	maxAge := 62 // Less than min

	c := Constraints{
		Participant: "Alice",
		MinSSAge:    &minAge,
		MaxSSAge:    &maxAge,
	}

	err := c.Validate()
	if err == nil {
		t.Error("Expected error for invalid SS age range")
	}
}

func TestConstraints_Validate_SSAgeBounds(t *testing.T) {
	// Test min age too low
	minAge := 60 // Below 62
	maxAge := 70

	c := Constraints{
		Participant: "Alice",
		MinSSAge:    &minAge,
		MaxSSAge:    &maxAge,
	}

	err := c.Validate()
	if err == nil {
		t.Error("Expected error for SS age below 62")
	}

	// Test max age too high
	minAge = 62
	maxAge = 72 // Above 70

	c = Constraints{
		Participant: "Alice",
		MinSSAge:    &minAge,
		MaxSSAge:    &maxAge,
	}

	err = c.Validate()
	if err == nil {
		t.Error("Expected error for SS age above 70")
	}
}

func TestConstraints_Validate_Valid(t *testing.T) {
	minDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	maxDate := time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC)
	minRate := decimal.NewFromFloat(0.02)
	maxRate := decimal.NewFromFloat(0.06)
	minAge := 62
	maxAge := 70

	c := Constraints{
		Participant:       "Alice",
		MinRetirementDate: &minDate,
		MaxRetirementDate: &maxDate,
		MinTSPRate:        &minRate,
		MaxTSPRate:        &maxRate,
		MinSSAge:          &minAge,
		MaxSSAge:          &maxAge,
	}

	err := c.Validate()
	if err != nil {
		t.Errorf("Expected no error for valid constraints, got: %v", err)
	}
}

func TestDefaultSolverOptions(t *testing.T) {
	opts := DefaultSolverOptions()

	if opts.Algorithm != "binary_search" {
		t.Errorf("Expected algorithm 'binary_search', got %s", opts.Algorithm)
	}

	if opts.GridResolution != 10 {
		t.Errorf("Expected grid resolution 10, got %d", opts.GridResolution)
	}

	expectedTol := decimal.NewFromInt(1000)
	if !opts.Tolerance.Equal(expectedTol) {
		t.Errorf("Expected tolerance 1000, got %s", opts.Tolerance.String())
	}

	if opts.MaxIterations != 50 {
		t.Errorf("Expected max iterations 50, got %d", opts.MaxIterations)
	}

	if opts.Parallel {
		t.Error("Expected Parallel to be false")
	}
}

func TestBreakEvenError(t *testing.T) {
	// Test error without cause
	err := &BreakEvenError{
		Operation: "test_op",
		Message:   "test message",
	}

	expected := "test_op: test message"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}

	// Test error with cause
	causeErr := &BreakEvenError{
		Operation: "cause_op",
		Message:   "cause message",
	}

	err = &BreakEvenError{
		Operation: "test_op",
		Message:   "test message",
		Cause:     causeErr,
	}

	expectedWithCause := "test_op: test message: cause_op: cause message"
	if err.Error() != expectedWithCause {
		t.Errorf("Expected error message '%s', got '%s'", expectedWithCause, err.Error())
	}

	// Test unwrap
	if err.Unwrap() != causeErr {
		t.Error("Unwrap() should return the cause error")
	}
}
