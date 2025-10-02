package scenes

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/rgehrsitz/rpgo/internal/tui/tuistyles"
)

// HomeModel represents the home dashboard scene
type HomeModel struct {
	config *domain.Configuration
	width  int
	height int
}

// NewHomeModel creates a new home scene model
func NewHomeModel() *HomeModel {
	return &HomeModel{}
}

// SetConfig updates the configuration
func (m *HomeModel) SetConfig(config *domain.Configuration) {
	m.config = config
}

// SetSize updates the model dimensions
func (m *HomeModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update handles messages for the home scene
func (m *HomeModel) Update(msg tea.Msg) (*HomeModel, tea.Cmd) {
	// Home scene is mostly passive - navigation handled by parent
	return m, nil
}

// View renders the home dashboard
func (m *HomeModel) View() string {
	if m.config == nil {
		return m.renderLoading()
	}

	var content strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(tuistyles.ColorPrimary).
		MarginBottom(1)
	content.WriteString(titleStyle.Render("RPGO - FERS Retirement Planning Calculator"))
	content.WriteString("\n\n")

	// Configuration Overview
	content.WriteString(m.renderConfigOverview())
	content.WriteString("\n\n")

	// Scenarios Overview
	content.WriteString(m.renderScenariosOverview())
	content.WriteString("\n\n")

	// Quick Actions
	content.WriteString(m.renderQuickActions())
	content.WriteString("\n\n")

	// Help
	content.WriteString(m.renderHelp())

	return tuistyles.BorderStyle.Render(content.String())
}

// renderLoading shows loading state
func (m *HomeModel) renderLoading() string {
	var content strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(tuistyles.ColorPrimary)
	content.WriteString(titleStyle.Render("RPGO - FERS Retirement Planning Calculator"))
	content.WriteString("\n\n")

	subtleStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorMuted)
	content.WriteString(subtleStyle.Render("Loading configuration..."))

	return tuistyles.BorderStyle.Render(content.String())
}

// renderConfigOverview shows household and configuration summary
func (m *HomeModel) renderConfigOverview() string {
	var content strings.Builder

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(tuistyles.ColorSecondary).
		MarginBottom(1)
	content.WriteString(sectionStyle.Render("📋 Configuration Overview"))
	content.WriteString("\n")

	labelStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorMuted)
	valueStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorForeground)

	// Household info
	if m.config.Household != nil {
		content.WriteString(labelStyle.Render("  Filing Status: "))
		content.WriteString(valueStyle.Render(formatFilingStatus(m.config.Household.FilingStatus)))
		content.WriteString("\n")

		// Participants
		participantCount := len(m.config.Household.Participants)
		content.WriteString(labelStyle.Render("  Participants: "))
		content.WriteString(valueStyle.Render(fmt.Sprintf("%d", participantCount)))
		content.WriteString("\n")

		// List participants
		for _, participant := range m.config.Household.Participants {
			content.WriteString(labelStyle.Render("    • "))
			content.WriteString(valueStyle.Render(participant.Name))
			if !participant.BirthDate.IsZero() {
				age := calculateAge(participant.BirthDate)
				content.WriteString(labelStyle.Render(fmt.Sprintf(" (age %d)", age)))
			}
			content.WriteString("\n")
		}
	}

	// Scenarios count
	scenarioCount := len(m.config.Scenarios)
	content.WriteString(labelStyle.Render("  Scenarios: "))
	content.WriteString(valueStyle.Render(fmt.Sprintf("%d configured", scenarioCount)))
	content.WriteString("\n")

	return content.String()
}

// renderScenariosOverview shows quick scenario summary
func (m *HomeModel) renderScenariosOverview() string {
	var content strings.Builder

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(tuistyles.ColorSecondary).
		MarginBottom(1)
	content.WriteString(sectionStyle.Render("📊 Available Scenarios"))
	content.WriteString("\n")

	if len(m.config.Scenarios) == 0 {
		subtleStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorMuted)
		content.WriteString(subtleStyle.Render("  No scenarios configured"))
		return content.String()
	}

	nameStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorPrimary)
	subtleStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorMuted)

	// Show up to 5 scenarios
	displayCount := min(5, len(m.config.Scenarios))
	for i := 0; i < displayCount; i++ {
		scenario := m.config.Scenarios[i]
		content.WriteString("  ")
		content.WriteString(nameStyle.Render(fmt.Sprintf("%d. %s", i+1, scenario.Name)))

		// Show participant count
		participantCount := len(scenario.ParticipantScenarios)
		content.WriteString(subtleStyle.Render(fmt.Sprintf(" (%d participant%s)",
			participantCount, pluralS(participantCount))))
		content.WriteString("\n")
	}

	if len(m.config.Scenarios) > 5 {
		content.WriteString(subtleStyle.Render(fmt.Sprintf("  ... and %d more",
			len(m.config.Scenarios)-5)))
		content.WriteString("\n")
	}

	return content.String()
}

// renderQuickActions shows available navigation shortcuts
func (m *HomeModel) renderQuickActions() string {
	var content strings.Builder

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(tuistyles.ColorSecondary).
		MarginBottom(1)
	content.WriteString(sectionStyle.Render("⚡ Quick Actions"))
	content.WriteString("\n")

	keyStyle := lipgloss.NewStyle().
		Foreground(tuistyles.ColorPrimary).
		Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorForeground)

	actions := []struct {
		key  string
		desc string
	}{
		{"s", "Browse and select scenarios"},
		{"p", "Edit scenario parameters"},
		{"c", "Compare multiple scenarios"},
		{"o", "Optimize TSP withdrawal rate"},
		{"r", "View calculation results"},
		{"?", "Show help"},
	}

	for _, action := range actions {
		content.WriteString("  ")
		content.WriteString(keyStyle.Render(action.key))
		content.WriteString(descStyle.Render("  " + action.desc))
		content.WriteString("\n")
	}

	return content.String()
}

// renderHelp shows getting started tips
func (m *HomeModel) renderHelp() string {
	var content strings.Builder

	subtleStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorMuted).Italic(true)

	content.WriteString(subtleStyle.Render("💡 Tip: Press 's' to browse scenarios and get started"))
	content.WriteString("\n")
	content.WriteString(subtleStyle.Render("    Press '?' at any time for help"))

	return content.String()
}

// Helper functions

func formatFilingStatus(status string) string {
	switch status {
	case "married_filing_jointly":
		return "Married Filing Jointly"
	case "single":
		return "Single"
	case "married_filing_separately":
		return "Married Filing Separately"
	case "head_of_household":
		return "Head of Household"
	default:
		return status
	}
}

func calculateAge(birthDate time.Time) int {
	// Simple age calculation - in a real implementation would use current date
	// For now, use a reference date from GlobalAssumptions
	currentYear := 2025 // This should come from config or time.Now()
	return currentYear - birthDate.Year()
}

func pluralS(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
