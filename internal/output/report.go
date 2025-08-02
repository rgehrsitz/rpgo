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
	case "detailed-csv":
		return generator.GenerateDetailedCSVReport(results)
	case "all":
		// Generate console report with detailed validation AND export CSV
		if err := generator.GenerateConsoleReport(results); err != nil {
			return err
		}
		return generator.GenerateDetailedCSVReport(results)
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
	
	// Document key assumptions
	fmt.Println("KEY ASSUMPTIONS:")
	fmt.Println("• General COLA (FERS pension & SS): 2.5% annually")
	fmt.Println("• FEHB premium inflation: 4.0% annually") 
	fmt.Println("• TSP growth pre-retirement: 7.0% annually")
	fmt.Println("• TSP growth post-retirement: 5.0% annually")
	fmt.Println("• Social Security wage base indexing: ~5% annually (2025 est: $168,600)")
	fmt.Println("• Tax brackets: 2025 levels held constant (no inflation indexing)")
	fmt.Println()

	// Current Net Income Breakdown
	fmt.Println("CURRENT NET INCOME BREAKDOWN (Pre-Retirement)")
	fmt.Println("=============================================")
	fmt.Printf("Combined Gross Salary: %s\n", FormatCurrency(decimal.NewFromFloat(367399.00))) // From config data
	fmt.Printf("Combined Net Income:  %s\n", FormatCurrency(results.BaselineNetIncome))
	fmt.Printf("Monthly Net Income:   %s\n", FormatCurrency(results.BaselineNetIncome.Div(decimal.NewFromInt(12))))
	fmt.Println()

	// Add detailed side-by-side comparison for validation
	rg.GenerateDetailedComparison(results)

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
			fmt.Printf("FIRST RETIREMENT YEAR (%d) INCOME BREAKDOWN:\n", actualYear)
			fmt.Println("(Note: Amounts shown are current-year cash received - may be partial year)")
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

	fmt.Printf("RECOMMENDATION (Dawn retires Aug 30, 2025 - FIXED):\n")
	fmt.Printf("Best scenario for Robert: %s\n", bestScenario.Name)
	fmt.Printf("Take-Home Income Change: %s (%s)\n", FormatCurrency(netIncomeChange), FormatPercentage(percentageChange))
	fmt.Printf("Monthly Change: %s\n", FormatCurrency(monthlyChange))
	fmt.Println()
	fmt.Println("KEY CONSIDERATIONS:")
	fmt.Println("• Dawn's retirement date (Aug 30, 2025) is already submitted and fixed")
	if strings.Contains(bestScenario.Name, "Dec 2025") {
		fmt.Println("• Robert should retire in December 2025 (4 months after Dawn)")
		fmt.Println("• This provides immediate joint retirement benefits")
		fmt.Println("• Both can start Social Security at 62")
	} else {
		fmt.Println("• Robert should wait until February 2027 to retire")
		fmt.Println("• This allows Robert to qualify for enhanced pension multiplier at age 62")
		fmt.Println("• Dawn will be in partial retirement mode for 18 months")
	}
	fmt.Println("• Evaluate healthcare costs and Medicare coordination")
	fmt.Println("• Consider the impact of one vs. both incomes during transition period")

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
			fmt.Println("(Current-year cash received)")
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

