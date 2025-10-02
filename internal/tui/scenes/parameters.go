package scenes

import (
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

// ParametersModel represents the parameter editing scene
type ParametersModel struct {
	scenario          *domain.GenericScenario
	participants      []string
	selectedParticipant int
	sliders           []*components.ParameterSlider
	focusedSlider     int
	width             int
	height            int
	modified          bool
}

// NewParametersModel creates a new parameters scene model
func NewParametersModel() *ParametersModel {
	return &ParametersModel{
		participants:      []string{},
		sliders:           []*components.ParameterSlider{},
		selectedParticipant: 0,
		focusedSlider:     0,
		modified:          false,
	}
}

// SetScenario updates the scenario being edited
func (m *ParametersModel) SetScenario(scenario *domain.GenericScenario) {
	if scenario == nil {
		return
	}

	m.scenario = scenario
	m.participants = []string{}
	m.selectedParticipant = 0

	// Extract participant names
	for name := range scenario.ParticipantScenarios {
		m.participants = append(m.participants, name)
	}

	// Build sliders for first participant
	if len(m.participants) > 0 {
		m.buildSliders()
	}
}

// buildSliders creates parameter sliders for the selected participant
func (m *ParametersModel) buildSliders() {
	if m.selectedParticipant >= len(m.participants) {
		return
	}

	participantName := m.participants[m.selectedParticipant]
	participant := m.scenario.ParticipantScenarios[participantName]

	m.sliders = []*components.ParameterSlider{}

	// SS Start Age slider
	ssAge := float64(participant.SSStartAge)
	if ssAge == 0 {
		ssAge = 62 // Default
	}
	ssSlider := components.NewParameterSlider("Social Security Claim Age", ssAge, 62, 70, 1).
		WithUnit(" years").
		WithFormat("%.0f").
		WithWidth(40).
		WithDescription("Age when Social Security benefits begin")
	m.sliders = append(m.sliders, ssSlider)

	// TSP Withdrawal Rate slider (if applicable)
	if participant.TSPWithdrawalRate != nil {
		rate := participant.TSPWithdrawalRate.Mul(decimal.NewFromInt(100)).InexactFloat64()
		rateSlider := components.NewParameterSlider("TSP Withdrawal Rate", rate, 1.0, 15.0, 0.1).
			WithUnit("%").
			WithFormat("%.1f").
			WithWidth(40).
			WithDescription("Annual TSP withdrawal percentage")
		m.sliders = append(m.sliders, rateSlider)
	}

	// Set focus on first slider
	if len(m.sliders) > 0 {
		m.focusedSlider = 0
		m.sliders[0].SetFocused(true)
	}
}

// GetScenario returns the current scenario being edited
func (m *ParametersModel) GetScenario() *domain.GenericScenario {
	return m.scenario
}

// SetSize updates the scene dimensions
func (m *ParametersModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update handles messages for the parameters scene
func (m *ParametersModel) Update(msg tea.Msg) (*ParametersModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}

	return m, nil
}

// handleKeyPress processes keyboard input
func (m *ParametersModel) handleKeyPress(msg tea.KeyMsg) (*ParametersModel, tea.Cmd) {
	if len(m.sliders) == 0 {
		return m, nil
	}

	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		m.moveFocusUp()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		m.moveFocusDown()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
		m.decrementValue()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
		m.incrementValue()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
		m.nextParticipant()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"))):
		m.prevParticipant()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		// Trigger calculation with modified parameters
		return m, m.calculateScenario()

	case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
		// Reset to original values
		m.buildSliders()
		m.modified = false
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+s"))):
		// Save modified scenario
		if m.modified && m.scenario != nil {
			return m, m.saveScenario()
		}
		return m, nil
	}

	return m, nil
}

// moveFocusUp moves focus to previous slider
func (m *ParametersModel) moveFocusUp() {
	if m.focusedSlider > 0 {
		m.sliders[m.focusedSlider].SetFocused(false)
		m.focusedSlider--
		m.sliders[m.focusedSlider].SetFocused(true)
	}
}

// moveFocusDown moves focus to next slider
func (m *ParametersModel) moveFocusDown() {
	if m.focusedSlider < len(m.sliders)-1 {
		m.sliders[m.focusedSlider].SetFocused(false)
		m.focusedSlider++
		m.sliders[m.focusedSlider].SetFocused(true)
	}
}

// incrementValue increases the focused slider's value
func (m *ParametersModel) incrementValue() {
	if m.focusedSlider < len(m.sliders) {
		m.sliders[m.focusedSlider].Increment()
		m.modified = true
		m.applyChanges()
	}
}

// decrementValue decreases the focused slider's value
func (m *ParametersModel) decrementValue() {
	if m.focusedSlider < len(m.sliders) {
		m.sliders[m.focusedSlider].Decrement()
		m.modified = true
		m.applyChanges()
	}
}

