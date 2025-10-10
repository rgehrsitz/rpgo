package calculation

import (
	"sort"
	"time"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/rgehrsitz/rpgo/internal/sequencing"
	"github.com/shopspring/decimal"
)

var (
	decimalOne    = decimal.NewFromInt(1)
	decimalZero   = decimal.Zero
	decimalTwelve = decimal.NewFromInt(12)
)

// tspContribForYear calculates TSP employee contributions based on policy
func tspContribForYear(wagesYear decimal.Decimal, contribPct decimal.Decimal, payPeriods int, retireDate *time.Time, yr int, startYear int, policy string) decimal.Decimal {
	if wagesYear.LessThanOrEqual(decimalZero) || contribPct.LessThanOrEqual(decimalZero) {
		return decimalZero
	}

	currentYear := startYear + yr
	if retireDate == nil {
		// Still working full year
		return wagesYear.Mul(contribPct)
	}

	switch policy {
	case "zero_in_retirement_view":
		// Once scenario is "retirement", contributions cease (comparison-mode)
		if retireDate.Year() <= currentYear {
			return decimalZero
		}
		return wagesYear.Mul(contribPct) // working full year
	default: // "continue_until_retirement"
		if retireDate.Year() < currentYear {
			return decimalZero // retired before this year
		}
		if retireDate.Year() > currentYear {
			return wagesYear.Mul(contribPct) // full year
		}
		// retiring this year: simple proration by pay periods through the last period before retirement
		periodsWorked := int(float64(payPeriods) * float64(retireDate.YearDay()) / 365.0)
		if periodsWorked < 0 {
			periodsWorked = 0
		}
		if periodsWorked > payPeriods {
			periodsWorked = payPeriods
		}
		return (wagesYear.Mul(contribPct)).Mul(decimal.NewFromInt(int64(periodsWorked))).Div(decimal.NewFromInt(int64(payPeriods)))
	}
}

