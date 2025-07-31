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

// GenerateConsoleReport generates a console-formatted report
func (rg *ReportGenerator) GenerateConsoleReport(results *domain.ScenarioComparison) error {
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("FERS RETIREMENT PLANNING CALCULATOR - RESULTS")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()
	
	// Pre-retirement baseline information
	fmt.Println("PRE-RETIREMENT INCOME (Current)")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Combined Net Income: $%s\n", results.BaselineNetIncome.StringFixed(2))
	fmt.Printf("Monthly Net Income: $%s\n", results.BaselineNetIncome.Div(decimal.NewFromInt(12)).StringFixed(2))
	fmt.Println()
	
	// Scenario comparison with clear pre/post retirement comparison
	fmt.Println("RETIREMENT SCENARIO ANALYSIS")
	fmt.Println(strings.Repeat("-", 40))
	
	for i, scenario := range results.Scenarios {
		fmt.Printf("%d. %s\n", i+1, scenario.Name)
		fmt.Println("   Pre-Retirement → Post-Retirement Comparison:")
		fmt.Printf("   • Current Annual Net: $%s\n", results.BaselineNetIncome.StringFixed(2))
		fmt.Printf("   • First Year Net:     $%s\n", scenario.FirstYearNetIncome.StringFixed(2))
		
		// Calculate the change
		change := scenario.FirstYearNetIncome.Sub(results.BaselineNetIncome)
		percentageChange := change.Div(results.BaselineNetIncome).Mul(decimal.NewFromInt(100))
		monthlyChange := change.Div(decimal.NewFromInt(12))
		
		if change.GreaterThanOrEqual(decimal.Zero) {
			fmt.Printf("   • CHANGE: +$%s (+%s%%)\n", change.StringFixed(2), percentageChange.StringFixed(2))
			fmt.Printf("   • Monthly Change: +$%s\n", monthlyChange.StringFixed(2))
		} else {
			fmt.Printf("   • CHANGE: $%s (%s%%)\n", change.StringFixed(2), percentageChange.StringFixed(2))
			fmt.Printf("   • Monthly Change: $%s\n", monthlyChange.StringFixed(2))
		}
		
		fmt.Println("   Long-term Projection:")
		fmt.Printf("   • Year 5 Net Income: $%s\n", scenario.Year5NetIncome.StringFixed(2))
		fmt.Printf("   • Year 10 Net Income: $%s\n", scenario.Year10NetIncome.StringFixed(2))
		fmt.Printf("   • TSP Longevity: %d years\n", scenario.TSPLongevity)
		fmt.Printf("   • Total Lifetime Income (PV): $%s\n", scenario.TotalLifetimeIncome.StringFixed(2))
		fmt.Println()
	}
	
	// Summary of best scenario
	fmt.Println("SUMMARY")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Recommended Scenario: %s\n", results.ImmediateImpact.RecommendedScenario)
	
	// Find the recommended scenario to show its specific change
	for _, scenario := range results.Scenarios {
		if scenario.Name == results.ImmediateImpact.RecommendedScenario {
			change := scenario.FirstYearNetIncome.Sub(results.BaselineNetIncome)
			percentageChange := change.Div(results.BaselineNetIncome).Mul(decimal.NewFromInt(100))
			monthlyChange := change.Div(decimal.NewFromInt(12))
			
			fmt.Printf("First Year Change: $%s (%s%%)\n", change.StringFixed(2), percentageChange.StringFixed(2))
			fmt.Printf("Monthly Change: $%s\n", monthlyChange.StringFixed(2))
			break
		}
	}
	fmt.Println()
	
	// Key considerations
	fmt.Println("KEY CONSIDERATIONS")
	fmt.Println(strings.Repeat("-", 40))
	for _, consideration := range results.ImmediateImpact.KeyConsiderations {
		fmt.Printf("• %s\n", consideration)
	}
	fmt.Println()
	
	// Long-term analysis
	fmt.Println("LONG-TERM ANALYSIS")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Best for Income: %s\n", results.LongTermProjection.BestScenarioForIncome)
	fmt.Printf("Best for Longevity: %s\n", results.LongTermProjection.BestScenarioForLongevity)
	fmt.Printf("Risk Assessment: %s\n", results.LongTermProjection.RiskAssessment)
	fmt.Println()
	
	fmt.Println("RECOMMENDATIONS")
	fmt.Println(strings.Repeat("-", 40))
	for _, recommendation := range results.LongTermProjection.Recommendations {
		fmt.Printf("• %s\n", recommendation)
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