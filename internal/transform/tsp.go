package transform

import (
	"fmt"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// ModifyTSPStrategy changes the TSP withdrawal strategy for a participant.
// Valid strategies: "4_percent_rule", "variable_percentage", "need_based", "fixed_amount"
type ModifyTSPStrategy struct {
	Participant   string // Name of the participant
	NewStrategy   string // New withdrawal strategy
	PreserveRate  bool   // If true, preserve existing withdrawal rate/target when switching strategies
}

func (mts *ModifyTSPStrategy) Name() string {
	return "modify_tsp_strategy"
}

func (mts *ModifyTSPStrategy) Description() string {
	return fmt.Sprintf("Change %s's TSP withdrawal strategy to %s", mts.Participant, mts.NewStrategy)
}

func (mts *ModifyTSPStrategy) Validate(base *domain.GenericScenario) error {
	if mts.Participant == "" {
		return NewTransformError(mts.Name(), "validate", "participant name cannot be empty", nil)
	}

	validStrategies := map[string]bool{
		"4_percent_rule":     true,
		"variable_percentage": true,
		"need_based":         true,
		"fixed_amount":       true,
	}

	if !validStrategies[mts.NewStrategy] {
		return NewTransformError(mts.Name(), "validate", fmt.Sprintf("invalid TSP strategy: %s", mts.NewStrategy), nil)
	}

	if base == nil {
		return NewTransformError(mts.Name(), "validate", "base scenario cannot be nil", nil)
	}

	_, exists := base.ParticipantScenarios[mts.Participant]
	if !exists {
		return NewTransformError(mts.Name(), "validate", fmt.Sprintf("participant %s not found in scenario", mts.Participant), nil)
	}

	return nil
}

func (mts *ModifyTSPStrategy) Apply(base *domain.GenericScenario) (*domain.GenericScenario, error) {
	// Create a deep copy
	modified := base.DeepCopy()

	// Get the participant scenario
	ps := modified.ParticipantScenarios[mts.Participant]

	// Set the new strategy
	ps.TSPWithdrawalStrategy = mts.NewStrategy

	// If not preserving rate/target, clear them (they'll use defaults)
	if !mts.PreserveRate {
		ps.TSPWithdrawalRate = nil
		ps.TSPWithdrawalTargetMonthly = nil
	}

	// Update the map
	modified.ParticipantScenarios[mts.Participant] = ps

	return modified, nil
}

// AdjustTSPRate changes the TSP withdrawal rate for percentage-based strategies.
// Rate should be expressed as a decimal (e.g., 0.04 for 4%).
type AdjustTSPRate struct {
	Participant string          // Name of the participant
	NewRate     decimal.Decimal // New withdrawal rate (e.g., 0.04 for 4%)
}

func (atr *AdjustTSPRate) Name() string {
	return "adjust_tsp_rate"
}

func (atr *AdjustTSPRate) Description() string {
	percentage := atr.NewRate.Mul(decimal.NewFromInt(100))
	return fmt.Sprintf("Change %s's TSP withdrawal rate to %s%%", atr.Participant, percentage.StringFixed(1))
}

func (atr *AdjustTSPRate) Validate(base *domain.GenericScenario) error {
	if atr.Participant == "" {
		return NewTransformError(atr.Name(), "validate", "participant name cannot be empty", nil)
	}

	if atr.NewRate.LessThanOrEqual(decimal.Zero) || atr.NewRate.GreaterThan(decimal.NewFromFloat(0.20)) {
		return NewTransformError(atr.Name(), "validate", fmt.Sprintf("TSP rate must be between 0 and 0.20, got %s", atr.NewRate.String()), nil)
	}

	if base == nil {
		return NewTransformError(atr.Name(), "validate", "base scenario cannot be nil", nil)
	}

	ps, exists := base.ParticipantScenarios[atr.Participant]
	if !exists {
		return NewTransformError(atr.Name(), "validate", fmt.Sprintf("participant %s not found in scenario", atr.Participant), nil)
	}

	// Check if strategy is compatible with rates
	if ps.TSPWithdrawalStrategy != "variable_percentage" && ps.TSPWithdrawalStrategy != "4_percent_rule" {
		return NewTransformError(atr.Name(), "validate", fmt.Sprintf("TSP rate only applicable to percentage-based strategies, current strategy is %s", ps.TSPWithdrawalStrategy), nil)
	}

	return nil
}

func (atr *AdjustTSPRate) Apply(base *domain.GenericScenario) (*domain.GenericScenario, error) {
	// Create a deep copy
	modified := base.DeepCopy()

	// Get the participant scenario
	ps := modified.ParticipantScenarios[atr.Participant]

	// Set the new rate
	rateCopy := atr.NewRate
	ps.TSPWithdrawalRate = &rateCopy

	// Update the map
	modified.ParticipantScenarios[atr.Participant] = ps

	return modified, nil
}

// SetTSPTargetIncome changes the monthly target income for need-based TSP withdrawals.
type SetTSPTargetIncome struct {
	Participant     string          // Name of the participant
	MonthlyTarget   decimal.Decimal // Target monthly income from TSP
}

func (stti *SetTSPTargetIncome) Name() string {
	return "set_tsp_target_income"
}

func (stti *SetTSPTargetIncome) Description() string {
	return fmt.Sprintf("Set %s's TSP monthly target to $%s", stti.Participant, stti.MonthlyTarget.StringFixed(0))
}

func (stti *SetTSPTargetIncome) Validate(base *domain.GenericScenario) error {
	if stti.Participant == "" {
		return NewTransformError(stti.Name(), "validate", "participant name cannot be empty", nil)
	}

	if stti.MonthlyTarget.LessThanOrEqual(decimal.Zero) {
		return NewTransformError(stti.Name(), "validate", fmt.Sprintf("monthly target must be positive, got %s", stti.MonthlyTarget.String()), nil)
	}

	if base == nil {
		return NewTransformError(stti.Name(), "validate", "base scenario cannot be nil", nil)
	}

	ps, exists := base.ParticipantScenarios[stti.Participant]
	if !exists {
		return NewTransformError(stti.Name(), "validate", fmt.Sprintf("participant %s not found in scenario", stti.Participant), nil)
	}

	// Check if strategy is compatible
	if ps.TSPWithdrawalStrategy != "need_based" {
		return NewTransformError(stti.Name(), "validate", fmt.Sprintf("monthly target only applicable to need_based strategy, current strategy is %s", ps.TSPWithdrawalStrategy), nil)
	}

	return nil
}

func (stti *SetTSPTargetIncome) Apply(base *domain.GenericScenario) (*domain.GenericScenario, error) {
	// Create a deep copy
	modified := base.DeepCopy()

	// Get the participant scenario
	ps := modified.ParticipantScenarios[stti.Participant]

	// Set the new target
	targetCopy := stti.MonthlyTarget
	ps.TSPWithdrawalTargetMonthly = &targetCopy

	// Update the map
	modified.ParticipantScenarios[stti.Participant] = ps

	return modified, nil
}
