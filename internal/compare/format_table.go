package compare

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// TableFormatter formats comparison results as a console table
type TableFormatter struct{}

// Format generates a formatted table comparing scenarios
func (tf *TableFormatter) Format(compSet *ComparisonSet) string {
	var sb strings.Builder

	// Header
	sb.WriteString("RETIREMENT SCENARIO COMPARISON\n")
	sb.WriteString(strings.Repeat("=", 80) + "\n")
	sb.WriteString(fmt.Sprintf("Base Scenario: %s\n", compSet.BaseScenarioName))
	sb.WriteString(fmt.Sprintf("Configuration: %s\n", compSet.ConfigPath))
	sb.WriteString("\n")

	// Create table with all scenarios

	// Column widths
	nameWidth := 25
	numWidth := 15

	// Table header
	sb.WriteString(fmt.Sprintf("%-*s %*s %*s %*s %*s\n",
		nameWidth, "Scenario",
		numWidth, "1st Year Income",
		numWidth, "Lifetime Income",
		numWidth, "TSP Longevity",
		numWidth, "Final TSP"))
	sb.WriteString(strings.Repeat("-", 80) + "\n")

	// Base scenario row
	base := compSet.BaseResult
	sb.WriteString(tf.formatRow(base, nameWidth, numWidth, true))

	// Alternative scenarios
	if len(compSet.AlternativeResults) > 0 {
		sb.WriteString(strings.Repeat("-", 80) + "\n")
		for _, alt := range compSet.AlternativeResults {
			sb.WriteString(tf.formatRow(&alt, nameWidth, numWidth, false))
		}
	}

	sb.WriteString(strings.Repeat("=", 80) + "\n")

	// Comparison details (deltas from base)
	if len(compSet.AlternativeResults) > 0 {
		sb.WriteString("\nCOMPARISON TO BASE\n")
		sb.WriteString(strings.Repeat("-", 80) + "\n")

		for _, alt := range compSet.AlternativeResults {
			sb.WriteString(fmt.Sprintf("\n%s:\n", alt.ScenarioName))

			// Income difference
			incomeSymbol := tf.deltaSymbol(alt.IncomeDiffFromBase)
			sb.WriteString(fmt.Sprintf("  Lifetime Income:  %s$%s (%s%%)\n",
				incomeSymbol,
				tf.formatDecimal(alt.IncomeDiffFromBase.Abs()),
				alt.IncomePctFromBase.StringFixed(1)))

			// TSP longevity difference
			if alt.TSPLongevityDiff != 0 {
				longevitySymbol := "+"
				if alt.TSPLongevityDiff < 0 {
					longevitySymbol = ""
				}
				sb.WriteString(fmt.Sprintf("  TSP Longevity:    %s%d years\n",
					longevitySymbol, alt.TSPLongevityDiff))
			}

			// Tax difference
			if !alt.TaxDiffFromBase.IsZero() {
				taxSymbol := tf.deltaSymbol(alt.TaxDiffFromBase.Neg()) // Lower taxes are better
				sb.WriteString(fmt.Sprintf("  Tax Impact:       %s$%s\n",
					taxSymbol,
					tf.formatDecimal(alt.TaxDiffFromBase.Abs())))
			}
		}
		sb.WriteString("\n")
	}

	// Recommendations
	if len(compSet.Recommendations) > 0 {
		sb.WriteString("\nRECOMMENDATIONS\n")
		sb.WriteString(strings.Repeat("-", 80) + "\n")
		for _, rec := range compSet.Recommendations {
			sb.WriteString(fmt.Sprintf("• %s\n", rec))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatRow formats a single scenario row
func (tf *TableFormatter) formatRow(result *ComparisonResult, nameWidth, numWidth int, isBase bool) string {
	name := result.ScenarioName
	if isBase {
		name += " (base)"
	}

	longevityStr := fmt.Sprintf("%d years", result.TSPLongevity)
	if result.TSPLongevity == 0 {
		longevityStr = "depleted"
	}

	return fmt.Sprintf("%-*s %*s %*s %*s %*s\n",
		nameWidth, tf.truncate(name, nameWidth),
		numWidth, "$"+tf.formatDecimal(result.FirstYearNetIncome),
		numWidth, "$"+tf.formatDecimal(result.LifetimeIncome),
		numWidth, longevityStr,
		numWidth, "$"+tf.formatDecimal(result.FinalTSPBalance))
}

// formatDecimal formats a decimal for display (in thousands)
func (tf *TableFormatter) formatDecimal(d decimal.Decimal) string {
	if d.Abs().GreaterThanOrEqual(decimal.NewFromInt(1000000)) {
		// Format in millions
		millions := d.Div(decimal.NewFromInt(1000000))
		return millions.StringFixed(2) + "M"
	} else if d.Abs().GreaterThanOrEqual(decimal.NewFromInt(1000)) {
		// Format in thousands
		thousands := d.Div(decimal.NewFromInt(1000))
		return thousands.StringFixed(1) + "K"
	}
	return d.StringFixed(0)
}

// deltaSymbol returns a + or - symbol for deltas (positive is green concept)
func (tf *TableFormatter) deltaSymbol(delta decimal.Decimal) string {
	if delta.IsPositive() {
		return "+"
	} else if delta.IsNegative() {
		return ""
	}
	return " "
}

// truncate truncates a string to maxLen
func (tf *TableFormatter) truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// FormatCompact creates a compact single-line summary for each scenario
func (tf *TableFormatter) FormatCompact(compSet *ComparisonSet) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Base: %s | ", compSet.BaseScenarioName))

	for i, alt := range compSet.AlternativeResults {
		if i > 0 {
			sb.WriteString(" | ")
		}
		incomeChange := "="
		if alt.IncomeDiffFromBase.IsPositive() {
			incomeChange = fmt.Sprintf("+$%s", tf.formatDecimal(alt.IncomeDiffFromBase))
		} else if alt.IncomeDiffFromBase.IsNegative() {
			incomeChange = fmt.Sprintf("-$%s", tf.formatDecimal(alt.IncomeDiffFromBase.Abs()))
		}

		sb.WriteString(fmt.Sprintf("%s: %s", alt.ScenarioName, incomeChange))
	}

	return sb.String()
}
