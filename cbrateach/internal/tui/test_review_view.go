package tui

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"cbrateach/internal/email"
	"cbrateach/internal/models"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
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

	case "f":
		// Send feedback to students
		if test.Status == "confirmed" {
			return m, m.sendFeedbackEmails()
		}

	case "x":
		// Export feedback files
		if test.Status == "confirmed" {
			return m, m.exportFeedbackFiles()
		}

	case "r":
		// Open file rename view
		if test.Status == "confirmed" {
			m.state = fileRenameView
			return m.initFileRenameView()
		}

	case "i":
		// Toggle incognito mode
		m.incognitoMode = !m.incognitoMode

	case "d":
		// Open data view
		m.state = testDataView
		return m, nil

	case "a":
		// Add missing student to test
		if test.Status == "review" {
			return m, m.addMissingStudentToTest()
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
	titleText := fmt.Sprintf("Test Review: %s - %s", test.Title, test.Topic)
	if m.incognitoMode {
		titleText += " [INCOGNITO MODE]"
	}
	title := titleStyle.Render(titleText)
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

	// Calculate averages first (needed for footer and stats)
	avgGrade := 0.0
	avgTotal := 0.0
	avgPerQuestion := make(map[string]float64)

	for _, score := range test.StudentScores {
		avgGrade += score.Grade
		avgTotal += score.TotalPoints
		for qID, points := range score.QuestionScores {
			avgPerQuestion[qID] += points
		}
	}
	if len(test.StudentScores) > 0 {
		avgGrade /= float64(len(test.StudentScores))
		avgTotal /= float64(len(test.StudentScores))
		for qID := range avgPerQuestion {
			avgPerQuestion[qID] /= float64(len(test.StudentScores))
		}
	}

	for i, score := range test.StudentScores {
		// Apply incognito mode to student name
		studentName := score.StudentName
		if m.incognitoMode {
			studentName = "*****"
		}

		row := table.Row{studentName}

		// Add question scores
		for j, q := range test.Questions {
			points := score.QuestionScores[q.ID]
			cellValue := fmt.Sprintf("%.1f", points)

			// Show editing indicator
			if m.selectedRow == i && m.selectedCol == j {
				if m.editingCell {
					cellValue = fmt.Sprintf("%s_", m.editValue)
				} else {
					cellValue = "→ " + cellValue
				}
			}

			row = append(row, cellValue)
		}

		// Add total and grade
		totalCell := fmt.Sprintf("%.1f", score.TotalPoints)

		// Mark grades < 4.0 with a visual indicator (no lipgloss styling to avoid conflicts)
		// Swiss grading system: grades below 4.0 are failing
		gradeCell := fmt.Sprintf("%.2f", score.Grade)
		if score.Grade < 4.0 {
			gradeCell = "⚠ " + gradeCell
		}

		row = append(row, totalCell)
		row = append(row, gradeCell)

		rows = append(rows, row)
	}

	// Add footer row with average points per task
	footerRow := table.Row{"Average"}
	for _, q := range test.Questions {
		footerRow = append(footerRow, fmt.Sprintf("%.1f", avgPerQuestion[q.ID]))
	}
	footerRow = append(footerRow, fmt.Sprintf("%.1f", avgTotal))
	footerRow = append(footerRow, fmt.Sprintf("%.2f", avgGrade))

	rows = append(rows, footerRow)

	// Create table - use more height now that graph is removed
	// Add 2 for header, and limit to reasonable max
	tableHeight := min(len(rows)+2, 30)
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tableHeight),
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
	maxPoints := 0.0
	for _, q := range test.Questions {
		maxPoints += q.MaxPoints
	}

	stats := fmt.Sprintf("Average Grade: %.2f  •  Max Points: %.1f  •  Students: %d  •  Weight: %.1f", avgGrade, maxPoints, len(test.StudentScores), test.Weight)
	b.WriteString(subtitleStyle.Render(stats) + "\n\n")

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
		help = append(help, "e: edit cell", "g: edit gifted points", "a: add missing student", "c: confirm test")
	} else {
		help = append(help, "u: unconfirm", "f: send feedback", "x: export feedback files", "r: rename submissions")
	}
	help = append(help, "d: data view", "i: incognito", "esc: back")

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

func (m Model) sendFeedbackEmails() tea.Cmd {
	return tea.ExecProcess(exec.Command("true"), func(err error) tea.Msg {
		if m.selectedTest >= len(m.tests) {
			return nil
		}

		test := m.tests[m.selectedTest]

		if m.selectedCourse >= len(m.courses) {
			return nil
		}

		course := m.courses[m.selectedCourse]

		// Automatically determine feedback directory using same structure as export
		baseDir := m.cfg.FeedbackDir
		if baseDir == "" {
			baseDir = "./feedback_export"
		}

		// Use same path structure as export: {feedbackDir}/{Topic}/{CourseName}/feedback
		topic := sanitizePathComponent(test.Topic)
		courseName := sanitizePathComponent(test.CourseName)
		feedbackPath := fmt.Sprintf("%s/%s/%s/feedback", baseDir, topic, courseName)

		// Optional: Show form only for custom message
		formResult, err := ShowCustomMessageForm()
		if err != nil {
			return nil
		}

		customMessage := formResult.CustomMessage

		// Preview loop - allow user to preview, edit, and re-preview
		for {
			// Prepare emails with current custom message
			emails, err := email.PrepareFeedbackEmails(m.cfg, test, course, feedbackPath, customMessage)
			if err != nil {
				ShowMessage("Error", fmt.Sprintf("Failed to prepare emails: %v", err))
				return nil
			}

			if len(emails) == 0 {
				ShowMessage("No Emails", "No students with email addresses found for this test.")
				return nil
			}

			// Show preview of first email
			preview := email.EmailPreview(emails[0], m.cfg.BCCEmail, true)
			previewResult, err := ShowEmailPreview(preview, customMessage, len(emails))
			if err != nil {
				return nil
			}

			switch previewResult.Action {
			case EmailPreviewSend:
				// User confirmed, proceed to send
				// Show final summary
				summary := email.EmailSummary(emails)
				confirmed, err := ShowConfirmation("Send Feedback Emails", summary, "Yes, send emails", "Cancel")
				if err != nil || !confirmed {
					return nil
				}

				// Send emails using pop for each student
				successCount := 0
				for i, e := range emails {
					// BCC on first email only
					addBCC := (i == 0)
					if err := m.sendFeedbackEmailWithPop(e, addBCC); err != nil {
						ShowMessage("Email Error", fmt.Sprintf("Failed to send email to %s: %v", e.StudentName, err))
						continue
					}
					successCount++

					// Rate limiting: wait 1.1 seconds after every 2 emails
					if (i+1) % 2 == 0 && i < len(emails)-1 {
						time.Sleep(1100 * time.Millisecond)
					}
				}

				ShowMessage("Emails Sent", fmt.Sprintf("Successfully sent %d out of %d emails.", successCount, len(emails)))
				return nil

			case EmailPreviewEdit:
				// User wants to edit, update custom message and loop again
				customMessage = previewResult.CustomMessage
				continue

			case EmailPreviewCancel:
				// User cancelled
				return nil
			}
		}
	})
}

