package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")).
			MarginBottom(1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2).
			Width(50)

	timeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	dateStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("117"))
)

type model struct {
	currentTime time.Time
}

func initialModel() model {
	return model{
		currentTime: time.Now(),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Quit
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m model) View() string {
	// Title
	title := titleStyle.Render("🐧 Pingu")

	// Date and Time section
	_, week := m.currentTime.ISOWeek()
	dateStr := dateStyle.Render(m.currentTime.Format("Monday, January 2, 2006"))
	timeStr := timeStyle.Render(m.currentTime.Format("15:04:05"))
	weekStr := dateStyle.Render(fmt.Sprintf("Week %d", week))

	dateTimeBox := boxStyle.Render(
		fmt.Sprintf("%s\n%s\n%s", dateStr, timeStr, weekStr),
	)

	return fmt.Sprintf("\n%s\n%s\n",
		title,
		dateTimeBox,
	)
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
