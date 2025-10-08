package calculation

import (
	"testing"
	"time"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestCalculateFERSSupplementYear(t *testing.T) {
	employee := &domain.Employee{
		BirthDate:   time.Date(1965, 1, 1, 0, 0, 0, 0, time.UTC),
		HireDate:    time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		SSBenefit62: decimal.NewFromInt(25000),
	}

	retirementDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	inflationRate := decimal.NewFromFloat(0.025)

	tests := []struct {
		name                 string
		yearsSinceRetirement int
		expectedZero         bool
		description          string
	}{
		{
			name:                 "Negative years",
			yearsSinceRetirement: -1,
			expectedZero:         true,
			description:          "Should return zero for negative years",
		},
		{
			name:                 "Year 0 (retirement year)",
			yearsSinceRetirement: 0,
			expectedZero:         false,
			description:          "Should calculate SRS for retirement year",
		},
		{
			name:                 "Year 5",
			yearsSinceRetirement: 5,
			expectedZero:         true, // Age 65, SRS stops at 62
			description:          "Should return zero when age >= 62",
		},
		{
			name:                 "Age 62 (SRS stops)",
			yearsSinceRetirement: 2, // Age 62 in 2027
			expectedZero:         true,
			description:          "Should return zero when age >= 62",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateFERSSupplementYear(employee, retirementDate, tt.yearsSinceRetirement, inflationRate)

			if tt.expectedZero {
				assert.True(t, result.IsZero(), tt.description)
			} else {
				// Debug: print the actual result to understand why it's zero
				t.Logf("Result for %s: %s", tt.name, result.String())
				assert.True(t, result.GreaterThan(decimal.Zero), tt.description)
			}
		})
	}
}

func TestCalculateFEHBPremium(t *testing.T) {
	employee := &domain.Employee{
		FEHBPremiumPerPayPeriod: decimal.NewFromInt(200), // $200 per pay period
	}

	fehbConfig := domain.FEHBConfig{
		PayPeriodsPerYear: 26,
	}

	premiumInflation := decimal.NewFromFloat(0.06) // 6% annual inflation

	tests := []struct {
		name        string
		year        int
		expectedMin decimal.Decimal
		description string
	}{
		{
			name:        "Year 0 (base year)",
			year:        0,
			expectedMin: decimal.NewFromInt(5200), // 200 * 26 = 5200
			description: "Should calculate base premium",
		},
		{
			name:        "Year 1",
			year:        1,
			expectedMin: decimal.NewFromInt(5500), // 5200 * 1.06
			description: "Should apply inflation for year 1",
		},
		{
			name:        "Year 5",
			year:        5,
			expectedMin: decimal.NewFromInt(6900), // 5200 * 1.06^5
			description: "Should apply compound inflation for year 5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateFEHBPremium(employee, tt.year, premiumInflation, fehbConfig)

			assert.True(t, result.GreaterThanOrEqual(tt.expectedMin),
				"%s: Expected at least %s, got %s",
				tt.description, tt.expectedMin.StringFixed(0), result.StringFixed(0))
		})
	}
}

func TestCalculateRMD(t *testing.T) {
	tests := []struct {
		name        string
		balance     decimal.Decimal
		birthYear   int
		age         int
		expectedMin decimal.Decimal
		description string
	}{
		{
			name:        "Age 72, born 1950",
			balance:     decimal.NewFromInt(1000000),
			birthYear:   1950,
			age:         72,
			expectedMin: decimal.NewFromInt(36000), // 1M / 27.4
			description: "Should calculate RMD for age 72",
		},
		{
			name:        "Age 80, born 1950",
			balance:     decimal.NewFromInt(500000),
			birthYear:   1950,
			age:         80,
			expectedMin: decimal.NewFromInt(20000), // 500K / 25.6
			description: "Should calculate RMD for age 80",
		},
		{
			name:        "Age 90, born 1950",
			balance:     decimal.NewFromInt(200000),
			birthYear:   1950,
			age:         90,
			expectedMin: decimal.NewFromInt(10000), // 200K / 20.2
			description: "Should calculate RMD for age 90",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateRMD(tt.balance, tt.birthYear, tt.age)

			assert.True(t, result.GreaterThanOrEqual(tt.expectedMin),
				"%s: Expected at least %s, got %s",
				tt.description, tt.expectedMin.StringFixed(0), result.StringFixed(0))
		})
	}
}

