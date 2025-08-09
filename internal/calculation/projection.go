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

	// Mortality derived dates (deterministic) if provided by age
	var robertDeathYearIndex *int
	var dawnDeathYearIndex *int
	if scenario.Mortality != nil {
		baseYear := ProjectionBaseYear
		if scenario.Mortality.Robert != nil {
			if scenario.Mortality.Robert.DeathDate != nil {
				y := scenario.Mortality.Robert.DeathDate.Year() - baseYear
				if y >= 0 && y < assumptions.ProjectionYears {
					robertDeathYearIndex = &y
				}
			} else if scenario.Mortality.Robert.DeathAge != nil {
				// compute calendar year when Robert reaches that age (approx: birth year + age)
				targetYear := robert.BirthDate.Year() + *scenario.Mortality.Robert.DeathAge
				y := targetYear - baseYear
				if y >= 0 && y < assumptions.ProjectionYears {
					robertDeathYearIndex = &y
				}
			}
		}
		if scenario.Mortality.Dawn != nil {
			if scenario.Mortality.Dawn.DeathDate != nil {
				y := scenario.Mortality.Dawn.DeathDate.Year() - baseYear
				if y >= 0 && y < assumptions.ProjectionYears {
					dawnDeathYearIndex = &y
				}
			} else if scenario.Mortality.Dawn.DeathAge != nil {
				targetYear := dawn.BirthDate.Year() + *scenario.Mortality.Dawn.DeathAge
				y := targetYear - baseYear
				if y >= 0 && y < assumptions.ProjectionYears {
					dawnDeathYearIndex = &y
				}
			}
		}
	}

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

		// Calculate Social Security benefits
		ssRobert := decimal.Zero
		if !robertIsDeceased {
			ssRobert = CalculateSSBenefitForYear(robert, scenario.Robert.SSStartAge, year, assumptions.COLAGeneralRate)
		}
		ssDawn := decimal.Zero
		if !dawnIsDeceased {
			ssDawn = CalculateSSBenefitForYear(dawn, scenario.Dawn.SSStartAge, year, assumptions.COLAGeneralRate)
		}
		// Survivor SS simplification: if one is deceased, survivor keeps max of their own vs deceased (Phase 1 naive annual)
		if robertIsDeceased && !dawnIsDeceased {
			if ssRobert.GreaterThan(ssDawn) {
				ssDawn = ssRobert
			}
		}
		if dawnIsDeceased && !robertIsDeceased {
			if ssDawn.GreaterThan(ssRobert) {
				ssRobert = ssDawn
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

		federalTax, stateTax, localTax, ficaTax := ce.calculateTaxes(
			robert, dawn, scenario, year, isRobertRetired && isDawnRetired, // Both retired for tax purposes
			pensionRobert, pensionDawn, tspWithdrawalRobert, tspWithdrawalDawn,
			ssRobert, ssDawn, assumptions,
			workingIncomeRobert, workingIncomeDawn, // Pass working income for transition years
		)

		// Calculate TSP contributions (only for working portion of year)
		var tspContributions decimal.Decimal
		if (!isRobertRetired || !isDawnRetired) && !(robertIsDeceased || dawnIsDeceased) {
			robertContributions := robert.TotalAnnualTSPContribution().Mul(robertWorkFraction)
			dawnContributions := dawn.TotalAnnualTSPContribution().Mul(dawnWorkFraction)
			tspContributions = robertContributions.Add(dawnContributions)
		}

		// Create annual cash flow
		cashFlow := domain.AnnualCashFlow{
			Year:                  year + 1,
			Date:                  projectionDate,
			AgeRobert:             ageRobert,
			AgeDawn:               ageDawn,
			SalaryRobert:          robert.CurrentSalary.Mul(robertWorkFraction),
			SalaryDawn:            dawn.CurrentSalary.Mul(dawnWorkFraction),
			PensionRobert:         pensionRobert,
			PensionDawn:           pensionDawn,
			TSPWithdrawalRobert:   tspWithdrawalRobert,
			TSPWithdrawalDawn:     tspWithdrawalDawn,
			SSBenefitRobert:       ssRobert,
			SSBenefitDawn:         ssDawn,
			FERSSupplementRobert:  srsRobert,
			FERSSupplementDawn:    srsDawn,
			FederalTax:            federalTax,
			StateTax:              stateTax,
			LocalTax:              localTax,
			FICATax:               ficaTax,
			TSPContributions:      tspContributions,
			FEHBPremium:           fehbPremium,
			MedicarePremium:       medicarePremium,
			TSPBalanceRobert:      currentTSPTraditionalRobert.Add(currentTSPRothRobert),
			TSPBalanceDawn:        currentTSPTraditionalDawn.Add(currentTSPRothDawn),
			TSPBalanceTraditional: currentTSPTraditionalRobert.Add(currentTSPTraditionalDawn),
			TSPBalanceRoth:        currentTSPRothRobert.Add(currentTSPRothDawn),
			IsRetired:             isRobertRetired && isDawnRetired, // Both retired
			IsMedicareEligible:    dateutil.IsMedicareEligible(robert.BirthDate, projectionDate) || dateutil.IsMedicareEligible(dawn.BirthDate, projectionDate),
			IsRMDYear:             dateutil.IsRMDYear(robert.BirthDate, projectionDate) || dateutil.IsRMDYear(dawn.BirthDate, projectionDate),
			RobertDeceased:        robertIsDeceased,
			DawnDeceased:          dawnIsDeceased,
			FilingStatusSingle:    false,
		}

		// Determine filing status for display (mirror simplified logic in taxes.go)
		if scenario.Mortality != nil && scenario.Mortality.Assumptions != nil && (robertIsDeceased != dawnIsDeceased) {
			mode := scenario.Mortality.Assumptions.FilingStatusSwitch
			// Reconstruct death year indexes (already computed earlier): reuse conditions
			if mode == "immediate" {
				cashFlow.FilingStatusSingle = true
			} else if mode == "next_year" {
				if robertDeathYearIndex != nil && robertIsDeceased && year > *robertDeathYearIndex {
					cashFlow.FilingStatusSingle = true
				}
				if dawnDeathYearIndex != nil && dawnIsDeceased && year > *dawnDeathYearIndex {
					cashFlow.FilingStatusSingle = true
				}
			}
		}

		// Apply survivor spending factor by scaling discretionary withdrawals (simple: scale TSP withdrawals post-death)
		if (robertIsDeceased || dawnIsDeceased) && survivorSpendingFactor.LessThan(decimal.NewFromFloat(0.999)) {
			cashFlow.TSPWithdrawalRobert = cashFlow.TSPWithdrawalRobert.Mul(survivorSpendingFactor)
			cashFlow.TSPWithdrawalDawn = cashFlow.TSPWithdrawalDawn.Mul(survivorSpendingFactor)
			cashFlow.PensionRobert = cashFlow.PensionRobert.Mul(survivorSpendingFactor)
			cashFlow.PensionDawn = cashFlow.PensionDawn.Mul(survivorSpendingFactor)
		}

		// Calculate total gross income and net income
		cashFlow.TotalGrossIncome = cashFlow.CalculateTotalIncome()
		cashFlow.CalculateNetIncome()

		projection[year] = cashFlow
	}

	return projection
}
