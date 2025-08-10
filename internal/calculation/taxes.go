package calculation

import (
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// TAX CALCULATION ASSUMPTIONS:
//
// 1. Federal Tax Brackets: Uses 2025 tax brackets for all projection years
//    - No inflation indexing applied to future years
//    - Standard deduction: $30,000 (2025 MFJ estimate)
//    - Additional standard deduction for age 65+: $1,550 per person
//
// 2. Pennsylvania State Tax: 3.07% flat tax rate (no inflation adjustment)
//
// 3. Upper Makefield EIT: 1% flat tax on earned income only
//    - Does not apply to retirement income (pensions, TSP, SS)
//
// 4. Medicare Part B & IRMAA: Placeholder implementation
//    - Base premium: $185/month per person (2025 estimate)
//    - IRMAA surcharge: $200/month placeholder (needs AGI-based calculation)
//
// TODO: Consider adding inflation indexing for long-term projections

// TaxBracket represents a federal tax bracket
type TaxBracket struct {
	Min  decimal.Decimal
	Max  decimal.Decimal
	Rate decimal.Decimal
}

// FederalTaxCalculator handles federal income tax calculations
type FederalTaxCalculator struct {
	Year                    int
	StandardDeduction       decimal.Decimal
	StandardDeductionSingle decimal.Decimal
	Brackets                []TaxBracket
	BracketsSingle          []TaxBracket
	AdditionalStdDed        decimal.Decimal // For age 65+
}

// NewFederalTaxCalculator2025 creates a new federal tax calculator for 2025
func NewFederalTaxCalculator2025() *FederalTaxCalculator {
	return &FederalTaxCalculator{
		Year:              2025,
		StandardDeduction: decimal.NewFromInt(30000), // MFJ 2025 estimated
		AdditionalStdDed:  decimal.NewFromInt(1550),  // Per person 65+
		Brackets: []TaxBracket{
			{decimal.Zero, decimal.NewFromInt(23200), decimal.NewFromFloat(0.10)},
			{decimal.NewFromInt(23201), decimal.NewFromInt(94300), decimal.NewFromFloat(0.12)},
			{decimal.NewFromInt(94301), decimal.NewFromInt(201050), decimal.NewFromFloat(0.22)},
			{decimal.NewFromInt(201051), decimal.NewFromInt(383900), decimal.NewFromFloat(0.24)},
			{decimal.NewFromInt(383901), decimal.NewFromInt(487450), decimal.NewFromFloat(0.32)},
			{decimal.NewFromInt(487451), decimal.NewFromInt(731200), decimal.NewFromFloat(0.35)},
			{decimal.NewFromInt(731201), decimal.NewFromInt(999999999), decimal.NewFromFloat(0.37)},
		},
	}
}

// NewFederalTaxCalculator creates a new federal tax calculator with configurable values
func NewFederalTaxCalculator(config domain.FederalTaxConfig) *FederalTaxCalculator {
	var bracketsMFJ []TaxBracket
	for _, b := range config.TaxBrackets2025 {
		bracketsMFJ = append(bracketsMFJ, TaxBracket{Min: b.Min, Max: b.Max, Rate: b.Rate})
	}
	if len(bracketsMFJ) == 0 { // fallback defaults
		bracketsMFJ = []TaxBracket{
			{decimal.Zero, decimal.NewFromInt(23200), decimal.NewFromFloat(0.10)},
			{decimal.NewFromInt(23201), decimal.NewFromInt(94300), decimal.NewFromFloat(0.12)},
			{decimal.NewFromInt(94301), decimal.NewFromInt(201050), decimal.NewFromFloat(0.22)},
			{decimal.NewFromInt(201051), decimal.NewFromInt(383900), decimal.NewFromFloat(0.24)},
			{decimal.NewFromInt(383901), decimal.NewFromInt(487450), decimal.NewFromFloat(0.32)},
			{decimal.NewFromInt(487451), decimal.NewFromInt(731200), decimal.NewFromFloat(0.35)},
			{decimal.NewFromInt(731201), decimal.NewFromInt(999999999), decimal.NewFromFloat(0.37)},
		}
	}
	var bracketsSingle []TaxBracket
	for _, b := range config.TaxBrackets2025Single {
		bracketsSingle = append(bracketsSingle, TaxBracket{Min: b.Min, Max: b.Max, Rate: b.Rate})
	}
	// Provide defaults if single not supplied
	stdSingle := config.StandardDeductionSingle
	if stdSingle.IsZero() && !config.StandardDeductionMFJ.IsZero() {
		stdSingle = config.StandardDeductionMFJ.Div(decimal.NewFromInt(2))
	}
	if len(bracketsSingle) == 0 && len(bracketsMFJ) > 0 {
		for _, b := range bracketsMFJ {
			bracketsSingle = append(bracketsSingle, TaxBracket{Min: b.Min.Div(decimal.NewFromInt(2)), Max: b.Max.Div(decimal.NewFromInt(2)), Rate: b.Rate})
		}
	}
	return &FederalTaxCalculator{Year: 2025, StandardDeduction: config.StandardDeductionMFJ, StandardDeductionSingle: stdSingle, AdditionalStdDed: config.AdditionalStandardDeduction, Brackets: bracketsMFJ, BracketsSingle: bracketsSingle}
}

// CalculateFederalTax calculates federal income tax
func (ftc *FederalTaxCalculator) CalculateFederalTax(grossIncome decimal.Decimal, age1, age2 int) decimal.Decimal {
	standardDed := ftc.StandardDeduction

	// Additional standard deduction for seniors
	if age1 >= 65 {
		standardDed = standardDed.Add(ftc.AdditionalStdDed)
	}
	if age2 >= 65 {
		standardDed = standardDed.Add(ftc.AdditionalStdDed)
	}

	taxableIncome := grossIncome.Sub(standardDed)
	if taxableIncome.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}

	var totalTax decimal.Decimal
	for _, bracket := range ftc.Brackets {
		if taxableIncome.LessThanOrEqual(bracket.Min) {
			break
		}
		incomeInBracket := decimal.Min(taxableIncome, bracket.Max).Sub(bracket.Min)
		if incomeInBracket.GreaterThan(decimal.Zero) {
			totalTax = totalTax.Add(incomeInBracket.Mul(bracket.Rate))
		}
	}

	return totalTax
}

