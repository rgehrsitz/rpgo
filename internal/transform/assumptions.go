package transform

import (
	"fmt"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// ModifyInflation changes the general inflation rate assumption.
// This affects purchasing power of fixed income and COLA calculations.
type ModifyInflation struct {
	NewRate decimal.Decimal // New inflation rate (e.g., 0.025 for 2.5%)
}

func (mi *ModifyInflation) Name() string {
	return "modify_inflation"
}

func (mi *ModifyInflation) Description() string {
	percentage := mi.NewRate.Mul(decimal.NewFromInt(100))
	return fmt.Sprintf("Change inflation rate to %s%%", percentage.StringFixed(1))
}

func (mi *ModifyInflation) Validate(base *domain.GenericScenario) error {
	if mi.NewRate.LessThan(decimal.Zero) || mi.NewRate.GreaterThan(decimal.NewFromFloat(0.10)) {
		return NewTransformError(mi.Name(), "validate", fmt.Sprintf("inflation rate must be between 0 and 0.10, got %s", mi.NewRate.String()), nil)
	}

	if base == nil {
		return NewTransformError(mi.Name(), "validate", "base scenario cannot be nil", nil)
	}

	return nil
}

func (mi *ModifyInflation) Apply(base *domain.GenericScenario) (*domain.GenericScenario, error) {
	// Create a deep copy
	modified := base.DeepCopy()

	// Note: This transform doesn't modify the scenario directly since inflation
	// is a GlobalAssumption, not part of the scenario. This transform serves as
	// a marker for the scenario comparison system to know to modify inflation
	// when running the projection. The actual inflation modification happens
	// in the projection engine.

	return modified, nil
}

// Note: Inflation, COLA, FEHB inflation, and TSP returns are GlobalAssumptions,
// not part of GenericScenario. These transforms are markers that the compare/solver
// systems will use to create modified Configuration objects before running projections.
// The transforms on GenericScenario only modify scenario-specific parameters
// (retirement dates, SS ages, TSP strategies, etc.).

// For now, we'll focus on transforms that directly modify the scenario.
// Assumption transforms will be handled by a separate mechanism in the compare command.
