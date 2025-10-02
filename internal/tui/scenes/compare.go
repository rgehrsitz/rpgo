package scenes

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shopspring/decimal"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/rgehrsitz/rpgo/internal/tui/tuimsg"
	"github.com/rgehrsitz/rpgo/internal/tui/tuistyles"
)

// CompareModel represents the scenario comparison scene
type CompareModel struct {
	scenarios         []domain.GenericScenario
	selectedScenarios map[int]bool // Track which scenarios are selected for comparison
	cursorIndex       int
	results           map[string]*domain.ScenarioSummary
	comparing         bool
	width             int
	height            int
}

// NewCompareModel creates a new compare scene model
func NewCompareModel() *CompareModel {
	return &CompareModel{
		scenarios:         []domain.GenericScenario{},
		selectedScenarios: make(map[int]bool),
		results:           make(map[string]*domain.ScenarioSummary),
		cursorIndex:       0,
		comparing:         false,
	}
}

// SetScenarios updates the scenarios list
func (m *CompareModel) SetScenarios(scenarios []domain.GenericScenario) {
	m.scenarios = scenarios
	m.selectedScenarios = make(map[int]bool)
	m.cursorIndex = 0
}

// SetResults stores comparison results
func (m *CompareModel) SetResults(results map[string]*domain.ScenarioSummary) {
	m.results = results
	m.comparing = false
}

// SetSize updates the model dimensions
func (m *CompareModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update handles messages for the compare scene
func (m *CompareModel) Update(msg tea.Msg) (*CompareModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if m.cursorIndex > 0 {
				m.cursorIndex--
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if m.cursorIndex < len(m.scenarios)-1 {
				m.cursorIndex++
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys(" ", "x"))):
			// Toggle selection
			m.selectedScenarios[m.cursorIndex] = !m.selectedScenarios[m.cursorIndex]
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			// Start comparison
			if len(m.getSelectedScenarios()) < 2 {
				return m, nil // Need at least 2 scenarios
			}
			m.comparing = true
			return m, m.startComparisonCmd()

		case key.Matches(msg, key.NewBinding(key.WithKeys("c"))):
			// Clear selections
			m.selectedScenarios = make(map[int]bool)
			m.results = make(map[string]*domain.ScenarioSummary)
			return m, nil
		}
	}

	return m, nil
}

// getSelectedScenarios returns the list of selected scenario names in index order
func (m *CompareModel) getSelectedScenarios() []string {
	var selected []string
	// Iterate in order by index to maintain consistent ordering
	for idx := 0; idx < len(m.scenarios); idx++ {
		if m.selectedScenarios[idx] {
			selected = append(selected, m.scenarios[idx].Name)
		}
	}
	return selected
}

// startComparisonCmd creates a command to start scenario comparison
func (m *CompareModel) startComparisonCmd() tea.Cmd {
	return func() tea.Msg {
		return tuimsg.ComparisonStartedMsg{
			ScenarioNames: m.getSelectedScenarios(),
		}
	}
}

// View renders the compare scene
func (m *CompareModel) View() string {
	if m.comparing {
		return m.renderLoading()
	}

	if len(m.results) > 0 {
		return m.renderComparison()
	}

	return m.renderSelection()
}

