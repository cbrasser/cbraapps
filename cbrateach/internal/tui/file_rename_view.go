package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// File rename view state
type fileRenameState struct {
	submissionsPath string            // Path to submissions directory
	files           []string          // List of files found in submissions
	matches         map[string]string // filename -> target name (email prefix)
	candidates      []string          // List of student email prefixes available for matching
	cursor          int               // Current cursor position in file list
	matchFocus      bool              // True if selecting candidate, false if selecting file
	candidateCursor int               // Cursor position in candidates list (unused but kept for consistency)
}

func (m Model) initFileRenameView() (Model, tea.Cmd) {
	if m.selectedTest >= len(m.tests) {
		return m, nil
	}

	test := m.tests[m.selectedTest]

	if m.selectedCourse >= len(m.courses) {
		return m, nil
	}

	course := m.courses[m.selectedCourse]

	// Build submissions path: {feedbackDir}/{Topic}/{CourseName}/submissions
	baseDir := m.cfg.FeedbackDir
	if baseDir == "" {
		baseDir = "./feedback_export"
	}

	topic := sanitizePathComponent(test.Topic)
	courseName := sanitizePathComponent(test.CourseName)
	submissionsPath := fmt.Sprintf("%s/%s/%s/submissions", baseDir, topic, courseName)

	// Initialize file rename state
	state := fileRenameState{
		submissionsPath: submissionsPath,
		matches:         make(map[string]string),
		candidates:      []string{},
		files:           []string{},
	}

	// Scan directory for files
	entries, err := os.ReadDir(submissionsPath)
	if err != nil {
		// Directory doesn't exist or can't be read
		m.fileRenameState = state
		return m, nil
	}

	// Collect all files
	for _, entry := range entries {
		if !entry.IsDir() {
			state.files = append(state.files, entry.Name())
		}
	}

	// Build candidates list from students who took the test
	// Match student names from test scores to course students to get their emails
	for _, score := range test.StudentScores {
		// Find this student in the course to get their email
		for _, student := range course.Students {
			if student.Name == score.StudentName && student.Email != "" {
				// Extract email prefix (before @)
				parts := strings.Split(student.Email, "@")
				if len(parts) > 0 {
					state.candidates = append(state.candidates, parts[0])
				}
				break
			}
		}
	}

	// Auto-match files where possible
	for _, filename := range state.files {
		filenameLower := strings.ToLower(filename)
		// Remove extension for matching
		filenameBase := strings.TrimSuffix(filenameLower, filepath.Ext(filenameLower))

		for _, candidate := range state.candidates {
			candidateLower := strings.ToLower(candidate)

			// Try multiple matching strategies:
			// 1. Direct substring match
			if strings.Contains(filenameLower, candidateLower) {
				state.matches[filename] = candidate
				break
			}

			// 2. Split candidate by common separators and check if parts appear in filename
			// Email formats like: firstname.lastname, lastname.firstname, firstnamelastname
			candidateParts := strings.FieldsFunc(candidateLower, func(r rune) bool {
				return r == '.' || r == '-' || r == '_'
			})

			if len(candidateParts) >= 2 {
				// Check if both parts (lastname and firstname) appear in filename
				allPartsFound := true
				for _, part := range candidateParts {
					if len(part) > 2 && !strings.Contains(filenameBase, part) {
						allPartsFound = false
						break
					}
				}
				if allPartsFound {
					state.matches[filename] = candidate
					break
				}
			}

			// 3. Check if filename contains lastname_firstname pattern matching candidate parts
			// Split filename by underscores and check against candidate parts
			filenameParts := strings.Split(filenameBase, "_")
			if len(filenameParts) >= 2 && len(candidateParts) >= 2 {
				// Check if lastname and firstname from filename match candidate
				// Common pattern: nachname_vorname in file vs vorname.nachname in email
				if len(filenameParts[0]) > 2 && len(filenameParts[1]) > 2 {
					// Check nachname_vorname vs vorname.nachname
					if (strings.Contains(candidateLower, filenameParts[0]) && strings.Contains(candidateLower, filenameParts[1])) {
						state.matches[filename] = candidate
						break
					}
				}
			}
		}
	}

	m.fileRenameState = state
	return m, nil
}

