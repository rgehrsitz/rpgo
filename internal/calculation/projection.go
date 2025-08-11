package calculation

import (
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// GenerateAnnualProjectionGeneric produces a basic projection for the generic participant model.
// (Minimal implementation after legacy removal – TODO: expand with full pension/SS/TSP logic.)
func (ce *CalculationEngine) GenerateAnnualProjectionGeneric(household *domain.Household, scenario *domain.GenericScenario, assumptions *domain.GlobalAssumptions, federalRules domain.FederalRules) []domain.AnnualCashFlow {
	var participantNames []string
	var startYear int
	var years int
	// Restore variable initializations
	participantNames = make([]string, len(household.Participants))
	for i, p := range household.Participants {
		participantNames[i] = p.Name
	}
	startYear = ProjectionBaseYear
	years = assumptions.ProjectionYears
	// var years int (removed stray declaration)
		years = 1
	}

	// Map of participant name to scenario details
	psMap := map[string]domain.ParticipantScenario{}
	if scenario != nil {
		for name, ps := range scenario.ParticipantScenarios {
			psMap[name] = ps
		}
	}

	// Track dynamic participant state across years
	type pState struct {
		salary          decimal.Decimal
		retired         bool
		retirementYear  *int
		pensionAnnual   decimal.Decimal
		ssStarted       bool
		ssAnnual        decimal.Decimal
		tspBalance      decimal.Decimal
		tspInflationAdj decimal.Decimal // for 4% rule base amount
		fehbPremium     decimal.Decimal // current annual FEHB premium (if primary)
	}
	states := map[string]*pState{}

	cola := assumptions.COLAGeneralRate
	infl := assumptions.InflationRate
	preRetReturn := assumptions.TSPReturnPreRetirement
	postRetReturn := assumptions.TSPReturnPostRetirement
	fehbInfl := assumptions.FEHBPremiumInflation

	// Initialize state
	for i := range household.Participants {
		p := &household.Participants[i]
		st := &pState{}
		if p.CurrentSalary != nil {
			st.salary = *p.CurrentSalary
		}
		if p.TSPBalanceTraditional != nil && p.TSPBalanceRoth != nil {
			st.tspBalance = p.TSPBalanceTraditional.Add(*p.TSPBalanceRoth)
		}
		if p.IsPrimaryFEHBHolder && p.FEHBPremiumPerPayPeriod != nil {
			st.fehbPremium = p.FEHBPremiumPerPayPeriod.Mul(decimal.NewFromInt(26))
		}
		// Determine retirement year index (year starting after retirement date)
		if ps, ok := psMap[p.Name]; ok && ps.RetirementDate != nil {
			ry := ps.RetirementDate.Year() - startYear
			if ry < 0 {
				ry = 0
			}
			st.retirementYear = &ry
		}
		states[p.Name] = st
	}

	// Death years
	deathYear := map[string]*int{}
	if scenario != nil && scenario.Mortality != nil && scenario.Mortality.Participants != nil {
		for name, ms := range scenario.Mortality.Participants {
			if ms == nil {
				continue
			}
			if ms.DeathDate != nil {
				y := ms.DeathDate.Year() - startYear
				if y >= 0 && y < years {
					deathYear[name] = &y
				}
			}
			if ms.DeathAge != nil {
				y := (*ms.DeathAge) + getBirthYear(household, name) - startYear
				if y >= 0 && y < years {
					deathYear[name] = &y
				}
			}
		}
	}

	// Helper lambdas
	fersMultiplier := func(ageAtRet int, yearsSvc decimal.Decimal) decimal.Decimal {
		if yearsSvc.GreaterThanOrEqual(decimal.NewFromInt(20)) && ageAtRet >= 62 {
			return decimal.NewFromFloat(0.011)
		}
		return decimal.NewFromFloat(0.01)
	}
	computePension := func(p *domain.Participant, retDate time.Time, age int) decimal.Decimal {
		if !p.IsFederal || p.High3Salary == nil || p.HireDate == nil {
			return decimal.Zero
		}
		yearsSvc := p.YearsOfService(retDate)
		mult := fersMultiplier(age, yearsSvc)
		return p.High3Salary.Mul(yearsSvc).Mul(mult)
	}
	computeSSBase := func(p *domain.Participant, startAge int) decimal.Decimal {
		if startAge <= 62 {
			return p.SSBenefit62
		}
		if startAge >= 70 {
			return p.SSBenefit70
		}
		fraAge := 67
		if startAge <= fraAge {
			span := fraAge - 62
			offset := startAge - 62
			diff := p.SSBenefitFRA.Sub(p.SSBenefit62)
			return p.SSBenefit62.Add(diff.Mul(decimal.NewFromInt(int64(offset))).Div(decimal.NewFromInt(int64(span))))
		}
		span := 70 - 67
		offset := startAge - 67
		diff := p.SSBenefit70.Sub(p.SSBenefitFRA)
		return p.SSBenefitFRA.Add(diff.Mul(decimal.NewFromInt(int64(offset))).Div(decimal.NewFromInt(int64(span))))
	}
	years := assumptions.ProjectionYears
	if years <= 0 {
		years = 1
	}
	projection := make([]domain.AnnualCashFlow, years)

	for yr := 0; yr < years; yr++ {
		dt := time.Date(startYear+yr, 1, 1, 0, 0, 0, 0, time.UTC)
		cf := domain.NewAnnualCashFlow(yr, dt, participantNames)

		for i := range household.Participants {
			p := &household.Participants[i]
			st := states[p.Name]
			age := dt.Year() - p.BirthDate.Year(); if dt.YearDay() < p.BirthDate.YearDay() { age-- }
			cf.Ages[p.Name] = age

			// Death handling
			if di, ok := deathYear[p.Name]; ok && yr >= *di { cf.IsDeceased[p.Name] = true; continue }

			// Retirement transition
			if st.retirementYear != nil && yr >= *st.retirementYear { if !st.retired { // retire this year
					// compute pension
					retDate := time.Date(startYear+*st.retirementYear, 1, 1, 0,0,0,0,time.UTC)
							if y >= 0 && y < assumptions.ProjectionYears {
					st.retired = true
					st.tspInflationAdj = st.tspBalance // base for 4% rule
				}
			}

							if y >= 0 && y < assumptions.ProjectionYears {
			if !st.retired && st.salary.GreaterThan(decimal.Zero) {
				if yr > 0 { st.salary = st.salary.Mul(decimal.NewFromInt(1).Add(cola)) }
				cf.Salaries[p.Name] = st.salary
			} else {
				cf.Salaries[p.Name] = decimal.Zero
			}

			// FEHB premium (grow with FEHB inflation)
			if st.fehbPremium.GreaterThan(decimal.Zero) {
				if yr > 0 { st.fehbPremium = st.fehbPremium.Mul(decimal.NewFromInt(1).Add(fehbInfl)) }
			}

			// Pension (apply COLA after first year of receipt)
			if st.retired && st.pensionAnnual.GreaterThan(decimal.Zero) {
				if yr > *st.retirementYear { st.pensionAnnual = st.pensionAnnual.Mul(decimal.NewFromInt(1).Add(cola)) }
				cf.Pensions[p.Name] = st.pensionAnnual
			}

			// Social Security start
			if ps, ok := psMap[p.Name]; ok {
				startAge := ps.SSStartAge
				if !st.ssStarted && age >= startAge { st.ssAnnual = computeSSBase(p, startAge); st.ssStarted = true }
			}
			if st.ssStarted { if yr > 0 { st.ssAnnual = st.ssAnnual.Mul(decimal.NewFromInt(1).Add(cola)) }; cf.SSBenefits[p.Name] = st.ssAnnual }

			// TSP contributions (pre-retirement)
			contribution := decimal.Zero
			if !st.retired && p.IsFederal && p.CurrentSalary != nil && p.TSPContributionPercent != nil && st.salary.GreaterThan(decimal.Zero) {
				contribution = st.salary.Mul(*p.TSPContributionPercent)
				agencyMatch := p.AgencyMatch() // uses current salary pointer
				contribution = contribution.Add(agencyMatch)
				st.tspBalance = st.tspBalance.Add(contribution)
				cf.ParticipantTSPContributions[p.Name] = contribution
			}

			// TSP withdrawals (post-retirement)
			withdrawal := decimal.Zero
			if st.retired && st.tspBalance.GreaterThan(decimal.Zero) {
				if ps, ok := psMap[p.Name]; ok {
					switch ps.TSPWithdrawalStrategy {
				var years int
				years = assumptions.ProjectionYears
				if years <= 0 {
					years = 1
				}
					case "4_percent_rule":
						// Base first-year withdrawal 4% of balance at retirement; inflation adjust
						base := st.tspInflationAdj.Mul(decimal.NewFromFloat(0.04))
						if yr > *st.retirementYear { base = base.Mul(decimal.NewFromInt(1).Add(infl).Pow(decimal.NewFromInt(int64(yr-*st.retirementYear)))) }
						withdrawal = base
					case "need_based":
						if ps.TSPWithdrawalTargetMonthly != nil { withdrawal = ps.TSPWithdrawalTargetMonthly.Mul(decimal.NewFromInt(12)) }
					case "variable_percentage":
						if ps.TSPWithdrawalRate != nil { withdrawal = st.tspBalance.Mul(*ps.TSPWithdrawalRate) }
					}
				}
				if withdrawal.GreaterThan(st.tspBalance) { withdrawal = st.tspBalance }
				st.tspBalance = st.tspBalance.Sub(withdrawal)
				cf.TSPWithdrawals[p.Name] = withdrawal
			}

			// TSP growth
			if !st.retired {
				if !st.tspBalance.IsZero() { st.tspBalance = st.tspBalance.Mul(decimal.NewFromInt(1).Add(preRetReturn)) }
			} else {
				if !st.tspBalance.IsZero() { st.tspBalance = st.tspBalance.Mul(decimal.NewFromInt(1).Add(postRetReturn)) }
			}
			cf.TSPBalances[p.Name] = st.tspBalance
		}

		// Household-level deductions (FEHB, TSP contributions)
		fehbTotal := decimal.Zero
		tspContribTotal := decimal.Zero
		for _, name := range participantNames {
			st := states[name]
			if st.fehbPremium.GreaterThan(decimal.Zero) { fehbTotal = fehbTotal.Add(st.fehbPremium) }
			tspContribTotal = tspContribTotal.Add(cf.ParticipantTSPContributions[name])
		}
		cf.FEHBPremium = fehbTotal
		cf.TotalTSPContributions = tspContribTotal

		// TAX CALCULATION
		// Aggregate household taxable income
		taxable := domain.TaxableIncome{
			Salary:             cf.GetTotalSalary(),
			FERSPension:        cf.GetTotalPension(),
			TSPWithdrawalsTrad: cf.GetTotalTSPWithdrawal(),
			TaxableSSBenefits:  cf.GetTotalSSBenefit(),
			OtherTaxableIncome: decimal.Zero,
			WageIncome:         cf.GetTotalSalary(),
			InterestIncome:     decimal.Zero,
		}
		// Determine filing status and seniors
		filingStatus := household.FilingStatus
		seniors := 0
		for _, age := range cf.Ages {
			if age >= 65 {
				seniors++
			}
		}
		// FICA only on wage income if not retired
		isRetired := true
		for _, st := range states {
			if !st.retired {
				isRetired = false
				break
			}
		}
		// Use engine's tax calculator
		federalTax := decimal.Zero
		stateTax := decimal.Zero
		localTax := decimal.Zero
		ficaTax := decimal.Zero
		if ce != nil && ce.TaxCalc != nil {
			federalTax = ce.TaxCalc.calculateFederalTaxWithStatus(taxable, filingStatus, seniors)
			stateTax = ce.TaxCalc.StateTaxCalc.CalculateTax(taxable, isRetired)
			localTax = ce.TaxCalc.LocalTaxCalc.CalculateEIT(taxable.WageIncome, isRetired)
			if !isRetired {
				ficaTax = ce.TaxCalc.FICATaxCalc.CalculateFICA(taxable.WageIncome, taxable.WageIncome)
			}
		}
		cf.FederalTax = federalTax
		cf.StateTax = stateTax
		cf.LocalTax = localTax
		cf.FICATax = ficaTax

		// Compute gross and net
		cf.TotalGrossIncome = cf.CalculateTotalIncome()
		cf.CalculateNetIncome()
		projection[yr] = *cf
	}
	return projection
}

func getBirthYear(h *domain.Household, name string) int {
	for _, p := range h.Participants {
		if p.Name == name {
			return p.BirthDate.Year()
		}
	}
	return ProjectionBaseYear
}

// Legacy two-person GenerateAnnualProjection removed; use GenerateAnnualProjectionGeneric.

// Helper function to get participant name for a given age (simplified)
func getParticipantNameForAge(ages map[string]int, targetAge int) string {
	for name, age := range ages {
		if age == targetAge {
			return name
		}
	}
	return ""
}
