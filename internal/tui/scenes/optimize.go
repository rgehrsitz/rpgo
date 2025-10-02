package scenes

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shopspring/decimal"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/rgehrsitz/rpgo/internal/tui/tuimsg"
	"github.com/rgehrsitz/rpgo/internal/tui/tuistyles"
)

// OptimizeModel represents the optimization/break-even solver scene
type OptimizeModel struct {
	scenarios         []domain.GenericScenario
	selectedScenario  int
	mode              OptimizeMode
	targetIncomeInput textinput.Model
	optimizing        bool
	result            *OptimizeResult
	width             int
	height            int
}

// OptimizeMode represents the type of optimization being performed
type OptimizeMode int

const (
	ModeSelectScenario OptimizeMode = iota
	ModeSetTarget
	ModeShowResults
)

// OptimizeResult holds the results of an optimization
type OptimizeResult struct {
	OptimalRate    decimal.Decimal
	TargetIncome   decimal.Decimal
	ActualIncome   decimal.Decimal
	Year           int
	CashFlow       *domain.AnnualCashFlow
	ScenarioName   string
}

// NewOptimizeModel creates a new optimize scene model
func NewOptimizeModel() *OptimizeModel {
	ti := textinput.New()
	ti.Placeholder = "e.g., 150000"
	ti.CharLimit = 10
	ti.Width = 20

	return &OptimizeModel{
		scenarios:         []domain.GenericScenario{},
		selectedScenario:  0,
		mode:              ModeSelectScenario,
		targetIncomeInput: ti,
		optimizing:        false,
	}
}

// SetScenarios updates the scenarios list
func (m *OptimizeModel) SetScenarios(scenarios []domain.GenericScenario) {
	m.scenarios = scenarios
	m.selectedScenario = 0
	m.mode = ModeSelectScenario
}

// SetSize updates the model dimensions
func (m *OptimizeModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// GetSelectedScenario returns the currently selected scenario
func (m *OptimizeModel) GetSelectedScenario() *domain.GenericScenario {
	if m.selectedScenario >= 0 && m.selectedScenario < len(m.scenarios) {
		return &m.scenarios[m.selectedScenario]
	}
	return nil
}

// Update handles messages for the optimize scene
func (m *OptimizeModel) Update(msg tea.Msg) (*OptimizeModel, tea.Cmd) {
	switch m.mode {
	case ModeSelectScenario:
		return m.updateScenarioSelection(msg)
	case ModeSetTarget:
		return m.updateTargetInput(msg)
	case ModeShowResults:
		return m.updateResults(msg)
	}
	return m, nil
}

// updateScenarioSelection handles scenario selection
func (m *OptimizeModel) updateScenarioSelection(msg tea.Msg) (*OptimizeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if m.selectedScenario > 0 {
				m.selectedScenario--
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if m.selectedScenario < len(m.scenarios)-1 {
				m.selectedScenario++
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			// Move to target input mode
			m.mode = ModeSetTarget
			m.targetIncomeInput.Focus()
			return m, textinput.Blink
		}
	}
	return m, nil
}

// updateTargetInput handles target income input
func (m *OptimizeModel) updateTargetInput(msg tea.Msg) (*OptimizeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			// Validate and start optimization
			if target, err := strconv.ParseFloat(m.targetIncomeInput.Value(), 64); err == nil && target > 0 {
				m.optimizing = true
				return m, m.startOptimizationCmd(decimal.NewFromFloat(target))
			}
			return m, nil

		case tea.KeyEsc:
			// Go back to scenario selection
			m.mode = ModeSelectScenario
			m.targetIncomeInput.Blur()
			return m, nil
		}
	}

	// Update text input
	var cmd tea.Cmd
	m.targetIncomeInput, cmd = m.targetIncomeInput.Update(msg)
	return m, cmd
}

// updateResults handles results display
func (m *OptimizeModel) updateResults(msg tea.Msg) (*OptimizeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
			// Start new optimization
			m.mode = ModeSelectScenario
			m.result = nil
			m.targetIncomeInput.SetValue("")
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			// Go back to scenario selection
			m.mode = ModeSelectScenario
			m.result = nil
			return m, nil
		}
	}
	return m, nil
}