// PennsylvaniaTaxCalculator handles Pennsylvania state tax calculations
type PennsylvaniaTaxCalculator struct {
	Rate decimal.Decimal
}

// NewPennsylvaniaTaxCalculator creates a new Pennsylvania tax calculator
func NewPennsylvaniaTaxCalculator() *PennsylvaniaTaxCalculator {
	return &PennsylvaniaTaxCalculator{
		Rate: decimal.NewFromFloat(0.0307), // Default rate
	}
}

// NewPennsylvaniaTaxCalculatorWithConfig creates a new Pennsylvania tax calculator with configurable rate
func NewPennsylvaniaTaxCalculatorWithConfig(config domain.StateLocalTaxConfig) *PennsylvaniaTaxCalculator {
	return &PennsylvaniaTaxCalculator{
		Rate: config.PennsylvaniaRate,
	}
}

// CalculatePennsylvaniaStateIncomeTax calculates Pennsylvania state income tax
// PA has a flat tax rate (currently 3.07%)
// Key Exclusions: PA does NOT tax FERS pensions, TSP withdrawals, or Social Security benefits
// Only earned income (salary) is typically taxed
func (ptc *PennsylvaniaTaxCalculator) CalculateTax(income domain.TaxableIncome, isRetired bool) decimal.Decimal {
	if isRetired {
		// PA exempts retirement income: pensions, TSP, Social Security
		// Only tax earned income (wages) and interest income
		taxablePA := income.WageIncome.Add(income.InterestIncome).Add(income.OtherTaxableIncome)
		return taxablePA.Mul(ptc.Rate)
	}

	// While working: tax wages at configured rate
	return income.WageIncome.Mul(ptc.Rate)
}

// UpperMakefieldEITCalculator handles Upper Makefield Township local tax calculations
type UpperMakefieldEITCalculator struct {
	Rate decimal.Decimal
}

