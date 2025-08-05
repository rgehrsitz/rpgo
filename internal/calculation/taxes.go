package calculation

import (
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
	Year              int
	StandardDeduction decimal.Decimal
	Brackets          []TaxBracket
	AdditionalStdDed  decimal.Decimal // For age 65+
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
	// Convert domain.TaxBracket to calculation.TaxBracket
	var brackets []TaxBracket
	for _, bracket := range config.TaxBrackets2025 {
		brackets = append(brackets, TaxBracket{
			Min:  bracket.Min,
			Max:  bracket.Max,
			Rate: bracket.Rate,
		})
	}

	return &FederalTaxCalculator{
		Year:              2025, // TODO: Make year configurable
		StandardDeduction: config.StandardDeductionMFJ,
		AdditionalStdDed:  config.AdditionalStandardDeduction,
		Brackets:          brackets,
	}
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

// CalculateTotalTaxes calculates all applicable taxes for a given income scenario
func (ctc *ComprehensiveTaxCalculator) CalculateTotalTaxes(income domain.TaxableIncome, isRetired bool, age1, age2 int, totalHouseholdWages decimal.Decimal) (decimal.Decimal, decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	// Calculate gross income for federal tax
	grossIncome := income.Salary.Add(income.FERSPension).Add(income.TSPWithdrawalsTrad).Add(income.TaxableSSBenefits).Add(income.OtherTaxableIncome)

	var federalTax, stateTax, localTax, ficaTax decimal.Decimal

	// Always use proper progressive tax calculation for federal tax
	federalTax = ctc.FederalTaxCalc.CalculateFederalTax(grossIncome, age1, age2)

	// State and local taxes
	stateTax = ctc.StateTaxCalc.CalculateTax(income, isRetired)
	localTax = ctc.LocalTaxCalc.CalculateEIT(income.WageIncome, isRetired)

	// FICA only applies to wages (working income)
	if !isRetired && totalHouseholdWages.GreaterThan(decimal.Zero) {
		ficaTax = ctc.FICATaxCalc.CalculateFICA(totalHouseholdWages, totalHouseholdWages)
	} else {
		ficaTax = decimal.Zero // No FICA in retirement
	}

	return federalTax, stateTax, localTax, ficaTax
}

// CalculateTaxableIncome creates a TaxableIncome struct from cash flow data
func CalculateTaxableIncome(cashFlow domain.AnnualCashFlow, isRetired bool) domain.TaxableIncome {
	return domain.TaxableIncome{
		Salary:             decimal.Zero, // No salary in retirement
		FERSPension:        cashFlow.PensionRobert.Add(cashFlow.PensionDawn),
		TSPWithdrawalsTrad: cashFlow.TSPWithdrawalRobert.Add(cashFlow.TSPWithdrawalDawn), // Assuming all TSP withdrawals are from traditional
		TaxableSSBenefits:  cashFlow.SSBenefitRobert.Add(cashFlow.SSBenefitDawn),         // Will be adjusted for taxation
		OtherTaxableIncome: decimal.Zero,
		WageIncome:         decimal.Zero, // No wages in retirement
		InterestIncome:     decimal.Zero, // Could be added if needed
	}
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
