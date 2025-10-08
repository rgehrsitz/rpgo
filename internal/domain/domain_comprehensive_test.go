package domain

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestGenericScenario_DeepCopy(t *testing.T) {
	// Create a complex scenario with nested structures
	original := &GenericScenario{
		Name: "Test Scenario",
		ParticipantScenarios: map[string]ParticipantScenario{
			"Alice": {
				ParticipantName:            "Alice",
				SSStartAge:                 62,
				TSPWithdrawalStrategy:      "fixed_amount",
				TSPWithdrawalTargetMonthly: &[]decimal.Decimal{decimal.NewFromInt(3000)}[0],
			},
			"Bob": {
				ParticipantName:       "Bob",
				SSStartAge:            67,
				TSPWithdrawalStrategy: "fixed_percentage",
				TSPWithdrawalRate:     &[]decimal.Decimal{decimal.NewFromFloat(0.04)}[0],
			},
		},
		Mortality: &GenericScenarioMortality{
			Participants: map[string]*MortalitySpec{
				"Alice": {
					DeathAge: &[]int{85}[0],
				},
			},
			Assumptions: &MortalityAssumptions{
				SurvivorSpendingFactor: decimal.NewFromFloat(0.5),
			},
		},
		WithdrawalSequencing: &WithdrawalSequencingConfig{
			Strategy:      "tax_efficient",
			TargetBracket: &[]int{22}[0],
			BracketBuffer: &[]int{1000}[0],
		},
	}

	// Test deep copy
	copied := original.DeepCopy()

	// Verify it's a different instance
	assert.NotSame(t, original, copied)
	assert.Equal(t, original.Name, copied.Name)

	// Verify participant scenarios are copied
	assert.Equal(t, len(original.ParticipantScenarios), len(copied.ParticipantScenarios))
	assert.Equal(t, original.ParticipantScenarios["Alice"].ParticipantName, copied.ParticipantScenarios["Alice"].ParticipantName)
	assert.Equal(t, original.ParticipantScenarios["Alice"].SSStartAge, copied.ParticipantScenarios["Alice"].SSStartAge)

	// Verify mortality is copied
	assert.NotSame(t, original.Mortality, copied.Mortality)
	assert.Equal(t, original.Mortality.Participants["Alice"].DeathAge, copied.Mortality.Participants["Alice"].DeathAge)

	// Verify withdrawal sequencing is copied
	assert.NotSame(t, original.WithdrawalSequencing, copied.WithdrawalSequencing)
	assert.Equal(t, original.WithdrawalSequencing.Strategy, copied.WithdrawalSequencing.Strategy)
	assert.Equal(t, *original.WithdrawalSequencing.TargetBracket, *copied.WithdrawalSequencing.TargetBracket)

	// Test that modifications to copy don't affect original
	copied.Name = "Modified Scenario"
	// Note: Can't modify map values directly, so we'll test a different approach
	copied.Mortality.Participants["Alice"] = &MortalitySpec{DeathAge: &[]int{90}[0]}

	assert.NotEqual(t, original.Name, copied.Name)
	assert.NotEqual(t, *original.Mortality.Participants["Alice"].DeathAge, *copied.Mortality.Participants["Alice"].DeathAge)
}

func TestGenericScenario_DeepCopy_NilFields(t *testing.T) {
	// Test deep copy with nil fields
	original := &GenericScenario{
		Name: "Minimal Scenario",
		ParticipantScenarios: map[string]ParticipantScenario{
			"Alice": {
				ParticipantName: "Alice",
				SSStartAge:      62,
			},
		},
		// Mortality and WithdrawalSequencing are nil
	}

	copied := original.DeepCopy()

	assert.NotSame(t, original, copied)
	assert.Equal(t, original.Name, copied.Name)
	assert.Nil(t, copied.Mortality)
	assert.Nil(t, copied.WithdrawalSequencing)
}

func TestParticipant_Age(t *testing.T) {
	participant := &Participant{
		BirthDate: time.Date(1963, 6, 15, 0, 0, 0, 0, time.UTC),
	}

	testCases := []struct {
		atDate   time.Time
		expected int
		desc     string
	}{
		{
			atDate:   time.Date(2025, 6, 14, 0, 0, 0, 0, time.UTC),
			expected: 61,
			desc:     "day before birthday",
		},
		{
			atDate:   time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
			expected: 62,
			desc:     "on birthday",
		},
		{
			atDate:   time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
			expected: 62,
			desc:     "end of year",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			age := participant.Age(tc.atDate)
			assert.Equal(t, tc.expected, age)
		})
	}
}