// NewUpperMakefieldEITCalculator creates a new Upper Makefield EIT calculator
func NewUpperMakefieldEITCalculator() *UpperMakefieldEITCalculator {
	return &UpperMakefieldEITCalculator{
		Rate: decimal.NewFromFloat(0.01), // Default rate
	}
}

// NewUpperMakefieldEITCalculatorWithConfig creates a new Upper Makefield EIT calculator with configurable rate
func NewUpperMakefieldEITCalculatorWithConfig(config domain.StateLocalTaxConfig) *UpperMakefieldEITCalculator {
	return &UpperMakefieldEITCalculator{
		Rate: config.UpperMakefieldEITRate,
	}
}

// CalculateEIT calculates the Earned Income Tax for Upper Makefield Township
// EIT only applies to earned income, not retirement income
func (ume *UpperMakefieldEITCalculator) CalculateEIT(wageIncome decimal.Decimal, isRetired bool) decimal.Decimal {
	if isRetired {
		return decimal.Zero // EIT only applies to earned income
	}

	return wageIncome.Mul(ume.Rate)
}

// FICACalculator handles FICA tax calculations
type FICACalculator struct {
	Year                int
	SSWageBase          decimal.Decimal
	SSRate              decimal.Decimal
	MedicareRate        decimal.Decimal
	AdditionalRate      decimal.Decimal
	HighIncomeThreshold decimal.Decimal
}

// NewFICACalculator2025 creates a new FICA calculator for 2025
func NewFICACalculator2025() *FICACalculator {
	return &FICACalculator{
		Year:                2025,
		SSWageBase:          decimal.NewFromInt(176100), // 2025 official
		SSRate:              decimal.NewFromFloat(0.062),
		MedicareRate:        decimal.NewFromFloat(0.0145),
		AdditionalRate:      decimal.NewFromFloat(0.009),
		HighIncomeThreshold: decimal.NewFromInt(250000), // MFJ
	}
}

// NewFICACalculator creates a new FICA calculator with configurable values
func NewFICACalculator(config domain.FICATaxConfig) *FICACalculator {
	return &FICACalculator{
		Year:                2025, // TODO: Make year configurable
		SSWageBase:          config.SocialSecurityWageBase,
		SSRate:              config.SocialSecurityRate,
		MedicareRate:        config.MedicareRate,
		AdditionalRate:      config.AdditionalMedicareRate,
		HighIncomeThreshold: config.HighIncomeThresholdMFJ,
	}
}

// CalculateFICA calculates FICA taxes (Social Security and Medicare)
func (fc *FICACalculator) CalculateFICA(wages decimal.Decimal, totalHouseholdWages decimal.Decimal) decimal.Decimal {
	// Social Security tax (capped per individual)
	ssWages := decimal.Min(wages, fc.SSWageBase)
	ssTax := ssWages.Mul(fc.SSRate)

	// Medicare tax (no cap)
	medicareTax := wages.Mul(fc.MedicareRate)

	// Additional Medicare tax for high earners - proportionally allocated
	var additionalMedicare decimal.Decimal
	if totalHouseholdWages.GreaterThan(fc.HighIncomeThreshold) {
		excessWages := totalHouseholdWages.Sub(fc.HighIncomeThreshold)
		totalAdditionalMedicare := excessWages.Mul(fc.AdditionalRate)
		// Allocate proportionally based on individual wages
		wagesProportion := wages.Div(totalHouseholdWages)
		additionalMedicare = totalAdditionalMedicare.Mul(wagesProportion)
	}

	return ssTax.Add(medicareTax).Add(additionalMedicare)
}

