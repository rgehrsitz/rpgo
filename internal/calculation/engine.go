package calculation

import (
	"fmt"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/rpgo/retirement-calculator/pkg/dateutil"
	"github.com/shopspring/decimal"
)

// CalculationEngine orchestrates all retirement calculations
type CalculationEngine struct {
	TaxCalc *ComprehensiveTaxCalculator
}

// NewCalculationEngine creates a new calculation engine
func NewCalculationEngine() *CalculationEngine {
	return &CalculationEngine{
		TaxCalc: NewComprehensiveTaxCalculator(),
	}
}

// RunScenario calculates a complete retirement scenario
func (ce *CalculationEngine) RunScenario(config *domain.Configuration, scenario *domain.Scenario) (*domain.ScenarioSummary, error) {
	robert := config.PersonalDetails["robert"]
	dawn := config.PersonalDetails["dawn"]

	// Validate retirement dates are after hire dates
	if scenario.Robert.RetirementDate.Before(robert.HireDate) {
		return nil, fmt.Errorf("Robert's retirement date (%s) cannot be before hire date (%s)", 
			scenario.Robert.RetirementDate.Format("2006-01-02"), robert.HireDate.Format("2006-01-02"))
	}
	if scenario.Dawn.RetirementDate.Before(dawn.HireDate) {
		return nil, fmt.Errorf("Dawn's retirement date (%s) cannot be before hire date (%s)", 
			scenario.Dawn.RetirementDate.Format("2006-01-02"), dawn.HireDate.Format("2006-01-02"))
	}

	// Validate inflation and return rates are reasonable
	if config.GlobalAssumptions.InflationRate.LessThan(decimal.Zero) || config.GlobalAssumptions.InflationRate.GreaterThan(decimal.NewFromFloat(0.20)) {
		return nil, fmt.Errorf("inflation rate must be between 0%% and 20%%, got %s%%", 
			config.GlobalAssumptions.InflationRate.Mul(decimal.NewFromInt(100)).StringFixed(2))
	}

	// Generate annual projections
	projection := ce.GenerateAnnualProjection(&robert, &dawn, scenario, &config.GlobalAssumptions)

	// Create scenario summary
	summary := &domain.ScenarioSummary{
		Name:               scenario.Name,
		FirstYearNetIncome: projection[0].NetIncome,
		Year5NetIncome:     projection[4].NetIncome,
		Year10NetIncome:    projection[9].NetIncome,
		Projection:         projection,
	}

	// Calculate total lifetime income (present value)
	var totalPV decimal.Decimal
	discountRate := decimal.NewFromFloat(0.03) // 3% discount rate
	for i, year := range projection {
		discountFactor := decimal.NewFromFloat(1).Add(discountRate).Pow(decimal.NewFromInt(int64(i)))
		totalPV = totalPV.Add(year.NetIncome.Div(discountFactor))
	}
	summary.TotalLifetimeIncome = totalPV

	// Determine TSP longevity
	for i, year := range projection {
		if year.IsTSPDepleted() {
			summary.TSPLongevity = i + 1
			break
		}
	}
	if summary.TSPLongevity == 0 {
		summary.TSPLongevity = len(projection) // Lasted full projection
	}

	// Set initial and final TSP balances
	if len(projection) > 0 {
		summary.InitialTSPBalance = projection[0].TSPBalanceRobert.Add(projection[0].TSPBalanceDawn)
		summary.FinalTSPBalance = projection[len(projection)-1].TSPBalanceRobert.Add(projection[len(projection)-1].TSPBalanceDawn)
	}

	return summary, nil
}