// GenerateDetailedComparison creates a side-by-side comparison of working vs retirement income
func (rg *ReportGenerator) GenerateDetailedComparison(results *domain.ScenarioComparison) {
	fmt.Println("=================================================================================")
	fmt.Println("DETAILED INCOME VALIDATION: WORKING vs RETIREMENT")
	fmt.Println("=================================================================================")
	
	// Find the first full retirement year for each scenario
	for i, scenario := range results.Scenarios {
		var firstRetirementYear *domain.AnnualCashFlow
		for _, yearData := range scenario.Projection {
			if yearData.IsRetired {
				firstRetirementYear = &yearData
				break
			}
		}
		
		if firstRetirementYear == nil {
			continue
		}
		
		// Create more descriptive scenario title
		var scenarioTitle string
		if strings.Contains(scenario.Name, "Dec 2025") {
			scenarioTitle = fmt.Sprintf("SCENARIO %d: Dawn Aug 2025, Robert Dec 2025", i+1)
		} else {
			scenarioTitle = fmt.Sprintf("SCENARIO %d: Dawn Aug 2025, Robert Feb 2027", i+1)
		}
		
		fmt.Printf("\n%s\n", scenarioTitle)
		fmt.Println(strings.Repeat("=", len(scenarioTitle)))
		fmt.Println()
		
		// Create side-by-side comparison table
		fmt.Printf("%-35s %15s %15s %15s\n", "COMPONENT", "WORKING", "RETIREMENT", "DIFFERENCE")
		fmt.Println(strings.Repeat("-", 80))
		
		// Calculate working values (these are estimates based on known baseline)
		workingGross := decimal.NewFromFloat(367399.00) // Robert + Dawn salary
		workingNetIncome := results.BaselineNetIncome
		
		// Income Sources
		fmt.Println("INCOME SOURCES:")
		
		// Salary
		rg.printComparisonLine("  Salary (Robert + Dawn)", 
			workingGross, 
			firstRetirementYear.SalaryRobert.Add(firstRetirementYear.SalaryDawn),
		)
		
		// FERS Pension
		rg.printComparisonLine("  FERS Pension", 
			decimal.Zero, 
			firstRetirementYear.PensionRobert.Add(firstRetirementYear.PensionDawn),
		)
		
		// TSP Withdrawals
		rg.printComparisonLine("  TSP Withdrawals", 
			decimal.Zero, 
			firstRetirementYear.TSPWithdrawalRobert.Add(firstRetirementYear.TSPWithdrawalDawn),
		)
		
		// Social Security
		rg.printComparisonLine("  Social Security", 
			decimal.Zero, 
			firstRetirementYear.SSBenefitRobert.Add(firstRetirementYear.SSBenefitDawn),
		)
		
		// FERS Supplement
		rg.printComparisonLine("  FERS Supplement", 
			decimal.Zero, 
			firstRetirementYear.FERSSupplementRobert.Add(firstRetirementYear.FERSSupplementDawn),
		)
		
		fmt.Println(strings.Repeat("-", 80))
		
		// Total Gross
		rg.printComparisonLine("TOTAL GROSS INCOME", 
			workingGross, 
			firstRetirementYear.TotalGrossIncome,
		)
		
		fmt.Println()
		fmt.Println("DEDUCTIONS & TAXES:")
		
		// Calculate working deductions (estimates)
		workingFederal := decimal.NewFromFloat(67060.18)
		workingState := decimal.NewFromFloat(11279.15)
		workingLocal := decimal.NewFromFloat(3673.99)
		workingFICA := decimal.NewFromFloat(16837.08)
		workingTSP := decimal.NewFromFloat(69812.52)
		workingFEHB := decimal.NewFromFloat(12700.74)
		
		rg.printComparisonLine("  Federal Tax", 
			workingFederal, 
			firstRetirementYear.FederalTax,
		)
		
		rg.printComparisonLine("  State Tax", 
			workingState, 
			firstRetirementYear.StateTax,
		)
		
		rg.printComparisonLine("  Local Tax", 
			workingLocal, 
			firstRetirementYear.LocalTax,
		)
		
		rg.printComparisonLine("  FICA Tax", 
			workingFICA, 
			firstRetirementYear.FICATax,
		)
		
		rg.printComparisonLine("  TSP Contributions", 
			workingTSP, 
			firstRetirementYear.TSPContributions,
		)
		
		rg.printComparisonLine("  FEHB Premium", 
			workingFEHB, 
			firstRetirementYear.FEHBPremium,
		)
		
		rg.printComparisonLine("  Medicare Premium", 
			decimal.Zero, 
			firstRetirementYear.MedicarePremium,
		)
		
		fmt.Println(strings.Repeat("-", 80))
		
		// Total Deductions
		workingTotalDeductions := workingFederal.Add(workingState).Add(workingLocal).Add(workingFICA).Add(workingTSP).Add(workingFEHB)
		retirementTotalDeductions := firstRetirementYear.FederalTax.Add(firstRetirementYear.StateTax).Add(firstRetirementYear.LocalTax).Add(firstRetirementYear.FICATax).Add(firstRetirementYear.TSPContributions).Add(firstRetirementYear.FEHBPremium).Add(firstRetirementYear.MedicarePremium)
		
		rg.printComparisonLine("TOTAL DEDUCTIONS", 
			workingTotalDeductions, 
			retirementTotalDeductions,
		)
		
		fmt.Println()
		fmt.Println(strings.Repeat("=", 80))
		
		// Net Income
		rg.printComparisonLine("NET TAKE-HOME INCOME", 
			workingNetIncome, 
			firstRetirementYear.NetIncome,
		)
		
		// Show the difference
		netDifference := firstRetirementYear.NetIncome.Sub(workingNetIncome)
		percentChange := netDifference.Div(workingNetIncome).Mul(decimal.NewFromInt(100))
		
		fmt.Println()
		fmt.Printf("KEY INSIGHTS:\n")
		fmt.Printf("• Working income is reduced by $%.2f in TSP contributions\n", workingTSP.InexactFloat64())
		fmt.Printf("• Working income is reduced by $%.2f in FICA taxes\n", workingFICA.InexactFloat64())
		fmt.Printf("• Retirement adds $%.2f in pension income\n", firstRetirementYear.PensionRobert.Add(firstRetirementYear.PensionDawn).InexactFloat64())
		fmt.Printf("• Retirement adds $%.2f in TSP withdrawals\n", firstRetirementYear.TSPWithdrawalRobert.Add(firstRetirementYear.TSPWithdrawalDawn).InexactFloat64())
		fmt.Printf("• Retirement adds $%.2f in Social Security\n", firstRetirementYear.SSBenefitRobert.Add(firstRetirementYear.SSBenefitDawn).InexactFloat64())
		if firstRetirementYear.FERSSupplementRobert.Add(firstRetirementYear.FERSSupplementDawn).GreaterThan(decimal.Zero) {
			fmt.Printf("• Retirement adds $%.2f in FERS supplement\n", firstRetirementYear.FERSSupplementRobert.Add(firstRetirementYear.FERSSupplementDawn).InexactFloat64())
		}
		
		fmt.Printf("\nNet Effect: %s (%s)\n", FormatCurrency(netDifference), FormatPercentage(percentChange))
		fmt.Println()
	}
}

