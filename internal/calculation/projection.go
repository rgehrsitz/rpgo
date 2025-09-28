package calculation

import (
	"time"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

var (
	decimalOne    = decimal.NewFromInt(1)
	decimalZero   = decimal.Zero
	decimalTwelve = decimal.NewFromInt(12)
)

// GenerateAnnualProjectionGeneric produces a projection for the generic participant model.
func (ce *CalculationEngine) GenerateAnnualProjectionGeneric(household *domain.Household, scenario *domain.GenericScenario, assumptions *domain.GlobalAssumptions, federalRules domain.FederalRules) []domain.AnnualCashFlow {
	if household == nil || assumptions == nil || len(household.Participants) == 0 {
		return nil
	}

	participantNames := make([]string, len(household.Participants))
	for i, p := range household.Participants {
		participantNames[i] = p.Name
	}

	startYear := ProjectionBaseYear
	years := assumptions.ProjectionYears
	if years <= 0 {
		years = 1
	}

	psMap := map[string]domain.ParticipantScenario{}
	if scenario != nil {
		for name, ps := range scenario.ParticipantScenarios {
			psMap[name] = ps
		}
	}

	tspTransferMode := ""
	survivorSpendingFactor := decimalOne
	if scenario != nil && scenario.Mortality != nil && scenario.Mortality.Assumptions != nil {
		tspTransferMode = scenario.Mortality.Assumptions.TSPSpousalTransfer
		if !scenario.Mortality.Assumptions.SurvivorSpendingFactor.IsZero() {
			sf := scenario.Mortality.Assumptions.SurvivorSpendingFactor
			if sf.LessThan(decimalZero) {
				sf = decimalZero
			}
			if sf.GreaterThan(decimalOne) {
				sf = decimalOne
			}
			survivorSpendingFactor = sf
		}
	}

	totalEmployerPercent := decimal.NewFromFloat(0.05)
	if federalRules.FERSRules.TSPMatchingRate.GreaterThan(decimalZero) {
		totalEmployerPercent = federalRules.FERSRules.TSPMatchingRate
	}
	baseMatchThreshold := decimal.NewFromFloat(0.05)
	if federalRules.FERSRules.TSPMatchingThreshold.GreaterThan(decimalZero) {
		baseMatchThreshold = federalRules.FERSRules.TSPMatchingThreshold
	}

	autoContributionPercent := decimal.NewFromFloat(0.01)
	if totalEmployerPercent.LessThanOrEqual(decimalZero) {
		autoContributionPercent = decimalZero
	} else if totalEmployerPercent.LessThan(autoContributionPercent) {
		autoContributionPercent = totalEmployerPercent
	}

	matchPoolPercent := totalEmployerPercent.Sub(autoContributionPercent)
	if matchPoolPercent.LessThan(decimalZero) {
		matchPoolPercent = decimalZero
	}

	if baseMatchThreshold.LessThanOrEqual(decimalZero) {
		baseMatchThreshold = decimal.NewFromFloat(0.05)
	}

	firstTierRatio := decimal.NewFromFloat(0.6) // 60% of threshold (first 3% when threshold is 5%)
	secondTierRatio := decimalOne.Sub(firstTierRatio)
	firstTierCapPercent := baseMatchThreshold.Mul(firstTierRatio)
	secondTierCapPercent := baseMatchThreshold.Mul(secondTierRatio)
	if secondTierCapPercent.LessThan(decimalZero) {
		secondTierCapPercent = decimalZero
	}
	secondTierMatchRate := decimal.NewFromFloat(0.5)

	type participantState struct {
		currentSalary              decimal.Decimal
		retired                    bool
		retirementYear             *int
		retirementDate             *time.Time
		pensionAnnual              decimal.Decimal
		pensionStartYear           *int
		survivorPension            decimal.Decimal
		survivorPensionIncome      decimal.Decimal
		survivorPensionLastUpdated int
		survivorPensionDistributed bool
		ssStartAge                 int
		ssStarted                  bool
		ssAnnual                   decimal.Decimal
		ssStartYear                *int
		tspBalance                 decimal.Decimal
		tspWithdrawalBase          decimal.Decimal
		fehbPremium                decimal.Decimal
	}

	states := make(map[string]*participantState, len(household.Participants))
	for i := range household.Participants {
		p := &household.Participants[i]
		st := &participantState{currentSalary: decimalZero, ssStartAge: 67, survivorPensionLastUpdated: -1}

		if p.CurrentSalary != nil {
			st.currentSalary = *p.CurrentSalary
		}

		if ps, ok := psMap[p.Name]; ok {
			if ps.RetirementDate != nil {
				ry := ps.RetirementDate.Year() - startYear
				if ry < 0 {
					ry = 0
				}
				st.retirementYear = new(int)
				*st.retirementYear = ry
				st.retirementDate = ps.RetirementDate
			}
			if ps.SSStartAge >= 62 && ps.SSStartAge <= 70 {
				st.ssStartAge = ps.SSStartAge
			}
		}

		if st.retirementYear == nil && p.EmploymentEndDate != nil {
			ry := p.EmploymentEndDate.Year() - startYear
			if ry >= 0 {
				st.retirementYear = new(int)
				*st.retirementYear = ry
				st.retirementDate = p.EmploymentEndDate
			}
		}

		st.tspBalance = decimalZero
		if p.TSPBalanceTraditional != nil {
			st.tspBalance = st.tspBalance.Add(*p.TSPBalanceTraditional)
		}
		if p.TSPBalanceRoth != nil {
			st.tspBalance = st.tspBalance.Add(*p.TSPBalanceRoth)
		}

		if p.IsPrimaryFEHBHolder && p.FEHBPremiumPerPayPeriod != nil {
			st.fehbPremium = p.FEHBPremiumPerPayPeriod.Mul(decimal.NewFromInt(26))
		}

		states[p.Name] = st
	}

	deathYears := map[string]*int{}
	if scenario != nil && scenario.Mortality != nil && scenario.Mortality.Participants != nil {
		for name, spec := range scenario.Mortality.Participants {
			if spec == nil {
				continue
			}
			if spec.DeathDate != nil {
				idx := spec.DeathDate.Year() - startYear
				if idx >= 0 && idx < years {
					v := new(int)
					*v = idx
					deathYears[name] = v
				}
			}
			if spec.DeathAge != nil {
				birthYear := getBirthYear(household, name)
				idx := *spec.DeathAge + birthYear - startYear
				if idx >= 0 && idx < years {
					v := new(int)
					*v = idx
					deathYears[name] = v
				}
			}
		}
	}

	cola := assumptions.COLAGeneralRate
	infl := assumptions.InflationRate
	fehbInfl := assumptions.FEHBPremiumInflation
	preRetReturn := assumptions.TSPReturnPreRetirement
	postRetReturn := assumptions.TSPReturnPostRetirement

	projection := make([]domain.AnnualCashFlow, years)

	for yr := 0; yr < years; yr++ {
		yearDate := time.Date(startYear+yr, 1, 1, 0, 0, 0, 0, time.UTC)
		cf := domain.NewAnnualCashFlow(yr, yearDate, participantNames)
		transferPool := decimalZero
		aliveNames := aliveParticipantsForYear(household, deathYears, yr)
		singleSurvivorName := ""
		if len(aliveNames) == 1 {
			singleSurvivorName = aliveNames[0]
		}

		for i := range household.Participants {
			p := &household.Participants[i]
			st := states[p.Name]

			if st.fehbPremium.GreaterThan(decimalZero) && yr > 0 {
				st.fehbPremium = st.fehbPremium.Mul(onePlus(fehbInfl))
			}

			age := p.Age(yearDate)
			cf.Ages[p.Name] = age
			if st.survivorPensionIncome.GreaterThan(decimalZero) {
				if st.survivorPensionLastUpdated < yr {
					st.survivorPensionIncome = st.survivorPensionIncome.Mul(onePlus(cola))
					st.survivorPensionLastUpdated = yr
				}
				cf.SurvivorPensions[p.Name] = st.survivorPensionIncome
			}

			var deathIdx *int
			if dy, ok := deathYears[p.Name]; ok {
				deathIdx = dy
			}

			isDeceased := deathIdx != nil && yr >= *deathIdx
			cf.IsDeceased[p.Name] = isDeceased

			if isDeceased {
				if !st.survivorPensionDistributed && st.survivorPension.GreaterThan(decimalZero) {
					if len(aliveNames) > 0 {
						share := st.survivorPension
						if len(aliveNames) > 1 {
							share = share.Div(decimal.NewFromInt(int64(len(aliveNames))))
						}
						for _, name := range aliveNames {
							recipientState := states[name]
							recipientState.survivorPensionIncome = recipientState.survivorPensionIncome.Add(share)
							recipientState.survivorPensionLastUpdated = yr
							cf.SurvivorPensions[name] = cf.SurvivorPensions[name].Add(share)
						}
					}
					st.survivorPensionDistributed = true
					st.survivorPension = decimalZero
				}
				if tspTransferMode == "merge" && deathIdx != nil && yr == *deathIdx && st.tspBalance.GreaterThan(decimalZero) {
					transferPool = transferPool.Add(st.tspBalance)
				}
				st.tspBalance = decimalZero
				cf.Salaries[p.Name] = decimalZero
				cf.Pensions[p.Name] = decimalZero
				cf.SSBenefits[p.Name] = decimalZero
				cf.TSPBalances[p.Name] = decimalZero
				continue
			}

			if st.retirementYear == nil || yr <= *st.retirementYear {
				if yr > 0 && st.currentSalary.GreaterThan(decimalZero) {
					st.currentSalary = st.currentSalary.Mul(onePlus(cola))
				}
			}

			salaryForYear := decimalZero
			if st.currentSalary.GreaterThan(decimalZero) {
				switch {
				case st.retirementYear == nil || yr < *st.retirementYear:
					salaryForYear = st.currentSalary
				case st.retirementYear != nil && yr == *st.retirementYear:
					fraction := computeWorkFraction(st.retirementDate, yearDate)
					salaryForYear = st.currentSalary.Mul(fraction)
				}
			}
			cf.Salaries[p.Name] = salaryForYear

			if st.retirementYear != nil && yr >= *st.retirementYear && !st.retired {
				st.retired = true
				if st.retirementDate == nil {
					rd := time.Date(startYear+*st.retirementYear, 1, 1, 0, 0, 0, 0, time.UTC)
					st.retirementDate = &rd
				}
				st.tspWithdrawalBase = st.tspBalance
				startYr := new(int)
				*startYr = yr
				st.pensionStartYear = startYr

				if p.IsFederal && p.High3Salary != nil && p.HireDate != nil {
					pension, survivor := calculateParticipantPension(p, *st.retirementDate)
					st.pensionAnnual = pension
					st.survivorPension = survivor
				} else if p.ExternalPension != nil {
					st.pensionAnnual = p.ExternalPension.MonthlyBenefit.Mul(decimalTwelve)
				}
			}

			employeeContributionAmount := decimalZero
			if salaryForYear.GreaterThan(decimalZero) && p.IsFederal {
				if autoContributionPercent.GreaterThan(decimalZero) {
					autoContribution := salaryForYear.Mul(autoContributionPercent)
					if autoContribution.GreaterThan(decimalZero) {
						st.tspBalance = st.tspBalance.Add(autoContribution)
					}
				}

				employeePct := decimalZero
				if p.TSPContributionPercent != nil {
					employeePct = *p.TSPContributionPercent
					if employeePct.LessThan(decimalZero) {
						employeePct = decimalZero
					}
				}

				if employeePct.GreaterThan(decimalZero) {
					employeeContribution := salaryForYear.Mul(employeePct)
					if employeeContribution.GreaterThan(decimalZero) {
						st.tspBalance = st.tspBalance.Add(employeeContribution)
						employeeContributionAmount = employeeContributionAmount.Add(employeeContribution)
					}

					matchContribution := decimalZero
					if matchPoolPercent.GreaterThan(decimalZero) {
						firstTierEmployee := employeePct
						if firstTierEmployee.GreaterThan(firstTierCapPercent) {
							firstTierEmployee = firstTierCapPercent
						}
						if firstTierEmployee.LessThan(decimalZero) {
							firstTierEmployee = decimalZero
						}

						firstTierMatchPercent := firstTierEmployee
						if firstTierMatchPercent.GreaterThan(matchPoolPercent) {
							firstTierMatchPercent = matchPoolPercent
						}

						remainingMatchPercent := matchPoolPercent.Sub(firstTierMatchPercent)
						if remainingMatchPercent.LessThan(decimalZero) {
							remainingMatchPercent = decimalZero
						}

						secondTierEmployee := decimalZero
						if employeePct.GreaterThan(firstTierCapPercent) {
							secondTierEmployee = employeePct.Sub(firstTierCapPercent)
							if secondTierEmployee.GreaterThan(secondTierCapPercent) {
								secondTierEmployee = secondTierCapPercent
							}
						}

						secondTierMatchPercent := secondTierEmployee.Mul(secondTierMatchRate)
						if secondTierMatchPercent.GreaterThan(remainingMatchPercent) {
							secondTierMatchPercent = remainingMatchPercent
						}

						matchPercentTotal := firstTierMatchPercent.Add(secondTierMatchPercent)
						if matchPercentTotal.GreaterThan(decimalZero) {
							matchContribution = salaryForYear.Mul(matchPercentTotal)
						}
					}

					if matchContribution.GreaterThan(decimalZero) {
						st.tspBalance = st.tspBalance.Add(matchContribution)
					}
				}
			}

			if employeeContributionAmount.GreaterThan(decimalZero) {
				cf.ParticipantTSPContributions[p.Name] = cf.ParticipantTSPContributions[p.Name].Add(employeeContributionAmount)
			}

			if st.retired && st.pensionAnnual.GreaterThan(decimalZero) {
				pensionValue := st.pensionAnnual
				if st.pensionStartYear != nil && yr > *st.pensionStartYear {
					if p.IsFederal {
						pensionValue = applyParticipantFERSCOLA(pensionValue, cola, age)
						st.pensionAnnual = pensionValue
						if st.survivorPension.GreaterThan(decimalZero) {
							st.survivorPension = applyParticipantFERSCOLA(st.survivorPension, cola, age)
						}
					} else if p.ExternalPension != nil {
						pensionValue = pensionValue.Mul(onePlus(p.ExternalPension.COLAAdjustment))
						st.pensionAnnual = pensionValue
						if st.survivorPension.GreaterThan(decimalZero) {
							st.survivorPension = st.survivorPension.Mul(onePlus(p.ExternalPension.COLAAdjustment))
						}
					} else {
						pensionValue = pensionValue.Mul(onePlus(cola))
						st.pensionAnnual = pensionValue
						if st.survivorPension.GreaterThan(decimalZero) {
							st.survivorPension = st.survivorPension.Mul(onePlus(cola))
						}
					}
				}

				if st.pensionStartYear != nil && yr == *st.pensionStartYear {
					fractionWorked := computeWorkFraction(st.retirementDate, yearDate)
					pensionValue = st.pensionAnnual.Mul(decimalOne.Sub(fractionWorked))
				}

				cf.Pensions[p.Name] = pensionValue
			}

			if !st.ssStarted && age >= st.ssStartAge {
				st.ssAnnual = computeSSAnnualBenefit(p, st.ssStartAge)
				st.ssStarted = true
				ssStart := new(int)
				*ssStart = yr
				st.ssStartYear = ssStart
			}
			if st.ssStarted {
				if st.ssStartYear != nil && yr > *st.ssStartYear {
					st.ssAnnual = st.ssAnnual.Mul(onePlus(cola))
				}
				cf.SSBenefits[p.Name] = st.ssAnnual
			}

			withdrawal := decimalZero
			if st.retired && st.tspBalance.GreaterThan(decimalZero) {
				if ps, ok := psMap[p.Name]; ok {
					switch ps.TSPWithdrawalStrategy {
					case "4_percent_rule":
						if st.tspWithdrawalBase.IsZero() {
							st.tspWithdrawalBase = st.tspBalance
						} else if st.retirementYear != nil && yr > *st.retirementYear {
							st.tspWithdrawalBase = st.tspWithdrawalBase.Mul(onePlus(infl))
						}
						withdrawal = st.tspWithdrawalBase.Mul(decimal.NewFromFloat(0.04))
					case "need_based":
						if ps.TSPWithdrawalTargetMonthly != nil {
							withdrawal = ps.TSPWithdrawalTargetMonthly.Mul(decimalTwelve)
						}
					case "variable_percentage":
						if ps.TSPWithdrawalRate != nil {
							withdrawal = st.tspBalance.Mul(*ps.TSPWithdrawalRate)
						}
					}
				}
				if singleSurvivorName != "" && p.Name == singleSurvivorName {
					if survivorSpendingFactor.LessThan(decimalOne) {
						withdrawal = withdrawal.Mul(survivorSpendingFactor)
					}
				}
				if withdrawal.GreaterThan(st.tspBalance) {
					withdrawal = st.tspBalance
				}
				st.tspBalance = st.tspBalance.Sub(withdrawal)
				cf.TSPWithdrawals[p.Name] = withdrawal
			}

			growthRate := preRetReturn
			if st.retired {
				growthRate = postRetReturn
			}
			if !st.tspBalance.IsZero() {
				st.tspBalance = st.tspBalance.Mul(onePlus(growthRate))
			}
			cf.TSPBalances[p.Name] = st.tspBalance
		}

		livingNames := make([]string, 0, len(participantNames))
		for _, name := range participantNames {
			if !cf.IsDeceased[name] {
				livingNames = append(livingNames, name)
			}
		}
		if transferPool.GreaterThan(decimalZero) && len(livingNames) > 0 {
			share := transferPool.Div(decimal.NewFromInt(int64(len(livingNames))))
			for _, name := range livingNames {
				st := states[name]
				st.tspBalance = st.tspBalance.Add(share)
				cf.TSPBalances[name] = st.tspBalance
			}
		}

		fehbTotal := decimalZero
		tspContributionTotal := decimalZero
		for _, name := range participantNames {
			st := states[name]
			if !cf.IsDeceased[name] && st.fehbPremium.GreaterThan(decimalZero) {
				fehbTotal = fehbTotal.Add(st.fehbPremium)
			}
			tspContributionTotal = tspContributionTotal.Add(cf.ParticipantTSPContributions[name])
		}
		cf.FEHBPremium = fehbTotal
		cf.TotalTSPContributions = tspContributionTotal

		livingCount := len(livingNames)
		filingStatus := household.FilingStatus
		if filingStatus == "" {
			filingStatus = "single"
		}
		if filingStatus != "single" && livingCount <= 1 {
			cf.FilingStatusSingle = true
			filingStatus = "single"
		} else {
			cf.FilingStatusSingle = filingStatus == "single"
		}
		cf.FederalFilingStatus = filingStatus

		seniors := 0
		for name, age := range cf.Ages {
			if cf.IsDeceased[name] {
				continue
			}
			if age >= 65 {
				seniors++
			}
		}

		taxable := domain.TaxableIncome{
			Salary:             cf.GetTotalSalary(),
			FERSPension:        cf.GetTotalPension(),
			TSPWithdrawalsTrad: cf.GetTotalTSPWithdrawal(),
			TaxableSSBenefits:  cf.GetTotalSSBenefit(),
			OtherTaxableIncome: decimalZero,
			WageIncome:         cf.GetTotalSalary(),
			InterestIncome:     decimalZero,
		}

		isRetiredHousehold := true
		for _, name := range participantNames {
			st := states[name]
			if !cf.IsDeceased[name] && !st.retired {
				isRetiredHousehold = false
				break
			}
		}

		if ce != nil && ce.TaxCalc != nil {
			cf.FederalTax = ce.TaxCalc.calculateFederalTaxWithStatus(taxable, filingStatus, seniors)
			cf.StateTax = ce.TaxCalc.StateTaxCalc.CalculateTax(taxable, isRetiredHousehold)
			cf.LocalTax = ce.TaxCalc.LocalTaxCalc.CalculateEIT(taxable.WageIncome, isRetiredHousehold)
			if !isRetiredHousehold {
				cf.FICATax = ce.TaxCalc.FICATaxCalc.CalculateFICA(taxable.WageIncome, taxable.WageIncome)
			}
		}

		cf.TotalGrossIncome = cf.CalculateTotalIncome()
		cf.CalculateNetIncome()

		projection[yr] = *cf
	}

	return projection
}

func computeWorkFraction(retirementDate *time.Time, yearStart time.Time) decimal.Decimal {
	if retirementDate == nil {
		return decimal.NewFromFloat(0.5)
	}
	if retirementDate.Year() < yearStart.Year() {
		return decimalZero
	}
	if retirementDate.Year() > yearStart.Year() {
		return decimalOne
	}
	if retirementDate.Before(yearStart) {
		return decimalZero
	}

	daysWorked := retirementDate.Sub(yearStart).Hours() / 24
	daysInYear := 365.0
	if isLeapYear(yearStart.Year()) {
		daysInYear = 366.0
	}

	fraction := daysWorked / daysInYear
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}

	return decimal.NewFromFloat(fraction)
}

