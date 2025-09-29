package domain

import (
	"github.com/shopspring/decimal"
)

// RegulatoryConfig contains all regulatory/legal data that applies uniformly
// This is loaded from regulatory.yaml and merged with personal scenario configs
type RegulatoryConfig struct {
	Metadata        RegulatoryMetadata    `yaml:"metadata" json:"metadata"`
	FederalTax      FederalTaxRules       `yaml:"federal_tax" json:"federal_tax"`
	FICA            FICARules             `yaml:"fica" json:"fica"`
	SocialSecurity  SocialSecurityRulesReg   `yaml:"social_security" json:"social_security"`
	Medicare        MedicareRules         `yaml:"medicare" json:"medicare"`
	States          map[string]StateRules `yaml:"states" json:"states"`
	TSPFunds        TSPFundData           `yaml:"tsp_funds" json:"tsp_funds"`
	FERS            FERSRules             `yaml:"fers" json:"fers"`
	FEHB            FEHBRules             `yaml:"fehb" json:"fehb"`
	MonteCarlo      MonteCarloDefaults    `yaml:"monte_carlo" json:"monte_carlo"`
}

// RegulatoryMetadata contains information about the regulatory data
type RegulatoryMetadata struct {
	DataYear     int    `yaml:"data_year" json:"data_year"`
	LastUpdated  string `yaml:"last_updated" json:"last_updated"`
	Description  string `yaml:"description" json:"description"`
}

// FederalTaxRules contains federal income tax rules
type FederalTaxRules struct {
	StandardDeduction        StandardDeductions `yaml:"standard_deduction" json:"standard_deduction"`
	AdditionalDeduction65Plus decimal.Decimal   `yaml:"additional_deduction_65_plus" json:"additional_deduction_65_plus"`
	BracketsMFJ              []TaxBracket       `yaml:"brackets_married_filing_jointly" json:"brackets_married_filing_jointly"`
}

// StandardDeductions contains standard deduction amounts by filing status
type StandardDeductions struct {
	MarriedFilingJointly decimal.Decimal `yaml:"married_filing_jointly" json:"married_filing_jointly"`
	Single               decimal.Decimal `yaml:"single" json:"single"`
	HeadOfHousehold      decimal.Decimal `yaml:"head_of_household" json:"head_of_household"`
}

// FICARules contains FICA tax rules
type FICARules struct {
	SocialSecurity SocialSecurityFICA `yaml:"social_security" json:"social_security"`
	Medicare       MedicareFICA       `yaml:"medicare" json:"medicare"`
}

// SocialSecurityFICA contains Social Security FICA rules
type SocialSecurityFICA struct {
	Rate     decimal.Decimal `yaml:"rate" json:"rate"`
	WageBase decimal.Decimal `yaml:"wage_base" json:"wage_base"`
}

// MedicareFICA contains Medicare FICA rules
type MedicareFICA struct {
	Rate                    decimal.Decimal `yaml:"rate" json:"rate"`
	AdditionalRate          decimal.Decimal `yaml:"additional_rate" json:"additional_rate"`
	HighIncomeThresholdMFJ  decimal.Decimal `yaml:"high_income_threshold_mfj" json:"high_income_threshold_mfj"`
}

// MedicareRules contains Medicare Part B premium rules
type MedicareRules struct {
	PartBBasePremium decimal.Decimal           `yaml:"part_b_base_premium" json:"part_b_base_premium"`
	IRMAAThresholds  []MedicareIRMAAThreshold `yaml:"irmaa_tiers" json:"irmaa_tiers"`
}

// StateRules contains state-specific tax rules
type StateRules struct {
	Rate                   decimal.Decimal `yaml:"rate" json:"rate"`
	PensionExemption       bool            `yaml:"pension_exemption" json:"pension_exemption"`
	SocialSecurityExemption bool           `yaml:"social_security_exemption" json:"social_security_exemption"`
}

// TSPFundData contains historical performance data for TSP funds
type TSPFundData struct {
	CFund TSPFundPerformance `yaml:"c_fund" json:"c_fund"`
	SFund TSPFundPerformance `yaml:"s_fund" json:"s_fund"`
	IFund TSPFundPerformance `yaml:"i_fund" json:"i_fund"`
	FFund TSPFundPerformance `yaml:"f_fund" json:"f_fund"`
	GFund TSPFundPerformance `yaml:"g_fund" json:"g_fund"`
}

