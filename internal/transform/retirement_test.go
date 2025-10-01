package transform

import (
	"testing"
	"time"
)

func TestPostponeRetirement_Validate(t *testing.T) {
	tests := []struct {
		name        string
		transform   *PostponeRetirement
		expectError bool
	}{
		{
			name:        "Valid postponement",
			transform:   &PostponeRetirement{Participant: "Alice", Months: 12},
			expectError: false,
		},
		{
			name:        "Zero months (valid)",
			transform:   &PostponeRetirement{Participant: "Alice", Months: 0},
			expectError: false,
		},
		{
			name:        "Empty participant name",
			transform:   &PostponeRetirement{Participant: "", Months: 12},
			expectError: true,
		},
		{
			name:        "Negative months",
			transform:   &PostponeRetirement{Participant: "Alice", Months: -6},
			expectError: true,
		},
		{
			name:        "Non-existent participant",
			transform:   &PostponeRetirement{Participant: "Charlie", Months: 12},
			expectError: true,
		},
	}

	base := createTestScenario()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.transform.Validate(base)
			if tt.expectError && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error, got: %v", err)
			}
		})
	}
}

func TestPostponeRetirement_Apply(t *testing.T) {
	tests := []struct {
		name           string
		participant    string
		months         int
		expectedOffset int
	}{
		{"Postpone 6 months", "Alice", 6, 6},
		{"Postpone 12 months", "Alice", 12, 12},
		{"Postpone 24 months", "Alice", 24, 24},
		{"Postpone 0 months", "Alice", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := createTestScenario()
			originalDate := *base.ParticipantScenarios[tt.participant].RetirementDate

			transform := &PostponeRetirement{
				Participant: tt.participant,
				Months:      tt.months,
			}

			result, err := transform.Apply(base)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			newDate := *result.ParticipantScenarios[tt.participant].RetirementDate
			expectedDate := originalDate.AddDate(0, tt.expectedOffset, 0)

			if !newDate.Equal(expectedDate) {
				t.Errorf("Expected date %v, got %v", expectedDate, newDate)
			}

			// Verify original is unchanged
			if !base.ParticipantScenarios[tt.participant].RetirementDate.Equal(originalDate) {
				t.Error("Original scenario was modified")
			}

			// Verify other participants are unchanged
			if tt.participant == "Alice" {
				bobDateBefore := *base.ParticipantScenarios["Bob"].RetirementDate
				bobDateAfter := *result.ParticipantScenarios["Bob"].RetirementDate
				if !bobDateAfter.Equal(bobDateBefore) {
					t.Error("Other participant's date was modified")
				}
			}
		})
	}
}

func TestSetRetirementDate_Validate(t *testing.T) {
	tests := []struct {
		name        string
		transform   *SetRetirementDate
		expectError bool
	}{
		{
			name:        "Valid date",
			transform:   &SetRetirementDate{Participant: "Alice", Date: time.Date(2028, 12, 31, 0, 0, 0, 0, time.UTC)},
			expectError: false,
		},
		{
			name:        "Empty participant",
			transform:   &SetRetirementDate{Participant: "", Date: time.Date(2028, 12, 31, 0, 0, 0, 0, time.UTC)},
			expectError: true,
		},
		{
			name:        "Zero date",
			transform:   &SetRetirementDate{Participant: "Alice", Date: time.Time{}},
			expectError: true,
		},
		{
			name:        "Non-existent participant",
			transform:   &SetRetirementDate{Participant: "Charlie", Date: time.Date(2028, 12, 31, 0, 0, 0, 0, time.UTC)},
			expectError: true,
		},
	}

	base := createTestScenario()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.transform.Validate(base)
			if tt.expectError && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error, got: %v", err)
			}
		})
	}
}

func TestSetRetirementDate_Apply(t *testing.T) {
	base := createTestScenario()
	newDate := time.Date(2029, 3, 15, 0, 0, 0, 0, time.UTC)

	transform := &SetRetirementDate{
		Participant: "Alice",
		Date:        newDate,
	}

	result, err := transform.Apply(base)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	actualDate := *result.ParticipantScenarios["Alice"].RetirementDate
	if !actualDate.Equal(newDate) {
		t.Errorf("Expected date %v, got %v", newDate, actualDate)
	}

	// Verify original is unchanged
	originalDate := *base.ParticipantScenarios["Alice"].RetirementDate
	expectedOriginal := time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC)
	if !originalDate.Equal(expectedOriginal) {
		t.Error("Original scenario was modified")
	}
}

func TestPostponeRetirement_NameAndDescription(t *testing.T) {
	transform := &PostponeRetirement{
		Participant: "Alice",
		Months:      12,
	}

	if transform.Name() != "postpone_retirement" {
		t.Errorf("Expected name 'postpone_retirement', got %s", transform.Name())
	}

	expectedDesc := "Postpone Alice's retirement by 12 months"
	if transform.Description() != expectedDesc {
		t.Errorf("Expected description %q, got %q", expectedDesc, transform.Description())
	}
}
