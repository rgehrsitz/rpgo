package transform

import (
	"fmt"
	"time"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// SetMortalityDate sets or modifies the death date for a participant.
// This is used for survivor analysis and estate planning scenarios.
type SetMortalityDate struct {
	Participant string    // Name of the participant
	DeathDate   time.Time // Date of death for scenario analysis
}

func (smd *SetMortalityDate) Name() string {
	return "set_mortality_date"
}

func (smd *SetMortalityDate) Description() string {
	return fmt.Sprintf("Set %s's death date to %s for mortality analysis", smd.Participant, smd.DeathDate.Format("2006-01-02"))
}

func (smd *SetMortalityDate) Validate(base *domain.GenericScenario) error {
	if smd.Participant == "" {
		return NewTransformError(smd.Name(), "validate", "participant name cannot be empty", nil)
	}

	if smd.DeathDate.IsZero() {
		return NewTransformError(smd.Name(), "validate", "death date cannot be zero", nil)
	}

	if base == nil {
		return NewTransformError(smd.Name(), "validate", "base scenario cannot be nil", nil)
	}

	_, exists := base.ParticipantScenarios[smd.Participant]
	if !exists {
		return NewTransformError(smd.Name(), "validate", fmt.Sprintf("participant %s not found in scenario", smd.Participant), nil)
	}

	return nil
}

func (smd *SetMortalityDate) Apply(base *domain.GenericScenario) (*domain.GenericScenario, error) {
	// Create a deep copy
	modified := base.DeepCopy()

	// Initialize mortality if it doesn't exist
	if modified.Mortality == nil {
		modified.Mortality = &domain.GenericScenarioMortality{
			Participants: make(map[string]*domain.MortalitySpec),
		}
	}

	// Ensure participants map exists
	if modified.Mortality.Participants == nil {
		modified.Mortality.Participants = make(map[string]*domain.MortalitySpec)
	}

	// Set the death date for this participant
	dateCopy := smd.DeathDate
	modified.Mortality.Participants[smd.Participant] = &domain.MortalitySpec{
		DeathDate: &dateCopy,
	}

	return modified, nil
}

// SetSurvivorSpendingFactor changes the spending adjustment after a spouse's death.
// Typical values: 0.75 (survivor needs 75% of couple's spending) to 0.85.
type SetSurvivorSpendingFactor struct {
	Factor decimal.Decimal // Spending factor (e.g., 0.75 for 75%)
}

func (sssf *SetSurvivorSpendingFactor) Name() string {
	return "set_survivor_spending"
}

func (sssf *SetSurvivorSpendingFactor) Description() string {
	percentage := sssf.Factor.Mul(decimal.NewFromInt(100))
	return fmt.Sprintf("Set survivor spending to %s%% of couple's spending", percentage.StringFixed(0))
}

func (sssf *SetSurvivorSpendingFactor) Validate(base *domain.GenericScenario) error {
	if sssf.Factor.LessThanOrEqual(decimal.Zero) || sssf.Factor.GreaterThan(decimal.NewFromInt(1)) {
		return NewTransformError(sssf.Name(), "validate", fmt.Sprintf("survivor spending factor must be between 0 and 1, got %s", sssf.Factor.String()), nil)
	}

	if base == nil {
		return NewTransformError(sssf.Name(), "validate", "base scenario cannot be nil", nil)
	}

	return nil
}

func (sssf *SetSurvivorSpendingFactor) Apply(base *domain.GenericScenario) (*domain.GenericScenario, error) {
	// Create a deep copy
	modified := base.DeepCopy()

	// Initialize mortality if it doesn't exist
	if modified.Mortality == nil {
		modified.Mortality = &domain.GenericScenarioMortality{}
	}

	// Initialize assumptions if they don't exist
	if modified.Mortality.Assumptions == nil {
		modified.Mortality.Assumptions = &domain.MortalityAssumptions{}
	}

	// Set the survivor spending factor
	modified.Mortality.Assumptions.SurvivorSpendingFactor = sssf.Factor

	return modified, nil
}

// SetTSPTransferMode changes how TSP assets are handled after a spouse's death.
// Valid modes: "merge" (combine balances), "keep_separate", "survivor_inherits"
type SetTSPTransferMode struct {
	Mode string // Transfer mode
}

func (sttm *SetTSPTransferMode) Name() string {
	return "set_tsp_transfer"
}

func (sttm *SetTSPTransferMode) Description() string {
	return fmt.Sprintf("Set TSP spousal transfer mode to %s", sttm.Mode)
}

func (sttm *SetTSPTransferMode) Validate(base *domain.GenericScenario) error {
	validModes := map[string]bool{
		"merge":             true,
		"keep_separate":     true,
		"survivor_inherits": true,
	}

	if !validModes[sttm.Mode] {
		return NewTransformError(sttm.Name(), "validate", fmt.Sprintf("invalid TSP transfer mode: %s", sttm.Mode), nil)
	}

	if base == nil {
		return NewTransformError(sttm.Name(), "validate", "base scenario cannot be nil", nil)
	}

	return nil
}

func (sttm *SetTSPTransferMode) Apply(base *domain.GenericScenario) (*domain.GenericScenario, error) {
	// Create a deep copy
	modified := base.DeepCopy()

	// Initialize mortality if it doesn't exist
	if modified.Mortality == nil {
		modified.Mortality = &domain.GenericScenarioMortality{}
	}

	// Initialize assumptions if they don't exist
	if modified.Mortality.Assumptions == nil {
		modified.Mortality.Assumptions = &domain.MortalityAssumptions{}
	}

	// Set the TSP transfer mode
	modified.Mortality.Assumptions.TSPSpousalTransfer = sttm.Mode

	return modified, nil
}