func (m Model) updateFileRenameView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	state := &m.fileRenameState

	// Get sorted file list for consistent ordering
	sortedFiles := make([]string, len(state.files))
	copy(sortedFiles, state.files)
	sort.Strings(sortedFiles)

	if state.matchFocus {
		// Selecting a candidate for the current file
		if state.cursor >= len(sortedFiles) {
			state.matchFocus = false
			return m, nil
		}

		currentFile := sortedFiles[state.cursor]
		currentMatch := state.matches[currentFile]

		// Get available candidates (not already matched to other files)
		availableCandidates := m.getAvailableFilenameCandidates(state, currentFile)

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
				state.matches[currentFile] = availableCandidates[idx]
			} else {
				delete(state.matches, currentFile)
			}
			state.matchFocus = false
			state.candidateCursor = 0
		case "esc":
			state.matchFocus = false
			state.candidateCursor = 0
		}

		// Update match live preview
		if idx >= 0 && idx < len(availableCandidates) {
			state.matches[currentFile] = availableCandidates[idx]
		} else {
			delete(state.matches, currentFile)
		}

		return m, nil
	}

	// Normal Navigation
	switch msg.String() {
	case "ctrl+c", "q":
		// Exit back to test review
		m.state = testReviewView
		return m, nil

	case "esc":
		// Exit back to test review
		m.state = testReviewView
		return m, nil

	case "up", "k":
		if state.cursor > 0 {
			state.cursor--
		}

	case "down", "j":
		if state.cursor < len(sortedFiles)-1 {
			state.cursor++
		}

	case "enter":
		// Enter edit match mode
		state.matchFocus = true
		state.candidateCursor = 0

	case "r":
		// Apply renames (perform actual file operations)
		return m, m.applyFileRenames()

	case "a":
		// Auto-match remaining files
		return m.autoMatchFiles(), nil
	}

	return m, nil
}

func (m Model) getAvailableFilenameCandidates(state *fileRenameState, currentFile string) []string {
	// Build set of already-matched candidates (excluding the current file's match)
	usedCandidates := make(map[string]bool)
	for file, candidate := range state.matches {
		if file != currentFile {
			usedCandidates[candidate] = true
		}
	}

	// Filter out used candidates
	var available []string
	for _, candidate := range state.candidates {
		if !usedCandidates[candidate] {
			available = append(available, candidate)
		}
	}

	return available
}

func (m Model) autoMatchFiles() Model {
	state := &m.fileRenameState

	for _, filename := range state.files {
		// Skip if already matched
		if _, exists := state.matches[filename]; exists {
			continue
		}

		filenameLower := strings.ToLower(filename)
		filenameBase := strings.TrimSuffix(filenameLower, filepath.Ext(filenameLower))

		// Try to find a match
		for _, candidate := range state.candidates {
			// Check if candidate is already used
			alreadyUsed := false
			for _, matchedCandidate := range state.matches {
				if matchedCandidate == candidate {
					alreadyUsed = true
					break
				}
			}

			if alreadyUsed {
				continue
			}

			candidateLower := strings.ToLower(candidate)

			// Try multiple matching strategies:
			// 1. Direct substring match
			if strings.Contains(filenameLower, candidateLower) {
				state.matches[filename] = candidate
				break
			}

			// 2. Split candidate by common separators and check if parts appear in filename
			candidateParts := strings.FieldsFunc(candidateLower, func(r rune) bool {
				return r == '.' || r == '-' || r == '_'
			})

			if len(candidateParts) >= 2 {
				// Check if both parts (lastname and firstname) appear in filename
				allPartsFound := true
				for _, part := range candidateParts {
					if len(part) > 2 && !strings.Contains(filenameBase, part) {
						allPartsFound = false
						break
					}
				}
				if allPartsFound {
					state.matches[filename] = candidate
					break
				}
			}

			// 3. Check if filename contains lastname_firstname pattern matching candidate parts
			filenameParts := strings.Split(filenameBase, "_")
			if len(filenameParts) >= 2 && len(candidateParts) >= 2 {
				if len(filenameParts[0]) > 2 && len(filenameParts[1]) > 2 {
					if (strings.Contains(candidateLower, filenameParts[0]) && strings.Contains(candidateLower, filenameParts[1])) {
						state.matches[filename] = candidate
						break
					}
				}
			}
		}
	}

	return m
}

