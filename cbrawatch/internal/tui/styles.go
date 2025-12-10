package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	cleanColor     = lipgloss.Color("#10B981") // Green
	warningColor   = lipgloss.Color("#F59E0B") // Amber
	dangerColor    = lipgloss.Color("#EF4444") // Red
	infoColor      = lipgloss.Color("#3B82F6") // Blue
	mutedColor     = lipgloss.Color("#6B7280") // Gray
	primaryColor   = lipgloss.Color("#8B5CF6") // Purple
	highlightColor = lipgloss.Color("#EC4899") // Pink

	// Status indicators
	cleanIndicator   = lipgloss.NewStyle().Foreground(cleanColor).Render("●")
	warningIndicator = lipgloss.NewStyle().Foreground(warningColor).Render("●")
	dangerIndicator  = lipgloss.NewStyle().Foreground(dangerColor).Render("●")
	infoIndicator    = lipgloss.NewStyle().Foreground(infoColor).Render("●")

	// Base styles
	baseStyle = lipgloss.NewStyle().
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginLeft(2)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	// List item styles
	listItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	listItemDescStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(mutedColor)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(primaryColor).
				Bold(true)

	selectedItemDescStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(highlightColor)

	// Cursor style for selected item
	cursorStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	// Status text styles
	cleanStyle = lipgloss.NewStyle().
			Foreground(cleanColor)

	warningStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	dangerStyle = lipgloss.NewStyle().
			Foreground(dangerColor)

	infoStyle = lipgloss.NewStyle().
			Foreground(infoColor)

	mutedStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Help text
	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Padding(1, 2)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	// Message/notification styles
	successStyle = lipgloss.NewStyle().
			Foreground(cleanColor).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(dangerColor).
			Bold(true)

	processingStyle = lipgloss.NewStyle().
			Foreground(infoColor).
			Italic(true)

	// Border box for messages
	messageBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1).
			MarginTop(1).
			MarginLeft(2).
			MarginRight(2)

	// Spinner style
	spinnerStyle = lipgloss.NewStyle().
			Foreground(primaryColor)

	// List container style
	listStyle = lipgloss.NewStyle().
			Padding(1, 2)
)