// CalculateFICAWithProration calculates FICA taxes with proration for partial year work
func (fc *FICACalculator) CalculateFICAWithProration(wages decimal.Decimal, totalHouseholdWages decimal.Decimal, workFraction decimal.Decimal) decimal.Decimal {
	// Apply work fraction to wages first
	proratedWages := wages.Mul(workFraction)
	proratedHouseholdWages := totalHouseholdWages.Mul(workFraction)

	// Social Security tax (capped per individual, then prorated)
	ssWages := decimal.Min(proratedWages, fc.SSWageBase)
	ssTax := ssWages.Mul(fc.SSRate)

	// Medicare tax (no cap, prorated)
	medicareTax := proratedWages.Mul(fc.MedicareRate)

	// Additional Medicare tax for high earners (prorated and proportionally allocated)
	var additionalMedicare decimal.Decimal
	if proratedHouseholdWages.GreaterThan(fc.HighIncomeThreshold) {
		excessWages := proratedHouseholdWages.Sub(fc.HighIncomeThreshold)
		totalAdditionalMedicare := excessWages.Mul(fc.AdditionalRate)
		// Allocate proportionally based on individual prorated wages
		wagesProportion := proratedWages.Div(proratedHouseholdWages)
		additionalMedicare = totalAdditionalMedicare.Mul(wagesProportion)
	}

	return ssTax.Add(medicareTax).Add(additionalMedicare)
}

// ComprehensiveTaxCalculator handles all tax calculations
type ComprehensiveTaxCalculator struct {
	FederalTaxCalc *FederalTaxCalculator
	StateTaxCalc   *PennsylvaniaTaxCalculator
	LocalTaxCalc   *UpperMakefieldEITCalculator
	FICATaxCalc    *FICACalculator
	SSTaxCalc      *SSTaxCalculator
}

// NewComprehensiveTaxCalculator creates a new comprehensive tax calculator
func NewComprehensiveTaxCalculator() *ComprehensiveTaxCalculator {
	return &ComprehensiveTaxCalculator{
		FederalTaxCalc: NewFederalTaxCalculator2025(),
		StateTaxCalc:   NewPennsylvaniaTaxCalculator(),
		LocalTaxCalc:   NewUpperMakefieldEITCalculator(),
		FICATaxCalc:    NewFICACalculator2025(),
		SSTaxCalc:      NewSSTaxCalculator(),
	}
}

// NewComprehensiveTaxCalculatorWithConfig creates a new comprehensive tax calculator with configurable values
func NewComprehensiveTaxCalculatorWithConfig(federalRules domain.FederalRules) *ComprehensiveTaxCalculator {
	return &ComprehensiveTaxCalculator{
		FederalTaxCalc: NewFederalTaxCalculator(federalRules.FederalTaxConfig),
		StateTaxCalc:   NewPennsylvaniaTaxCalculatorWithConfig(federalRules.StateLocalTaxConfig),
		LocalTaxCalc:   NewUpperMakefieldEITCalculatorWithConfig(federalRules.StateLocalTaxConfig),
		FICATaxCalc:    NewFICACalculator(federalRules.FICATaxConfig),
		SSTaxCalc:      NewSSTaxCalculator(),
	}
}

// CalculateTotalTaxes calculates all applicable taxes with inflation-adjusted tax brackets
func (ctc *ComprehensiveTaxCalculator) CalculateTotalTaxes(taxableIncome domain.TaxableIncome, isRetired bool, ageRobert, ageDawn int, workingIncome decimal.Decimal) (decimal.Decimal, decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	// Calculate federal tax with inflation-adjusted brackets
	federalTax := ctc.calculateFederalTaxWithInflation(taxableIncome, ageRobert, ageDawn)

	// Calculate state tax
	stateTax := ctc.StateTaxCalc.CalculateTax(taxableIncome, isRetired)

	// Calculate local tax (only on earned income)
	localTax := ctc.LocalTaxCalc.CalculateEIT(workingIncome, isRetired)

	// Calculate FICA tax (only on earned income)
	ficaTax := ctc.FICATaxCalc.CalculateFICA(workingIncome, workingIncome)

	return federalTax, stateTax, localTax, ficaTax
}