// printComparisonLine prints a formatted comparison line
func (rg *ReportGenerator) printComparisonLine(label string, working, retirement decimal.Decimal) {
	difference := retirement.Sub(working)
	fmt.Printf("%-35s %15s %15s %15s\n", 
		label, 
		FormatCurrency(working), 
		FormatCurrency(retirement), 
		FormatCurrency(difference),
	)
}

// GenerateDetailedCSVReport generates a comprehensive CSV with all financial data
func (rg *ReportGenerator) GenerateDetailedCSVReport(results *domain.ScenarioComparison) error {
	filename := fmt.Sprintf("retirement_detailed_analysis_%s.csv", time.Now().Format("20060102_150405"))
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"Scenario", "Year", "Robert_Age", "Dawn_Age", "Retirement_Status",
		"Salary_Robert", "Salary_Dawn", "Pension_Robert", "Pension_Dawn",
		"TSP_Withdrawal_Robert", "TSP_Withdrawal_Dawn", "SS_Benefit_Robert", "SS_Benefit_Dawn",
		"FERS_Supplement_Robert", "FERS_Supplement_Dawn", "Total_Gross_Income",
		"Federal_Tax", "State_Tax", "Local_Tax", "FICA_Tax", "TSP_Contributions",
		"FEHB_Premium", "Medicare_Premium", "Total_Deductions", "Net_Income",
		"TSP_Balance_Robert", "TSP_Balance_Dawn", "Total_TSP_Balance",
		"Net_Income_Change_vs_Current", "Percent_Change_vs_Current",
	}
	writer.Write(header)

	// Write current working income as baseline
	currentRow := []string{
		"CURRENT_WORKING", "2024", "59", "61", "WORKING",
		"190779.00", "176620.00", "0.00", "0.00",
		"0.00", "0.00", "0.00", "0.00",
		"0.00", "0.00", "367399.00",
		"67060.18", "11279.15", "3673.99", "16837.08", "69812.52",
		"12700.74", "0.00", "181363.66", results.BaselineNetIncome.StringFixed(2),
		"1966168.86", "1525175.90", "3491344.76",
		"0.00", "0.00",
	}
	writer.Write(currentRow)

	// Write scenario data
	for _, scenario := range results.Scenarios {
		for j, yearData := range scenario.Projection {
			if j > 10 { // Limit to first 10 years to keep CSV manageable
				break
			}
			
			actualYear := 2025 + j
			retirementStatus := "WORKING"
			if yearData.IsRetired {
				retirementStatus = "RETIRED"
			} else if yearData.SalaryRobert.LessThan(decimal.NewFromFloat(190779)) || yearData.SalaryDawn.LessThan(decimal.NewFromFloat(176620)) {
				retirementStatus = "PARTIAL_RETIREMENT"
			}

			// Calculate changes vs current
			netIncomeChange := yearData.NetIncome.Sub(results.BaselineNetIncome)
			percentChange := netIncomeChange.Div(results.BaselineNetIncome).Mul(decimal.NewFromInt(100))
			totalDeductions := yearData.FederalTax.Add(yearData.StateTax).Add(yearData.LocalTax).
				Add(yearData.FICATax).Add(yearData.TSPContributions).Add(yearData.FEHBPremium).Add(yearData.MedicarePremium)

			row := []string{
				scenario.Name, strconv.Itoa(actualYear),
				strconv.Itoa(yearData.AgeRobert), strconv.Itoa(yearData.AgeDawn), retirementStatus,
				yearData.SalaryRobert.StringFixed(2), yearData.SalaryDawn.StringFixed(2),
				yearData.PensionRobert.StringFixed(2), yearData.PensionDawn.StringFixed(2),
				yearData.TSPWithdrawalRobert.StringFixed(2), yearData.TSPWithdrawalDawn.StringFixed(2),
				yearData.SSBenefitRobert.StringFixed(2), yearData.SSBenefitDawn.StringFixed(2),
				yearData.FERSSupplementRobert.StringFixed(2), yearData.FERSSupplementDawn.StringFixed(2),
				yearData.TotalGrossIncome.StringFixed(2),
				yearData.FederalTax.StringFixed(2), yearData.StateTax.StringFixed(2),
				yearData.LocalTax.StringFixed(2), yearData.FICATax.StringFixed(2),
				yearData.TSPContributions.StringFixed(2), yearData.FEHBPremium.StringFixed(2),
				yearData.MedicarePremium.StringFixed(2), totalDeductions.StringFixed(2),
				yearData.NetIncome.StringFixed(2),
				yearData.TSPBalanceRobert.StringFixed(2), yearData.TSPBalanceDawn.StringFixed(2),
				yearData.TotalTSPBalance().StringFixed(2),
				netIncomeChange.StringFixed(2), percentChange.StringFixed(2),
			}
			writer.Write(row)
		}
	}

	fmt.Printf("\nDetailed CSV analysis exported to: %s\n", filename)
	return nil
}
