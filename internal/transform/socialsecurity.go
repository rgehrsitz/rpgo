package transform

import (
	"fmt"

	"github.com/rgehrsitz/rpgo/internal/domain"
)

// DelaySSClaim changes the Social Security claiming age for a participant.
// Delaying SS increases monthly benefits (8% per year from FRA to 70).
type DelaySSClaim struct {
	Participant string // Name of the participant
	NewAge      int    // New SS start age (must be 62-70)
}

func (dss *DelaySSClaim) Name() string {
	return "delay_ss_claim"
}

func (dss *DelaySSClaim) Description() string {
	return fmt.Sprintf("Change %s's Social Security start age to %d", dss.Participant, dss.NewAge)
}

func (dss *DelaySSClaim) Validate(base *domain.GenericScenario) error {
	if dss.Participant == "" {
		return NewTransformError(dss.Name(), "validate", "participant name cannot be empty", nil)
	}

	if dss.NewAge < 62 || dss.NewAge > 70 {
		return NewTransformError(dss.Name(), "validate", fmt.Sprintf("SS start age must be between 62 and 70, got %d", dss.NewAge), nil)
	}

	if base == nil {
		return NewTransformError(dss.Name(), "validate", "base scenario cannot be nil", nil)
	}

	_, exists := base.ParticipantScenarios[dss.Participant]
	if !exists {
		return NewTransformError(dss.Name(), "validate", fmt.Sprintf("participant %s not found in scenario", dss.Participant), nil)
	}

	return nil
}

func (dss *DelaySSClaim) Apply(base *domain.GenericScenario) (*domain.GenericScenario, error) {
	// Create a deep copy
	modified := base.DeepCopy()

	// Get the participant scenario
	ps := modified.ParticipantScenarios[dss.Participant]

	// Set the new SS start age
	ps.SSStartAge = dss.NewAge

	// Update the map
	modified.ParticipantScenarios[dss.Participant] = ps

	return modified, nil
}
