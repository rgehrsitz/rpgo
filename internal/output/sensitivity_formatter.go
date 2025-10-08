package output

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// SensitivityFormatter defines a formatter for sensitivity analysis
type SensitivityFormatter interface {
	FormatSensitivityAnalysis(analysis interface{}) (string, error)
	Name() string
}

// SensitivityConsoleFormatter formats sensitivity analysis output for console
type SensitivityConsoleFormatter struct{}

func (scf SensitivityConsoleFormatter) Name() string { return "console" }

func (scf SensitivityConsoleFormatter) FormatSensitivityAnalysis(analysis interface{}) (string, error) {
	var buf bytes.Buffer

	switch a := analysis.(type) {
	case *domain.ParameterSensitivityAnalysis:
		return scf.formatSingleAnalysis(&buf, a)
	case *domain.SensitivityMatrix:
		return scf.formatMatrixAnalysis(&buf, a)
	default:
		return "", fmt.Errorf("unsupported analysis type: %T", analysis)
	}
}

func (scf SensitivityConsoleFormatter) formatSingleAnalysis(buf *bytes.Buffer, analysis *domain.ParameterSensitivityAnalysis) (string, error) {
	if len(analysis.Parameters) == 0 || len(analysis.Results) == 0 {
		return "", fmt.Errorf("no parameters or results in analysis")
	}

	param := analysis.Parameters[0]

	fmt.Fprintf(buf, "SENSITIVITY ANALYSIS: %s\n", strings.ToUpper(strings.ReplaceAll(param.Name, "_", " ")))
	fmt.Fprintf(buf, "=================================================================\n")
	fmt.Fprintf(buf, "Base Case: %s = %.1f%%\n", param.Name, param.BaseValue.Mul(decimal.NewFromInt(100)).InexactFloat64())
	fmt.Fprintf(buf, "Range: %.1f%% to %.1f%% (%d steps)\n",
		param.MinValue.Mul(decimal.NewFromInt(100)).InexactFloat64(),
		param.MaxValue.Mul(decimal.NewFromInt(100)).InexactFloat64(),
		param.Steps)
	fmt.Fprintf(buf, "Description: %s\n", param.Description)
	fmt.Fprintln(buf)

	// Find base case result
	var baseResult *domain.SensitivityResult
	minDiff := decimal.NewFromFloat(999999)

	for i := range analysis.Results {
		diff := analysis.Results[i].ParameterValues[param.Name].Sub(param.BaseValue).Abs()
		if diff.LessThan(minDiff) {
			minDiff = diff
			baseResult = &analysis.Results[i]
		}
	}

	if baseResult == nil {
		return "", fmt.Errorf("could not find base case result")
	}

	// Results table
	fmt.Fprintf(buf, "%-12s %-12s %-12s %-12s %-12s %-12s\n",
		param.Name, "Year 5 Net", "Year 10 Net", "TSP Longevity", "Lifetime Income", "IRMAA Cost")
	fmt.Fprintln(buf, strings.Repeat("-", 80))

	for _, result := range analysis.Results {
		paramValue := result.ParameterValues[param.Name]
		isBaseCase := paramValue.Equal(param.BaseValue)

		paramValueStr := fmt.Sprintf("%.1f%%", paramValue.Mul(decimal.NewFromInt(100)).InexactFloat64())
		if isBaseCase {
			paramValueStr += " ← BASE"
		}

		fmt.Fprintf(buf, "%-12s %-12s %-12s %-12d %-12s %-12s\n",
			paramValueStr,
			FormatCurrency(result.KeyMetrics.Year5NetIncome),
			FormatCurrency(result.KeyMetrics.Year10NetIncome),
			result.KeyMetrics.TSPLongevity,
			FormatCurrency(result.KeyMetrics.TotalLifetimeIncome),
			FormatCurrency(result.KeyMetrics.IRMAATotalCost))
	}

	fmt.Fprintln(buf)

	// Sensitivity analysis
	fmt.Fprintln(buf, "SENSITIVITY:")

	// Calculate sensitivity metrics
	maxSensitivity := decimal.Zero

	for _, result := range analysis.Results {
		if result.ParameterValues[param.Name].Equal(param.BaseValue) {
			continue // Skip base case
		}

		paramChange := result.ParameterValues[param.Name].Sub(param.BaseValue).Div(param.BaseValue).Mul(decimal.NewFromInt(100))
		netIncomeChange := result.KeyMetrics.Year5NetIncome.Sub(baseResult.KeyMetrics.Year5NetIncome)
		netIncomeChangePct := netIncomeChange.Div(baseResult.KeyMetrics.Year5NetIncome).Mul(decimal.NewFromInt(100))

		sensitivityScore := netIncomeChangePct.Abs().Div(paramChange.Abs())

		if sensitivityScore.GreaterThan(maxSensitivity) {
			maxSensitivity = sensitivityScore
		}

		fmt.Fprintf(buf, "  %+.1f%% %s → %s Year 5 income (%+.1f%%)\n",
			paramChange.InexactFloat64(),
			param.Name,
			FormatCurrency(netIncomeChange),
			netIncomeChangePct.InexactFloat64())
	}

	// TSP longevity sensitivity
	longevitySensitivity := decimal.Zero
	for _, result := range analysis.Results {
		if result.ParameterValues[param.Name].Equal(param.BaseValue) {
			continue
		}

		paramChange := result.ParameterValues[param.Name].Sub(param.BaseValue).Div(param.BaseValue).Mul(decimal.NewFromInt(100))
		longevityChange := decimal.NewFromInt(int64(result.KeyMetrics.TSPLongevity - baseResult.KeyMetrics.TSPLongevity))

		if longevityChange.Abs().GreaterThan(longevitySensitivity) {
			longevitySensitivity = longevityChange.Abs()
		}

		fmt.Fprintf(buf, "  %+.1f%% %s → %+d years TSP longevity\n",
			paramChange.InexactFloat64(),
			param.Name,
			result.KeyMetrics.TSPLongevity-baseResult.KeyMetrics.TSPLongevity)
	}

	fmt.Fprintln(buf)

	// Risk assessment
	riskLevel := analysis.Summary.RiskLevel
	riskEmoji := ""
	switch riskLevel {
	case "LOW":
		riskEmoji = "✅"
	case "MEDIUM":
		riskEmoji = "⚠️"
	case "HIGH":
		riskEmoji = "🔴"
	case "CRITICAL":
		riskEmoji = "🚨"
	}

	fmt.Fprintf(buf, "RISK LEVEL: %s %s\n", riskEmoji, riskLevel)
	fmt.Fprintln(buf)

	// Recommendations
	fmt.Fprintln(buf, "RECOMMENDATIONS:")
	for _, rec := range analysis.Summary.Recommendations {
		fmt.Fprintf(buf, "  • %s\n", rec)
	}

	return buf.String(), nil
}

