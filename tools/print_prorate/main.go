package main

import (
	"fmt"
	"time"

	"github.com/rgehrsitz/rpgo/internal/calculation"
	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/rgehrsitz/rpgo/pkg/dateutil"
	"github.com/shopspring/decimal"
)

func main() {
	ce := calculation.NewCalculationEngine()

	fmt.Println("SS Scenario projection row 0:")
	ssProjection := runProjection(ce, ssHousehold(), ssScenario(), baseAssumptions())
	if len(ssProjection) == 0 {
		fmt.Println("no projection data")
		return
	}
	row := ssProjection[0]
	fmt.Printf("SSBenefits[person_a]: %s\n", row.SSBenefits["person_a"].StringFixed(2))
	fmt.Printf("SSBenefits[person_b]: %s\n", row.SSBenefits["person_b"].StringFixed(2))

	full := calculation.CalculateSSBenefitForYear(ssEmployee(), 62, 0, decimal.Zero)
	fmt.Printf("Full-year SS (calc): %s\n", full.StringFixed(2))

	fmt.Println("\nRMD Scenario projection row 0:")
	rmdProjection := runProjection(ce, rmdHousehold(), rmdScenario(), baseAssumptions())
	if len(rmdProjection) == 0 {
		fmt.Println("no projection data")
		return
	}
	rmdRow := rmdProjection[0]
	fmt.Printf("TSPWithdrawals[person_a]: %s\n", rmdRow.TSPWithdrawals["person_a"].StringFixed(2))
	fmt.Printf("TSPBalances[person_a]: %s\n", rmdRow.TSPBalances["person_a"].StringFixed(2))

	fullRMD := calculation.CalculateRMD(decimal.RequireFromString("500000"), 1953, dateutil.GetRMDAge(1953))
	fmt.Printf("Full RMD: %s\n", fullRMD.StringFixed(2))
}

func runProjection(ce *calculation.CalculationEngine, household *domain.Household, scenario *domain.GenericScenario, assumptions domain.GlobalAssumptions) []domain.AnnualCashFlow {
	return ce.GenerateAnnualProjectionGeneric(household, scenario, &assumptions, assumptions.FederalRules)
}

func baseAssumptions() domain.GlobalAssumptions {
	return domain.GlobalAssumptions{
		InflationRate:           decimal.Zero,
		FEHBPremiumInflation:    decimal.Zero,
		TSPReturnPreRetirement:  decimal.Zero,
		TSPReturnPostRetirement: decimal.Zero,
		COLAGeneralRate:         decimal.Zero,
		ProjectionYears:         3,
	}
}

func ssHousehold() *domain.Household {
	return &domain.Household{
		FilingStatus: "married_filing_jointly",
		Participants: []domain.Participant{
			{
				Name:                           "person_a",
				BirthDate:                      time.Date(1963, 7, 31, 0, 0, 0, 0, time.UTC),
				IsFederal:                      true,
				HireDate:                       timePtr(time.Date(1995, 7, 11, 0, 0, 0, 0, time.UTC)),
				CurrentSalary:                  decimalPtr("0"),
				High3Salary:                    decimalPtr("176620"),
				TSPBalanceTraditional:          decimalPtr("1525175.90"),
				TSPBalanceRoth:                 decimalPtr("0"),
				TSPContributionPercent:         decimalPtr("0.15"),
				SSBenefitFRA:                   decimal.RequireFromString("3826"),
				SSBenefit62:                    decimal.RequireFromString("2527"),
				SSBenefit70:                    decimal.RequireFromString("4860"),
				FEHBPremiumPerPayPeriod:        decimalPtr("0"),
				SurvivorBenefitElectionPercent: decimalPtr("0"),
			},
			{
				Name:                           "person_b",
				BirthDate:                      time.Date(1965, 2, 25, 0, 0, 0, 0, time.UTC),
				IsFederal:                      true,
				HireDate:                       timePtr(time.Date(1987, 6, 22, 0, 0, 0, 0, time.UTC)),
				CurrentSalary:                  decimalPtr("0"),
				High3Salary:                    decimalPtr("190779"),
				TSPBalanceTraditional:          decimalPtr("1966168.86"),
				TSPBalanceRoth:                 decimalPtr("0"),
				TSPContributionPercent:         decimalPtr("0.15"),
				SSBenefitFRA:                   decimal.RequireFromString("4012"),
				SSBenefit62:                    decimal.RequireFromString("2795"),
				SSBenefit70:                    decimal.RequireFromString("5000"),
				FEHBPremiumPerPayPeriod:        decimalPtr("875"),
				SurvivorBenefitElectionPercent: decimalPtr("0"),
				IsPrimaryFEHBHolder:            true,
			},
		},
	}
}

