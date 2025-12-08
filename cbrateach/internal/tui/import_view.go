package tui

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"cbrateach/internal/storage"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) updateImportView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.importStep {
	case 0: // Select File
		return m.updateImportFileSelection(msg)
	case 1: // Details
		return m.updateImportDetails(msg)
	case 2: // Match
		return m.updateImportMatching(msg)
	}
	return m, nil
}

func (m Model) renderImportView() string {
	switch m.importStep {
	case 0:
		return m.renderImportFileSelection()
	case 1:
		return m.renderImportDetails()
	case 2:
		return m.renderImportMatching()
	}
	return ""
}

// Step 0: File Selection

func (m Model) initImportView() (Model, tea.Cmd) {
	// Reset state
	m.importStep = 0
	m.importData = nil
	m.importMatches = make(map[string]string)
	m.importUnmatched = nil
	m.importCursor = 0
	m.importName = ""
	m.importTopic = ""
	m.importWeight = "1.0"

	// Initialize Huh File Picker Form
	// Start at ExportDir from config, or current dir if not set
	path := m.cfg.ExportDir
	if path == "" {
		path, _ = os.Getwd()
	}

	m.importFilePickerForm = huh.NewForm(
		huh.NewGroup(
			huh.NewFilePicker().
				Title("Select Test File").
				Description("Choose a .json file to import").
				AllowedTypes([]string{".json"}).
				CurrentDirectory(path).
				Value(&m.importFile),
		),
	)

	return m, m.importFilePickerForm.Init()
}

func (m Model) updateImportFileSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle global cancel manually if needed, or rely on form
	switch msg.String() {
	case "ctrl+c", "esc":
		// huh handles ctrl+c?
		// If generic form logic doesn't catch esc, we can.
		// Usually huh forms return error on cancel.
		// But update logic:
	}

	form, cmd := m.importFilePickerForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.importFilePickerForm = f
	}

	if m.importFilePickerForm.State == huh.StateCompleted {
		// File Selected
		if m.importFile != "" {
			// Parse JSON
			data, err := m.storage.ParseTestJSON(m.importFile)
			if err != nil {
				m.err = err
				// Reset or show error?
				return m, nil
			}
			m.importData = data

			// Pre-fill details
			m.importName = data.ExamName
			if m.selectedCourse < len(m.courses) {
				m.importTopic = m.courses[m.selectedCourse].CurrentTopic
			}

			// Run auto-match
			if m.selectedCourse < len(m.courses) {
				course := m.courses[m.selectedCourse]
				m.importMatches, m.importUnmatched = m.storage.MatchStudents(data, course.Students)

				// Prepare candidates list for manual matching
				var candidates []string
				for _, s := range course.Students {
					candidates = append(candidates, s.Name)
				}
				sort.Strings(candidates)
				m.importCandidates = candidates
			}

			// Move to next step
			m.importStep = 1
			m.importCursor = 0
		}
	}

	return m, cmd
}

func (m Model) renderImportFileSelection() string {
	return m.importFilePickerForm.View()
}

// Step 1: Details (Simple field navigation)
// We'll reuse the cursor for field selection: 0=Name, 1=Topic, 2=Weight

func (m Model) updateImportDetails(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.importStep = 0
		// Re-init picker?
		return m.initImportView()

	case "tab", "down", "j":
		m.importCursor = (m.importCursor + 1) % 3

	case "shift+tab", "up", "k":
		m.importCursor--
		if m.importCursor < 0 {
			m.importCursor = 2
		}

	case "enter":
		// Validate and move to matching
		if m.importName != "" && m.importTopic != "" {
			m.importStep = 2
			m.importCursor = 0
			m.importMatchFocus = false
		}

	case "backspace":
		switch m.importCursor {
		case 0:
			if len(m.importName) > 0 {
				m.importName = m.importName[:len(m.importName)-1]
			}
		case 1:
			if len(m.importTopic) > 0 {
				m.importTopic = m.importTopic[:len(m.importTopic)-1]
			}
		case 2:
			if len(m.importWeight) > 0 {
				m.importWeight = m.importWeight[:len(m.importWeight)-1]
			}
		}

	default:
		// Text input
		char := msg.String()
		if len(char) == 1 {
			switch m.importCursor {
			case 0:
				m.importName += char
			case 1:
				m.importTopic += char
			case 2:
				m.importWeight += char
			}
		}
	}
	return m, nil
}

func (m Model) renderImportDetails() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Import Test: Details") + "\n\n")

	renderInput := func(label, value string, selected bool) string {
		style := lipgloss.NewStyle()
		if selected {
			style = style.Foreground(primaryColor).Bold(true)
			// label = "> " + label
			return fmt.Sprintf("%s\n%s_", style.Render(label), value) // Fake cursor
		}
		return fmt.Sprintf("%s\n%s", label, value)
	}

	b.WriteString(renderInput("Exam Name", m.importName, m.importCursor == 0) + "\n\n")
	b.WriteString(renderInput("Topic", m.importTopic, m.importCursor == 1) + "\n\n")
	b.WriteString(renderInput("Weight", m.importWeight, m.importCursor == 2) + "\n\n")

	b.WriteString("\n" + helpStyle.Render("tab/arrow: next field • enter: continue • esc: back"))
	return baseStyle.Render(b.String())
}

