package domain

import (
	"github.com/shopspring/decimal"
)

// HealthcareConfig represents healthcare coverage configuration for a participant
type HealthcareConfig struct {
	// Pre-Medicare (before age 65)
	PreMedicareCoverage       string          `yaml:"pre_medicare_coverage" json:"pre_medicare_coverage"`               // fehb | cobra | marketplace | retiree_plan
	PreMedicareMonthlyPremium decimal.Decimal `yaml:"pre_medicare_monthly_premium" json:"pre_medicare_monthly_premium"` // If not FEHB

	// Medicare (age 65+)
	MedicarePartB     bool   `yaml:"medicare_part_b" json:"medicare_part_b"`           // Default true
	MedicarePartD     bool   `yaml:"medicare_part_d" json:"medicare_part_d"`           // Default true
	MedicarePartDPlan string `yaml:"medicare_part_d_plan" json:"medicare_part_d_plan"` // standard | enhanced
	MedigapPlan       string `yaml:"medigap_plan" json:"medigap_plan"`                 // A-N, or none

	// Transition
	DropFEHBAt65 bool `yaml:"drop_fehb_at_65" json:"drop_fehb_at_65"` // Stop FEHB when Medicare eligible
}

// HealthcareCostBreakdown provides detailed breakdown of healthcare costs
type HealthcareCostBreakdown struct {
	FEHBPremium        decimal.Decimal `json:"fehbPremium"`        // FEHB premium (if applicable)
	MarketplacePremium decimal.Decimal `json:"marketplacePremium"` // Marketplace/COBRA premium
	MedicarePartB      decimal.Decimal `json:"medicarePartB"`      // Medicare Part B premium + IRMAA
	MedicarePartD      decimal.Decimal `json:"medicarePartD"`      // Medicare Part D premium + IRMAA
	Medigap            decimal.Decimal `json:"medigap"`            // Medigap premium
	Total              decimal.Decimal `json:"total"`              // Total healthcare cost
}

// HealthcareInflationRates represents inflation rates for different healthcare cost types
type HealthcareInflationRates struct {
	FEHB        decimal.Decimal `yaml:"fehb" json:"fehb"`               // FEHB premium inflation
	MedicareB   decimal.Decimal `yaml:"medicare_b" json:"medicare_b"`   // Medicare Part B inflation
	MedicareD   decimal.Decimal `yaml:"medicare_d" json:"medicare_d"`   // Medicare Part D inflation
	Medigap     decimal.Decimal `yaml:"medigap" json:"medigap"`         // Medigap inflation
	Marketplace decimal.Decimal `yaml:"marketplace" json:"marketplace"` // Marketplace inflation
}

// MedicarePartDCosts represents Medicare Part D premium costs
type MedicarePartDCosts struct {
	StandardBasePremium decimal.Decimal `yaml:"standard_base_premium" json:"standard_base_premium"` // ~$35/month
	EnhancedBasePremium decimal.Decimal `yaml:"enhanced_base_premium" json:"enhanced_base_premium"` // ~$50/month
}

// MedicarePartDIRMAA represents Medicare Part D IRMAA surcharges (same thresholds as Part B)
type MedicarePartDIRMAA struct {
	IncomeThresholdSingle decimal.Decimal `yaml:"income_threshold_single" json:"income_threshold_single"`
	IncomeThresholdJoint  decimal.Decimal `yaml:"income_threshold_joint" json:"income_threshold_joint"`
	MonthlySurcharge      decimal.Decimal `yaml:"monthly_surcharge" json:"monthly_surcharge"`
}

// MedigapCosts represents Medigap plan costs by plan type and age
type MedigapCosts struct {
	PlanType string                  `yaml:"plan_type" json:"plan_type"` // A, B, C, D, F, G, K, L, M, N
	BaseCost decimal.Decimal         `yaml:"base_cost" json:"base_cost"` // Monthly base cost
	AgeRates map[int]decimal.Decimal `yaml:"age_rates" json:"age_rates"` // Age-specific multipliers
}

// DefaultHealthcareConfig returns a default healthcare configuration
func DefaultHealthcareConfig() HealthcareConfig {
	return HealthcareConfig{
		PreMedicareCoverage:       "fehb",
		PreMedicareMonthlyPremium: decimal.Zero,
		MedicarePartB:             true,
		MedicarePartD:             true,
		MedicarePartDPlan:         "standard",
		MedigapPlan:               "G",
		DropFEHBAt65:              true,
	}
}

// DefaultHealthcareInflationRates returns default healthcare inflation rates
func DefaultHealthcareInflationRates() HealthcareInflationRates {
	return HealthcareInflationRates{
		FEHB:        decimal.NewFromFloat(0.06), // 6% annual increase
		MedicareB:   decimal.NewFromFloat(0.05), // 5% annual increase
		MedicareD:   decimal.NewFromFloat(0.05), // 5% annual increase
		Medigap:     decimal.NewFromFloat(0.04), // 4% annual increase
		Marketplace: decimal.NewFromFloat(0.07), // 7% annual increase
	}
}

// DefaultMedicarePartDCosts returns default Medicare Part D costs for 2025
func DefaultMedicarePartDCosts() MedicarePartDCosts {
	return MedicarePartDCosts{
		StandardBasePremium: decimal.NewFromFloat(35.0), // $35/month
		EnhancedBasePremium: decimal.NewFromFloat(50.0), // $50/month
	}
}

// DefaultMedicarePartDIRMAA returns default Medicare Part D IRMAA thresholds for 2025
func DefaultMedicarePartDIRMAA() []MedicarePartDIRMAA {
	return []MedicarePartDIRMAA{
		{
			IncomeThresholdSingle: decimal.NewFromInt(103000),
			IncomeThresholdJoint:  decimal.NewFromInt(206000),
			MonthlySurcharge:      decimal.NewFromFloat(12.90),
		},
		{
			IncomeThresholdSingle: decimal.NewFromInt(129000),
			IncomeThresholdJoint:  decimal.NewFromInt(258000),
			MonthlySurcharge:      decimal.NewFromFloat(33.20),
		},
		{
			IncomeThresholdSingle: decimal.NewFromInt(161000),
			IncomeThresholdJoint:  decimal.NewFromInt(322000),
			MonthlySurcharge:      decimal.NewFromFloat(53.50),
		},
		{
			IncomeThresholdSingle: decimal.NewFromInt(193000),
			IncomeThresholdJoint:  decimal.NewFromInt(386000),
			MonthlySurcharge:      decimal.NewFromFloat(73.80),
		},
		{
			IncomeThresholdSingle: decimal.NewFromInt(500000),
			IncomeThresholdJoint:  decimal.NewFromInt(750000),
			MonthlySurcharge:      decimal.NewFromFloat(81.90),
		},
	}
}

// DefaultMedigapCosts returns default Medigap costs for Plan G (typical)
func DefaultMedigapCosts() MedigapCosts {
	return MedigapCosts{
		PlanType: "G",
		BaseCost: decimal.NewFromFloat(200.0), // $200/month base
		AgeRates: map[int]decimal.Decimal{
			65: decimal.NewFromFloat(1.0), // 100% at age 65
			70: decimal.NewFromFloat(1.1), // 110% at age 70
			75: decimal.NewFromFloat(1.2), // 120% at age 75
			80: decimal.NewFromFloat(1.3), // 130% at age 80
			85: decimal.NewFromFloat(1.4), // 140% at age 85
			90: decimal.NewFromFloat(1.5), // 150% at age 90+
		},
	}
}
