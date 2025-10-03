package output

import (
	"fmt"
	"strings"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// RothConversionFormatter defines a formatter for Roth conversion plans
type RothConversionFormatter interface {
	FormatRothConversionPlan(plan *domain.RothConversionPlan) (string, error)
	Name() string
}

// NewFormatter creates a formatter based on the format name
func NewFormatter(format string) RothConversionFormatter {
	switch strings.ToLower(format) {
	case "table":
		return &RothConversionTableFormatter{}
	case "json":
		return &RothConversionJSONFormatter{}
	default:
		return &RothConversionTableFormatter{}
	}
}

// RothConversionTableFormatter formats Roth conversion plans as a table
type RothConversionTableFormatter struct{}

func (f *RothConversionTableFormatter) Name() string {
	return "table"
}

func (f *RothConversionTableFormatter) FormatRothConversionPlan(plan *domain.RothConversionPlan) (string, error) {
	if plan == nil {
		return "", fmt.Errorf("plan cannot be nil")
	}

	var output strings.Builder

	// Header
	output.WriteString("ROTH CONVERSION ANALYSIS\n")
	output.WriteString("=================================================================\n")
	output.WriteString(fmt.Sprintf("Participant: %s\n", plan.Participant))
	output.WriteString(fmt.Sprintf("Conversion Window: %s\n", plan.ConversionWindow.String()))
	output.WriteString(fmt.Sprintf("Target Bracket: %d%%\n", plan.TargetBracket))
	output.WriteString(fmt.Sprintf("Objective: %s\n\n", plan.Objective.String()))

	// Baseline analysis
	if plan.Baseline != nil {
		output.WriteString("BASELINE (No Conversions):\n")
		output.WriteString(fmt.Sprintf("  Lifetime Federal Tax:     %s\n", FormatCurrency(plan.Baseline.Projection[0].FederalTax)))
		output.WriteString(fmt.Sprintf("  Lifetime IRMAA:           %s\n", FormatCurrency(decimal.Zero))) // TODO: Calculate from projection
		output.WriteString(fmt.Sprintf("  Total Cost:               %s\n", FormatCurrency(plan.Baseline.Projection[0].FederalTax)))
		output.WriteString(fmt.Sprintf("  Final Traditional TSP:    %s\n", FormatCurrency(decimal.Zero)))   // TODO: Calculate from projection
		output.WriteString(fmt.Sprintf("  Final Roth TSP:           %s\n\n", FormatCurrency(decimal.Zero))) // TODO: Calculate from projection
	}

	// Recommended strategy
	if plan.Recommended != nil {
		output.WriteString("RECOMMENDED STRATEGY:\n")
		output.WriteString(fmt.Sprintf("  %d: Convert %s  (fills %d%% bracket, tax = %s)\n",
			plan.Recommended.Strategy.Year,
			FormatCurrency(plan.Recommended.Strategy.Amount),
			plan.TargetBracket,
			FormatCurrency(plan.Recommended.LifetimeTax.Sub(plan.Baseline.Projection[0].FederalTax))))
		output.WriteString(fmt.Sprintf("\n  Total Converted: %s\n", FormatCurrency(plan.Recommended.Strategy.Amount)))
		output.WriteString(fmt.Sprintf("  Total Tax Paid:  %s (%d%% effective on conversions)\n\n",
			FormatCurrency(plan.Recommended.LifetimeTax.Sub(plan.Baseline.Projection[0].FederalTax)),
			plan.TargetBracket))
	}

	// Analysis
	if plan.Analysis != nil {
		output.WriteString("OUTCOME WITH CONVERSIONS:\n")
		output.WriteString(fmt.Sprintf("  Lifetime Federal Tax:     %s\n", FormatCurrency(plan.Recommended.LifetimeTax)))
		output.WriteString(fmt.Sprintf("  Lifetime IRMAA:           %s\n", FormatCurrency(plan.Recommended.LifetimeIRMAA)))
		output.WriteString(fmt.Sprintf("  Total Cost:               %s\n", FormatCurrency(plan.Recommended.LifetimeTax.Add(plan.Recommended.LifetimeIRMAA))))
		output.WriteString(fmt.Sprintf("  Final Traditional TSP:    %s\n", FormatCurrency(plan.Recommended.FinalBalances.TraditionalTSP)))
		output.WriteString(fmt.Sprintf("  Final Roth TSP:           %s\n\n", FormatCurrency(plan.Recommended.FinalBalances.RothTSP)))

		output.WriteString("NET BENEFIT OVER 30 YEARS:\n")
		output.WriteString(fmt.Sprintf("  Tax Cost:      %s (paid upfront)\n", FormatCurrency(plan.Analysis.TotalTaxPaid)))
		output.WriteString(fmt.Sprintf("  IRMAA Savings: %s\n", FormatCurrency(plan.Analysis.IRMAASavings)))
		output.WriteString(fmt.Sprintf("  RMD Reduction: %s (lower taxes on smaller RMDs)\n", FormatCurrency(plan.Analysis.RMDTaxReduction)))
		output.WriteString(fmt.Sprintf("  Total Benefit: %s (NET SAVINGS)\n", FormatCurrency(plan.Analysis.NetBenefit)))
		output.WriteString(fmt.Sprintf("  ROI: %.1f%% return on conversion tax paid\n\n", plan.Analysis.ROI.InexactFloat64()))

		output.WriteString(fmt.Sprintf("RECOMMENDATION: %s\n\n", plan.Analysis.Recommendation))

		// Sensitivity analysis
		output.WriteString("SENSITIVITY:\n")
		output.WriteString(fmt.Sprintf("  If convert 20%% more:  Net benefit = %s\n", FormatCurrency(plan.Analysis.SensitivityAnalysis.Plus20Percent)))
		output.WriteString(fmt.Sprintf("  If convert 20%% less:  Net benefit = %s\n", FormatCurrency(plan.Analysis.SensitivityAnalysis.Minus20Percent)))
		output.WriteString(fmt.Sprintf("  Optimal range: %s\n", plan.Analysis.SensitivityAnalysis.OptimalRange))
	}

	return output.String(), nil
}

// RothConversionJSONFormatter formats Roth conversion plans as JSON
type RothConversionJSONFormatter struct{}

func (f *RothConversionJSONFormatter) Name() string {
	return "json"
}

func (f *RothConversionJSONFormatter) FormatRothConversionPlan(plan *domain.RothConversionPlan) (string, error) {
	if plan == nil {
		return "", fmt.Errorf("plan cannot be nil")
	}

	// For now, return a simple JSON representation
	// TODO: Implement proper JSON marshaling
	return fmt.Sprintf(`{
  "participant": "%s",
  "conversionWindow": "%s",
  "targetBracket": %d,
  "objective": "%s",
  "recommendedStrategy": {
    "year": %d,
    "amount": "%s"
  },
  "netBenefit": "%s",
  "roi": "%.1f%%"
}`,
		plan.Participant,
		plan.ConversionWindow.String(),
		plan.TargetBracket,
		plan.Objective.String(),
		plan.Recommended.Strategy.Year,
		plan.Recommended.Strategy.Amount.String(),
		plan.Analysis.NetBenefit.String(),
		plan.Analysis.ROI.InexactFloat64()), nil
}
