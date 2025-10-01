package compare

import (
	"encoding/csv"
	"fmt"
	"strings"
)

// CSVFormatter formats comparison results as CSV
type CSVFormatter struct{}

// Format generates CSV output for comparison results
func (cf *CSVFormatter) Format(compSet *ComparisonSet) (string, error) {
	var sb strings.Builder
	writer := csv.NewWriter(&sb)

	// Write header
	header := []string{
		"Scenario",
		"Type",
		"First Year Income",
		"Lifetime Income",
		"TSP Longevity (Years)",
		"Final TSP Balance",
		"Lifetime Taxes",
		"Income Diff from Base",
		"Income % Change",
		"TSP Longevity Diff",
		"Tax Diff from Base",
	}
	if err := writer.Write(header); err != nil {
		return "", err
	}

	// Write base scenario
	if err := writer.Write(cf.formatRow(compSet.BaseResult, "base")); err != nil {
		return "", err
	}

	// Write alternative scenarios
	for _, alt := range compSet.AlternativeResults {
		if err := writer.Write(cf.formatRow(&alt, "alternative")); err != nil {
			return "", err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}

	return sb.String(), nil
}

// formatRow formats a comparison result as a CSV row
func (cf *CSVFormatter) formatRow(result *ComparisonResult, scenarioType string) []string {
	return []string{
		result.ScenarioName,
		scenarioType,
		result.FirstYearNetIncome.StringFixed(2),
		result.LifetimeIncome.StringFixed(2),
		formatInt(result.TSPLongevity),
		result.FinalTSPBalance.StringFixed(2),
		result.LifetimeTaxes.StringFixed(2),
		result.IncomeDiffFromBase.StringFixed(2),
		result.IncomePctFromBase.StringFixed(2),
		formatInt(result.TSPLongevityDiff),
		result.TaxDiffFromBase.StringFixed(2),
	}
}

func formatInt(i int) string {
	return fmt.Sprintf("%d", i)
}