func (scf SensitivityConsoleFormatter) formatMatrixAnalysis(buf *bytes.Buffer, matrix *domain.SensitivityMatrix) (string, error) {
	fmt.Fprintf(buf, "SENSITIVITY MATRIX ANALYSIS\n")
	fmt.Fprintf(buf, "=================================================================\n")
	fmt.Fprintf(buf, "Parameter 1: %s (%.1f%% to %.1f%%)\n",
		matrix.Parameter1.Name,
		matrix.Parameter1.MinValue.Mul(decimal.NewFromInt(100)).InexactFloat64(),
		matrix.Parameter1.MaxValue.Mul(decimal.NewFromInt(100)).InexactFloat64())
	fmt.Fprintf(buf, "Parameter 2: %s (%.1f%% to %.1f%%)\n",
		matrix.Parameter2.Name,
		matrix.Parameter2.MinValue.Mul(decimal.NewFromInt(100)).InexactFloat64(),
		matrix.Parameter2.MaxValue.Mul(decimal.NewFromInt(100)).InexactFloat64())
	fmt.Fprintln(buf)

	// Matrix table
	fmt.Fprintf(buf, "%-12s", matrix.Parameter2.Name)
	for j := range matrix.MatrixResults[0] {
		param2Value := matrix.MatrixResults[0][j].ParameterValues[matrix.Parameter2.Name]
		fmt.Fprintf(buf, " %-10s", fmt.Sprintf("%.1f%%", param2Value.Mul(decimal.NewFromInt(100)).InexactFloat64()))
	}
	fmt.Fprintln(buf)

	fmt.Fprintf(buf, "%s", matrix.Parameter1.Name)
	for range matrix.MatrixResults[0] {
		fmt.Fprintf(buf, " %-10s", "Year 5 Net")
	}
	fmt.Fprintln(buf)

	fmt.Fprintln(buf, strings.Repeat("-", 12+11*len(matrix.MatrixResults[0])))

	for i := range matrix.MatrixResults {
		param1Value := matrix.MatrixResults[i][0].ParameterValues[matrix.Parameter1.Name]
		fmt.Fprintf(buf, "%-12s", fmt.Sprintf("%.1f%%", param1Value.Mul(decimal.NewFromInt(100)).InexactFloat64()))

		for j := range matrix.MatrixResults[i] {
			result := matrix.MatrixResults[i][j]
			fmt.Fprintf(buf, " %-10s", FormatCurrency(result.KeyMetrics.Year5NetIncome))
		}
		fmt.Fprintln(buf)
	}

	fmt.Fprintln(buf)

	// Summary
	fmt.Fprintf(buf, "MOST SENSITIVE COMBINATION: %s\n", matrix.Summary.MostSensitiveCombination)
	fmt.Fprintf(buf, "INTERACTION EFFECT: %.2f\n", matrix.Summary.InteractionEffect.InexactFloat64())
	fmt.Fprintf(buf, "RISK LEVEL: %s\n", matrix.Summary.RiskLevel)
	fmt.Fprintln(buf)

	// Recommendations
	fmt.Fprintln(buf, "RECOMMENDATIONS:")
	for _, rec := range matrix.Summary.Recommendations {
		fmt.Fprintf(buf, "  • %s\n", rec)
	}

	return buf.String(), nil
}

