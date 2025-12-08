package tui

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"cbrateach/internal/storage"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var debugLog *log.Logger

func init() {
	// Create debug log file
	f, err := os.OpenFile("/tmp/cbrateach_debug.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err == nil {
		debugLog = log.New(f, "", log.Ldate|log.Ltime|log.Lshortfile)
	}
}

func (m Model) updateImportView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.importStep {
	case 0: // Select File - forward to generic handler which handles the form
		return m.updateImportFileSelectionGeneric(msg)
	case 1: // Details
		return m.updateImportDetails(msg)
	case 2: // Match
		return m.updateImportMatching(msg)
	}
	return m, nil
}

func (m Model) updateImportViewGeneric(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Don't process if showing confirmation dialog
	if m.showingConfirmation {
		return m, nil
	}

	switch m.importStep {
	case 0: // Select File - pass all messages to form
		return m.updateImportFileSelectionGeneric(msg)
	}
	return m, nil
}

func (m Model) renderImportView() string {
	if debugLog != nil {
		debugLog.Printf("renderImportView: step=%d", m.importStep)
	}

	switch m.importStep {
	case 0:
		return m.renderImportFileSelection()
	case 1:
		if debugLog != nil {
			debugLog.Printf("Rendering details: name=%q, topic=%q, weight=%q", m.importName, m.importTopic, m.importWeight)
		}
		return m.renderImportDetails()
	case 2:
		return m.renderImportMatching()
	}
	return fmt.Sprintf("DEBUG: Unknown import step: %d", m.importStep)
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

	// Use actual terminal dimensions if available, otherwise defaults
	formWidth := 120
	formHeight := 30
	pickerHeight := 20

	if m.width > 0 {
		formWidth = m.width
	}
	if m.height > 0 {
		formHeight = m.height
		pickerHeight = m.height - 10 // Leave room for title/description
		if pickerHeight < 10 {
			pickerHeight = 10
		}
	}

	// Create a local variable for the file picker value binding
	// This is a workaround for the pointer issue with Bubbletea model copying
	var selectedFile string

	m.importFilePickerForm = huh.NewForm(
		huh.NewGroup(
			huh.NewFilePicker().
				Key("filepath").
				Title("Select Test File").
				Description("Choose a .json file to import").
				AllowedTypes([]string{".json"}).
				CurrentDirectory(path).
				Height(pickerHeight).
				Value(&selectedFile),
		),
	).WithWidth(formWidth).WithHeight(formHeight)

	return m, m.importFilePickerForm.Init()
}