func computeSSAnnualBenefit(p *domain.Participant, startAge int) decimal.Decimal {
	return computeSSMonthlyBenefit(p, startAge).Mul(decimalTwelve)
}

func computeSSMonthlyBenefit(p *domain.Participant, startAge int) decimal.Decimal {
	if startAge <= 62 {
		return p.SSBenefit62
	}
	if startAge >= 70 {
		return p.SSBenefit70
	}

	fraAge := 67
	if startAge <= fraAge {
		span := fraAge - 62
		if span <= 0 {
			return p.SSBenefitFRA
		}
		offset := startAge - 62
		diff := p.SSBenefitFRA.Sub(p.SSBenefit62)
		return p.SSBenefit62.Add(diff.Mul(decimal.NewFromInt(int64(offset))).Div(decimal.NewFromInt(int64(span))))
	}

	span := 70 - fraAge
	if span <= 0 {
		return p.SSBenefitFRA
	}
	offset := startAge - fraAge
	diff := p.SSBenefit70.Sub(p.SSBenefitFRA)
	return p.SSBenefitFRA.Add(diff.Mul(decimal.NewFromInt(int64(offset))).Div(decimal.NewFromInt(int64(span))))
}

func calculateParticipantPension(p *domain.Participant, retirementDate time.Time) (decimal.Decimal, decimal.Decimal) {
	if !p.IsFederal || p.High3Salary == nil || p.HireDate == nil {
		return decimalZero, decimalZero
	}

	serviceYears := p.YearsOfService(retirementDate)
	retirementAge := p.Age(retirementDate)

	multiplier := decimal.NewFromFloat(0.01)
	if retirementAge >= 62 && serviceYears.GreaterThanOrEqual(decimal.NewFromInt(20)) {
		multiplier = decimal.NewFromFloat(0.011)
	}

	pensionBase := p.High3Salary.Mul(serviceYears).Mul(multiplier)

	survivorElection := decimalZero
	if p.SurvivorBenefitElectionPercent != nil {
		survivorElection = *p.SurvivorBenefitElectionPercent
	}

	reduced := pensionBase
	survivor := decimalZero
	if survivorElection.GreaterThan(decimalZero) {
		half := decimal.NewFromFloat(0.5)
		quarter := decimal.NewFromFloat(0.25)
		if survivorElection.GreaterThan(decimal.NewFromFloat(0.4)) {
			survivorElection = half
		} else if survivorElection.GreaterThan(decimal.NewFromFloat(0.20)) && survivorElection.LessThan(decimal.NewFromFloat(0.30)) {
			survivorElection = quarter
		}

		if survivorElection.Equals(half) {
			reduced = pensionBase.Mul(decimal.NewFromFloat(0.90))
			survivor = pensionBase.Mul(half)
		} else if survivorElection.Equals(quarter) {
			reduced = pensionBase.Mul(decimal.NewFromFloat(0.95))
			survivor = pensionBase.Mul(quarter)
		}
	}

	return reduced, survivor
}

