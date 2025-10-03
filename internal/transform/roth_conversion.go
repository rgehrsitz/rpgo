package transform

import (
	"fmt"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// EnableRothConversion adds a Roth conversion schedule to a participant's scenario.
// This transform enables Roth conversions by adding conversion events to the scenario.
type EnableRothConversion struct {
	Participant string                  // Name of the participant
	Conversions []domain.RothConversion // List of conversions to add
}

func (erc *EnableRothConversion) Name() string {
	return "enable_roth_conversion"
}

func (erc *EnableRothConversion) Description() string {
	if len(erc.Conversions) == 0 {
		return fmt.Sprintf("Enable Roth conversions for %s (no conversions specified)", erc.Participant)
	}
	return fmt.Sprintf("Enable %d Roth conversions for %s", len(erc.Conversions), erc.Participant)
}

func (erc *EnableRothConversion) Validate(base *domain.GenericScenario) error {
	if erc.Participant == "" {
		return NewTransformError(erc.Name(), "validate", "participant name cannot be empty", nil)
	}

	if base == nil {
		return NewTransformError(erc.Name(), "validate", "base scenario cannot be nil", nil)
	}

	_, exists := base.ParticipantScenarios[erc.Participant]
	if !exists {
		return NewTransformError(erc.Name(), "validate", fmt.Sprintf("participant %s not found in scenario", erc.Participant), nil)
	}

	// Validate each conversion
	for i, conversion := range erc.Conversions {
		if conversion.Amount.LessThanOrEqual(decimal.Zero) {
			return NewTransformError(erc.Name(), "validate", fmt.Sprintf("conversion %d: amount must be positive, got %s", i, conversion.Amount.String()), nil)
		}

		if conversion.Year < 2020 || conversion.Year > 2100 {
			return NewTransformError(erc.Name(), "validate", fmt.Sprintf("conversion %d: year must be between 2020-2100, got %d", i, conversion.Year), nil)
		}

		validSources := map[string]bool{
			"traditional_tsp": true,
			"traditional_ira": true,
		}

		if !validSources[conversion.Source] {
			return NewTransformError(erc.Name(), "validate", fmt.Sprintf("conversion %d: invalid source %s", i, conversion.Source), nil)
		}
	}

	return nil
}

func (erc *EnableRothConversion) Apply(base *domain.GenericScenario) (*domain.GenericScenario, error) {
	// Create a deep copy
	modified := base.DeepCopy()

	// Get the participant scenario
	ps := modified.ParticipantScenarios[erc.Participant]

	// Create or update the Roth conversion schedule
	if ps.RothConversions == nil {
		ps.RothConversions = &domain.RothConversionSchedule{
			Conversions: make([]domain.RothConversion, 0),
		}
	}

	// Add the new conversions
	ps.RothConversions.Conversions = append(ps.RothConversions.Conversions, erc.Conversions...)

	// Update the map
	modified.ParticipantScenarios[erc.Participant] = ps

	return modified, nil
}

// ModifyRothConversion changes an existing Roth conversion amount.
type ModifyRothConversion struct {
	Participant string          // Name of the participant
	Year        int             // Year of the conversion to modify
	NewAmount   decimal.Decimal // New conversion amount
}

func (mrc *ModifyRothConversion) Name() string {
	return "modify_roth_conversion"
}

func (mrc *ModifyRothConversion) Description() string {
	return fmt.Sprintf("Modify %s's Roth conversion in %d to $%s", mrc.Participant, mrc.Year, mrc.NewAmount.StringFixed(0))
}

func (mrc *ModifyRothConversion) Validate(base *domain.GenericScenario) error {
	if mrc.Participant == "" {
		return NewTransformError(mrc.Name(), "validate", "participant name cannot be empty", nil)
	}

	if mrc.NewAmount.LessThanOrEqual(decimal.Zero) {
		return NewTransformError(mrc.Name(), "validate", fmt.Sprintf("conversion amount must be positive, got %s", mrc.NewAmount.String()), nil)
	}

	if base == nil {
		return NewTransformError(mrc.Name(), "validate", "base scenario cannot be nil", nil)
	}

	ps, exists := base.ParticipantScenarios[mrc.Participant]
	if !exists {
		return NewTransformError(mrc.Name(), "validate", fmt.Sprintf("participant %s not found in scenario", mrc.Participant), nil)
	}

	if ps.RothConversions == nil || len(ps.RothConversions.Conversions) == 0 {
		return NewTransformError(mrc.Name(), "validate", fmt.Sprintf("participant %s has no Roth conversions to modify", mrc.Participant), nil)
	}

	// Check if the year exists
	found := false
	for _, conversion := range ps.RothConversions.Conversions {
		if conversion.Year == mrc.Year {
			found = true
			break
		}
	}

	if !found {
		return NewTransformError(mrc.Name(), "validate", fmt.Sprintf("no Roth conversion found for year %d", mrc.Year), nil)
	}

	return nil
}

func (mrc *ModifyRothConversion) Apply(base *domain.GenericScenario) (*domain.GenericScenario, error) {
	// Create a deep copy
	modified := base.DeepCopy()

	// Get the participant scenario
	ps := modified.ParticipantScenarios[mrc.Participant]

	// Find and modify the conversion
	for i, conversion := range ps.RothConversions.Conversions {
		if conversion.Year == mrc.Year {
			ps.RothConversions.Conversions[i].Amount = mrc.NewAmount
			break
		}
	}

	// Update the map
	modified.ParticipantScenarios[mrc.Participant] = ps

	return modified, nil
}

// RemoveRothConversion removes a Roth conversion from a participant's scenario.
type RemoveRothConversion struct {
	Participant string // Name of the participant
	Year        int    // Year of the conversion to remove
}

func (rrc *RemoveRothConversion) Name() string {
	return "remove_roth_conversion"
}

func (rrc *RemoveRothConversion) Description() string {
	return fmt.Sprintf("Remove %s's Roth conversion in %d", rrc.Participant, rrc.Year)
}

func (rrc *RemoveRothConversion) Validate(base *domain.GenericScenario) error {
	if rrc.Participant == "" {
		return NewTransformError(rrc.Name(), "validate", "participant name cannot be empty", nil)
	}

	if base == nil {
		return NewTransformError(rrc.Name(), "validate", "base scenario cannot be nil", nil)
	}

	ps, exists := base.ParticipantScenarios[rrc.Participant]
	if !exists {
		return NewTransformError(rrc.Name(), "validate", fmt.Sprintf("participant %s not found in scenario", rrc.Participant), nil)
	}

	if ps.RothConversions == nil || len(ps.RothConversions.Conversions) == 0 {
		return NewTransformError(rrc.Name(), "validate", fmt.Sprintf("participant %s has no Roth conversions to remove", rrc.Participant), nil)
	}

	// Check if the year exists
	found := false
	for _, conversion := range ps.RothConversions.Conversions {
		if conversion.Year == rrc.Year {
			found = true
			break
		}
	}

	if !found {
		return NewTransformError(rrc.Name(), "validate", fmt.Sprintf("no Roth conversion found for year %d", rrc.Year), nil)
	}

	return nil
}

func (rrc *RemoveRothConversion) Apply(base *domain.GenericScenario) (*domain.GenericScenario, error) {
	// Create a deep copy
	modified := base.DeepCopy()

	// Get the participant scenario
	ps := modified.ParticipantScenarios[rrc.Participant]

	// Remove the conversion
	newConversions := make([]domain.RothConversion, 0, len(ps.RothConversions.Conversions)-1)
	for _, conversion := range ps.RothConversions.Conversions {
		if conversion.Year != rrc.Year {
			newConversions = append(newConversions, conversion)
		}
	}

	ps.RothConversions.Conversions = newConversions

	// If no conversions left, remove the schedule
	if len(ps.RothConversions.Conversions) == 0 {
		ps.RothConversions = nil
	}

	// Update the map
	modified.ParticipantScenarios[rrc.Participant] = ps

	return modified, nil
}
