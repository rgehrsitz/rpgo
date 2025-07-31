package calculation

import (
	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

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
		
		taxableInBracket := decimal.Min(taxableIncome, bracket.Max).Sub(bracket.Min)
		if taxableInBracket.GreaterThan(decimal.Zero) {
			totalTax = totalTax.Add(taxableInBracket.Mul(bracket.Rate))
		}
	}
	
	return totalTax
}

// PennsylvaniaTaxCalculator handles Pennsylvania state tax calculations
type PennsylvaniaTaxCalculator struct{}

// NewPennsylvaniaTaxCalculator creates a new Pennsylvania tax calculator
func NewPennsylvaniaTaxCalculator() *PennsylvaniaTaxCalculator {
	return &PennsylvaniaTaxCalculator{}
}

// CalculatePennsylvaniaStateIncomeTax calculates Pennsylvania state income tax
// PA has a flat tax rate (currently 3.07%)
// Key Exclusions: PA does NOT tax FERS pensions, TSP withdrawals, or Social Security benefits
// Only earned income (salary) is typically taxed
func (ptc *PennsylvaniaTaxCalculator) CalculateTax(income domain.TaxableIncome, isRetired bool) decimal.Decimal {
	paRate := decimal.NewFromFloat(0.0307)
	
	if isRetired {
		// PA exempts retirement income: pensions, TSP, Social Security
		taxablePA := income.WageIncome.Add(income.InterestIncome).Add(income.OtherTaxableIncome)
		return taxablePA.Mul(paRate)
	}
	
	// While working: tax wages at 3.07%
	return income.WageIncome.Mul(paRate)
}

// UpperMakefieldEITCalculator handles Upper Makefield Township local tax calculations
type UpperMakefieldEITCalculator struct{}

// NewUpperMakefieldEITCalculator creates a new Upper Makefield EIT calculator
func NewUpperMakefieldEITCalculator() *UpperMakefieldEITCalculator {
	return &UpperMakefieldEITCalculator{}
}

// CalculateEIT calculates the Earned Income Tax for Upper Makefield Township
// EIT only applies to earned income, not retirement income
func (ume *UpperMakefieldEITCalculator) CalculateEIT(wageIncome decimal.Decimal, isRetired bool) decimal.Decimal {
	if isRetired {
		return decimal.Zero // EIT only applies to earned income
	}
	
	eitRate := decimal.NewFromFloat(0.01) // 1% on earned income
	return wageIncome.Mul(eitRate)
}

// FICACalculator handles FICA tax calculations
type FICACalculator struct {
	Year                 int
	SSWageBase           decimal.Decimal
	SSRate               decimal.Decimal
	MedicareRate         decimal.Decimal
	AdditionalRate       decimal.Decimal
	HighIncomeThreshold  decimal.Decimal
}

// NewFICACalculator2025 creates a new FICA calculator for 2025
func NewFICACalculator2025() *FICACalculator {
	return &FICACalculator{
		Year:                2025,
		SSWageBase:          decimal.NewFromInt(168600), // 2025 estimated
		SSRate:              decimal.NewFromFloat(0.062),
		MedicareRate:        decimal.NewFromFloat(0.0145),
		AdditionalRate:      decimal.NewFromFloat(0.009),
		HighIncomeThreshold: decimal.NewFromInt(250000), // MFJ
	}
}

// CalculateFICA calculates FICA taxes (Social Security and Medicare)
func (fc *FICACalculator) CalculateFICA(wages decimal.Decimal, totalHouseholdWages decimal.Decimal) decimal.Decimal {
	// Social Security tax (capped)
	ssWages := decimal.Min(wages, fc.SSWageBase)
	ssTax := ssWages.Mul(fc.SSRate)
	
	// Medicare tax (no cap)
	medicareTax := wages.Mul(fc.MedicareRate)
	
	// Additional Medicare tax for high earners
	var additionalMedicare decimal.Decimal
	if totalHouseholdWages.GreaterThan(fc.HighIncomeThreshold) {
		excessWages := totalHouseholdWages.Sub(fc.HighIncomeThreshold)
		applicableExcess := decimal.Min(excessWages, wages)
		additionalMedicare = applicableExcess.Mul(fc.AdditionalRate)
	}
	
	return ssTax.Add(medicareTax).Add(additionalMedicare)
}

