package calculation

import (
	"testing"
	"time"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

func TestCalculateMAGI(t *testing.T) {
	tests := []struct {
		name     string
		acf      *domain.AnnualCashFlow
		expected decimal.Decimal
	}{
		{
			name: "Simple retirement income",
			acf: &domain.AnnualCashFlow{
				Year:           1,
				Date:           time.Now(),
				Pensions:       map[string]decimal.Decimal{"Alice": decimal.NewFromInt(50000)},
				TSPWithdrawals: map[string]decimal.Decimal{"Alice": decimal.NewFromInt(20000)},
				SSBenefits:     map[string]decimal.Decimal{"Alice": decimal.NewFromInt(30000)},
			},
			// MAGI = 50000 (pension) + 20000 (TSP) + 25500 (85% of SS) = 95500
			expected: decimal.NewFromInt(95500),
		},
		{
			name: "Dual income couple",
			acf: &domain.AnnualCashFlow{
				Year: 1,
				Date: time.Now(),
				Pensions: map[string]decimal.Decimal{
					"Alice": decimal.NewFromInt(50000),
					"Bob":   decimal.NewFromInt(45000),
				},
				TSPWithdrawals: map[string]decimal.Decimal{
					"Alice": decimal.NewFromInt(20000),
					"Bob":   decimal.NewFromInt(15000),
				},
				SSBenefits: map[string]decimal.Decimal{
					"Alice": decimal.NewFromInt(30000),
					"Bob":   decimal.NewFromInt(28000),
				},
			},
			// MAGI = (50000+45000) + (20000+15000) + 0.85*(30000+28000) = 179300
			expected: decimal.NewFromInt(179300),
		},
		{
			name: "Working with salary",
			acf: &domain.AnnualCashFlow{
				Year:       1,
				Date:       time.Now(),
				Salaries:   map[string]decimal.Decimal{"Alice": decimal.NewFromInt(100000)},
				SSBenefits: map[string]decimal.Decimal{},
			},
			// MAGI = 100000 (salary) = 100000
			expected: decimal.NewFromInt(100000),
		},
		{
			name: "With FERS supplement",
			acf: &domain.AnnualCashFlow{
				Year:            1,
				Date:            time.Now(),
				Pensions:        map[string]decimal.Decimal{"Alice": decimal.NewFromInt(40000)},
				FERSSupplements: map[string]decimal.Decimal{"Alice": decimal.NewFromInt(10000)},
				TSPWithdrawals:  map[string]decimal.Decimal{"Alice": decimal.NewFromInt(15000)},
			},
			// MAGI = 40000 + 10000 + 15000 = 65000
			expected: decimal.NewFromInt(65000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateMAGI(tt.acf)
			if !result.Equal(tt.expected) {
				t.Errorf("CalculateMAGI() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateIRMAARiskStatus(t *testing.T) {
	mc := NewMedicareCalculator()

	tests := []struct {
		name                   string
		magi                   decimal.Decimal
		isMarriedFilingJointly bool
		expectedRisk           domain.IRMAARisk
		expectedTier           string
		minSurcharge           decimal.Decimal // Minimum expected surcharge
	}{
		{
			name:                   "Safe - well below threshold (single)",
			magi:                   decimal.NewFromInt(80000),
			isMarriedFilingJointly: false,
			expectedRisk:           domain.IRMAARiskSafe,
			expectedTier:           "None",
			minSurcharge:           decimal.Zero,
		},
		{
			name:                   "Warning - within $10K of threshold (single)",
			magi:                   decimal.NewFromInt(98000), // $103K threshold, $5K away
			isMarriedFilingJointly: false,
			expectedRisk:           domain.IRMAARiskWarning,
			expectedTier:           "None",
			minSurcharge:           decimal.Zero,
		},
		{
			name:                   "Breach - exceeds first threshold (single)",
			magi:                   decimal.NewFromInt(110000),
			isMarriedFilingJointly: false,
			expectedRisk:           domain.IRMAARiskBreach,
			expectedTier:           "Tier1",
			minSurcharge:           decimal.NewFromFloat(60), // At least $60/month
		},
		{
			name:                   "Safe - well below threshold (married)",
			magi:                   decimal.NewFromInt(150000),
			isMarriedFilingJointly: true,
			expectedRisk:           domain.IRMAARiskSafe,
			expectedTier:           "None",
			minSurcharge:           decimal.Zero,
		},
		{
			name:                   "Warning - within $10K of threshold (married)",
			magi:                   decimal.NewFromInt(200000), // $206K threshold, $6K away
			isMarriedFilingJointly: true,
			expectedRisk:           domain.IRMAARiskWarning,
			expectedTier:           "None",
			minSurcharge:           decimal.Zero,
		},
		{
			name:                   "Breach - exceeds first threshold (married)",
			magi:                   decimal.NewFromInt(220000),
			isMarriedFilingJointly: true,
			expectedRisk:           domain.IRMAARiskBreach,
			expectedTier:           "Tier1",
			minSurcharge:           decimal.NewFromFloat(60),
		},
		{
			name:                   "Breach - exceeds multiple thresholds (married)",
			magi:                   decimal.NewFromInt(350000),
			isMarriedFilingJointly: true,
			expectedRisk:           domain.IRMAARiskBreach,
			expectedTier:           "Tier3", // Should be in tier 3
			minSurcharge:           decimal.NewFromFloat(200),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			risk, tier, surcharge, _ := CalculateIRMAARiskStatus(
				tt.magi,
				tt.isMarriedFilingJointly,
				mc,
			)

			if risk != tt.expectedRisk {
				t.Errorf("Risk status = %v, expected %v", risk, tt.expectedRisk)
			}

			if tier != tt.expectedTier {
				t.Errorf("Tier = %v, expected %v", tier, tt.expectedTier)
			}

			if surcharge.LessThan(tt.minSurcharge) {
				t.Errorf("Surcharge = %v, expected at least %v", surcharge, tt.minSurcharge)
			}
		})
	}
}

func TestAnalyzeIRMAARisk(t *testing.T) {
	mc := NewMedicareCalculator()

	// Create a projection with mixed risk years
	projection := []domain.AnnualCashFlow{
		// Year 1-2: Not Medicare eligible yet
		{
			Year:               1,
			IsMedicareEligible: false,
			MAGI:               decimal.NewFromInt(150000),
		},
		{
			Year:               2,
			IsMedicareEligible: false,
			MAGI:               decimal.NewFromInt(160000),
		},
		// Year 3-4: Medicare eligible, safe
		{
			Year:               3,
			IsMedicareEligible: true,
			MAGI:               decimal.NewFromInt(150000), // Safe for married
		},
		{
			Year:               4,
			IsMedicareEligible: true,
			MAGI:               decimal.NewFromInt(180000), // Safe for married
		},
		// Year 5: Warning
		{
			Year:               5,
			IsMedicareEligible: true,
			MAGI:               decimal.NewFromInt(200000), // Warning (within $6K of $206K)
		},
		// Year 6-7: Breach
		{
			Year:               6,
			IsMedicareEligible: true,
			MAGI:               decimal.NewFromInt(220000), // Breach tier 1
		},
		{
			Year:               7,
			IsMedicareEligible: true,
			MAGI:               decimal.NewFromInt(270000), // Breach tier 2
		},
	}

	analysis := AnalyzeIRMAARisk(projection, true, mc)

	// Verify warnings
	if len(analysis.YearsWithWarnings) != 1 {
		t.Errorf("Expected 1 warning year, got %d", len(analysis.YearsWithWarnings))
	}
	if len(analysis.YearsWithWarnings) > 0 && analysis.YearsWithWarnings[0] != 5 {
		t.Errorf("Expected warning in year 5, got year %d", analysis.YearsWithWarnings[0])
	}

	// Verify breaches
	if len(analysis.YearsWithBreaches) != 2 {
		t.Errorf("Expected 2 breach years, got %d", len(analysis.YearsWithBreaches))
	}
	if analysis.FirstBreachYear != 6 {
		t.Errorf("Expected first breach in year 6, got year %d", analysis.FirstBreachYear)
	}

	// Verify total IRMAA cost is calculated
	if analysis.TotalIRMAACost.LessThanOrEqual(decimal.Zero) {
		t.Errorf("Expected positive total IRMAA cost, got %v", analysis.TotalIRMAACost)
	}

	// Verify recommendations exist
	if len(analysis.Recommendations) == 0 {
		t.Error("Expected recommendations to be generated")
	}

	// Verify high risk years
	if len(analysis.HighRiskYears) != 3 { // 1 warning + 2 breaches
		t.Errorf("Expected 3 high risk years, got %d", len(analysis.HighRiskYears))
	}
}

func TestAnalyzeIRMAARisk_NoBreaches(t *testing.T) {
	mc := NewMedicareCalculator()

	// Create a projection with all safe years
	projection := []domain.AnnualCashFlow{
		{
			Year:               1,
			IsMedicareEligible: true,
			MAGI:               decimal.NewFromInt(150000),
		},
		{
			Year:               2,
			IsMedicareEligible: true,
			MAGI:               decimal.NewFromInt(160000),
		},
		{
			Year:               3,
			IsMedicareEligible: true,
			MAGI:               decimal.NewFromInt(170000),
		},
	}

	analysis := AnalyzeIRMAARisk(projection, true, mc)

	if len(analysis.YearsWithBreaches) != 0 {
		t.Errorf("Expected 0 breach years, got %d", len(analysis.YearsWithBreaches))
	}

	if len(analysis.YearsWithWarnings) != 0 {
		t.Errorf("Expected 0 warning years, got %d", len(analysis.YearsWithWarnings))
	}

	if !analysis.TotalIRMAACost.Equal(decimal.Zero) {
		t.Errorf("Expected zero IRMAA cost, got %v", analysis.TotalIRMAACost)
	}

	if analysis.FirstBreachYear != 0 {
		t.Errorf("Expected no first breach year, got %d", analysis.FirstBreachYear)
	}

	// Should still have recommendations (positive feedback)
	if len(analysis.Recommendations) == 0 {
		t.Error("Expected recommendations even with no breaches")
	}
}
