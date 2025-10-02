package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rgehrsitz/rpgo/internal/tui/tuistyles"
)

// ProgressBar displays a progress indicator
type ProgressBar struct {
	Current     int
	Total       int
	Width       int
	Label       string
	ShowPercent bool
	ShowCount   bool
}

// NewProgressBar creates a new progress bar
func NewProgressBar(current, total int) *ProgressBar {
	return &ProgressBar{
		Current:     current,
		Total:       total,
		Width:       40,
		ShowPercent: true,
		ShowCount:   true,
	}
}

// WithLabel sets the progress label
func (p *ProgressBar) WithLabel(label string) *ProgressBar {
	p.Label = label
	return p
}

// WithWidth sets the bar width
func (p *ProgressBar) WithWidth(width int) *ProgressBar {
	p.Width = width
	return p
}

// Update updates the progress
func (p *ProgressBar) Update(current int) {
	p.Current = current
}

// Percentage returns the completion percentage
func (p *ProgressBar) Percentage() float64 {
	if p.Total == 0 {
		return 0
	}
	return float64(p.Current) / float64(p.Total) * 100
}

// IsComplete returns true if progress is at 100%
func (p *ProgressBar) IsComplete() bool {
	return p.Current >= p.Total
}

// Render returns the styled progress bar
func (p *ProgressBar) Render() string {
	var content strings.Builder

	// Label
	if p.Label != "" {
		labelStyle := lipgloss.NewStyle().
			Foreground(tuistyles.ColorForeground).
			Bold(true)
		content.WriteString(labelStyle.Render(p.Label))
		content.WriteString("\n")
	}

	// Progress bar
	percentage := p.Percentage()
	filled := int(float64(p.Width) * percentage / 100)
	if filled > p.Width {
		filled = p.Width
	}
	empty := p.Width - filled

	barStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorSuccess)
	emptyStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorBorder)

	content.WriteString("[")
	if filled > 0 {
		content.WriteString(barStyle.Render(strings.Repeat("█", filled)))
	}
	if empty > 0 {
		content.WriteString(emptyStyle.Render(strings.Repeat("░", empty)))
	}
	content.WriteString("]")

	// Stats
	var stats []string
	if p.ShowPercent {
		percentStyle := lipgloss.NewStyle().
			Foreground(tuistyles.ColorPrimary).
			Bold(true)
		stats = append(stats, percentStyle.Render(fmt.Sprintf("%.1f%%", percentage)))
	}
	if p.ShowCount {
		countStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorMuted)
		stats = append(stats, countStyle.Render(fmt.Sprintf("%d/%d", p.Current, p.Total)))
	}

	if len(stats) > 0 {
		content.WriteString(" ")
		content.WriteString(strings.Join(stats, " • "))
	}

	return content.String()
}

// ProgressPanel displays multiple progress bars or status updates
type ProgressPanel struct {
	Title   string
	Items   []ProgressItem
	Width   int
	Height  int
}

// ProgressItem is a single item in the progress panel
type ProgressItem struct {
	Label    string
	Status   string // "pending", "running", "complete", "error"
	Progress *ProgressBar
	Message  string
}

// NewProgressPanel creates a new progress panel
func NewProgressPanel(title string) *ProgressPanel {
	return &ProgressPanel{
		Title:  title,
		Items:  []ProgressItem{},
		Width:  60,
		Height: 20,
	}
}

// AddItem adds a progress item
func (p *ProgressPanel) AddItem(item ProgressItem) *ProgressPanel {
	p.Items = append(p.Items, item)
	return p
}

// UpdateItem updates an item by index
func (p *ProgressPanel) UpdateItem(index int, item ProgressItem) {
	if index >= 0 && index < len(p.Items) {
		p.Items[index] = item
	}
}

// Render returns the styled progress panel
func (p *ProgressPanel) Render() string {
	var content strings.Builder

	// Title
	if p.Title != "" {
		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(tuistyles.ColorPrimary)
		content.WriteString(titleStyle.Render(p.Title))
		content.WriteString("\n\n")
	}

	// Items
	for _, item := range p.Items {
		content.WriteString(p.renderItem(item))
		content.WriteString("\n")
	}

	// Wrap in border
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tuistyles.ColorBorder).
		Padding(1, 2).
		Width(p.Width)

	return panelStyle.Render(content.String())
}

// renderItem renders a single progress item
func (p *ProgressPanel) renderItem(item ProgressItem) string {
	var parts []string

	// Status icon
	icon := p.getStatusIcon(item.Status)
	iconStyle := p.getStatusStyle(item.Status)
	parts = append(parts, iconStyle.Render(icon))

	// Label
	labelStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorForeground)
	parts = append(parts, labelStyle.Render(item.Label))

	line := strings.Join(parts, " ")

	// Progress bar if present
	if item.Progress != nil {
		line += "\n  " + item.Progress.Render()
	}

	// Message if present
	if item.Message != "" {
		msgStyle := lipgloss.NewStyle().
			Foreground(tuistyles.ColorMuted).
			Italic(true)
		line += "\n  " + msgStyle.Render(item.Message)
	}

	return line
}

// getStatusIcon returns an icon for the status
func (p *ProgressPanel) getStatusIcon(status string) string {
	switch status {
	case "pending":
		return "○"
	case "running":
		return "◐"
	case "complete":
		return "●"
	case "error":
		return "✗"
	default:
		return "○"
	}
}

// getStatusStyle returns a style for the status
func (p *ProgressPanel) getStatusStyle(status string) lipgloss.Style {
	switch status {
	case "pending":
		return lipgloss.NewStyle().Foreground(tuistyles.ColorMuted)
	case "running":
		return lipgloss.NewStyle().Foreground(tuistyles.ColorInfo)
	case "complete":
		return lipgloss.NewStyle().Foreground(tuistyles.ColorSuccess)
	case "error":
		return lipgloss.NewStyle().Foreground(tuistyles.ColorDanger)
	default:
		return lipgloss.NewStyle().Foreground(tuistyles.ColorMuted)
	}
}

// Spinner represents an animated spinner for loading states
type Spinner struct {
	Frame   int
	Message string
}

// NewSpinner creates a new spinner
func NewSpinner() *Spinner {
	return &Spinner{
		Frame: 0,
	}
}

// WithMessage sets the spinner message
func (s *Spinner) WithMessage(message string) *Spinner {
	s.Message = message
	return s
}

// Next advances the spinner to the next frame
func (s *Spinner) Next() {
	s.Frame++
}

// Render returns the current spinner frame
func (s *Spinner) Render() string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := frames[s.Frame%len(frames)]

	spinnerStyle := lipgloss.NewStyle().
		Foreground(tuistyles.ColorPrimary).
		Bold(true)

	rendered := spinnerStyle.Render(frame)

	if s.Message != "" {
		messageStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorForeground)
		rendered += " " + messageStyle.Render(s.Message)
	}

	return rendered
}
