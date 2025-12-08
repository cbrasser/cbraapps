package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) updateTestListView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "esc":
		// Return to classbook view
		m.state = classbookView
		return m, nil

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.tests)-1 {
			m.cursor++
		}

	case "enter":
		// Open test review for selected test
		if len(m.tests) > 0 && m.cursor < len(m.tests) {
			m.selectedTest = m.cursor
			m.selectedRow = 0
			m.selectedCol = 0
			m.editingCell = false
			m.editingGifted = false
			m.state = testReviewView
		}

	case "a":
		// Open import wizard
		var cmd tea.Cmd
		m, cmd = m.initImportView()
		m.state = importTestView
		return m, cmd

	case "d":
		// Delete test
		if len(m.tests) > 0 && m.cursor < len(m.tests) {
			return m, func() tea.Msg {
				test := m.tests[m.cursor]
				confirmed, err := ShowConfirmation("Delete Test", fmt.Sprintf("Are you sure you want to delete '%s'?", test.Title), "Yes, delete", "Cancel")
				if err != nil || !confirmed {
					return nil
				}

				if err := m.storage.DeleteTest(test.CourseID, test.ID); err != nil {
					ShowMessage("Error", fmt.Sprintf("Failed to delete test: %v", err))
					return nil
				}

				// Force reload of tests by switching states
				// We can return a Msg to reload
				return m.loadTestsCmd(test.CourseID)
			}
		}
	}

	return m, nil
}

func (m Model) renderTestListView() string {
	if m.selectedCourse >= len(m.courses) {
		m.state = listView
		return m.renderListView()
	}

	course := m.courses[m.selectedCourse]

	var b strings.Builder

	// Title
	title := titleStyle.Render(fmt.Sprintf("Tests: %s", course.Name))
	b.WriteString(title + "\n\n")

	// Test list
	if len(m.tests) == 0 {
		b.WriteString(subtitleStyle.Render("No tests yet. Use 'cbrateach add-test' to add one."))
	} else {
		for i, test := range m.tests {
			cursor := " "
			style := listItemStyle

			if i == m.cursor {
				cursor = ">"
				style = selectedItemStyle
			}

			statusIcon := "📝" // Review
			if test.Status == "confirmed" {
				statusIcon = "✓" // Confirmed
			}

			line := fmt.Sprintf("%s %s %s - %s (%s)",
				cursor,
				statusIcon,
				test.Title,
				test.Topic,
				test.Date.Format("2006-01-02"))

			b.WriteString(style.Render(line) + "\n")
		}
	}

	// Help text
	b.WriteString("\n")
	help := []string{
		"↑/k: up",
		"↓/j: down",
		"enter: open test",
		"a: add test",
		"d: delete test",
		"esc: back",
	}
	b.WriteString(helpStyle.Render(strings.Join(help, " • ")))

	return baseStyle.Render(b.String())
}
