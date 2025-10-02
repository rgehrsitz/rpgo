package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/rgehrsitz/rpgo/internal/tui/tuistyles"
)

// MetricCard displays a single metric with label, value, and optional trend
type MetricCard struct {
	Label       string
	Value       string
	Trend       *Trend
	Description string
	Width       int
}

// Trend represents a metric's change direction and amount
type Trend struct {
	IsPositive bool
	Change     string // e.g., "+$5,234" or "-2.3%"
}

// NewMetricCard creates a new metric card
func NewMetricCard(label, value string) *MetricCard {
	return &MetricCard{
		Label: label,
		Value: value,
		Width: 30,
	}
}

// WithTrend adds a trend indicator to the metric card
func (m *MetricCard) WithTrend(isPositive bool, change string) *MetricCard {
	m.Trend = &Trend{
		IsPositive: isPositive,
		Change:     change,
	}
	return m
}

// WithDescription adds a description/subtitle
func (m *MetricCard) WithDescription(desc string) *MetricCard {
	m.Description = desc
	return m
}

// WithWidth sets the card width
func (m *MetricCard) WithWidth(width int) *MetricCard {
	m.Width = width
	return m
}

// Render returns the styled metric card
func (m *MetricCard) Render() string {
	// Label
	label := tuistyles.MetricLabelStyle.Render(m.Label)

	// Value with appropriate styling
	value := tuistyles.MetricValueStyle.Render(m.Value)

	// Trend indicator if present
	var trend string
	if m.Trend != nil {
		arrow := tuistyles.TrendIndicator(m.Trend.IsPositive)
		trendStyle := tuistyles.MetricTrendStyle(m.Trend.IsPositive)
		trend = "\n" + trendStyle.Render(fmt.Sprintf("%s %s", arrow, m.Trend.Change))
	}

	// Description if present
	var desc string
	if m.Description != "" {
		desc = "\n" + tuistyles.SubtitleStyle.Render(m.Description)
	}

	content := label + "\n" + value + trend + desc

	// Wrap in card style
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tuistyles.ColorBorder).
		Padding(1, 2).
		Width(m.Width)

	return cardStyle.Render(content)
}

// RenderCompact returns a compact inline version without border
func (m *MetricCard) RenderCompact() string {
	label := tuistyles.MetricLabelStyle.Render(m.Label + ":")
	value := tuistyles.MetricValueStyle.Render(m.Value)

	var trend string
	if m.Trend != nil {
		arrow := tuistyles.TrendIndicator(m.Trend.IsPositive)
		trendStyle := tuistyles.MetricTrendStyle(m.Trend.IsPositive)
		trend = " " + trendStyle.Render(fmt.Sprintf("%s %s", arrow, m.Trend.Change))
	}

	return label + " " + value + trend
}

// MetricGrid renders multiple metric cards in a grid layout
func MetricGrid(cards []*MetricCard, columns int) string {
	if len(cards) == 0 {
		return ""
	}

	rows := []string{}
	currentRow := []string{}

	for i, card := range cards {
		currentRow = append(currentRow, card.Render())

		// Start new row when we reach column limit or end of cards
		if (i+1)%columns == 0 || i == len(cards)-1 {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, currentRow...))
			currentRow = []string{}
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
