package calculation

import (
	"fmt"
	"time"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// PartTimeWorkCalculator handles part-time work calculations
type PartTimeWorkCalculator struct {
	earningsTest domain.FERSSupplementEarningsTest
}

// NewPartTimeWorkCalculator creates a new part-time work calculator
func NewPartTimeWorkCalculator() *PartTimeWorkCalculator {
	return &PartTimeWorkCalculator{
		earningsTest: domain.DefaultFERSSupplementEarningsTest(),
	}
}

// CalculatePartTimeWorkForYear calculates part-time work impact for a specific year
func (ptwc *PartTimeWorkCalculator) CalculatePartTimeWorkForYear(
	participant domain.Participant,
	participantScenario domain.ParticipantScenario,
	year int,
	age int,
) (*domain.PartTimeWorkAnalysis, error) {

	// Check if participant has part-time work schedule
	if participantScenario.PartTimeWork == nil {
		return &domain.PartTimeWorkAnalysis{
			Year:       year,
			IsPartTime: false,
		}, nil
	}

	// Get active part-time period for this year
	activePeriod := participantScenario.PartTimeWork.GetActivePeriodForYear(year)
	if activePeriod == nil {
		return &domain.PartTimeWorkAnalysis{
			Year:       year,
			IsPartTime: false,
		}, nil
	}

	// Calculate part-time work analysis
	analysis := &domain.PartTimeWorkAnalysis{
		Year:           year,
		IsPartTime:     true,
		PartTimePeriod: activePeriod,
		AnnualSalary:   activePeriod.AnnualSalary,
		MonthlySalary:  activePeriod.AnnualSalary.Div(decimal.NewFromInt(12)),
		WorkType:       activePeriod.WorkType,
	}

	// Calculate TSP contributions
	analysis.TSPContributions = activePeriod.AnnualSalary.Mul(activePeriod.TSPContributionPercent)

	// Calculate taxes based on work type
	if activePeriod.WorkType == "w2" {
		// W-2 employee: standard FICA calculation
		analysis.FICATax = ptwc.calculateFICATax(activePeriod.AnnualSalary)
	} else if activePeriod.WorkType == "1099" {
		// 1099 contractor: self-employment tax
		analysis.SelfEmploymentTax = activePeriod.CalculateSelfEmploymentTax(activePeriod.AnnualSalary)
		// Self-employment tax includes both employer and employee portions of FICA
		analysis.FICATax = analysis.SelfEmploymentTax
	}

	// Calculate FERS supplement reduction due to earnings
	analysis.FERSSupplementReduction = activePeriod.CalculateFERSSupplementReduction(
		activePeriod.AnnualSalary,
		ptwc.earningsTest,
		age,
	)

	// Calculate net income
	analysis.NetIncome = activePeriod.AnnualSalary.
		Sub(analysis.TSPContributions).
		Sub(analysis.FICATax).
		Sub(analysis.SelfEmploymentTax)

	// Note: Federal, state, and local taxes are calculated separately in the main projection engine
	// This analysis focuses on the part-time work specific calculations

	return analysis, nil
}

// calculateFICATax calculates FICA tax for W-2 employees
func (ptwc *PartTimeWorkCalculator) calculateFICATax(annualSalary decimal.Decimal) decimal.Decimal {
	// 2025 FICA rates: 6.2% Social Security + 1.45% Medicare = 7.65%
	// Social Security wage base: $176,100 for 2025
	socialSecurityRate := decimal.NewFromFloat(0.062)
	medicareRate := decimal.NewFromFloat(0.0145)
	socialSecurityWageBase := decimal.NewFromInt(176100)

	var socialSecurityTax decimal.Decimal
	if annualSalary.LessThanOrEqual(socialSecurityWageBase) {
		socialSecurityTax = annualSalary.Mul(socialSecurityRate)
	} else {
		socialSecurityTax = socialSecurityWageBase.Mul(socialSecurityRate)
	}

	medicareTax := annualSalary.Mul(medicareRate)

	return socialSecurityTax.Add(medicareTax)
}

// CalculateFERSSupplementImpact calculates the impact of part-time earnings on FERS supplement
func (ptwc *PartTimeWorkCalculator) CalculateFERSSupplementImpact(
	participantScenario domain.ParticipantScenario,
	year int,
	age int,
	originalSupplement decimal.Decimal,
) decimal.Decimal {

	if participantScenario.PartTimeWork == nil {
		return originalSupplement
	}

	activePeriod := participantScenario.PartTimeWork.GetActivePeriodForYear(year)
	if activePeriod == nil {
		return originalSupplement
	}

	// Calculate reduction due to earnings
	reduction := activePeriod.CalculateFERSSupplementReduction(
		activePeriod.AnnualSalary,
		ptwc.earningsTest,
		age,
	)

	// Return remaining supplement after reduction
	remainingSupplement := originalSupplement.Sub(reduction)
	if remainingSupplement.LessThan(decimal.Zero) {
		return decimal.Zero
	}

	return remainingSupplement
}

// ValidatePartTimeWorkSchedule validates a part-time work schedule
func (ptwc *PartTimeWorkCalculator) ValidatePartTimeWorkSchedule(
	participantScenario domain.ParticipantScenario,
) error {

	if participantScenario.PartTimeWork == nil {
		return nil
	}

	// Validate the schedule structure
	if err := participantScenario.PartTimeWork.ValidatePartTimeSchedule(); err != nil {
		return fmt.Errorf("invalid part-time work schedule: %w", err)
	}

	// Additional business logic validations

	// Check that part-time work doesn't overlap with full retirement
	if participantScenario.RetirementDate != nil {
		for _, period := range participantScenario.PartTimeWork.Schedule {
			if period.PeriodEnd.After(*participantScenario.RetirementDate) {
				return fmt.Errorf("part-time work period extends beyond retirement date")
			}
		}
	}

	// Check that periods don't overlap
	for i := 0; i < len(participantScenario.PartTimeWork.Schedule)-1; i++ {
		current := participantScenario.PartTimeWork.Schedule[i]
		next := participantScenario.PartTimeWork.Schedule[i+1]

		if current.PeriodEnd.After(next.PeriodStart) {
			return fmt.Errorf("part-time work periods cannot overlap")
		}
	}

	return nil
}

// GetPartTimeWorkSummary returns a summary of part-time work for a participant
func (ptwc *PartTimeWorkCalculator) GetPartTimeWorkSummary(
	participantScenario domain.ParticipantScenario,
) *PartTimeWorkSummary {

	if participantScenario.PartTimeWork == nil {
		return &PartTimeWorkSummary{
			HasPartTimeWork: false,
		}
	}

	summary := &PartTimeWorkSummary{
		HasPartTimeWork: true,
		StartDate:       participantScenario.PartTimeWork.StartDate,
		EndDate:         participantScenario.PartTimeWork.EndDate,
		TotalPeriods:    len(participantScenario.PartTimeWork.Schedule),
		Periods:         make([]PartTimePeriodSummary, len(participantScenario.PartTimeWork.Schedule)),
	}

	for i, period := range participantScenario.PartTimeWork.Schedule {
		summary.Periods[i] = PartTimePeriodSummary{
			PeriodStart:            period.PeriodStart,
			PeriodEnd:              period.PeriodEnd,
			AnnualSalary:           period.AnnualSalary,
			TSPContributionPercent: period.TSPContributionPercent,
			WorkType:               period.WorkType,
		}
	}

	return summary
}

// PartTimeWorkSummary provides a summary of part-time work
type PartTimeWorkSummary struct {
	HasPartTimeWork bool                    `json:"hasPartTimeWork"`
	StartDate       time.Time               `json:"startDate"`
	EndDate         time.Time               `json:"endDate"`
	TotalPeriods    int                     `json:"totalPeriods"`
	Periods         []PartTimePeriodSummary `json:"periods"`
}

// PartTimePeriodSummary provides a summary of a part-time work period
type PartTimePeriodSummary struct {
	PeriodStart            time.Time       `json:"periodStart"`
	PeriodEnd              time.Time       `json:"periodEnd"`
	AnnualSalary           decimal.Decimal `json:"annualSalary"`
	TSPContributionPercent decimal.Decimal `json:"tspContributionPercent"`
	WorkType               string          `json:"workType"`
}
