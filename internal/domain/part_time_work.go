package domain

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// PartTimeWorkPeriod represents a specific period of part-time work
type PartTimeWorkPeriod struct {
	PeriodStart            time.Time        `yaml:"period_start" json:"periodStart"`
	PeriodEnd              time.Time        `yaml:"period_end" json:"periodEnd"`
	AnnualSalary           decimal.Decimal  `yaml:"annual_salary" json:"annualSalary"`
	TSPContributionPercent decimal.Decimal  `yaml:"tsp_contribution_percent" json:"tspContributionPercent"`
	WorkType               string           `yaml:"work_type" json:"workType"` // "w2" | "1099"
	HoursPerWeek           *decimal.Decimal `yaml:"hours_per_week,omitempty" json:"hoursPerWeek,omitempty"`
	WorkDaysPerWeek        *int             `yaml:"work_days_per_week,omitempty" json:"workDaysPerWeek,omitempty"`
}

// PartTimeWorkSchedule represents a complete part-time work schedule
type PartTimeWorkSchedule struct {
	StartDate time.Time            `yaml:"start_date" json:"startDate"`
	EndDate   time.Time            `yaml:"end_date" json:"endDate"`
	Schedule  []PartTimeWorkPeriod `yaml:"schedule" json:"schedule"`
}

// FERSSupplementEarningsTest represents the earnings test for FERS supplement
type FERSSupplementEarningsTest struct {
	AnnualEarningsLimit decimal.Decimal `yaml:"annual_earnings_limit" json:"annualEarningsLimit"`
	ReductionRate       decimal.Decimal `yaml:"reduction_rate" json:"reductionRate"`
	ExemptionAge        int             `yaml:"exemption_age" json:"exemptionAge"`
}

// PartTimeWorkAnalysis represents analysis of part-time work impact
type PartTimeWorkAnalysis struct {
	Year             int                 `json:"year"`
	IsPartTime       bool                `json:"isPartTime"`
	PartTimePeriod   *PartTimeWorkPeriod `json:"partTimePeriod,omitempty"`
	AnnualSalary     decimal.Decimal     `json:"annualSalary"`
	MonthlySalary    decimal.Decimal     `json:"monthlySalary"`
	TSPContributions decimal.Decimal     `json:"tspContributions"`
	FICATax          decimal.Decimal     `json:"ficaTax"`
	FederalTax       decimal.Decimal     `json:"federalTax"`
	StateTax         decimal.Decimal     `json:"stateTax"`
	LocalTax         decimal.Decimal     `json:"localTax"`
	NetIncome        decimal.Decimal     `json:"netIncome"`

	// FERS Supplement impact
	FERSSupplementReduction decimal.Decimal `json:"fersSupplementReduction"`
	FERSSupplementRemaining decimal.Decimal `json:"fersSupplementRemaining"`
	EarningsTestApplied     bool            `json:"earningsTestApplied"`

	// Work type specific
	SelfEmploymentTax decimal.Decimal `json:"selfEmploymentTax"`
	WorkType          string          `json:"workType"`
}

// DefaultFERSSupplementEarningsTest returns default earnings test parameters for 2025
func DefaultFERSSupplementEarningsTest() FERSSupplementEarningsTest {
	return FERSSupplementEarningsTest{
		AnnualEarningsLimit: decimal.NewFromInt(23400),  // $23,400 for 2025
		ReductionRate:       decimal.NewFromFloat(0.50), // 50% reduction for earnings above limit
		ExemptionAge:        62,                         // No earnings test at age 62+
	}
}

// CalculateFERSSupplementReduction calculates the reduction in FERS supplement due to earnings
func (ptw *PartTimeWorkPeriod) CalculateFERSSupplementReduction(
	annualEarnings decimal.Decimal,
	earningsTest FERSSupplementEarningsTest,
	age int,
) decimal.Decimal {

	// No earnings test at exemption age or older
	if age >= earningsTest.ExemptionAge {
		return decimal.Zero
	}

	// No reduction if earnings are below limit
	if annualEarnings.LessThanOrEqual(earningsTest.AnnualEarningsLimit) {
		return decimal.Zero
	}

	// Calculate reduction: 50% of earnings above the limit
	excessEarnings := annualEarnings.Sub(earningsTest.AnnualEarningsLimit)
	reduction := excessEarnings.Mul(earningsTest.ReductionRate)

	return reduction
}

// CalculateSelfEmploymentTax calculates self-employment tax for 1099 work
func (ptw *PartTimeWorkPeriod) CalculateSelfEmploymentTax(annualEarnings decimal.Decimal) decimal.Decimal {
	if ptw.WorkType != "1099" {
		return decimal.Zero
	}

	// Self-employment tax rate: 15.3% (12.4% Social Security + 2.9% Medicare)
	// Applied to 92.35% of net earnings (after business expenses deduction)
	selfEmploymentRate := decimal.NewFromFloat(0.153)
	businessExpenseRate := decimal.NewFromFloat(0.9235)

	taxableEarnings := annualEarnings.Mul(businessExpenseRate)
	selfEmploymentTax := taxableEarnings.Mul(selfEmploymentRate)

	return selfEmploymentTax
}

// GetWorkFraction calculates the fraction of full-time work for this period
func (ptw *PartTimeWorkPeriod) GetWorkFraction(fullTimeSalary decimal.Decimal) decimal.Decimal {
	if fullTimeSalary.IsZero() {
		return decimal.Zero
	}
	return ptw.AnnualSalary.Div(fullTimeSalary)
}

// IsActiveInYear checks if this period is active in the given year
func (ptw *PartTimeWorkPeriod) IsActiveInYear(year int) bool {
	yearStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)

	return !ptw.PeriodStart.After(yearEnd) && !ptw.PeriodEnd.Before(yearStart)
}

// GetActivePeriodForYear returns the active part-time work period for a given year
func (ptws *PartTimeWorkSchedule) GetActivePeriodForYear(year int) *PartTimeWorkPeriod {
	for i := range ptws.Schedule {
		if ptws.Schedule[i].IsActiveInYear(year) {
			return &ptws.Schedule[i]
		}
	}
	return nil
}

// IsPartTimeInYear checks if participant is working part-time in the given year
func (ptws *PartTimeWorkSchedule) IsPartTimeInYear(year int) bool {
	return ptws.GetActivePeriodForYear(year) != nil
}

// ValidatePartTimeSchedule validates the part-time work schedule
func (ptws *PartTimeWorkSchedule) ValidatePartTimeSchedule() error {
	if ptws.StartDate.After(ptws.EndDate) {
		return fmt.Errorf("part-time work start date cannot be after end date")
	}

	if len(ptws.Schedule) == 0 {
		return fmt.Errorf("part-time work schedule must have at least one period")
	}

	for i, period := range ptws.Schedule {
		if period.PeriodStart.After(period.PeriodEnd) {
			return fmt.Errorf("period %d: start date cannot be after end date", i+1)
		}

		if period.AnnualSalary.LessThanOrEqual(decimal.Zero) {
			return fmt.Errorf("period %d: annual salary must be positive", i+1)
		}

		if period.TSPContributionPercent.LessThan(decimal.Zero) || period.TSPContributionPercent.GreaterThan(decimal.NewFromFloat(1.0)) {
			return fmt.Errorf("period %d: TSP contribution percent must be between 0 and 1", i+1)
		}

		if period.WorkType != "w2" && period.WorkType != "1099" {
			return fmt.Errorf("period %d: work type must be 'w2' or '1099'", i+1)
		}
	}

	return nil
}
