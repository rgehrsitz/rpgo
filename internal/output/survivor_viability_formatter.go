package output

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// SurvivorViabilityConsoleFormatter formats survivor viability analysis output for console
type SurvivorViabilityConsoleFormatter struct{}

func (svf SurvivorViabilityConsoleFormatter) Name() string { return "console" }

func (svf SurvivorViabilityConsoleFormatter) FormatSurvivorViabilityAnalysis(analysis *domain.SurvivorViabilityAnalysis) (string, error) {
	var buf bytes.Buffer

	// Header
	fmt.Fprintln(&buf, "SURVIVOR VIABILITY ANALYSIS")
	fmt.Fprintln(&buf, strings.Repeat("=", 80))
	fmt.Fprintf(&buf, "Scenario: %s\n", analysis.ScenarioName)
	fmt.Fprintf(&buf, "Deceased: %s (age %d in %d)\n", analysis.DeceasedParticipant, analysis.DeathAge, analysis.DeathYear)
	fmt.Fprintf(&buf, "Survivor: %s (age %d at time of death)\n", analysis.SurvivorParticipant, analysis.SurvivorAge)
	fmt.Fprintln(&buf)

	// Pre-death analysis
	fmt.Fprintf(&buf, "PRE-DEATH (%s, %d):\n", analysis.PreDeathAnalysis.FilingStatus, analysis.PreDeathAnalysis.Year)
	fmt.Fprintf(&buf, "  Combined Net Income:    %s/year\n", FormatCurrency(analysis.PreDeathAnalysis.NetIncome))
	fmt.Fprintf(&buf, "  Monthly:                %s\n", FormatCurrency(analysis.PreDeathAnalysis.MonthlyIncome))
	fmt.Fprintf(&buf, "  Healthcare Costs:       %s/year\n", FormatCurrency(analysis.PreDeathAnalysis.HealthcareCosts))
	fmt.Fprintf(&buf, "  Tax Impact:             %s/year\n", FormatCurrency(analysis.PreDeathAnalysis.TaxImpact))
	fmt.Fprintf(&buf, "  IRMAA Risk:             %s\n", analysis.PreDeathAnalysis.IRMAARisk)
	fmt.Fprintln(&buf)

	// Post-death analysis
	fmt.Fprintf(&buf, "POST-DEATH (%s, %d+):\n", analysis.PostDeathAnalysis.FilingStatus, analysis.PostDeathAnalysis.Year)
	fmt.Fprintf(&buf, "  %s's Net Income:       %s/year (%.1f%%)\n",
		analysis.SurvivorParticipant,
		FormatCurrency(analysis.PostDeathAnalysis.NetIncome),
		analysis.PostDeathAnalysis.NetIncome.Div(analysis.PreDeathAnalysis.NetIncome).Mul(decimal.NewFromInt(100)).InexactFloat64())
	fmt.Fprintf(&buf, "  Monthly:                %s (%.1f%%)\n",
		FormatCurrency(analysis.PostDeathAnalysis.MonthlyIncome),
		analysis.PostDeathAnalysis.MonthlyIncome.Div(analysis.PreDeathAnalysis.MonthlyIncome).Mul(decimal.NewFromInt(100)).InexactFloat64())
	fmt.Fprintf(&buf, "  Healthcare Costs:       %s/year\n", FormatCurrency(analysis.PostDeathAnalysis.HealthcareCosts))
	fmt.Fprintf(&buf, "  Tax Impact:             %s/year\n", FormatCurrency(analysis.PostDeathAnalysis.TaxImpact))
	fmt.Fprintf(&buf, "  IRMAA Risk:             %s\n", analysis.PostDeathAnalysis.IRMAARisk)
	fmt.Fprintln(&buf)

	// Income sources breakdown
	fmt.Fprintf(&buf, "  Income Sources:\n")
	fmt.Fprintf(&buf, "    %s's Pension:        %s\n", analysis.SurvivorParticipant, FormatCurrency(analysis.PostDeathAnalysis.IncomeSources.SurvivorPension))
	fmt.Fprintf(&buf, "    %s's SS:             %s\n", analysis.SurvivorParticipant, FormatCurrency(analysis.PostDeathAnalysis.IncomeSources.SurvivorSS))
	fmt.Fprintf(&buf, "    %s's Survivor SS:    %s\n", analysis.DeceasedParticipant, FormatCurrency(analysis.PostDeathAnalysis.IncomeSources.DeceasedSurvivorSS))
	fmt.Fprintf(&buf, "    TSP Withdrawal:       %s\n", FormatCurrency(analysis.PostDeathAnalysis.IncomeSources.TSPWithdrawals))
	fmt.Fprintf(&buf, "    Other Income:         %s\n", FormatCurrency(analysis.PostDeathAnalysis.IncomeSources.OtherIncome))
	fmt.Fprintf(&buf, "    Total Income:         %s\n", FormatCurrency(analysis.PostDeathAnalysis.IncomeSources.TotalIncome))
	fmt.Fprintln(&buf)

	// Filing status change
	fmt.Fprintf(&buf, "  Filing Status:        %s (starting %d)\n", analysis.PostDeathAnalysis.FilingStatus, analysis.PostDeathAnalysis.Year)
	fmt.Fprintf(&buf, "  Tax Impact Change:     %s/year\n", FormatCurrency(analysis.ViabilityAssessment.TaxImpactChange))
	fmt.Fprintln(&buf)

	// Viability assessment
	fmt.Fprintln(&buf, "VIABILITY ASSESSMENT:")
	fmt.Fprintf(&buf, "  Survivor Income vs. Target (%.0f%% of couple):\n",
		analysis.ViabilityAssessment.TargetIncome.Div(analysis.PreDeathAnalysis.NetIncome).Mul(decimal.NewFromInt(100)).InexactFloat64())
	fmt.Fprintf(&buf, "    Target:    %s\n", FormatCurrency(analysis.ViabilityAssessment.TargetIncome))
	fmt.Fprintf(&buf, "    Actual:    %s\n", FormatCurrency(analysis.ViabilityAssessment.ActualIncome))
	fmt.Fprintf(&buf, "    Shortfall: %s (%.1f%%)  %s %s\n",
		FormatCurrency(analysis.ViabilityAssessment.IncomeShortfall),
		analysis.ViabilityAssessment.ShortfallPercentage.InexactFloat64(),
		analysis.ViabilityAssessment.ViabilityColor,
		analysis.ViabilityAssessment.ViabilityScore)
	fmt.Fprintln(&buf)

	// Key metrics
	fmt.Fprintf(&buf, "  Key Changes:\n")
	fmt.Fprintf(&buf, "    Tax Impact:           %s/year\n", FormatCurrency(analysis.ViabilityAssessment.TaxImpactChange))
	fmt.Fprintf(&buf, "    Healthcare Costs:     %s/year\n", FormatCurrency(analysis.ViabilityAssessment.HealthcareCostChange))
	fmt.Fprintf(&buf, "    IRMAA Risk:           %s\n", analysis.ViabilityAssessment.IRMAARiskChange)
	fmt.Fprintln(&buf)

	// Life insurance needs
	fmt.Fprintln(&buf, "LIFE INSURANCE NEEDS:")
	fmt.Fprintf(&buf, "  To bridge %s/year gap for %d years:\n",
		FormatCurrency(analysis.LifeInsuranceNeeds.AnnualShortfall),
		analysis.LifeInsuranceNeeds.YearsToBridge)
	fmt.Fprintf(&buf, "    Present Value (%.0f%% discount): %s\n",
		analysis.LifeInsuranceNeeds.DiscountRate.Mul(decimal.NewFromInt(100)).InexactFloat64(),
		FormatCurrency(analysis.LifeInsuranceNeeds.PresentValue))
	fmt.Fprintf(&buf, "    Recommended coverage: %s\n", FormatCurrency(analysis.LifeInsuranceNeeds.RecommendedCoverage))
	fmt.Fprintln(&buf)

	// Alternative strategies
	if len(analysis.LifeInsuranceNeeds.AlternativeStrategies) > 0 {
		fmt.Fprintln(&buf, "ALTERNATIVE STRATEGIES:")
		for _, strategy := range analysis.LifeInsuranceNeeds.AlternativeStrategies {
			fmt.Fprintf(&buf, "  %s\n", strategy)
		}
		fmt.Fprintln(&buf)
	}

	// Recommendations
	if len(analysis.Recommendations) > 0 {
		fmt.Fprintln(&buf, "RECOMMENDATIONS:")
		for _, rec := range analysis.Recommendations {
			fmt.Fprintf(&buf, "  %s\n", rec)
		}
		fmt.Fprintln(&buf)
	}

	return buf.String(), nil
}

// FormatSurvivorViabilityAnalysisJSON formats survivor viability analysis for JSON output
func (svf SurvivorViabilityConsoleFormatter) FormatSurvivorViabilityAnalysisJSON(analysis *domain.SurvivorViabilityAnalysis) (string, error) {
	// This would use json.Marshal in a real implementation
	// For now, return a simple string representation
	return fmt.Sprintf("Survivor Viability Analysis: %s -> %s (%.1f%% shortfall)",
		analysis.DeceasedParticipant,
		analysis.SurvivorParticipant,
		analysis.ViabilityAssessment.ShortfallPercentage.InexactFloat64()), nil
}
