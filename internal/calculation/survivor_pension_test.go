package calculation

import (
	"testing"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// TestSurvivorPensionFlow verifies survivor annuity begins the year of death and replaces deceased pension with elected percentage.
func TestSurvivorPensionFlow(t *testing.T) {
	// Setup minimal employees
	robert := &domain.Employee{
		Name:    "Robert",
		BirthDate: time.Date(1962,7,1,0,0,0,0,time.UTC),
		HireDate:  time.Date(1992,7,1,0,0,0,0,time.UTC),
		CurrentSalary: decimal.NewFromInt(150000),
		High3Salary:   decimal.NewFromInt(120000),
		SurvivorBenefitElectionPercent: decimal.NewFromFloat(0.5),
		SSBenefitFRA: decimal.NewFromInt(2500),
	}
	dawn := &domain.Employee{
		Name:    "Dawn",
		BirthDate: time.Date(1965,7,1,0,0,0,0,time.UTC),
		HireDate:  time.Date(1995,7,1,0,0,0,0,time.UTC),
		CurrentSalary: decimal.NewFromInt(100000),
		High3Salary:   decimal.NewFromInt(80000),
		SurvivorBenefitElectionPercent: decimal.NewFromFloat(0.5),
		SSBenefitFRA: decimal.NewFromInt(2000),
	}

	// Scenario: Robert dies in 2030 (index 5 if base 2025)
	deathYear := 2030
	scenario := &domain.Scenario{
		Name: "Survivor Pension Test",
		Robert: domain.RetirementScenario{EmployeeName: robert.Name, RetirementDate: time.Date(2027,7,1,0,0,0,0,time.UTC), SSStartAge: 67, TSPWithdrawalStrategy: "4_percent_rule"},
		Dawn:   domain.RetirementScenario{EmployeeName: dawn.Name, RetirementDate: time.Date(2027,7,1,0,0,0,0,time.UTC), SSStartAge: 67, TSPWithdrawalStrategy: "4_percent_rule"},
		Mortality: &domain.ScenarioMortality{Robert: &domain.MortalitySpec{DeathDate: &[]time.Time{time.Date(deathYear,1,1,0,0,0,0,time.UTC)}[0]}, Assumptions: &domain.MortalityAssumptions{FilingStatusSwitch: "next_year"}},
	}
	assumptions := &domain.GlobalAssumptions{ProjectionYears: 10, InflationRate: decimal.NewFromFloat(0.02), COLAGeneralRate: decimal.NewFromFloat(0.02)}
	federal := domain.FederalRules{}
	ce := NewCalculationEngine()
	projection := ce.GenerateAnnualProjection(robert, dawn, scenario, assumptions, federal)

	// Find death index
	deathIdx := deathYear - ProjectionBaseYear
	if deathIdx < 0 || deathIdx >= len(projection) { t.Fatalf("death index out of range") }

	// Pension calc at retirement to have baseline
	retDate := scenario.Robert.RetirementDate
	baseCalc := CalculateFERSPension(robert, retDate)
	if baseCalc.SurvivorAnnuity.IsZero() { t.Fatalf("expected survivor annuity > 0") }

	// Year before death should have no survivor pension and full reduced pension (if retired)
	preIdx := deathIdx - 1
	if preIdx >= 0 {
		cfPre := projection[preIdx]
		if !cfPre.RobertDeceased && !cfPre.SurvivorPensionDawn.IsZero() { t.Errorf("unexpected survivor pension before death year") }
	}
	// Death year onward should show survivor pension for Dawn and RobertDeceased true
	for y := deathIdx; y < len(projection); y++ {
		cf := projection[y]
		if y == deathIdx && !cf.RobertDeceased { t.Errorf("expected RobertDeceased true in death year") }
		if cf.RobertDeceased {
			if cf.SurvivorPensionDawn.IsZero() { t.Errorf("expected survivor pension for Dawn year %d", cf.Date.Year()) }
			// Survivor pension should approximate elected share of unreduced base with COLA (allow small tolerance)
			yearsSinceRet := y - (retDate.Year() - ProjectionBaseYear)
			if yearsSinceRet < 0 { yearsSinceRet = 0 }
			expected := baseCalc.SurvivorAnnuity
			curr := expected
			for cy := 1; cy <= yearsSinceRet; cy++ {
				projDate := retDate.AddDate(cy,0,0)
				ageAt := robert.Age(projDate)
				curr = ApplyFERSPensionCOLA(curr, assumptions.InflationRate, ageAt)
			}
			// Compare with tolerance 1 dollar
			diff := cf.SurvivorPensionDawn.Sub(curr).Abs()
			if diff.GreaterThan(decimal.NewFromInt(1)) { t.Errorf("survivor pension mismatch year %d got %s expected %s", cf.Date.Year(), cf.SurvivorPensionDawn, curr) }
		}
	}
}