func ssScenario() *domain.GenericScenario {
	return &domain.GenericScenario{
		Name: "ss-prorate",
		ParticipantScenarios: map[string]domain.ParticipantScenario{
			"person_a": {
				ParticipantName:       "person_a",
				RetirementDate:        timePtr(time.Date(2025, 8, 30, 0, 0, 0, 0, time.UTC)),
				SSStartAge:            62,
				TSPWithdrawalStrategy: "4_percent_rule",
			},
			"person_b": {
				ParticipantName:            "person_b",
				RetirementDate:             timePtr(time.Date(2027, 2, 28, 0, 0, 0, 0, time.UTC)),
				SSStartAge:                 62,
				TSPWithdrawalStrategy:      "4_percent_rule",
				TSPWithdrawalTargetMonthly: decimalPtr("3000"),
			},
		},
	}
}

func ssEmployee() *domain.Employee {
	return &domain.Employee{
		Name:                           "PersonA",
		BirthDate:                      time.Date(1963, 7, 31, 0, 0, 0, 0, time.UTC),
		HireDate:                       time.Date(1995, 7, 11, 0, 0, 0, 0, time.UTC),
		CurrentSalary:                  decimal.Zero,
		High3Salary:                    decimal.RequireFromString("176620"),
		TSPBalanceTraditional:          decimal.RequireFromString("1525175.90"),
		TSPBalanceRoth:                 decimal.Zero,
		TSPContributionPercent:         decimal.RequireFromString("0.15"),
		SSBenefitFRA:                   decimal.RequireFromString("3826"),
		SSBenefit62:                    decimal.RequireFromString("2527"),
		SSBenefit70:                    decimal.RequireFromString("4860"),
		FEHBPremiumPerPayPeriod:        decimal.Zero,
		SurvivorBenefitElectionPercent: decimal.Zero,
	}
}

func rmdHousehold() *domain.Household {
	return &domain.Household{
		FilingStatus: "married_filing_jointly",
		Participants: []domain.Participant{
			{
				Name:                   "person_a",
				BirthDate:              time.Date(1953, 7, 1, 0, 0, 0, 0, time.UTC),
				IsFederal:              true,
				HireDate:               timePtr(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)),
				CurrentSalary:          decimalPtr("0"),
				High3Salary:            decimalPtr("0"),
				TSPBalanceTraditional:  decimalPtr("500000"),
				TSPBalanceRoth:         decimalPtr("0"),
				TSPContributionPercent: decimalPtr("0.0"),
				SSBenefitFRA:           decimal.RequireFromString("1800"),
				SSBenefit62:            decimal.RequireFromString("1600"),
				SSBenefit70:            decimal.RequireFromString("2200"),
			},
			{
				Name:                   "person_b",
				BirthDate:              time.Date(1965, 1, 1, 0, 0, 0, 0, time.UTC),
				IsFederal:              true,
				HireDate:               timePtr(time.Date(1992, 1, 1, 0, 0, 0, 0, time.UTC)),
				CurrentSalary:          decimalPtr("0"),
				High3Salary:            decimalPtr("0"),
				TSPBalanceTraditional:  decimalPtr("0"),
				TSPBalanceRoth:         decimalPtr("0"),
				TSPContributionPercent: decimalPtr("0.0"),
				SSBenefitFRA:           decimal.RequireFromString("1500"),
				SSBenefit62:            decimal.RequireFromString("1300"),
				SSBenefit70:            decimal.RequireFromString("1900"),
			},
		},
	}
}

func rmdScenario() *domain.GenericScenario {
	return &domain.GenericScenario{
		Name: "rmd-prorate",
		ParticipantScenarios: map[string]domain.ParticipantScenario{
			"person_a": {
				ParticipantName:       "person_a",
				RetirementDate:        timePtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
				SSStartAge:            62,
				TSPWithdrawalStrategy: "4_percent_rule",
			},
			"person_b": {
				ParticipantName:       "person_b",
				RetirementDate:        timePtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
				SSStartAge:            62,
				TSPWithdrawalStrategy: "4_percent_rule",
			},
		},
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func decimalPtr(value string) *decimal.Decimal {
	d := decimal.RequireFromString(value)
	return &d
}
