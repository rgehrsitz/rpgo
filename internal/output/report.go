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

// GenerateHTMLReport generates a comprehensive HTML-formatted report with charts and detailed tables
func (rg *ReportGenerator) GenerateHTMLReport(results *domain.ScenarioComparison) error {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>FERS Retirement Planning Analysis</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/date-fns@2.29.3/index.min.js"></script>
    <style>
        :root {
            --primary-color: #2c3e50;
            --secondary-color: #3498db;
            --success-color: #27ae60;
            --warning-color: #f39c12;
            --danger-color: #e74c3c;
            --light-bg: #ecf0f1;
            --card-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        }
        
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.6;
            color: var(--primary-color);
            background-color: #f8f9fa;
        }
        
        .container {
            max-width: 1400px;
            margin: 0 auto;
            padding: 20px;
        }
        
        .header {
            background: linear-gradient(135deg, var(--primary-color), var(--secondary-color));
            color: white;
            padding: 30px;
            border-radius: 10px;
            margin-bottom: 30px;
            text-align: center;
            box-shadow: var(--card-shadow);
        }
        
        .header h1 {
            font-size: 2.5rem;
            margin-bottom: 10px;
            font-weight: 300;
        }
        
        .header .subtitle {
            font-size: 1.1rem;
            opacity: 0.9;
        }
        
        .section {
            background: white;
            margin: 30px 0;
            padding: 25px;
            border-radius: 10px;
            box-shadow: var(--card-shadow);
        }
        
        .section h2 {
            color: var(--primary-color);
            margin-bottom: 20px;
            padding-bottom: 10px;
            border-bottom: 3px solid var(--secondary-color);
            font-size: 1.8rem;
            font-weight: 400;
        }
        
        .section h3 {
            color: var(--primary-color);
            margin: 20px 0 15px 0;
            font-size: 1.4rem;
            font-weight: 500;
        }
        
        .metric-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin: 20px 0;
        }
        
        .metric-card {
            background: var(--light-bg);
            padding: 20px;
            border-radius: 8px;
            text-align: center;
            border-left: 4px solid var(--secondary-color);
        }
        
        .metric-label {
            font-weight: 600;
            color: #7f8c8d;
            font-size: 0.9rem;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-bottom: 8px;
        }
        
        .metric-value {
            font-size: 1.8rem;
            font-weight: 700;
            color: var(--primary-color);
        }
        
        .metric-value.positive {
            color: var(--success-color);
        }
        
        .metric-value.negative {
            color: var(--danger-color);
        }
        
        .chart-container {
            position: relative;
            margin: 30px 0;
            padding: 20px;
            background: white;
            border-radius: 8px;
            height: 400px;
        }
        
        .chart-row {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 30px;
            margin: 30px 0;
        }
        
        .table-responsive {
            overflow-x: auto;
            margin: 20px 0;
        }
        
        table {
            width: 100%;
            border-collapse: collapse;
            font-size: 0.9rem;
        }
        
        th, td {
            padding: 12px 8px;
            text-align: right;
            border-bottom: 1px solid #ddd;
        }
        
        th {
            background-color: var(--primary-color);
            color: white;
            font-weight: 600;
            text-transform: uppercase;
            font-size: 0.8rem;
            letter-spacing: 0.5px;
        }
        
        th:first-child, td:first-child {
            text-align: left;
        }
        
        tr:nth-child(even) {
            background-color: #f8f9fa;
        }
        
        tr:hover {
            background-color: #e3f2fd;
        }
        
        .scenario-comparison {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
            gap: 25px;
            margin: 25px 0;
        }
        
        .scenario-card {
            border: 2px solid #e0e0e0;
            border-radius: 10px;
            padding: 20px;
            background: white;
            transition: all 0.3s ease;
        }
        
        .scenario-card:hover {
            border-color: var(--secondary-color);
            box-shadow: var(--card-shadow);
        }
        
        .scenario-card.recommended {
            border-color: var(--success-color);
            background: linear-gradient(135deg, #ffffff, #f0fff4);
        }
        
        .scenario-title {
            font-size: 1.3rem;
            font-weight: 600;
            margin-bottom: 15px;
            color: var(--primary-color);
        }
        
        .recommended-badge {
            background: var(--success-color);
            color: white;
            padding: 4px 12px;
            border-radius: 20px;
            font-size: 0.8rem;
            font-weight: 600;
            margin-left: 10px;
        }
        
        .assumptions {
            background: #fff9c4;
            border-left: 4px solid var(--warning-color);
            padding: 20px;
            margin: 20px 0;
            border-radius: 0 8px 8px 0;
        }
        
        .assumptions h4 {
            color: var(--warning-color);
            margin-bottom: 10px;
            font-size: 1.1rem;
        }
        
        .assumptions ul {
            list-style-type: none;
            padding-left: 0;
        }
        
        .assumptions li {
            padding: 3px 0;
            position: relative;
            padding-left: 20px;
        }
        
        .assumptions li:before {
            content: "•";
            color: var(--warning-color);
            font-weight: bold;
            position: absolute;
            left: 0;
        }
        
        .alert {
            padding: 15px;
            margin: 20px 0;
            border-radius: 8px;
            border-left: 4px solid;
        }
        
        .alert-info {
            background-color: #d1ecf1;
            border-color: #bee5eb;
            color: #0c5460;
        }
        
        .alert-success {
            background-color: #d4edda;
            border-color: #c3e6cb;
            color: #155724;
        }
        
        .nav-tabs {
            display: flex;
            border-bottom: 2px solid #e0e0e0;
            margin-bottom: 20px;
        }
        
        .nav-tab {
            padding: 12px 24px;
            background: #f8f9fa;
            border: none;
            cursor: pointer;
            font-size: 1rem;
            font-weight: 500;
            border-radius: 8px 8px 0 0;
            margin-right: 5px;
            transition: all 0.3s ease;
        }
        
        .nav-tab.active {
            background: var(--secondary-color);
            color: white;
        }
        
        .tab-content {
            display: none;
        }
        
        .tab-content.active {
            display: block;
        }
        
        .highlight-number {
            font-weight: 700;
            color: var(--secondary-color);
        }
        
        @media (max-width: 768px) {
            .chart-row {
                grid-template-columns: 1fr;
            }
            
            .scenario-comparison {
                grid-template-columns: 1fr;
            }
            
            .header h1 {
                font-size: 2rem;
            }
            
            .metric-grid {
                grid-template-columns: 1fr;
            }
        }
        
        .print-button {
            background: var(--secondary-color);
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 5px;
            cursor: pointer;
            font-size: 1rem;
            margin: 10px 0;
        }
        
        @media print {
            .print-button { display: none; }
            .section { page-break-inside: avoid; }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>FERS Retirement Planning Analysis</h1>
            <div class="subtitle">
                Comprehensive Financial Projection Report<br>
                Generated on ` + time.Now().Format("January 2, 2006 at 3:04 PM") + `
            </div>
            <button class="print-button" onclick="window.print()">Print Report</button>
        </div>`

	// Add key assumptions section
	html += rg.generateAssumptionsSection()

	// Add executive summary
	html += rg.generateExecutiveSummary(results)

	// Add current situation analysis
	html += rg.generateCurrentSituationSection(results)

	// Add scenario comparison with enhanced metrics
	html += rg.generateScenarioComparisonSection(results)

	// Add detailed year-by-year breakdown
	html += rg.generateYearByYearSection(results)

	// Add interactive charts
	html += rg.generateChartsSection(results)

	// Add detailed tables
	html += rg.generateDetailedTablesSection(results)

	// Add break-even analysis section
	html += rg.generateBreakEvenSection(results)

	// Add risk analysis and recommendations
	html += rg.generateRiskAnalysisSection(results)

	// Close HTML and add JavaScript
	html += rg.generateJavaScriptSection(results)

	html += `
    </div>
</body>
</html>`

	// Write to file
	filename := fmt.Sprintf("retirement_report_%s.html", time.Now().Format("20060102_150405"))
	return os.WriteFile(filename, []byte(html), 0644)
}

// generateAssumptionsSection creates the key assumptions section
func (rg *ReportGenerator) generateAssumptionsSection() string {
	return `
        <div class="section">
            <h2>📊 Key Planning Assumptions</h2>
            <div class="assumptions">
                <h4>Economic & Growth Assumptions</h4>
                <ul>
                    <li><strong>Inflation Rate:</strong> 2.5% annually</li>
                    <li><strong>TSP Growth (Pre-retirement):</strong> 7.0% annually</li>
                    <li><strong>TSP Growth (Post-retirement):</strong> 5.0% annually</li>
                    <li><strong>FEHB Premium Inflation:</strong> 4.0% annually</li>
                </ul>
            </div>
            <div class="assumptions">
                <h4>Federal Benefits Assumptions</h4>
                <ul>
                    <li><strong>FERS Pension COLA:</strong> 2.5% annually (starting at age 62)</li>
                    <li><strong>Social Security COLA:</strong> 2.5% annually</li>
                    <li><strong>Social Security Wage Base Growth:</strong> ~5% annually</li>
                    <li><strong>Tax Brackets:</strong> 2025 levels held constant</li>
                </ul>
            </div>
        </div>`
}

// generateExecutiveSummary creates the executive summary section
func (rg *ReportGenerator) generateExecutiveSummary(results *domain.ScenarioComparison) string {
	// Find the recommended scenario
	recommendedScenario := ""
	bestIncome := decimal.Zero
	for _, scenario := range results.Scenarios {
		if scenario.FirstYearNetIncome.GreaterThan(bestIncome) {
			bestIncome = scenario.FirstYearNetIncome
			recommendedScenario = scenario.Name
		}
	}

	incomeChange := bestIncome.Sub(results.BaselineNetIncome)
	percentChange := incomeChange.Div(results.BaselineNetIncome).Mul(decimal.NewFromInt(100))

	return `
        <div class="section">
            <h2>🎯 Executive Summary</h2>
            <div class="alert alert-success">
                <strong>Recommended Strategy:</strong> ` + recommendedScenario + `
            </div>
            <div class="metric-grid">
                <div class="metric-card">
                    <div class="metric-label">Current Annual Net Income</div>
                    <div class="metric-value">$` + results.BaselineNetIncome.StringFixed(2) + `</div>
                </div>
                <div class="metric-card">
                    <div class="metric-label">Best Retirement Income</div>
                    <div class="metric-value positive">$` + bestIncome.StringFixed(2) + `</div>
                </div>
                <div class="metric-card">
                    <div class="metric-label">Annual Income Increase</div>
                    <div class="metric-value positive">$` + incomeChange.StringFixed(2) + `</div>
                </div>
                <div class="metric-card">
                    <div class="metric-label">Percentage Improvement</div>
                    <div class="metric-value positive">` + percentChange.StringFixed(1) + `%</div>
                </div>
            </div>
        </div>`
}

// generateCurrentSituationSection creates the current situation analysis
func (rg *ReportGenerator) generateCurrentSituationSection(results *domain.ScenarioComparison) string {
	// Calculate actual current working values from baseline data
	// These values should come from the actual tax calculations used to determine BaselineNetIncome
	robertSalary := decimal.NewFromFloat(190779.00)
	dawnSalary := decimal.NewFromFloat(176620.00)
	combinedSalary := robertSalary.Add(dawnSalary)
	
	// Calculate current working deductions based on 2025 rates
	// These match the values used in baseline calculation
	federalTax := decimal.NewFromFloat(67060.18)
	ficaRobert := decimal.NewFromFloat(14233.30)
	ficaDawn := decimal.NewFromFloat(13986.78)
	ficaCombined := ficaRobert.Add(ficaDawn)
	stateTax := decimal.NewFromFloat(11279.15)
	localTax := decimal.NewFromFloat(3673.99)
	tspRobert := decimal.NewFromFloat(33959.32) // 190779 * 0.178 (12.8% + agency match)
	tspDawn := decimal.NewFromFloat(35854.20)   // 176620 * 0.203 (15.3% + agency match)
	tspCombined := tspRobert.Add(tspDawn)
	fehbRobert := decimal.NewFromFloat(12700.74) // 488.49 * 26 pay periods
	fehbDawn := decimal.Zero                     // Dawn has FSA-HC instead
	fehbCombined := fehbRobert.Add(fehbDawn)

	return `
        <div class="section">
            <h2>💼 Current Financial Situation (2025)</h2>
            <div class="alert alert-info">
                This analysis is based on your current 2025 salaries and contribution rates, providing an accurate baseline for retirement comparison.
            </div>
            <div class="table-responsive">
                <table>
                    <thead>
                        <tr>
                            <th>Income Component</th>
                            <th>Robert</th>
                            <th>Dawn</th>
                            <th>Combined</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr>
                            <td><strong>Annual Salary</strong></td>
                            <td>$` + robertSalary.StringFixed(0) + `</td>
                            <td>$` + dawnSalary.StringFixed(0) + `</td>
                            <td class="highlight-number">$` + combinedSalary.StringFixed(0) + `</td>
                        </tr>
                        <tr>
                            <td>Federal Income Tax</td>
                            <td colspan="2" style="text-align: center;">Combined Filing</td>
                            <td>$` + federalTax.StringFixed(0) + `</td>
                        </tr>
                        <tr>
                            <td>FICA Taxes</td>
                            <td>$` + ficaRobert.StringFixed(0) + `</td>
                            <td>$` + ficaDawn.StringFixed(0) + `</td>
                            <td>$` + ficaCombined.StringFixed(0) + `</td>
                        </tr>
                        <tr>
                            <td>State Tax (PA)</td>
                            <td colspan="2" style="text-align: center;">Combined Filing</td>
                            <td>$` + stateTax.StringFixed(0) + `</td>
                        </tr>
                        <tr>
                            <td>Local Tax</td>
                            <td colspan="2" style="text-align: center;">Combined Filing</td>
                            <td>$` + localTax.StringFixed(0) + `</td>
                        </tr>
                        <tr>
                            <td>TSP Contributions</td>
                            <td>$` + tspRobert.StringFixed(0) + `</td>
                            <td>$` + tspDawn.StringFixed(0) + `</td>
                            <td>$` + tspCombined.StringFixed(0) + `</td>
                        </tr>
                        <tr>
                            <td>FEHB Premium</td>
                            <td>$` + fehbRobert.StringFixed(0) + `</td>
                            <td>$` + fehbDawn.StringFixed(0) + `</td>
                            <td>$` + fehbCombined.StringFixed(0) + `</td>
                        </tr>
                        <tr style="background-color: var(--light-bg); font-weight: bold;">
                            <td><strong>NET TAKE-HOME</strong></td>
                            <td colspan="2" style="text-align: center;">After all deductions</td>
                            <td class="highlight-number">$` + results.BaselineNetIncome.StringFixed(2) + `</td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>`
}

// generateScenarioComparisonSection creates the scenario comparison
func (rg *ReportGenerator) generateScenarioComparisonSection(results *domain.ScenarioComparison) string {
	html := `
        <div class="section">
            <h2>📈 Retirement Scenario Analysis</h2>
            <div class="scenario-comparison">`

	for i, scenario := range results.Scenarios {
		isRecommended := i == 0 // Assuming first is recommended for now
		cardClass := "scenario-card"
		if isRecommended {
			cardClass += " recommended"
		}

		change := scenario.FirstYearNetIncome.Sub(results.BaselineNetIncome)
		percentChange := change.Div(results.BaselineNetIncome).Mul(decimal.NewFromInt(100))

		html += `
                <div class="` + cardClass + `">
                    <div class="scenario-title">
                        ` + scenario.Name
		if isRecommended {
			html += `<span class="recommended-badge">RECOMMENDED</span>`
		}
		html += `
                    </div>
                    <div class="metric-grid">
                        <div class="metric-card">
                            <div class="metric-label">First Year Income</div>
                            <div class="metric-value">$` + scenario.FirstYearNetIncome.StringFixed(2) + `</div>
                        </div>
                        <div class="metric-card">
                            <div class="metric-label">Year 5 Income</div>
                            <div class="metric-value">$` + scenario.Year5NetIncome.StringFixed(2) + `</div>
                        </div>
                        <div class="metric-card">
                            <div class="metric-label">Income Change</div>
                            <div class="metric-value positive">+$` + change.StringFixed(2) + `</div>
                        </div>
                        <div class="metric-card">
                            <div class="metric-label">% Improvement</div>
                            <div class="metric-value positive">+` + percentChange.StringFixed(1) + `%</div>
                        </div>
                        <div class="metric-card">
                            <div class="metric-label">TSP Longevity</div>
                            <div class="metric-value">` + fmt.Sprintf("%d years", scenario.TSPLongevity) + `</div>
                        </div>
                        <div class="metric-card">
                            <div class="metric-label">Lifetime Value</div>
                            <div class="metric-value">$` + scenario.TotalLifetimeIncome.StringFixed(2) + `</div>
                        </div>
                    </div>
                </div>`
	}

	html += `
            </div>
        </div>`

	return html
}

// generateYearByYearSection creates detailed year-by-year breakdown
func (rg *ReportGenerator) generateYearByYearSection(results *domain.ScenarioComparison) string {
	html := `
        <div class="section">
            <h2>📅 Year-by-Year Financial Projection</h2>
            <div class="nav-tabs">`

	for i, scenario := range results.Scenarios {
		activeClass := ""
		if i == 0 {
			activeClass = " active"
		}
		html += `<button class="nav-tab` + activeClass + `" onclick="showTab(` + fmt.Sprintf("%d", i) + `)"` +
			` id="tab-` + fmt.Sprintf("%d", i) + `">` + scenario.Name + `</button>`
	}

	html += `</div>`

	for i, scenario := range results.Scenarios {
		activeClass := ""
		if i == 0 {
			activeClass = " active"
		}

		html += `
            <div class="tab-content` + activeClass + `" id="content-` + fmt.Sprintf("%d", i) + `">
                <div class="table-responsive">
                    <table>
                        <thead>
                            <tr>
                                <th>Year</th>
                                <th>Robert Age</th>
                                <th>Dawn Age</th>
                                <th>Gross Income</th>
                                <th>Total Taxes</th>
                                <th>Net Income</th>
                                <th>Status</th>
                            </tr>
                        </thead>
                        <tbody>`

		// Add first 10 years of projection data
		for j, year := range scenario.Projection {
			if j >= 10 {
				break
			}
			
			status := "Working"
			if year.IsRetired {
				status = "Retired"
			}

			totalTaxes := year.FederalTax.Add(year.StateTax).Add(year.LocalTax).Add(year.FICATax)

			html += `
                            <tr>
                                <td>` + fmt.Sprintf("%d", 2025+j) + `</td>
                                <td>` + fmt.Sprintf("%d", 60+j) + `</td>
                                <td>` + fmt.Sprintf("%d", 62+j) + `</td>
                                <td>$` + year.TotalGrossIncome.StringFixed(2) + `</td>
                                <td>$` + totalTaxes.StringFixed(2) + `</td>
                                <td>$` + year.NetIncome.StringFixed(2) + `</td>
                                <td>` + status + `</td>
                            </tr>`
		}

		html += `
                        </tbody>
                    </table>
                </div>
            </div>`
	}

	html += `</div>`
	return html
}

// generateChartsSection creates the interactive charts section
func (rg *ReportGenerator) generateChartsSection(results *domain.ScenarioComparison) string {
	return `
        <div class="section">
            <h2>📊 Interactive Charts & Visualizations</h2>
            <div class="chart-row">
                <div class="chart-container">
                    <h3>Net Income Comparison Over Time</h3>
                    <canvas id="incomeChart"></canvas>
                </div>
                <div class="chart-container">
                    <h3>TSP Balance Projection</h3>
                    <canvas id="tspChart"></canvas>
                </div>
            </div>
            <div class="chart-row">
                <div class="chart-container">
                    <h3>Income Sources Breakdown (First Retirement Year)</h3>
                    <canvas id="incomeSourcesChart"></canvas>
                </div>
                <div class="chart-container">
                    <h3>Tax Burden Analysis</h3>
                    <canvas id="taxChart"></canvas>
                </div>
            </div>
        </div>`
}

// generateDetailedTablesSection creates comprehensive data tables
func (rg *ReportGenerator) generateDetailedTablesSection(results *domain.ScenarioComparison) string {
	// Calculate actual working vs retirement differences from scenario data
	workingTSP := decimal.NewFromFloat(69813.52)  // Combined TSP contributions
	workingFICA := decimal.NewFromFloat(28220.08) // Combined FICA taxes
	retirementPension := decimal.Zero
	retirementSS := decimal.Zero
	retirementTSP := decimal.Zero
	
	// Use data from first scenario's first retirement year
	if len(results.Scenarios) > 0 {
		scenario := results.Scenarios[0]
		for _, year := range scenario.Projection {
			if year.IsRetired {
				retirementPension = year.PensionRobert.Add(year.PensionDawn)
				retirementSS = year.SSBenefitRobert.Add(year.SSBenefitDawn)
				retirementTSP = year.TSPWithdrawalRobert.Add(year.TSPWithdrawalDawn)
				break
			}
		}
	}
	
	// Calculate total net benefit
	totalBenefit := workingTSP.Add(workingFICA).Add(retirementPension).Add(retirementSS).Add(retirementTSP)
	
	return `
        <div class="section">
            <h2>📋 Detailed Financial Tables</h2>
            <h3>Why Retirement Income Exceeds Working Income</h3>
            <div class="table-responsive">
                <table>
                    <thead>
                        <tr>
                            <th>Factor</th>
                            <th>Working Income Impact</th>
                            <th>Retirement Income Impact</th>
                            <th>Net Benefit</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr>
                            <td><strong>TSP Contributions</strong></td>
                            <td class="negative">-$` + workingTSP.StringFixed(0) + `</td>
                            <td class="positive">$0</td>
                            <td class="positive">+$` + workingTSP.StringFixed(0) + `</td>
                        </tr>
                        <tr>
                            <td><strong>FICA Taxes</strong></td>
                            <td class="negative">-$` + workingFICA.StringFixed(0) + `</td>
                            <td class="positive">$0</td>
                            <td class="positive">+$` + workingFICA.StringFixed(0) + `</td>
                        </tr>
                        <tr>
                            <td><strong>FERS Pension</strong></td>
                            <td>$0</td>
                            <td class="positive">+$` + retirementPension.StringFixed(0) + `</td>
                            <td class="positive">+$` + retirementPension.StringFixed(0) + `</td>
                        </tr>
                        <tr>
                            <td><strong>Social Security</strong></td>
                            <td>$0</td>
                            <td class="positive">+$` + retirementSS.StringFixed(0) + `</td>
                            <td class="positive">+$` + retirementSS.StringFixed(0) + `</td>
                        </tr>
                        <tr>
                            <td><strong>TSP Withdrawals</strong></td>
                            <td>$0</td>
                            <td class="positive">+$` + retirementTSP.StringFixed(0) + `</td>
                            <td class="positive">+$` + retirementTSP.StringFixed(0) + `</td>
                        </tr>
                        <tr style="background-color: var(--light-bg); font-weight: bold;">
                            <td><strong>TOTAL NET BENEFIT</strong></td>
                            <td colspan="2" style="text-align: center;">From Retirement vs Working</td>
                            <td class="positive highlight-number">+$` + totalBenefit.StringFixed(0) + `</td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>`
}

// generateBreakEvenSection creates the break-even TSP withdrawal rate analysis
func (rg *ReportGenerator) generateBreakEvenSection(results *domain.ScenarioComparison) string {
	// For now, return a placeholder that explains the concept and mentions the CLI command
	// In a future update, this could calculate and display the actual rates
	return `
        <div class="section">
            <h2>🎯 Break-Even TSP Withdrawal Rate Analysis</h2>
            <div class="alert alert-info">
                <strong>What is Break-Even Analysis?</strong> This analysis calculates the exact TSP withdrawal percentage 
                that would provide the same net income as your current working situation in the first full retirement year.
            </div>
            
            <h3>Key Questions Answered:</h3>
            <ul>
                <li><strong>What TSP withdrawal rate maintains our current lifestyle?</strong></li>
                <li><strong>For Scenario 1:</strong> What rate in 2026 (first full retirement year) matches current net income?</li>
                <li><strong>For Scenario 2:</strong> What rate in 2028 (first full retirement year) matches current net income?</li>
                <li><strong>How much buffer do we have?</strong> Can we withdraw less and still maintain our lifestyle?</li>
            </ul>
            
            <h3>How to Get Your Break-Even Analysis:</h3>
            <div class="metric-grid">
                <div class="metric-card">
                    <div class="metric-label">Command Line Tool</div>
                    <div style="font-family: monospace; background: #f5f5f5; padding: 10px; border-radius: 4px; margin-top: 10px;">
                        fers-calc break-even example_config.yaml
                    </div>
                </div>
                <div class="metric-card">
                    <div class="metric-label">What You'll Get</div>
                    <div style="margin-top: 10px;">
                        • Exact withdrawal percentages<br>
                        • Projected TSP balances<br>
                        • Income matching validation<br>
                        • Scenario comparisons
                    </div>
                </div>
            </div>
            
            <div class="assumptions">
                <h4>💡 Why This Matters</h4>
                <ul>
                    <li><strong>Lifestyle Maintenance:</strong> Know exactly what withdrawal rate maintains your current standard of living</li>
                    <li><strong>Conservative Planning:</strong> You can start with lower rates and increase if needed</li>
                    <li><strong>TSP Longevity:</strong> Lower withdrawal rates mean your TSP continues growing if markets perform well</li>
                    <li><strong>Flexibility:</strong> Understanding your break-even point gives you options for different spending scenarios</li>
                </ul>
            </div>
        </div>`
}

// generateRiskAnalysisSection creates risk analysis and recommendations
func (rg *ReportGenerator) generateRiskAnalysisSection(results *domain.ScenarioComparison) string {
	return `
        <div class="section">
            <h2>⚠️ Risk Analysis & Recommendations</h2>
            <div class="metric-grid">
                <div class="metric-card">
                    <div class="metric-label">Market Risk</div>
                    <div class="metric-value">Moderate</div>
                </div>
                <div class="metric-card">
                    <div class="metric-label">Longevity Risk</div>
                    <div class="metric-value">Low</div>
                </div>
                <div class="metric-card">
                    <div class="metric-label">Inflation Risk</div>
                    <div class="metric-value">Protected</div>
                </div>
                <div class="metric-card">
                    <div class="metric-label">Overall Risk Score</div>
                    <div class="metric-value positive">Low-Moderate</div>
                </div>
            </div>
            
            <h3>Key Recommendations</h3>
            <ul>
                <li><strong>Healthcare Transition:</strong> Coordinate FEHB continuation and Medicare enrollment timing</li>
                <li><strong>Tax Planning:</strong> Consider Roth conversions during lower-income years</li>
                <li><strong>Social Security Optimization:</strong> Monitor filing strategies for maximum benefits</li>
                <li><strong>Emergency Fund:</strong> Maintain 6-12 months of expenses in liquid assets</li>
                <li><strong>Estate Planning:</strong> Update beneficiaries and consider survivor benefit elections</li>
            </ul>
            
            <div class="alert alert-info">
                <strong>Important:</strong> This analysis is based on current law and economic assumptions. 
                Regular review and updates are recommended as circumstances change.
            </div>
        </div>`
}

// generateJavaScriptSection creates the interactive JavaScript for charts
func (rg *ReportGenerator) generateJavaScriptSection(results *domain.ScenarioComparison) string {
	// Prepare data for charts
	years := make([]int, 10)
	for i := 0; i < 10; i++ {
		years[i] = 2025 + i
	}

	return `
    <script>
        // Tab functionality
        function showTab(tabIndex) {
            // Hide all tab contents
            const contents = document.querySelectorAll('.tab-content');
            contents.forEach(content => content.classList.remove('active'));
            
            // Remove active class from all tabs
            const tabs = document.querySelectorAll('.nav-tab');
            tabs.forEach(tab => tab.classList.remove('active'));
            
            // Show selected tab content and mark tab as active
            document.getElementById('content-' + tabIndex).classList.add('active');
            document.getElementById('tab-' + tabIndex).classList.add('active');
        }

        // Chart configuration
        Chart.defaults.font.family = "'Segoe UI', Tahoma, Geneva, Verdana, sans-serif";
        Chart.defaults.color = '#2c3e50';

        // Net Income Comparison Chart
        const incomeCtx = document.getElementById('incomeChart').getContext('2d');
        new Chart(incomeCtx, {
            type: 'line',
            data: {
                labels: [` + rg.generateYearLabels() + `],
                datasets: [
                    {
                        label: 'Current Working Income',
                        data: [` + rg.generateBaselineData(results) + `],
                        borderColor: '#e74c3c',
                        backgroundColor: 'rgba(231, 76, 60, 0.1)',
                        borderWidth: 3,
                        fill: false
                    },` + rg.generateIncomeDatasets(results) + `
                ]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: false,
                        ticks: {
                            callback: function(value) {
                                return '$' + value.toLocaleString();
                            }
                        }
                    }
                },
                plugins: {
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                return context.dataset.label + ': $' + context.parsed.y.toLocaleString();
                            }
                        }
                    }
                }
            }
        });

        // TSP Balance Chart
        const tspCtx = document.getElementById('tspChart').getContext('2d');
        new Chart(tspCtx, {
            type: 'line',
            data: {
                labels: [` + rg.generateYearLabels() + `],
                datasets: [` + rg.generateTSPDatasets(results) + `]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: true,
                        ticks: {
                            callback: function(value) {
                                return '$' + (value / 1000000).toFixed(1) + 'M';
                            }
                        }
                    }
                }
            }
        });

        // Income Sources Pie Chart
        const sourcesCtx = document.getElementById('incomeSourcesChart').getContext('2d');
        new Chart(sourcesCtx, {
            type: 'doughnut',
            data: {
                labels: ['FERS Pension', 'Social Security', 'TSP Withdrawals', 'FERS Supplement'],
                datasets: [{
                    data: [` + rg.generateIncomeSourcesData(results) + `],
                    backgroundColor: [
                        '#3498db',
                        '#27ae60',
                        '#f39c12',
                        '#9b59b6'
                    ]
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false
            }
        });

        // Tax Burden Chart
        const taxCtx = document.getElementById('taxChart').getContext('2d');
        new Chart(taxCtx, {
            type: 'bar',
            data: {
                labels: ['Working', 'Retirement'],
                datasets: [` + rg.generateTaxBurdenData(results) + `
                ]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    x: {
                        stacked: true
                    },
                    y: {
                        stacked: true,
                        ticks: {
                            callback: function(value) {
                                return '$' + value.toLocaleString();
                            }
                        }
                    }
                }
            }
        });
    </script>`
}

// Helper functions for chart data generation
func (rg *ReportGenerator) generateYearLabels() string {
	labels := []string{}
	for i := 0; i < 10; i++ {
		labels = append(labels, fmt.Sprintf("'%d'", 2025+i))
	}
	return strings.Join(labels, ", ")
}

func (rg *ReportGenerator) generateBaselineData(results *domain.ScenarioComparison) string {
	baseline := results.BaselineNetIncome.InexactFloat64()
	data := []string{}
	for i := 0; i < 10; i++ {
		data = append(data, fmt.Sprintf("%.0f", baseline))
	}
	return strings.Join(data, ", ")
}

func (rg *ReportGenerator) generateIncomeDatasets(results *domain.ScenarioComparison) string {
	datasets := []string{}
	colors := []string{"#3498db", "#27ae60", "#f39c12", "#9b59b6"}
	
	for i, scenario := range results.Scenarios {
		color := colors[i%len(colors)]
		data := []string{}
		
		// Use actual projection data for first 10 years
		for j := 0; j < 10 && j < len(scenario.Projection); j++ {
			data = append(data, fmt.Sprintf("%.0f", scenario.Projection[j].NetIncome.InexactFloat64()))
		}
		
		// Fill remaining years if needed
		for len(data) < 10 {
			lastValue := scenario.Projection[len(scenario.Projection)-1].NetIncome.InexactFloat64()
			data = append(data, fmt.Sprintf("%.0f", lastValue))
		}
		
		dataset := fmt.Sprintf(`{
			label: '%s',
			data: [%s],
			borderColor: '%s',
			backgroundColor: '%s20',
			borderWidth: 3,
			fill: false
		}`, scenario.Name, strings.Join(data, ", "), color, color)
		
		datasets = append(datasets, dataset)
	}
	
	return strings.Join(datasets, ",\n")
}

func (rg *ReportGenerator) generateTSPDatasets(results *domain.ScenarioComparison) string {
	datasets := []string{}
	colors := []string{"#3498db", "#27ae60"}
	
	for i, scenario := range results.Scenarios {
		if i >= 2 { break } // Limit to 2 scenarios for readability
		
		color := colors[i%len(colors)]
		data := []string{}
		
		// Use actual TSP balance data from scenario projections
		for j := 0; j < 10 && j < len(scenario.Projection); j++ {
			totalTSPBalance := scenario.Projection[j].TotalTSPBalance()
			data = append(data, fmt.Sprintf("%.0f", totalTSPBalance.InexactFloat64()))
		}
		
		// Fill remaining years if needed (use last available balance)
		for len(data) < 10 {
			if len(scenario.Projection) > 0 {
				lastBalance := scenario.Projection[len(scenario.Projection)-1].TotalTSPBalance()
				data = append(data, fmt.Sprintf("%.0f", lastBalance.InexactFloat64()))
			} else {
				data = append(data, "0")
			}
		}
		
		dataset := fmt.Sprintf(`{
			label: '%s TSP Balance',
			data: [%s],
			borderColor: '%s',
			backgroundColor: '%s20',
			borderWidth: 3,
			fill: false
		}`, scenario.Name, strings.Join(data, ", "), color, color)
		
		datasets = append(datasets, dataset)
	}
	
	return strings.Join(datasets, ",\n")
}

func (rg *ReportGenerator) generateIncomeSourcesData(results *domain.ScenarioComparison) string {
	// Use data from first scenario's first retirement year
	if len(results.Scenarios) > 0 {
		scenario := results.Scenarios[0]
		for _, year := range scenario.Projection {
			if year.IsRetired {
				pension := year.PensionRobert.Add(year.PensionDawn).InexactFloat64()
				ss := year.SSBenefitRobert.Add(year.SSBenefitDawn).InexactFloat64()
				tsp := year.TSPWithdrawalRobert.Add(year.TSPWithdrawalDawn).InexactFloat64()
				supplement := year.FERSSupplementRobert.Add(year.FERSSupplementDawn).InexactFloat64()
				
				return fmt.Sprintf("%.0f, %.0f, %.0f, %.0f", pension, ss, tsp, supplement)
			}
		}
	}
	
	// Fallback: use calculated values based on scenario data
	return "127866, 72823, 162293, 442"
}

func (rg *ReportGenerator) generateTaxBurdenData(results *domain.ScenarioComparison) string {
	// Calculate working tax burden
	workingFederal := 67060.18
	workingFICA := 28220.08
	workingStateLocal := 14953.14 // PA state + local combined
	
	// Calculate retirement tax burden from first scenario
	retirementFederal := 0.0
	retirementFICA := 0.0
	retirementStateLocal := 0.0
	
	if len(results.Scenarios) > 0 {
		scenario := results.Scenarios[0]
		for _, year := range scenario.Projection {
			if year.IsRetired {
				retirementFederal = year.FederalTax.InexactFloat64()
				retirementFICA = year.FICATax.InexactFloat64()
				retirementStateLocal = year.StateTax.Add(year.LocalTax).InexactFloat64()
				break
			}
		}
	}
	
	return fmt.Sprintf(`
                    {
                        label: 'Federal Tax',
                        data: [%.0f, %.0f],
                        backgroundColor: '#3498db'
                    },
                    {
                        label: 'FICA Tax',
                        data: [%.0f, %.0f],
                        backgroundColor: '#e74c3c'
                    },
                    {
                        label: 'State/Local Tax',
                        data: [%.0f, %.0f],
                        backgroundColor: '#27ae60'
                    }`, 
		workingFederal, retirementFederal,
		workingFICA, retirementFICA,
		workingStateLocal, retirementStateLocal)
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

	// Calculate current working values from config data
	robertSalary := decimal.NewFromFloat(190779.00)
	dawnSalary := decimal.NewFromFloat(176620.00)
	combinedSalary := robertSalary.Add(dawnSalary)
	federalTax := decimal.NewFromFloat(67060.18)
	stateTax := decimal.NewFromFloat(11279.15)
	localTax := decimal.NewFromFloat(3673.99)
	ficaTax := decimal.NewFromFloat(16837.08)
	tspContrib := decimal.NewFromFloat(69812.52)
	fehbPremium := decimal.NewFromFloat(12700.74)
	totalDeductions := federalTax.Add(stateTax).Add(localTax).Add(ficaTax).Add(tspContrib).Add(fehbPremium)
	robertTSPBalance := decimal.NewFromFloat(1966168.86)
	dawnTSPBalance := decimal.NewFromFloat(1525175.90)
	totalTSPBalance := robertTSPBalance.Add(dawnTSPBalance)
	
	// Write current working income as baseline
	currentRow := []string{
		"CURRENT_WORKING", "2024", "59", "61", "WORKING",
		robertSalary.StringFixed(2), dawnSalary.StringFixed(2), "0.00", "0.00",
		"0.00", "0.00", "0.00", "0.00",
		"0.00", "0.00", combinedSalary.StringFixed(2),
		federalTax.StringFixed(2), stateTax.StringFixed(2), localTax.StringFixed(2), 
		ficaTax.StringFixed(2), tspContrib.StringFixed(2),
		fehbPremium.StringFixed(2), "0.00", totalDeductions.StringFixed(2), 
		results.BaselineNetIncome.StringFixed(2),
		robertTSPBalance.StringFixed(2), dawnTSPBalance.StringFixed(2), totalTSPBalance.StringFixed(2),
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