// ComprehensiveTaxCalculator handles all tax calculations
type ComprehensiveTaxCalculator struct {
	FederalTaxCalc    *FederalTaxCalculator
	StateTaxCalc      *PennsylvaniaTaxCalculator
	LocalTaxCalc      *UpperMakefieldEITCalculator
	FICATaxCalc       *FICACalculator
	SSTaxCalc         *SSTaxCalculator
}

// NewComprehensiveTaxCalculator creates a new comprehensive tax calculator
func NewComprehensiveTaxCalculator() *ComprehensiveTaxCalculator {
	return &ComprehensiveTaxCalculator{
		FederalTaxCalc:    NewFederalTaxCalculator2025(),
		StateTaxCalc:      NewPennsylvaniaTaxCalculator(),
		LocalTaxCalc:      NewUpperMakefieldEITCalculator(),
		FICATaxCalc:       NewFICACalculator2025(),
		SSTaxCalc:         NewSSTaxCalculator(),
	}
}

// CalculateTotalTaxes calculates all applicable taxes for a given income scenario
func (ctc *ComprehensiveTaxCalculator) CalculateTotalTaxes(income domain.TaxableIncome, isRetired bool, age1, age2 int, totalHouseholdWages decimal.Decimal) (decimal.Decimal, decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	// Calculate gross income for federal tax
	grossIncome := income.Salary.Add(income.FERSPension).Add(income.TSPWithdrawalsTrad).Add(income.TaxableSSBenefits).Add(income.OtherTaxableIncome)
	
	// Calculate federal tax
	federalTax := ctc.FederalTaxCalc.CalculateFederalTax(grossIncome, age1, age2)
	
	// Calculate state tax
	stateTax := ctc.StateTaxCalc.CalculateTax(income, isRetired)
	
	// Calculate local tax
	localTax := ctc.LocalTaxCalc.CalculateEIT(income.WageIncome, isRetired)
	
	// Calculate FICA (only applies to earned income, not retirement income)
	var ficaTax decimal.Decimal
	if !isRetired {
		ficaTax = ctc.FICATaxCalc.CalculateFICA(income.WageIncome, totalHouseholdWages)
	}
	
	return federalTax, stateTax, localTax, ficaTax
}

// CalculateTaxableIncome creates a TaxableIncome struct from cash flow data
func CalculateTaxableIncome(cashFlow domain.AnnualCashFlow, isRetired bool) domain.TaxableIncome {
	return domain.TaxableIncome{
		Salary:             decimal.Zero, // No salary in retirement
		FERSPension:        cashFlow.PensionRobert.Add(cashFlow.PensionDawn),
		TSPWithdrawalsTrad: cashFlow.TSPWithdrawalRobert.Add(cashFlow.TSPWithdrawalDawn), // Assuming all TSP withdrawals are from traditional
		TaxableSSBenefits:  cashFlow.SSBenefitRobert.Add(cashFlow.SSBenefitDawn), // Will be adjusted for taxation
		OtherTaxableIncome: decimal.Zero,
		WageIncome:         decimal.Zero, // No wages in retirement
		InterestIncome:     decimal.Zero, // Could be added if needed
	}
}

// CalculateCurrentTaxableIncome creates a TaxableIncome struct for current employment
func CalculateCurrentTaxableIncome(robertSalary, dawnSalary decimal.Decimal) domain.TaxableIncome {
	return domain.TaxableIncome{
		Salary:             robertSalary.Add(dawnSalary),
		FERSPension:        decimal.Zero,
		TSPWithdrawalsTrad: decimal.Zero,
		TaxableSSBenefits:  decimal.Zero,
		OtherTaxableIncome: decimal.Zero,
		WageIncome:         robertSalary.Add(dawnSalary),
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