package components

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rgehrsitz/rpgo/internal/tui/tuistyles"
)

// DataSeries represents a single line in a chart
type DataSeries struct {
	Name   string
	Points []float64
	Color  lipgloss.Color
}

// ASCIIChart displays a simple line chart
type ASCIIChart struct {
	Title      string
	Series     []*DataSeries
	Labels     []string // X-axis labels
	Width      int
	Height     int
	ShowLegend bool
	YAxisLabel string
	XAxisLabel string
}

// NewASCIIChart creates a new ASCII chart
func NewASCIIChart(title string) *ASCIIChart {
	return &ASCIIChart{
		Title:      title,
		Series:     []*DataSeries{},
		Labels:     []string{},
		Width:      60,
		Height:     15,
		ShowLegend: true,
	}
}

// AddSeries adds a data series to the chart
func (c *ASCIIChart) AddSeries(name string, points []float64, color lipgloss.Color) *ASCIIChart {
	c.Series = append(c.Series, &DataSeries{
		Name:   name,
		Points: points,
		Color:  color,
	})
	return c
}

// WithLabels sets the X-axis labels
func (c *ASCIIChart) WithLabels(labels []string) *ASCIIChart {
	c.Labels = labels
	return c
}

// WithSize sets the chart dimensions
func (c *ASCIIChart) WithSize(width, height int) *ASCIIChart {
	c.Width = width
	c.Height = height
	return c
}

// WithAxisLabels sets axis labels
func (c *ASCIIChart) WithAxisLabels(xLabel, yLabel string) *ASCIIChart {
	c.XAxisLabel = xLabel
	c.YAxisLabel = yLabel
	return c
}

// Render returns the styled chart
func (c *ASCIIChart) Render() string {
	if len(c.Series) == 0 {
		return tuistyles.InfoStyle.Render("No data to display")
	}

	var content strings.Builder

	// Title
	if c.Title != "" {
		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(tuistyles.ColorPrimary)
		content.WriteString(titleStyle.Render(c.Title))
		content.WriteString("\n\n")
	}

	// Find global min/max across all series
	globalMin, globalMax := c.getGlobalMinMax()

	// Render the chart grid
	chartGrid := c.renderGrid(globalMin, globalMax)
	content.WriteString(chartGrid)

	// X-axis label
	if c.XAxisLabel != "" {
		content.WriteString("\n")
		labelStyle := lipgloss.NewStyle().
			Foreground(tuistyles.ColorMuted).
			Italic(true)
		content.WriteString(labelStyle.Render(c.XAxisLabel))
	}

	// Legend
	if c.ShowLegend && len(c.Series) > 1 {
		content.WriteString("\n\n")
		content.WriteString(c.renderLegend())
	}

	return content.String()
}

// getGlobalMinMax finds the min and max values across all series
func (c *ASCIIChart) getGlobalMinMax() (float64, float64) {
	if len(c.Series) == 0 {
		return 0, 0
	}

	globalMin := math.Inf(1)
	globalMax := math.Inf(-1)

	for _, series := range c.Series {
		for _, point := range series.Points {
			if point < globalMin {
				globalMin = point
			}
			if point > globalMax {
				globalMax = point
			}
		}
	}

	// Add 10% padding
	padding := (globalMax - globalMin) * 0.1
	globalMin -= padding
	globalMax += padding

	return globalMin, globalMax
}

