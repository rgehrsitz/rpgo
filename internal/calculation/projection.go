package calculation

import (
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/rpgo/retirement-calculator/pkg/dateutil"
	"github.com/shopspring/decimal"
)

// GenerateAnnualProjection generates annual cash flow projections for a scenario
func (ce *CalculationEngine) GenerateAnnualProjection(robert, dawn *domain.Employee, scenario *domain.Scenario, assumptions *domain.GlobalAssumptions, federalRules domain.FederalRules) []domain.AnnualCashFlow {
	projection := make([]domain.AnnualCashFlow, assumptions.ProjectionYears)

	// Determine retirement year (0-based index)
	// Projection starts at ProjectionBaseYear (first year of projection)
	projectionStartYear := ProjectionBaseYear
	retirementYear := scenario.Robert.RetirementDate.Year() - projectionStartYear
	if retirementYear < 0 {
		retirementYear = 0
	}

	// Initialize TSP balances
	currentTSPTraditionalRobert := robert.TSPBalanceTraditional
	currentTSPRothRobert := robert.TSPBalanceRoth
	currentTSPTraditionalDawn := dawn.TSPBalanceTraditional
	currentTSPRothDawn := dawn.TSPBalanceRoth

	// Create TSP withdrawal strategies
	// For Scenario 2, we need to account for extra growth before withdrawals start
	robertStrategy := ce.createTSPStrategy(&scenario.Robert, currentTSPTraditionalRobert.Add(currentTSPRothRobert), assumptions.InflationRate)
	dawnStrategy := ce.createTSPStrategy(&scenario.Dawn, currentTSPTraditionalDawn.Add(currentTSPRothDawn), assumptions.InflationRate)

	// Mortality derived dates using helper
	robertDeathYearIndex, dawnDeathYearIndex := deriveDeathYearIndexes(scenario, robert, dawn, assumptions.ProjectionYears)

	survivorSpendingFactor := decimal.NewFromFloat(1.0)
	if scenario.Mortality != nil && scenario.Mortality.Assumptions != nil && !scenario.Mortality.Assumptions.SurvivorSpendingFactor.IsZero() {
		survivorSpendingFactor = scenario.Mortality.Assumptions.SurvivorSpendingFactor
	}

	robertIsDeceased := false
	dawnIsDeceased := false

	for year := 0; year < assumptions.ProjectionYears; year++ {
		projectionDate := time.Date(projectionStartYear, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(year, 0, 0)
		ageRobert := robert.Age(projectionDate)
		ageDawn := dawn.Age(projectionDate)

		// Calculate partial year retirement for each person
		// Projection starts at ProjectionBaseYear, so year 0 = ProjectionBaseYear, etc.
		projectionStartYear := ProjectionBaseYear
		robertRetirementYear := scenario.Robert.RetirementDate.Year() - projectionStartYear
		dawnRetirementYear := scenario.Dawn.RetirementDate.Year() - projectionStartYear

		// Determine if each person is retired for this year
		isRobertRetired := year >= robertRetirementYear
		isDawnRetired := year >= dawnRetirementYear

		// Calculate partial year factors (what portion of the year each person works)
		var robertWorkFraction, dawnWorkFraction decimal.Decimal

		if year == robertRetirementYear && robertRetirementYear >= 0 {
			// Robert retires during this year - calculate work fraction
			robertRetirementDate := scenario.Robert.RetirementDate
			yearStart := time.Date(projectionDate.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
			daysWorked := robertRetirementDate.Sub(yearStart).Hours() / 24
			daysInYear := 365.0
			robertWorkFraction = decimal.NewFromFloat(daysWorked / daysInYear)
		} else if isRobertRetired {
			robertWorkFraction = decimal.Zero
		} else {
			robertWorkFraction = decimal.NewFromInt(1)
		}

		if year == dawnRetirementYear && dawnRetirementYear >= 0 {
			// Dawn retires during this year - calculate work fraction
			dawnRetirementDate := scenario.Dawn.RetirementDate
			yearStart := time.Date(projectionDate.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
			daysWorked := dawnRetirementDate.Sub(yearStart).Hours() / 24
			daysInYear := 365.0
			dawnWorkFraction = decimal.NewFromFloat(daysWorked / daysInYear)
		} else if isDawnRetired {
			dawnWorkFraction = decimal.Zero
		} else {
			dawnWorkFraction = decimal.NewFromInt(1)
		}

		// Apply death events at start-of-year (Phase 1: incomes stop this year)
		if robertDeathYearIndex != nil && year >= *robertDeathYearIndex {
			robertIsDeceased = true
		}
		if dawnDeathYearIndex != nil && year >= *dawnDeathYearIndex {
			dawnIsDeceased = true
		}

		// If a spouse just became deceased this year and transfer mode is merge, merge TSP balances into survivor (traditional+roth)
		if scenario.Mortality != nil && scenario.Mortality.Assumptions != nil && scenario.Mortality.Assumptions.TSPSpousalTransfer == "merge" {
			if robertIsDeceased && !dawnIsDeceased {
				// Move Robert balances into Dawn's (simple add)
				currentTSPTraditionalDawn = currentTSPTraditionalDawn.Add(currentTSPTraditionalRobert)
				currentTSPRothDawn = currentTSPRothDawn.Add(currentTSPRothRobert)
				currentTSPTraditionalRobert = decimal.Zero
				currentTSPRothRobert = decimal.Zero
			}
			if dawnIsDeceased && !robertIsDeceased {
				currentTSPTraditionalRobert = currentTSPTraditionalRobert.Add(currentTSPTraditionalDawn)
				currentTSPRothRobert = currentTSPRothRobert.Add(currentTSPRothDawn)
				currentTSPTraditionalDawn = decimal.Zero
				currentTSPRothDawn = decimal.Zero
			}
		}

		// Calculate FERS pensions (only for retired portion of year, and not after death)
		var pensionRobert, pensionDawn decimal.Decimal
		var survivorPensionRobert, survivorPensionDawn decimal.Decimal
		if isRobertRetired && !robertIsDeceased {
			pensionRobert = CalculatePensionForYear(robert, scenario.Robert.RetirementDate, year-robertRetirementYear, assumptions.InflationRate)
			// Adjust for partial year if retiring this year
			if year == robertRetirementYear {
				pensionRobert = pensionRobert.Mul(decimal.NewFromInt(1).Sub(robertWorkFraction))
			}

			// Debug output for pension calculation
			if ce.Debug && year == robertRetirementYear {
				ce.Logger.Debugf("DEBUG: Robert's pension calculation for year %d", ProjectionBaseYear+year)
				ce.Logger.Debugf("  Retirement date: %s", scenario.Robert.RetirementDate.Format("2006-01-02"))
				ce.Logger.Debugf("  Age at retirement: %d", robert.Age(scenario.Robert.RetirementDate))
				ce.Logger.Debugf("  Years of service: %s", robert.YearsOfService(scenario.Robert.RetirementDate).StringFixed(2))
				ce.Logger.Debugf("  High-3 salary: %s", robert.High3Salary.StringFixed(2))

				// Get detailed pension calculation
				pensionCalc := CalculateFERSPension(robert, scenario.Robert.RetirementDate)
				ce.Logger.Debugf("  Multiplier: %s", pensionCalc.Multiplier.StringFixed(4))
				ce.Logger.Debugf("  ANNUAL pension (before reduction): $%s", pensionCalc.AnnualPension.StringFixed(2))
				ce.Logger.Debugf("  Survivor election: %s", pensionCalc.SurvivorElection.StringFixed(4))
				ce.Logger.Debugf("  ANNUAL pension (final): $%s", pensionCalc.ReducedPension.StringFixed(2))
				ce.Logger.Debugf("  MONTHLY pension amount: $%s", pensionCalc.ReducedPension.Div(decimal.NewFromInt(12)).StringFixed(2))
				ce.Logger.Debugf("  Current-year cash received (partial): $%s", pensionRobert.StringFixed(2))
			}
		}
		if isDawnRetired && !dawnIsDeceased {
			pensionDawn = CalculatePensionForYear(dawn, scenario.Dawn.RetirementDate, year-dawnRetirementYear, assumptions.InflationRate)
			// Adjust for partial year if retiring this year
			if year == dawnRetirementYear {
				pensionDawn = pensionDawn.Mul(decimal.NewFromInt(1).Sub(dawnWorkFraction))
			}
		}

		// Survivor pension logic with pro-rating in death year
		if scenario.Mortality != nil {
			if robertIsDeceased && !dawnIsDeceased && isRobertRetired {
				baseCalc := CalculateFERSPension(robert, scenario.Robert.RetirementDate)
				yearsSinceRet := year - robertRetirementYear
				if yearsSinceRet < 0 {
					yearsSinceRet = 0
				}
				currentSurvivor := baseCalc.SurvivorAnnuity
				for cy := 1; cy <= yearsSinceRet; cy++ {
					projDate := scenario.Robert.RetirementDate.AddDate(cy, 0, 0)
					ageAt := robert.Age(projDate)
					currentSurvivor = ApplyFERSPensionCOLA(currentSurvivor, assumptions.InflationRate, ageAt)
				}
				if robertDeathYearIndex != nil && year >= *robertDeathYearIndex {
					// Pro-rate in death year: survivor receives only portion AFTER death
					var deathDate *time.Time
					if scenario.Mortality.Robert != nil {
						deathDate = scenario.Mortality.Robert.DeathDate
					}
					frac, occurred := deathFractionInYear(robertDeathYearIndex, year, deathDate)
					if occurred {
						// Pension stream for deceased stops at death; survivor annuity starts month after death -> approximate with (1-frac)
						survivorPensionDawn = currentSurvivor.Mul(decimal.NewFromInt(1).Sub(frac))
					} else {
						survivorPensionDawn = currentSurvivor
					}
				}
			}
			if dawnIsDeceased && !robertIsDeceased && isDawnRetired {
				baseCalc := CalculateFERSPension(dawn, scenario.Dawn.RetirementDate)
				yearsSinceRet := year - dawnRetirementYear
				if yearsSinceRet < 0 {
					yearsSinceRet = 0
				}
				currentSurvivor := baseCalc.SurvivorAnnuity
				for cy := 1; cy <= yearsSinceRet; cy++ {
					projDate := scenario.Dawn.RetirementDate.AddDate(cy, 0, 0)
					ageAt := dawn.Age(projDate)
					currentSurvivor = ApplyFERSPensionCOLA(currentSurvivor, assumptions.InflationRate, ageAt)
				}
				if dawnDeathYearIndex != nil && year >= *dawnDeathYearIndex {
					var deathDate *time.Time
					if scenario.Mortality.Dawn != nil {
						deathDate = scenario.Mortality.Dawn.DeathDate
					}
					frac, occurred := deathFractionInYear(dawnDeathYearIndex, year, deathDate)
					if occurred {
						survivorPensionRobert = currentSurvivor.Mul(decimal.NewFromInt(1).Sub(frac))
					} else {
						survivorPensionRobert = currentSurvivor
					}
				}
			}
		}

		// Calculate Social Security benefits
		ssRobert := decimal.Zero
		if !robertIsDeceased {
			ssRobert = CalculateSSBenefitForYear(robert, scenario.Robert.SSStartAge, year, assumptions.COLAGeneralRate)
		}
		ssDawn := decimal.Zero
		if !dawnIsDeceased {
			ssDawn = CalculateSSBenefitForYear(dawn, scenario.Dawn.SSStartAge, year, assumptions.COLAGeneralRate)
		}
		// Survivor SS refined: compute survivor benefit factoring early-claim reduction
		if robertIsDeceased && !dawnIsDeceased {
			fra := dateutil.FullRetirementAge(dawn.BirthDate)
			// Use deceased's current-year benefit (pre-death). If zero (due to modeling order), recalc directly.
			deceasedBenefit := CalculateSSBenefitForYear(robert, scenario.Robert.SSStartAge, year, assumptions.COLAGeneralRate)
			candidate := CalculateSurvivorSSBenefit(deceasedBenefit, ageDawn, fra)
			if candidate.GreaterThan(ssDawn) {
				ssDawn = candidate
			}
		}
		if dawnIsDeceased && !robertIsDeceased {
			fra := dateutil.FullRetirementAge(robert.BirthDate)
			deceasedBenefit := CalculateSSBenefitForYear(dawn, scenario.Dawn.SSStartAge, year, assumptions.COLAGeneralRate)
			candidate := CalculateSurvivorSSBenefit(deceasedBenefit, ageRobert, fra)
			if candidate.GreaterThan(ssRobert) {
				ssRobert = candidate
			}
		}

		// Adjust Social Security for partial year based on eligibility and retirement timing
		if year == robertRetirementYear && robertRetirementYear >= 0 {
			// Robert can start SS when he retires (if 62+) or when he turns 62, whichever is later
			ageAtRetirement := robert.Age(scenario.Robert.RetirementDate)
			if ageAtRetirement >= scenario.Robert.SSStartAge {
				// Can start SS immediately upon retirement
				ssRobert = ssRobert.Mul(decimal.NewFromInt(1).Sub(robertWorkFraction))
			} else {
				// Will start SS later when turns 62
				ssRobert = decimal.Zero
			}
		}
		if year == dawnRetirementYear && dawnRetirementYear >= 0 {
			// Dawn turns 62 on July 31, 2025 and retires August 30, 2025
			// She can start SS immediately upon retirement in August 2025
			ageAtRetirement := dawn.Age(scenario.Dawn.RetirementDate)
			if ageAtRetirement >= scenario.Dawn.SSStartAge {
				// Dawn can start SS in September 2025 (month after retirement)
				retirementDate := scenario.Dawn.RetirementDate
				ssStartDate := time.Date(retirementDate.Year(), retirementDate.Month()+1, 1, 0, 0, 0, 0, time.UTC)
				monthsOfBenefits := 12 - int(ssStartDate.Month()) + 1 // Sept(9) to Dec(12) = 4 months

				// Prorate SS for partial year
				ssMonthlyBenefit := ssDawn.Div(decimal.NewFromInt(12))
				ssDawn = ssMonthlyBenefit.Mul(decimal.NewFromInt(int64(monthsOfBenefits)))
			} else {
				ssDawn = decimal.Zero
			}
		}

		// Calculate FERS Special Retirement Supplement (only if retired)
		var srsRobert, srsDawn decimal.Decimal
		if isRobertRetired && !robertIsDeceased {
			srsRobert = CalculateFERSSupplementYear(robert, scenario.Robert.RetirementDate, year-robertRetirementYear, assumptions.InflationRate)
			// Adjust for partial year if retiring this year
			if year == robertRetirementYear {
				srsRobert = srsRobert.Mul(decimal.NewFromInt(1).Sub(robertWorkFraction))
			}
		}
		if isDawnRetired && !dawnIsDeceased {
			srsDawn = CalculateFERSSupplementYear(dawn, scenario.Dawn.RetirementDate, year-dawnRetirementYear, assumptions.InflationRate)
			// Adjust for partial year if retiring this year
			if year == dawnRetirementYear {
				srsDawn = srsDawn.Mul(decimal.NewFromInt(1).Sub(dawnWorkFraction))
			}
		}

		// Calculate TSP withdrawals and update balances
		var tspWithdrawalRobert, tspWithdrawalDawn decimal.Decimal
		if isRobertRetired && !robertIsDeceased {
			// For 4% rule: Always withdraw 4% of initial balance (adjusted for inflation)
			if scenario.Robert.TSPWithdrawalStrategy == "4_percent_rule" {
				// Use the 4% rule strategy to calculate withdrawals
				tspWithdrawalRobert = robertStrategy.CalculateWithdrawal(
					currentTSPTraditionalRobert.Add(currentTSPRothRobert),
					year-robertRetirementYear+1,
					decimal.Zero, // Not used for 4% rule
					ageRobert,
					dateutil.IsRMDYear(robert.BirthDate, projectionDate),
					CalculateRMD(currentTSPTraditionalRobert, robert.BirthDate.Year(), ageRobert),
				)
				// Adjust for partial year if retiring this year
				if year == robertRetirementYear {
					tspWithdrawalRobert = tspWithdrawalRobert.Mul(decimal.NewFromInt(1).Sub(robertWorkFraction))
				}
			} else {
				// For need_based: Use the target monthly amount
				targetIncome := pensionRobert.Add(pensionDawn).Add(ssRobert).Add(ssDawn).Add(srsRobert).Add(srsDawn)

				// Calculate withdrawals
				tspWithdrawalRobert = robertStrategy.CalculateWithdrawal(
					currentTSPTraditionalRobert.Add(currentTSPRothRobert),
					year-robertRetirementYear+1,
					targetIncome,
					ageRobert,
					dateutil.IsRMDYear(robert.BirthDate, projectionDate),
					CalculateRMD(currentTSPTraditionalRobert, robert.BirthDate.Year(), ageRobert),
				)
				// Adjust for partial year if retiring this year
				if year == robertRetirementYear {
					tspWithdrawalRobert = tspWithdrawalRobert.Mul(decimal.NewFromInt(1).Sub(robertWorkFraction))
				}
			}
		}

		if isDawnRetired && !dawnIsDeceased {
			if scenario.Dawn.TSPWithdrawalStrategy == "4_percent_rule" {
				tspWithdrawalDawn = dawnStrategy.CalculateWithdrawal(
					currentTSPTraditionalDawn.Add(currentTSPRothDawn),
					year-dawnRetirementYear+1,
					decimal.Zero, // Not used for 4% rule
					ageDawn,
					dateutil.IsRMDYear(dawn.BirthDate, projectionDate),
					CalculateRMD(currentTSPTraditionalDawn, dawn.BirthDate.Year(), ageDawn),
				)
				// Adjust for partial year if retiring this year
				if year == dawnRetirementYear {
					tspWithdrawalDawn = tspWithdrawalDawn.Mul(decimal.NewFromInt(1).Sub(dawnWorkFraction))
				}
			} else {
				// For need_based: Use the target monthly amount
				targetIncome := pensionRobert.Add(pensionDawn).Add(ssRobert).Add(ssDawn).Add(srsRobert).Add(srsDawn)

				// Calculate withdrawals
				tspWithdrawalDawn = dawnStrategy.CalculateWithdrawal(
					currentTSPTraditionalDawn.Add(currentTSPRothDawn),
					year-dawnRetirementYear+1,
					targetIncome,
					ageDawn,
					dateutil.IsRMDYear(dawn.BirthDate, projectionDate),
					CalculateRMD(currentTSPTraditionalDawn, dawn.BirthDate.Year(), ageDawn),
				)
				// Adjust for partial year if retiring this year
				if year == dawnRetirementYear {
					tspWithdrawalDawn = tspWithdrawalDawn.Mul(decimal.NewFromInt(1).Sub(dawnWorkFraction))
				}
			}
		}

		// Update TSP balances
		if isRobertRetired {
			// Post-retirement TSP growth with withdrawals
			// Use lifecycle fund allocation if available, otherwise use default return rate
			if robert.TSPLifecycleFund != nil || robert.TSPAllocation != nil {
				// Apply withdrawal first
				if tspWithdrawalRobert.GreaterThan(currentTSPTraditionalRobert) {
					// Take from Roth if traditional is insufficient
					remainingWithdrawal := tspWithdrawalRobert.Sub(currentTSPTraditionalRobert)
					currentTSPTraditionalRobert = decimal.Zero
					if remainingWithdrawal.GreaterThan(currentTSPRothRobert) {
						currentTSPRothRobert = decimal.Zero
					} else {
						currentTSPRothRobert = currentTSPRothRobert.Sub(remainingWithdrawal)
					}
				} else {
					currentTSPTraditionalRobert = currentTSPTraditionalRobert.Sub(tspWithdrawalRobert)
				}

				// Apply growth using lifecycle fund allocation
				allocation := ce.getTSPAllocationForEmployee(robert, projectionDate)
				weightedReturn := ce.calculateTSPReturnWithAllocation(allocation, projectionDate.Year())

				currentTSPTraditionalRobert = currentTSPTraditionalRobert.Mul(decimal.NewFromFloat(1).Add(weightedReturn))
				currentTSPRothRobert = currentTSPRothRobert.Mul(decimal.NewFromFloat(1).Add(weightedReturn))
			} else {
				currentTSPTraditionalRobert, currentTSPRothRobert = ce.updateTSPBalances(
					currentTSPTraditionalRobert, currentTSPRothRobert, tspWithdrawalRobert,
					assumptions.TSPReturnPostRetirement,
				)
			}
		} else {
			// Pre-retirement TSP growth with contributions
			// Use lifecycle fund allocation if available, otherwise use default return rate
			if robert.TSPLifecycleFund != nil || robert.TSPAllocation != nil {
				currentTSPTraditionalRobert = ce.growTSPBalanceWithAllocation(robert, currentTSPTraditionalRobert, robert.TotalAnnualTSPContribution(), projectionDate)
				currentTSPRothRobert = ce.growTSPBalanceWithAllocation(robert, currentTSPRothRobert, decimal.Zero, projectionDate)
			} else {
				currentTSPTraditionalRobert = ce.growTSPBalance(currentTSPTraditionalRobert, robert.TotalAnnualTSPContribution(), assumptions.TSPReturnPreRetirement)
				currentTSPRothRobert = ce.growTSPBalance(currentTSPRothRobert, decimal.Zero, assumptions.TSPReturnPreRetirement)
			}
		}

		if isDawnRetired {
			// Post-retirement TSP growth with withdrawals
			// Use lifecycle fund allocation if available, otherwise use default return rate
			if dawn.TSPLifecycleFund != nil || dawn.TSPAllocation != nil {
				// Apply withdrawal first
				if tspWithdrawalDawn.GreaterThan(currentTSPTraditionalDawn) {
					// Take from Roth if traditional is insufficient
					remainingWithdrawal := tspWithdrawalDawn.Sub(currentTSPTraditionalDawn)
					currentTSPTraditionalDawn = decimal.Zero
					if remainingWithdrawal.GreaterThan(currentTSPRothDawn) {
						currentTSPRothDawn = decimal.Zero
					} else {
						currentTSPRothDawn = currentTSPRothDawn.Sub(remainingWithdrawal)
					}
				} else {
					currentTSPTraditionalDawn = currentTSPTraditionalDawn.Sub(tspWithdrawalDawn)
				}

				// Apply growth using lifecycle fund allocation
				allocation := ce.getTSPAllocationForEmployee(dawn, projectionDate)
				weightedReturn := ce.calculateTSPReturnWithAllocation(allocation, projectionDate.Year())

				currentTSPTraditionalDawn = currentTSPTraditionalDawn.Mul(decimal.NewFromFloat(1).Add(weightedReturn))
				currentTSPRothDawn = currentTSPRothDawn.Mul(decimal.NewFromFloat(1).Add(weightedReturn))
			} else {
				currentTSPTraditionalDawn, currentTSPRothDawn = ce.updateTSPBalances(
					currentTSPTraditionalDawn, currentTSPRothDawn, tspWithdrawalDawn,
					assumptions.TSPReturnPostRetirement,
				)
			}
		} else {
			// Pre-retirement TSP growth with contributions
			// Use lifecycle fund allocation if available, otherwise use default return rate
			if dawn.TSPLifecycleFund != nil || dawn.TSPAllocation != nil {
				currentTSPTraditionalDawn = ce.growTSPBalanceWithAllocation(dawn, currentTSPTraditionalDawn, dawn.TotalAnnualTSPContribution(), projectionDate)
				currentTSPRothDawn = ce.growTSPBalanceWithAllocation(dawn, currentTSPRothDawn, decimal.Zero, projectionDate)
			} else {
				currentTSPTraditionalDawn = ce.growTSPBalance(currentTSPTraditionalDawn, dawn.TotalAnnualTSPContribution(), assumptions.TSPReturnPreRetirement)
				currentTSPRothDawn = ce.growTSPBalance(currentTSPRothDawn, decimal.Zero, assumptions.TSPReturnPreRetirement)
			}
		}

		// Debug TSP balances for Scenario 2 to show extra growth
		if ce.Debug && year == 1 && scenario.Robert.RetirementDate.Year() == 2027 {
			ce.Logger.Debugf("TSP Growth in Scenario 2 (year %d)", ProjectionBaseYear+year)
			ce.Logger.Debugf("  Robert's TSP balance: %s", currentTSPTraditionalRobert.Add(currentTSPRothRobert).StringFixed(2))
			ce.Logger.Debugf("  Dawn's TSP balance: %s", currentTSPTraditionalDawn.Add(currentTSPRothDawn).StringFixed(2))
			ce.Logger.Debugf("  Combined TSP balance: %s", currentTSPTraditionalRobert.Add(currentTSPRothRobert).Add(currentTSPTraditionalDawn).Add(currentTSPRothDawn).StringFixed(2))
			ce.Logger.Debugf("")
		}

		// Calculate FEHB premiums
		fehbPremium := CalculateFEHBPremium(robert, year, assumptions.FEHBPremiumInflation, federalRules.FEHBConfig)

		// Calculate Medicare premiums (if applicable)
		medicarePremium := ce.calculateMedicarePremium(robert, dawn, projectionDate,
			pensionRobert, pensionDawn, tspWithdrawalRobert, tspWithdrawalDawn, ssRobert, ssDawn)

		// Calculate taxes - handle transition years properly
		// Pass the actual working income and retirement income separately
		workingIncomeRobert := robert.CurrentSalary.Mul(robertWorkFraction)
		workingIncomeDawn := dawn.CurrentSalary.Mul(dawnWorkFraction)

		federalTax, stateTax, localTax, ficaTax, _, _, _, _ := ce.calculateTaxes(
			robert, dawn, scenario, year, isRobertRetired && isDawnRetired,
			pensionRobert, pensionDawn, survivorPensionRobert, survivorPensionDawn,
			tspWithdrawalRobert, tspWithdrawalDawn,
			ssRobert, ssDawn,
			workingIncomeRobert, workingIncomeDawn,
		)

		// Calculate TSP contributions (only for working portion of year)
		var tspContributions decimal.Decimal
		if (!isRobertRetired || !isDawnRetired) && !(robertIsDeceased || dawnIsDeceased) {
			robertContributions := robert.TotalAnnualTSPContribution().Mul(robertWorkFraction)
			dawnContributions := dawn.TotalAnnualTSPContribution().Mul(dawnWorkFraction)
			tspContributions = robertContributions.Add(dawnContributions)
		}

		cf := domain.NewAnnualCashFlow(year+1, projectionDate, []string{"robert", "dawn"})
		cf.Ages["robert"], cf.Ages["dawn"] = ageRobert, ageDawn
		cf.Salaries["robert"], cf.Salaries["dawn"] = robert.CurrentSalary.Mul(robertWorkFraction), dawn.CurrentSalary.Mul(dawnWorkFraction)
		cf.Pensions["robert"], cf.Pensions["dawn"] = pensionRobert, pensionDawn
		cf.TSPWithdrawals["robert"], cf.TSPWithdrawals["dawn"] = tspWithdrawalRobert, tspWithdrawalDawn
		cf.SSBenefits["robert"], cf.SSBenefits["dawn"] = ssRobert, ssDawn
		cf.FERSSupplements["robert"], cf.FERSSupplements["dawn"] = srsRobert, srsDawn
		cf.TSPBalances["robert"], cf.TSPBalances["dawn"] = currentTSPTraditionalRobert.Add(currentTSPRothRobert), currentTSPTraditionalDawn.Add(currentTSPRothDawn)
		cf.FederalTax, cf.StateTax, cf.LocalTax, cf.FICATax = federalTax, stateTax, localTax, ficaTax
		cf.FEHBPremium, cf.MedicarePremium = fehbPremium, medicarePremium
		cf.IsRetired = isRobertRetired && isDawnRetired
		cf.IsMedicareEligible = dateutil.IsMedicareEligible(robert.BirthDate, projectionDate) || dateutil.IsMedicareEligible(dawn.BirthDate, projectionDate)
		cf.IsRMDYear = dateutil.IsRMDYear(robert.BirthDate, projectionDate) || dateutil.IsRMDYear(dawn.BirthDate, projectionDate)
		cf.TotalTSPContributions = tspContributions
		cf.IsDeceased["robert"], cf.IsDeceased["dawn"] = robertIsDeceased, dawnIsDeceased

		// Filing status after a death
		if scenario.Mortality != nil && scenario.Mortality.Assumptions != nil && (robertIsDeceased != dawnIsDeceased) {
			mode := scenario.Mortality.Assumptions.FilingStatusSwitch
			switch mode {
			case "immediate":
				cf.FilingStatusSingle = true
			case "next_year":
				if robertDeathYearIndex != nil && robertIsDeceased && year > *robertDeathYearIndex {
					cf.FilingStatusSingle = true
				}
				if dawnDeathYearIndex != nil && dawnIsDeceased && year > *dawnDeathYearIndex {
					cf.FilingStatusSingle = true
				}
			}
		}

		// Survivor pensions (store in participant map)
		cf.SurvivorPensions["robert"] = survivorPensionRobert
		cf.SurvivorPensions["dawn"] = survivorPensionDawn

		// Apply survivor spending factor to withdrawals & pensions (not survivor annuity itself)
		if (robertIsDeceased || dawnIsDeceased) && survivorSpendingFactor.LessThan(decimal.NewFromFloat(0.999)) {
			cf.TSPWithdrawals["robert"] = cf.TSPWithdrawals["robert"].Mul(survivorSpendingFactor)
			cf.TSPWithdrawals["dawn"] = cf.TSPWithdrawals["dawn"].Mul(survivorSpendingFactor)
			cf.Pensions["robert"] = cf.Pensions["robert"].Mul(survivorSpendingFactor)
			cf.Pensions["dawn"] = cf.Pensions["dawn"].Mul(survivorSpendingFactor)
		}

		// Recalculate total gross income & net income after adjustments
		cf.TotalGrossIncome = cf.CalculateTotalIncome()
		cf.CalculateNetIncome()

		projection[year] = *cf
	}

	return projection
}

// GenerateAnnualProjectionGeneric generates annual cash flow projections for a household scenario using participant-based logic
func (ce *CalculationEngine) GenerateAnnualProjectionGeneric(household *domain.Household, scenario *domain.GenericScenario, assumptions *domain.GlobalAssumptions, federalRules domain.FederalRules) []domain.AnnualCashFlow {
	projection := make([]domain.AnnualCashFlow, assumptions.ProjectionYears)
	projectionStartYear := ProjectionBaseYear

	// Get participant names for initialization
	participantNames := make([]string, len(household.Participants))
	for i, p := range household.Participants {
		participantNames[i] = p.Name
	}

	// Initialize participant data structures
	currentTSPBalances := make(map[string]decimal.Decimal)
	tspStrategies := make(map[string]TSPWithdrawalStrategy)
	retirementYears := make(map[string]int)
	isRetiredMap := make(map[string]bool)
	isDeceasedMap := make(map[string]bool)
	deathYearIndexes := make(map[string]*int)

	// Initialize TSP balances and strategies for each participant
	for _, participant := range household.Participants {
		if participant.IsFederal {
			tspBalance := decimal.Zero
			if participant.TSPBalanceTraditional != nil {
				tspBalance = tspBalance.Add(*participant.TSPBalanceTraditional)
			}
			if participant.TSPBalanceRoth != nil {
				tspBalance = tspBalance.Add(*participant.TSPBalanceRoth)
			}
			currentTSPBalances[participant.Name] = tspBalance

			// Create TSP withdrawal strategy if participant scenario exists
			if participantScenario, exists := scenario.ParticipantScenarios[participant.Name]; exists {
				retirementScenario := domain.RetirementScenario{
					EmployeeName:               participant.Name,
					RetirementDate:             *participantScenario.RetirementDate,
					SSStartAge:                 participantScenario.SSStartAge,
					TSPWithdrawalStrategy:      participantScenario.TSPWithdrawalStrategy,
					TSPWithdrawalTargetMonthly: participantScenario.TSPWithdrawalTargetMonthly,
					TSPWithdrawalRate:          participantScenario.TSPWithdrawalRate,
				}
				tspStrategies[participant.Name] = ce.createTSPStrategy(&retirementScenario, tspBalance, assumptions.InflationRate)
				retirementYears[participant.Name] = participantScenario.RetirementDate.Year() - projectionStartYear
			}
		}

		isRetiredMap[participant.Name] = false
		isDeceasedMap[participant.Name] = false

		// Initialize death year indexes from mortality data
		if scenario.Mortality != nil && scenario.Mortality.Participants != nil {
			if mortalitySpec, exists := scenario.Mortality.Participants[participant.Name]; exists && mortalitySpec != nil {
				if mortalitySpec.DeathAge != nil {
					deathYear := participant.BirthDate.Year() + *mortalitySpec.DeathAge - projectionStartYear
					if deathYear >= 0 && deathYear < assumptions.ProjectionYears {
						deathYearIndexes[participant.Name] = &deathYear
					}
				} else if mortalitySpec.DeathDate != nil {
					deathYear := mortalitySpec.DeathDate.Year() - projectionStartYear
					if deathYear >= 0 && deathYear < assumptions.ProjectionYears {
						deathYearIndexes[participant.Name] = &deathYear
					}
				}
			}
		}
	}

	// Generate projection for each year
	for year := 0; year < assumptions.ProjectionYears; year++ {
		projectionDate := time.Date(projectionStartYear, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(year, 0, 0)

		// Create new cash flow for this year
		cashFlow := domain.NewAnnualCashFlow(year, projectionDate, participantNames)

		// Process death events at start of year
		for participantName, deathYearIndex := range deathYearIndexes {
			if deathYearIndex != nil && year >= *deathYearIndex {
				isDeceasedMap[participantName] = true
				cashFlow.IsDeceased[participantName] = true
			}
		}

		// Calculate each participant's data for this year
		for _, participant := range household.Participants {
			name := participant.Name
			age := participant.Age(projectionDate)
			cashFlow.Ages[name] = age

			// Skip processing if participant is deceased
			if isDeceasedMap[name] {
				continue
			}

			// Determine if participant is retired this year
			if participantScenario, exists := scenario.ParticipantScenarios[name]; exists && participantScenario.RetirementDate != nil {
				retirementYear := retirementYears[name]
				wasRetired := isRetiredMap[name]
				isRetiredThisYear := year >= retirementYear
				isRetiredMap[name] = isRetiredThisYear

				// Calculate work fraction for partial retirement year
				workFraction := decimal.NewFromInt(1)
				if year == retirementYear && retirementYear >= 0 {
					yearStart := time.Date(projectionDate.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
					daysWorked := participantScenario.RetirementDate.Sub(yearStart).Hours() / 24
					workFraction = decimal.NewFromFloat(daysWorked / 365.0)
				} else if isRetiredThisYear {
					workFraction = decimal.Zero
				}

				// Calculate salary for working portion
				if participant.IsFederal && participant.CurrentSalary != nil {
					cashFlow.Salaries[name] = participant.CurrentSalary.Mul(workFraction)
				}

				// Calculate FERS pension (only if retired and federal employee)
				if isRetiredThisYear && participant.IsFederal && !wasRetired {
					// Convert participant to employee for pension calculation
					if employee, err := participant.ToEmployee(); err == nil {
						pension := CalculatePensionForYear(employee, *participantScenario.RetirementDate, year-retirementYear, assumptions.InflationRate)
						// Adjust for partial year if retiring this year
						if year == retirementYear {
							pension = pension.Mul(decimal.NewFromInt(1).Sub(workFraction))
						}
						cashFlow.Pensions[name] = pension
					}
				} else if isRetiredThisYear && participant.IsFederal {
					// Continue existing pension with COLA adjustment
					if employee, err := participant.ToEmployee(); err == nil {
						pension := CalculatePensionForYear(employee, *participantScenario.RetirementDate, year-retirementYear, assumptions.InflationRate)
						cashFlow.Pensions[name] = pension
					}
				}

				// Calculate Social Security benefits
				if age >= participantScenario.SSStartAge {
					var ssBenefit decimal.Decimal
					switch participantScenario.SSStartAge {
					case 62:
						ssBenefit = participant.SSBenefit62.Mul(decimal.NewFromInt(12))
					case 67:
						ssBenefit = participant.SSBenefitFRA.Mul(decimal.NewFromInt(12))
					case 70:
						ssBenefit = participant.SSBenefit70.Mul(decimal.NewFromInt(12))
					default:
						// Use FRA as default
						ssBenefit = participant.SSBenefitFRA.Mul(decimal.NewFromInt(12))
					}
					cashFlow.SSBenefits[name] = ssBenefit
				}

				// Calculate TSP withdrawals
				if participant.IsFederal && isRetiredThisYear {
					if strategy, exists := tspStrategies[name]; exists {
						// Calculate target income based on spending needs
						targetIncome := decimal.Zero
						if participantScenario.TSPWithdrawalTargetMonthly != nil {
							targetIncome = participantScenario.TSPWithdrawalTargetMonthly.Mul(decimal.NewFromInt(12))
						}

						// Check if RMD year (age 73+)
						isRMDYear := age >= 73
						rmdAmount := decimal.Zero
						if isRMDYear {
							// Simplified RMD calculation - would need proper RMD tables
							rmdAmount = currentTSPBalances[name].Div(decimal.NewFromFloat(26.5)) // Approximate RMD factor
						}

						withdrawal := strategy.CalculateWithdrawal(
							currentTSPBalances[name],
							year-retirementYear+1, // Years since retirement
							targetIncome,
							age,
							isRMDYear,
							rmdAmount,
						)
						cashFlow.TSPWithdrawals[name] = withdrawal

						// Update TSP balance (simplified - just subtract withdrawal)
						newBalance := currentTSPBalances[name].Sub(withdrawal)
						if newBalance.LessThan(decimal.Zero) {
							newBalance = decimal.Zero
						}
						currentTSPBalances[name] = newBalance
						cashFlow.TSPBalances[name] = newBalance
					}
				} else if participant.IsFederal {
					// Not retired yet, just track balance growth
					if currentBalance, exists := currentTSPBalances[name]; exists {
						// Apply growth (simplified)
						growthRate := assumptions.TSPReturnPreRetirement
						if isRetiredThisYear {
							growthRate = assumptions.TSPReturnPostRetirement
						}
						newBalance := currentBalance.Mul(decimal.NewFromInt(1).Add(growthRate))
						currentTSPBalances[name] = newBalance
						cashFlow.TSPBalances[name] = newBalance
					}
				}
			}
		}

		// Calculate household-level totals and taxes
		cashFlow.TotalGrossIncome = cashFlow.GetTotalSalary().
			Add(cashFlow.GetTotalPension()).
			Add(cashFlow.GetTotalTSPWithdrawal()).
			Add(cashFlow.GetTotalSSBenefit()).
			Add(cashFlow.GetTotalFERSSupplement())

		// Determine if any participant is retired for household retirement status
		cashFlow.IsRetired = false
		for _, isRetired := range isRetiredMap {
			if isRetired {
				cashFlow.IsRetired = true
				break
			}
		}

		// Calculate taxes (simplified for now - would need full tax calculation integration)
		// For now, we'll use placeholder logic
		ages := make([]int, 0)
		for _, age := range cashFlow.Ages {
			if !cashFlow.IsDeceased[getParticipantNameForAge(cashFlow.Ages, age)] {
				ages = append(ages, age)
			}
		}

		if len(ages) > 0 {
			// Use first living participant's age for tax calculation (simplified)
			primaryAge := ages[0]
			secondaryAge := 0
			if len(ages) > 1 {
				secondaryAge = ages[1]
			}

			// Calculate taxes using existing tax calculator
			taxableIncome := CalculateCurrentTaxableIncome(cashFlow.GetTotalSalary(), decimal.Zero)
			federalTax, stateTax, localTax, _ := ce.TaxCalc.CalculateTotalTaxes(taxableIncome, cashFlow.IsRetired, primaryAge, secondaryAge, cashFlow.TotalGrossIncome)

			cashFlow.FederalTax = federalTax
			cashFlow.StateTax = stateTax
			cashFlow.LocalTax = localTax
		}

		// Calculate net income
		cashFlow.NetIncome = cashFlow.TotalGrossIncome.Sub(cashFlow.CalculateTotalDeductions())

		projection[year] = *cashFlow
	}

	return projection
}

// Helper function to get participant name for a given age (simplified)
func getParticipantNameForAge(ages map[string]int, targetAge int) string {
	for name, age := range ages {
		if age == targetAge {
			return name
		}
	}
	return ""
}