// startOptimizationCmd creates a command to start optimization
func (m *OptimizeModel) startOptimizationCmd(targetIncome decimal.Decimal) tea.Cmd {
	return func() tea.Msg {
		scenario := m.GetSelectedScenario()
		if scenario == nil {
			return tuimsg.OptimizationCompleteMsg{
				Results: nil,
				Err:     fmt.Errorf("no scenario selected"),
			}
		}

		return tuimsg.OptimizationStartedMsg{
			ScenarioName: scenario.Name,
			TargetIncome: targetIncome,
		}
	}
}

// SetResult updates the optimization result
func (m *OptimizeModel) SetResult(result *OptimizeResult) {
	m.result = result
	m.optimizing = false
	m.mode = ModeShowResults
}

// View renders the optimize scene
func (m *OptimizeModel) View() string {
	if m.optimizing {
		return m.renderOptimizing()
	}

	switch m.mode {
	case ModeSelectScenario:
		return m.renderScenarioSelection()
	case ModeSetTarget:
		return m.renderTargetInput()
	case ModeShowResults:
		return m.renderResults()
	}

	return ""
}

// renderScenarioSelection shows scenario selection interface
func (m *OptimizeModel) renderScenarioSelection() string {
	var content strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(tuistyles.ColorPrimary)
	title := titleStyle.Render("Break-Even TSP Withdrawal Rate Optimizer")
	content.WriteString(title)
	content.WriteString("\n\n")

	subtleStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorMuted)
	description := subtleStyle.Render(
		"Find the optimal TSP withdrawal rate to meet your target retirement income.",
	)
	content.WriteString(description)
	content.WriteString("\n\n")

	if len(m.scenarios) == 0 {
		content.WriteString(tuistyles.ErrorStyle.Render("No scenarios available"))
		return tuistyles.BorderStyle.Render(content.String())
	}

	instructions := subtleStyle.Render("Use ↑/↓ to navigate • Enter to select")
	content.WriteString(instructions)
	content.WriteString("\n\n")

	// Scenario list
	for idx, scenario := range m.scenarios {
		var line strings.Builder

		// Cursor
		cursorStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorPrimary)
		if idx == m.selectedScenario {
			line.WriteString(cursorStyle.Render("❯ "))
		} else {
			line.WriteString("  ")
		}

		// Scenario name
		scenarioName := scenario.Name
		if idx == m.selectedScenario {
			highlightStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorPrimary).Bold(true)
			scenarioName = highlightStyle.Render(scenarioName)
		}
		line.WriteString(scenarioName)

		// Show participant info
		participantCount := len(scenario.ParticipantScenarios)
		participantInfo := subtleStyle.Render(
			fmt.Sprintf(" (%d participant%s)", participantCount, pluralize(participantCount)),
		)
		line.WriteString(participantInfo)

		content.WriteString(line.String())
		content.WriteString("\n")
	}

	return tuistyles.BorderStyle.Render(content.String())
}

// renderTargetInput shows target income input interface
func (m *OptimizeModel) renderTargetInput() string {
	var content strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(tuistyles.ColorPrimary)
	title := titleStyle.Render("Set Target Annual Net Income")
	content.WriteString(title)
	content.WriteString("\n\n")

	subtleStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorMuted)
	scenario := m.GetSelectedScenario()
	if scenario != nil {
		content.WriteString(subtleStyle.Render("Scenario: "))
		content.WriteString(scenario.Name)
		content.WriteString("\n\n")
	}

	content.WriteString(subtleStyle.Render("Enter your target annual net income (after taxes):"))
	content.WriteString("\n\n")

	// Input field
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tuistyles.ColorPrimary).
		Padding(0, 1)

	content.WriteString(inputStyle.Render("$ " + m.targetIncomeInput.View()))
	content.WriteString("\n\n")

	help := subtleStyle.Render("Enter to optimize • ESC to go back")
	content.WriteString(help)

	return tuistyles.BorderStyle.Render(content.String())
}

