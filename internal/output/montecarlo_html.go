package output

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rpgo/retirement-calculator/internal/calculation"
	"github.com/shopspring/decimal"
)

// MonteCarloHTMLReport generates an interactive HTML report for FERS Monte Carlo results
type MonteCarloHTMLReport struct {
	Result *calculation.FERSMonteCarloResult
	Config calculation.FERSMonteCarloConfig
}

// GenerateHTMLReport creates an interactive HTML report with charts
func (m *MonteCarloHTMLReport) GenerateHTMLReport(outputPath string) error {
	// Create output directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate the HTML content
	htmlContent := m.generateHTMLContent()

	// Write to file
	if err := os.WriteFile(outputPath, []byte(htmlContent), 0644); err != nil {
		return fmt.Errorf("failed to write HTML report: %w", err)
	}

	return nil
}

// generateHTMLContent creates the complete HTML report with embedded JavaScript
func (m *MonteCarloHTMLReport) generateHTMLContent() string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>FERS Monte Carlo Analysis Report</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/chartjs-adapter-date-fns"></script>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            margin: 0;
            padding: 20px;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            min-height: 100vh;
        }
        .container {
            max-width: 1400px;
            margin: 0 auto;
            background: white;
            border-radius: 15px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #2c3e50 0%%, #3498db 100%%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        .header h1 {
            margin: 0;
            font-size: 2.5em;
            font-weight: 300;
        }
        .header .subtitle {
            margin-top: 10px;
            opacity: 0.9;
            font-size: 1.1em;
        }
        .content {
            padding: 30px;
        }
        .summary-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .summary-card {
            background: #f8f9fa;
            border-radius: 10px;
            padding: 20px;
            text-align: center;
            border-left: 4px solid #3498db;
        }
        .summary-card.success {
            border-left-color: #27ae60;
        }
        .summary-card.warning {
            border-left-color: #f39c12;
        }
        .summary-card.danger {
            border-left-color: #e74c3c;
        }
        .summary-card h3 {
            margin: 0 0 10px 0;
            color: #2c3e50;
            font-size: 1.1em;
        }
        .summary-card .value {
            font-size: 2em;
            font-weight: bold;
            color: #3498db;
        }
        .summary-card.success .value {
            color: #27ae60;
        }
        .summary-card.warning .value {
            color: #f39c12;
        }
        .summary-card.danger .value {
            color: #e74c3c;
        }
        .chart-container {
            background: white;
            border-radius: 10px;
            padding: 20px;
            margin-bottom: 30px;
            box-shadow: 0 5px 15px rgba(0,0,0,0.08);
        }
        .chart-container h3 {
            margin: 0 0 20px 0;
            color: #2c3e50;
            font-size: 1.3em;
        }
        .chart-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 20px;
            margin-bottom: 30px;
        }
        .full-width {
            grid-column: 1 / -1;
        }
        .percentile-table {
            width: 100%%;
            border-collapse: collapse;
            margin-top: 20px;
        }
        .percentile-table th,
        .percentile-table td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }
        .percentile-table th {
            background: #f8f9fa;
            font-weight: 600;
            color: #2c3e50;
        }
        .percentile-table tr:hover {
            background: #f8f9fa;
        }
        .risk-assessment {
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
            padding: 25px;
            border-radius: 10px;
            margin-bottom: 30px;
        }
        .risk-assessment h3 {
            margin: 0 0 15px 0;
            font-size: 1.4em;
        }
        .recommendations {
            background: #e8f5e8;
            border: 1px solid #27ae60;
            border-radius: 10px;
            padding: 20px;
            margin-bottom: 30px;
        }
        .recommendations h3 {
            margin: 0 0 15px 0;
            color: #27ae60;
        }
        .recommendations ul {
            margin: 0;
            padding-left: 20px;
        }
        .recommendations li {
            margin-bottom: 8px;
            color: #2c3e50;
        }
        .footer {
            background: #2c3e50;
            color: white;
            text-align: center;
            padding: 20px;
            font-size: 0.9em;
        }
        @media (max-width: 768px) {
            .chart-grid {
                grid-template-columns: 1fr;
            }
            .summary-grid {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎯 FERS Monte Carlo Analysis</h1>
            <div class="subtitle">Comprehensive Retirement Scenario Analysis</div>
        </div>
        
        <div class="content">
            <!-- Summary Cards -->
            <div class="summary-grid">
                <div class="summary-card %s">
                    <h3>Success Rate</h3>
                    <div class="value">%.1f%%</div>
                </div>
                <div class="summary-card">
                    <h3>Median Net Income</h3>
                    <div class="value">$%s</div>
                </div>
                <div class="summary-card">
                    <h3>Simulations</h3>
                    <div class="value">%d</div>
                </div>
                <div class="summary-card">
                    <h3>Risk Level</h3>
                    <div class="value">%s</div>
                </div>
            </div>

            <!-- Charts Grid -->
            <div class="chart-grid">
                <div class="chart-container">
                    <h3>📊 Net Income Distribution</h3>
                    <canvas id="netIncomeChart" width="400" height="300"></canvas>
                </div>
                <div class="chart-container">
                    <h3>💰 TSP Balance Distribution</h3>
                    <canvas id="tspBalanceChart" width="400" height="300"></canvas>
                </div>
            </div>

            <div class="chart-container full-width">
                <h3>📈 Net Income Percentiles</h3>
                <canvas id="percentileChart" width="800" height="400"></canvas>
            </div>

            <!-- Percentile Table -->
            <div class="chart-container">
                <h3>📋 Detailed Percentile Analysis</h3>
                <table class="percentile-table">
                    <thead>
                        <tr>
                            <th>Percentile</th>
                            <th>Net Income</th>
                            <th>Interpretation</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr>
                            <td>10th</td>
                            <td>$%s</td>
                            <td>Worst 10%% of scenarios</td>
                        </tr>
                        <tr>
                            <td>25th</td>
                            <td>$%s</td>
                            <td>Below average scenarios</td>
                        </tr>
                        <tr>
                            <td>50th (Median)</td>
                            <td>$%s</td>
                            <td>Typical scenario</td>
                        </tr>
                        <tr>
                            <td>75th</td>
                            <td>$%s</td>
                            <td>Above average scenarios</td>
                        </tr>
                        <tr>
                            <td>90th</td>
                            <td>$%s</td>
                            <td>Best 10%% of scenarios</td>
                        </tr>
                    </tbody>
                </table>
            </div>

            <!-- Risk Assessment -->
            <div class="risk-assessment">
                <h3>⚠️ Risk Assessment</h3>
                <p><strong>Risk Level:</strong> %s</p>
                <p><strong>Primary Concerns:</strong> %s</p>
                <p><strong>Market Sensitivity:</strong> %s</p>
            </div>

            <!-- Recommendations -->
            <div class="recommendations">
                <h3>💡 Recommendations</h3>
                <ul>
                    %s
                </ul>
            </div>
        </div>

        <div class="footer">
            <p>Generated on %s | FERS Monte Carlo Analysis Tool</p>
        </div>
    </div>

    <script>
        // Chart.js configuration
        Chart.defaults.font.family = "'Segoe UI', Tahoma, Geneva, Verdana, sans-serif";
        Chart.defaults.color = '#2c3e50';

        // Net Income Distribution Chart
        const netIncomeCtx = document.getElementById('netIncomeChart').getContext('2d');
        new Chart(netIncomeCtx, {
            type: 'bar',
            data: %s,
            options: {
                responsive: true,
                maintainAspectRatio: true,
                aspectRatio: 2,
                animation: {
                    duration: 0
                },
                plugins: {
                    legend: {
                        display: false
                    }
                },
                scales: {
                    x: {
                        title: {
                            display: true,
                            text: 'Net Income ($)'
                        }
                    },
                    y: {
                        title: {
                            display: true,
                            text: 'Frequency'
                        }
                    }
                }
            }
        });

        // TSP Balance Distribution Chart
        const tspBalanceCtx = document.getElementById('tspBalanceChart').getContext('2d');
        new Chart(tspBalanceCtx, {
            type: 'bar',
            data: %s,
            options: {
                responsive: true,
                maintainAspectRatio: true,
                aspectRatio: 2,
                animation: {
                    duration: 0
                },
                plugins: {
                    legend: {
                        display: false
                    }
                },
                scales: {
                    x: {
                        title: {
                            display: true,
                            text: 'TSP Balance ($)'
                        }
                    },
                    y: {
                        title: {
                            display: true,
                            text: 'Frequency'
                        }
                    }
                }
            }
        });

        // Percentile Chart
        const percentileCtx = document.getElementById('percentileChart').getContext('2d');
        new Chart(percentileCtx, {
            type: 'line',
            data: {
                labels: ['10th', '25th', '50th', '75th', '90th'],
                datasets: [{
                    label: 'Net Income by Percentile',
                    data: %s,
                    borderColor: 'rgba(52, 152, 219, 1)',
                    backgroundColor: 'rgba(52, 152, 219, 0.1)',
                    borderWidth: 3,
                    fill: true,
                    tension: 0.4
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: true,
                aspectRatio: 2,
                animation: {
                    duration: 0
                },
                plugins: {
                    legend: {
                        display: false
                    }
                },
                scales: {
                    x: {
                        title: {
                            display: true,
                            text: 'Percentile'
                        }
                    },
                    y: {
                        title: {
                            display: true,
                            text: 'Net Income ($)'
                        }
                    }
                }
            }
        });
    </script>
</body>
</html>`,
		m.getSuccessRateClass(),
		m.Result.SuccessRate.Mul(decimal.NewFromFloat(100)).InexactFloat64(),
		m.formatCurrency(m.Result.MedianNetIncome),
		m.Config.NumSimulations,
		m.getRiskLevel(),
		m.formatCurrency(m.Result.NetIncomePercentiles.P10),
		m.formatCurrency(m.Result.NetIncomePercentiles.P25),
		m.formatCurrency(m.Result.NetIncomePercentiles.P50),
		m.formatCurrency(m.Result.NetIncomePercentiles.P75),
		m.formatCurrency(m.Result.NetIncomePercentiles.P90),
		m.getRiskLevel(),
		m.getPrimaryConcerns(),
		m.getMarketSensitivity(),
		m.generateRecommendationsHTML(),
		time.Now().Format("January 2, 2006 at 3:04 PM"),
		m.generateNetIncomeData(),
		m.generateTSPBalanceData(),
		m.generatePercentileData())
}

// Helper methods for HTML generation
func (m *MonteCarloHTMLReport) getSuccessRateClass() string {
	rate := m.Result.SuccessRate.Mul(decimal.NewFromFloat(100))
	rateFloat, _ := rate.Float64()
	if rateFloat >= 90 {
		return "success"
	} else if rateFloat >= 70 {
		return "warning"
	}
	return "danger"
}

func (m *MonteCarloHTMLReport) getRiskLevel() string {
	rate := m.Result.SuccessRate.Mul(decimal.NewFromFloat(100))
	rateFloat, _ := rate.Float64()
	if rateFloat >= 90 {
		return "🟢 Low"
	} else if rateFloat >= 70 {
		return "🟡 Moderate"
	}
	return "🔴 High"
}

func (m *MonteCarloHTMLReport) getPrimaryConcerns() string {
	rate := m.Result.SuccessRate.Mul(decimal.NewFromFloat(100))
	rateFloat, _ := rate.Float64()
	if rateFloat >= 90 {
		return "Minimal concerns. Your retirement plan appears robust."
	} else if rateFloat >= 70 {
		return "Market volatility could impact retirement income. Consider conservative strategies."
	}
	return "Significant risk of income shortfall. Immediate action recommended."
}

func (m *MonteCarloHTMLReport) getMarketSensitivity() string {
	// Calculate coefficient of variation for net income
	median := m.Result.MedianNetIncome
	if median.IsZero() {
		return "Unable to determine"
	}

	// Use the range between 10th and 90th percentile as a proxy for variability
	incomeRange := m.Result.NetIncomePercentiles.P90.Sub(m.Result.NetIncomePercentiles.P10)
	cv := incomeRange.Div(median)

	cvFloat, _ := cv.Float64()
	if cvFloat < 0.5 {
		return "Low - Income is relatively stable across market conditions"
	} else if cvFloat < 1.0 {
		return "Moderate - Income varies with market performance"
	}
	return "High - Income is highly sensitive to market conditions"
}

func (m *MonteCarloHTMLReport) generateRecommendationsHTML() string {
	rate := m.Result.SuccessRate.Mul(decimal.NewFromFloat(100))
	rateFloat, _ := rate.Float64()
	var recommendations []string

	if rateFloat < 90 {
		recommendations = append(recommendations, "Consider increasing TSP contributions to improve retirement security")
		recommendations = append(recommendations, "Review withdrawal strategies to optimize income sustainability")
	}

	if rateFloat < 70 {
		recommendations = append(recommendations, "Consider delaying retirement to increase benefits")
		recommendations = append(recommendations, "Explore additional income sources or part-time work")
		recommendations = append(recommendations, "Consult with a financial advisor for personalized planning")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Maintain current retirement strategy")
		recommendations = append(recommendations, "Regularly review and adjust plan as circumstances change")
	}

	html := ""
	for _, rec := range recommendations {
		html += fmt.Sprintf("<li>%s</li>", rec)
	}
	return html
}

func (m *MonteCarloHTMLReport) formatCurrency(amount decimal.Decimal) string {
	return amount.StringFixed(0)
}

func (m *MonteCarloHTMLReport) generateNetIncomeData() string {
	// Create histogram bins for net income distribution
	var incomes []decimal.Decimal
	for _, sim := range m.Result.Simulations {
		if sim.Success {
			incomes = append(incomes, sim.NetIncomeMetrics.AverageNetIncome)
		}
	}

	if len(incomes) == 0 {
		return "{labels: [], datasets: [{data: []}]}"
	}

	// Create bins (simplified approach)
	bins := m.createHistogramBins(incomes, 10)
	
	// Convert to Chart.js format
	labels := "["
	data := "["
	for i, bin := range bins {
		if i > 0 {
			labels += ", "
			data += ", "
		}
		labels += fmt.Sprintf("\"$%s\"", bin.Label)
		data += fmt.Sprintf("%d", bin.Count)
	}
	labels += "]"
	data += "]"

	return fmt.Sprintf("{labels: %s, datasets: [{label: 'Simulations', data: %s, backgroundColor: 'rgba(52, 152, 219, 0.6)', borderColor: 'rgba(52, 152, 219, 1)', borderWidth: 1}]}", labels, data)
}

func (m *MonteCarloHTMLReport) generateTSPBalanceData() string {
	// For now, use a simplified TSP balance proxy based on TSP longevity
	// This should be enhanced with actual TSP balance tracking
	var balances []decimal.Decimal
	for _, sim := range m.Result.Simulations {
		if sim.Success {
			// Estimate TSP balance based on longevity and income
			// This is a simplified proxy - real implementation would track actual TSP balances
			estimatedBalance := sim.NetIncomeMetrics.AverageNetIncome.Mul(decimal.NewFromInt(int64(sim.TSPMetrics.Longevity)))
			balances = append(balances, estimatedBalance)
		}
	}

	if len(balances) == 0 {
		return "{labels: [], datasets: [{data: []}]}"
	}

	// Create bins for TSP balance distribution
	bins := m.createHistogramBins(balances, 10)
	
	// Convert to Chart.js format
	labels := "["
	data := "["
	for i, bin := range bins {
		if i > 0 {
			labels += ", "
			data += ", "
		}
		labels += "\"$" + bin.Label + "\""
		data += fmt.Sprintf("%d", bin.Count)
	}
	labels += "]"
	data += "]"

	return fmt.Sprintf("{labels: %s, datasets: [{label: 'Simulations', data: %s, backgroundColor: 'rgba(39, 174, 96, 0.6)', borderColor: 'rgba(39, 174, 96, 1)', borderWidth: 1}]}", labels, data)
}

// HistogramBin represents a bin in a histogram
type HistogramBin struct {
	Label string
	Count int
	Min   decimal.Decimal
	Max   decimal.Decimal
}

func (m *MonteCarloHTMLReport) createHistogramBins(values []decimal.Decimal, numBins int) []HistogramBin {
	if len(values) == 0 {
		return []HistogramBin{}
	}

	// Find min and max
	min := values[0]
	max := values[0]
	for _, v := range values {
		if v.LessThan(min) {
			min = v
		}
		if v.GreaterThan(max) {
			max = v
		}
	}

	// Create bins
	binWidth := max.Sub(min).Div(decimal.NewFromInt(int64(numBins)))
	bins := make([]HistogramBin, numBins)
	
	for i := 0; i < numBins; i++ {
		binMin := min.Add(binWidth.Mul(decimal.NewFromInt(int64(i))))
		binMax := min.Add(binWidth.Mul(decimal.NewFromInt(int64(i + 1))))
		
		bins[i] = HistogramBin{
			Label: binMin.Div(decimal.NewFromInt(1000)).StringFixed(0) + "K",
			Min:   binMin,
			Max:   binMax,
			Count: 0,
		}
	}

	// Count values in each bin
	for _, value := range values {
		for i := range bins {
			if value.GreaterThanOrEqual(bins[i].Min) && (i == len(bins)-1 || value.LessThan(bins[i].Max)) {
				bins[i].Count++
				break
			}
		}
	}

	return bins
}

func (m *MonteCarloHTMLReport) generatePercentileData() string {
	// Generate data for percentile chart
	return fmt.Sprintf("[%s, %s, %s, %s, %s]",
		m.Result.NetIncomePercentiles.P10.StringFixed(0),
		m.Result.NetIncomePercentiles.P25.StringFixed(0),
		m.Result.NetIncomePercentiles.P50.StringFixed(0),
		m.Result.NetIncomePercentiles.P75.StringFixed(0),
		m.Result.NetIncomePercentiles.P90.StringFixed(0))
}
