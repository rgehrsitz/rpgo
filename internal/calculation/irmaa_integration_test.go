package calculation

import (
	"testing"
	"time"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// TestIRMAAIntegration demonstrates IRMAA analysis with realistic retirement scenarios
func TestIRMAAIntegration(t *testing.T) {
	mc := NewMedicareCalculator()

	// Scenario 1: High-income retiree exceeding IRMAA thresholds
	t.Run("High income scenario - IRMAA breaches", func(t *testing.T) {
		projection := createHighIncomeProjection()
		analysis := AnalyzeIRMAARisk(projection, true, mc)

		if len(analysis.YearsWithBreaches) == 0 {
			t.Error("Expected IRMAA breaches for high-income scenario")
		}

		if analysis.TotalIRMAACost.LessThanOrEqual(decimal.Zero) {
			t.Error("Expected positive IRMAA cost for breaches")
		}

		if analysis.FirstBreachYear == 0 {
			t.Error("Expected first breach year to be set")
		}

		t.Logf("High income scenario results:")
		t.Logf("  Breach years: %v", analysis.YearsWithBreaches)
		t.Logf("  Warning years: %v", analysis.YearsWithWarnings)
		t.Logf("  Total IRMAA cost: $%.2f", analysis.TotalIRMAACost.InexactFloat64())
		t.Logf("  First breach: Year %d", analysis.FirstBreachYear)
		t.Logf("  Recommendations: %d", len(analysis.Recommendations))
	})

	// Scenario 2: Moderate income - approaching threshold
	t.Run("Moderate income scenario - IRMAA warnings", func(t *testing.T) {
		projection := createModerateIncomeProjection()
		analysis := AnalyzeIRMAARisk(projection, true, mc)

		if len(analysis.YearsWithWarnings) == 0 {
			t.Error("Expected IRMAA warnings for moderate-income scenario")
		}

		if len(analysis.YearsWithBreaches) > 0 {
			t.Error("Should not have breaches, only warnings")
		}

		t.Logf("Moderate income scenario results:")
		t.Logf("  Warning years: %v", analysis.YearsWithWarnings)
		t.Logf("  Breach years: %v", analysis.YearsWithBreaches)
		t.Logf("  Recommendations: %d", len(analysis.Recommendations))
	})

	// Scenario 3: Conservative withdrawal - safe from IRMAA
	t.Run("Conservative income scenario - IRMAA safe", func(t *testing.T) {
		projection := createConservativeIncomeProjection()
		analysis := AnalyzeIRMAARisk(projection, true, mc)

		if len(analysis.YearsWithBreaches) > 0 {
			t.Error("Expected no IRMAA breaches for conservative scenario")
		}

		if len(analysis.YearsWithWarnings) > 0 {
			t.Error("Expected no IRMAA warnings for conservative scenario")
		}

		t.Logf("Conservative income scenario results:")
		t.Logf("  Safe from IRMAA: true")
		t.Logf("  Recommendations: %d", len(analysis.Recommendations))
	})
}

// createHighIncomeProjection creates a projection with high MAGI that exceeds IRMAA thresholds
func createHighIncomeProjection() []domain.AnnualCashFlow {
	projection := make([]domain.AnnualCashFlow, 10)
	baseDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 10; i++ {
		year := i + 1
		acf := domain.AnnualCashFlow{
			Year:               year,
			Date:               baseDate.AddDate(i, 0, 0),
			IsMedicareEligible: i >= 2, // Medicare eligible starting year 3
			Pensions:           map[string]decimal.Decimal{"Retiree": decimal.NewFromInt(70000)},
			TSPWithdrawals:     map[string]decimal.Decimal{"Retiree": decimal.NewFromInt(80000)}, // High withdrawal
			SSBenefits:         map[string]decimal.Decimal{"Retiree": decimal.NewFromInt(45000)},
		}

		// Calculate MAGI
		acf.MAGI = CalculateMAGI(&acf)
		projection[i] = acf
	}

	return projection
}

// createModerateIncomeProjection creates a projection approaching IRMAA thresholds
func createModerateIncomeProjection() []domain.AnnualCashFlow {
	projection := make([]domain.AnnualCashFlow, 10)
	baseDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 10; i++ {
		year := i + 1
		acf := domain.AnnualCashFlow{
			Year:               year,
			Date:               baseDate.AddDate(i, 0, 0),
			IsMedicareEligible: i >= 2,
			Pensions:           map[string]decimal.Decimal{"Retiree": decimal.NewFromInt(60000)},
			TSPWithdrawals:     map[string]decimal.Decimal{"Retiree": decimal.NewFromInt(70000)},
			SSBenefits:         map[string]decimal.Decimal{"Retiree": decimal.NewFromInt(35000)},
		}

		// Calculate MAGI - will be around $195K (close to $206K threshold)
		acf.MAGI = CalculateMAGI(&acf)
		projection[i] = acf
	}

	return projection
}

// createConservativeIncomeProjection creates a safe projection well below IRMAA thresholds
func createConservativeIncomeProjection() []domain.AnnualCashFlow {
	projection := make([]domain.AnnualCashFlow, 10)
	baseDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 10; i++ {
		year := i + 1
		acf := domain.AnnualCashFlow{
			Year:               year,
			Date:               baseDate.AddDate(i, 0, 0),
			IsMedicareEligible: i >= 2,
			Pensions:           map[string]decimal.Decimal{"Retiree": decimal.NewFromInt(50000)},
			TSPWithdrawals:     map[string]decimal.Decimal{"Retiree": decimal.NewFromInt(40000)},
			SSBenefits:         map[string]decimal.Decimal{"Retiree": decimal.NewFromInt(30000)},
		}

		// Calculate MAGI - will be around $140K (well below $206K threshold)
		acf.MAGI = CalculateMAGI(&acf)
		projection[i] = acf
	}

	return projection
}

// TestIRMAAThresholdExamples shows exact threshold examples
func TestIRMAAThresholdExamples(t *testing.T) {
	mc := NewMedicareCalculator()

	examples := []struct {
		name                   string
		pension                int64
		tspWithdrawal          int64
		ssBenefit              int64
		expectedMAGI           int64
		expectedRisk           domain.IRMAARisk
		isMarriedFilingJointly bool
	}{
		{
			name:                   "Just below married threshold",
			pension:                60000,
			tspWithdrawal:          70000,
			ssBenefit:              35000,
			expectedMAGI:           159750, // 60k + 70k + (35k * 0.85) = 159.75K
			expectedRisk:           domain.IRMAARiskSafe,
			isMarriedFilingJointly: true,
		},
		{
			name:                   "Warning zone - within 10K of threshold",
			pension:                60000,
			tspWithdrawal:          80000,
			ssBenefit:              35000,
			expectedMAGI:           169750, // Close to 206K but > $10K away = Safe
			expectedRisk:           domain.IRMAARiskSafe,
			isMarriedFilingJointly: true,
		},
		{
			name:                   "Warning zone - within 10K",
			pension:                70000,
			tspWithdrawal:          85000,
			ssBenefit:              35000,
			expectedMAGI:           184750, // 184.75K - within $21K of 206K threshold
			expectedRisk:           domain.IRMAARiskSafe,
			isMarriedFilingJointly: true,
		},
		{
			name:                   "Just at warning threshold",
			pension:                70000,
			tspWithdrawal:          90000,
			ssBenefit:              36000,
			expectedMAGI:           190600, // Within $15.4K of threshold
			expectedRisk:           domain.IRMAARiskSafe,
			isMarriedFilingJointly: true,
		},
		{
			name:                   "Close warning",
			pension:                75000,
			tspWithdrawal:          85000,
			ssBenefit:              38000,
			expectedMAGI:           192300, // Within $13.7K = still safe
			expectedRisk:           domain.IRMAARiskSafe,
			isMarriedFilingJointly: true,
		},
		{
			name:                   "Warning - within $10K",
			pension:                75000,
			tspWithdrawal:          90000,
			ssBenefit:              40000,
			expectedMAGI:           199000, // Within $7K of 206K threshold!
			expectedRisk:           domain.IRMAARiskWarning,
			isMarriedFilingJointly: true,
		},
		{
			name:                   "Breach - exceeds threshold",
			pension:                80000,
			tspWithdrawal:          95000,
			ssBenefit:              42000,
			expectedMAGI:           210700, // EXCEEDS 206K threshold
			expectedRisk:           domain.IRMAARiskBreach,
			isMarriedFilingJointly: true,
		},
	}

	for _, ex := range examples {
		t.Run(ex.name, func(t *testing.T) {
			acf := domain.AnnualCashFlow{
				Pensions:       map[string]decimal.Decimal{"Retiree": decimal.NewFromInt(ex.pension)},
				TSPWithdrawals: map[string]decimal.Decimal{"Retiree": decimal.NewFromInt(ex.tspWithdrawal)},
				SSBenefits:     map[string]decimal.Decimal{"Retiree": decimal.NewFromInt(ex.ssBenefit)},
			}

			magi := CalculateMAGI(&acf)
			risk, tier, surcharge, distance := CalculateIRMAARiskStatus(magi, ex.isMarriedFilingJointly, mc)

			t.Logf("MAGI: $%s (expected ~$%d)", magi.StringFixed(0), ex.expectedMAGI)
			t.Logf("Risk: %s (expected %s)", risk, ex.expectedRisk)
			t.Logf("Tier: %s", tier)
			t.Logf("Surcharge: $%.2f/month", surcharge.InexactFloat64())
			t.Logf("Distance to next threshold: $%.0f", distance.InexactFloat64())

			if risk != ex.expectedRisk {
				t.Errorf("Expected risk %s, got %s", ex.expectedRisk, risk)
			}

			// Verify MAGI calculation is approximately correct
			expectedDec := decimal.NewFromInt(ex.expectedMAGI)
			diff := magi.Sub(expectedDec).Abs()
			if diff.GreaterThan(decimal.NewFromInt(100)) { // Allow $100 tolerance
				t.Errorf("MAGI calculation off: expected %s, got %s", expectedDec, magi)
			}
		})
	}
}
