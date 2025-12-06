package tui

import (
	"fmt"
	"strconv"
	"strings"

	"cbrateach/internal/models"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) updateTestReviewView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.selectedTest >= len(m.tests) {
		m.state = testListView
		return m, nil
	}

	test := &m.tests[m.selectedTest]

	// Handle editing mode
	if m.editingCell {
		switch msg.String() {
		case "enter":
			// Save the edit
			if err := m.saveEditedCell(); err == nil {
				m.editingCell = false
				m.editValue = ""
			}
			return m, nil
		case "esc":
			m.editingCell = false
			m.editValue = ""
			return m, nil
		case "backspace":
			if len(m.editValue) > 0 {
				m.editValue = m.editValue[:len(m.editValue)-1]
			}
			return m, nil
		default:
			// Add character to edit value
			if len(msg.String()) == 1 {
				m.editValue += msg.String()
			}
			return m, nil
		}
	}

	// Handle gifted points editing
	if m.editingGifted {
		switch msg.String() {
		case "enter":
			// Save gifted points
			if val, err := strconv.ParseFloat(m.editValue, 64); err == nil {
				test.GiftedPoints = val
				m.storage.RecalculateTestGrades(test)
				m.storage.UpdateTest(*test)
			}
			m.editingGifted = false
			m.editValue = ""
			return m, nil
		case "esc":
			m.editingGifted = false
			m.editValue = ""
			return m, nil
		case "backspace":
			if len(m.editValue) > 0 {
				m.editValue = m.editValue[:len(m.editValue)-1]
			}
			return m, nil
		default:
			if len(msg.String()) == 1 {
				m.editValue += msg.String()
			}
			return m, nil
		}
	}

	// Normal navigation
	switch msg.String() {
	case "ctrl+c", "q", "esc":
		m.state = testListView
		return m, nil

	case "up", "k":
		if m.selectedRow > 0 {
			m.selectedRow--
		}

	case "down", "j":
		if m.selectedRow < len(test.StudentScores)-1 {
			m.selectedRow++
		}

	case "left", "h":
		if m.selectedCol > 0 {
			m.selectedCol--
		}

	case "right", "l":
		numCols := len(test.Questions) + 2 // questions + total + grade
		if m.selectedCol < numCols-1 {
			m.selectedCol++
		}

	case "e":
		// Start editing selected cell (only question cells)
		if m.selectedCol < len(test.Questions) && test.Status == "review" {
			m.editingCell = true
			// Get current value
			questionID := test.Questions[m.selectedCol].ID
			currentValue := test.StudentScores[m.selectedRow].QuestionScores[questionID]
			m.editValue = fmt.Sprintf("%.1f", currentValue)
		}

	case "g":
		// Edit gifted points
		if test.Status == "review" {
			m.editingGifted = true
			m.editValue = fmt.Sprintf("%.1f", test.GiftedPoints)
		}

	case "c":
		// Confirm test
		if test.Status == "review" {
			test.Status = "confirmed"
			m.storage.UpdateTest(*test)
		}

	case "u":
		// Unconfirm test (back to review)
		if test.Status == "confirmed" {
			test.Status = "review"
			m.storage.UpdateTest(*test)
		}
	}

	return m, nil
}

func (m Model) saveEditedCell() error {
	if m.selectedTest >= len(m.tests) {
		return fmt.Errorf("invalid test")
	}

	test := &m.tests[m.selectedTest]

	if m.selectedRow >= len(test.StudentScores) {
		return fmt.Errorf("invalid row")
	}

	if m.selectedCol >= len(test.Questions) {
		return fmt.Errorf("invalid column")
	}

	// Parse new value
	newValue, err := strconv.ParseFloat(m.editValue, 64)
	if err != nil {
		return err
	}

	// Update score
	questionID := test.Questions[m.selectedCol].ID
	test.StudentScores[m.selectedRow].QuestionScores[questionID] = newValue

	// Recalculate
	m.storage.RecalculateTestGrades(test)

	// Save
	return m.storage.UpdateTest(*test)
}