// renderGrid renders the chart grid with data points
func (c *ASCIIChart) renderGrid(minVal, maxVal float64) string {
	// Calculate chart area dimensions (excluding Y-axis labels)
	yAxisWidth := 12 // Space for Y-axis values
	chartWidth := c.Width - yAxisWidth

	// Create grid
	grid := make([][]rune, c.Height)
	for i := range grid {
		grid[i] = make([]rune, chartWidth)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	// Plot each series
	for seriesIdx, series := range c.Series {
		if len(series.Points) == 0 {
			continue
		}

		// Determine point character for this series
		pointChar := c.getSeriesChar(seriesIdx)

		// Plot points
		for i, point := range series.Points {
			// Map data point to grid position
			x := int(float64(i) / float64(len(series.Points)-1) * float64(chartWidth-1))
			y := c.Height - 1 - int((point-minVal)/(maxVal-minVal)*float64(c.Height-1))

			// Clamp to grid bounds
			if x >= 0 && x < chartWidth && y >= 0 && y < c.Height {
				grid[y][x] = pointChar
			}

			// Connect points with lines if close enough
			if i > 0 && i < len(series.Points) {
				prevX := int(float64(i-1) / float64(len(series.Points)-1) * float64(chartWidth-1))
				prevY := c.Height - 1 - int((series.Points[i-1]-minVal)/(maxVal-minVal)*float64(c.Height-1))

				c.drawLine(grid, prevX, prevY, x, y, pointChar)
			}
		}
	}

	// Render grid with Y-axis
	var output strings.Builder
	valueRange := maxVal - minVal

	for i, row := range grid {
		// Y-axis value
		yValue := maxVal - (float64(i)/float64(c.Height-1))*valueRange
		yAxisStyle := lipgloss.NewStyle().
			Foreground(tuistyles.ColorMuted).
			Width(yAxisWidth).
			Align(lipgloss.Right)

		yAxisStr := formatChartValue(yValue)
		output.WriteString(yAxisStyle.Render(yAxisStr))
		output.WriteString(" │ ")

		// Chart data
		output.WriteString(string(row))
		output.WriteString("\n")
	}

	// X-axis
	output.WriteString(strings.Repeat(" ", yAxisWidth))
	output.WriteString(" └")
	output.WriteString(strings.Repeat("─", chartWidth))
	output.WriteString("\n")

	// X-axis labels if provided
	if len(c.Labels) > 0 {
		output.WriteString(c.renderXAxisLabels(yAxisWidth, chartWidth))
	}

	return output.String()
}

// getSeriesChar returns the character to use for a series
func (c *ASCIIChart) getSeriesChar(index int) rune {
	chars := []rune{'●', '■', '▲', '♦'}
	return chars[index%len(chars)]
}

// drawLine draws a simple line between two points using Bresenham's algorithm
func (c *ASCIIChart) drawLine(grid [][]rune, x0, y0, x1, y1 int, char rune) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)

	sx := -1
	if x0 < x1 {
		sx = 1
	}

	sy := -1
	if y0 < y1 {
		sy = 1
	}

	err := dx - dy

	x, y := x0, y0

	for {
		// Set point if in bounds
		if x >= 0 && x < len(grid[0]) && y >= 0 && y < len(grid) {
			if grid[y][x] == ' ' {
				grid[y][x] = char
			}
		}

		if x == x1 && y == y1 {
			break
		}

		e2 := 2 * err

		if e2 > -dy {
			err -= dy
			x += sx
		}

		if e2 < dx {
			err += dx
			y += sy
		}
	}
}

// renderXAxisLabels renders X-axis labels
func (c *ASCIIChart) renderXAxisLabels(yAxisWidth, chartWidth int) string {
	if len(c.Labels) == 0 {
		return ""
	}

	// Show max 5 labels evenly spaced
	maxLabels := 5
	step := len(c.Labels) / maxLabels
	if step == 0 {
		step = 1
	}

	labelStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorMuted)
	var output strings.Builder

	output.WriteString(strings.Repeat(" ", yAxisWidth + 3))

	for i := 0; i < len(c.Labels); i += step {
		if i > 0 {
			// Add spacing between labels
			spacing := chartWidth / maxLabels
			output.WriteString(strings.Repeat(" ", spacing-len(c.Labels[i-step])))
		}
		output.WriteString(labelStyle.Render(c.Labels[i]))
	}

	return output.String()
}

// renderLegend renders the chart legend
func (c *ASCIIChart) renderLegend() string {
	var items []string

	for i, series := range c.Series {
		char := string(c.getSeriesChar(i))
		style := lipgloss.NewStyle().Foreground(series.Color)
		symbol := style.Render(char)

		nameStyle := lipgloss.NewStyle().Foreground(tuistyles.ColorForeground)
		name := nameStyle.Render(series.Name)

		items = append(items, fmt.Sprintf("%s %s", symbol, name))
	}

	legendStyle := lipgloss.NewStyle().
		Foreground(tuistyles.ColorMuted)

	return legendStyle.Render("Legend: " + strings.Join(items, " • "))
}

// formatChartValue formats a value for display on Y-axis
func formatChartValue(value float64) string {
	if math.Abs(value) >= 1000000 {
		return fmt.Sprintf("$%.1fM", value/1000000)
	} else if math.Abs(value) >= 1000 {
		return fmt.Sprintf("$%.0fK", value/1000)
	}
	return fmt.Sprintf("$%.0f", value)
}

// abs returns absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