func (m Model) sendFeedbackEmailWithPop(e email.FeedbackEmail, addBCC bool) error {
	// Build pop arguments
	args := []string{}

	// Add recipient
	args = append(args, "--to", e.StudentEmail)

	// Add BCC if configured and requested (first email only)
	if addBCC && m.cfg.BCCEmail != "" {
		args = append(args, "--bcc", m.cfg.BCCEmail)
	}

	// Add subject
	args = append(args, "--subject", e.Subject)

	// Add body
	args = append(args, "--body", e.Body)

	// Add from if configured
	if m.cfg.SenderEmail != "" && m.cfg.SenderEmail != "teacher@example.com" {
		args = append(args, "--from", m.cfg.SenderEmail)
	}

	// Add attachments
	for _, attachment := range e.Attachments {
		args = append(args, "--attach", attachment)
	}

	// Note: pop sends by default when --preview is not specified
	// No additional flag needed

	cmd := exec.Command("pop", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pop command failed: %w (output: %s)", err, string(output))
	}

	return nil
}

func (m Model) exportFeedbackFiles() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if m.selectedTest >= len(m.tests) {
			return nil
		}

		test := &m.tests[m.selectedTest]

		if m.selectedCourse >= len(m.courses) {
			ShowMessage("Error", "Course not found")
			return nil
		}

		course := m.courses[m.selectedCourse]

		// Use feedback directory from config with default fallback
		baseDir := m.cfg.FeedbackDir
		if baseDir == "" {
			baseDir = "./feedback_export"
		}

		// Create directory structure: {feedbackDir}/{Topic}/{CourseName}/feedback
		// Use sanitizePathComponent to properly clean path parts
		topic := sanitizePathComponent(test.Topic)
		courseName := sanitizePathComponent(test.CourseName)
		feedbackPath := fmt.Sprintf("%s/%s/%s/feedback", baseDir, topic, courseName)

		// Export feedback files (template is now embedded in the code)
		err := m.storage.ExportFeedbackFiles(test, course, feedbackPath)
		if err != nil {
			ShowMessage("Export Error", fmt.Sprintf("Failed to export feedback files: %v", err))
			return nil
		}

		ShowMessage("Export Successful", fmt.Sprintf("Feedback files exported to:\n%s", feedbackPath))
		return nil
	})
}

// sanitizePathComponent sanitizes a string for use in file paths
func sanitizePathComponent(s string) string {
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	return s
}

func (m Model) addMissingStudentToTest() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if m.selectedTest >= len(m.tests) {
			return nil
		}

		test := &m.tests[m.selectedTest]

		if m.selectedCourse >= len(m.courses) {
			ShowMessage("Error", "Course not found")
			return nil
		}

		course := m.courses[m.selectedCourse]

		// Find missing students
		missingStudents := []models.Student{}
		for _, student := range course.Students {
			found := false
			for _, score := range test.StudentScores {
				if score.StudentName == student.Name {
					found = true
					break
				}
			}
			if !found {
				missingStudents = append(missingStudents, student)
			}
		}

		if len(missingStudents) == 0 {
			ShowMessage("No Missing Students", "All students from this course are already in the test.")
			return nil
		}

		// Show selection dialog
		selectedStudent, err := ShowMissingStudentSelection(missingStudents)
		if err != nil {
			// User cancelled or error
			return nil
		}

		// Create new student score with 0.0 for all questions
		newScore := models.StudentScore{
			StudentName:      selectedStudent.Name,
			QuestionScores:   make(map[string]float64),
			QuestionComments: make(map[string]string),
			TotalPoints:      0.0,
			Grade:            6.0, // Worst grade in Swiss system
		}

		// Initialize all question scores to 0.0
		for _, q := range test.Questions {
			newScore.QuestionScores[q.ID] = 0.0
		}

		// Calculate grade
		newScore.Grade = test.CalculateGrade(&newScore)

		// Add to test
		test.StudentScores = append(test.StudentScores, newScore)

		// Save updated test
		if err := m.storage.UpdateTest(*test); err != nil {
			ShowMessage("Error", fmt.Sprintf("Failed to add student: %v", err))
			return nil
		}

		ShowMessage("Student Added", fmt.Sprintf("%s has been added to the test with 0.0 points for all questions.", selectedStudent.Name))

		return nil
	})
}