func (m Model) renderTestReviewView() string {
	if m.selectedTest >= len(m.tests) {
		m.state = testListView
		return m.renderTestListView()
	}

	test := m.tests[m.selectedTest]

	var b strings.Builder

	// Title
	title := titleStyle.Render(fmt.Sprintf("Test Review: %s - %s", test.Title, test.Topic))
	b.WriteString(title + "\n")

	// Status and gifted points
	statusText := "Status: "
	if test.Status == "review" {
		statusText += "📝 Review Mode"
	} else {
		statusText += "✓ Confirmed"
	}

	giftedText := fmt.Sprintf("Gifted Points: %.1f", test.GiftedPoints)
	if m.editingGifted {
		editCellStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000")).
			Background(lipgloss.Color("#FFA500")).
			Bold(true)
		giftedText = fmt.Sprintf("Gifted Points: %s", editCellStyle.Render(fmt.Sprintf("%s_", m.editValue)))
	}

	b.WriteString(subtitleStyle.Render(statusText+"  •  "+giftedText) + "\n\n")

	// Build table
	columns := []table.Column{
		{Title: "Student", Width: 20},
	}

	// Add question columns
	for _, q := range test.Questions {
		columns = append(columns, table.Column{
			Title: fmt.Sprintf("%s\n(%.0f)", q.Title, q.MaxPoints),
			Width: 8,
		})
	}

	// Add total and grade columns
	columns = append(columns, table.Column{Title: "Total", Width: 8})
	columns = append(columns, table.Column{Title: "Grade", Width: 6})

	// Build rows
	var rows []table.Row

	for i, score := range test.StudentScores {
		row := table.Row{score.StudentName}

		// Add question scores
		for j, q := range test.Questions {
			points := score.QuestionScores[q.ID]
			cellValue := fmt.Sprintf("%.1f", points)

			// Highlight if editing this cell
			if m.editingCell && m.selectedRow == i && m.selectedCol == j {
				editCellStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#000")).
					Background(lipgloss.Color("#FFA500")).
					Bold(true)
				cellValue = editCellStyle.Render(fmt.Sprintf("%s_", m.editValue))
			}

			row = append(row, cellValue)
		}

		// Add total and grade
		row = append(row, fmt.Sprintf("%.1f", score.TotalPoints))
		row = append(row, fmt.Sprintf("%.2f", score.Grade))

		rows = append(rows, row)
	}

	// Create table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(min(len(rows)+1, 20)),
	)

	// Style table
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(primaryColor).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#000")).
		Background(primaryColor).
		Bold(true)
	t.SetStyles(s)

	// Set cursor position
	if m.selectedRow < len(rows) {
		t.SetCursor(m.selectedRow)
	}

	b.WriteString(t.View() + "\n\n")

	// Statistics
	avgGrade := 0.0
	for _, score := range test.StudentScores {
		avgGrade += score.Grade
	}
	if len(test.StudentScores) > 0 {
		avgGrade /= float64(len(test.StudentScores))
	}

	stats := fmt.Sprintf("Average Grade: %.2f  •  Students: %d  •  Weight: %.1f", avgGrade, len(test.StudentScores), test.Weight)
	b.WriteString(subtitleStyle.Render(stats) + "\n\n")

	// Grade distribution chart
	b.WriteString(m.renderGradeDistribution(test) + "\n")

	// Find and display missing students
	if m.selectedCourse < len(m.courses) {
		course := m.courses[m.selectedCourse]
		missingStudents := []string{}
		for _, student := range course.Students {
			found := false
			for _, score := range test.StudentScores {
				if score.StudentName == student.Name {
					found = true
					break
				}
			}
			if !found {
				missingStudents = append(missingStudents, student.Name)
			}
		}

		if len(missingStudents) > 0 {
			b.WriteString(subtitleStyle.Render("Missing Students: "))
			b.WriteString(strings.Join(missingStudents, ", ") + "\n\n")
		}
	}

	// Help text
	help := []string{
		"↑↓←→/hjkl: navigate",
	}
	if test.Status == "review" {
		help = append(help, "e: edit cell", "g: edit gifted points", "c: confirm test")
	} else {
		help = append(help, "u: unconfirm")
	}
	help = append(help, "esc: back")

	b.WriteString(helpStyle.Render(strings.Join(help, " • ")))

	return baseStyle.Render(b.String())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m Model) renderGradeDistribution(test models.Test) string {
	// Count grades
	distribution := make(map[float64]int)
	for _, score := range test.StudentScores {
		distribution[score.Grade]++
	}

	var b strings.Builder
	b.WriteString(subtitleStyle.Render("Grade Distribution (Verteilung)") + "\n")

	// Find max count for scaling
	maxCount := 0
	for _, count := range distribution {
		if count > maxCount {
			maxCount = count
		}
	}

	if maxCount == 0 {
		return ""
	}

	// Define grade range
	grades := []float64{1.0, 1.5, 2.0, 2.5, 3.0, 3.5, 4.0, 4.5, 5.0, 5.5, 6.0}

	// Print vertical bars from top to bottom
	height := 8 // Fixed height for chart
	scale := float64(maxCount) / float64(height)

	for h := height; h > 0; h-- {
		threshold := float64(h) * scale
		for _, grade := range grades {
			count := distribution[grade]
			if float64(count) >= threshold {
				b.WriteString("█ ")
			} else {
				b.WriteString("  ")
			}
		}
		if h%2 == 0 {
			b.WriteString(fmt.Sprintf(" %d", int(threshold)))
		}
		b.WriteString("\n")
	}

	// Print grade labels
	for _, grade := range grades {
		b.WriteString(fmt.Sprintf("%-2.1f", grade))
	}
	b.WriteString("\n")

	// Print counts
	for _, grade := range grades {
		count := distribution[grade]
		if count > 0 {
			b.WriteString(fmt.Sprintf("%-2d", count))
		} else {
			b.WriteString("  ")
		}
	}
	b.WriteString("\n")

	return b.String()
}
