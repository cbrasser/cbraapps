package tui

import (
	"cbrateach/internal/models"
	"cbrateach/internal/storage"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) updateListView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.courses)-1 {
			m.cursor++
		}

	case "enter":
		// Open classbook for selected course
		if len(m.courses) > 0 && m.cursor < len(m.courses) {
			m.selectedCourse = m.cursor
			m.selectedStudent = 0
			m.state = classbookView
		}

	case "e":
		// Send email to all students in selected course
		if len(m.courses) > 0 && m.cursor < len(m.courses) {
			return m, m.sendEmailToCourse(m.cursor)
		}

	case "n":
		// Open course note in $EDITOR
		if len(m.courses) > 0 && m.cursor < len(m.courses) {
			return m, m.openCourseNote(m.cursor)
		}

	case "r":
		// Fill out after-class review
		if len(m.courses) > 0 && m.cursor < len(m.courses) {
			return m, m.openReviewForm(m.cursor)
		}

	case "t":
		// Open tests for selected course
		if len(m.courses) > 0 && m.cursor < len(m.courses) {
			m.selectedCourse = m.cursor
			course := m.courses[m.cursor]
			tests, _ := m.storage.LoadTests(course.ID)
			m.tests = tests
			m.cursor = 0
			m.state = testListView
		}

	case "a":
		// Add new course
		return m, m.addCourse()
	}

	return m, nil
}

func (m Model) renderListView() string {
	var b strings.Builder

	// Title
	title := titleStyle.Render("cbrateach - Course Management")
	b.WriteString(title + "\n\n")

	// Course list
	if len(m.courses) == 0 {
		b.WriteString(subtitleStyle.Render("No courses yet. Press 'a' to add one."))
	} else {
		for i, course := range m.courses {
			cursor := " "
			style := listItemStyle

			if i == m.cursor {
				cursor = ">"
				style = selectedItemStyle
			}

			// Format course info
			line := fmt.Sprintf("%s %s - %s (%s %s, Room %s)",
				cursor,
				course.Name,
				course.Subject,
				course.Weekday,
				course.Time,
				course.Room)

			// Add topic tag if present
			if course.CurrentTopic != "" {
				topicStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#FFF")).
					Background(primaryColor).
					Padding(0, 1).
					MarginLeft(1)
				line += " " + topicStyle.Render(course.CurrentTopic)
			}

			b.WriteString(style.Render(line) + "\n")
		}
	}

	// Help text
	b.WriteString("\n")
	help := []string{
		"↑/k: up",
		"↓/j: down",
		"enter: open classbook",
		"e: email all students",
		"n: open note",
		"r: after-class review",
		"t: tests",
		"a: add course",
		"q: quit",
	}
	b.WriteString(helpStyle.Render(strings.Join(help, " • ")))

	return baseStyle.Render(b.String())
}

func (m Model) sendEmailToCourse(idx int) tea.Cmd {
	course := m.courses[idx]

	// Collect all student emails
	var emails []string
	for _, student := range course.Students {
		if student.Email != "" {
			emails = append(emails, student.Email)
		}
	}

	if len(emails) == 0 {
		return nil
	}

	// Build pop arguments
	args := []string{}

	// Add --from if configured
	if m.cfg.SenderEmail != "" && m.cfg.SenderEmail != "teacher@example.com" {
		args = append(args, "--from", m.cfg.SenderEmail)
	}

	// Add multiple --to flags, one for each recipient
	for _, email := range emails {
		args = append(args, "--to", email)
	}

	// Add subject
	args = append(args, "--subject", fmt.Sprintf("[%s] Course Update", course.Name))

	cmd := exec.Command("pop", args...)

	// This will open pop's interactive email editor
	// ExecProcess suspends the bubbletea program and gives control to pop
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return nil
	})
}

func (m Model) openCourseNote(idx int) tea.Cmd {
	return func() tea.Msg {
		course := m.courses[idx]
		notePath := m.cfg.CourseNotesDir + "/" + course.NoteFile

		// Ensure the note file exists
		if _, err := os.Stat(notePath); os.IsNotExist(err) {
			if err := m.storage.CreateCourseNote(&m.courses[idx]); err != nil {
				return nil
			}
		}

		// Get editor from environment
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "nano" // fallback
		}

		// Open in editor
		cmd := exec.Command(editor, notePath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		_ = cmd.Run()

		return nil
	}
}

func (m Model) openReviewForm(idx int) tea.Cmd {
	return tea.ExecProcess(exec.Command("true"), func(err error) tea.Msg {
		course := m.courses[idx]

		// Show review form
		formResult, err := ShowReviewForm(course)
		if err != nil {
			return nil
		}

		// Save review and update student marks
		if err := SaveReview(m.storage, course, formResult); err != nil {
			return nil
		}

		// Reload courses to get updated marks
		courses, _ := m.storage.LoadCourses()
		m.courses = courses

		return nil
	})
}

func (m Model) addCourse() tea.Cmd {
	return tea.ExecProcess(exec.Command("true"), func(err error) tea.Msg {
		// Show course form
		formResult, err := ShowCourseForm()
		if err != nil {
			return nil
		}

		// Create new course
		course := models.Course{
			ID:           storage.GenerateID(),
			Name:         formResult.Name,
			Subject:      formResult.Subject,
			Weekday:      formResult.Weekday,
			Time:         formResult.Time,
			Room:         formResult.Room,
			CurrentTopic: formResult.CurrentTopic,
			Students:     []models.Student{},
		}

		// Create course note file
		if err := m.storage.CreateCourseNote(&course); err != nil {
			return nil
		}

		// Add to courses and save
		m.courses = append(m.courses, course)
		m.storage.SaveCourses(m.courses)

		return nil
	})
}
