package calculation

import (
	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// HealthcareCostCalculator handles comprehensive healthcare cost calculations
type HealthcareCostCalculator struct {
	MedicareCalc   *MedicareCalculator
	PartDCosts     domain.MedicarePartDCosts
	PartDIRMAA     []domain.MedicarePartDIRMAA
	MedigapCosts   domain.MedigapCosts
	InflationRates domain.HealthcareInflationRates
}

// NewHealthcareCostCalculator creates a new healthcare cost calculator with default values
func NewHealthcareCostCalculator() *HealthcareCostCalculator {
	return &HealthcareCostCalculator{
		MedicareCalc:   NewMedicareCalculator(),
		PartDCosts:     domain.DefaultMedicarePartDCosts(),
		PartDIRMAA:     domain.DefaultMedicarePartDIRMAA(),
		MedigapCosts:   domain.DefaultMedigapCosts(),
		InflationRates: domain.DefaultHealthcareInflationRates(),
	}
}

// NewHealthcareCostCalculatorWithConfig creates a healthcare cost calculator with configurable values
func NewHealthcareCostCalculatorWithConfig(
	medicareConfig domain.MedicareConfig,
	partDCosts domain.MedicarePartDCosts,
	partDIRMAA []domain.MedicarePartDIRMAA,
	medigapCosts domain.MedigapCosts,
	inflationRates domain.HealthcareInflationRates,
) *HealthcareCostCalculator {
	return &HealthcareCostCalculator{
		MedicareCalc:   NewMedicareCalculatorWithConfig(medicareConfig),
		PartDCosts:     partDCosts,
		PartDIRMAA:     partDIRMAA,
		MedigapCosts:   medigapCosts,
		InflationRates: inflationRates,
	}
}

// CalculateHealthcareCosts calculates comprehensive healthcare costs for a participant
func (hcc *HealthcareCostCalculator) CalculateHealthcareCosts(
	participant *domain.Participant,
	age int,
	year int,
	magi decimal.Decimal,
	filingStatus string,
) domain.HealthcareCostBreakdown {

	breakdown := domain.HealthcareCostBreakdown{}

	// Use default healthcare config if not specified
	healthcare := participant.Healthcare
	if healthcare == nil {
		defaultConfig := domain.DefaultHealthcareConfig()
		healthcare = &defaultConfig
	}

	if age < 65 {
		// Pre-Medicare coverage
		hcc.calculatePreMedicareCosts(participant, healthcare, year, &breakdown)
	} else {
		// Medicare coverage (age 65+)
		hcc.calculateMedicareCosts(participant, healthcare, age, year, magi, filingStatus, &breakdown)
	}

	// Calculate total
	breakdown.Total = breakdown.FEHBPremium.
		Add(breakdown.MarketplacePremium).
		Add(breakdown.MedicarePartB).
		Add(breakdown.MedicarePartD).
		Add(breakdown.Medigap)

	return breakdown
}

// calculatePreMedicareCosts calculates healthcare costs before Medicare eligibility
func (hcc *HealthcareCostCalculator) calculatePreMedicareCosts(
	participant *domain.Participant,
	healthcare *domain.HealthcareConfig,
	year int,
	breakdown *domain.HealthcareCostBreakdown,
) {
	switch healthcare.PreMedicareCoverage {
	case "fehb":
		// Use FEHB premium from participant
		if participant.FEHBPremiumPerPayPeriod != nil {
			basePremium := participant.FEHBPremiumPerPayPeriod.Mul(decimal.NewFromInt(26)) // 26 pay periods
			inflatedPremium := hcc.inflateFromBase(basePremium, year, hcc.InflationRates.FEHB)
			breakdown.FEHBPremium = inflatedPremium
		}
	case "marketplace":
		// Use marketplace premium
		basePremium := healthcare.PreMedicareMonthlyPremium.Mul(decimal.NewFromInt(12))
		inflatedPremium := hcc.inflateFromBase(basePremium, year, hcc.InflationRates.Marketplace)
		breakdown.MarketplacePremium = inflatedPremium
	case "cobra":
		// COBRA typically costs more than FEHB
		if participant.FEHBPremiumPerPayPeriod != nil {
			basePremium := participant.FEHBPremiumPerPayPeriod.Mul(decimal.NewFromInt(26)).Mul(decimal.NewFromFloat(1.5)) // 150% of FEHB
			inflatedPremium := hcc.inflateFromBase(basePremium, year, hcc.InflationRates.Marketplace)
			breakdown.MarketplacePremium = inflatedPremium
		}
	case "retiree_plan":
		// Retiree plan (if available)
		if participant.FEHBPremiumPerPayPeriod != nil {
			basePremium := participant.FEHBPremiumPerPayPeriod.Mul(decimal.NewFromInt(26))
			inflatedPremium := hcc.inflateFromBase(basePremium, year, hcc.InflationRates.FEHB)
			breakdown.FEHBPremium = inflatedPremium
		}
	}
}

// calculateMedicareCosts calculates healthcare costs after Medicare eligibility
func (hcc *HealthcareCostCalculator) calculateMedicareCosts(
	participant *domain.Participant,
	healthcare *domain.HealthcareConfig,
	age int,
	year int,
	magi decimal.Decimal,
	filingStatus string,
	breakdown *domain.HealthcareCostBreakdown,
) {
	isMarried := filingStatus == "married_filing_jointly"

	// Medicare Part B
	if healthcare.MedicarePartB {
		basePremium := decimal.NewFromFloat(174.70) // 2025 standard Part B premium
		annualBasePremium := basePremium.Mul(decimal.NewFromInt(12))
		inflatedPremium := hcc.inflateFromBase(annualBasePremium, year, hcc.InflationRates.MedicareB)

		// Add IRMAA surcharge
		irmaaSurcharge := hcc.MedicareCalc.calculateIRMAASurcharge(magi, isMarried)
		annualIRMAA := irmaaSurcharge.Mul(decimal.NewFromInt(12))

		breakdown.MedicarePartB = inflatedPremium.Add(annualIRMAA)
	}

	// Medicare Part D
	if healthcare.MedicarePartD {
		var basePremium decimal.Decimal
		switch healthcare.MedicarePartDPlan {
		case "standard":
			basePremium = hcc.PartDCosts.StandardBasePremium
		case "enhanced":
			basePremium = hcc.PartDCosts.EnhancedBasePremium
		default:
			basePremium = hcc.PartDCosts.StandardBasePremium
		}

		annualBasePremium := basePremium.Mul(decimal.NewFromInt(12))
		inflatedPremium := hcc.inflateFromBase(annualBasePremium, year, hcc.InflationRates.MedicareD)

		// Add Part D IRMAA surcharge
		partDIRMAA := hcc.calculatePartDIRMAA(magi, isMarried)
		annualPartDIRMAA := partDIRMAA.Mul(decimal.NewFromInt(12))

		breakdown.MedicarePartD = inflatedPremium.Add(annualPartDIRMAA)
	}

	// Medigap
	if healthcare.MedigapPlan != "" {
		baseCost := hcc.getMedigapBaseCost(healthcare.MedigapPlan, age)
		annualBaseCost := baseCost.Mul(decimal.NewFromInt(12))
		inflatedCost := hcc.inflateFromBase(annualBaseCost, year, hcc.InflationRates.Medigap)
		breakdown.Medigap = inflatedCost
	}

	// FEHB (if not dropped at 65)
	if !healthcare.DropFEHBAt65 && participant.FEHBPremiumPerPayPeriod != nil {
		basePremium := participant.FEHBPremiumPerPayPeriod.Mul(decimal.NewFromInt(26))
		inflatedPremium := hcc.inflateFromBase(basePremium, year, hcc.InflationRates.FEHB)
		breakdown.FEHBPremium = inflatedPremium
	}
}

// calculatePartDIRMAA calculates Medicare Part D IRMAA surcharge
func (hcc *HealthcareCostCalculator) calculatePartDIRMAA(magi decimal.Decimal, isMarried bool) decimal.Decimal {
	var totalSurcharge decimal.Decimal

	for _, threshold := range hcc.PartDIRMAA {
		var incomeThreshold decimal.Decimal
		if isMarried {
			incomeThreshold = threshold.IncomeThresholdJoint
		} else {
			incomeThreshold = threshold.IncomeThresholdSingle
		}

		if magi.GreaterThan(incomeThreshold) {
			totalSurcharge = totalSurcharge.Add(threshold.MonthlySurcharge)
		} else {
			break
		}
	}

	return totalSurcharge
}

// getMedigapBaseCost gets the base cost for a Medigap plan at a specific age
func (hcc *HealthcareCostCalculator) getMedigapBaseCost(planType string, age int) decimal.Decimal {
	// For now, use Plan G costs as default
	baseCost := hcc.MedigapCosts.BaseCost

	// Apply age-based multiplier
	ageMultiplier := decimal.NewFromFloat(1.0)
	for ageThreshold, multiplier := range hcc.MedigapCosts.AgeRates {
		if age >= ageThreshold {
			ageMultiplier = multiplier
		}
	}

	return baseCost.Mul(ageMultiplier)
}

// inflateFromBase applies inflation to a base amount
func (hcc *HealthcareCostCalculator) inflateFromBase(baseAmount decimal.Decimal, year int, inflationRate decimal.Decimal) decimal.Decimal {
	baseYear := 2025
	yearsFromBase := year - baseYear

	if yearsFromBase <= 0 {
		return baseAmount
	}

	inflationFactor := decimal.NewFromFloat(1).Add(inflationRate).Pow(decimal.NewFromInt(int64(yearsFromBase)))
	return baseAmount.Mul(inflationFactor)
}

// CalculateHouseholdHealthcareCosts calculates total healthcare costs for a household
func (hcc *HealthcareCostCalculator) CalculateHouseholdHealthcareCosts(
	participants []domain.Participant,
	ages map[string]int,
	year int,
	magi decimal.Decimal,
	filingStatus string,
) domain.HealthcareCostBreakdown {

	householdBreakdown := domain.HealthcareCostBreakdown{}

	for _, participant := range participants {
		age := ages[participant.Name]
		participantBreakdown := hcc.CalculateHealthcareCosts(&participant, age, year, magi, filingStatus)

		householdBreakdown.FEHBPremium = householdBreakdown.FEHBPremium.Add(participantBreakdown.FEHBPremium)
		householdBreakdown.MarketplacePremium = householdBreakdown.MarketplacePremium.Add(participantBreakdown.MarketplacePremium)
		householdBreakdown.MedicarePartB = householdBreakdown.MedicarePartB.Add(participantBreakdown.MedicarePartB)
		householdBreakdown.MedicarePartD = householdBreakdown.MedicarePartD.Add(participantBreakdown.MedicarePartD)
		householdBreakdown.Medigap = householdBreakdown.Medigap.Add(participantBreakdown.Medigap)
	}

	householdBreakdown.Total = householdBreakdown.FEHBPremium.
		Add(householdBreakdown.MarketplacePremium).
		Add(householdBreakdown.MedicarePartB).
		Add(householdBreakdown.MedicarePartD).
		Add(householdBreakdown.Medigap)

	return householdBreakdown
}
