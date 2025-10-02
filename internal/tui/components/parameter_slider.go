package components

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rgehrsitz/rpgo/internal/tui/tuistyles"
)

// ParameterSlider displays an adjustable parameter with visual slider
type ParameterSlider struct {
	Label       string
	Value       float64
	Min         float64
	Max         float64
	Step        float64
	Unit        string // e.g., "%", "years", "$"
	Format      string // e.g., "%.2f", "%.0f"
	Width       int    // Total width of slider bar
	IsFocused   bool
	Description string
}

// NewParameterSlider creates a new parameter slider
func NewParameterSlider(label string, value, min, max, step float64) *ParameterSlider {
	return &ParameterSlider{
		Label:  label,
		Value:  value,
		Min:    min,
		Max:    max,
		Step:   step,
		Format: "%.2f",
		Width:  30,
	}
}

// WithUnit sets the unit suffix
func (p *ParameterSlider) WithUnit(unit string) *ParameterSlider {
	p.Unit = unit
	return p
}

// WithFormat sets the value format string
func (p *ParameterSlider) WithFormat(format string) *ParameterSlider {
	p.Format = format
	return p
}

// WithWidth sets the slider width
func (p *ParameterSlider) WithWidth(width int) *ParameterSlider {
	p.Width = width
	return p
}

// SetFocused sets the focus state
func (p *ParameterSlider) SetFocused(focused bool) *ParameterSlider {
	p.IsFocused = focused
	return p
}

// WithDescription adds a description/help text
func (p *ParameterSlider) WithDescription(desc string) *ParameterSlider {
	p.Description = desc
	return p
}

// Increment increases the value by step
func (p *ParameterSlider) Increment() {
	newValue := p.Value + p.Step
	if newValue <= p.Max {
		p.Value = newValue
	}
}

// Decrement decreases the value by step
func (p *ParameterSlider) Decrement() {
	newValue := p.Value - p.Step
	if newValue >= p.Min {
		p.Value = newValue
	}
}

// SetValue sets the value directly, clamping to min/max
func (p *ParameterSlider) SetValue(value float64) {
	p.Value = math.Max(p.Min, math.Min(p.Max, value))
}

// Percentage returns the value as a percentage of the range
func (p *ParameterSlider) Percentage() float64 {
	if p.Max == p.Min {
		return 0
	}
	return (p.Value - p.Min) / (p.Max - p.Min)
}

// Render returns the styled parameter slider
func (p *ParameterSlider) Render() string {
	var content strings.Builder

	// Label
	labelStyle := tuistyles.ParameterLabelStyle
	if p.IsFocused {
		labelStyle = labelStyle.Foreground(tuistyles.ColorPrimary)
	}
	content.WriteString(labelStyle.Render(p.Label))
	content.WriteString("\n")

	// Value display
	valueStr := fmt.Sprintf(p.Format, p.Value)
	if p.Unit != "" {
		valueStr += p.Unit
	}
	valueStyle := tuistyles.ParameterValueStyle
	if p.IsFocused {
		valueStyle = valueStyle.Foreground(tuistyles.ColorAccent)
	}
	content.WriteString(valueStyle.Render(valueStr))
	content.WriteString("\n")

	// Slider bar
	content.WriteString(p.renderSliderBar())

	// Range indicator
	rangeStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorMuted)
	minStr := fmt.Sprintf(p.Format, p.Min)
	maxStr := fmt.Sprintf(p.Format, p.Max)
	if p.Unit != "" {
		minStr += p.Unit
		maxStr += p.Unit
	}
	rangeText := fmt.Sprintf("%s  ─  %s", minStr, maxStr)
	content.WriteString("\n")
	content.WriteString(rangeStyle.Render(rangeText))

	// Description if present
	if p.Description != "" {
		content.WriteString("\n")
		descStyle := lipgloss.NewStyle().
			Foreground(tuistyles.ColorMuted).
			Italic(true)
		content.WriteString(descStyle.Render(p.Description))
	}

	// Control hints if focused
	if p.IsFocused {
		content.WriteString("\n")
		hintStyle := lipgloss.NewStyle().
			Foreground(tuistyles.ColorInfo).
			Italic(true)
		content.WriteString(hintStyle.Render("← → to adjust • ↑↓ to navigate"))
	}

	return content.String()
}

// renderSliderBar creates the visual slider bar
func (p *ParameterSlider) renderSliderBar() string {
	percentage := p.Percentage()
	filled := int(math.Round(float64(p.Width) * percentage))
	empty := p.Width - filled

	// Ensure we stay within bounds
	if filled < 0 {
		filled = 0
	}
	if filled > p.Width {
		filled = p.Width
	}
	if empty < 0 {
		empty = 0
	}

	var bar strings.Builder

	// Track style
	trackStyle := tuistyles.SliderTrackStyle
	thumbStyle := tuistyles.SliderThumbStyle

	if p.IsFocused {
		thumbStyle = thumbStyle.Foreground(tuistyles.ColorAccent)
	}

	// Build the bar
	bar.WriteString("[")

	if filled > 0 {
		// Filled portion with thumb at the end
		if filled > 1 {
			bar.WriteString(thumbStyle.Render(strings.Repeat("━", filled-1)))
		}
		bar.WriteString(thumbStyle.Render("●"))
	} else {
		// Thumb at start
		bar.WriteString(thumbStyle.Render("●"))
	}

	if empty > 1 {
		bar.WriteString(trackStyle.Render(strings.Repeat("─", empty-1)))
	}

	bar.WriteString("]")

	return bar.String()
}

// RenderCompact returns a compact single-line version
func (p *ParameterSlider) RenderCompact() string {
	valueStr := fmt.Sprintf(p.Format, p.Value)
	if p.Unit != "" {
		valueStr += p.Unit
	}

	labelStyle := tuistyles.ParameterLabelStyle
	valueStyle := tuistyles.ParameterValueStyle

	if p.IsFocused {
		labelStyle = labelStyle.Foreground(tuistyles.ColorPrimary)
		valueStyle = valueStyle.Foreground(tuistyles.ColorAccent)
	}

	label := labelStyle.Render(p.Label + ":")
	value := valueStyle.Render(valueStr)

	// Mini slider
	miniBar := p.renderMiniSliderBar(10)

	return fmt.Sprintf("%s %s %s", label, value, miniBar)
}

// renderMiniSliderBar creates a compact slider bar
func (p *ParameterSlider) renderMiniSliderBar(width int) string {
	percentage := p.Percentage()
	filled := int(math.Round(float64(width) * percentage))

	var bar strings.Builder
	bar.WriteString("[")

	thumbStyle := tuistyles.SliderThumbStyle
	trackStyle := tuistyles.SliderTrackStyle

	for i := 0; i < width; i++ {
		if i == filled {
			bar.WriteString(thumbStyle.Render("●"))
		} else if i < filled {
			bar.WriteString(thumbStyle.Render("━"))
		} else {
			bar.WriteString(trackStyle.Render("─"))
		}
	}

	bar.WriteString("]")
	return bar.String()
}
