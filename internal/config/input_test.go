package config

import (
	"testing"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// (Legacy format detection test removed - generic only)

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
		Scenarios: []domain.GenericScenario{
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

// (Legacy conversion test removed)