func applyParticipantFERSCOLA(currentPension decimal.Decimal, inflationRate decimal.Decimal, annuitantAge int) decimal.Decimal {
	if annuitantAge < 62 {
		return currentPension
	}

	if inflationRate.LessThanOrEqual(decimal.NewFromFloat(0.02)) {
		return currentPension.Mul(onePlus(inflationRate))
	}
	if inflationRate.LessThanOrEqual(decimal.NewFromFloat(0.03)) {
		return currentPension.Mul(onePlus(decimal.NewFromFloat(0.02)))
	}

	return currentPension.Mul(onePlus(inflationRate.Sub(decimal.NewFromFloat(0.01))))
}

func onePlus(value decimal.Decimal) decimal.Decimal {
	return decimalOne.Add(value)
}

func isLeapYear(year int) bool {
	if year%400 == 0 {
		return true
	}
	if year%100 == 0 {
		return false
	}
	return year%4 == 0
}

func getBirthYear(h *domain.Household, name string) int {
	for _, p := range h.Participants {
		if p.Name == name {
			return p.BirthDate.Year()
		}
	}
	return ProjectionBaseYear
}

func aliveParticipantsForYear(h *domain.Household, deathYears map[string]*int, year int) []string {
	names := make([]string, 0, len(h.Participants))
	for _, p := range h.Participants {
		if dy, ok := deathYears[p.Name]; ok && dy != nil {
			if year >= *dy {
				continue
			}
		}
		names = append(names, p.Name)
	}
	return names
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
