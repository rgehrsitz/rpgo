package transform

import (
	"fmt"
	"time"

	"github.com/rgehrsitz/rpgo/internal/domain"
)

// PostponeRetirement delays a participant's retirement date by a specified number of months.
// This is useful for exploring "work one more year" scenarios.
type PostponeRetirement struct {
	Participant string // Name of the participant whose retirement to postpone
	Months      int    // Number of months to postpone (positive integer)
}

func (pt *PostponeRetirement) Name() string {
	return "postpone_retirement"
}

func (pt *PostponeRetirement) Description() string {
	return fmt.Sprintf("Postpone %s's retirement by %d months", pt.Participant, pt.Months)
}

func (pt *PostponeRetirement) Validate(base *domain.GenericScenario) error {
	if pt.Participant == "" {
		return NewTransformError(pt.Name(), "validate", "participant name cannot be empty", nil)
	}

	if pt.Months < 0 {
		return NewTransformError(pt.Name(), "validate", fmt.Sprintf("months must be non-negative, got %d", pt.Months), nil)
	}

	if base == nil {
		return NewTransformError(pt.Name(), "validate", "base scenario cannot be nil", nil)
	}

	ps, exists := base.ParticipantScenarios[pt.Participant]
	if !exists {
		return NewTransformError(pt.Name(), "validate", fmt.Sprintf("participant %s not found in scenario", pt.Participant), nil)
	}

	if ps.RetirementDate == nil {
		return NewTransformError(pt.Name(), "validate", fmt.Sprintf("participant %s has no retirement date", pt.Participant), nil)
	}

	return nil
}

func (pt *PostponeRetirement) Apply(base *domain.GenericScenario) (*domain.GenericScenario, error) {
	// Create a deep copy
	modified := base.DeepCopy()

	// Get the participant scenario
	ps := modified.ParticipantScenarios[pt.Participant]

	// Add months to retirement date
	newDate := ps.RetirementDate.AddDate(0, pt.Months, 0)
	ps.RetirementDate = &newDate

	// Update the map
	modified.ParticipantScenarios[pt.Participant] = ps

	return modified, nil
}

// SetRetirementDate sets a participant's retirement date to an absolute date.
// Unlike PostponeRetirement which is relative, this sets an exact date.
type SetRetirementDate struct {
	Participant string    // Name of the participant
	Date        time.Time // The new retirement date
}

func (srd *SetRetirementDate) Name() string {
	return "set_retirement_date"
}

func (srd *SetRetirementDate) Description() string {
	return fmt.Sprintf("Set %s's retirement date to %s", srd.Participant, srd.Date.Format("2006-01-02"))
}

func (srd *SetRetirementDate) Validate(base *domain.GenericScenario) error {
	if srd.Participant == "" {
		return NewTransformError(srd.Name(), "validate", "participant name cannot be empty", nil)
	}

	if srd.Date.IsZero() {
		return NewTransformError(srd.Name(), "validate", "date cannot be zero", nil)
	}

	if base == nil {
		return NewTransformError(srd.Name(), "validate", "base scenario cannot be nil", nil)
	}

	_, exists := base.ParticipantScenarios[srd.Participant]
	if !exists {
		return NewTransformError(srd.Name(), "validate", fmt.Sprintf("participant %s not found in scenario", srd.Participant), nil)
	}

	return nil
}

func (srd *SetRetirementDate) Apply(base *domain.GenericScenario) (*domain.GenericScenario, error) {
	// Create a deep copy
	modified := base.DeepCopy()

	// Get the participant scenario
	ps := modified.ParticipantScenarios[srd.Participant]

	// Set the new date
	dateCopy := srd.Date
	ps.RetirementDate = &dateCopy

	// Update the map
	modified.ParticipantScenarios[srd.Participant] = ps

	return modified, nil
}