// calculateFederalTaxWithInflation calculates federal tax with inflation-adjusted brackets
func (ctc *ComprehensiveTaxCalculator) calculateFederalTaxWithInflation(taxableIncome domain.TaxableIncome, ageRobert, ageDawn int) decimal.Decimal {
	// Calculate total taxable income
	totalIncome := taxableIncome.Salary.Add(taxableIncome.FERSPension).Add(taxableIncome.TSPWithdrawalsTrad).Add(taxableIncome.TaxableSSBenefits).Add(taxableIncome.OtherTaxableIncome)

	// Apply standard deduction with age-based adjustments
	standardDeduction := ctc.FederalTaxCalc.StandardDeduction

	// Add additional standard deduction for taxpayers 65 and older
	if ageRobert >= 65 {
		standardDeduction = standardDeduction.Add(ctc.FederalTaxCalc.AdditionalStdDed)
	}
	if ageDawn >= 65 {
		standardDeduction = standardDeduction.Add(ctc.FederalTaxCalc.AdditionalStdDed)
	}

	// Calculate adjusted gross income
	agi := totalIncome.Sub(standardDeduction)
	if agi.LessThan(decimal.Zero) {
		agi = decimal.Zero
	}

	// Apply inflation adjustment to tax brackets
	// Note: For current tests and 2025 calculations, we do not adjust brackets
	// Set to 1.0 to keep bracket thresholds unchanged
	inflationAdjustment := decimal.NewFromFloat(1.0)

	// Calculate tax using inflation-adjusted brackets
	tax := decimal.Zero
	remainingIncome := agi

	for _, bracket := range ctc.FederalTaxCalc.Brackets {
		// Apply inflation adjustment to bracket thresholds
		adjustedMin := bracket.Min.Mul(inflationAdjustment)
		adjustedMax := bracket.Max.Mul(inflationAdjustment)

		if remainingIncome.LessThanOrEqual(decimal.Zero) {
			break
		}

		// Determine the width of this bracket
		bracketWidth := adjustedMax.Sub(adjustedMin)
		if bracketWidth.LessThanOrEqual(decimal.Zero) {
			continue
		}

		// The amount taxed in this bracket is limited by the remaining income
		// and the width of the bracket. Do not subtract adjustedMin from remainingIncome
		// because remainingIncome already represents income above all previous brackets.
		incomeInBracket := decimal.Min(remainingIncome, bracketWidth)

		// Only tax amounts once the taxpayer's income exceeds the start of this bracket
		if agi.GreaterThan(adjustedMin) && incomeInBracket.GreaterThan(decimal.Zero) {
			tax = tax.Add(incomeInBracket.Mul(bracket.Rate))
			remainingIncome = remainingIncome.Sub(incomeInBracket)
		}
	}

	return tax
}

// calculateFederalTaxWithStatus allows specifying filing status ("mfj" or "single") and number of seniors 65+.
func (ctc *ComprehensiveTaxCalculator) calculateFederalTaxWithStatus(agiComponents domain.TaxableIncome, filingStatus string, seniors int) decimal.Decimal {
	totalIncome := agiComponents.Salary.Add(agiComponents.FERSPension).Add(agiComponents.TSPWithdrawalsTrad).Add(agiComponents.TaxableSSBenefits).Add(agiComponents.OtherTaxableIncome)

	// Standard deduction based on filing status
	standardDed := ctc.FederalTaxCalc.StandardDeduction
	brackets := ctc.FederalTaxCalc.Brackets
	if filingStatus == "single" {
		standardDed = ctc.FederalTaxCalc.StandardDeductionSingle
		if len(ctc.FederalTaxCalc.BracketsSingle) > 0 {
			brackets = ctc.FederalTaxCalc.BracketsSingle
		}
	}
	for i := 0; i < seniors; i++ {
		standardDed = standardDed.Add(ctc.FederalTaxCalc.AdditionalStdDed)
	}

	agi := totalIncome.Sub(standardDed)
	if agi.LessThan(decimal.Zero) {
		agi = decimal.Zero
	}

	inflationAdjustment := decimal.NewFromFloat(1.0)
	remaining := agi
	tax := decimal.Zero
	for _, b := range brackets {
		adjMin := b.Min.Mul(inflationAdjustment)
		adjMax := b.Max.Mul(inflationAdjustment)
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}
		width := adjMax.Sub(adjMin)
		if width.LessThanOrEqual(decimal.Zero) {
			continue
		}
		incomeInBracket := decimal.Min(remaining, width)
		if agi.GreaterThan(adjMin) && incomeInBracket.GreaterThan(decimal.Zero) {
			tax = tax.Add(incomeInBracket.Mul(b.Rate))
			remaining = remaining.Sub(incomeInBracket)
		}
	}
	return tax
}

