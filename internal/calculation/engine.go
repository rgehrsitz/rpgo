package calculation

import (
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/rpgo/retirement-calculator/pkg/dateutil"
	"github.com/shopspring/decimal"
)

// CalculationEngine orchestrates all retirement calculations
type CalculationEngine struct {
	TaxCalc       *ComprehensiveTaxCalculator
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
	
	// Determine retirement year
	retirementYear := scenario.Robert.RetirementDate.Year() - time.Now().Year()
	if retirementYear < 0 {
		retirementYear = 0
	}
	
	// Initialize TSP balances
	currentTSPTraditionalRobert := robert.TSPBalanceTraditional
	currentTSPRothRobert := robert.TSPBalanceRoth
	currentTSPTraditionalDawn := dawn.TSPBalanceTraditional
	currentTSPRothDawn := dawn.TSPBalanceRoth
	
	// Create TSP withdrawal strategies
	robertStrategy := ce.createTSPStrategy(&scenario.Robert, currentTSPTraditionalRobert.Add(currentTSPRothRobert), assumptions.InflationRate)
	dawnStrategy := ce.createTSPStrategy(&scenario.Dawn, currentTSPTraditionalDawn.Add(currentTSPRothDawn), assumptions.InflationRate)
	
	for year := 0; year < assumptions.ProjectionYears; year++ {
		projectionDate := time.Now().AddDate(year, 0, 0)
		ageRobert := robert.Age(projectionDate)
		ageDawn := dawn.Age(projectionDate)
		
		// Determine if retired
		isRetired := year >= retirementYear
		
		// Calculate FERS pensions
		var pensionRobert, pensionDawn decimal.Decimal
		if isRetired {
			pensionRobert = ce.calculatePensionForYear(robert, scenario.Robert.RetirementDate, year-retirementYear, assumptions.InflationRate)
			pensionDawn = ce.calculatePensionForYear(dawn, scenario.Dawn.RetirementDate, year-retirementYear, assumptions.InflationRate)
		}
		
		// Calculate Social Security benefits
		ssRobert := ce.calculateSSBenefitForYear(robert, scenario.Robert.SSStartAge, year, assumptions.COLAGeneralRate)
		ssDawn := ce.calculateSSBenefitForYear(dawn, scenario.Dawn.SSStartAge, year, assumptions.COLAGeneralRate)
		
		// Calculate FERS Special Retirement Supplement
		srsRobert := ce.calculateFERSSupplement(robert, scenario.Robert.RetirementDate, year, assumptions.InflationRate)
		srsDawn := ce.calculateFERSSupplement(dawn, scenario.Dawn.RetirementDate, year, assumptions.InflationRate)
		
		// Calculate TSP withdrawals and update balances
		var tspWithdrawalRobert, tspWithdrawalDawn decimal.Decimal
		if isRetired {
			// Calculate target income for need-based withdrawal
			targetIncome := pensionRobert.Add(pensionDawn).Add(ssRobert).Add(ssDawn).Add(srsRobert).Add(srsDawn)
			
			// Calculate withdrawals
			tspWithdrawalRobert = robertStrategy.CalculateWithdrawal(
				currentTSPTraditionalRobert.Add(currentTSPRothRobert),
				year-retirementYear+1,
				targetIncome,
				ageRobert,
				dateutil.IsRMDYear(robert.BirthDate, projectionDate),
				ce.calculateRMD(currentTSPTraditionalRobert, robert.BirthDate.Year(), ageRobert),
			)
			
			tspWithdrawalDawn = dawnStrategy.CalculateWithdrawal(
				currentTSPTraditionalDawn.Add(currentTSPRothDawn),
				year-retirementYear+1,
				targetIncome,
				ageDawn,
				dateutil.IsRMDYear(dawn.BirthDate, projectionDate),
				ce.calculateRMD(currentTSPTraditionalDawn, dawn.BirthDate.Year(), ageDawn),
			)
			
			// Update TSP balances
			currentTSPTraditionalRobert, currentTSPRothRobert = ce.updateTSPBalances(
				currentTSPTraditionalRobert, currentTSPRothRobert, tspWithdrawalRobert,
				assumptions.TSPReturnPostRetirement,
			)
			currentTSPTraditionalDawn, currentTSPRothDawn = ce.updateTSPBalances(
				currentTSPTraditionalDawn, currentTSPRothDawn, tspWithdrawalDawn,
				assumptions.TSPReturnPostRetirement,
			)
		} else {
			// Pre-retirement TSP growth
			currentTSPTraditionalRobert = ce.growTSPBalance(currentTSPTraditionalRobert, robert.TotalAnnualTSPContribution(), assumptions.TSPReturnPreRetirement)
			currentTSPRothRobert = ce.growTSPBalance(currentTSPRothRobert, decimal.Zero, assumptions.TSPReturnPreRetirement)
			currentTSPTraditionalDawn = ce.growTSPBalance(currentTSPTraditionalDawn, dawn.TotalAnnualTSPContribution(), assumptions.TSPReturnPreRetirement)
			currentTSPRothDawn = ce.growTSPBalance(currentTSPRothDawn, decimal.Zero, assumptions.TSPReturnPreRetirement)
		}
		
		// Calculate FEHB premiums
		fehbPremium := ce.calculateFEHBPremium(robert, year, dateutil.IsMedicareEligible(robert.BirthDate, projectionDate), assumptions.FEHBPremiumInflation)
		
		// Calculate Medicare premiums (if applicable)
		medicarePremium := ce.calculateMedicarePremium(robert, dawn, projectionDate, assumptions)
		
		// Calculate taxes
		federalTax, stateTax, localTax := ce.calculateTaxes(
			robert, dawn, scenario, year, isRetired,
			pensionRobert, pensionDawn, tspWithdrawalRobert, tspWithdrawalDawn,
			ssRobert, ssDawn, assumptions,
		)
		
		// Create annual cash flow
		cashFlow := domain.AnnualCashFlow{
			Year:                    year + 1,
			Date:                    projectionDate,
			AgeRobert:               ageRobert,
			AgeDawn:                 ageDawn,
			PensionRobert:           pensionRobert,
			PensionDawn:             pensionDawn,
			TSPWithdrawalRobert:     tspWithdrawalRobert,
			TSPWithdrawalDawn:       tspWithdrawalDawn,
			SSBenefitRobert:         ssRobert,
			SSBenefitDawn:           ssDawn,
			FERSSupplementRobert:    srsRobert,
			FERSSupplementDawn:      srsDawn,
			FederalTax:              federalTax,
			StateTax:                stateTax,
			LocalTax:                localTax,
			FEHBPremium:             fehbPremium,
			MedicarePremium:         medicarePremium,
			TSPBalanceRobert:        currentTSPTraditionalRobert.Add(currentTSPRothRobert),
			TSPBalanceDawn:          currentTSPTraditionalDawn.Add(currentTSPRothDawn),
			TSPBalanceTraditional:   currentTSPTraditionalRobert.Add(currentTSPTraditionalDawn),
			TSPBalanceRoth:          currentTSPRothRobert.Add(currentTSPRothDawn),
			IsRetired:               isRetired,
			IsMedicareEligible:      dateutil.IsMedicareEligible(robert.BirthDate, projectionDate) || dateutil.IsMedicareEligible(dawn.BirthDate, projectionDate),
			IsRMDYear:               dateutil.IsRMDYear(robert.BirthDate, projectionDate) || dateutil.IsRMDYear(dawn.BirthDate, projectionDate),
		}
		
		// Calculate net income
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
func (ce *CalculationEngine) calculateFERSSupplement(employee *domain.Employee, retirementDate time.Time, year int, inflationRate decimal.Decimal) decimal.Decimal {
	if year < 0 {
		return decimal.Zero
	}
	
	projectionDate := time.Now().AddDate(year, 0, 0)
	age := employee.Age(projectionDate)
	
	if age >= 62 {
		return decimal.Zero // SRS stops at age 62
	}
	
	// Calculate SRS
	serviceYears := employee.YearsOfService(retirementDate)
	srs := CalculateFERSSpecialRetirementSupplement(employee.SSBenefit62, serviceYears, age)
	
	// Apply inflation adjustments
	for y := 0; y < year; y++ {
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
	
	return traditional, roth
}

// growTSPBalance grows a TSP balance with contributions and returns
func (ce *CalculationEngine) growTSPBalance(balance, contribution, returnRate decimal.Decimal) decimal.Decimal {
	return balance.Add(contribution).Mul(decimal.NewFromFloat(1).Add(returnRate))
}

// calculateFEHBPremium calculates FEHB premium for a given year
func (ce *CalculationEngine) calculateFEHBPremium(employee *domain.Employee, year int, _ bool, premiumInflation decimal.Decimal) decimal.Decimal {
	// Inflate premium from base year
	inflationFactor := decimal.NewFromFloat(1).Add(premiumInflation)
	adjustedPremium := employee.FEHBPremiumMonthly.Mul(inflationFactor.Pow(decimal.NewFromInt(int64(year))))
	
	// Convert to annual
	return adjustedPremium.Mul(decimal.NewFromInt(12))
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
func (ce *CalculationEngine) calculateTaxes(robert, dawn *domain.Employee, _ *domain.Scenario, year int, isRetired bool, pensionRobert, pensionDawn, tspWithdrawalRobert, tspWithdrawalDawn, ssRobert, ssDawn decimal.Decimal, _ *domain.GlobalAssumptions) (decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	projectionDate := time.Now().AddDate(year, 0, 0)
	ageRobert := robert.Age(projectionDate)
	ageDawn := dawn.Age(projectionDate)
	
	// Calculate total income for tax purposes
	totalIncome := pensionRobert.Add(pensionDawn).Add(tspWithdrawalRobert).Add(tspWithdrawalDawn).Add(ssRobert).Add(ssDawn)
	
	// Calculate Social Security taxation
	taxableSS := ce.TaxCalc.CalculateSocialSecurityTaxation(ssRobert.Add(ssDawn), totalIncome.Sub(ssRobert).Sub(ssDawn))
	
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
	
	// Calculate taxes
	federalTax, stateTax, localTax, _ := ce.TaxCalc.CalculateTotalTaxes(taxableIncome, isRetired, ageRobert, ageDawn, decimal.Zero)
	
	return federalTax, stateTax, localTax
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
	
	// Calculate FEHB premiums (pre-tax while working)
	fehbPremium := robert.FEHBPremiumMonthly.Add(dawn.FEHBPremiumMonthly).Mul(decimal.NewFromInt(12))
	
	// Calculate TSP contributions (pre-tax)
	tspContributions := robert.TotalAnnualTSPContribution().Add(dawn.TotalAnnualTSPContribution())
	
	// Calculate taxable income (unused variable removed)
	_ = grossIncome.Sub(tspContributions).Sub(fehbPremium)
	
	// Calculate taxes
	ageRobert := robert.Age(time.Now())
	ageDawn := dawn.Age(time.Now())
	
	currentTaxableIncome := CalculateCurrentTaxableIncome(robert.CurrentSalary, dawn.CurrentSalary)
	federalTax, stateTax, localTax, ficaTax := ce.TaxCalc.CalculateTotalTaxes(currentTaxableIncome, false, ageRobert, ageDawn, grossIncome)
	
	// Calculate net income
	netIncome := grossIncome.Sub(federalTax).Sub(stateTax).Sub(localTax).Sub(ficaTax).Sub(fehbPremium)
	
	return netIncome
}

// generateImpactAnalysis generates impact analysis for scenarios
func (ce *CalculationEngine) generateImpactAnalysis(baselineNetIncome decimal.Decimal, scenarios []domain.ScenarioSummary) domain.ImpactAnalysis {
	var bestScenario string
	var bestIncome decimal.Decimal
	
	for _, scenario := range scenarios {
		if scenario.FirstYearNetIncome.GreaterThan(bestIncome) {
			bestIncome = scenario.FirstYearNetIncome
			bestScenario = scenario.Name
		}
	}
	
	incomeChange := bestIncome.Sub(baselineNetIncome)
	percentageChange := incomeChange.Div(baselineNetIncome).Mul(decimal.NewFromInt(100))
	monthlyChange := incomeChange.Div(decimal.NewFromInt(12))
	
	return domain.ImpactAnalysis{
		CurrentToFirstYear: domain.IncomeChange{
			ScenarioName:     bestScenario,
			NetIncomeChange:  incomeChange,
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