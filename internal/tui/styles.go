package tui

import "github.com/charmbracelet/lipgloss"

// Color palette inspired by modern terminal UIs
var (
	// Primary colors
	ColorPrimary   = lipgloss.Color("#00D4AA") // Teal/cyan - primary actions
	ColorSecondary = lipgloss.Color("#7C3AED") // Purple - secondary elements
	ColorAccent    = lipgloss.Color("#F59E0B") // Amber - highlights and warnings
	ColorSuccess   = lipgloss.Color("#10B981") // Green - positive metrics
	ColorDanger    = lipgloss.Color("#EF4444") // Red - negative metrics
	ColorInfo      = lipgloss.Color("#3B82F6") // Blue - informational

	// Neutral colors
	ColorBackground = lipgloss.Color("#1A1B26") // Dark background
	ColorForeground = lipgloss.Color("#C0CAF5") // Light text
	ColorMuted      = lipgloss.Color("#565F89") // Muted text
	ColorBorder     = lipgloss.Color("#414868") // Borders and dividers

	// Chart colors
	ColorChartLine1 = lipgloss.Color("#00D4AA")
	ColorChartLine2 = lipgloss.Color("#F59E0B")
	ColorChartLine3 = lipgloss.Color("#7C3AED")
	ColorChartLine4 = lipgloss.Color("#3B82F6")
)

// Base styles
var (
	// App container
	AppStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// Title bar
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			PaddingBottom(1)

	// Subtitle/breadcrumb
	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorForeground).
			Background(ColorBorder).
			Padding(0, 1)

	StatusKeyStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	// Borders and boxes
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2)

	ActiveBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(1, 2)

	// Lists and menus
	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true).
				PaddingLeft(2)

	UnselectedItemStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				PaddingLeft(2)

	// Metric cards
	MetricLabelStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Bold(true)

	MetricValueStyle = lipgloss.NewStyle().
				Foreground(ColorForeground).
				Bold(true).
				MarginTop(1)

	MetricPositiveStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Bold(true)

	MetricNegativeStyle = lipgloss.NewStyle().
				Foreground(ColorDanger).
				Bold(true)

	// Parameter controls
	ParameterLabelStyle = lipgloss.NewStyle().
				Foreground(ColorForeground).
				Bold(true)

	ParameterValueStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true)

	SliderTrackStyle = lipgloss.NewStyle().
				Foreground(ColorBorder)

	SliderThumbStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true)

	// Help text
	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Error and info messages
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDanger)

	InfoStyle = lipgloss.NewStyle().
			Foreground(ColorInfo).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorInfo)

	// Tables
	TableHeaderStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(ColorBorder)

	TableCellStyle = lipgloss.NewStyle().
			Foreground(ColorForeground).
			Padding(0, 1)

	TableHighlightStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Background(ColorBorder).
				Padding(0, 1)
)

// Helper functions for dynamic styling

// MetricTrendStyle returns appropriate style based on trend direction
func MetricTrendStyle(isPositive bool) lipgloss.Style {
	if isPositive {
		return MetricPositiveStyle
	}
	return MetricNegativeStyle
}

// TrendIndicator returns an arrow indicator for trends
func TrendIndicator(isPositive bool) string {
	if isPositive {
		return "↑"
	}
	return "↓"
}

// FormatCurrency formats a decimal value as currency with appropriate styling
func FormatCurrency(value float64) string {
	if value >= 0 {
		return MetricPositiveStyle.Render(formatCurrencyValue(value))
	}
	return MetricNegativeStyle.Render(formatCurrencyValue(value))
}

func formatCurrencyValue(value float64) string {
	// Simple formatting helper - actual formatting done by output package
	return ""
}
