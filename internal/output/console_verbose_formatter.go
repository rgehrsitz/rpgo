package output

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// ConsoleVerboseFormatter renders the original detailed console report via the pluggable interface.
type ConsoleVerboseFormatter struct{}

func (c ConsoleVerboseFormatter) Name() string { return "console" }

func (c ConsoleVerboseFormatter) Format(results *domain.ScenarioComparison) ([]byte, error) {
	var buf bytes.Buffer

	fmt.Fprintln(&buf, "=================================================================================")
	fmt.Fprintln(&buf, "DETAILED FERS RETIREMENT INCOME ANALYSIS")
	fmt.Fprintln(&buf, "=================================================================================")
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "KEY ASSUMPTIONS:")
	for _, a := range DefaultAssumptions {
		fmt.Fprintf(&buf, "• %s\n", a)
	}
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "CURRENT NET INCOME BREAKDOWN (Pre-Retirement)")
	fmt.Fprintln(&buf, "=============================================")
	fmt.Fprintf(&buf, "Combined Gross Salary: %s\n", FormatCurrency(decimal.NewFromFloat(367399.00)))
	fmt.Fprintf(&buf, "Combined Net Income:  %s\n", FormatCurrency(results.BaselineNetIncome))
	fmt.Fprintf(&buf, "Monthly Net Income:   %s\n", FormatCurrency(results.BaselineNetIncome.Div(decimal.NewFromInt(12))))
	fmt.Fprintln(&buf)

	// Detailed comparison (condensed from original GenerateDetailedComparison)
	writeDetailedComparison(&buf, results)

	for i, scenario := range results.Scenarios {
		fmt.Fprintf(&buf, "SCENARIO %d: %s\n", i+1, scenario.Name)
		fmt.Fprintln(&buf, strings.Repeat("=", 50))
		// first retirement year
		var firstRetirementYear domain.AnnualCashFlow
		var firstRetirementYearIndex int
		found := false
		for yIdx, y := range scenario.Projection {
			if y.IsRetired {
				firstRetirementYear = y
				firstRetirementYearIndex = yIdx
				found = true
				break
			}
		}
		if found {
			actualYear := 2025 + firstRetirementYearIndex
			fmt.Fprintf(&buf, "FIRST RETIREMENT YEAR (%d) INCOME BREAKDOWN:\n", actualYear)
			fmt.Fprintln(&buf, "(Note: Amounts shown are current-year cash received - may be partial year)")
			fmt.Fprintln(&buf, "----------------------------------------")
			fmt.Fprintln(&buf, "INCOME SOURCES:")
			fmt.Fprintf(&buf, "  Robert's Salary:        %s\n", FormatCurrency(firstRetirementYear.SalaryRobert))
			fmt.Fprintf(&buf, "  Dawn's Salary:          %s\n", FormatCurrency(firstRetirementYear.SalaryDawn))
			fmt.Fprintf(&buf, "  Robert's FERS Pension:  %s\n", FormatCurrency(firstRetirementYear.PensionRobert))
			fmt.Fprintf(&buf, "  Dawn's FERS Pension:    %s\n", FormatCurrency(firstRetirementYear.PensionDawn))
			fmt.Fprintf(&buf, "  Robert's TSP Withdrawal: %s\n", FormatCurrency(firstRetirementYear.TSPWithdrawalRobert))
			fmt.Fprintf(&buf, "  Dawn's TSP Withdrawal:   %s\n", FormatCurrency(firstRetirementYear.TSPWithdrawalDawn))
			fmt.Fprintf(&buf, "  Robert's Social Security: %s\n", FormatCurrency(firstRetirementYear.SSBenefitRobert))
			fmt.Fprintf(&buf, "  Dawn's Social Security:   %s\n", FormatCurrency(firstRetirementYear.SSBenefitDawn))
			fmt.Fprintf(&buf, "  Robert's FERS SRS:       %s\n", FormatCurrency(firstRetirementYear.FERSSupplementRobert))
			fmt.Fprintf(&buf, "  Dawn's FERS SRS:         %s\n", FormatCurrency(firstRetirementYear.FERSSupplementDawn))
			fmt.Fprintf(&buf, "  TOTAL GROSS INCOME:      %s\n", FormatCurrency(firstRetirementYear.TotalGrossIncome))
			fmt.Fprintln(&buf)
			fmt.Fprintln(&buf, "DEDUCTIONS & TAXES:")
			fmt.Fprintf(&buf, "  Federal Tax:            %s\n", FormatCurrency(firstRetirementYear.FederalTax))
			fmt.Fprintf(&buf, "  State Tax:              %s\n", FormatCurrency(firstRetirementYear.StateTax))
			fmt.Fprintf(&buf, "  Local Tax:              %s\n", FormatCurrency(firstRetirementYear.LocalTax))
			fmt.Fprintf(&buf, "  FICA Tax:               %s\n", FormatCurrency(firstRetirementYear.FICATax))
			fmt.Fprintf(&buf, "  TSP Contributions:      %s\n", FormatCurrency(firstRetirementYear.TSPContributions))
			fmt.Fprintf(&buf, "  FEHB Premium:           %s\n", FormatCurrency(firstRetirementYear.FEHBPremium))
			fmt.Fprintf(&buf, "  Medicare Premium:       %s\n", FormatCurrency(firstRetirementYear.MedicarePremium))
			fmt.Fprintf(&buf, "  TOTAL DEDUCTIONS:       %s\n", FormatCurrency(firstRetirementYear.CalculateTotalDeductions()))
			fmt.Fprintln(&buf)
			fmt.Fprintln(&buf, "NET INCOME COMPARISON:")
			fmt.Fprintln(&buf, "----------------------")
			fmt.Fprintf(&buf, "  Current Net Income:     %s\n", FormatCurrency(results.BaselineNetIncome))
			fmt.Fprintf(&buf, "  Retirement Net Income:  %s\n", FormatCurrency(firstRetirementYear.NetIncome))
			change := firstRetirementYear.NetIncome.Sub(results.BaselineNetIncome)
			percentageChange := change.Div(results.BaselineNetIncome).Mul(decimal.NewFromInt(100))
			if change.GreaterThan(decimal.Zero) {
				fmt.Fprintf(&buf, "  CHANGE: +%s (+%s)\n", FormatCurrency(change), FormatPercentage(percentageChange))
			} else {
				fmt.Fprintf(&buf, "  CHANGE: %s (%s)\n", FormatCurrency(change), FormatPercentage(percentageChange))
			}
			monthlyChange := change.Div(decimal.NewFromInt(12))
			if monthlyChange.GreaterThan(decimal.Zero) {
				fmt.Fprintf(&buf, "  Monthly Change: +%s\n", FormatCurrency(monthlyChange))
			} else {
				fmt.Fprintf(&buf, "  Monthly Change: %s\n", FormatCurrency(monthlyChange))
			}
			fmt.Fprintln(&buf, "RETIREMENT STATUS:")
			fmt.Fprintf(&buf, "  Is Retired:             %t\n", firstRetirementYear.IsRetired)
			fmt.Fprintf(&buf, "  Medicare Eligible:      %t\n", firstRetirementYear.IsMedicareEligible)
			fmt.Fprintf(&buf, "  RMD Year:               %t\n", firstRetirementYear.IsRMDYear)
			fmt.Fprintf(&buf, "  Robert's Age:           %d\n", firstRetirementYear.AgeRobert)
			fmt.Fprintf(&buf, "  Dawn's Age:             %d\n", firstRetirementYear.AgeDawn)
			fmt.Fprintln(&buf)
		}

		// long term projection summary
		fmt.Fprintln(&buf, "LONG-TERM PROJECTION:")
		fmt.Fprintln(&buf, "---------------------")
		fmt.Fprintf(&buf, "  Year 5 Net Income:       %s\n", FormatCurrency(scenario.Year5NetIncome))
		fmt.Fprintf(&buf, "  Year 10 Net Income:      %s\n", FormatCurrency(scenario.Year10NetIncome))
		fmt.Fprintf(&buf, "  TSP Longevity:           %d years\n", scenario.TSPLongevity)
		fmt.Fprintf(&buf, "  Total Lifetime Income:   %s\n", FormatCurrency(scenario.TotalLifetimeIncome))
		fmt.Fprintln(&buf)
		fmt.Fprintln(&buf)
	}

	// Recommendation section using existing AnalyzeScenarios logic
	rec := AnalyzeScenarios(results)
	if rec.ScenarioName != "" {
		fmt.Fprintln(&buf, "SUMMARY & RECOMMENDATIONS")
		fmt.Fprintln(&buf, "=========================")
		fmt.Fprintf(&buf, "Best scenario for Robert: %s\n", rec.ScenarioName)
		fmt.Fprintf(&buf, "Take-Home Income Change: %s (%s)\n", FormatCurrency(rec.NetIncomeChange), FormatPercentage(rec.PercentageChange))
		fmt.Fprintf(&buf, "Monthly Change: %s\n", FormatCurrency(rec.NetIncomeChange.Div(decimal.NewFromInt(12))))
	}

	return buf.Bytes(), nil
}

