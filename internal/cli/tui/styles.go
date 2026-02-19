package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Colors
var (
	colorPrimary   = lipgloss.Color("86")  // Cyan
	colorSecondary = lipgloss.Color("240") // Gray
	colorSuccess   = lipgloss.Color("82")  // Green
	colorWarning   = lipgloss.Color("214") // Orange
	colorDanger    = lipgloss.Color("196") // Red
	colorMuted     = lipgloss.Color("245") // Light gray
)

// Styles
var (
	// Title bar
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginBottom(0)

	// Help text
	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// Section headers
	sectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPrimary)

	// Progress bar
	progressBarEmptyStyle = lipgloss.NewStyle().
				Foreground(colorSecondary)

	// Table
	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPrimary).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(colorSecondary)

	tableCellStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	// Values
	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	labelStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// Error
	errorStyle = lipgloss.NewStyle().
			Foreground(colorDanger).
			Bold(true)

	// Positive/Negative deltas
	positiveDeltaStyle = lipgloss.NewStyle().
				Foreground(colorWarning)

	negativeDeltaStyle = lipgloss.NewStyle().
				Foreground(colorSuccess)
)

// getProgressColor returns color based on usage percentage
func getProgressColor(percent float64) lipgloss.Color {
	switch {
	case percent >= 90:
		return colorDanger
	case percent >= 70:
		return colorWarning
	default:
		return colorSuccess
	}
}
