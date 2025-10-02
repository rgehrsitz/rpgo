package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rgehrsitz/rpgo/internal/tui/tuistyles"
)

// ScenarioCard displays a compact scenario overview
type ScenarioCard struct {
	Name        string
	Description string
	Participant string
	Highlights  []string // Key parameters/metrics
	IsSelected  bool
	Width       int
}

// NewScenarioCard creates a new scenario card
func NewScenarioCard(name string) *ScenarioCard {
	return &ScenarioCard{
		Name:       name,
		Highlights: []string{},
		Width:      50,
	}
}

// WithDescription adds a description
func (s *ScenarioCard) WithDescription(desc string) *ScenarioCard {
	s.Description = desc
	return s
}

// WithParticipant sets the participant name
func (s *ScenarioCard) WithParticipant(participant string) *ScenarioCard {
	s.Participant = participant
	return s
}

// AddHighlight adds a key metric or parameter
func (s *ScenarioCard) AddHighlight(highlight string) *ScenarioCard {
	s.Highlights = append(s.Highlights, highlight)
	return s
}

// SetSelected marks the card as selected
func (s *ScenarioCard) SetSelected(selected bool) *ScenarioCard {
	s.IsSelected = selected
	return s
}

// WithWidth sets the card width
func (s *ScenarioCard) WithWidth(width int) *ScenarioCard {
	s.Width = width
	return s
}

// Render returns the styled scenario card
func (s *ScenarioCard) Render() string {
	var content strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(tuistyles.ColorPrimary)
	content.WriteString(titleStyle.Render(s.Name))
	content.WriteString("\n")

	// Participant if present
	if s.Participant != "" {
		participantStyle := lipgloss.NewStyle().
			Foreground(tuistyles.ColorMuted).
			Italic(true)
		content.WriteString(participantStyle.Render("→ " + s.Participant))
		content.WriteString("\n")
	}

	// Description if present
	if s.Description != "" {
		content.WriteString("\n")
		descStyle := lipgloss.NewStyle().
			Foreground(tuistyles.ColorForeground)
		content.WriteString(descStyle.Render(s.Description))
		content.WriteString("\n")
	}

	// Highlights
	if len(s.Highlights) > 0 {
		content.WriteString("\n")
		highlightStyle := lipgloss.NewStyle().
			Foreground(tuistyles.ColorMuted)
		for _, h := range s.Highlights {
			content.WriteString(highlightStyle.Render("• " + h))
			content.WriteString("\n")
		}
	}

	// Choose border style based on selection
	var cardStyle lipgloss.Style
	if s.IsSelected {
		cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tuistyles.ColorPrimary).
			Padding(1, 2).
			Width(s.Width)
	} else {
		cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tuistyles.ColorBorder).
			Padding(1, 2).
			Width(s.Width)
	}

	return cardStyle.Render(strings.TrimRight(content.String(), "\n"))
}

// RenderCompact returns a compact single-line version
func (s *ScenarioCard) RenderCompact() string {
	var parts []string

	// Name
	nameStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(tuistyles.ColorPrimary)
	parts = append(parts, nameStyle.Render(s.Name))

	// Participant
	if s.Participant != "" {
		participantStyle := lipgloss.NewStyle().
			Foreground(tuistyles.ColorMuted)
		parts = append(parts, participantStyle.Render("("+s.Participant+")"))
	}

	// First highlight only
	if len(s.Highlights) > 0 {
		highlightStyle := lipgloss.NewStyle().
			Foreground(tuistyles.ColorMuted)
		parts = append(parts, highlightStyle.Render("• "+s.Highlights[0]))
	}

	return strings.Join(parts, " ")
}

// ScenarioList renders a list of scenario cards
func ScenarioList(cards []*ScenarioCard) string {
	if len(cards) == 0 {
		return tuistyles.InfoStyle.Render("No scenarios available")
	}

	rendered := make([]string, len(cards))
	for i, card := range cards {
		rendered[i] = card.Render()
	}

	return lipgloss.JoinVertical(lipgloss.Left, rendered...)
}

// ScenarioListCompact renders a compact list for selection menus
func ScenarioListCompact(cards []*ScenarioCard, selectedIndex int) string {
	if len(cards) == 0 {
		return tuistyles.InfoStyle.Render("No scenarios available")
	}

	rendered := make([]string, len(cards))
	for i, card := range cards {
		prefix := "  "
		style := tuistyles.UnselectedItemStyle

		if i == selectedIndex {
			prefix = "▸ "
			style = tuistyles.SelectedItemStyle
		}

		rendered[i] = style.Render(fmt.Sprintf("%s%s", prefix, card.RenderCompact()))
	}

	return strings.Join(rendered, "\n")
}
