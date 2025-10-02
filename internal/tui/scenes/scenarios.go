package scenes

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/shopspring/decimal"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/rgehrsitz/rpgo/internal/tui/components"
	"github.com/rgehrsitz/rpgo/internal/tui/tuimsg"
	"github.com/rgehrsitz/rpgo/internal/tui/tuistyles"
)

// ScenariosModel represents the scenarios browsing scene
type ScenariosModel struct {
	scenarios     []domain.GenericScenario
	selectedIndex int
	cards         []*components.ScenarioCard
	width         int
	height        int
}

// NewScenariosModel creates a new scenarios scene model
func NewScenariosModel() *ScenariosModel {
	return &ScenariosModel{
		scenarios:     []domain.GenericScenario{},
		selectedIndex: 0,
		cards:         []*components.ScenarioCard{},
	}
}

// SetScenarios updates the scenarios list
func (m *ScenariosModel) SetScenarios(scenarios []domain.GenericScenario) {
	m.scenarios = scenarios
	m.cards = []*components.ScenarioCard{}

	// Build scenario cards
	for _, scenario := range scenarios {
		// Create card for this scenario
		card := components.NewScenarioCard(scenario.Name).
			WithWidth(50)

		// Add participant info if available
		if len(scenario.ParticipantScenarios) > 0 {
			// Get first participant as primary
			for participantName := range scenario.ParticipantScenarios {
				card.WithParticipant(participantName)
				break
			}
		}

		// Add highlights based on scenario data
		if len(scenario.ParticipantScenarios) > 0 {
			card.AddHighlight(formatParticipantCount(len(scenario.ParticipantScenarios)))
		}

		m.cards = append(m.cards, card)
	}

	// Reset selection if out of bounds
	if m.selectedIndex >= len(m.scenarios) {
		m.selectedIndex = 0
	}
}

// SetSize updates the scene dimensions
func (m *ScenariosModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SelectedScenario returns the currently selected scenario name
func (m *ScenariosModel) SelectedScenario() string {
	if m.selectedIndex >= 0 && m.selectedIndex < len(m.scenarios) {
		return m.scenarios[m.selectedIndex].Name
	}
	return ""
}

// Update handles messages for the scenarios scene
func (m *ScenariosModel) Update(msg tea.Msg) (*ScenariosModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}

	return m, nil
}

// handleKeyPress processes keyboard input
func (m *ScenariosModel) handleKeyPress(msg tea.KeyMsg) (*ScenariosModel, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		m.moveUp()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		m.moveDown()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		// Select scenario and trigger calculation
		return m, m.selectScenario()

	case key.Matches(msg, key.NewBinding(key.WithKeys("g"))):
		// Go to top
		m.selectedIndex = 0
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("G"))):
		// Go to bottom
		m.selectedIndex = len(m.scenarios) - 1
		if m.selectedIndex < 0 {
			m.selectedIndex = 0
		}
		return m, nil
	}

	return m, nil
}

// moveUp moves selection up
func (m *ScenariosModel) moveUp() {
	if m.selectedIndex > 0 {
		m.selectedIndex--
	}
}

// moveDown moves selection down
func (m *ScenariosModel) moveDown() {
	if m.selectedIndex < len(m.scenarios)-1 {
		m.selectedIndex++
	}
}

// selectScenario returns a command to select the current scenario
func (m *ScenariosModel) selectScenario() tea.Cmd {
	scenarioName := m.SelectedScenario()
	if scenarioName == "" {
		return nil
	}

	return func() tea.Msg {
		return tuimsg.ScenarioSelectedMsg{ScenarioName: scenarioName}
	}
}

// View renders the scenarios scene
func (m *ScenariosModel) View() string {
	if len(m.scenarios) == 0 {
		return renderEmptyState()
	}

	// Update card selection states
	for i, card := range m.cards {
		card.SetSelected(i == m.selectedIndex)
	}

	// Split view: list on left, details on right
	leftPane := renderScenarioList(m.cards, m.selectedIndex)
	rightPane := renderScenarioDetails(m.scenarios[m.selectedIndex])

	// Join horizontally
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPane,
		"  ", // Spacer
		rightPane,
	)

	// Add help text
	content += "\n\n"
	content += renderScenariosHelp()

	return content
}

