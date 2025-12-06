package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("#7C3AED")
	secondaryColor = lipgloss.Color("#3B82F6")
	successColor   = lipgloss.Color("#10B981")
	dangerColor    = lipgloss.Color("#EF4444")
	mutedColor     = lipgloss.Color("#6B7280")
	bgColor        = lipgloss.Color("#1F2937")

	// Base styles
	baseStyle = lipgloss.NewStyle().
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginBottom(1)

	// List styles
	listItemStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			MarginBottom(0)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("#000")).
				Background(primaryColor).
				Bold(true)

	// Help text
	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

	// Error style
	errorStyle = lipgloss.NewStyle().
			Foreground(dangerColor).
			Bold(true)

	// Info boxes
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			MarginRight(2)

	// Positive/Negative indicators
	positiveStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	negativeStyle = lipgloss.NewStyle().
			Foreground(dangerColor).
			Bold(true)
)