// SSMonthsPaidInYear returns the number of benefit payments in `year` if claiming at `claimAgeYears`
// Rule: first payment is the month AFTER the claim month (SSA timing)
func SSMonthsPaidInYear(dob time.Time, claimAgeYears int, year int) int {
	claimMonth := time.Date(dob.Year()+claimAgeYears, dob.Month(), 1, 0, 0, 0, 0, time.UTC)
	firstPaid := claimMonth.AddDate(0, 1, 0)
	if firstPaid.Year() > year {
		return 0
	}
	if firstPaid.Year() < year {
		return 12
	}
	return 12 - int(firstPaid.Month()) + 1
}

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
		// Sort participant names for deterministic processing order
		var participantNames []string
		for name := range scenario.ParticipantScenarios {
			participantNames = append(participantNames, name)
		}
		sort.Strings(participantNames)

		for _, name := range participantNames {
			psMap[name] = scenario.ParticipantScenarios[name]
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
		ssAnnualFull               decimal.Decimal
		ssStartYear                *int
		tspBalance                 decimal.Decimal // total (legacy)
		tspBalanceTraditional      decimal.Decimal // new split tracking
		tspBalanceRoth             decimal.Decimal // new split tracking
		taxableBalance             decimal.Decimal // taxable brokerage aggregate per participant
		taxableBasis               decimal.Decimal // cost basis
		tspWithdrawalBase          decimal.Decimal
		fehbPremium                decimal.Decimal
		fersSupplementAnnual       decimal.Decimal
		fersSupplementStartYear    *int
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
		st.tspBalanceTraditional = decimalZero
		st.tspBalanceRoth = decimalZero
		st.taxableBalance = decimalZero
		st.taxableBasis = decimalZero
		if p.TSPBalanceTraditional != nil {
			st.tspBalanceTraditional = st.tspBalanceTraditional.Add(*p.TSPBalanceTraditional)
		}
		if p.TSPBalanceRoth != nil {
			st.tspBalanceRoth = st.tspBalanceRoth.Add(*p.TSPBalanceRoth)
		}
		if p.TaxableAccountBalance != nil {
			st.taxableBalance = st.taxableBalance.Add(*p.TaxableAccountBalance)
		}
		if p.TaxableAccountBasis != nil {
			st.taxableBasis = st.taxableBasis.Add(*p.TaxableAccountBasis)
		}
		st.tspBalance = st.tspBalanceTraditional.Add(st.tspBalanceRoth)

		if p.IsPrimaryFEHBHolder && p.FEHBPremiumPerPayPeriod != nil {
			st.fehbPremium = p.FEHBPremiumPerPayPeriod.Mul(decimal.NewFromInt(26))
		}

		states[p.Name] = st
	}

	deathYears := map[string]*int{}
	if scenario != nil && scenario.Mortality != nil && scenario.Mortality.Participants != nil {
		// Sort participant names for deterministic processing order
		var mortalityNames []string
		for name := range scenario.Mortality.Participants {
			mortalityNames = append(mortalityNames, name)
		}
		sort.Strings(mortalityNames)

		for _, name := range mortalityNames {
			spec := scenario.Mortality.Participants[name]
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
		yearEnd := time.Date(startYear+yr, 12, 31, 23, 59, 59, 0, time.UTC)
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
			ageEnd := p.Age(yearEnd)
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

			workFraction := decimalZero
			salaryForYear := decimalZero
			if st.currentSalary.GreaterThan(decimalZero) {
				switch {
				case st.retirementYear == nil || yr < *st.retirementYear:
					workFraction = decimalOne
					salaryForYear = st.currentSalary
				case st.retirementYear != nil && yr == *st.retirementYear:
					fraction := computeWorkFraction(st.retirementDate, yearDate)
					workFraction = fraction
					salaryForYear = st.currentSalary.Mul(workFraction)
				default:
					workFraction = decimalZero
				}
			}
			cf.Salaries[p.Name] = salaryForYear

			// Calculate part-time work impact
			var participantScenario domain.ParticipantScenario
			if scenario != nil && scenario.ParticipantScenarios != nil {
				if ps, exists := scenario.ParticipantScenarios[p.Name]; exists {
					participantScenario = ps
				}
			}

			partTimeCalc := NewPartTimeWorkCalculator()
			partTimeAnalysis, err := partTimeCalc.CalculatePartTimeWorkForYear(
				*p,
				participantScenario,
				startYear+yr,
				cf.Ages[p.Name],
			)
			if err != nil {
				// Log error and continue with default values
				partTimeAnalysis = &domain.PartTimeWorkAnalysis{
					Year:       startYear + yr,
					IsPartTime: false,
				}
			}

			// Update cash flow with part-time work data
			cf.IsPartTime[p.Name] = partTimeAnalysis.IsPartTime
			if partTimeAnalysis.IsPartTime {
				cf.PartTimeSalary[p.Name] = partTimeAnalysis.AnnualSalary
				cf.PartTimeTSPContributions[p.Name] = partTimeAnalysis.TSPContributions
				cf.FERSSupplementReduction[p.Name] = partTimeAnalysis.FERSSupplementReduction

				// Override salary with part-time salary if working part-time
				cf.Salaries[p.Name] = partTimeAnalysis.AnnualSalary
			} else {
				cf.PartTimeSalary[p.Name] = decimalZero
				cf.PartTimeTSPContributions[p.Name] = decimalZero
				cf.FERSSupplementReduction[p.Name] = decimalZero
			}

			retiredThisYear := st.retirementYear != nil && yr == *st.retirementYear
			retiredFraction := decimalZero
			if retiredThisYear {
				retiredFraction = decimalOne.Sub(workFraction)
				if retiredFraction.LessThan(decimalZero) {
					retiredFraction = decimalZero
				}
			}

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

					// Calculate FERS Special Retirement Supplement if eligible
					retirementAge := p.Age(*st.retirementDate)
					if retirementAge < 62 && p.SSBenefit62.GreaterThan(decimalZero) {
						serviceYears := p.YearsOfService(*st.retirementDate)
						st.fersSupplementAnnual = CalculateFERSSpecialRetirementSupplement(p.SSBenefit62, serviceYears, retirementAge)
						st.fersSupplementStartYear = startYr
					}
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
					// Get TSP contribution policy from assumptions
					policy := "continue_until_retirement" // default
					if assumptions.TSPContribPolicy != "" {
						policy = assumptions.TSPContribPolicy
					}

					// Use policy-aware contribution calculation
					employeeContribution := tspContribForYear(salaryForYear, employeePct, 26, st.retirementDate, yr, startYear, policy)
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

			// Add part-time TSP contributions
			if cf.PartTimeTSPContributions[p.Name].GreaterThan(decimalZero) {
				cf.ParticipantTSPContributions[p.Name] = cf.ParticipantTSPContributions[p.Name].Add(cf.PartTimeTSPContributions[p.Name])
				st.tspBalance = st.tspBalance.Add(cf.PartTimeTSPContributions[p.Name])
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

			// Handle FERS Special Retirement Supplement
			fersSupplementValue := decimalZero
			if st.retired && st.fersSupplementAnnual.GreaterThan(decimalZero) && age < 62 {
				// Apply COLA to FERS supplement (same as pension COLA rules)
				if st.fersSupplementStartYear != nil && yr > *st.fersSupplementStartYear {
					if p.IsFederal {
						st.fersSupplementAnnual = applyParticipantFERSCOLA(st.fersSupplementAnnual, cola, age)
					} else {
						st.fersSupplementAnnual = st.fersSupplementAnnual.Mul(onePlus(cola))
					}
				}

				fersSupplementValue = st.fersSupplementAnnual

				// Apply retirement year proration if applicable
				if st.fersSupplementStartYear != nil && yr == *st.fersSupplementStartYear {
					fractionWorked := computeWorkFraction(st.retirementDate, yearDate)
					fersSupplementValue = st.fersSupplementAnnual.Mul(decimalOne.Sub(fractionWorked))
				}
			}
			cf.FERSSupplements[p.Name] = fersSupplementValue

			// Apply FERS supplement reduction due to part-time work earnings
			if cf.FERSSupplementReduction[p.Name].GreaterThan(decimalZero) {
				reducedSupplement := fersSupplementValue.Sub(cf.FERSSupplementReduction[p.Name])
				if reducedSupplement.LessThan(decimalZero) {
					reducedSupplement = decimalZero
				}
				cf.FERSSupplements[p.Name] = reducedSupplement
			}

			ssBenefit := decimalZero
			if st.ssStarted {
				if st.ssStartYear != nil && yr > *st.ssStartYear {
					st.ssAnnualFull = st.ssAnnualFull.Mul(onePlus(cola))
				}
				ssBenefit = st.ssAnnualFull
			} else if ageEnd >= st.ssStartAge {
				fullAnnual := computeSSAnnualBenefit(p, st.ssStartAge)
				benefit := fullAnnual
				benefit = computeSSBirthdayProration(benefit, p, st.ssStartAge, st.retirementYear, st.retirementDate, yr, yearDate, yearEnd, age, ageEnd)
				benefit = computeSSRetirementAdjustment(benefit, fullAnnual, p, st.ssStartAge, st.retirementYear, st.retirementDate, yr, yearDate)
				if benefit.GreaterThan(decimalZero) {
					st.ssAnnualFull = fullAnnual
					st.ssStarted = true
					startIdx := new(int)
					*startIdx = yr
					st.ssStartYear = startIdx
					ssBenefit = benefit
				}
			}
			if st.ssStarted {
				cf.SSBenefits[p.Name] = ssBenefit
			}

			// Calculate withdrawal using sequencing strategy
			if st.retired && (st.tspBalance.GreaterThan(decimalZero) || (p.TaxableAccountBalance != nil && p.TaxableAccountBalance.GreaterThan(decimalZero))) {
				withdrawal := decimalZero

				// Check for RMD requirement first
				isRMDYear := age >= 73 // RMD age is 73 for 2025+
				rmdAmount := decimalZero
				if isRMDYear && st.tspBalanceTraditional.GreaterThan(decimalZero) {
					// Use proper RMD calculation
					rmdCalc := NewRMDCalculator(p.BirthDate.Year())
					rmdAmount = rmdCalc.CalculateRMD(st.tspBalanceTraditional, age)
				}

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

					// Ensure RMD is met if required
					if isRMDYear && rmdAmount.GreaterThan(withdrawal) {
						withdrawal = rmdAmount
					}
				}
				if retiredThisYear && withdrawal.GreaterThan(decimalZero) {
					withdrawal = withdrawal.Mul(retiredFraction)
				}
				if singleSurvivorName != "" && p.Name == singleSurvivorName {
					if survivorSpendingFactor.LessThan(decimalOne) {
						withdrawal = withdrawal.Mul(survivorSpendingFactor)
					}
				}
				// Use sequencing strategy if withdrawal sequencing is configured
				if scenario.WithdrawalSequencing != nil && withdrawal.GreaterThan(decimalZero) {
					// Determine if this is an RMD year
					isRMDYear := age >= 73 // RMD age is 73 for 2025+
					rmdAmount := decimalZero
					if isRMDYear && st.tspBalanceTraditional.GreaterThan(decimalZero) {
						// Simple RMD calculation: balance / (life expectancy)
						// Using 27.4 as divisor for age 73 (simplified)
						rmdAmount = st.tspBalanceTraditional.Div(decimal.NewFromFloat(27.4))
					}

					// Create withdrawal sources
					sources := sequencing.CreateWithdrawalSources(
						p,
						st.tspBalanceTraditional,
						st.tspBalanceRoth,
						isRMDYear,
						rmdAmount,
					)

					// Create strategy context
					currentOrdinaryIncome := cf.GetTotalPension().Add(cf.GetTotalSalary())
					magiCurrent := cf.MAGI
					ctx := sequencing.CreateStrategyContext(
						withdrawal,
						currentOrdinaryIncome,
						magiCurrent,
						isRMDYear,
						scenario.WithdrawalSequencing,
					)

					// Create and execute strategy
					strategy := sequencing.CreateStrategy(scenario.WithdrawalSequencing)
					plan := strategy.Plan(sources, ctx)

					// Apply the withdrawal plan
					totalWithdrawn := decimalZero
					taxableWithdrawn := decimalZero
					traditionalWithdrawn := decimalZero
					rothWithdrawn := decimalZero

					for _, allocation := range plan.Allocations {
						switch allocation.Source {
						case "taxable":
							if p.TaxableAccountBalance != nil {
								withdrawAmount := allocation.Gross
								if withdrawAmount.GreaterThan(*p.TaxableAccountBalance) {
									withdrawAmount = *p.TaxableAccountBalance
								}
								*p.TaxableAccountBalance = p.TaxableAccountBalance.Sub(withdrawAmount)
								taxableWithdrawn = taxableWithdrawn.Add(withdrawAmount)
								totalWithdrawn = totalWithdrawn.Add(withdrawAmount)
							}
						case "traditional":
							withdrawAmount := allocation.Gross
							if withdrawAmount.GreaterThan(st.tspBalanceTraditional) {
								withdrawAmount = st.tspBalanceTraditional
							}
							st.tspBalanceTraditional = st.tspBalanceTraditional.Sub(withdrawAmount)
							traditionalWithdrawn = traditionalWithdrawn.Add(withdrawAmount)
							totalWithdrawn = totalWithdrawn.Add(withdrawAmount)
						case "roth":
							withdrawAmount := allocation.Gross
							if withdrawAmount.GreaterThan(st.tspBalanceRoth) {
								withdrawAmount = st.tspBalanceRoth
							}
							st.tspBalanceRoth = st.tspBalanceRoth.Sub(withdrawAmount)
							rothWithdrawn = rothWithdrawn.Add(withdrawAmount)
							totalWithdrawn = totalWithdrawn.Add(withdrawAmount)
						}
					}

					// Update balances and cash flow
					st.tspBalance = st.tspBalanceTraditional.Add(st.tspBalanceRoth)
					cf.TSPWithdrawals[p.Name] = traditionalWithdrawn.Add(rothWithdrawn)
					cf.WithdrawalTaxable = cf.WithdrawalTaxable.Add(taxableWithdrawn)
					cf.WithdrawalTraditional = cf.WithdrawalTraditional.Add(traditionalWithdrawn)
					cf.WithdrawalRoth = cf.WithdrawalRoth.Add(rothWithdrawn)
				} else {
					// Fallback to proportional withdrawal if no sequencing configured
					if withdrawal.GreaterThan(st.tspBalance) {
						withdrawal = st.tspBalance
					}

					// Proportionally split between traditional and Roth based on starting mix
					tradPortion := decimalZero
					rothPortion := decimalZero
					totalBefore := st.tspBalance
					if totalBefore.GreaterThan(decimalZero) {
						tradRatio := decimalZero
						if st.tspBalanceTraditional.GreaterThan(decimalZero) {
							tradRatio = st.tspBalanceTraditional.Div(totalBefore)
						}
						tradPortion = withdrawal.Mul(tradRatio)
						rothPortion = withdrawal.Sub(tradPortion)
					}

					st.tspBalanceTraditional = st.tspBalanceTraditional.Sub(tradPortion)
					if st.tspBalanceTraditional.LessThan(decimalZero) {
						st.tspBalanceTraditional = decimalZero
					}
					st.tspBalanceRoth = st.tspBalanceRoth.Sub(rothPortion)
					if st.tspBalanceRoth.LessThan(decimalZero) {
						st.tspBalanceRoth = decimalZero
					}
					st.tspBalance = st.tspBalance.Sub(withdrawal)
					cf.TSPWithdrawals[p.Name] = withdrawal
					cf.WithdrawalTraditional = cf.WithdrawalTraditional.Add(tradPortion)
					cf.WithdrawalRoth = cf.WithdrawalRoth.Add(rothPortion)
				}
			}

			// Apply Roth conversions for this year
			if ps, exists := scenario.ParticipantScenarios[p.Name]; exists && ps.RothConversions != nil {
				currentYear := startYear + yr
				for _, conversion := range ps.RothConversions.Conversions {
					if conversion.Year == currentYear {
						// Convert from Traditional to Roth
						conversionAmount := conversion.Amount
						if conversionAmount.GreaterThan(st.tspBalanceTraditional) {
							conversionAmount = st.tspBalanceTraditional
						}

						// Move from Traditional to Roth
						st.tspBalanceTraditional = st.tspBalanceTraditional.Sub(conversionAmount)
						st.tspBalanceRoth = st.tspBalanceRoth.Add(conversionAmount)

						// Update total TSP balance
						st.tspBalance = st.tspBalanceTraditional.Add(st.tspBalanceRoth)

						// Add conversion amount to taxable income for this year
						// This will be picked up in the tax calculation
						cf.TSPWithdrawals[p.Name] = cf.TSPWithdrawals[p.Name].Add(conversionAmount)
					}
				}
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

		// Check if any participant is in an RMD year (household-level)
		cf.IsRMDYear = false
		cf.RMDAmount = decimalZero
		for _, name := range participantNames {
			if !cf.IsDeceased[name] {
				age := cf.Ages[name]
				if age >= 73 {
					cf.IsRMDYear = true
					// Calculate RMD for the participant with Traditional TSP balance
					if st, ok := states[name]; ok && st.tspBalanceTraditional.GreaterThan(decimalZero) {
						// Find the participant in household.Participants
						for _, p := range household.Participants {
							if p.Name == name {
								rmdCalc := NewRMDCalculator(p.BirthDate.Year())
								rmdAmount := rmdCalc.CalculateRMD(st.tspBalanceTraditional, age)
								if rmdAmount.GreaterThan(cf.RMDAmount) {
									cf.RMDAmount = rmdAmount
								}
								break
							}
						}
					}
				}
			}
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

		// Legacy FEHB calculation for backward compatibility
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

		// Calculate comprehensive healthcare costs
		healthcareCalc := NewHealthcareCostCalculator()

		// Get living participants for healthcare calculation
		livingParticipants := make([]domain.Participant, 0, len(livingNames))
		for _, name := range livingNames {
			for _, p := range household.Participants {
				if p.Name == name {
					livingParticipants = append(livingParticipants, p)
					break
				}
			}
		}

		// Calculate household healthcare costs
		cf.HealthcareCosts = healthcareCalc.CalculateHouseholdHealthcareCosts(
			livingParticipants,
			cf.Ages,
			startYear+yr,
			cf.MAGI,
			filingStatus,
		)

		seniors := 0
		// Sort participant names for deterministic processing order
		var ageNames []string
		for name := range cf.Ages {
			ageNames = append(ageNames, name)
		}
		sort.Strings(ageNames)

		for _, name := range ageNames {
			age := cf.Ages[name]
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
		cf.IsRetired = isRetiredHousehold

		if ce != nil && ce.TaxCalc != nil {
			cf.FederalTax = ce.TaxCalc.calculateFederalTaxWithStatus(taxable, filingStatus, seniors)
			cf.StateTax = ce.TaxCalc.StateTaxCalc.CalculateTax(taxable, isRetiredHousehold)
			hasWageIncome := taxable.WageIncome.GreaterThan(decimalZero)
			applyRetiredExemption := isRetiredHousehold && !hasWageIncome
			cf.LocalTax = ce.TaxCalc.LocalTaxCalc.CalculateEIT(taxable.WageIncome, applyRetiredExemption)
			if hasWageIncome {
				// Calculate FICA per person with separate wage-base caps
				participantWages := make([]decimal.Decimal, 0, len(participantNames))
				for _, name := range participantNames {
					if !cf.IsDeceased[name] {
						participantWages = append(participantWages, cf.Salaries[name])
					}
				}

				// Handle different numbers of living participants
				if len(participantWages) == 2 {
					cf.FICATax = ce.TaxCalc.FICATaxCalc.CalculateFICAForTwoPersons(participantWages[0], participantWages[1])
				} else if len(participantWages) == 1 {
					cf.FICATax = ce.TaxCalc.FICATaxCalc.CalculateFICA(participantWages[0], participantWages[0])
				} else {
					// Fallback to original method for more than 2 people
					cf.FICATax = ce.TaxCalc.FICATaxCalc.CalculateFICA(taxable.WageIncome, taxable.WageIncome)
				}
			} else {
				cf.FICATax = decimalZero
			}
		}

		cf.TotalGrossIncome = cf.CalculateTotalIncome()
		cf.CalculateNetIncome()

		// Determine Medicare eligibility (any participant age 65+)
		cf.IsMedicareEligible = false
		// Sort participant names for deterministic processing order
		var medicareNames []string
		for name := range cf.Ages {
			medicareNames = append(medicareNames, name)
		}
		sort.Strings(medicareNames)
		for _, name := range medicareNames {
			if cf.Ages[name] >= 65 {
				cf.IsMedicareEligible = true
				break
			}
		}

		// Calculate MAGI for IRMAA determination
		cf.MAGI = CalculateMAGI(cf)

		// Calculate IRMAA risk if Medicare eligible
		if cf.IsMedicareEligible {
			isMarried := household.FilingStatus == "married_filing_jointly"
			mc := NewMedicareCalculator()

			risk, tier, surcharge, distance := CalculateIRMAARiskStatus(
				cf.MAGI,
				isMarried,
				mc,
			)

			cf.IRMAARiskStatus = string(risk)
			cf.IRMAALevel = tier
			cf.IRMAASurcharge = surcharge
			cf.IRMAADistanceToNext = distance
		}

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

func computeSSBirthdayProration(currentBenefit decimal.Decimal, p *domain.Participant, ssStartAge int, retirementYear *int, retirementDate *time.Time, yearIdx int, yearStart, yearEnd time.Time, ageAtStart, ageAtEnd int) decimal.Decimal {
	if currentBenefit.LessThanOrEqual(decimalZero) {
		return currentBenefit
	}

	if ageAtStart >= ssStartAge || ageAtEnd < ssStartAge {
		return currentBenefit
	}

	if retirementYear != nil && retirementDate != nil && yearIdx == *retirementYear {
		birthdayThisYear := time.Date(yearStart.Year(), p.BirthDate.Month(), p.BirthDate.Day(), 0, 0, 0, 0, time.UTC)
		if retirementDate.Before(birthdayThisYear) {
			return currentBenefit
		}
	}

	birthdayThisYear := time.Date(yearStart.Year(), p.BirthDate.Month(), p.BirthDate.Day(), 0, 0, 0, 0, time.UTC)
	daysAfter := yearEnd.Sub(birthdayThisYear).Hours() / 24.0
	daysInYear := 365.0
	if isLeapYear(yearStart.Year()) {
		daysInYear = 366.0
	}
	fraction := daysAfter / daysInYear
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}

	return currentBenefit.Mul(decimal.NewFromFloat(fraction))
}

func computeSSRetirementAdjustment(currentBenefit, fullAnnual decimal.Decimal, p *domain.Participant, ssStartAge int, retirementYear *int, retirementDate *time.Time, yearIdx int, yearStart time.Time) decimal.Decimal {
	if retirementYear == nil || retirementDate == nil || yearIdx != *retirementYear {
		return currentBenefit
	}

	ageAtRetirement := p.Age(*retirementDate)
	if ageAtRetirement < ssStartAge {
		return decimalZero
	}

	birthdayThisYear := time.Date(yearStart.Year(), p.BirthDate.Month(), p.BirthDate.Day(), 0, 0, 0, 0, time.UTC)
	if retirementDate.Before(birthdayThisYear) {
		ssStartDate := time.Date(retirementDate.Year(), retirementDate.Month()+1, 1, 0, 0, 0, 0, time.UTC)
		monthsOfBenefits := 12 - int(ssStartDate.Month()) + 1
		if ssStartDate.Year() > retirementDate.Year() {
			monthsOfBenefits = 0
		}
		if monthsOfBenefits < 0 {
			monthsOfBenefits = 0
		}
		if monthsOfBenefits > 12 {
			monthsOfBenefits = 12
		}
		monthlyBenefit := fullAnnual.Div(decimalTwelve)
		return monthlyBenefit.Mul(decimal.NewFromInt(int64(monthsOfBenefits)))
	}

	return currentBenefit
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