// renderEmptyState renders the empty state when no scenarios are loaded
func renderEmptyState() string {
	return `No scenarios available.

Please load a configuration file with scenarios defined.

Press ESC to return to home.`
}

// renderScenariosHelp renders keyboard shortcuts help
func renderScenariosHelp() string {
	return "↑/k up • ↓/j down • Enter select • g top • G bottom • ESC back"
}

// formatParticipantCount formats the participant count
func formatParticipantCount(count int) string {
	if count == 1 {
		return "1 participant"
	}
	return formatInt(count) + " participants"
}

// formatInt formats an integer with proper grammar
func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	return fmt.Sprintf("%d", n)
}

// renderScenarioList renders the scenario list pane
func renderScenarioList(cards []*components.ScenarioCard, selectedIndex int) string {
	listStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tuistyles.ColorBorder).
		Padding(1, 2).
		Width(40)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(tuistyles.ColorPrimary).
		MarginBottom(1)

	title := titleStyle.Render("Scenarios")
	list := components.ScenarioListCompact(cards, selectedIndex)

	return listStyle.Render(title + "\n" + list)
}

// renderScenarioDetails renders detailed information about a scenario
func renderScenarioDetails(scenario domain.GenericScenario) string {
	detailStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tuistyles.ColorPrimary).
		Padding(1, 2).
		Width(60)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(tuistyles.ColorPrimary)

	labelStyle := lipgloss.NewStyle().
		Foreground(tuistyles.ColorMuted).
		Bold(true)

	valueStyle := lipgloss.NewStyle().
		Foreground(tuistyles.ColorForeground)

	var content strings.Builder

	// Title
	content.WriteString(titleStyle.Render(scenario.Name))
	content.WriteString("\n\n")

	// Participants section
	content.WriteString(labelStyle.Render("Participants:"))
	content.WriteString("\n")

	if len(scenario.ParticipantScenarios) == 0 {
		content.WriteString(valueStyle.Render("  No participants defined"))
		content.WriteString("\n")
	} else {
		for participantName, participant := range scenario.ParticipantScenarios {
			content.WriteString(valueStyle.Render(fmt.Sprintf("  • %s", participantName)))
			content.WriteString("\n")

			// Show key details
			if participant.RetirementDate != nil {
				content.WriteString(valueStyle.Render(fmt.Sprintf("    Retirement: %s", participant.RetirementDate.Format("2006-01-02"))))
				content.WriteString("\n")
			}

			if participant.SSStartAge > 0 {
				content.WriteString(valueStyle.Render(fmt.Sprintf("    SS Claim Age: %d", participant.SSStartAge)))
				content.WriteString("\n")
			}

			if participant.TSPWithdrawalStrategy != "" {
				content.WriteString(valueStyle.Render(fmt.Sprintf("    TSP Strategy: %s", participant.TSPWithdrawalStrategy)))
				content.WriteString("\n")
			}

			if participant.TSPWithdrawalRate != nil {
				rate := participant.TSPWithdrawalRate.Mul(decimal.NewFromInt(100))
				content.WriteString(valueStyle.Render(fmt.Sprintf("    TSP Rate: %.2f%%", rate.InexactFloat64())))
				content.WriteString("\n")
			}
		}
	}

	// Mortality assumptions if present
	if scenario.Mortality != nil {
		content.WriteString("\n")
		content.WriteString(labelStyle.Render("Mortality:"))
		content.WriteString("\n")

		if len(scenario.Mortality.Participants) > 0 {
			content.WriteString(valueStyle.Render(fmt.Sprintf("  %d participant-specific mortality specs", len(scenario.Mortality.Participants))))
			content.WriteString("\n")
		} else {
			content.WriteString(valueStyle.Render("  Using default mortality assumptions"))
			content.WriteString("\n")
		}
	}

	// Press Enter hint
	content.WriteString("\n")
	hintStyle := lipgloss.NewStyle().
		Foreground(tuistyles.ColorInfo).
		Italic(true)
	content.WriteString(hintStyle.Render("Press Enter to calculate this scenario"))

	return detailStyle.Render(content.String())
}
