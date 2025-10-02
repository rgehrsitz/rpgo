package tui

import "github.com/rgehrsitz/rpgo/internal/tui/tuistyles"

// Re-export styles from tuistyles to avoid import cycles
var (
	// Colors
	ColorPrimary   = tuistyles.ColorPrimary
	ColorSecondary = tuistyles.ColorSecondary
	ColorAccent    = tuistyles.ColorAccent
	ColorSuccess   = tuistyles.ColorSuccess
	ColorDanger    = tuistyles.ColorDanger
	ColorInfo      = tuistyles.ColorInfo

	ColorBackground = tuistyles.ColorBackground
	ColorForeground = tuistyles.ColorForeground
	ColorMuted      = tuistyles.ColorMuted
	ColorBorder     = tuistyles.ColorBorder

	ColorChartLine1 = tuistyles.ColorChartLine1
	ColorChartLine2 = tuistyles.ColorChartLine2
	ColorChartLine3 = tuistyles.ColorChartLine3
	ColorChartLine4 = tuistyles.ColorChartLine4

	// Base styles
	AppStyle              = tuistyles.AppStyle
	TitleStyle            = tuistyles.TitleStyle
	SubtitleStyle         = tuistyles.SubtitleStyle
	StatusBarStyle        = tuistyles.StatusBarStyle
	StatusKeyStyle        = tuistyles.StatusKeyStyle
	BorderStyle           = tuistyles.BorderStyle
	ActiveBorderStyle     = tuistyles.ActiveBorderStyle
	SelectedItemStyle     = tuistyles.SelectedItemStyle
	UnselectedItemStyle   = tuistyles.UnselectedItemStyle
	MetricLabelStyle      = tuistyles.MetricLabelStyle
	MetricValueStyle      = tuistyles.MetricValueStyle
	MetricPositiveStyle   = tuistyles.MetricPositiveStyle
	MetricNegativeStyle   = tuistyles.MetricNegativeStyle
	ParameterLabelStyle   = tuistyles.ParameterLabelStyle
	ParameterValueStyle   = tuistyles.ParameterValueStyle
	SliderTrackStyle      = tuistyles.SliderTrackStyle
	SliderThumbStyle      = tuistyles.SliderThumbStyle
	HelpKeyStyle          = tuistyles.HelpKeyStyle
	HelpDescStyle         = tuistyles.HelpDescStyle
	ErrorStyle            = tuistyles.ErrorStyle
	InfoStyle             = tuistyles.InfoStyle
	TableHeaderStyle      = tuistyles.TableHeaderStyle
	TableCellStyle        = tuistyles.TableCellStyle
	TableHighlightStyle   = tuistyles.TableHighlightStyle
)

// Re-export helper functions
var (
	MetricTrendStyle = tuistyles.MetricTrendStyle
	TrendIndicator   = tuistyles.TrendIndicator
	FormatCurrency   = tuistyles.FormatCurrency
)