// Step 2: Matching
// List of import students.

func (m Model) updateImportMatching(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Flatten students list from map for consistent ordering
	sortedKeys := getSortedStudentKeys(m.importData)

	if m.importMatchFocus {
		// Selecting a candidate for the current student
		currentKey := sortedKeys[m.importCursor]
		currentMatch := m.importMatches[currentKey]

		idx := -1
		for i, c := range m.importCandidates {
			if c == currentMatch {
				idx = i
				break
			}
		}

		switch msg.String() {
		case "up", "k":
			idx--
			if idx < -1 {
				idx = len(m.importCandidates) - 1
			}
		case "down", "j":
			idx++
			if idx >= len(m.importCandidates) {
				idx = -1
			}
		case "enter":
			// Confirm match
			if idx >= 0 {
				m.importMatches[currentKey] = m.importCandidates[idx]
			} else {
				delete(m.importMatches, currentKey)
			}
			m.importMatchFocus = false
		case "esc":
			m.importMatchFocus = false
		}

		// Update match live preview
		if idx >= 0 {
			m.importMatches[currentKey] = m.importCandidates[idx] // Temporary update
		} else {
			delete(m.importMatches, currentKey)
		}

		return m, nil
	}

	// Normal Navigation
	switch msg.String() {
	case "ctrl+c", "q":
		m.state = testListView
		return m, nil

	case "esc":
		m.importStep = 1
		return m, nil

	case "up", "k":
		if m.importCursor > 0 {
			m.importCursor--
		}

	case "down", "j":
		if m.importCursor < len(sortedKeys)-1 {
			m.importCursor++
		}

	case "enter":
		// Enter edit match mode
		m.importMatchFocus = true

	case "i":
		// Execute Import
		return m, m.cmdImportTest()
	}

	return m, nil
}

func (m Model) renderImportMatching() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Import Test: Match Students") + "\n")
	b.WriteString(subtitleStyle.Render("Review matches. Press Enter to change a match.") + "\n\n")

	sortedKeys := getSortedStudentKeys(m.importData)

	// Scroll window
	start := 0
	end := len(sortedKeys)
	if end > 15 {
		start = m.importCursor - 7
		if start < 0 {
			start = 0
		}
		end = start + 15
		if end > len(sortedKeys) {
			end = len(sortedKeys)
			start = end - 15
			if start < 0 {
				start = 0
			}
		}
	}

	for i := start; i < end; i++ {
		key := sortedKeys[i]
		origName := m.importData.Students[key].Name
		matchName, hasMatch := m.importMatches[key]

		prefix := " "
		if i == m.importCursor {
			prefix = ">"
		}

		var status string
		if hasMatch {
			status = fmt.Sprintf("→ %s", matchName)
			if i == m.importCursor && m.importMatchFocus {
				status = fmt.Sprintf("→ %s ◀", matchName) // Indicate editing
			}
		} else {
			status = "→ (No Match)"
			if i == m.importCursor && m.importMatchFocus {
				status = "→ (No Match) ◀"
			}
		}

		// Styles
		lineStyle := lipgloss.NewStyle()
		if i == m.importCursor {
			lineStyle = lineStyle.Bold(true).Background(primaryColor).Foreground(lipgloss.Color("#000"))
		}

		statusStyle := lipgloss.NewStyle().Foreground(successColor)
		if !hasMatch {
			statusStyle = statusStyle.Foreground(dangerColor)
		}
		if i == m.importCursor {
			statusStyle = statusStyle.Foreground(lipgloss.Color("#000")) // Ensure visible on selection
		}

		line := fmt.Sprintf("%s %-25s %s", prefix, origName, statusStyle.Render(status))
		b.WriteString(lineStyle.Render(line) + "\n")
	}

	b.WriteString("\n" + helpStyle.Render("↑/↓: navigate • enter: edit match • i: finish import • esc: back"))

	return baseStyle.Render(b.String())
}

func (m Model) cmdImportTest() tea.Cmd {
	return func() tea.Msg {
		if m.selectedCourse >= len(m.courses) {
			return nil
		}
		course := m.courses[m.selectedCourse]

		w, _ := strconv.ParseFloat(m.importWeight, 64)

		test, err := m.storage.CreateTestFromJSON(
			m.importData,
			m.importMatches,
			course.ID,
			course.Name,
			m.importName,
			m.importTopic,
			w,
		)

		if err != nil {
			return nil
		}

		m.storage.RecalculateTestGrades(test)
		m.storage.AddTest(*test)

		return saveCoursesMsg{}
	}
}

// Helpers

func getSortedStudentKeys(data *storage.JSONImport) []string {
	if data == nil {
		return nil
	}
	var keys []string
	for k := range data.Students {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
