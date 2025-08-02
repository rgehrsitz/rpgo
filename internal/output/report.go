package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
	"gopkg.in/yaml.v3"
)

// ReportGenerator handles report generation in various formats
type ReportGenerator struct{}

// NewReportGenerator creates a new report generator
func NewReportGenerator() *ReportGenerator {
	return &ReportGenerator{}
}

// GenerateReport generates a report in the specified format
func GenerateReport(results *domain.ScenarioComparison, format string) error {
	generator := NewReportGenerator()

	switch format {
	case "console":
		return generator.GenerateConsoleReport(results)
	case "html":
		return generator.GenerateHTMLReport(results)
	case "json":
		return generator.GenerateJSONReport(results)
	case "csv":
		return generator.GenerateCSVReport(results)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// GenerateConsoleReport generates a detailed console-formatted report
func (rg *ReportGenerator) GenerateConsoleReport(results *domain.ScenarioComparison) error {
	fmt.Println("=================================================================================")
	fmt.Println("DETAILED FERS RETIREMENT INCOME ANALYSIS")
	fmt.Println("=================================================================================")
	fmt.Println()

	// Current Net Income Breakdown
	fmt.Println("CURRENT NET INCOME BREAKDOWN (Pre-Retirement)")
	fmt.Println("=============================================")
	fmt.Printf("Combined Gross Salary: %s\n", FormatCurrency(decimal.NewFromFloat(367399.00))) // From config data
	fmt.Printf("Combined Net Income:  %s\n", FormatCurrency(results.BaselineNetIncome))
	fmt.Printf("Monthly Net Income:   %s\n", FormatCurrency(results.BaselineNetIncome.Div(decimal.NewFromInt(12))))
	fmt.Println()

	// Detailed Scenario Analysis
	for i, scenario := range results.Scenarios {
		fmt.Printf("SCENARIO %d: %s\n", i+1, scenario.Name)
		fmt.Println(strings.Repeat("=", 50))

		// Find first full year when both are retired
		var firstRetirementYear domain.AnnualCashFlow
		var firstRetirementYearIndex int
		foundRetirementYear := false

		for yearIndex, yearData := range scenario.Projection {
			if yearData.IsRetired {
				firstRetirementYear = yearData
				firstRetirementYearIndex = yearIndex
				foundRetirementYear = true
				break
			}
		}

		if foundRetirementYear {
			// Calculate the actual year (2025 + yearIndex)
			actualYear := 2025 + firstRetirementYearIndex
			fmt.Printf("FIRST FULL RETIREMENT YEAR (%d) INCOME BREAKDOWN:\n", actualYear)
			fmt.Println("----------------------------------------")

			// Income Sources
			fmt.Println("INCOME SOURCES:")
			fmt.Printf("  Robert's Salary:        %s\n", FormatCurrency(firstRetirementYear.SalaryRobert))
			fmt.Printf("  Dawn's Salary:          %s\n", FormatCurrency(firstRetirementYear.SalaryDawn))
			fmt.Printf("  Robert's FERS Pension:  %s\n", FormatCurrency(firstRetirementYear.PensionRobert))
			fmt.Printf("  Dawn's FERS Pension:    %s\n", FormatCurrency(firstRetirementYear.PensionDawn))
			fmt.Printf("  Robert's TSP Withdrawal: %s\n", FormatCurrency(firstRetirementYear.TSPWithdrawalRobert))
			fmt.Printf("  Dawn's TSP Withdrawal:   %s\n", FormatCurrency(firstRetirementYear.TSPWithdrawalDawn))
			fmt.Printf("  Robert's Social Security: %s\n", FormatCurrency(firstRetirementYear.SSBenefitRobert))
			fmt.Printf("  Dawn's Social Security:   %s\n", FormatCurrency(firstRetirementYear.SSBenefitDawn))
			fmt.Printf("  Robert's FERS SRS:       %s\n", FormatCurrency(firstRetirementYear.FERSSupplementRobert))
			fmt.Printf("  Dawn's FERS SRS:         %s\n", FormatCurrency(firstRetirementYear.FERSSupplementDawn))
			fmt.Printf("  TOTAL GROSS INCOME:      %s\n", FormatCurrency(firstRetirementYear.TotalGrossIncome))
			fmt.Println()

			// Deductions and Taxes
			fmt.Println("DEDUCTIONS & TAXES:")
			fmt.Printf("  Federal Tax:            %s\n", FormatCurrency(firstRetirementYear.FederalTax))
			fmt.Printf("  State Tax:              %s\n", FormatCurrency(firstRetirementYear.StateTax))
			fmt.Printf("  Local Tax:              %s\n", FormatCurrency(firstRetirementYear.LocalTax))
			fmt.Printf("  FICA Tax:               %s\n", FormatCurrency(firstRetirementYear.FICATax))
			fmt.Printf("  TSP Contributions:      %s\n", FormatCurrency(firstRetirementYear.TSPContributions))
			fmt.Printf("  FEHB Premium:           %s\n", FormatCurrency(firstRetirementYear.FEHBPremium))
			fmt.Printf("  Medicare Premium:       %s\n", FormatCurrency(firstRetirementYear.MedicarePremium))
			fmt.Printf("  TOTAL DEDUCTIONS:       %s\n", FormatCurrency(firstRetirementYear.CalculateTotalDeductions()))
			fmt.Println()

			// Net Income Comparison
			fmt.Println("NET INCOME COMPARISON:")
			fmt.Println("----------------------")
			fmt.Printf("  Current Net Income:     %s\n", FormatCurrency(results.BaselineNetIncome))
			fmt.Printf("  Retirement Net Income:  %s\n", FormatCurrency(firstRetirementYear.NetIncome))

			change := firstRetirementYear.NetIncome.Sub(results.BaselineNetIncome)
			percentageChange := change.Div(results.BaselineNetIncome).Mul(decimal.NewFromInt(100))

			if change.GreaterThan(decimal.Zero) {
				fmt.Printf("  CHANGE: +%s (+%s)\n", FormatCurrency(change), FormatPercentage(percentageChange))
			} else {
				fmt.Printf("  CHANGE: %s (%s)\n", FormatCurrency(change), FormatPercentage(percentageChange))
			}

			monthlyChange := change.Div(decimal.NewFromInt(12))
			if monthlyChange.GreaterThan(decimal.Zero) {
				fmt.Printf("  Monthly Change: +%s\n", FormatCurrency(monthlyChange))
			} else {
				fmt.Printf("  Monthly Change: %s\n", FormatCurrency(monthlyChange))
			}


			// Retirement Status
			fmt.Println("RETIREMENT STATUS:")
			fmt.Printf("  Is Retired:             %t\n", firstRetirementYear.IsRetired)
			fmt.Printf("  Medicare Eligible:      %t\n", firstRetirementYear.IsMedicareEligible)
			fmt.Printf("  RMD Year:               %t\n", firstRetirementYear.IsRMDYear)
			fmt.Printf("  Robert's Age:           %d\n", firstRetirementYear.AgeRobert)
			fmt.Printf("  Dawn's Age:             %d\n", firstRetirementYear.AgeDawn)
			fmt.Println()

			// Show first FULL retirement year (no working income)
			var firstFullRetirementYear domain.AnnualCashFlow
			var firstFullRetirementYearIndex int
			foundFullRetirementYear := false

			for yearIndex, yearData := range scenario.Projection {
				if yearData.IsRetired && yearData.SalaryRobert.Equal(decimal.Zero) && yearData.SalaryDawn.Equal(decimal.Zero) {
					firstFullRetirementYear = yearData
					firstFullRetirementYearIndex = yearIndex
					foundFullRetirementYear = true
					break
				}
			}

			if foundFullRetirementYear {
				fullRetirementYear := 2025 + firstFullRetirementYearIndex
				fmt.Printf("FIRST FULL RETIREMENT YEAR (NO WORKING INCOME) (%d):\n", fullRetirementYear)
				fmt.Println("--------------------------------------------------------")
				fmt.Printf("  Total Gross Income:      %s\n", FormatCurrency(firstFullRetirementYear.TotalGrossIncome))
				fmt.Printf("  Net Income:              %s\n", FormatCurrency(firstFullRetirementYear.NetIncome))

				fullRetirementChange := firstFullRetirementYear.NetIncome.Sub(results.BaselineNetIncome)
				fullRetirementPercentageChange := fullRetirementChange.Div(results.BaselineNetIncome).Mul(decimal.NewFromInt(100))

				if fullRetirementChange.GreaterThan(decimal.Zero) {
					fmt.Printf("  CHANGE vs Current:       +%s (+%s)\n", FormatCurrency(fullRetirementChange), FormatPercentage(fullRetirementPercentageChange))
				} else {
					fmt.Printf("  CHANGE vs Current:       %s (%s)\n", FormatCurrency(fullRetirementChange), FormatPercentage(fullRetirementPercentageChange))
				}
				fmt.Println()
			}
		} else {
			fmt.Println("No retirement year found in projection")
			fmt.Println()
		}

		// Long-term Projection
		fmt.Println("LONG-TERM PROJECTION:")
		fmt.Println("---------------------")
		fmt.Printf("  Year 5 Net Income:       %s\n", FormatCurrency(scenario.Year5NetIncome))
		fmt.Printf("  Year 10 Net Income:      %s\n", FormatCurrency(scenario.Year10NetIncome))
		fmt.Printf("  TSP Longevity:           %d years\n", scenario.TSPLongevity)
		fmt.Printf("  Total Lifetime Income:   %s\n", FormatCurrency(scenario.TotalLifetimeIncome))
		fmt.Println()
		fmt.Println()
	}

	// Summary
	fmt.Println("SUMMARY & RECOMMENDATIONS")
	fmt.Println("=========================")

	// Find best scenario based on net income comparison (true take-home to take-home)
	var bestScenario domain.ScenarioSummary
	var bestRetirementIncome decimal.Decimal

	// Use current take-home income as baseline (no TSP additions - this is actual spendable money)
	currentTakeHome := results.BaselineNetIncome

	// Debug output to see what's happening
	fmt.Println("DEBUG: Recommendation Logic:")
	fmt.Printf("  Current Take-Home Income: %s\n", FormatCurrency(currentTakeHome))

	for i, scenario := range results.Scenarios {
		// Find the first full retirement year for this scenario
		var firstRetirementIncome decimal.Decimal
		for _, yearData := range scenario.Projection {
			if yearData.IsRetired {
				firstRetirementIncome = yearData.NetIncome
				break
			}
		}

		fmt.Printf("  Scenario %d (%s): %s (first retirement year)\n", i+1, scenario.Name, FormatCurrency(firstRetirementIncome))

		if firstRetirementIncome.GreaterThan(bestRetirementIncome) {
			bestRetirementIncome = firstRetirementIncome
			bestScenario = scenario
			fmt.Printf("    -> NEW BEST (income: %s)\n", FormatCurrency(bestRetirementIncome))
		}
	}

	// Calculate net income change using the first retirement year for the best scenario
	var bestScenarioFirstRetirementIncome decimal.Decimal
	for _, yearData := range bestScenario.Projection {
		if yearData.IsRetired {
			bestScenarioFirstRetirementIncome = yearData.NetIncome
			break
		}
	}
	netIncomeChange := bestScenarioFirstRetirementIncome.Sub(currentTakeHome)
	percentageChange := netIncomeChange.Div(currentTakeHome).Mul(decimal.NewFromInt(100))
	monthlyChange := netIncomeChange.Div(decimal.NewFromInt(12))

	fmt.Printf("Recommended Scenario: %s\n", bestScenario.Name)
	fmt.Printf("Take-Home Income Change: %s (%s)\n", FormatCurrency(netIncomeChange), FormatPercentage(percentageChange))
	fmt.Printf("Monthly Change: %s\n", FormatCurrency(monthlyChange))
	fmt.Println("• Evaluate healthcare costs and Medicare coordination")

	// Add detailed year-by-year breakdown
	fmt.Println()
	fmt.Println("=================================================================================")
	fmt.Println("DETAILED YEAR-BY-YEAR BREAKDOWN (2025-2030)")
	fmt.Println("=================================================================================")

	for i, scenario := range results.Scenarios {
		fmt.Printf("\nSCENARIO %d: %s\n", i+1, scenario.Name)
		fmt.Println(strings.Repeat("=", 80))

		// Show years 2025-2030 (indices 0-5)
		for yearIndex := 0; yearIndex <= 5 && yearIndex < len(scenario.Projection); yearIndex++ {
			yearData := scenario.Projection[yearIndex]
			actualYear := 2025 + yearIndex

			fmt.Printf("\nYEAR %d (%s):\n", actualYear, yearData.Date.Format("2006"))
			fmt.Println(strings.Repeat("-", 40))

			// Income Sources
			fmt.Println("INCOME SOURCES:")
			fmt.Printf("  Robert's Salary:        %s\n", FormatCurrency(yearData.SalaryRobert))
			fmt.Printf("  Dawn's Salary:          %s\n", FormatCurrency(yearData.SalaryDawn))
			fmt.Printf("  Robert's FERS Pension:  %s\n", FormatCurrency(yearData.PensionRobert))
			fmt.Printf("  Dawn's FERS Pension:    %s\n", FormatCurrency(yearData.PensionDawn))
			fmt.Printf("  Robert's TSP Withdrawal: %s\n", FormatCurrency(yearData.TSPWithdrawalRobert))
			fmt.Printf("  Dawn's TSP Withdrawal:   %s\n", FormatCurrency(yearData.TSPWithdrawalDawn))
			fmt.Printf("  Robert's Social Security: %s\n", FormatCurrency(yearData.SSBenefitRobert))
			fmt.Printf("  Dawn's Social Security:   %s\n", FormatCurrency(yearData.SSBenefitDawn))
			fmt.Printf("  Robert's FERS SRS:       %s\n", FormatCurrency(yearData.FERSSupplementRobert))
			fmt.Printf("  Dawn's FERS SRS:         %s\n", FormatCurrency(yearData.FERSSupplementDawn))
			fmt.Printf("  TOTAL GROSS INCOME:      %s\n", FormatCurrency(yearData.TotalGrossIncome))

			// Deductions
			fmt.Println("\nDEDUCTIONS:")
			fmt.Printf("  Federal Tax:            %s\n", FormatCurrency(yearData.FederalTax))
			fmt.Printf("  State Tax:              %s\n", FormatCurrency(yearData.StateTax))
			fmt.Printf("  Local Tax:              %s\n", FormatCurrency(yearData.LocalTax))
			fmt.Printf("  FICA Tax:               %s\n", FormatCurrency(yearData.FICATax))
			fmt.Printf("  TSP Contributions:      %s\n", FormatCurrency(yearData.TSPContributions))
			fmt.Printf("  FEHB Premium:           %s\n", FormatCurrency(yearData.FEHBPremium))
			fmt.Printf("  Medicare Premium:       %s\n", FormatCurrency(yearData.MedicarePremium))
			fmt.Printf("  TOTAL DEDUCTIONS:       %s\n", FormatCurrency(yearData.CalculateTotalDeductions()))

			// Net Income
			fmt.Printf("\nNET INCOME:               %s\n", FormatCurrency(yearData.NetIncome))
			fmt.Printf("Monthly Net Income:       %s\n", FormatCurrency(yearData.NetIncome.Div(decimal.NewFromInt(12))))

			// Comparison to current
			change := yearData.NetIncome.Sub(results.BaselineNetIncome)
			percentageChange := change.Div(results.BaselineNetIncome).Mul(decimal.NewFromInt(100))
			if change.GreaterThan(decimal.Zero) {
				fmt.Printf("CHANGE vs Current:        +%s (+%s)\n", FormatCurrency(change), FormatPercentage(percentageChange))
			} else {
				fmt.Printf("CHANGE vs Current:        %s (%s)\n", FormatCurrency(change), FormatPercentage(percentageChange))
			}

			// Status
			fmt.Printf("Retirement Status:        %s\n", func() string {
				if yearData.IsRetired {
					return "FULLY RETIRED"
				} else if !yearData.SalaryRobert.Equal(decimal.Zero) || !yearData.SalaryDawn.Equal(decimal.Zero) {
					return "PARTIAL RETIREMENT"
				} else {
					return "WORKING"
				}
			}())
			fmt.Printf("Robert's Age:             %d\n", yearData.AgeRobert)
			fmt.Printf("Dawn's Age:               %d\n", yearData.AgeDawn)
		}
	}

	return nil
}

// GenerateHTMLReport generates an HTML-formatted report
func (rg *ReportGenerator) GenerateHTMLReport(results *domain.ScenarioComparison) error {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>FERS Retirement Planning Results</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; }
        .section { margin: 20px 0; }
        .scenario { border: 1px solid #ddd; padding: 15px; margin: 10px 0; border-radius: 5px; }
        .metric { display: inline-block; margin: 10px 20px 10px 0; }
        .metric-label { font-weight: bold; color: #666; }
        .metric-value { font-size: 1.2em; color: #333; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <div class="header">
        <h1>FERS Retirement Planning Calculator - Results</h1>
        <p>Generated on: ` + time.Now().Format("2006-01-02 15:04:05") + `</p>
    </div>
    
    <div class="section">
        <h2>Current Situation</h2>
        <div class="metric">
            <div class="metric-label">Current Net Income</div>
            <div class="metric-value">$` + results.BaselineNetIncome.StringFixed(2) + `</div>
        </div>
    </div>
    
    <div class="section">
        <h2>Scenario Comparison</h2>`

	for i, scenario := range results.Scenarios {
		html += `
        <div class="scenario">
            <h3>` + fmt.Sprintf("%d. %s", i+1, scenario.Name) + `</h3>
            <div class="metric">
                <div class="metric-label">First Year Net Income</div>
                <div class="metric-value">$` + scenario.FirstYearNetIncome.StringFixed(2) + `</div>
            </div>
            <div class="metric">
                <div class="metric-label">Year 5 Net Income</div>
                <div class="metric-value">$` + scenario.Year5NetIncome.StringFixed(2) + `</div>
            </div>
            <div class="metric">
                <div class="metric-label">TSP Longevity</div>
                <div class="metric-value">` + fmt.Sprintf("%d years", scenario.TSPLongevity) + `</div>
            </div>
            <div class="metric">
                <div class="metric-label">Total Lifetime Income (PV)</div>
                <div class="metric-value">$` + scenario.TotalLifetimeIncome.StringFixed(2) + `</div>
            </div>
        </div>`
	}

	html += `
    </div>
    
    <div class="section">
        <h2>Impact Analysis</h2>
        <div class="metric">
            <div class="metric-label">Recommended Scenario</div>
            <div class="metric-value">` + results.ImmediateImpact.RecommendedScenario + `</div>
        </div>
        <div class="metric">
            <div class="metric-label">Net Income Change</div>
            <div class="metric-value">$` + results.ImmediateImpact.CurrentToFirstYear.NetIncomeChange.StringFixed(2) + `</div>
        </div>
        <div class="metric">
            <div class="metric-label">Percentage Change</div>
            <div class="metric-value">` + results.ImmediateImpact.CurrentToFirstYear.PercentageChange.StringFixed(2) + `%</div>
        </div>
    </div>
    
    <div class="section">
        <h2>Key Considerations</h2>
        <ul>`

	for _, consideration := range results.ImmediateImpact.KeyConsiderations {
		html += `<li>` + consideration + `</li>`
	}

	html += `
        </ul>
    </div>
    
    <div class="section">
        <h2>Long-Term Analysis</h2>
        <div class="metric">
            <div class="metric-label">Best for Income</div>
            <div class="metric-value">` + results.LongTermProjection.BestScenarioForIncome + `</div>
        </div>
        <div class="metric">
            <div class="metric-label">Best for Longevity</div>
            <div class="metric-value">` + results.LongTermProjection.BestScenarioForLongevity + `</div>
        </div>
        <div class="metric">
            <div class="metric-label">Risk Assessment</div>
            <div class="metric-value">` + results.LongTermProjection.RiskAssessment + `</div>
        </div>
    </div>
    
    <div class="section">
        <h2>Recommendations</h2>
        <ul>`

	for _, recommendation := range results.LongTermProjection.Recommendations {
		html += `<li>` + recommendation + `</li>`
	}

	html += `
        </ul>
    </div>
</body>
</html>`

	// Write to file
	filename := fmt.Sprintf("retirement_report_%s.html", time.Now().Format("20060102_150405"))
	return os.WriteFile(filename, []byte(html), 0644)
}

// GenerateJSONReport generates a JSON-formatted report
func (rg *ReportGenerator) GenerateJSONReport(results *domain.ScenarioComparison) error {
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("retirement_report_%s.json", time.Now().Format("20060102_150405"))
	return os.WriteFile(filename, jsonData, 0644)
}

// GenerateCSVReport generates a CSV-formatted report
func (rg *ReportGenerator) GenerateCSVReport(results *domain.ScenarioComparison) error {
	filename := fmt.Sprintf("retirement_report_%s.csv", time.Now().Format("20060102_150405"))
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"Scenario", "First Year Net Income", "Year 5 Net Income", "Year 10 Net Income",
		"TSP Longevity", "Total Lifetime Income (PV)", "Initial TSP Balance", "Final TSP Balance",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write scenario data
	for _, scenario := range results.Scenarios {
		row := []string{
			scenario.Name,
			scenario.FirstYearNetIncome.StringFixed(2),
			scenario.Year5NetIncome.StringFixed(2),
			scenario.Year10NetIncome.StringFixed(2),
			strconv.Itoa(scenario.TSPLongevity),
			scenario.TotalLifetimeIncome.StringFixed(2),
			scenario.InitialTSPBalance.StringFixed(2),
			scenario.FinalTSPBalance.StringFixed(2),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// SaveConfiguration saves a configuration to a file
func SaveConfiguration(config *domain.Configuration, filename string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// FormatCurrency formats a decimal as currency
func FormatCurrency(amount decimal.Decimal) string {
	return "$" + amount.StringFixed(2)
}

// FormatPercentage formats a decimal as percentage
func FormatPercentage(amount decimal.Decimal) string {
	return amount.StringFixed(2) + "%"
}
