package tui

import (
	"cbrateach/internal/models"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) updateClassbookView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "esc":
		// Return to list view
		m.state = listView
		return m, nil

	case "up", "k":
		if m.selectedStudent > 0 {
			m.selectedStudent--
		}

	case "down", "j":
		course := m.courses[m.selectedCourse]
		if m.selectedStudent < len(course.Students)-1 {
			m.selectedStudent++
		}

	case "e":
		// Send email to selected student
		return m, m.sendEmailToStudent()

	case "n":
		// Edit note for selected student
		return m, m.editStudentNote()

	case "a":
		// Add new student
		return m, m.addStudent()

	case "x":
		// Delete selected student
		return m, m.deleteStudent()

	case "d":
		// Edit course details
		return m, m.editCourseDetails()

	case "t":
		// Open tests for this course
		course := m.courses[m.selectedCourse]
		tests, _ := m.storage.LoadTests(course.ID)
		m.tests = tests
		m.cursor = 0
		m.state = testListView
		return m, nil

	case "g":
		// Export final grades
		return m, m.exportFinalGrades()
	}

	return m, nil
}

func (m Model) renderClassbookView() string {
	if m.selectedCourse >= len(m.courses) {
		m.state = listView
		return m.renderListView()
	}

	course := m.courses[m.selectedCourse]

	// Two-pane layout: Students on left, details on right
	studentsPane := m.renderStudentsPane(course)
	detailsPane := m.renderCourseDetails(course)

	// Combine panes side by side
	combined := lipgloss.JoinHorizontal(
		lipgloss.Top,
		boxStyle.Width(40).Render(studentsPane),
		boxStyle.Width(50).Render(detailsPane),
	)

	// Title
	title := titleStyle.Render(fmt.Sprintf("Classbook: %s", course.Name))

	// Help
	help := []string{
		"↑/k: up",
		"↓/j: down",
		"e: email student",
		"n: edit note",
		"a: add student",
		"x: delete student",
		"d: edit details",
		"t: tests",
		"g: export final grades",
		"esc: back",
	}
	helpText := helpStyle.Render(strings.Join(help, " • "))

	return baseStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			"",
			combined,
			"",
			helpText,
		),
	)
}

func (m Model) renderStudentsPane(course models.Course) string {
	var b strings.Builder

	b.WriteString(subtitleStyle.Render(fmt.Sprintf("Students (%d)", len(course.Students))) + "\n\n")

	if len(course.Students) == 0 {
		b.WriteString(subtitleStyle.Render("No students yet"))
		return b.String()
	}

	for i, student := range course.Students {
		cursor := " "
		style := listItemStyle

		if i == m.selectedStudent {
			cursor = ">"
			style = selectedItemStyle
		}

		// Show positive/negative marks
		indicators := ""
		if len(student.PositiveMarks) > 0 {
			indicators += positiveStyle.Render(fmt.Sprintf(" +%d", len(student.PositiveMarks)))
		}
		if len(student.NegativeMarks) > 0 {
			indicators += negativeStyle.Render(fmt.Sprintf(" -%d", len(student.NegativeMarks)))
		}

		line := fmt.Sprintf("%s %s%s", cursor, student.Name, indicators)
		b.WriteString(style.Render(line) + "\n")
	}

	return b.String()
}

func (m Model) renderCourseDetails(course models.Course) string {
	var b strings.Builder

	// Course info
	b.WriteString(subtitleStyle.Render("Course Details") + "\n\n")
	b.WriteString(fmt.Sprintf("Subject: %s\n", course.Subject))
	b.WriteString(fmt.Sprintf("Schedule: %s at %s\n", course.Weekday, course.Time))
	b.WriteString(fmt.Sprintf("Room: %s\n\n", course.Room))

	b.WriteString(fmt.Sprintf("Current Topic:\n%s\n\n", course.CurrentTopic))

	// Selected student details
	if len(course.Students) > 0 && m.selectedStudent < len(course.Students) {
		student := course.Students[m.selectedStudent]

		b.WriteString(subtitleStyle.Render("Selected Student") + "\n\n")
		b.WriteString(fmt.Sprintf("Name: %s\n", student.Name))
		b.WriteString(fmt.Sprintf("Email: %s\n", student.Email))

		if student.Note != "" {
			b.WriteString(fmt.Sprintf("\nNote:\n%s\n", student.Note))
		}

		// Show recent marks
		if len(student.PositiveMarks) > 0 {
			b.WriteString("\n" + positiveStyle.Render("Positive marks:") + "\n")
			for _, mark := range student.PositiveMarks {
				b.WriteString(fmt.Sprintf("  • %s: %s\n",
					mark.Date.Format("2006-01-02"),
					mark.Reason))
			}
		}

		if len(student.NegativeMarks) > 0 {
			b.WriteString("\n" + negativeStyle.Render("Negative marks:") + "\n")
			for _, mark := range student.NegativeMarks {
				b.WriteString(fmt.Sprintf("  • %s: %s\n",
					mark.Date.Format("2006-01-02"),
					mark.Reason))
			}
		}

		// Load and display student's test grades
		tests, err := m.storage.LoadTests(course.ID)
		if err == nil && len(tests) > 0 {
			b.WriteString("\n" + subtitleStyle.Render("Test Grades:") + "\n")

			var totalWeightedGrade float64
			var totalWeight float64

			for _, test := range tests {
				// Find this student's score in the test
				for _, score := range test.StudentScores {
					if score.StudentName == student.Name {
						weight := test.Weight
						if weight <= 0 {
							weight = 1.0
						}

						statusIcon := "📝"
						if test.Status == "confirmed" {
							statusIcon = "✓"
							totalWeightedGrade += score.Grade * weight
							totalWeight += weight
						}

						b.WriteString(fmt.Sprintf("  %s %s: %.2f (weight: %.1f)\n",
							statusIcon, test.Title, score.Grade, weight))
						break
					}
				}
			}

			// Calculate and show average
			if totalWeight > 0 {
				avgGrade := totalWeightedGrade / totalWeight
				b.WriteString("\n")
				avgStyle := lipgloss.NewStyle().
					Foreground(successColor).
					Bold(true)
				b.WriteString(avgStyle.Render(fmt.Sprintf("Average Grade: %.2f", avgGrade)) + "\n")
			}
		}
	}

	return b.String()
}