func TestNewPartTimeWorkCalculator(t *testing.T) {
	calculator := NewPartTimeWorkCalculator()

	assert.NotNil(t, calculator, "Should create calculator")
	assert.NotNil(t, calculator.earningsTest, "Should initialize earnings test")
}

func TestPartTimeWorkCalculator_CalculatePartTimeWorkForYear_NoPartTimeWork(t *testing.T) {
	calculator := NewPartTimeWorkCalculator()

	participant := domain.Participant{
		Name: "test",
	}

	participantScenario := domain.ParticipantScenario{
		// No PartTimeWork configured
	}

	result, err := calculator.CalculatePartTimeWorkForYear(participant, participantScenario, 2025, 60)

	assert.NoError(t, err, "Should not error when no part-time work")
	assert.NotNil(t, result, "Should return result")
	assert.False(t, result.IsPartTime, "Should not be part-time work")
	assert.Equal(t, 2025, result.Year, "Should set correct year")
}

func TestPartTimeWorkCalculator_CalculatePartTimeWorkForYear_NoActivePeriod(t *testing.T) {
	calculator := NewPartTimeWorkCalculator()

	participant := domain.Participant{
		Name: "test",
	}

	// Create a part-time work schedule that doesn't cover the test year
	partTimeWork := &domain.PartTimeWorkSchedule{
		StartDate: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2035, 12, 31, 0, 0, 0, 0, time.UTC),
		Schedule: []domain.PartTimeWorkPeriod{
			{
				PeriodStart:            time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
				PeriodEnd:              time.Date(2035, 12, 31, 0, 0, 0, 0, time.UTC),
				AnnualSalary:           decimal.NewFromInt(50000),
				TSPContributionPercent: decimal.NewFromFloat(0.05),
				WorkType:               "w2",
				HoursPerWeek:           decimalPtr(decimal.NewFromFloat(20)),
			},
		},
	}

	participantScenario := domain.ParticipantScenario{
		PartTimeWork: partTimeWork,
	}

	result, err := calculator.CalculatePartTimeWorkForYear(participant, participantScenario, 2025, 60)

	assert.NoError(t, err, "Should not error when no active period")
	assert.NotNil(t, result, "Should return result")
	assert.False(t, result.IsPartTime, "Should not be part-time work")
	assert.Equal(t, 2025, result.Year, "Should set correct year")
}

func TestPartTimeWorkCalculator_CalculatePartTimeWorkForYear_ActivePeriod(t *testing.T) {
	calculator := NewPartTimeWorkCalculator()

	participant := domain.Participant{
		Name:          "test",
		CurrentSalary: decimalPtr(decimal.NewFromInt(100000)),
	}

	// Create a part-time work schedule that covers the test year
	partTimeWork := &domain.PartTimeWorkSchedule{
		StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2027, 12, 31, 0, 0, 0, 0, time.UTC),
		Schedule: []domain.PartTimeWorkPeriod{
			{
				PeriodStart:            time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				PeriodEnd:              time.Date(2027, 12, 31, 0, 0, 0, 0, time.UTC),
				AnnualSalary:           decimal.NewFromInt(50000),
				TSPContributionPercent: decimal.NewFromFloat(0.05),
				WorkType:               "w2",
				HoursPerWeek:           decimalPtr(decimal.NewFromFloat(20)),
			},
		},
	}

	participantScenario := domain.ParticipantScenario{
		PartTimeWork: partTimeWork,
	}

	result, err := calculator.CalculatePartTimeWorkForYear(participant, participantScenario, 2025, 60)

	assert.NoError(t, err, "Should not error for active period")
	assert.NotNil(t, result, "Should return result")
	assert.True(t, result.IsPartTime, "Should be part-time work")
	assert.Equal(t, 2025, result.Year, "Should set correct year")
	assert.True(t, result.AnnualSalary.GreaterThan(decimal.Zero), "Should calculate annual salary")
}

// Helper function for creating decimal pointers
func decimalPtr(d decimal.Decimal) *decimal.Decimal {
	return &d
}
