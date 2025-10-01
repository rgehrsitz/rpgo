package breakeven

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// TableFormatter formats optimization results as a console table
type TableFormatter struct{}

// Format generates a formatted table for optimization result
func (tf *TableFormatter) Format(result *OptimizationResult) string {
	var sb strings.Builder

	sb.WriteString("BREAK-EVEN OPTIMIZATION RESULTS\n")
	sb.WriteString(strings.Repeat("=", 80) + "\n")

	// Optimization metadata
	sb.WriteString(fmt.Sprintf("Optimization Target: %s\n", result.Request.Target))
	sb.WriteString(fmt.Sprintf("Optimization Goal:   %s\n", result.Request.Goal))
	sb.WriteString(fmt.Sprintf("Participant:         %s\n", result.Request.Constraints.Participant))
	sb.WriteString(fmt.Sprintf("Status:              %s\n", tf.formatStatus(result.Success)))
	sb.WriteString(fmt.Sprintf("Iterations:          %d\n", result.Iterations))
	if result.ConvergenceInfo != "" {
		sb.WriteString(fmt.Sprintf("Convergence:         %s\n", result.ConvergenceInfo))
	}
	sb.WriteString("\n")

	// Optimal parameters found
	sb.WriteString("OPTIMAL PARAMETERS\n")
	sb.WriteString(strings.Repeat("-", 80) + "\n")

	if result.OptimalRetirementDate != nil {
		sb.WriteString(fmt.Sprintf("Retirement Date:     %s\n", result.OptimalRetirementDate.Format("January 2, 2006")))
	}
	if result.OptimalTSPRate != nil {
		pct := result.OptimalTSPRate.Mul(decimal.NewFromInt(100))
		sb.WriteString(fmt.Sprintf("TSP Withdrawal Rate: %s%%\n", pct.StringFixed(2)))
	}
	if result.OptimalSSAge != nil {
		sb.WriteString(fmt.Sprintf("SS Claiming Age:     %d\n", *result.OptimalSSAge))
	}
	if result.OptimalTSPBalance != nil {
		sb.WriteString(fmt.Sprintf("Required TSP Balance: $%s\n", tf.formatCurrency(*result.OptimalTSPBalance)))
	}
	sb.WriteString("\n")

	// Results at optimal parameters
	sb.WriteString("PROJECTED RESULTS\n")
	sb.WriteString(strings.Repeat("-", 80) + "\n")
	sb.WriteString(fmt.Sprintf("First Year Net Income: $%s\n", tf.formatCurrency(result.FirstYearNetIncome)))
	sb.WriteString(fmt.Sprintf("Lifetime Income:       $%s\n", tf.formatCurrency(result.LifetimeIncome)))
	sb.WriteString(fmt.Sprintf("TSP Longevity:         %d years\n", result.TSPLongevity))
	sb.WriteString(fmt.Sprintf("Lifetime Taxes:        $%s\n", tf.formatCurrency(result.LifetimeTaxes)))
	sb.WriteString("\n")

	// Comparison to base (if applicable)
	if result.BaseScenarioSummary != nil {
		sb.WriteString("COMPARISON TO BASE SCENARIO\n")
		sb.WriteString(strings.Repeat("-", 80) + "\n")
		if !result.IncomeDiffFromBase.IsZero() {
			symbol := tf.deltaSymbol(result.IncomeDiffFromBase)
			sb.WriteString(fmt.Sprintf("Lifetime Income Change: %s$%s\n",
				symbol, tf.formatCurrency(result.IncomeDiffFromBase.Abs())))
		}
		if !result.TaxDiffFromBase.IsZero() {
			symbol := tf.deltaSymbol(result.TaxDiffFromBase.Neg()) // Lower taxes are better
			sb.WriteString(fmt.Sprintf("Tax Impact:             %s$%s\n",
				symbol, tf.formatCurrency(result.TaxDiffFromBase.Abs())))
		}
		sb.WriteString("\n")
	}

	// Goal-specific information
	if result.Request.Goal == GoalMatchIncome && result.Request.Constraints.TargetIncome != nil {
		sb.WriteString("TARGET INCOME MATCH\n")
		sb.WriteString(strings.Repeat("-", 80) + "\n")
		sb.WriteString(fmt.Sprintf("Target Income:    $%s\n", tf.formatCurrency(*result.Request.Constraints.TargetIncome)))
		sb.WriteString(fmt.Sprintf("Achieved Income:  $%s\n", tf.formatCurrency(result.FirstYearNetIncome)))
		diff := result.FirstYearNetIncome.Sub(*result.Request.Constraints.TargetIncome)
		sb.WriteString(fmt.Sprintf("Difference:       %s$%s\n", tf.deltaSymbol(diff), tf.formatCurrency(diff.Abs())))
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatMultiDimensional formats results from multiple optimizations
func (tf *TableFormatter) FormatMultiDimensional(result *MultiDimensionalResult) string {
	var sb strings.Builder

	sb.WriteString("MULTI-DIMENSIONAL OPTIMIZATION RESULTS\n")
	sb.WriteString(strings.Repeat("=", 80) + "\n\n")

	// Summary table of all results
	sb.WriteString("SUMMARY OF ALL OPTIMIZATIONS\n")
	sb.WriteString(strings.Repeat("-", 80) + "\n")
	sb.WriteString(fmt.Sprintf("%-20s %15s %15s %12s %15s\n",
		"Optimization", "Lifetime Income", "TSP Longevity", "Lifetime Taxes", "First Year Income"))
	sb.WriteString(strings.Repeat("-", 80) + "\n")

	for _, res := range result.Results {
		targetStr := string(res.Request.Target)
		sb.WriteString(fmt.Sprintf("%-20s %15s %15s %12s %15s\n",
			tf.truncate(targetStr, 20),
			"$"+tf.formatShort(res.LifetimeIncome),
			fmt.Sprintf("%d years", res.TSPLongevity),
			"$"+tf.formatShort(res.LifetimeTaxes),
			"$"+tf.formatShort(res.FirstYearNetIncome)))
	}
	sb.WriteString("\n")

	// Best scenarios
	sb.WriteString("BEST SCENARIOS\n")
	sb.WriteString(strings.Repeat("-", 80) + "\n")

	if result.BestByIncome != nil {
		sb.WriteString(fmt.Sprintf("Best Income:     %s ($%s lifetime)\n",
			result.BestByIncome.Request.Target,
			tf.formatCurrency(result.BestByIncome.LifetimeIncome)))
	}
	if result.BestByLongevity != nil {
		sb.WriteString(fmt.Sprintf("Best Longevity:  %s (%d years)\n",
			result.BestByLongevity.Request.Target,
			result.BestByLongevity.TSPLongevity))
	}
	if result.BestByTaxes != nil {
		sb.WriteString(fmt.Sprintf("Lowest Taxes:    %s ($%s lifetime)\n",
			result.BestByTaxes.Request.Target,
			tf.formatCurrency(result.BestByTaxes.LifetimeTaxes)))
	}
	sb.WriteString("\n")

	// Recommendations
	if len(result.Recommendations) > 0 {
		sb.WriteString("RECOMMENDATIONS\n")
		sb.WriteString(strings.Repeat("-", 80) + "\n")
		for _, rec := range result.Recommendations {
			sb.WriteString(fmt.Sprintf("• %s\n", rec))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// JSONFormatter formats results as JSON
type JSONFormatter struct {
	Pretty bool
}

// Format generates JSON output
func (jf *JSONFormatter) Format(result *OptimizationResult) (string, error) {
	var data []byte
	var err error

	if jf.Pretty {
		data, err = json.MarshalIndent(result, "", "  ")
	} else {
		data, err = json.Marshal(result)
	}

	if err != nil {
		return "", err
	}

	return string(data), nil
}

// FormatMultiDimensional formats multi-dimensional results as JSON
func (jf *JSONFormatter) FormatMultiDimensional(result *MultiDimensionalResult) (string, error) {
	var data []byte
	var err error

	if jf.Pretty {
		data, err = json.MarshalIndent(result, "", "  ")
	} else {
		data, err = json.Marshal(result)
	}

	if err != nil {
		return "", err
	}

	return string(data), nil
}

// Helper methods

func (tf *TableFormatter) formatStatus(success bool) string {
	if success {
		return "✓ Converged"
	}
	return "⚠ Did not converge"
}

func (tf *TableFormatter) formatCurrency(d decimal.Decimal) string {
	return d.StringFixed(2)
}

func (tf *TableFormatter) formatShort(d decimal.Decimal) string {
	if d.Abs().GreaterThanOrEqual(decimal.NewFromInt(1000000)) {
		millions := d.Div(decimal.NewFromInt(1000000))
		return millions.StringFixed(2) + "M"
	} else if d.Abs().GreaterThanOrEqual(decimal.NewFromInt(1000)) {
		thousands := d.Div(decimal.NewFromInt(1000))
		return thousands.StringFixed(1) + "K"
	}
	return d.StringFixed(0)
}

func (tf *TableFormatter) deltaSymbol(delta decimal.Decimal) string {
	if delta.IsPositive() {
		return "+"
	} else if delta.IsNegative() {
		return ""
	}
	return " "
}

func (tf *TableFormatter) truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