func TestParticipant_YearsOfService(t *testing.T) {
	hireDate := time.Date(1985, 3, 20, 0, 0, 0, 0, time.UTC)
	participant := &Participant{
		IsFederal: true,
		HireDate:  &hireDate,
	}

	testCases := []struct {
		atDate   time.Time
		expected string
		desc     string
	}{
		{
			atDate:   time.Date(2025, 3, 19, 0, 0, 0, 0, time.UTC),
			expected: "39.9973",
			desc:     "day before anniversary",
		},
		{
			atDate:   time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC),
			expected: "40.0000",
			desc:     "on anniversary",
		},
		{
			atDate:   time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
			expected: "40.7830",
			desc:     "end of year",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			years := participant.YearsOfService(tc.atDate)
			assert.Equal(t, tc.expected, years.StringFixed(4))
		})
	}
}

func TestParticipant_YearsOfService_NilHireDate(t *testing.T) {
	participant := &Participant{
		// HireDate is nil
	}

	atDate := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	years := participant.YearsOfService(atDate)

	// Should return zero when hire date is nil
	assert.True(t, years.Equal(decimal.Zero))
}

func TestParticipant_TotalTSPBalance(t *testing.T) {
	participant := &Participant{
		IsFederal:             true,
		TSPBalanceTraditional: &[]decimal.Decimal{decimal.NewFromInt(450000)}[0],
		TSPBalanceRoth:        &[]decimal.Decimal{decimal.NewFromInt(50000)}[0],
	}

	total := participant.TotalTSPBalance()
	expected := decimal.NewFromInt(500000)
	assert.True(t, total.Equal(expected))
}

func TestParticipant_TotalTSPBalance_NilBalances(t *testing.T) {
	participant := &Participant{
		// Both balances are nil
	}

	total := participant.TotalTSPBalance()
	assert.True(t, total.Equal(decimal.Zero))
}

func TestParticipant_AnnualTSPContribution(t *testing.T) {
	participant := &Participant{
		IsFederal:              true,
		CurrentSalary:          &[]decimal.Decimal{decimal.NewFromInt(95000)}[0],
		TSPContributionPercent: &[]decimal.Decimal{decimal.NewFromFloat(0.15)}[0],
	}

	contribution := participant.AnnualTSPContribution()
	expected := decimal.NewFromInt(14250)
	assert.True(t, contribution.Equal(expected))
}

func TestParticipant_AnnualTSPContribution_NilValues(t *testing.T) {
	participant := &Participant{
		// Both values are nil
	}

	contribution := participant.AnnualTSPContribution()
	assert.True(t, contribution.Equal(decimal.Zero))
}

func TestParticipant_AgencyMatch(t *testing.T) {
	participant := &Participant{
		IsFederal:              true,
		CurrentSalary:          &[]decimal.Decimal{decimal.NewFromInt(95000)}[0],
		TSPContributionPercent: &[]decimal.Decimal{decimal.NewFromFloat(0.15)}[0],
	}

	match := participant.AgencyMatch()
	expected := decimal.NewFromInt(4750) // 95000 * 0.05
	assert.True(t, match.Equal(expected))
}

func TestParticipant_AgencyMatch_LimitedByContribution(t *testing.T) {
	participant := &Participant{
		CurrentSalary:          &[]decimal.Decimal{decimal.NewFromInt(95000)}[0],
		TSPContributionPercent: &[]decimal.Decimal{decimal.NewFromFloat(0.03)}[0], // Only 3%
	}

	match := participant.AgencyMatch()
	assert.True(t, match.Equal(decimal.Zero)) // No match because contribution < 5%
}

func TestHousehold_GetFederalParticipants(t *testing.T) {
	household := &Household{
		Participants: []Participant{
			{
				Name:      "Alice",
				IsFederal: true,
			},
			{
				Name:      "Bob",
				IsFederal: false,
			},
			{
				Name:      "Charlie",
				IsFederal: true,
			},
		},
	}

	federalParticipants := household.GetFederalParticipants()
	assert.Len(t, federalParticipants, 2)
	assert.Equal(t, "Alice", federalParticipants[0].Name)
	assert.Equal(t, "Charlie", federalParticipants[1].Name)
}

func TestHousehold_GetFederalParticipants_NoFederal(t *testing.T) {
	household := &Household{
		Participants: []Participant{
			{
				Name:      "Alice",
				IsFederal: false,
			},
			{
				Name:      "Bob",
				IsFederal: false,
			},
		},
	}

	federalParticipants := household.GetFederalParticipants()
	assert.Len(t, federalParticipants, 0)
}

