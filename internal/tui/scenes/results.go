package scenes

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shopspring/decimal"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/rgehrsitz/rpgo/internal/tui/components"
	"github.com/rgehrsitz/rpgo/internal/tui/tuistyles"
)

// ResultsModel represents the results display scene
type ResultsModel struct {
	scenarioName string
	summary      *domain.ScenarioSummary
	viewport     viewport.Model
	ready        bool
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
	m.ready = false // Reset viewport when new results arrive
}

// SetSize updates the scene dimensions
func (m *ResultsModel) SetSize(width, height int) {
	m.width = width
	m.height = height

	if !m.ready {
		// Initialize viewport with proper dimensions
		// Reserve space for header (4 lines) and help (2 lines)
		headerHeight := 6
		m.viewport = viewport.New(width, height-headerHeight)
		m.viewport.YPosition = headerHeight
		m.ready = true
	} else {
		// Update existing viewport size
		m.viewport.Width = width
		m.viewport.Height = height - 6
	}
}

// Update handles messages for the results scene
func (m *ResultsModel) Update(msg tea.Msg) (*ResultsModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			m.viewport.ScrollUp(1)
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			m.viewport.ScrollDown(1)
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgup", "b"))):
			m.viewport.PageUp()
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgdown", "f", " "))):
			m.viewport.PageDown()
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("g"))):
			m.viewport.GotoTop()
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("G"))):
			m.viewport.GotoBottom()
			return m, nil
		}
	}

	// Update viewport
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the results scene
func (m *ResultsModel) View() string {
	if m.summary == nil {
		return renderNoResultsState()
	}

	// Build header (shown above viewport)
	header := renderResultsHeader(m.scenarioName)

	// Build scrollable content (metrics + IRMAA + withdrawal sequencing + all years)
	metrics := renderKeyMetrics(m.summary)
	irmaaSection := renderIRMAAAnalysis(m.summary)
	withdrawalSection := renderWithdrawalSequencing(m.summary)
	yearSummary := renderYearSummaryFull(m.summary) // Show ALL years

	sections := []string{metrics}
	if irmaaSection != "" {
		sections = append(sections, "", irmaaSection)
	}
	if withdrawalSection != "" {
		sections = append(sections, "", withdrawalSection)
	}
	sections = append(sections, "", yearSummary)

	scrollableContent := lipgloss.JoinVertical(
		lipgloss.Left,
		sections...,
	)

	// Set viewport content if ready
	if m.ready {
		m.viewport.SetContent(scrollableContent)
	}

	// Build help (shown below viewport)
	help := renderResultsHelpScrollable()

	// Combine: header + viewport + help
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		m.viewport.View(),
		"",
		help,
	)
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

// renderYearSummaryFull renders ALL years (for scrollable viewport)
func renderYearSummaryFull(summary *domain.ScenarioSummary) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(tuistyles.ColorPrimary).
		MarginBottom(1)

	var content strings.Builder
	content.WriteString(titleStyle.Render("Year-by-Year Summary (Scroll to see all)"))
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

	// Show ALL years
	if summary.Projection != nil {
		for _, year := range summary.Projection {
			grossIncome := formatCurrencyShort(year.TotalGrossIncome.InexactFloat64())
			fedTax := formatCurrencyShort(year.FederalTax.InexactFloat64())
			netIncome := formatCurrencyShort(year.NetIncome.InexactFloat64())

			row := fmt.Sprintf("%-6d  %-20s  %-20s  %-20s",
				year.Year, grossIncome, fedTax, netIncome)
			content.WriteString(row)
			content.WriteString("\n")
		}
	}

	return content.String()
}

// renderResultsHelpScrollable renders keyboard shortcuts with scroll instructions
func renderResultsHelpScrollable() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(tuistyles.ColorMuted)

	return helpStyle.Render("↑/↓ scroll • PgUp/PgDn page • g/G top/bottom • ESC back • s scenarios • h home")
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

// renderIRMAAAnalysis renders IRMAA risk analysis if available
func renderIRMAAAnalysis(summary *domain.ScenarioSummary) string {
	if summary.IRMAAAnalysis == nil {
		return ""
	}

	analysis := summary.IRMAAAnalysis

	var content strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(tuistyles.ColorPrimary).
		Underline(true)

	content.WriteString(titleStyle.Render("IRMAA Risk Analysis"))
	content.WriteString("\n\n")

	// Summary metrics
	if len(analysis.YearsWithBreaches) > 0 {
		warningStyle := lipgloss.NewStyle().
			Foreground(tuistyles.ColorDanger).
			Bold(true)

		content.WriteString(warningStyle.Render("⚠️  IRMAA Breaches Detected"))
		content.WriteString("\n\n")

		content.WriteString(fmt.Sprintf("Years with breaches: %d\n", len(analysis.YearsWithBreaches)))
		if analysis.FirstBreachYear > 0 {
			content.WriteString(fmt.Sprintf("First breach: Year %d\n", analysis.FirstBreachYear))
		}
		content.WriteString(fmt.Sprintf("Total IRMAA cost: %s\n", formatCurrency(analysis.TotalIRMAACost.InexactFloat64())))
	} else if len(analysis.YearsWithWarnings) > 0 {
		warningStyle := lipgloss.NewStyle().
			Foreground(tuistyles.ColorAccent).
			Bold(true)

		content.WriteString(warningStyle.Render("⚠️  IRMAA Warnings"))
		content.WriteString("\n\n")

		content.WriteString(fmt.Sprintf("Years within $10K of threshold: %d\n", len(analysis.YearsWithWarnings)))
	} else {
		safeStyle := lipgloss.NewStyle().
			Foreground(tuistyles.ColorSuccess).
			Bold(true)

		content.WriteString(safeStyle.Render("✓ No IRMAA Concerns"))
		content.WriteString("\n\n")
		content.WriteString("MAGI remains comfortably below IRMAA thresholds\n")
	}

	// High risk years detail
	if len(analysis.HighRiskYears) > 0 {
		content.WriteString("\n")
		headerStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(tuistyles.ColorSecondary)

		content.WriteString(headerStyle.Render("High Risk Years:"))
		content.WriteString("\n\n")

		// Table header
		content.WriteString("Year   MAGI        Status    Tier    Annual Cost\n")
		content.WriteString("────── ─────────── ───────── ─────── ───────────\n")

		for _, yr := range analysis.HighRiskYears {
			var statusColor lipgloss.Color
			var statusIcon string

			switch yr.RiskStatus {
			case domain.IRMAARiskBreach:
				statusColor = tuistyles.ColorDanger
				statusIcon = "✗"
			case domain.IRMAARiskWarning:
				statusColor = tuistyles.ColorAccent
				statusIcon = "⚠"
			default:
				statusColor = tuistyles.ColorSuccess
				statusIcon = "✓"
			}

			statusStyle := lipgloss.NewStyle().Foreground(statusColor)

			content.WriteString(fmt.Sprintf("%-6d %-11s %s %-8s %-7s %s\n",
				yr.Year,
				formatCurrencyShort(yr.MAGI.InexactFloat64()),
				statusStyle.Render(statusIcon),
				string(yr.RiskStatus),
				yr.TierLevel,
				formatCurrencyShort(yr.AnnualCost.InexactFloat64()),
			))
		}
	}

	// Recommendations
	if len(analysis.Recommendations) > 0 {
		content.WriteString("\n")
		headerStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(tuistyles.ColorInfo)

		content.WriteString(headerStyle.Render("Recommendations:"))
		content.WriteString("\n\n")

	}

	return content.String()
}

