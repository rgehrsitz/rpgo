package calculation

import (
	"fmt"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

const (
	// IRMAAWarningDistance is the threshold for warning status (within $10K of threshold)
	IRMAAWarningDistance = 10000
)

// CalculateMAGI calculates Modified Adjusted Gross Income for IRMAA purposes
// MAGI for IRMAA includes:
// - Wages and salaries
// - Pensions (FERS, etc.)
// - Traditional TSP withdrawals (not Roth)
// - Taxable portion of Social Security benefits
// - Tax-exempt interest (if applicable)
// - Other ordinary income
func CalculateMAGI(acf *domain.AnnualCashFlow) decimal.Decimal {
	magi := decimal.Zero

	// Add all salaries
	magi = magi.Add(acf.GetTotalSalary())

	// Add all pensions
	magi = magi.Add(acf.GetTotalPension())
	magi = magi.Add(acf.GetTotalSurvivorPension())

	// Add Traditional TSP withdrawals
	// Note: In a real implementation, we'd need to distinguish between
	// Traditional and Roth TSP withdrawals. For now, we'll assume
	// all TSP withdrawals are from Traditional accounts.
	magi = magi.Add(acf.GetTotalTSPWithdrawal())

	// Add FERS supplement (if any)
	magi = magi.Add(acf.GetTotalFERSSupplement())

	// Taxable portion of Social Security is already calculated in the tax engine
	// For IRMAA purposes, we need to add the taxable SS benefits
	// Note: The actual taxable SS calculation is complex and done elsewhere
	// For now, we'll use a simplified approach: 85% of SS benefits
	// In a production system, this should use the actual taxable amount from tax calculation
	taxableSSBenefits := acf.GetTotalSSBenefit().Mul(decimal.NewFromFloat(0.85))
	magi = magi.Add(taxableSSBenefits)

	return magi
}

// CalculateIRMAARiskStatus determines the IRMAA risk status for a given year
// based on MAGI and filing status
func CalculateIRMAARiskStatus(
	magi decimal.Decimal,
	isMarriedFilingJointly bool,
	mc *MedicareCalculator,
) (domain.IRMAARisk, string, decimal.Decimal, decimal.Decimal) {

	if len(mc.IRMAAThresholds) == 0 {
		return domain.IRMAARiskSafe, "None", decimal.Zero, decimal.Zero
	}

	// Find the first threshold
	firstThreshold := mc.IRMAAThresholds[0]
	var applicableThreshold decimal.Decimal
	if isMarriedFilingJointly {
		applicableThreshold = firstThreshold.IncomeThresholdJoint
	} else {
		applicableThreshold = firstThreshold.IncomeThresholdSingle
	}

	// Check if we're below the first threshold
	if magi.LessThan(applicableThreshold) {
		distanceToThreshold := applicableThreshold.Sub(magi)
		warningThreshold := decimal.NewFromInt(IRMAAWarningDistance)

		if distanceToThreshold.LessThanOrEqual(warningThreshold) {
			return domain.IRMAARiskWarning, "None", decimal.Zero, distanceToThreshold
		}
		return domain.IRMAARiskSafe, "None", decimal.Zero, distanceToThreshold
	}

	// We've exceeded at least one threshold - calculate surcharge
	tierLevel := ""
	surcharge := decimal.Zero
	nextThreshold := decimal.Zero

	for i, threshold := range mc.IRMAAThresholds {
		var incomeThreshold decimal.Decimal
		if isMarriedFilingJointly {
			incomeThreshold = threshold.IncomeThresholdJoint
		} else {
			incomeThreshold = threshold.IncomeThresholdSingle
		}

		if magi.GreaterThan(incomeThreshold) {
			surcharge = surcharge.Add(threshold.MonthlySurcharge)
			tierLevel = getTierName(i + 1)

			// Check if there's a next threshold
			if i+1 < len(mc.IRMAAThresholds) {
				nextThresholdData := mc.IRMAAThresholds[i+1]
				if isMarriedFilingJointly {
					nextThreshold = nextThresholdData.IncomeThresholdJoint
				} else {
					nextThreshold = nextThresholdData.IncomeThresholdSingle
				}
			}
		} else {
			// This is the next threshold we haven't exceeded
			nextThreshold = incomeThreshold
			break
		}
	}

	// Calculate distance to next threshold
	distanceToNext := decimal.Zero
	if !nextThreshold.IsZero() {
		distanceToNext = nextThreshold.Sub(magi)
	}

	return domain.IRMAARiskBreach, tierLevel, surcharge, distanceToNext
}

// getTierName returns a human-readable tier name
func getTierName(tier int) string {
	switch tier {
	case 1:
		return "Tier1"
	case 2:
		return "Tier2"
	case 3:
		return "Tier3"
	case 4:
		return "Tier4"
	case 5:
		return "Tier5"
	default:
		return "Unknown"
	}
}

// AnalyzeIRMAARisk performs a comprehensive IRMAA risk analysis across all projection years
func AnalyzeIRMAARisk(
	projection []domain.AnnualCashFlow,
	isMarriedFilingJointly bool,
	mc *MedicareCalculator,
) *domain.IRMAAAnalysis {

	analysis := &domain.IRMAAAnalysis{
		YearsWithBreaches: []int{},
		YearsWithWarnings: []int{},
		TotalIRMAACost:    decimal.Zero,
		FirstBreachYear:   0,
		HighRiskYears:     []domain.IRMAAYearRisk{},
		Recommendations:   []string{},
	}

	firstThreshold := decimal.Zero
	if len(mc.IRMAAThresholds) > 0 {
		if isMarriedFilingJointly {
			firstThreshold = mc.IRMAAThresholds[0].IncomeThresholdJoint
		} else {
			firstThreshold = mc.IRMAAThresholds[0].IncomeThresholdSingle
		}
	}

	for _, acf := range projection {
		// Skip years where participant is not Medicare-eligible
		if !acf.IsMedicareEligible {
			continue
		}

		magi := acf.MAGI
		if magi.IsZero() {
			// MAGI not calculated yet, calculate it
			magi = CalculateMAGI(&acf)
		}

		riskStatus, tierLevel, monthlySurcharge, distanceToNext := CalculateIRMAARiskStatus(
			magi,
			isMarriedFilingJointly,
			mc,
		)

		// Track breaches and warnings
		if riskStatus == domain.IRMAARiskBreach {
			analysis.YearsWithBreaches = append(analysis.YearsWithBreaches, acf.Year)
			if analysis.FirstBreachYear == 0 {
				analysis.FirstBreachYear = acf.Year
			}

			// Calculate annual cost (surcharge is per person per month)
			// For married couples, both pay the surcharge
			personsCount := decimal.NewFromInt(1)
			if isMarriedFilingJointly {
				personsCount = decimal.NewFromInt(2)
			}
			annualCost := monthlySurcharge.Mul(decimal.NewFromInt(12)).Mul(personsCount)
			analysis.TotalIRMAACost = analysis.TotalIRMAACost.Add(annualCost)

			// Add to high risk years
			analysis.HighRiskYears = append(analysis.HighRiskYears, domain.IRMAAYearRisk{
				Year:                acf.Year,
				MAGI:                magi,
				Threshold:           firstThreshold,
				DistanceToThreshold: distanceToNext.Neg(), // Negative because we're over
				RiskStatus:          riskStatus,
				TierLevel:           tierLevel,
				MonthlySurcharge:    monthlySurcharge,
				AnnualCost:          annualCost,
			})
		} else if riskStatus == domain.IRMAARiskWarning {
			analysis.YearsWithWarnings = append(analysis.YearsWithWarnings, acf.Year)

			// Add to high risk years
			analysis.HighRiskYears = append(analysis.HighRiskYears, domain.IRMAAYearRisk{
				Year:                acf.Year,
				MAGI:                magi,
				Threshold:           firstThreshold,
				DistanceToThreshold: distanceToNext,
				RiskStatus:          riskStatus,
				TierLevel:           tierLevel,
				MonthlySurcharge:    decimal.Zero,
				AnnualCost:          decimal.Zero,
			})
		}
	}

	// Generate recommendations
	analysis.Recommendations = generateIRMAARecommendations(analysis, firstThreshold)

	return analysis
}

// generateIRMAARecommendations generates actionable recommendations based on IRMAA analysis
func generateIRMAARecommendations(analysis *domain.IRMAAAnalysis, threshold decimal.Decimal) []string {
	recommendations := []string{}

	if len(analysis.YearsWithBreaches) > 0 {
		// Calculate total cost (average per breach year can be derived if needed)
		recommendations = append(recommendations,
			"⚠️  IRMAA breaches detected - consider strategies to reduce MAGI")

		// Specific cost-based recommendations
		if analysis.TotalIRMAACost.GreaterThan(decimal.NewFromInt(10000)) {
			recommendations = append(recommendations,
				fmt.Sprintf("💰 High IRMAA cost ($%.0f over %d years) - significant savings possible through optimization",
					analysis.TotalIRMAACost.InexactFloat64(), len(analysis.YearsWithBreaches)))
		}

		// Check if breaches are early or late in retirement
		if analysis.FirstBreachYear <= 5 {
			recommendations = append(recommendations,
				"💡 Breaches occur early - consider Roth conversions BEFORE retirement to reduce future MAGI")
		} else {
			recommendations = append(recommendations,
				"💡 Breaches occur mid-retirement - review Social Security timing and TSP withdrawal strategy")
		}

		// Check for consecutive breaches
		consecutiveYears := countConsecutiveYears(analysis.YearsWithBreaches)
		if consecutiveYears >= 3 {
			recommendations = append(recommendations,
				fmt.Sprintf("📊 %d consecutive breach years detected - systematic withdrawal strategy change recommended", consecutiveYears))
		}

		// General strategies
		recommendations = append(recommendations,
			"💡 Withdraw from Roth TSP instead of Traditional TSP in breach years (Roth doesn't count toward MAGI)")

		recommendations = append(recommendations,
			"💡 Time large Traditional TSP withdrawals for low-income years (before SS starts or after RMDs begin)")

		// Check severity of breaches
		maxCost := decimal.Zero
		for _, yr := range analysis.HighRiskYears {
			if yr.AnnualCost.GreaterThan(maxCost) {
				maxCost = yr.AnnualCost
			}
		}
		if maxCost.GreaterThan(decimal.NewFromInt(5000)) {
			recommendations = append(recommendations,
				"⚠️  Peak IRMAA cost exceeds $5,000/year - consider delaying Social Security or reducing TSP withdrawal rate")
		}

	} else if len(analysis.YearsWithWarnings) > 0 {
		// Find how close to threshold
		minDistance := decimal.NewFromInt(10000)
		for _, yr := range analysis.HighRiskYears {
			if yr.RiskStatus == domain.IRMAARiskWarning && yr.DistanceToThreshold.LessThan(minDistance) {
				minDistance = yr.DistanceToThreshold
			}
		}

		recommendations = append(recommendations,
			"⚠️  Close to IRMAA thresholds - monitor MAGI carefully")

		if minDistance.LessThan(decimal.NewFromInt(5000)) {
			recommendations = append(recommendations,
				fmt.Sprintf("💡 Only $%.0f away from breach - small TSP withdrawal adjustments could prevent surcharges",
					minDistance.InexactFloat64()))
		} else {
			recommendations = append(recommendations,
				"💡 Moderate buffer to thresholds - maintain current withdrawal strategy but monitor annually")
		}

		recommendations = append(recommendations,
			"💡 Consider maintaining emergency Roth TSP reserve for years approaching thresholds")

	} else {
		recommendations = append(recommendations,
			"✓ No IRMAA concerns - MAGI remains comfortably below thresholds")

		recommendations = append(recommendations,
			"💡 Current income strategy avoids Medicare premium surcharges - no changes needed for IRMAA")

		recommendations = append(recommendations,
			"📈 You have flexibility to increase withdrawals if needed without triggering IRMAA")
	}

	return recommendations
}

// countConsecutiveYears finds the longest sequence of consecutive years
func countConsecutiveYears(years []int) int {
	if len(years) == 0 {
		return 0
	}

	// Sort years first
	sortedYears := make([]int, len(years))
	copy(sortedYears, years)
	for i := 0; i < len(sortedYears)-1; i++ {
		for j := i + 1; j < len(sortedYears); j++ {
			if sortedYears[i] > sortedYears[j] {
				sortedYears[i], sortedYears[j] = sortedYears[j], sortedYears[i]
			}
		}
	}

	maxConsecutive := 1
	currentConsecutive := 1

	for i := 1; i < len(sortedYears); i++ {
		if sortedYears[i] == sortedYears[i-1]+1 {
			currentConsecutive++
			if currentConsecutive > maxConsecutive {
				maxConsecutive = currentConsecutive
			}
		} else {
			currentConsecutive = 1
		}
	}

	return maxConsecutive
}