func (m Model) applyFileRenames() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		state := m.fileRenameState

		successCount := 0
		failCount := 0

		for oldName, emailPrefix := range state.matches {
			oldPath := filepath.Join(state.submissionsPath, oldName)

			// Build new filename: emailPrefix with dots replaced by dashes + extension
			// e.g., "firstname.lastname" becomes "firstname-lastname.pdf"
			ext := filepath.Ext(oldName)
			normalizedPrefix := strings.ReplaceAll(emailPrefix, ".", "-")
			newName := normalizedPrefix + ext
			newPath := filepath.Join(state.submissionsPath, newName)

			// Check if target already exists
			if _, err := os.Stat(newPath); err == nil {
				// Target exists, skip
				failCount++
				continue
			}

			// Perform rename
			if err := os.Rename(oldPath, newPath); err != nil {
				failCount++
				continue
			}

			successCount++
		}

		unmatchedCount := len(state.files) - len(state.matches)
		ShowMessage("Rename Complete",
			fmt.Sprintf("Renamed %d files.\nFailed: %d\nRemaining unmatched: %d",
				successCount, failCount, unmatchedCount))

		return nil
	})
}

func (m Model) renderFileRenameView() string {
	state := m.fileRenameState

	var b strings.Builder

	// Title
	title := titleStyle.Render("File Rename - Match Submissions to Students")
	b.WriteString(title + "\n")
	b.WriteString(subtitleStyle.Render("Review matches. Press Enter to change a match.") + "\n\n")

	// Check if directory exists
	if _, err := os.Stat(state.submissionsPath); os.IsNotExist(err) {
		b.WriteString(errorStyle.Render("Submissions directory does not exist.\n"))
		b.WriteString(subtitleStyle.Render(fmt.Sprintf("Path: %s\n", state.submissionsPath)))
		b.WriteString(helpStyle.Render("Create the directory and add submission files, then try again.\n"))
		b.WriteString(helpStyle.Render("esc: back"))
		return baseStyle.Render(b.String())
	}

	// Get sorted files
	sortedFiles := make([]string, len(state.files))
	copy(sortedFiles, state.files)
	sort.Strings(sortedFiles)

	// Scroll window
	start := 0
	end := len(sortedFiles)
	if end > 15 {
		start = state.cursor - 7
		if start < 0 {
			start = 0
		}
		end = start + 15
		if end > len(sortedFiles) {
			end = len(sortedFiles)
			start = end - 15
			if start < 0 {
				start = 0
			}
		}
	}

	for i := start; i < end; i++ {
		filename := sortedFiles[i]
		matchName, hasMatch := state.matches[filename]

		prefix := " "
		if i == state.cursor {
			prefix = ">"
		}

		var status string
		if hasMatch {
			status = fmt.Sprintf("→ %s", matchName)
			if i == state.cursor && state.matchFocus {
				status = fmt.Sprintf("→ %s ◀", matchName) // Indicate editing
			}
		} else {
			status = "→ (No Match)"
			if i == state.cursor && state.matchFocus {
				status = "→ (No Match) ◀"
			}
		}

		// Styles
		lineStyle := lipgloss.NewStyle()
		if i == state.cursor {
			lineStyle = lineStyle.Bold(true).Background(primaryColor).Foreground(lipgloss.Color("#000"))
		}

		statusStyle := lipgloss.NewStyle().Foreground(successColor)
		if !hasMatch {
			statusStyle = statusStyle.Foreground(dangerColor)
		}
		if i == state.cursor {
			statusStyle = statusStyle.Foreground(lipgloss.Color("#000")) // Ensure visible on selection
		}

		line := fmt.Sprintf("%s %-40s %s", prefix, truncate(filename, 38), statusStyle.Render(status))
		b.WriteString(lineStyle.Render(line) + "\n")
	}

	// Statistics
	matchedCount := len(state.matches)
	unmatchedCount := len(state.files) - matchedCount
	stats := fmt.Sprintf("\nTotal: %d  •  Matched: %d  •  Unmatched: %d",
		len(state.files), matchedCount, unmatchedCount)
	b.WriteString(subtitleStyle.Render(stats) + "\n\n")

	// Help text
	help := []string{
		"↑/↓: navigate",
		"enter: edit match",
		"a: auto-match",
	}
	if len(state.matches) > 0 {
		help = append(help, "r: apply renames")
	}
	help = append(help, "esc: back")

	b.WriteString(helpStyle.Render(strings.Join(help, " • ")))

	return baseStyle.Render(b.String())
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