// GenerateAnnualProjection generates annual cash flow projections for a scenario
func (ce *CalculationEngine) GenerateAnnualProjection(robert, dawn *domain.Employee, scenario *domain.Scenario, assumptions *domain.GlobalAssumptions) []domain.AnnualCashFlow {
	projection := make([]domain.AnnualCashFlow, assumptions.ProjectionYears)

	// Determine retirement year (0-based index)
	// Projection starts in 2025 (first year of projection)
	projectionStartYear := 2025
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

	for year := 0; year < assumptions.ProjectionYears; year++ {
		projectionDate := time.Date(projectionStartYear, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(year, 0, 0)
		ageRobert := robert.Age(projectionDate)
		ageDawn := dawn.Age(projectionDate)

		// Calculate partial year retirement for each person
		// Projection starts in 2025, so year 0 = 2025, year 1 = 2026, etc.
		projectionStartYear := 2025
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

		// Calculate FERS pensions (only for retired portion of year)
		var pensionRobert, pensionDawn decimal.Decimal
		if isRobertRetired {
			pensionRobert = ce.calculatePensionForYear(robert, scenario.Robert.RetirementDate, year-robertRetirementYear, assumptions.InflationRate)
			// Adjust for partial year if retiring this year
			if year == robertRetirementYear {
				pensionRobert = pensionRobert.Mul(decimal.NewFromInt(1).Sub(robertWorkFraction))
			}

			// Debug output for pension calculation
			if year == robertRetirementYear {
				fmt.Printf("DEBUG: Robert's pension calculation for year %d:\n", 2025+year)
				fmt.Printf("  Retirement date: %s\n", scenario.Robert.RetirementDate.Format("2006-01-02"))
				fmt.Printf("  Age at retirement: %d\n", robert.Age(scenario.Robert.RetirementDate))
				fmt.Printf("  Years of service: %s\n", robert.YearsOfService(scenario.Robert.RetirementDate).StringFixed(2))
				fmt.Printf("  High-3 salary: %s\n", robert.High3Salary.StringFixed(2))

				// Get detailed pension calculation
				pensionCalc := CalculateFERSPension(robert, scenario.Robert.RetirementDate)
				fmt.Printf("  Multiplier: %s\n", pensionCalc.Multiplier.StringFixed(4))
				fmt.Printf("  Annual pension (before reduction): %s\n", pensionCalc.AnnualPension.StringFixed(2))
				fmt.Printf("  Survivor election: %s\n", pensionCalc.SurvivorElection.StringFixed(4))
				fmt.Printf("  Final pension: %s\n", pensionCalc.ReducedPension.StringFixed(2))
				fmt.Printf("  Monthly pension amount: %s\n", pensionCalc.ReducedPension.Div(decimal.NewFromInt(12)).StringFixed(2))
				fmt.Println()
			}
		}
		if isDawnRetired {
			pensionDawn = ce.calculatePensionForYear(dawn, scenario.Dawn.RetirementDate, year-dawnRetirementYear, assumptions.InflationRate)
			// Adjust for partial year if retiring this year
			if year == dawnRetirementYear {
				pensionDawn = pensionDawn.Mul(decimal.NewFromInt(1).Sub(dawnWorkFraction))
			}
		}

		// Calculate Social Security benefits
		ssRobert := ce.calculateSSBenefitForYear(robert, scenario.Robert.SSStartAge, year, assumptions.COLAGeneralRate)
		ssDawn := ce.calculateSSBenefitForYear(dawn, scenario.Dawn.SSStartAge, year, assumptions.COLAGeneralRate)

		// Adjust Social Security for partial year if retiring this year
		if year == robertRetirementYear && robertRetirementYear >= 0 {
			// Robert retires during this year - adjust SS for partial year
			ssRobert = ssRobert.Mul(decimal.NewFromInt(1).Sub(robertWorkFraction))
		}
		if year == dawnRetirementYear && dawnRetirementYear >= 0 {
			// Dawn retires during this year - adjust SS for partial year
			ssDawn = ssDawn.Mul(decimal.NewFromInt(1).Sub(dawnWorkFraction))
		}

		// Calculate FERS Special Retirement Supplement (only if retired)
		var srsRobert, srsDawn decimal.Decimal
		if isRobertRetired {
			srsRobert = ce.calculateFERSSupplement(robert, scenario.Robert.RetirementDate, year-robertRetirementYear, assumptions.InflationRate)
			// Adjust for partial year if retiring this year
			if year == robertRetirementYear {
				srsRobert = srsRobert.Mul(decimal.NewFromInt(1).Sub(robertWorkFraction))
			}
		}
		if isDawnRetired {
			srsDawn = ce.calculateFERSSupplement(dawn, scenario.Dawn.RetirementDate, year-dawnRetirementYear, assumptions.InflationRate)
			// Adjust for partial year if retiring this year
			if year == dawnRetirementYear {
				srsDawn = srsDawn.Mul(decimal.NewFromInt(1).Sub(dawnWorkFraction))
			}
		}

		// Calculate TSP withdrawals and update balances
		var tspWithdrawalRobert, tspWithdrawalDawn decimal.Decimal
		if isRobertRetired {
			// For 4% rule: Always withdraw 4% of initial balance (adjusted for inflation)
			if scenario.Robert.TSPWithdrawalStrategy == "4_percent_rule" {
				// Use the 4% rule strategy to calculate withdrawals
				tspWithdrawalRobert = robertStrategy.CalculateWithdrawal(
					currentTSPTraditionalRobert.Add(currentTSPRothRobert),
					year-robertRetirementYear+1,
					decimal.Zero, // Not used for 4% rule
					ageRobert,
					dateutil.IsRMDYear(robert.BirthDate, projectionDate),
					ce.calculateRMD(currentTSPTraditionalRobert, robert.BirthDate.Year(), ageRobert),
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
					ce.calculateRMD(currentTSPTraditionalRobert, robert.BirthDate.Year(), ageRobert),
				)
				// Adjust for partial year if retiring this year
				if year == robertRetirementYear {
					tspWithdrawalRobert = tspWithdrawalRobert.Mul(decimal.NewFromInt(1).Sub(robertWorkFraction))
				}
			}
		}

		if isDawnRetired {
			if scenario.Dawn.TSPWithdrawalStrategy == "4_percent_rule" {
				tspWithdrawalDawn = dawnStrategy.CalculateWithdrawal(
					currentTSPTraditionalDawn.Add(currentTSPRothDawn),
					year-dawnRetirementYear+1,
					decimal.Zero, // Not used for 4% rule
					ageDawn,
					dateutil.IsRMDYear(dawn.BirthDate, projectionDate),
					ce.calculateRMD(currentTSPTraditionalDawn, dawn.BirthDate.Year(), ageDawn),
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
					ce.calculateRMD(currentTSPTraditionalDawn, dawn.BirthDate.Year(), ageDawn),
				)
				// Adjust for partial year if retiring this year
				if year == dawnRetirementYear {
					tspWithdrawalDawn = tspWithdrawalDawn.Mul(decimal.NewFromInt(1).Sub(dawnWorkFraction))
				}
			}
		}

		// Update TSP balances
		if isRobertRetired {
			currentTSPTraditionalRobert, currentTSPRothRobert = ce.updateTSPBalances(
				currentTSPTraditionalRobert, currentTSPRothRobert, tspWithdrawalRobert,
				assumptions.TSPReturnPostRetirement,
			)
		} else {
			// Pre-retirement TSP growth with contributions
			currentTSPTraditionalRobert = ce.growTSPBalance(currentTSPTraditionalRobert, robert.TotalAnnualTSPContribution(), assumptions.TSPReturnPreRetirement)
			currentTSPRothRobert = ce.growTSPBalance(currentTSPRothRobert, decimal.Zero, assumptions.TSPReturnPreRetirement)
		}

		if isDawnRetired {
			currentTSPTraditionalDawn, currentTSPRothDawn = ce.updateTSPBalances(
				currentTSPTraditionalDawn, currentTSPRothDawn, tspWithdrawalDawn,
				assumptions.TSPReturnPostRetirement,
			)
		} else {
			// Pre-retirement TSP growth with contributions
			currentTSPTraditionalDawn = ce.growTSPBalance(currentTSPTraditionalDawn, dawn.TotalAnnualTSPContribution(), assumptions.TSPReturnPreRetirement)
			currentTSPRothDawn = ce.growTSPBalance(currentTSPRothDawn, decimal.Zero, assumptions.TSPReturnPreRetirement)
		}

		// Debug TSP balances for Scenario 2 to show extra growth
		if year == 1 && scenario.Robert.RetirementDate.Year() == 2027 {
			fmt.Printf("DEBUG: TSP Growth in Scenario 2 (year %d):\n", 2025+year)
			fmt.Printf("  Robert's TSP balance: %s\n", currentTSPTraditionalRobert.Add(currentTSPRothRobert).StringFixed(2))
			fmt.Printf("  Dawn's TSP balance: %s\n", currentTSPTraditionalDawn.Add(currentTSPRothDawn).StringFixed(2))
			fmt.Printf("  Combined TSP balance: %s\n", currentTSPTraditionalRobert.Add(currentTSPRothRobert).Add(currentTSPTraditionalDawn).Add(currentTSPRothDawn).StringFixed(2))
			fmt.Println()
		}

		// Calculate FEHB premiums
		fehbPremium := ce.calculateFEHBPremium(robert, year, dateutil.IsMedicareEligible(robert.BirthDate, projectionDate), assumptions.FEHBPremiumInflation)

		// Calculate Medicare premiums (if applicable)
		medicarePremium := ce.calculateMedicarePremium(robert, dawn, projectionDate, assumptions)

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
		if !isRobertRetired || !isDawnRetired {
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
		}

		// Calculate total gross income and net income
		cashFlow.TotalGrossIncome = cashFlow.CalculateTotalIncome()
		cashFlow.CalculateNetIncome()

		projection[year] = cashFlow
	}

	return projection
}

// createTSPStrategy creates a TSP withdrawal strategy based on scenario configuration
func (ce *CalculationEngine) createTSPStrategy(scenario *domain.RetirementScenario, initialBalance decimal.Decimal, inflationRate decimal.Decimal) TSPWithdrawalStrategy {
	switch scenario.TSPWithdrawalStrategy {
	case "4_percent_rule":
		return NewFourPercentRule(initialBalance, inflationRate)
	case "need_based":
		if scenario.TSPWithdrawalTargetMonthly != nil {
			return NewNeedBasedWithdrawal(*scenario.TSPWithdrawalTargetMonthly)
		}
		// Fallback to 4% rule if target not specified
		return NewFourPercentRule(initialBalance, inflationRate)
	default:
		// Default to 4% rule
		return NewFourPercentRule(initialBalance, inflationRate)
	}
}

// calculatePensionForYear calculates the FERS pension for a specific year
func (ce *CalculationEngine) calculatePensionForYear(employee *domain.Employee, retirementDate time.Time, year int, inflationRate decimal.Decimal) decimal.Decimal {
	if year < 0 {
		return decimal.Zero
	}
	return CalculatePensionForYear(employee, retirementDate, year, inflationRate)
}

// calculateSSBenefitForYear calculates the Social Security benefit for a specific year
func (ce *CalculationEngine) calculateSSBenefitForYear(employee *domain.Employee, ssStartAge int, year int, colaRate decimal.Decimal) decimal.Decimal {
	return CalculateSSBenefitForYear(employee, ssStartAge, year, colaRate)
}

// calculateFERSSupplement calculates the FERS Special Retirement Supplement
func (ce *CalculationEngine) calculateFERSSupplement(employee *domain.Employee, retirementDate time.Time, yearsSinceRetirement int, inflationRate decimal.Decimal) decimal.Decimal {
	if yearsSinceRetirement < 0 {
		return decimal.Zero
	}

	// Calculate the projection date (years since retirement)
	projectionDate := retirementDate.AddDate(yearsSinceRetirement, 0, 0)
	age := employee.Age(projectionDate)

	if age >= 62 {
		return decimal.Zero // SRS stops at age 62
	}

	// Calculate SRS
	serviceYears := employee.YearsOfService(retirementDate)
	srs := CalculateFERSSpecialRetirementSupplement(employee.SSBenefit62, serviceYears, age)

	// Apply inflation adjustments for each year since retirement
	for y := 0; y < yearsSinceRetirement; y++ {
		srs = srs.Mul(decimal.NewFromFloat(1).Add(inflationRate))
	}

	return srs
}

// updateTSPBalances updates TSP balances after withdrawal
func (ce *CalculationEngine) updateTSPBalances(traditional, roth, withdrawal, returnRate decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	// Apply growth first
	traditional = traditional.Mul(decimal.NewFromFloat(1).Add(returnRate))
	roth = roth.Mul(decimal.NewFromFloat(1).Add(returnRate))

	// Withdraw from Roth first, then traditional
	if withdrawal.LessThanOrEqual(roth) {
		roth = roth.Sub(withdrawal)
	} else {
		remainingWithdrawal := withdrawal.Sub(roth)
		roth = decimal.Zero
		traditional = traditional.Sub(remainingWithdrawal)
		if traditional.LessThan(decimal.Zero) {
			traditional = decimal.Zero
		}
	}

	// Ensure balances never go negative
	if traditional.LessThan(decimal.Zero) {
		traditional = decimal.Zero
	}
	if roth.LessThan(decimal.Zero) {
		roth = decimal.Zero
	}

	return traditional, roth
}

// growTSPBalance grows a TSP balance with contributions and returns
func (ce *CalculationEngine) growTSPBalance(balance, contribution, returnRate decimal.Decimal) decimal.Decimal {
	return balance.Add(contribution).Mul(decimal.NewFromFloat(1).Add(returnRate))
}

// calculateFEHBPremium calculates FEHB premium for a given year
func (ce *CalculationEngine) calculateFEHBPremium(employee *domain.Employee, year int, _ bool, premiumInflation decimal.Decimal) decimal.Decimal {
	inflationFactor := decimal.NewFromFloat(1).Add(premiumInflation)
	adjustedPremium := employee.FEHBPremiumMonthly.Mul(inflationFactor.Pow(decimal.NewFromInt(int64(year))))
	return adjustedPremium.Mul(decimal.NewFromInt(26)) // 26 pay periods per year
}

// calculateMedicarePremium calculates Medicare premiums (simplified)
func (ce *CalculationEngine) calculateMedicarePremium(robert, dawn *domain.Employee, projectionDate time.Time, _ *domain.GlobalAssumptions) decimal.Decimal {
	// Simplified Medicare calculation - could be enhanced with IRMAA tiers
	if dateutil.IsMedicareEligible(robert.BirthDate, projectionDate) || dateutil.IsMedicareEligible(dawn.BirthDate, projectionDate) {
		// Basic Medicare Part B premium (simplified)
		return decimal.NewFromInt(174).Mul(decimal.NewFromInt(12)) // Monthly premium * 12
	}
	return decimal.Zero
}

// calculateRMD calculates Required Minimum Distribution
func (ce *CalculationEngine) calculateRMD(balance decimal.Decimal, birthYear, age int) decimal.Decimal {
	rmdCalc := NewRMDCalculator(birthYear)
	return rmdCalc.CalculateRMD(balance, age)
}

// calculateTaxes calculates all applicable taxes
func (ce *CalculationEngine) calculateTaxes(robert, dawn *domain.Employee, _ *domain.Scenario, year int, isRetired bool, pensionRobert, pensionDawn, tspWithdrawalRobert, tspWithdrawalDawn, ssRobert, ssDawn decimal.Decimal, _ *domain.GlobalAssumptions, workingIncomeRobert, workingIncomeDawn decimal.Decimal) (decimal.Decimal, decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	projectionStartYear := 2025
	projectionDate := time.Date(projectionStartYear, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(year, 0, 0)
	ageRobert := robert.Age(projectionDate)
	ageDawn := dawn.Age(projectionDate)

	// Check if this is a transition year (has both working and retirement income)
	isTransitionYear := (workingIncomeRobert.GreaterThan(decimal.Zero) || workingIncomeDawn.GreaterThan(decimal.Zero)) &&
		(pensionRobert.GreaterThan(decimal.Zero) || pensionDawn.GreaterThan(decimal.Zero) || tspWithdrawalRobert.GreaterThan(decimal.Zero) || tspWithdrawalDawn.GreaterThan(decimal.Zero) || ssRobert.GreaterThan(decimal.Zero) || ssDawn.GreaterThan(decimal.Zero))

	if isTransitionYear {
		// Transition year: combine working and retirement income
		totalWorkingIncome := workingIncomeRobert.Add(workingIncomeDawn)
		totalRetirementIncome := pensionRobert.Add(pensionDawn).Add(tspWithdrawalRobert).Add(tspWithdrawalDawn)

		// Calculate Social Security taxation
		totalSSBenefits := ssRobert.Add(ssDawn)
		taxableSS := ce.TaxCalc.CalculateSocialSecurityTaxation(totalSSBenefits, totalRetirementIncome)

		// Create taxable income structure for transition year
		taxableIncome := domain.TaxableIncome{
			Salary:             totalWorkingIncome,
			FERSPension:        pensionRobert.Add(pensionDawn),
			TSPWithdrawalsTrad: tspWithdrawalRobert.Add(tspWithdrawalDawn),
			TaxableSSBenefits:  taxableSS,
			OtherTaxableIncome: decimal.Zero,
			WageIncome:         totalWorkingIncome,
			InterestIncome:     decimal.Zero,
		}

		// Calculate taxes for transition year (FICA only on working income, with proration)
		federalTax, stateTax, localTax, _ := ce.TaxCalc.CalculateTotalTaxes(taxableIncome, false, ageRobert, ageDawn, totalWorkingIncome)

		// Calculate FICA with proration for transition year
		// Calculate work fractions for FICA proration
		robertWorkFraction := workingIncomeRobert.Div(robert.CurrentSalary)
		dawnWorkFraction := workingIncomeDawn.Div(dawn.CurrentSalary)

		// Calculate FICA for each person separately with proration
		robertFICA := ce.TaxCalc.FICATaxCalc.CalculateFICAWithProration(robert.CurrentSalary, totalWorkingIncome, robertWorkFraction)
		dawnFICA := ce.TaxCalc.FICATaxCalc.CalculateFICAWithProration(dawn.CurrentSalary, totalWorkingIncome, dawnWorkFraction)
		ficaTax := robertFICA.Add(dawnFICA)

		return federalTax, stateTax, localTax, ficaTax
	} else if isRetired {
		// Fully retired year
		// Calculate other income (excluding Social Security)
		otherIncome := pensionRobert.Add(pensionDawn).Add(tspWithdrawalRobert).Add(tspWithdrawalDawn)

		// Calculate Social Security taxation
		totalSSBenefits := ssRobert.Add(ssDawn)
		taxableSS := ce.TaxCalc.CalculateSocialSecurityTaxation(totalSSBenefits, otherIncome)

		// Create taxable income structure
		taxableIncome := domain.TaxableIncome{
			Salary:             decimal.Zero, // No salary in retirement
			FERSPension:        pensionRobert.Add(pensionDawn),
			TSPWithdrawalsTrad: tspWithdrawalRobert.Add(tspWithdrawalDawn), // Assuming all TSP withdrawals are from traditional
			TaxableSSBenefits:  taxableSS,
			OtherTaxableIncome: decimal.Zero,
			WageIncome:         decimal.Zero,
			InterestIncome:     decimal.Zero,
		}

		// Calculate taxes (no FICA in retirement)
		federalTax, stateTax, localTax, _ := ce.TaxCalc.CalculateTotalTaxes(taxableIncome, isRetired, ageRobert, ageDawn, decimal.Zero)

		return federalTax, stateTax, localTax, decimal.Zero
	} else {
		// Pre-retirement: calculate current working income
		currentTaxableIncome := CalculateCurrentTaxableIncome(robert.CurrentSalary, dawn.CurrentSalary)
		federalTax, stateTax, localTax, ficaTax := ce.TaxCalc.CalculateTotalTaxes(currentTaxableIncome, isRetired, ageRobert, ageDawn, robert.CurrentSalary.Add(dawn.CurrentSalary))

		return federalTax, stateTax, localTax, ficaTax
	}
}

// RunScenarios runs all scenarios and returns a comparison
func (ce *CalculationEngine) RunScenarios(config *domain.Configuration) (*domain.ScenarioComparison, error) {
	scenarios := make([]domain.ScenarioSummary, len(config.Scenarios))

	for i, scenario := range config.Scenarios {
		summary, err := ce.RunScenario(config, &scenario)
		if err != nil {
			return nil, err
		}
		scenarios[i] = *summary
	}

	// Calculate baseline (current net income)
	robert := config.PersonalDetails["robert"]
	dawn := config.PersonalDetails["dawn"]
	baselineNetIncome := ce.calculateCurrentNetIncome(&robert, &dawn, &config.GlobalAssumptions)

	comparison := &domain.ScenarioComparison{
		BaselineNetIncome: baselineNetIncome,
		Scenarios:         scenarios,
	}

	// Generate impact analysis
	comparison.ImmediateImpact = ce.generateImpactAnalysis(baselineNetIncome, scenarios)
	comparison.LongTermProjection = ce.generateLongTermAnalysis(scenarios)

	return comparison, nil
}

// calculateCurrentNetIncome calculates current net income
func (ce *CalculationEngine) calculateCurrentNetIncome(robert, dawn *domain.Employee, _ *domain.GlobalAssumptions) decimal.Decimal {
	// Calculate gross income
	grossIncome := robert.CurrentSalary.Add(dawn.CurrentSalary)

	// Calculate FEHB premiums (only Robert pays FEHB, Dawn has FSA-HC)
	fehbPremium := robert.FEHBPremiumMonthly.Mul(decimal.NewFromInt(26)) // 26 pay periods per year

	// Calculate TSP contributions (pre-tax)
	tspContributions := robert.TotalAnnualTSPContribution().Add(dawn.TotalAnnualTSPContribution())

	// Calculate taxes - use projection start date for age calculation
	projectionStartYear := 2025
	projectionStartDate := time.Date(projectionStartYear, 1, 1, 0, 0, 0, 0, time.UTC)
	ageRobert := robert.Age(projectionStartDate)
	ageDawn := dawn.Age(projectionStartDate)

	currentTaxableIncome := CalculateCurrentTaxableIncome(robert.CurrentSalary, dawn.CurrentSalary)
	federalTax, stateTax, localTax, ficaTax := ce.TaxCalc.CalculateTotalTaxes(currentTaxableIncome, false, ageRobert, ageDawn, grossIncome)

	// Calculate net income: gross - taxes - FEHB - TSP contributions
	netIncome := grossIncome.Sub(federalTax).Sub(stateTax).Sub(localTax).Sub(ficaTax).Sub(fehbPremium).Sub(tspContributions)


	// Debug output for verification
	fmt.Println("CURRENT NET INCOME CALCULATION BREAKDOWN:")
	fmt.Println("=========================================")
	fmt.Printf("Robert's Salary:        $%s\n", robert.CurrentSalary.StringFixed(2))
	fmt.Printf("Dawn's Salary:          $%s\n", dawn.CurrentSalary.StringFixed(2))
	fmt.Printf("Combined Gross Income:  $%s\n", grossIncome.StringFixed(2))
	fmt.Println()
	fmt.Printf("DEDUCTIONS:\n")
	fmt.Printf("  Federal Tax:          $%s\n", federalTax.StringFixed(2))
	fmt.Printf("  State Tax:            $%s\n", stateTax.StringFixed(2))
	fmt.Printf("  Local Tax:            $%s\n", localTax.StringFixed(2))
	fmt.Printf("  FICA Tax:             $%s\n", ficaTax.StringFixed(2))
	fmt.Printf("  FEHB Premium (Robert): $%s\n", fehbPremium.StringFixed(2))
	fmt.Printf("  TSP Contributions:    $%s\n", tspContributions.StringFixed(2))
	fmt.Printf("  Total Deductions:     $%s\n", federalTax.Add(stateTax).Add(localTax).Add(ficaTax).Add(fehbPremium).Add(tspContributions).StringFixed(2))
	fmt.Println()
	fmt.Printf("CURRENT NET TAKE-HOME:  $%s\n", netIncome.StringFixed(2))
	fmt.Printf("Monthly Take-Home:      $%s\n", netIncome.Div(decimal.NewFromInt(12)).StringFixed(2))
	fmt.Println()

	return netIncome
}

// generateImpactAnalysis generates impact analysis for scenarios
func (ce *CalculationEngine) generateImpactAnalysis(baselineNetIncome decimal.Decimal, scenarios []domain.ScenarioSummary) domain.ImpactAnalysis {
	var bestScenario string
	var bestRetirementIncome decimal.Decimal

	// Use baseline net income as-is (true take-home after all deductions including TSP)
	currentTakeHome := baselineNetIncome

	for _, scenario := range scenarios {
		// Compare retirement net income directly to current take-home
		scenarioNetIncome := scenario.FirstYearNetIncome
		if scenarioNetIncome.GreaterThan(bestRetirementIncome) {
			bestRetirementIncome = scenarioNetIncome
			bestScenario = scenario.Name
		}
	}

	// Calculate net income change for the recommendation
	netIncomeChange := bestRetirementIncome.Sub(currentTakeHome)
	percentageChange := netIncomeChange.Div(currentTakeHome).Mul(decimal.NewFromInt(100))
	monthlyChange := netIncomeChange.Div(decimal.NewFromInt(12))

	return domain.ImpactAnalysis{
		CurrentToFirstYear: domain.IncomeChange{
			ScenarioName:     bestScenario,
			NetIncomeChange:  netIncomeChange,
			PercentageChange: percentageChange,
			MonthlyChange:    monthlyChange,
		},
		RecommendedScenario: bestScenario,
		KeyConsiderations:   []string{"Consider healthcare costs", "Evaluate TSP withdrawal strategy", "Review Social Security timing"},
	}
}

// generateLongTermAnalysis generates long-term analysis
func (ce *CalculationEngine) generateLongTermAnalysis(scenarios []domain.ScenarioSummary) domain.LongTermAnalysis {
	var bestIncomeScenario, bestLongevityScenario string
	var bestIncome, bestLongevity decimal.Decimal

	for _, scenario := range scenarios {
		if scenario.TotalLifetimeIncome.GreaterThan(bestIncome) {
			bestIncome = scenario.TotalLifetimeIncome
			bestIncomeScenario = scenario.Name
		}
		if decimal.NewFromInt(int64(scenario.TSPLongevity)).GreaterThan(bestLongevity) {
			bestLongevity = decimal.NewFromInt(int64(scenario.TSPLongevity))
			bestLongevityScenario = scenario.Name
		}
	}

	return domain.LongTermAnalysis{
		BestScenarioForIncome:    bestIncomeScenario,
		BestScenarioForLongevity: bestLongevityScenario,
		RiskAssessment:           "Consider market volatility and inflation risks",
		Recommendations:          []string{"Diversify TSP allocations", "Monitor withdrawal rates", "Plan for healthcare costs"},
	}
}