// renderSelection shows scenario selection interface
func (m *CompareModel) renderSelection() string {
	var content strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(tuistyles.ColorPrimary)
	title := titleStyle.Render("Select Scenarios to Compare")
	content.WriteString(title)
	content.WriteString("\n\n")

	if len(m.scenarios) == 0 {
		content.WriteString(tuistyles.ErrorStyle.Render("No scenarios available"))
		return tuistyles.BorderStyle.Render(content.String())
	}

	// Instructions
	subtleStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorMuted)
	instructions := subtleStyle.Render(
		"Use ↑/↓ to navigate • Space/x to select • Enter to compare • c to clear",
	)
	content.WriteString(instructions)
	content.WriteString("\n\n")

	// Scenario list with checkboxes
	for idx, scenario := range m.scenarios {
		var line strings.Builder

		// Cursor
		cursorStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorPrimary)
		if idx == m.cursorIndex {
			line.WriteString(cursorStyle.Render("❯ "))
		} else {
			line.WriteString("  ")
		}

		// Checkbox
		highlightStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorPrimary).Bold(true)
		if m.selectedScenarios[idx] {
			line.WriteString(highlightStyle.Render("[✓] "))
		} else {
			line.WriteString(subtleStyle.Render("[ ] "))
		}

		// Scenario name
		scenarioName := scenario.Name
		if idx == m.cursorIndex {
			scenarioName = highlightStyle.Render(scenarioName)
		}
		line.WriteString(scenarioName)

		// Show participant count
		participantCount := len(scenario.ParticipantScenarios)
		participantInfo := subtleStyle.Render(
			fmt.Sprintf(" (%d participant%s)", participantCount, plural(participantCount)),
		)
		line.WriteString(participantInfo)

		content.WriteString(line.String())
		content.WriteString("\n")
	}

	// Selection summary
	selectedCount := len(m.getSelectedScenarios())
	content.WriteString("\n")
	warningStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorAccent)
	successStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorSuccess)
	if selectedCount == 0 {
		content.WriteString(subtleStyle.Render("Select at least 2 scenarios to compare"))
	} else if selectedCount == 1 {
		content.WriteString(warningStyle.Render(
			fmt.Sprintf("Selected: %d scenario (need at least 2)", selectedCount),
		))
	} else {
		content.WriteString(successStyle.Render(
			fmt.Sprintf("Selected: %d scenarios • Press Enter to compare", selectedCount),
		))
	}

	return tuistyles.BorderStyle.Render(content.String())
}

// renderLoading shows loading state during comparison
func (m *CompareModel) renderLoading() string {
	var content strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(tuistyles.ColorPrimary)
	title := titleStyle.Render("Calculating Scenario Comparisons...")
	content.WriteString(title)
	content.WriteString("\n\n")

	highlightStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorPrimary).Bold(true)
	content.WriteString("⠋ Running calculations for ")
	content.WriteString(highlightStyle.Render(fmt.Sprintf("%d scenarios", len(m.getSelectedScenarios()))))
	content.WriteString("...\n\n")

	subtleStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorMuted)
	content.WriteString(subtleStyle.Render("This may take a few moments..."))

	return tuistyles.BorderStyle.Render(content.String())
}

// renderComparison shows the comparison results
func (m *CompareModel) renderComparison() string {
	var content strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(tuistyles.ColorPrimary)
	title := titleStyle.Render("Scenario Comparison Results")
	content.WriteString(title)
	content.WriteString("\n\n")

	subtleStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorMuted)
	if len(m.results) == 0 {
		content.WriteString(subtleStyle.Render("No comparison results available"))
		return tuistyles.BorderStyle.Render(content.String())
	}

	// Debug: Show what scenarios we're looking for vs what we have
	selectedNames := m.getSelectedScenarios()
	content.WriteString(subtleStyle.Render(fmt.Sprintf("Selected (%d): ", len(selectedNames))))
	for i, name := range selectedNames {
		content.WriteString(subtleStyle.Render(fmt.Sprintf("[%d:%s] ", i, name)))
	}
	content.WriteString("\n")
	content.WriteString(subtleStyle.Render(fmt.Sprintf("Results (%d): ", len(m.results))))
	for key := range m.results {
		content.WriteString(subtleStyle.Render(fmt.Sprintf("[%s] ", key)))
	}
	content.WriteString("\n\n")

	// Build comparison table
	content.WriteString(m.renderComparisonTable())
	content.WriteString("\n\n")

	// Help text
	help := subtleStyle.Render("c to start new comparison • ESC to go back")
	content.WriteString(help)

	return tuistyles.BorderStyle.Render(content.String())
}