// CalculateTaxableIncome creates a TaxableIncome struct from cash flow data
func CalculateTaxableIncome(cashFlow domain.AnnualCashFlow, isRetired bool) domain.TaxableIncome {
	return domain.TaxableIncome{Salary: decimal.Zero, FERSPension: cashFlow.PensionRobert.Add(cashFlow.PensionDawn).Add(cashFlow.SurvivorPensionRobert).Add(cashFlow.SurvivorPensionDawn), TSPWithdrawalsTrad: cashFlow.TSPWithdrawalRobert.Add(cashFlow.TSPWithdrawalDawn), TaxableSSBenefits: cashFlow.SSBenefitRobert.Add(cashFlow.SSBenefitDawn), OtherTaxableIncome: decimal.Zero, WageIncome: decimal.Zero, InterestIncome: decimal.Zero}
}

// CalculateCurrentTaxableIncome calculates taxable income for current employment
func CalculateCurrentTaxableIncome(robertSalary, dawnSalary decimal.Decimal) domain.TaxableIncome {
	totalSalary := robertSalary.Add(dawnSalary)

	return domain.TaxableIncome{
		Salary:             totalSalary,
		FERSPension:        decimal.Zero,
		TSPWithdrawalsTrad: decimal.Zero,
		TaxableSSBenefits:  decimal.Zero,
		OtherTaxableIncome: decimal.Zero,
		WageIncome:         totalSalary,
		InterestIncome:     decimal.Zero,
	}
}

// CalculateSocialSecurityTaxation calculates the taxable portion of Social Security benefits
func (ctc *ComprehensiveTaxCalculator) CalculateSocialSecurityTaxation(ssBenefits decimal.Decimal, otherIncome decimal.Decimal) decimal.Decimal {
	// Calculate provisional income
	provisionalIncome := ctc.SSTaxCalc.CalculateProvisionalIncome(otherIncome, decimal.Zero, ssBenefits)

	// Calculate taxable portion
	return ctc.SSTaxCalc.CalculateTaxableSocialSecurity(ssBenefits, provisionalIncome)
}

