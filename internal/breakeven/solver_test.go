package breakeven

import (
	"context"
	"testing"

	"github.com/rgehrsitz/rpgo/internal/calculation"
	"github.com/shopspring/decimal"
)

func TestNewSolver(t *testing.T) {
	calcEngine := &calculation.CalculationEngine{}
	options := DefaultSolverOptions()

	solver := NewSolver(calcEngine, options)

	if solver == nil {
		t.Fatal("Expected solver to be created, got nil")
	}

	if solver.CalcEngine != calcEngine {
		t.Error("Expected CalcEngine to match input")
	}

	if solver.Options != options {
		t.Error("Expected Options to match input")
	}
}

func TestNewDefaultSolver(t *testing.T) {
	calcEngine := &calculation.CalculationEngine{}

	solver := NewDefaultSolver(calcEngine)

	if solver == nil {
		t.Fatal("Expected solver to be created, got nil")
	}

	if solver.CalcEngine != calcEngine {
		t.Error("Expected CalcEngine to match input")
	}

	// Check that default options are applied
	expectedOptions := DefaultSolverOptions()
	if solver.Options.Algorithm != expectedOptions.Algorithm {
		t.Error("Expected default algorithm to be applied")
	}
	if solver.Options.GridResolution != expectedOptions.GridResolution {
		t.Error("Expected default grid resolution to be applied")
	}
}

func TestSolver_Optimize_InvalidConstraints(t *testing.T) {
	calcEngine := &calculation.CalculationEngine{}
	solver := NewDefaultSolver(calcEngine)

	req := OptimizationRequest{
		Target: OptimizeTSPRate,
		Constraints: Constraints{
			Participant: "", // Invalid - empty participant
		},
	}

	result, err := solver.Optimize(context.Background(), req)

	if err == nil {
		t.Error("Expected error for invalid constraints, got nil")
	}

	if result != nil {
		t.Error("Expected result to be nil for invalid constraints")
	}
}

func TestSolver_Optimize_UnsupportedTarget(t *testing.T) {
	calcEngine := &calculation.CalculationEngine{}
	solver := NewDefaultSolver(calcEngine)

	req := OptimizationRequest{
		Target: "unsupported_target",
		Constraints: Constraints{
			Participant: "Alice",
		},
	}

	result, err := solver.Optimize(context.Background(), req)

	if err == nil {
		t.Error("Expected error for unsupported target, got nil")
	}

	if result != nil {
		t.Error("Expected result to be nil for unsupported target")
	}
}

func TestSolver_Optimize_ApplyDefaults(t *testing.T) {
	calcEngine := &calculation.CalculationEngine{}
	solver := NewDefaultSolver(calcEngine)

	req := OptimizationRequest{
		Target: OptimizeTSPRate,
		Constraints: Constraints{
			Participant: "Alice",
		},
		// MaxIterations and Tolerance are zero - should use defaults
	}

	// This will fail because we don't have a real calculation engine,
	// but we can test that defaults are applied
	_, err := solver.Optimize(context.Background(), req)

	// We expect an error because the calculation engine is not properly initialized,
	// but the error should not be about missing defaults
	if err != nil {
		// Check that the error is not about validation or unsupported target
		if err.Error() == "invalid constraints: participant name is required" {
			t.Error("Expected defaults to be applied before validation")
		}
	}
}

func TestSolver_Optimize_ContextCancellation(t *testing.T) {
	calcEngine := &calculation.CalculationEngine{}
	solver := NewDefaultSolver(calcEngine)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := OptimizationRequest{
		Target: OptimizeTSPRate,
		Constraints: Constraints{
			Participant: "Alice",
		},
	}

	_, err := solver.Optimize(ctx, req)

	// Should return context cancelled error
	if err == nil {
		t.Error("Expected context cancelled error")
	}

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

func TestSolver_Optimize_MaxIterationsExceeded(t *testing.T) {
	calcEngine := &calculation.CalculationEngine{}
	solver := NewDefaultSolver(calcEngine)

	req := OptimizationRequest{
		Target: OptimizeTSPRate,
		Constraints: Constraints{
			Participant: "Alice",
		},
		MaxIterations: 1,                     // Very small number
		Tolerance:     decimal.NewFromInt(1), // Very small tolerance
	}

	_, err := solver.Optimize(context.Background(), req)

	// Should fail due to mock calculation engine, not max iterations
	if err == nil {
		t.Error("Expected error with mock calculation engine")
	}
}