// TSPFundPerformance contains performance data for a single TSP fund
type TSPFundPerformance struct {
	Mean        decimal.Decimal `yaml:"mean" json:"mean"`
	StandardDev decimal.Decimal `yaml:"standard_dev" json:"standard_dev"`
	DataSource  string          `yaml:"data_source" json:"data_source"`
	LastUpdated string          `yaml:"last_updated" json:"last_updated"`
}

// FEHBRules contains FEHB program rules
type FEHBRules struct {
	PayPeriodsPerYear              int             `yaml:"pay_periods_per_year" json:"pay_periods_per_year"`
	RetirementCalculationMethod    string          `yaml:"retirement_calculation_method" json:"retirement_calculation_method"`
	RetirementPremiumMultiplier    decimal.Decimal `yaml:"retirement_premium_multiplier" json:"retirement_premium_multiplier"`
}

// MonteCarloDefaults contains default Monte Carlo simulation parameters
type MonteCarloDefaults struct {
	TSPReturnVariability decimal.Decimal       `yaml:"tsp_return_variability" json:"tsp_return_variability"`
	InflationVariability decimal.Decimal       `yaml:"inflation_variability" json:"inflation_variability"`
	COLAVariability      decimal.Decimal       `yaml:"cola_variability" json:"cola_variability"`
	FEHBVariability      decimal.Decimal       `yaml:"fehb_variability" json:"fehb_variability"`
	MaxReasonableIncome  decimal.Decimal       `yaml:"max_reasonable_income" json:"max_reasonable_income"`
	DefaultTSPAllocation DefaultTSPAllocation  `yaml:"default_tsp_allocation" json:"default_tsp_allocation"`
}

// DefaultTSPAllocation contains default TSP fund allocation
type DefaultTSPAllocation struct {
	CFund decimal.Decimal `yaml:"c_fund" json:"c_fund"`
	SFund decimal.Decimal `yaml:"s_fund" json:"s_fund"`
	IFund decimal.Decimal `yaml:"i_fund" json:"i_fund"`
	FFund decimal.Decimal `yaml:"f_fund" json:"f_fund"`
	GFund decimal.Decimal `yaml:"g_fund" json:"g_fund"`
}

// SocialSecurityRulesReg contains Social Security benefit and taxation rules
type SocialSecurityRulesReg struct {
	TaxationThresholds  SocialSecurityTaxThresholdsReg `yaml:"taxation_thresholds" json:"taxation_thresholds"`
	BenefitAdjustments  SocialSecurityBenefitRules     `yaml:"benefit_adjustments" json:"benefit_adjustments"`
}

// SocialSecurityTaxThresholdsReg contains income thresholds for SS taxation
type SocialSecurityTaxThresholdsReg struct {
	MarriedFilingJointly TaxThreshold `yaml:"married_filing_jointly" json:"married_filing_jointly"`
	Single               TaxThreshold `yaml:"single" json:"single"`
}

// TaxThreshold contains the two SS taxation thresholds
type TaxThreshold struct {
	Threshold1 decimal.Decimal `yaml:"threshold_1" json:"threshold_1"`
	Threshold2 decimal.Decimal `yaml:"threshold_2" json:"threshold_2"`
}

// SocialSecurityBenefitRules contains benefit calculation adjustments
type SocialSecurityBenefitRules struct {
	EarlyRetirementReduction EarlyRetirementRates `yaml:"early_retirement_reduction" json:"early_retirement_reduction"`
	DelayedRetirementCredit  decimal.Decimal      `yaml:"delayed_retirement_credit" json:"delayed_retirement_credit"`
}

// EarlyRetirementRates contains the rates for early retirement reduction
type EarlyRetirementRates struct {
	First36MonthsRate    decimal.Decimal `yaml:"first_36_months_rate" json:"first_36_months_rate"`
	AdditionalMonthsRate decimal.Decimal `yaml:"additional_months_rate" json:"additional_months_rate"`
}