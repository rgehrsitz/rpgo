package config

import (
	"testing"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

func TestConfigurationFormats(t *testing.T) {
	// Test format detection
	legacyConfig := &domain.Configuration{
		PersonalDetails: map[string]domain.Employee{
			"robert": {},
			"dawn":   {},
		},
	}

	if !legacyConfig.IsLegacyFormat() {
		t.Error("Should detect legacy format")
	}

	if legacyConfig.IsNewFormat() {
		t.Error("Should not detect new format for legacy config")
	}

	newConfig := &domain.Configuration{
		Household: &domain.Household{
			Participants: []domain.Participant{{}},
		},
	}

	if !newConfig.IsNewFormat() {
		t.Error("Should detect new format")
	}

	if newConfig.IsLegacyFormat() {
		t.Error("Should not detect legacy format for new config")
	}
}

func TestParticipantValidation(t *testing.T) {
	hireDate := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	retirementDate := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	validConfig := &domain.Configuration{
		Household: &domain.Household{
			FilingStatus: "single",
			Participants: []domain.Participant{
				{
					Name:                           "John Doe",
					IsFederal:                      true,
					BirthDate:                      time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
					HireDate:                       &hireDate,
					CurrentSalary:                  &[]decimal.Decimal{decimal.NewFromInt(100000)}[0],
					High3Salary:                    &[]decimal.Decimal{decimal.NewFromInt(95000)}[0],
					TSPBalanceTraditional:          &[]decimal.Decimal{decimal.NewFromInt(400000)}[0],
					TSPBalanceRoth:                 &[]decimal.Decimal{decimal.NewFromInt(50000)}[0],
					TSPContributionPercent:         &[]decimal.Decimal{decimal.NewFromFloat(0.15)}[0],
					SSBenefitFRA:                   decimal.NewFromInt(2500),
					SSBenefit62:                    decimal.NewFromInt(1750),
					SSBenefit70:                    decimal.NewFromInt(3100),
					FEHBPremiumPerPayPeriod:        &[]decimal.Decimal{decimal.NewFromInt(500)}[0],
					IsPrimaryFEHBHolder:            true,
					SurvivorBenefitElectionPercent: &[]decimal.Decimal{decimal.Zero}[0],
				},
			},
		},
		GenericScenarios: []domain.GenericScenario{
			{
				Name: "Test Scenario",
				ParticipantScenarios: map[string]domain.ParticipantScenario{
					"John Doe": {
						ParticipantName:       "John Doe",
						RetirementDate:        &retirementDate,
						SSStartAge:            62,
						TSPWithdrawalStrategy: "4_percent_rule",
					},
				},
			},
		},
		GlobalAssumptions: domain.GlobalAssumptions{
			InflationRate:           decimal.NewFromFloat(0.025),
			FEHBPremiumInflation:    decimal.NewFromFloat(0.06),
			TSPReturnPreRetirement:  decimal.NewFromFloat(0.07),
			TSPReturnPostRetirement: decimal.NewFromFloat(0.06),
			COLAGeneralRate:         decimal.NewFromFloat(0.025),
			ProjectionYears:         25,
			CurrentLocation: domain.Location{
				State:        "TestState",
				County:       "TestCounty",
				Municipality: "TestMunicipality",
			},
		},
	}

	parser := NewInputParser()
	err := parser.ValidateConfiguration(validConfig)
	if err != nil {
		t.Errorf("Expected valid configuration but got error: %s", err.Error())
	}
}

func TestConversion(t *testing.T) {
	// Test Employee -> Participant -> Employee round trip
	employee := domain.Employee{
		Name:                           "Test Employee",
		BirthDate:                      time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		HireDate:                       time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		CurrentSalary:                  decimal.NewFromInt(100000),
		High3Salary:                    decimal.NewFromInt(95000),
		TSPBalanceTraditional:          decimal.NewFromInt(400000),
		TSPBalanceRoth:                 decimal.NewFromInt(50000),
		TSPContributionPercent:         decimal.NewFromFloat(0.15),
		SSBenefitFRA:                   decimal.NewFromInt(2500),
		SSBenefit62:                    decimal.NewFromInt(1750),
		SSBenefit70:                    decimal.NewFromInt(3100),
		FEHBPremiumPerPayPeriod:        decimal.NewFromInt(500),
		SurvivorBenefitElectionPercent: decimal.NewFromFloat(0.25),
	}

	// Convert to Participant
	participant := domain.ParticipantFromEmployee(&employee)
	if participant.Name != employee.Name {
		t.Errorf("Expected name %s, got %s", employee.Name, participant.Name)
	}

	if !participant.IsFederal {
		t.Error("Expected participant to be federal")
	}

	// Convert back to Employee
	backToEmployee, err := participant.ToEmployee()
	if err != nil {
		t.Fatalf("Failed to convert back to employee: %s", err)
	}

	// Verify they match
	if employee.Name != backToEmployee.Name {
		t.Errorf("Names don't match: %s vs %s", employee.Name, backToEmployee.Name)
	}
	if !employee.CurrentSalary.Equal(backToEmployee.CurrentSalary) {
		t.Errorf("Salaries don't match: %s vs %s", employee.CurrentSalary, backToEmployee.CurrentSalary)
	}
	if !employee.SSBenefitFRA.Equal(backToEmployee.SSBenefitFRA) {
		t.Errorf("SS benefits don't match: %s vs %s", employee.SSBenefitFRA, backToEmployee.SSBenefitFRA)
	}
}
