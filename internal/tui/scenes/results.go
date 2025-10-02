package scenes

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/rgehrsitz/rpgo/internal/tui/components"
	"github.com/rgehrsitz/rpgo/internal/tui/tuistyles"
)

// ResultsModel represents the results display scene
type ResultsModel struct {
	scenarioName string
	summary      *domain.ScenarioSummary
	width        int
	height       int
}

// NewResultsModel creates a new results scene model
func NewResultsModel() *ResultsModel {
	return &ResultsModel{}
}

// SetResults updates the results to display
func (m *ResultsModel) SetResults(scenarioName string, summary *domain.ScenarioSummary) {
	m.scenarioName = scenarioName
	m.summary = summary
}

// SetSize updates the scene dimensions
func (m *ResultsModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update handles messages for the results scene
func (m *ResultsModel) Update(msg tea.Msg) (*ResultsModel, tea.Cmd) {
	// Results scene is mostly read-only
	return m, nil
}

// View renders the results scene
func (m *ResultsModel) View() string {
	if m.summary == nil {
		return renderNoResultsState()
	}

	// Build header
	header := renderResultsHeader(m.scenarioName)

	// Build key metrics cards
	metrics := renderKeyMetrics(m.summary)

	// Build year-by-year summary (first 5 years)
	yearSummary := renderYearSummary(m.summary)

	// Build help
	help := renderResultsHelp()

	// Combine sections
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		metrics,
		"",
		yearSummary,
		"",
		help,
	)

	return content
}

// renderNoResultsState renders empty state
func renderNoResultsState() string {
	return `No results to display.

Please calculate a scenario first from the Parameters screen.

Press ESC to go back.`
}

// renderResultsHeader renders the header with scenario name
func renderResultsHeader(scenarioName string) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(tuistyles.ColorPrimary)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(tuistyles.ColorMuted).
		Italic(true)

	title := titleStyle.Render("Calculation Results")
	subtitle := subtitleStyle.Render("Scenario: " + scenarioName)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		subtitle,
	)
}

// renderKeyMetrics renders the key metrics as cards
func renderKeyMetrics(summary *domain.ScenarioSummary) string {
	cards := []*components.MetricCard{}

	// First year income
	card := components.NewMetricCard(
		"First Year Net Income",
		formatCurrency(summary.FirstYearNetIncome.InexactFloat64()),
	).WithWidth(30)
	cards = append(cards, card)

	// TSP final balance
	card = components.NewMetricCard(
		"TSP Final Balance",
		formatCurrency(summary.FinalTSPBalance.InexactFloat64()),
	).WithWidth(30)
	cards = append(cards, card)

	// TSP longevity
	if summary.TSPLongevity > 0 {
		card = components.NewMetricCard(
			"TSP Longevity",
			fmt.Sprintf("%d years", summary.TSPLongevity),
		).WithWidth(30)
		cards = append(cards, card)
	}

	// Total lifetime income
	card = components.NewMetricCard(
		"Total Lifetime Income",
		formatCurrency(summary.TotalLifetimeIncome.InexactFloat64()),
	).WithWidth(30)
	cards = append(cards, card)

	if len(cards) == 0 {
		return "No summary metrics available."
	}

	// Display in grid (3 columns)
	return components.MetricGrid(cards, 3)
}

// renderYearSummary renders a summary of the first few years
func renderYearSummary(summary *domain.ScenarioSummary) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(tuistyles.ColorPrimary).
		MarginBottom(1)

	tableStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tuistyles.ColorBorder).
		Padding(1, 2)

	var content strings.Builder
	content.WriteString(titleStyle.Render("Year-by-Year Summary"))
	content.WriteString("\n\n")

	// Table header
	headerStyle := lipgloss.NewStyle().
		Foreground(tuistyles.ColorPrimary).
		Bold(true)

	header := fmt.Sprintf("%-6s  %-20s  %-20s  %-20s",
		"Year", "Gross Income", "Federal Tax", "Net Income")
	content.WriteString(headerStyle.Render(header))
	content.WriteString("\n")
	content.WriteString(strings.Repeat("─", 70))
	content.WriteString("\n")

	// Show first 10 years (or fewer if not available)
	maxYears := 10
	if summary.Projection != nil && len(summary.Projection) < maxYears {
		maxYears = len(summary.Projection)
	}

	if summary.Projection != nil {
		for i := 0; i < maxYears && i < len(summary.Projection); i++ {
			year := summary.Projection[i]

			grossIncome := formatCurrencyShort(year.TotalGrossIncome.InexactFloat64())
			fedTax := formatCurrencyShort(year.FederalTax.InexactFloat64())
			netIncome := formatCurrencyShort(year.NetIncome.InexactFloat64())

			row := fmt.Sprintf("%-6d  %-20s  %-20s  %-20s",
				year.Year, grossIncome, fedTax, netIncome)
			content.WriteString(row)
			content.WriteString("\n")
		}
	}

	if summary.Projection != nil && len(summary.Projection) > maxYears {
		moreStyle := lipgloss.NewStyle().
			Foreground(tuistyles.ColorMuted).
			Italic(true)
		content.WriteString("\n")
		content.WriteString(moreStyle.Render(fmt.Sprintf("... and %d more years", len(summary.Projection)-maxYears)))
	}

	return tableStyle.Render(content.String())
}

// renderResultsHelp renders keyboard shortcuts
func renderResultsHelp() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(tuistyles.ColorMuted)

	return helpStyle.Render("ESC back to parameters • s scenarios • h home • p parameters again")
}

// formatCurrency formats a currency value
func formatCurrency(value float64) string {
	if value < 0 {
		return fmt.Sprintf("-$%.0f", -value)
	}
	return fmt.Sprintf("$%.0f", value)
}

// formatCurrencyShort formats currency in short form
func formatCurrencyShort(value float64) string {
	if value >= 1000000 {
		return fmt.Sprintf("$%.1fM", value/1000000)
	} else if value >= 1000 {
		return fmt.Sprintf("$%.0fK", value/1000)
	}
	return fmt.Sprintf("$%.0f", value)
}