func (m Model) updateImportFileSelectionGeneric(msg tea.Msg) (tea.Model, tea.Cmd) {
	if debugLog != nil {
		debugLog.Printf("updateImportFileSelectionGeneric: msg type=%T, form state=%v", msg, m.importFilePickerForm.State)
	}

	// Check if form was aborted/cancelled first
	if m.importFilePickerForm != nil && m.importFilePickerForm.State == huh.StateAborted {
		if debugLog != nil {
			debugLog.Println("Form aborted, returning to test list")
		}
		m.state = testListView
		return m, nil
	}

	// Pass message to form
	form, cmd := m.importFilePickerForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.importFilePickerForm = f
	}

	// After update, get the value from the form
	if debugLog != nil {
		debugLog.Printf("After update: m.importFile=%q, form state=%v", m.importFile, m.importFilePickerForm.State)
	}

	if m.importFilePickerForm.State == huh.StateCompleted {
		if debugLog != nil {
			debugLog.Printf("Form completed! File selected: %q", m.importFile)
		}

		// Get value using GetString with the key we set
		if m.importFile == "" {
			// Try to get the value using the key
			if val := m.importFilePickerForm.GetString("filepath"); val != "" {
				m.importFile = val
				if debugLog != nil {
					debugLog.Printf("Got file from GetString('filepath'): %q", m.importFile)
				}
			}
		}

		// Also log what keys are available
		if debugLog != nil && m.importFile == "" {
			debugLog.Printf("Still no file after GetString attempt")
		}

		// File Selected - check if actually selected
		if m.importFile == "" {
			// No file selected, restart the picker
			if debugLog != nil {
				debugLog.Println("No file selected, reinitializing picker")
			}
			return m.initImportView()
		}

		if m.importFile != "" {
			// Parse JSON
			data, err := m.storage.ParseTestJSON(m.importFile)
			if err != nil {
				if debugLog != nil {
					debugLog.Printf("Failed to parse JSON: %v", err)
				}
				m.err = fmt.Errorf("failed to parse JSON: %w", err)
				m.state = testListView // Go back on error
				return m, nil
			}

			if debugLog != nil {
				debugLog.Printf("JSON parsed successfully: exam_name=%q, students=%d", data.ExamName, len(data.Students))
			}

			// Validate the data has the required fields
			if data.ExamName == "" || len(data.Students) == 0 {
				if debugLog != nil {
					debugLog.Printf("Validation failed: exam_name=%q, students=%d", data.ExamName, len(data.Students))
				}
				m.err = fmt.Errorf("invalid test JSON format: missing exam_name or students")
				m.state = testListView
				return m, nil
			}

			m.importData = data

			// Pre-fill details
			m.importName = data.ExamName
			if m.selectedCourse < len(m.courses) {
				m.importTopic = m.courses[m.selectedCourse].CurrentTopic
			} else {
				m.importTopic = "" // Set default if no course selected
			}

			if debugLog != nil {
				debugLog.Printf("Moving to step 1: name=%q, topic=%q, weight=%q", m.importName, m.importTopic, m.importWeight)
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
			// Don't pass the form's completion command to next step
			return m, nil
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
		// Show placeholder if empty
		displayValue := value
		if displayValue == "" {
			displayValue = "(empty)"
		}

		style := lipgloss.NewStyle()
		if selected {
			style = style.Foreground(primaryColor).Bold(true)
			return fmt.Sprintf("%s\n%s_", style.Render(label), displayValue) // Fake cursor
		}
		return fmt.Sprintf("%s\n%s", label, displayValue)
	}

	b.WriteString(renderInput("Exam Name", m.importName, m.importCursor == 0) + "\n\n")
	b.WriteString(renderInput("Topic", m.importTopic, m.importCursor == 1) + "\n\n")
	b.WriteString(renderInput("Weight", m.importWeight, m.importCursor == 2) + "\n\n")

	// Debug info
	b.WriteString(fmt.Sprintf("\nDebug: step=%d, name=%q, topic=%q, weight=%q\n",
		m.importStep, m.importName, m.importTopic, m.importWeight))

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

		// Get available candidates (not already matched to other students)
		availableCandidates := getAvailableCandidates(m.importCandidates, m.importMatches, currentKey)

		idx := -1
		for i, c := range availableCandidates {
			if c == currentMatch {
				idx = i
				break
			}
		}

		switch msg.String() {
		case "up", "k":
			idx--
			if idx < -1 {
				idx = len(availableCandidates) - 1
			}
		case "down", "j":
			idx++
			if idx >= len(availableCandidates) {
				idx = -1
			}
		case "enter":
			// Confirm match
			if idx >= 0 && idx < len(availableCandidates) {
				m.importMatches[currentKey] = availableCandidates[idx]
			} else {
				delete(m.importMatches, currentKey)
			}
			m.importMatchFocus = false
		case "esc":
			m.importMatchFocus = false
		}

		// Update match live preview
		if idx >= 0 && idx < len(availableCandidates) {
			m.importMatches[currentKey] = availableCandidates[idx] // Temporary update
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
		if debugLog != nil {
			debugLog.Println("cmdImportTest: Starting test import")
		}

		if m.selectedCourse >= len(m.courses) {
			if debugLog != nil {
				debugLog.Println("cmdImportTest: No course selected")
			}
			return nil
		}
		course := m.courses[m.selectedCourse]

		w, _ := strconv.ParseFloat(m.importWeight, 64)
		if w == 0 {
			w = 1.0 // Default weight
		}

		if debugLog != nil {
			debugLog.Printf("cmdImportTest: Creating test with name=%q, topic=%q, weight=%f", m.importName, m.importTopic, w)
		}

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
			if debugLog != nil {
				debugLog.Printf("cmdImportTest: Error creating test: %v", err)
			}
			return nil
		}

		m.storage.RecalculateTestGrades(test)
		err = m.storage.AddTest(*test)
		if err != nil {
			if debugLog != nil {
				debugLog.Printf("cmdImportTest: Error saving test: %v", err)
			}
			return nil
		}

		if debugLog != nil {
			debugLog.Println("cmdImportTest: Test imported successfully")
		}

		// Return message to trigger state change and reload
		return testImportedMsg{courseID: course.ID}
	}
}

// Helpers

type testImportedMsg struct {
	courseID string
}

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

func getAvailableCandidates(allCandidates []string, currentMatches map[string]string, currentKey string) []string {
	// Build set of already-matched names (excluding the current student's match)
	usedNames := make(map[string]bool)
	for key, name := range currentMatches {
		if key != currentKey {
			usedNames[name] = true
		}
	}

	// Filter out used names
	var available []string
	for _, name := range allCandidates {
		if !usedNames[name] {
			available = append(available, name)
		}
	}

	return available
}