// applyChanges applies slider values back to the scenario
func (m *ParametersModel) applyChanges() {
	if m.selectedParticipant >= len(m.participants) || m.scenario == nil {
		return
	}

	participantName := m.participants[m.selectedParticipant]
	participant := m.scenario.ParticipantScenarios[participantName]

	// Apply each slider value
	for i, slider := range m.sliders {
		switch i {
		case 0: // SS Start Age
			participant.SSStartAge = int(slider.Value)
		case 1: // TSP Withdrawal Rate (if present)
			if participant.TSPWithdrawalRate != nil {
				rate := decimal.NewFromFloat(slider.Value / 100.0)
				participant.TSPWithdrawalRate = &rate
			}
		}
	}

	// Update the scenario
	m.scenario.ParticipantScenarios[participantName] = participant
}

// nextParticipant switches to the next participant
func (m *ParametersModel) nextParticipant() {
	if m.selectedParticipant < len(m.participants)-1 {
		m.selectedParticipant++
		m.buildSliders()
	}
}

// prevParticipant switches to the previous participant
func (m *ParametersModel) prevParticipant() {
	if m.selectedParticipant > 0 {
		m.selectedParticipant--
		m.buildSliders()
	}
}

// calculateScenario triggers a scenario calculation
func (m *ParametersModel) calculateScenario() tea.Cmd {
	if m.scenario == nil {
		return nil
	}

	return func() tea.Msg {
		return tuimsg.CalculationStartedMsg{
			ScenarioName: m.scenario.Name,
		}
	}
}

// saveScenario triggers a save operation for the modified scenario
func (m *ParametersModel) saveScenario() tea.Cmd {
	if m.scenario == nil {
		return nil
	}

	return func() tea.Msg {
		return tuimsg.SaveScenarioMsg{
			Scenario: m.scenario,
			Filename: "", // Will be filled in by main model with config path
		}
	}
}

// View renders the parameters scene
func (m *ParametersModel) View() string {
	if m.scenario == nil || len(m.participants) == 0 {
		return renderNoScenarioState()
	}

	// Build header with participant selector
	header := renderParticipantSelector(m.participants, m.selectedParticipant)

	// Build sliders section
	slidersView := renderSliders(m.sliders)

	// Build status section
	status := renderParameterStatus(m.modified)

	// Build help section
	help := renderParameterHelp()

	// Combine sections
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		slidersView,
		"",
		status,
		"",
		help,
	)

	return content
}

// renderNoScenarioState renders empty state
func renderNoScenarioState() string {
	return `No scenario selected.

Please select a scenario from the Scenarios screen (press 's').

Press ESC to return to home.`
}

// renderParticipantSelector renders the participant selection header
func renderParticipantSelector(participants []string, selected int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(tuistyles.ColorPrimary).
		MarginBottom(1)

	participantStyle := lipgloss.NewStyle().
		Foreground(tuistyles.ColorForeground).
		Padding(0, 1)

	selectedStyle := lipgloss.NewStyle().
		Foreground(tuistyles.ColorAccent).
		Bold(true).
		Padding(0, 1).
		Background(tuistyles.ColorBorder)

	title := titleStyle.Render("Edit Parameters")

	var tabs []string
	for i, name := range participants {
		if i == selected {
			tabs = append(tabs, selectedStyle.Render(name))
		} else {
			tabs = append(tabs, participantStyle.Render(name))
		}
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	hint := lipgloss.NewStyle().
		Foreground(tuistyles.ColorMuted).
		Italic(true).
		Render("Tab / Shift+Tab to switch participants")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		tabBar,
		hint,
	)
}

// renderSliders renders all parameter sliders
func renderSliders(sliders []*components.ParameterSlider) string {
	if len(sliders) == 0 {
		return "No adjustable parameters for this participant."
	}

	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tuistyles.ColorBorder).
		Padding(2, 4).
		Width(70)

	var rendered []string
	for _, slider := range sliders {
		rendered = append(rendered, slider.Render())
		rendered = append(rendered, "") // Spacer
	}

	content := strings.Join(rendered, "\n")
	return containerStyle.Render(content)
}

// renderParameterStatus renders modification status
func renderParameterStatus(modified bool) string {
	if !modified {
		return ""
	}

	statusStyle := lipgloss.NewStyle().
		Foreground(tuistyles.ColorInfo).
		Bold(true)

	return statusStyle.Render("⚠ Modified - Press Enter to calculate or 'r' to reset")
}

// renderParameterHelp renders keyboard shortcuts
func renderParameterHelp() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(tuistyles.ColorMuted)

	return helpStyle.Render("↑/↓ navigate • ←/→ adjust • Enter calculate • r reset • Ctrl+S save • Tab switch participant • ESC back")
}
