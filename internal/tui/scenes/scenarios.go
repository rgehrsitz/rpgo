package scenes

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/rgehrsitz/rpgo/internal/tui/tuimsg"
	"github.com/rgehrsitz/rpgo/internal/tui/components"
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

	// Render as compact list for now
	// In future, could switch to grid view for many scenarios
	content := components.ScenarioListCompact(m.cards, m.selectedIndex)

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