func TestHousehold_AgesAt(t *testing.T) {
	household := &Household{
		Participants: []Participant{
			{
				Name:      "Alice",
				BirthDate: time.Date(1963, 6, 15, 0, 0, 0, 0, time.UTC),
			},
			{
				Name:      "Bob",
				BirthDate: time.Date(1965, 3, 20, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	atDate := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	ages := household.AgesAt(atDate)

	assert.Len(t, ages, 2)
	assert.Equal(t, 62, ages["Alice"])
	assert.Equal(t, 60, ages["Bob"])
}

func TestHousehold_Survivors(t *testing.T) {
	household := &Household{
		Participants: []Participant{
			{
				Name:      "Alice",
				BirthDate: time.Date(1963, 6, 15, 0, 0, 0, 0, time.UTC),
			},
			{
				Name:      "Bob",
				BirthDate: time.Date(1965, 3, 20, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	mortality := map[string]*MortalitySpec{
		"Alice": {
			DeathAge: &[]int{85}[0], // Dies at age 85 (year 2048)
		},
		"Bob": {
			DeathAge: &[]int{90}[0], // Dies at age 90 (year 2055)
		},
	}

	// Test survivors in year 2045 (Alice age 82, Bob age 80) - both alive
	survivors2045 := household.Survivors(2045, mortality)
	assert.Len(t, survivors2045, 2)

	// Test survivors in year 2050 (Alice age 87, Bob age 85) - Alice dead, Bob alive
	survivors2050 := household.Survivors(2050, mortality)
	assert.Len(t, survivors2050, 1)
	assert.Equal(t, "Bob", survivors2050[0].Name)

	// Test survivors in year 2060 (Alice age 97, Bob age 95) - both dead
	survivors2060 := household.Survivors(2060, mortality)
	assert.Len(t, survivors2060, 0)
}

func TestHousehold_Survivors_NoMortality(t *testing.T) {
	household := &Household{
		Participants: []Participant{
			{
				Name:      "Alice",
				BirthDate: time.Date(1963, 6, 15, 0, 0, 0, 0, time.UTC),
			},
			{
				Name:      "Bob",
				BirthDate: time.Date(1965, 3, 20, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	// No mortality specification - everyone survives
	survivors := household.Survivors(2050, nil)
	assert.Len(t, survivors, 2)
}

func TestGlobalAssumptions_GenerateAssumptions(t *testing.T) {
	assumptions := &GlobalAssumptions{
		InflationRate:           decimal.NewFromFloat(0.025),
		TSPReturnPreRetirement:  decimal.NewFromFloat(0.06),
		TSPReturnPostRetirement: decimal.NewFromFloat(0.04),
		COLAGeneralRate:         decimal.NewFromFloat(0.02),
		FEHBPremiumInflation:    decimal.NewFromFloat(0.06),
		ProjectionYears:         30,
	}

	generated := assumptions.GenerateAssumptions()

	// Should generate a list of assumption strings
	assert.NotEmpty(t, generated)
	assert.Contains(t, generated, "General COLA (FERS pension & SS): 2.0% annually")
	assert.Contains(t, generated, "FEHB premium inflation: 6.0% annually")
	assert.Contains(t, generated, "TSP growth pre-retirement: 6.0% annually")
	assert.Contains(t, generated, "TSP growth post-retirement: 4.0% annually")
	assert.Contains(t, generated, "Social Security wage base indexing: ~5% annually (2025 est: $168,600)")
	assert.Contains(t, generated, "Tax brackets: 2025 levels held constant (no inflation indexing)")
}

func TestMortalitySpec_Validation(t *testing.T) {
	// Test with death age
	deathAge := 85
	spec := &MortalitySpec{
		DeathAge: &deathAge,
	}

	assert.Equal(t, 85, *spec.DeathAge)
	assert.Nil(t, spec.DeathDate)

	// Test with death date
	deathDate := time.Date(2048, 6, 15, 0, 0, 0, 0, time.UTC)
	spec2 := &MortalitySpec{
		DeathDate: &deathDate,
	}

	assert.Equal(t, deathDate, *spec2.DeathDate)
	assert.Nil(t, spec2.DeathAge)
}

func TestMortalityAssumptions_Validation(t *testing.T) {
	assumptions := &MortalityAssumptions{
		SurvivorSpendingFactor: decimal.NewFromFloat(0.5),
		TSPSpousalTransfer:     "merge",
		FilingStatusSwitch:     "immediate",
	}

	assert.True(t, assumptions.SurvivorSpendingFactor.Equal(decimal.NewFromFloat(0.5)))
	assert.Equal(t, "merge", assumptions.TSPSpousalTransfer)
	assert.Equal(t, "immediate", assumptions.FilingStatusSwitch)
}

func TestWithdrawalSequencingConfig_Validation(t *testing.T) {
	config := &WithdrawalSequencingConfig{
		Strategy:       "tax_efficient",
		CustomSequence: []string{"taxable", "roth", "traditional"},
		TargetBracket:  &[]int{22}[0],
		BracketBuffer:  &[]int{1000}[0],
	}

	assert.Equal(t, "tax_efficient", config.Strategy)
	assert.Len(t, config.CustomSequence, 3)
	assert.Equal(t, "taxable", config.CustomSequence[0])
	assert.Equal(t, "roth", config.CustomSequence[1])
	assert.Equal(t, "traditional", config.CustomSequence[2])
	assert.Equal(t, 22, *config.TargetBracket)
	assert.Equal(t, 1000, *config.BracketBuffer)
}
