package output

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/rgehrsitz/rpgo/internal/domain"
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
	assumptions := results.Assumptions
	if len(assumptions) == 0 {
		assumptions = DefaultAssumptions
	}
	for _, a := range assumptions {
		fmt.Fprintf(&buf, "• %s\n", a)
	}
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "CURRENT NET INCOME BREAKDOWN (Pre-Retirement)")
	fmt.Fprintln(&buf, "=============================================")
	// Calculate dynamic gross salary from first scenario's working data
	var workingGross decimal.Decimal
	if len(results.Scenarios) > 0 && len(results.Scenarios[0].Projection) > 0 {
		// Find first working year (not retired)
		for _, year := range results.Scenarios[0].Projection {
			if !year.IsRetired && year.GetTotalSalary().GreaterThan(decimal.Zero) {
				workingGross = year.GetTotalSalary()
				break
			}
		}
	}
	if workingGross.IsZero() {
		workingGross = decimal.NewFromFloat(367399.00) // fallback to known value
	}
	fmt.Fprintf(&buf, "Combined Gross Salary: %s\n", FormatCurrency(workingGross))
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
			// Display income for each participant dynamically
			for participantName, salary := range firstRetirementYear.Salaries {
				if !salary.IsZero() {
					fmt.Fprintf(&buf, "  %s's Salary:        %s\n", participantName, FormatCurrency(salary))
				}
			}
			for participantName, pension := range firstRetirementYear.Pensions {
				if !pension.IsZero() {
					fmt.Fprintf(&buf, "  %s's FERS Pension:  %s\n", participantName, FormatCurrency(pension))
				}
			}
			for participantName, tspWithdrawal := range firstRetirementYear.TSPWithdrawals {
				if !tspWithdrawal.IsZero() {
					fmt.Fprintf(&buf, "  %s's TSP Withdrawal: %s\n", participantName, FormatCurrency(tspWithdrawal))
				}
			}
			for participantName, ssBenefit := range firstRetirementYear.SSBenefits {
				if !ssBenefit.IsZero() {
					fmt.Fprintf(&buf, "  %s's Social Security: %s\n", participantName, FormatCurrency(ssBenefit))
				}
			}
			for participantName, fersSupplement := range firstRetirementYear.FERSSupplements {
				if !fersSupplement.IsZero() {
					fmt.Fprintf(&buf, "  %s's FERS SRS:       %s\n", participantName, FormatCurrency(fersSupplement))
				}
			}
			fmt.Fprintf(&buf, "  TOTAL GROSS INCOME:      %s\n", FormatCurrency(firstRetirementYear.TotalGrossIncome))
			fmt.Fprintln(&buf)
			fmt.Fprintln(&buf, "DEDUCTIONS & TAXES:")
			fmt.Fprintf(&buf, "  Federal Tax:            %s\n", FormatCurrency(firstRetirementYear.FederalTax))
			fmt.Fprintf(&buf, "  State Tax:              %s\n", FormatCurrency(firstRetirementYear.StateTax))
			fmt.Fprintf(&buf, "  Local Tax:              %s\n", FormatCurrency(firstRetirementYear.LocalTax))
			fmt.Fprintf(&buf, "  FICA Tax:               %s\n", FormatCurrency(firstRetirementYear.FICATax))
			fmt.Fprintf(&buf, "  TSP Contributions:      %s\n", FormatCurrency(firstRetirementYear.TotalTSPContributions))
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
			// Display ages for each participant dynamically
			for participantName, age := range firstRetirementYear.Ages {
				fmt.Fprintf(&buf, "  %s's Age:           %d\n", participantName, age)
			}
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
		fmt.Fprintf(&buf, "Best scenario: %s\n", rec.ScenarioName)
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
		// Use scenario name as title
		title := fmt.Sprintf("SCENARIO %d: %s", i+1, scenario.Name)
		fmt.Fprintf(buf, "\n%s\n", title)
		fmt.Fprintln(buf, strings.Repeat("=", len(title)))
		fmt.Fprintln(buf)
		fmt.Fprintf(buf, "%-35s %15s %15s %15s\n", "COMPONENT", "WORKING", "RETIREMENT", "DIFFERENCE")
		fmt.Fprintln(buf, strings.Repeat("-", 80))
		// Calculate dynamic working gross from the scenario data
		var workingGross decimal.Decimal
		if len(results.Scenarios) > 0 && len(results.Scenarios[0].Projection) > 0 {
			for _, year := range results.Scenarios[0].Projection {
				if !year.IsRetired && year.GetTotalSalary().GreaterThan(decimal.Zero) {
					workingGross = year.GetTotalSalary()
					break
				}
			}
		}
		if workingGross.IsZero() {
			workingGross = decimal.NewFromFloat(367399.00) // fallback
		}
		workingNet := results.BaselineNetIncome
		fmt.Fprintln(buf, "INCOME SOURCES:")
		cmpLine(buf, "  Combined Salary", workingGross, firstRetirementYear.GetTotalSalary())
		cmpLine(buf, "  FERS Pension", decimal.Zero, firstRetirementYear.GetTotalPension())
		cmpLine(buf, "  TSP Withdrawals", decimal.Zero, firstRetirementYear.GetTotalTSPWithdrawal())
		cmpLine(buf, "  Social Security", decimal.Zero, firstRetirementYear.GetTotalSSBenefit())
		cmpLine(buf, "  FERS Supplement", decimal.Zero, firstRetirementYear.GetTotalFERSSupplement())
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
		cmpLine(buf, "  TSP Contributions", workingTSP, firstRetirementYear.TotalTSPContributions)
		cmpLine(buf, "  FEHB Premium", workingFEHB, firstRetirementYear.FEHBPremium)
		cmpLine(buf, "  Medicare Premium", decimal.Zero, firstRetirementYear.MedicarePremium)
		fmt.Fprintln(buf, strings.Repeat("-", 80))
		workingTotalDeductions := workingFederal.Add(workingState).Add(workingLocal).Add(workingFICA).Add(workingTSP).Add(workingFEHB)
		retirementTotalDeductions := firstRetirementYear.FederalTax.Add(firstRetirementYear.StateTax).Add(firstRetirementYear.LocalTax).Add(firstRetirementYear.FICATax).Add(firstRetirementYear.TotalTSPContributions).Add(firstRetirementYear.FEHBPremium).Add(firstRetirementYear.MedicarePremium)
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
		// Aggregate map-based values for insights
		pensionTotal := firstRetirementYear.GetTotalPension()
		withdrawalTotal := firstRetirementYear.GetTotalTSPWithdrawal()
		ssTotal := firstRetirementYear.GetTotalSSBenefit()
		fersSuppTotal := firstRetirementYear.GetTotalFERSSupplement()
		fmt.Fprintf(buf, "• Retirement adds $%.2f in pension income\n", pensionTotal.InexactFloat64())
		fmt.Fprintf(buf, "• Retirement adds $%.2f in TSP withdrawals\n", withdrawalTotal.InexactFloat64())
		fmt.Fprintf(buf, "• Retirement adds $%.2f in Social Security\n", ssTotal.InexactFloat64())
		if fersSuppTotal.GreaterThan(decimal.Zero) {
			fmt.Fprintf(buf, "• Retirement adds $%.2f in FERS supplement\n", fersSuppTotal.InexactFloat64())
		}
		fmt.Fprintf(buf, "\nNet Effect: %s (%s)\n", FormatCurrency(netDiff), FormatPercentage(percentChange))
		fmt.Fprintln(buf)
	}
}

func cmpLine(buf *bytes.Buffer, label string, working, retirement decimal.Decimal) {
	diff := retirement.Sub(working)
	fmt.Fprintf(buf, "%-35s %15s %15s %15s\n", label, FormatCurrency(working), FormatCurrency(retirement), FormatCurrency(diff))
}