func (m Model) sendEmailToStudent() tea.Cmd {
	course := m.courses[m.selectedCourse]
	if m.selectedStudent >= len(course.Students) {
		return nil
	}

	student := course.Students[m.selectedStudent]
	if student.Email == "" {
		return nil
	}

	// Build pop arguments - keep it simple to ensure interactive mode works
	args := []string{}

	// Add --to with student email
	args = append(args, "--to", student.Email)

	// Add --subject
	args = append(args, "--subject", fmt.Sprintf("[%s] Message from teacher", course.Name))

	// Add --from if configured (optional, pop will use default if not set)
	if m.cfg.SenderEmail != "" && m.cfg.SenderEmail != "teacher@example.com" {
		args = append(args, "--from", m.cfg.SenderEmail)
	}

	cmd := exec.Command("pop", args...)

	// ExecProcess suspends the bubbletea program and gives control to pop
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		// Return to normal state after pop exits
		return nil
	})
}

func (m Model) editStudentNote() tea.Cmd {
	return tea.ExecProcess(exec.Command("true"), func(err error) tea.Msg {
		if m.selectedStudent >= len(m.courses[m.selectedCourse].Students) {
			return nil
		}

		currentNote := m.courses[m.selectedCourse].Students[m.selectedStudent].Note

		// Show edit note form
		newNote, err := ShowEditNoteForm(currentNote)
		if err != nil {
			return nil
		}

		// Update student note
		m.courses[m.selectedCourse].Students[m.selectedStudent].Note = newNote
		m.storage.SaveCourses(m.courses)

		return nil
	})
}

func (m Model) addStudent() tea.Cmd {
	return tea.ExecProcess(exec.Command("true"), func(err error) tea.Msg {
		// Show student form
		formResult, err := ShowStudentForm()
		if err != nil {
			return nil
		}

		// Create new student
		student := models.Student{
			Name:  formResult.Name,
			Email: formResult.Email,
			Note:  formResult.Note,
		}

		// Add to course and save
		m.courses[m.selectedCourse].Students = append(
			m.courses[m.selectedCourse].Students,
			student,
		)
		m.storage.SaveCourses(m.courses)

		return nil
	})
}

func (m Model) editCourseDetails() tea.Cmd {
	return tea.ExecProcess(exec.Command("true"), func(err error) tea.Msg {
		if m.selectedCourse >= len(m.courses) {
			return nil
		}

		// Show course edit form
		formResult, err := ShowCourseEditForm(&m.courses[m.selectedCourse])
		if err != nil {
			return nil
		}

		// Update course details
		m.courses[m.selectedCourse].Subject = formResult.Subject
		m.courses[m.selectedCourse].Weekday = formResult.Weekday
		m.courses[m.selectedCourse].Time = formResult.Time
		m.courses[m.selectedCourse].Room = formResult.Room
		m.courses[m.selectedCourse].CurrentTopic = formResult.CurrentTopic

		// Save changes
		m.storage.SaveCourses(m.courses)

		return nil
	})
}

func (m Model) deleteStudent() tea.Cmd {
	return tea.ExecProcess(exec.Command("true"), func(err error) tea.Msg {
		if m.selectedCourse >= len(m.courses) {
			return nil
		}

		course := &m.courses[m.selectedCourse]
		if m.selectedStudent >= len(course.Students) {
			return nil
		}

		// Remove the student at selectedStudent index
		course.Students = append(course.Students[:m.selectedStudent], course.Students[m.selectedStudent+1:]...)

		// Adjust selectedStudent if needed
		if m.selectedStudent >= len(course.Students) && m.selectedStudent > 0 {
			m.selectedStudent--
		}

		// Save changes
		m.storage.SaveCourses(m.courses)

		return nil
	})
}

func (m Model) exportFinalGrades() tea.Cmd {
	return tea.ExecProcess(exec.Command("true"), func(err error) tea.Msg {
		if m.selectedCourse >= len(m.courses) {
			return nil
		}

		course := m.courses[m.selectedCourse]

		// Ask user to choose format
		format, err := ShowExportFormatChoice()
		if err != nil {
			return nil
		}

		// Generate filename with timestamp
		timestamp := time.Now().Format("2006-01-02")
		sanitizedName := strings.ToLower(strings.ReplaceAll(course.Name, " ", "_"))
		var outputPath string

		switch format {
		case "csv":
			filename := fmt.Sprintf("%s_final_grades_%s.csv", sanitizedName, timestamp)
			outputPath = filepath.Join(m.cfg.ExportDir, filename)
			err = m.storage.ExportGrades(course.ID, outputPath)
		case "xlsx":
			filename := fmt.Sprintf("%s_final_grades_%s.xlsx", sanitizedName, timestamp)
			outputPath = filepath.Join(m.cfg.ExportDir, filename)
			err = m.storage.ExportGradesXLSX(course.ID, outputPath)
		default:
			return nil
		}

		if err != nil {
			// Show error message
			ShowMessage("Export Error", err.Error())
			return nil
		}

		// Show success message
		ShowMessage("Export Successful", fmt.Sprintf("Grades exported to:\n%s", outputPath))

		return nil
	})
}