// renderOptimizing shows optimization progress
func (m *OptimizeModel) renderOptimizing() string {
	var content strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(tuistyles.ColorPrimary)
	title := titleStyle.Render("Optimizing TSP Withdrawal Rate...")
	content.WriteString(title)
	content.WriteString("\n\n")

	content.WriteString("⠋ Running break-even analysis...")
	content.WriteString("\n\n")

	subtleStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorMuted)
	content.WriteString(subtleStyle.Render("This may take a few moments..."))

	return tuistyles.BorderStyle.Render(content.String())
}

// renderResults shows optimization results
func (m *OptimizeModel) renderResults() string {
	var content strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(tuistyles.ColorPrimary)
	title := titleStyle.Render("Optimization Results")
	content.WriteString(title)
	content.WriteString("\n\n")

	if m.result == nil {
		subtleStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorMuted)
		content.WriteString(subtleStyle.Render("No results available"))
		return tuistyles.BorderStyle.Render(content.String())
	}

	// Result summary
	successStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorSuccess).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorMuted)

	content.WriteString(labelStyle.Render("Scenario: "))
	content.WriteString(m.result.ScenarioName)
	content.WriteString("\n\n")

	content.WriteString(labelStyle.Render("Target Annual Net Income: "))
	content.WriteString(formatCurrency(m.result.TargetIncome.InexactFloat64()))
	content.WriteString("\n")

	content.WriteString(labelStyle.Render("Optimal TSP Withdrawal Rate: "))
	ratePercent := m.result.OptimalRate.Mul(decimal.NewFromInt(100)).InexactFloat64()
	content.WriteString(successStyle.Render(fmt.Sprintf("%.2f%%", ratePercent)))
	content.WriteString("\n\n")

	// Achieved income
	content.WriteString(labelStyle.Render("Projected Net Income: "))
	if m.result.CashFlow != nil {
		actualIncome := m.result.CashFlow.NetIncome.InexactFloat64()
		content.WriteString(formatCurrency(actualIncome))

		// Show difference from target
		diff := m.result.CashFlow.NetIncome.Sub(m.result.TargetIncome).InexactFloat64()
		diffStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorSuccess)
		if diff < 0 {
			diffStyle = lipgloss.NewStyle().Foreground(tuistyles.ColorDanger)
		}
		content.WriteString(" ")
		content.WriteString(diffStyle.Render(fmt.Sprintf("(%s)", formatSignedCurrency(diff))))
	}
	content.WriteString("\n\n")

	// Additional details
	if m.result.CashFlow != nil {
		content.WriteString(titleStyle.Render("Income Breakdown"))
		content.WriteString("\n\n")

		content.WriteString(labelStyle.Render("Total Gross Income: "))
		content.WriteString(formatCurrency(m.result.CashFlow.TotalGrossIncome.InexactFloat64()))
		content.WriteString("\n")

		content.WriteString(labelStyle.Render("Federal Tax: "))
		content.WriteString(formatCurrency(m.result.CashFlow.FederalTax.InexactFloat64()))
		content.WriteString("\n")

		if !m.result.CashFlow.StateTax.IsZero() {
			content.WriteString(labelStyle.Render("State Tax: "))
			content.WriteString(formatCurrency(m.result.CashFlow.StateTax.InexactFloat64()))
			content.WriteString("\n")
		}

		// Healthcare costs (FEHB + Medicare)
		healthcareCosts := m.result.CashFlow.FEHBPremium.Add(m.result.CashFlow.MedicarePremium)
		if !healthcareCosts.IsZero() {
			content.WriteString(labelStyle.Render("Healthcare Costs: "))
			content.WriteString(formatCurrency(healthcareCosts.InexactFloat64()))
			content.WriteString("\n")
		}
	}

	content.WriteString("\n")
	subtleStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorMuted)
	help := subtleStyle.Render("n for new optimization • ESC to go back")
	content.WriteString(help)

	return tuistyles.BorderStyle.Render(content.String())
}

// Helper functions

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func formatSignedCurrency(amount float64) string {
	sign := ""
	if amount > 0 {
		sign = "+"
	}
	return fmt.Sprintf("%s%s", sign, formatCurrency(amount))
}