// renderComparisonTable creates a side-by-side comparison table
func (m *CompareModel) renderComparisonTable() string {
	var table strings.Builder

	// Get scenario names in order
	scenarioNames := m.getSelectedScenarios()
	if len(scenarioNames) == 0 {
		return ""
	}

	// Table header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(tuistyles.ColorPrimary)

	// Metric column
	metricWidth := 25
	table.WriteString(headerStyle.Render(padRight("Metric", metricWidth)))
	table.WriteString(" ")

	// Scenario columns
	colWidth := 18
	for _, name := range scenarioNames {
		shortName := truncate(name, colWidth)
		table.WriteString(headerStyle.Render(padRight(shortName, colWidth)))
		table.WriteString(" ")
	}
	table.WriteString("\n")

	// Separator
	totalWidth := metricWidth + (len(scenarioNames) * (colWidth + 1))
	table.WriteString(strings.Repeat("─", totalWidth))
	table.WriteString("\n")

	// Metrics rows
	metrics := []struct {
		label    string
		getValue func(*domain.ScenarioSummary) string
	}{
		{
			label: "First Year Income",
			getValue: func(s *domain.ScenarioSummary) string {
				return formatCompactCurrency(s.FirstYearNetIncome.InexactFloat64())
			},
		},
		{
			label: "TSP Longevity",
			getValue: func(s *domain.ScenarioSummary) string {
				return fmt.Sprintf("%d years", s.TSPLongevity)
			},
		},
		{
			label: "Final TSP Balance",
			getValue: func(s *domain.ScenarioSummary) string {
				return formatCompactCurrency(s.FinalTSPBalance.InexactFloat64())
			},
		},
		{
			label: "Lifetime Income",
			getValue: func(s *domain.ScenarioSummary) string {
				return formatCompactCurrency(s.TotalLifetimeIncome.InexactFloat64())
			},
		},
	}

	subtleStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorMuted)

	// Debug: Check what we have
	if len(m.results) == 0 {
		table.WriteString("No results available\n")
		return table.String()
	}

	for _, metric := range metrics {
		// Metric label
		table.WriteString(subtleStyle.Render(padRight(metric.label, metricWidth)))
		table.WriteString(" ")

		// Find best value for highlighting
		var bestValue decimal.Decimal
		bestValueSet := false
		for _, name := range scenarioNames {
			if result, ok := m.results[name]; ok {
				switch metric.label {
				case "First Year Income", "TSP Longevity", "Lifetime Income":
					value := getMetricValue(result, metric.label)
					if !bestValueSet || value.GreaterThan(bestValue) {
						bestValue = value
						bestValueSet = true
					}
				case "Final TSP Balance":
					// For final balance, higher is better if positive
					value := getMetricValue(result, metric.label)
					if !bestValueSet || value.GreaterThan(bestValue) {
						bestValue = value
						bestValueSet = true
					}
				}
			}
		}

		// Values for each scenario
		for i, name := range scenarioNames {
			if result, ok := m.results[name]; ok {
				valueStr := metric.getValue(result)
				value := getMetricValue(result, metric.label)

				// Highlight best value
				successStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorSuccess)
				if bestValueSet && value.Equal(bestValue) {
					valueStr = successStyle.Render(valueStr + " ★")
				}

				// Debug: Add scenario index to see which column
				debugStr := fmt.Sprintf("[%d]%s", i, valueStr)
				table.WriteString(padRight(debugStr, colWidth))
			} else {
				// Debug: show that we didn't find this scenario
				debugStr := fmt.Sprintf("[%d:miss]", i)
				table.WriteString(padRight(debugStr, colWidth))
			}
			table.WriteString(" ")
		}
		table.WriteString("\n")
	}

	return table.String()
}

// Helper functions

func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func padRight(s string, width int) string {
	// Use lipgloss.Width to account for ANSI escape codes
	currentWidth := lipgloss.Width(s)
	if currentWidth >= width {
		return s
	}
	return s + strings.Repeat(" ", width-currentWidth)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func formatCompactCurrency(amount float64) string {
	if amount >= 1000000 {
		return fmt.Sprintf("$%.2fM", amount/1000000)
	} else if amount >= 1000 {
		return fmt.Sprintf("$%.1fK", amount/1000)
	}
	return fmt.Sprintf("$%.0f", amount)
}

func getMetricValue(summary *domain.ScenarioSummary, metricLabel string) decimal.Decimal {
	switch metricLabel {
	case "First Year Income":
		return summary.FirstYearNetIncome
	case "TSP Longevity":
		return decimal.NewFromInt(int64(summary.TSPLongevity))
	case "Final TSP Balance":
		return summary.FinalTSPBalance
	case "Lifetime Income":
		return summary.TotalLifetimeIncome
	default:
		return decimal.Zero
	}
}