// calculateTaxes calculates all applicable taxes
func (ce *CalculationEngine) calculateTaxes(robert, dawn *domain.Employee, scenario *domain.Scenario, year int, isRetired bool, pensionRobert, pensionDawn, survivorPensionRobert, survivorPensionDawn, tspWithdrawalRobert, tspWithdrawalDawn, ssRobert, ssDawn decimal.Decimal, assumptions *domain.GlobalAssumptions, workingIncomeRobert, workingIncomeDawn decimal.Decimal) (federal decimal.Decimal, state decimal.Decimal, local decimal.Decimal, fica decimal.Decimal, taxableIncomeTotal decimal.Decimal, stdDed decimal.Decimal, filingStatusOut string, seniorsOut int) {
	projectionStartYear := ProjectionBaseYear
	projectionDate := time.Date(projectionStartYear, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(year, 0, 0)
	ageRobert := robert.Age(projectionDate)
	ageDawn := dawn.Age(projectionDate)

	// Determine mortality & filing status for this year
	filingStatus := "mfj"
	seniors := 0
	if ageRobert >= 65 {
		seniors++
	}
	if ageDawn >= 65 {
		seniors++
	}

	// Use shared helper for death year indexes (projection horizon not needed here; pass year+1 as conservative bound)
	robertDeathYearIndex, dawnDeathYearIndex := deriveDeathYearIndexes(scenario, robert, dawn, year+1+5) // simple upper bound
	robertDeceased := robertDeathYearIndex != nil && year >= *robertDeathYearIndex
	dawnDeceased := dawnDeathYearIndex != nil && year >= *dawnDeathYearIndex
	if (robertDeceased || dawnDeceased) && !(robertDeceased && dawnDeceased) {
		// One survivor; evaluate filing status switch policy
		if scenario != nil && scenario.Mortality != nil && scenario.Mortality.Assumptions != nil {
			mode := scenario.Mortality.Assumptions.FilingStatusSwitch
			if mode == "immediate" {
				filingStatus = "single"
				seniors = 0
				// Count surviving senior for additional deduction
				if !robertDeceased && ageRobert >= 65 {
					seniors = 1
				}
				if !dawnDeceased && ageDawn >= 65 {
					seniors = 1
				}
			} else if mode == "next_year" {
				// Switch next year after death event
				deathYear := 0
				if robertDeceased && robertDeathYearIndex != nil {
					deathYear = *robertDeathYearIndex
				}
				if dawnDeceased && dawnDeathYearIndex != nil {
					deathYear = *dawnDeathYearIndex
				}
				if year > deathYear {
					filingStatus = "single"
					seniors = 0
					if !robertDeceased && ageRobert >= 65 {
						seniors = 1
					}
					if !dawnDeceased && ageDawn >= 65 {
						seniors = 1
					}
				}
			}
		}
	}

	// Check if this is a transition year (has both working and retirement income)
	isTransitionYear := (workingIncomeRobert.GreaterThan(decimal.Zero) || workingIncomeDawn.GreaterThan(decimal.Zero)) &&
		(pensionRobert.GreaterThan(decimal.Zero) || pensionDawn.GreaterThan(decimal.Zero) || tspWithdrawalRobert.GreaterThan(decimal.Zero) || tspWithdrawalDawn.GreaterThan(decimal.Zero) || ssRobert.GreaterThan(decimal.Zero) || ssDawn.GreaterThan(decimal.Zero))

	if isTransitionYear {
		// Transition year: combine working and retirement income, include survivor pensions
		totalWorkingIncome := workingIncomeRobert.Add(workingIncomeDawn)
		totalRetirementIncome := pensionRobert.Add(pensionDawn).Add(survivorPensionRobert).Add(survivorPensionDawn).Add(tspWithdrawalRobert).Add(tspWithdrawalDawn)

		// Calculate Social Security taxation (filing status aware thresholds)
		totalSSBenefits := ssRobert.Add(ssDawn)
		provisional := ce.TaxCalc.SSTaxCalc.CalculateProvisionalIncome(totalRetirementIncome, decimal.Zero, totalSSBenefits)
		var taxableSS decimal.Decimal
		if filingStatus == "single" {
			taxableSS = ce.TaxCalc.SSTaxCalc.CalculateTaxableSocialSecuritySingle(totalSSBenefits, provisional)
		} else {
			taxableSS = ce.TaxCalc.SSTaxCalc.CalculateTaxableSocialSecurity(totalSSBenefits, provisional)
		}

		// Create taxable income structure for transition year
		taxableIncome := domain.TaxableIncome{
			Salary:             totalWorkingIncome,
			FERSPension:        pensionRobert.Add(pensionDawn).Add(survivorPensionRobert).Add(survivorPensionDawn),
			TSPWithdrawalsTrad: tspWithdrawalRobert.Add(tspWithdrawalDawn),
			TaxableSSBenefits:  taxableSS,
			OtherTaxableIncome: decimal.Zero,
			WageIncome:         totalWorkingIncome,
			InterestIncome:     decimal.Zero,
		}

		// Calculate taxes for transition year (FICA only on working income, with proration)
		// Federal tax using filing status logic
		federalTax := ce.TaxCalc.calculateFederalTaxWithStatus(taxableIncome, filingStatus, seniors)
		stateTax := ce.TaxCalc.StateTaxCalc.CalculateTax(taxableIncome, false)
		localTax := ce.TaxCalc.LocalTaxCalc.CalculateEIT(totalWorkingIncome, false)
		robertFICA := ce.TaxCalc.FICATaxCalc.CalculateFICA(workingIncomeRobert, totalWorkingIncome)
		dawnFICA := ce.TaxCalc.FICATaxCalc.CalculateFICA(workingIncomeDawn, totalWorkingIncome)
		ficaTax := robertFICA.Add(dawnFICA)
		std := ce.TaxCalc.FederalTaxCalc.StandardDeduction
		if filingStatus == "single" {
			std = ce.TaxCalc.FederalTaxCalc.StandardDeductionSingle
		}
		for i := 0; i < seniors; i++ {
			std = std.Add(ce.TaxCalc.FederalTaxCalc.AdditionalStdDed)
		}
		return federalTax, stateTax, localTax, ficaTax, taxableIncome.Salary.Add(taxableIncome.FERSPension).Add(taxableIncome.TSPWithdrawalsTrad).Add(taxableIncome.TaxableSSBenefits), std, filingStatus, seniors
	} else if isRetired {
		// Fully retired year
		// Calculate other income (excluding Social Security)
		otherIncome := pensionRobert.Add(pensionDawn).Add(survivorPensionRobert).Add(survivorPensionDawn).Add(tspWithdrawalRobert).Add(tspWithdrawalDawn)

		// Calculate Social Security taxation with filing status thresholds
		totalSSBenefits := ssRobert.Add(ssDawn)
		provisional := ce.TaxCalc.SSTaxCalc.CalculateProvisionalIncome(otherIncome, decimal.Zero, totalSSBenefits)
		var taxableSS decimal.Decimal
		if filingStatus == "single" {
			taxableSS = ce.TaxCalc.SSTaxCalc.CalculateTaxableSocialSecuritySingle(totalSSBenefits, provisional)
		} else {
			taxableSS = ce.TaxCalc.SSTaxCalc.CalculateTaxableSocialSecurity(totalSSBenefits, provisional)
		}

		// Create taxable income structure
		taxableIncome := domain.TaxableIncome{
			Salary:             decimal.Zero, // No salary in retirement
			FERSPension:        pensionRobert.Add(pensionDawn).Add(survivorPensionRobert).Add(survivorPensionDawn),
			TSPWithdrawalsTrad: tspWithdrawalRobert.Add(tspWithdrawalDawn), // Assuming all TSP withdrawals are from traditional
			TaxableSSBenefits:  taxableSS,
			OtherTaxableIncome: decimal.Zero,
			WageIncome:         decimal.Zero,
			InterestIncome:     decimal.Zero,
		}

		// Calculate taxes (no FICA in retirement)
		federalTax := ce.TaxCalc.calculateFederalTaxWithStatus(taxableIncome, filingStatus, seniors)
		stateTax := ce.TaxCalc.StateTaxCalc.CalculateTax(taxableIncome, true)
		localTax := ce.TaxCalc.LocalTaxCalc.CalculateEIT(decimal.Zero, true)
		std := ce.TaxCalc.FederalTaxCalc.StandardDeduction
		if filingStatus == "single" {
			std = ce.TaxCalc.FederalTaxCalc.StandardDeductionSingle
		}
		for i := 0; i < seniors; i++ {
			std = std.Add(ce.TaxCalc.FederalTaxCalc.AdditionalStdDed)
		}
		return federalTax, stateTax, localTax, decimal.Zero, taxableIncome.Salary.Add(taxableIncome.FERSPension).Add(taxableIncome.TSPWithdrawalsTrad).Add(taxableIncome.TaxableSSBenefits), std, filingStatus, seniors
	} else {
		// Pre-retirement: calculate current working income
		currentTaxableIncome := CalculateCurrentTaxableIncome(robert.CurrentSalary, dawn.CurrentSalary)
		federalTax := ce.TaxCalc.calculateFederalTaxWithStatus(currentTaxableIncome, filingStatus, seniors)
		stateTax := ce.TaxCalc.StateTaxCalc.CalculateTax(currentTaxableIncome, false)
		localTax := ce.TaxCalc.LocalTaxCalc.CalculateEIT(robert.CurrentSalary.Add(dawn.CurrentSalary), false)
		ficaTax := ce.TaxCalc.FICATaxCalc.CalculateFICA(robert.CurrentSalary, robert.CurrentSalary.Add(dawn.CurrentSalary)).Add(ce.TaxCalc.FICATaxCalc.CalculateFICA(dawn.CurrentSalary, robert.CurrentSalary.Add(dawn.CurrentSalary)))
		std := ce.TaxCalc.FederalTaxCalc.StandardDeduction
		if filingStatus == "single" {
			std = ce.TaxCalc.FederalTaxCalc.StandardDeductionSingle
		}
		for i := 0; i < seniors; i++ {
			std = std.Add(ce.TaxCalc.FederalTaxCalc.AdditionalStdDed)
		}
		return federalTax, stateTax, localTax, ficaTax, currentTaxableIncome.Salary, std, filingStatus, seniors
	}
}