// SensitivityCSVFormatter formats sensitivity analysis output as CSV
type SensitivityCSVFormatter struct{}

func (scf SensitivityCSVFormatter) Name() string { return "csv" }

func (scf SensitivityCSVFormatter) FormatSensitivityAnalysis(analysis interface{}) (string, error) {
	var buf bytes.Buffer

	switch a := analysis.(type) {
	case *domain.ParameterSensitivityAnalysis:
		return scf.formatSingleAnalysisCSV(&buf, a)
	case *domain.SensitivityMatrix:
		return scf.formatMatrixAnalysisCSV(&buf, a)
	default:
		return "", fmt.Errorf("unsupported analysis type: %T", analysis)
	}
}

func (scf SensitivityCSVFormatter) formatSingleAnalysisCSV(buf *bytes.Buffer, analysis *domain.ParameterSensitivityAnalysis) (string, error) {
	if len(analysis.Parameters) == 0 || len(analysis.Results) == 0 {
		return "", fmt.Errorf("no parameters or results in analysis")
	}

	param := analysis.Parameters[0]

	// CSV header
	fmt.Fprintf(buf, "parameter_name,parameter_value,year_5_net_income,year_10_net_income,tsp_longevity,total_lifetime_income,irmaa_total_cost\n")

	// CSV data
	for _, result := range analysis.Results {
		paramValue := result.ParameterValues[param.Name]
		fmt.Fprintf(buf, "%s,%.4f,%s,%s,%d,%s,%s\n",
			param.Name,
			paramValue.InexactFloat64(),
			result.KeyMetrics.Year5NetIncome.String(),
			result.KeyMetrics.Year10NetIncome.String(),
			result.KeyMetrics.TSPLongevity,
			result.KeyMetrics.TotalLifetimeIncome.String(),
			result.KeyMetrics.IRMAATotalCost.String())
	}

	return buf.String(), nil
}

func (scf SensitivityCSVFormatter) formatMatrixAnalysisCSV(buf *bytes.Buffer, matrix *domain.SensitivityMatrix) (string, error) {
	// CSV header
	fmt.Fprintf(buf, "parameter_1_name,parameter_1_value,parameter_2_name,parameter_2_value,year_5_net_income,year_10_net_income,tsp_longevity,total_lifetime_income,irmaa_total_cost\n")

	// CSV data
	for i := range matrix.MatrixResults {
		for j := range matrix.MatrixResults[i] {
			result := matrix.MatrixResults[i][j]
			param1Value := result.ParameterValues[matrix.Parameter1.Name]
			param2Value := result.ParameterValues[matrix.Parameter2.Name]

			fmt.Fprintf(buf, "%s,%.4f,%s,%.4f,%s,%s,%d,%s,%s\n",
				matrix.Parameter1.Name,
				param1Value.InexactFloat64(),
				matrix.Parameter2.Name,
				param2Value.InexactFloat64(),
				result.KeyMetrics.Year5NetIncome.String(),
				result.KeyMetrics.Year10NetIncome.String(),
				result.KeyMetrics.TSPLongevity,
				result.KeyMetrics.TotalLifetimeIncome.String(),
				result.KeyMetrics.IRMAATotalCost.String())
		}
	}

	return buf.String(), nil
}

// SensitivityJSONFormatter formats sensitivity analysis output as JSON
type SensitivityJSONFormatter struct{}

func (sjf SensitivityJSONFormatter) Name() string { return "json" }

func (sjf SensitivityJSONFormatter) FormatSensitivityAnalysis(analysis interface{}) (string, error) {
	// For now, return a simple string representation
	// In a real implementation, this would use json.Marshal
	switch a := analysis.(type) {
	case *domain.ParameterSensitivityAnalysis:
		return fmt.Sprintf("Sensitivity Analysis: %s with %d results",
			a.BaseScenarioName, len(a.Results)), nil
	case *domain.SensitivityMatrix:
		return fmt.Sprintf("Sensitivity Matrix: %s vs %s with %dx%d results",
			a.Parameter1.Name, a.Parameter2.Name,
			len(a.MatrixResults), len(a.MatrixResults[0])), nil
	default:
		return "", fmt.Errorf("unsupported analysis type: %T", analysis)
	}
}

// NewSensitivityFormatter creates a sensitivity formatter based on the format name
func NewSensitivityFormatter(format string) SensitivityFormatter {
	switch NormalizeFormatName(format) {
	case "console", "table":
		return SensitivityConsoleFormatter{}
	case "csv":
		return SensitivityCSVFormatter{}
	case "json":
		return SensitivityJSONFormatter{}
	default:
		return SensitivityConsoleFormatter{} // Default to console
	}
}