// renderWithdrawalSequencing renders withdrawal sequencing information if available
func renderWithdrawalSequencing(summary *domain.ScenarioSummary) string {
	if summary.Projection == nil {
		return ""
	}

	// Find years with withdrawal breakdown
	var withdrawalYears []domain.AnnualCashFlow
	for _, year := range summary.Projection {
		if year.WithdrawalTaxable.GreaterThan(decimal.Zero) ||
			year.WithdrawalTraditional.GreaterThan(decimal.Zero) ||
			year.WithdrawalRoth.GreaterThan(decimal.Zero) {
			withdrawalYears = append(withdrawalYears, year)
		}
	}

	if len(withdrawalYears) == 0 {
		return ""
	}

	var content strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(tuistyles.ColorPrimary).
		Underline(true)

	content.WriteString(titleStyle.Render("Withdrawal Sequencing Analysis"))
	content.WriteString("\n\n")

	// Show first few years with withdrawal breakdown
	maxYears := 5
	if len(withdrawalYears) < maxYears {
		maxYears = len(withdrawalYears)
	}

	// Table header
	headerStyle := lipgloss.NewStyle().
		Foreground(tuistyles.ColorPrimary).
		Bold(true)

	header := fmt.Sprintf("%-6s  %-15s  %-15s  %-15s  %-15s",
		"Year", "Taxable", "Traditional", "Roth", "Total")
	content.WriteString(headerStyle.Render(header))
	content.WriteString("\n")
	content.WriteString(strings.Repeat("─", 70))
	content.WriteString("\n")

	// Show withdrawal breakdown for each year
	for i := 0; i < maxYears; i++ {
		year := withdrawalYears[i]
		taxable := formatCurrencyShort(year.WithdrawalTaxable.InexactFloat64())
		traditional := formatCurrencyShort(year.WithdrawalTraditional.InexactFloat64())
		roth := formatCurrencyShort(year.WithdrawalRoth.InexactFloat64())
		total := formatCurrencyShort(year.WithdrawalTaxable.Add(year.WithdrawalTraditional).Add(year.WithdrawalRoth).InexactFloat64())

		row := fmt.Sprintf("%-6d  %-15s  %-15s  %-15s  %-15s",
			year.Year, taxable, traditional, roth, total)
		content.WriteString(row)
		content.WriteString("\n")
	}

	if len(withdrawalYears) > maxYears {
		content.WriteString(fmt.Sprintf("\n... and %d more years with withdrawal sequencing\n", len(withdrawalYears)-maxYears))
	}

	// Add strategy insights
	content.WriteString("\n")
	insightStyle := lipgloss.NewStyle().
		Foreground(tuistyles.ColorAccent).
		Italic(true)

	// Analyze the pattern
	taxableTotal := decimal.Zero
	traditionalTotal := decimal.Zero
	rothTotal := decimal.Zero

	for _, year := range withdrawalYears {
		taxableTotal = taxableTotal.Add(year.WithdrawalTaxable)
		traditionalTotal = traditionalTotal.Add(year.WithdrawalTraditional)
		rothTotal = rothTotal.Add(year.WithdrawalRoth)
	}

	if taxableTotal.GreaterThan(decimal.Zero) && traditionalTotal.IsZero() && rothTotal.IsZero() {
		content.WriteString(insightStyle.Render("Strategy: Taxable-first (Standard)"))
	} else if rothTotal.GreaterThan(decimal.Zero) && traditionalTotal.IsZero() && taxableTotal.IsZero() {
		content.WriteString(insightStyle.Render("Strategy: Roth-first (Tax Efficient)"))
	} else if traditionalTotal.GreaterThan(decimal.Zero) && rothTotal.IsZero() && taxableTotal.IsZero() {
		content.WriteString(insightStyle.Render("Strategy: Traditional-first"))
	} else {
		content.WriteString(insightStyle.Render("Strategy: Mixed withdrawal sources"))
	}

	return content.String()
}