// writeDetailedComparison migrates the original GenerateDetailedComparison output (condensed)
func writeDetailedComparison(buf *bytes.Buffer, results *domain.ScenarioComparison) {
	fmt.Fprintln(buf, "=================================================================================")
	fmt.Fprintln(buf, "DETAILED INCOME VALIDATION: WORKING vs RETIREMENT")
	fmt.Fprintln(buf, "=================================================================================")
	for i, scenario := range results.Scenarios {
		var firstRetirementYear *domain.AnnualCashFlow
		for _, y := range scenario.Projection {
			if y.IsRetired {
				firstRetirementYear = &y
				break
			}
		}
		if firstRetirementYear == nil {
			continue
		}
		// descriptive title
		var title string
		if strings.Contains(scenario.Name, "Dec 2025") {
			title = fmt.Sprintf("SCENARIO %d: Dawn Aug 2025, Robert Dec 2025", i+1)
		} else {
			title = fmt.Sprintf("SCENARIO %d: Dawn Aug 2025, Robert Feb 2027", i+1)
		}
		fmt.Fprintf(buf, "\n%s\n", title)
		fmt.Fprintln(buf, strings.Repeat("=", len(title)))
		fmt.Fprintln(buf)
		fmt.Fprintf(buf, "%-35s %15s %15s %15s\n", "COMPONENT", "WORKING", "RETIREMENT", "DIFFERENCE")
		fmt.Fprintln(buf, strings.Repeat("-", 80))
		workingGross := decimal.NewFromFloat(367399.00)
		workingNet := results.BaselineNetIncome
		fmt.Fprintln(buf, "INCOME SOURCES:")
		cmpLine(buf, "  Salary (Robert + Dawn)", workingGross, firstRetirementYear.SalaryRobert.Add(firstRetirementYear.SalaryDawn))
		cmpLine(buf, "  FERS Pension", decimal.Zero, firstRetirementYear.PensionRobert.Add(firstRetirementYear.PensionDawn))
		cmpLine(buf, "  TSP Withdrawals", decimal.Zero, firstRetirementYear.TSPWithdrawalRobert.Add(firstRetirementYear.TSPWithdrawalDawn))
		cmpLine(buf, "  Social Security", decimal.Zero, firstRetirementYear.SSBenefitRobert.Add(firstRetirementYear.SSBenefitDawn))
		cmpLine(buf, "  FERS Supplement", decimal.Zero, firstRetirementYear.FERSSupplementRobert.Add(firstRetirementYear.FERSSupplementDawn))
		fmt.Fprintln(buf, strings.Repeat("-", 80))
		cmpLine(buf, "TOTAL GROSS INCOME", workingGross, firstRetirementYear.TotalGrossIncome)
		fmt.Fprintln(buf)
		fmt.Fprintln(buf, "DEDUCTIONS & TAXES:")
		workingFederal := decimal.NewFromFloat(67060.18)
		workingState := decimal.NewFromFloat(11279.15)
		workingLocal := decimal.NewFromFloat(3673.99)
		workingFICA := decimal.NewFromFloat(16837.08)
		workingTSP := decimal.NewFromFloat(69812.52)
		workingFEHB := decimal.NewFromFloat(12700.74)
		cmpLine(buf, "  Federal Tax", workingFederal, firstRetirementYear.FederalTax)
		cmpLine(buf, "  State Tax", workingState, firstRetirementYear.StateTax)
		cmpLine(buf, "  Local Tax", workingLocal, firstRetirementYear.LocalTax)
		cmpLine(buf, "  FICA Tax", workingFICA, firstRetirementYear.FICATax)
		cmpLine(buf, "  TSP Contributions", workingTSP, firstRetirementYear.TSPContributions)
		cmpLine(buf, "  FEHB Premium", workingFEHB, firstRetirementYear.FEHBPremium)
		cmpLine(buf, "  Medicare Premium", decimal.Zero, firstRetirementYear.MedicarePremium)
		fmt.Fprintln(buf, strings.Repeat("-", 80))
		workingTotalDeductions := workingFederal.Add(workingState).Add(workingLocal).Add(workingFICA).Add(workingTSP).Add(workingFEHB)
		retirementTotalDeductions := firstRetirementYear.FederalTax.Add(firstRetirementYear.StateTax).Add(firstRetirementYear.LocalTax).Add(firstRetirementYear.FICATax).Add(firstRetirementYear.TSPContributions).Add(firstRetirementYear.FEHBPremium).Add(firstRetirementYear.MedicarePremium)
		cmpLine(buf, "TOTAL DEDUCTIONS", workingTotalDeductions, retirementTotalDeductions)
		fmt.Fprintln(buf)
		fmt.Fprintln(buf, strings.Repeat("=", 80))
		cmpLine(buf, "NET TAKE-HOME INCOME", workingNet, firstRetirementYear.NetIncome)
		netDiff := firstRetirementYear.NetIncome.Sub(workingNet)
		percentChange := netDiff.Div(workingNet).Mul(decimal.NewFromInt(100))
		fmt.Fprintln(buf)
		fmt.Fprintln(buf, "KEY INSIGHTS:")
		fmt.Fprintf(buf, "• Working income is reduced by $%.2f in TSP contributions\n", workingTSP.InexactFloat64())
		fmt.Fprintf(buf, "• Working income is reduced by $%.2f in FICA taxes\n", workingFICA.InexactFloat64())
		fmt.Fprintf(buf, "• Retirement adds $%.2f in pension income\n", firstRetirementYear.PensionRobert.Add(firstRetirementYear.PensionDawn).InexactFloat64())
		fmt.Fprintf(buf, "• Retirement adds $%.2f in TSP withdrawals\n", firstRetirementYear.TSPWithdrawalRobert.Add(firstRetirementYear.TSPWithdrawalDawn).InexactFloat64())
		fmt.Fprintf(buf, "• Retirement adds $%.2f in Social Security\n", firstRetirementYear.SSBenefitRobert.Add(firstRetirementYear.SSBenefitDawn).InexactFloat64())
		if firstRetirementYear.FERSSupplementRobert.Add(firstRetirementYear.FERSSupplementDawn).GreaterThan(decimal.Zero) {
			fmt.Fprintf(buf, "• Retirement adds $%.2f in FERS supplement\n", firstRetirementYear.FERSSupplementRobert.Add(firstRetirementYear.FERSSupplementDawn).InexactFloat64())
		}
		fmt.Fprintf(buf, "\nNet Effect: %s (%s)\n", FormatCurrency(netDiff), FormatPercentage(percentChange))
		fmt.Fprintln(buf)
	}
}

func cmpLine(buf *bytes.Buffer, label string, working, retirement decimal.Decimal) {
	diff := retirement.Sub(working)
	fmt.Fprintf(buf, "%-35s %15s %15s %15s\n", label, FormatCurrency(working), FormatCurrency(retirement), FormatCurrency(diff))
}
