package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cbranotes/internal/config"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Focus states for the editor
type focusState int

const (
	focusFilePicker focusState = iota
	focusSearch
	focusEditor
	focusConfirmClose // Confirmation dialog for unsaved changes
)

// EditorModel is the main model for the note editor
type EditorModel struct {
	config        *config.Config
	filePicker    filepicker.Model
	searchInput   textinput.Model
	textArea      textarea.Model
	focus         focusState
	currentFile   string
	fileContent   string
	hasChanges    bool
	width         int
	height        int
	statusMsg     string
	quitting      bool
	fileOpen      bool
	filteredFiles []string
	allFiles      []string
	confirmAction string // "close" or "quit" - what action triggered the confirmation
}

// Styles
var (
	editorTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true).
				Padding(0, 1)

	editorPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))

	editorActivePaneStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("205"))

	editorStatusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Padding(0, 1)

	editorStatusSuccessStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("78")).
					Padding(0, 1)

	editorStatusErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Padding(0, 1)

	editorSearchStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("117"))

	editorHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	editorDialogStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("205")).
				Padding(1, 2).
				Align(lipgloss.Center)

	editorDialogTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)
)

// NewEditorModel creates a new editor model
func NewEditorModel(cfg *config.Config) EditorModel {
	// Initialize file picker with very small initial height
	// Will be properly sized once we receive WindowSizeMsg
	fp := filepicker.New()
	fp.CurrentDirectory = cfg.NotesPath
	fp.AllowedTypes = []string{".md", ".txt", ".org", ".norg"}
	fp.ShowHidden = false
	fp.ShowPermissions = false
	fp.ShowSize = false
	fp.Height = 5

	// Initialize search input
	si := textinput.New()
	si.Placeholder = "Search..."
	si.Width = 15

	// Initialize text area with small initial size
	ta := textarea.New()
	ta.Placeholder = "Select a file to edit..."
	ta.ShowLineNumbers = true
	ta.SetWidth(30)
	ta.SetHeight(5)

	return EditorModel{
		config:      cfg,
		filePicker:  fp,
		searchInput: si,
		textArea:    ta,
		focus:       focusFilePicker,
	}
}

func (m EditorModel) Init() tea.Cmd {
	return tea.Batch(
		m.filePicker.Init(),
		m.loadAllFiles(),
	)
}

// loadAllFiles scans the notes directory for files
func (m EditorModel) loadAllFiles() tea.Cmd {
	return func() tea.Msg {
		var files []string
		filepath.Walk(m.config.NotesPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() {
				ext := strings.ToLower(filepath.Ext(path))
				if ext == ".md" || ext == ".txt" || ext == ".org" || ext == ".norg" {
					relPath, _ := filepath.Rel(m.config.NotesPath, path)
					files = append(files, relPath)
				}
			}
			return nil
		})
		return filesLoadedMsg{files: files}
	}
}

type filesLoadedMsg struct {
	files []string
}

type fileReadMsg struct {
	content string
	err     error
}

type fileSavedMsg struct {
	err error
}

type systemEditorDoneMsg struct {
	err error
}

type closeFileAfterSaveMsg struct{}

func (m EditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.updateDimensions()

	case tea.KeyMsg:
		key := msg.String()

		// Handle confirmation dialog first
		if m.focus == focusConfirmClose {
			switch key {
			case "y", "Y":
				// Save and then perform the action
				if m.confirmAction == "quit" {
					// Save then quit
					m.quitting = true
					return m, tea.Batch(m.saveFile(), tea.Quit)
				}
				// Save then close file
				return m, tea.Batch(m.saveFile(), func() tea.Msg {
					return closeFileAfterSaveMsg{}
				})
			case "n", "N":
				// Discard and perform the action
				if m.confirmAction == "quit" {
					m.quitting = true
					return m, tea.Quit
				}
				// Close without saving
				m.fileOpen = false
				m.currentFile = ""
				m.textArea.SetValue("")
				m.hasChanges = false
				m.statusMsg = "File closed (changes discarded)"
				m.focus = focusFilePicker
				m.confirmAction = ""
				return m, nil
			case "esc", "ctrl+c":
				// Cancel - go back to editor
				m.focus = focusEditor
				m.confirmAction = ""
				m.statusMsg = ""
				return m, nil
			}
			return m, nil
		}

		// Global quit - ctrl+c always works, plus configurable hotkey
		if key == "ctrl+c" || key == m.config.Editor.Hotkeys.Quit {
			if m.hasChanges && m.fileOpen {
				m.focus = focusConfirmClose
				m.confirmAction = "quit"
				m.statusMsg = ""
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		}

		// Check hotkey bindings
		if key == m.config.Editor.Hotkeys.Save && m.fileOpen {
			return m, m.saveFile()
		}
		if key == m.config.Editor.Hotkeys.CloseFile && m.fileOpen {
			if m.hasChanges {
				m.focus = focusConfirmClose
				m.confirmAction = "close"
				m.statusMsg = ""
				return m, nil
			}
			m.fileOpen = false
			m.currentFile = ""
			m.textArea.SetValue("")
			m.hasChanges = false
			m.statusMsg = "File closed"
			m.focus = focusFilePicker
			return m, nil
		}
		if key == m.config.Editor.Hotkeys.SwitchToFilePicker && !m.config.Editor.EditorInMainWindow {
			m.focus = focusFilePicker
			m.searchInput.Blur()
			return m, nil
		}

		// Handle focus switching
		if key == "tab" && !m.config.Editor.EditorInMainWindow {
			switch m.focus {
			case focusFilePicker:
				m.focus = focusSearch
				m.searchInput.Focus()
			case focusSearch:
				if m.fileOpen {
					m.focus = focusEditor
					m.searchInput.Blur()
					m.textArea.Focus()
				} else {
					m.focus = focusFilePicker
					m.searchInput.Blur()
				}
			case focusEditor:
				m.focus = focusFilePicker
				m.textArea.Blur()
			}
			return m, nil
		}

		// Handle escape - go back to file picker or close search
		if key == "esc" {
			if m.focus == focusSearch {
				m.focus = focusFilePicker
				m.searchInput.Blur()
				m.searchInput.SetValue("")
				return m, nil
			}
			if m.focus == focusEditor && !m.config.Editor.EditorInMainWindow {
				m.focus = focusFilePicker
				m.textArea.Blur()
				return m, nil
			}
		}

		// Handle search shortcut
		if key == "/" && m.focus == focusFilePicker {
			m.focus = focusSearch
			m.searchInput.Focus()
			return m, nil
		}

	case filesLoadedMsg:
		m.allFiles = msg.files
		m.filteredFiles = msg.files

	case fileReadMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.fileContent = msg.content
			m.textArea.SetValue(msg.content)
			m.hasChanges = false
			m.fileOpen = true
			m.statusMsg = fmt.Sprintf("Opened: %s", filepath.Base(m.currentFile))

			// If using system editor, open it
			if m.config.Editor.UseSystemEditor {
				return m, m.openSystemEditor()
			}

			// Switch focus to editor
			if m.config.Editor.EditorInMainWindow {
				m.focus = focusEditor
				m.textArea.Focus()
			} else {
				m.focus = focusEditor
				m.textArea.Focus()
			}
		}

	case fileSavedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Save failed: %v", msg.err)
		} else {
			m.hasChanges = false
			m.fileContent = m.textArea.Value()
			m.statusMsg = "Saved!"
		}

	case closeFileAfterSaveMsg:
		// Called after saving when user chose to save before closing
		m.fileOpen = false
		m.currentFile = ""
		m.textArea.SetValue("")
		m.hasChanges = false
		m.statusMsg = "File saved and closed"
		m.focus = focusFilePicker
		m.confirmAction = ""

	case systemEditorDoneMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Editor error: %v", msg.err)
		} else {
			// Reload the file content
			return m, m.readFile(m.currentFile)
		}
	}

	// Update components based on focus
	switch m.focus {
	case focusFilePicker:
		var cmd tea.Cmd
		m.filePicker, cmd = m.filePicker.Update(msg)
		cmds = append(cmds, cmd)

		// Prevent navigating outside the notes directory
		if !strings.HasPrefix(m.filePicker.CurrentDirectory, m.config.NotesPath) {
			m.filePicker.CurrentDirectory = m.config.NotesPath
		}

		// Check if a file was selected
		if didSelect, path := m.filePicker.DidSelectFile(msg); didSelect {
			m.currentFile = path
			m.statusMsg = "Loading..."
			cmds = append(cmds, m.readFile(path))
		}

	case focusSearch:
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		cmds = append(cmds, cmd)
		// Filter files based on search
		m.filterFiles()

	case focusEditor:
		if !m.config.Editor.UseSystemEditor {
			var cmd tea.Cmd
			oldValue := m.textArea.Value()
			m.textArea, cmd = m.textArea.Update(msg)
			if m.textArea.Value() != oldValue {
				m.hasChanges = m.textArea.Value() != m.fileContent
			}
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *EditorModel) filterFiles() {
	query := strings.ToLower(m.searchInput.Value())
	if query == "" {
		m.filteredFiles = m.allFiles
		return
	}

	var filtered []string
	for _, f := range m.allFiles {
		if fuzzyMatch(strings.ToLower(f), query) {
			filtered = append(filtered, f)
		}
	}
	m.filteredFiles = filtered
}

// fuzzyMatch performs a simple fuzzy match
func fuzzyMatch(str, pattern string) bool {
	patternIdx := 0
	for i := 0; i < len(str) && patternIdx < len(pattern); i++ {
		if str[i] == pattern[patternIdx] {
			patternIdx++
		}
	}
	return patternIdx == len(pattern)
}

// truncateHeight limits a string to maxLines lines
func truncateHeight(s string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	return strings.Join(lines[:maxLines], "\n")
}

func (m EditorModel) readFile(path string) tea.Cmd {
	return func() tea.Msg {
		content, err := os.ReadFile(path)
		return fileReadMsg{content: string(content), err: err}
	}
}

func (m EditorModel) saveFile() tea.Cmd {
	return func() tea.Msg {
		err := os.WriteFile(m.currentFile, []byte(m.textArea.Value()), 0644)
		return fileSavedMsg{err: err}
	}
}

func (m EditorModel) openSystemEditor() tea.Cmd {
	return func() tea.Msg {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = os.Getenv("VISUAL")
		}
		if editor == "" {
			editor = "vim"
		}

		cmd := exec.Command(editor, m.currentFile)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		return systemEditorDoneMsg{err: err}
	}
}

func (m EditorModel) updateDimensions() EditorModel {
	// Ensure minimum dimensions
	if m.height < 12 {
		m.height = 12
	}
	if m.width < 45 {
		m.width = 45
	}

	// Pane height calculation: total height - title(1) - status(1) - help(1)
	paneHeight := m.height - 5
	if paneHeight < 6 {
		paneHeight = 6
	}
	innerHeight := paneHeight - 2 // Account for borders

	if m.config.Editor.EditorInMainWindow {
		// Full width mode
		paneWidth := m.width - 2
		fpHeight := innerHeight - 2 // Leave room for search bar
		if fpHeight < 4 {
			fpHeight = 4
		}
		m.filePicker.Height = fpHeight
		m.searchInput.Width = paneWidth - 6 // Account for border and emoji
		m.textArea.SetWidth(m.width - 4)
		m.textArea.SetHeight(innerHeight)
	} else {
		// Split view - match renderSplitView calculations
		availableWidth := m.width - 1
		leftWidth := availableWidth / 3
		if leftWidth < 20 {
			leftWidth = 20
		}
		rightWidth := availableWidth - leftWidth
		if rightWidth < 20 {
			rightWidth = 20
		}
		if leftWidth+rightWidth > m.width {
			leftWidth = m.width / 2
			rightWidth = m.width - leftWidth - 1
		}

		// File picker height: inner height - search bar (2 lines)
		fpHeight := innerHeight - 2
		if fpHeight < 4 {
			fpHeight = 4
		}

		m.filePicker.Height = fpHeight
		m.searchInput.Width = leftWidth - 6 // Account for border and emoji
		m.textArea.SetWidth(rightWidth - 4) // Account for border and padding
		m.textArea.SetHeight(innerHeight - 2)
	}
	return m
}

func (m EditorModel) View() string {
	if m.quitting {
		return ""
	}

	// Show confirmation dialog if active
	if m.focus == focusConfirmClose {
		return m.renderConfirmDialog()
	}

	if m.config.Editor.EditorInMainWindow {
		if m.fileOpen && !m.config.Editor.UseSystemEditor {
			return m.renderMainWindowView()
		}
		return m.renderFilePickerOnlyView()
	}

	return m.renderSplitView()
}

func (m EditorModel) renderConfirmDialog() string {
	// Dialog content
	fileName := filepath.Base(m.currentFile)
	title := editorDialogTitleStyle.Render("⚠ Unsaved Changes")
	message := fmt.Sprintf("\nFile '%s' has unsaved changes.\n\nDo you want to save before closing?\n\n", fileName)
	options := "[Y] Save  [N] Discard  [Esc] Cancel"

	dialogContent := title + message + editorHelpStyle.Render(options)
	dialog := editorDialogStyle.Width(46).Render(dialogContent)

	// Use lipgloss.Place to center the dialog in the terminal
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

func (m EditorModel) renderMainWindowView() string {
	var b strings.Builder

	// Title
	title := editorTitleStyle.Render("📝 " + filepath.Base(m.currentFile))
	if m.hasChanges {
		title += " [modified]"
	}
	b.WriteString(title + "\n\n")

	// Editor
	b.WriteString(m.textArea.View())
	b.WriteString("\n\n")

	// Status
	b.WriteString(m.renderStatus())
	b.WriteString("\n")

	// Help
	help := fmt.Sprintf("%s: save • %s: close • %s: quit",
		m.config.Editor.Hotkeys.Save,
		m.config.Editor.Hotkeys.CloseFile,
		m.config.Editor.Hotkeys.Quit)
	b.WriteString(editorHelpStyle.Render(help))

	return b.String()
}

func (m EditorModel) renderFilePickerOnlyView() string {
	var b strings.Builder

	// Title
	b.WriteString(editorTitleStyle.Render("📝 cbranotes editor") + "\n")

	// Calculate dimensions
	paneHeight := m.height - 5
	if paneHeight < 6 {
		paneHeight = 6
	}
	innerHeight := paneHeight - 2

	// Full width pane for file picker
	paneWidth := m.width - 2
	if paneWidth < 30 {
		paneWidth = 30
	}

	// File picker content - truncate to fit
	fpMaxLines := innerHeight - 2
	if fpMaxLines < 3 {
		fpMaxLines = 3
	}
	fpContent := truncateHeight(m.filePicker.View(), fpMaxLines)

	// Search bar
	searchLabel := editorSearchStyle.Render("🔍 ")
	searchBar := searchLabel + m.searchInput.View()

	content := fpContent + "\n" + searchBar

	// Style the pane
	paneStyle := editorActivePaneStyle
	pane := paneStyle.Width(paneWidth).Height(paneHeight).Render(content)

	b.WriteString(pane)
	b.WriteString("\n")

	// Status
	b.WriteString(m.renderStatus())

	// Help
	help := fmt.Sprintf("enter: open file • /: search • %s: quit",
		m.config.Editor.Hotkeys.Quit)
	b.WriteString(editorHelpStyle.Render(help))

	return b.String()
}

func (m EditorModel) renderSplitView() string {
	var b strings.Builder

	// Title
	b.WriteString(editorTitleStyle.Render("📝 cbranotes editor") + "\n")

	// Calculate dimensions
	// Reserve lines for: title(1), status(1), help(1) = 3 lines outside panes
	// Pane borders take 2 lines (top + bottom)
	// Inner content height = total height - outside lines - border lines
	paneHeight := m.height - 5
	if paneHeight < 6 {
		paneHeight = 6
	}
	innerHeight := paneHeight - 2 // Account for top and bottom borders

	// Calculate widths - ensure both panes fit within terminal
	// Leave 1 char gap between panes
	availableWidth := m.width - 1
	leftWidth := availableWidth / 3
	if leftWidth < 20 {
		leftWidth = 20
	}
	rightWidth := availableWidth - leftWidth
	if rightWidth < 20 {
		rightWidth = 20
	}
	// Cap to prevent overflow
	if leftWidth+rightWidth > m.width {
		leftWidth = m.width / 2
		rightWidth = m.width - leftWidth - 1
	}

	// Left pane: file picker + search
	leftPaneStyle := editorPaneStyle
	if m.focus == focusFilePicker || m.focus == focusSearch {
		leftPaneStyle = editorActivePaneStyle
	}

	// File picker content - truncate to fit
	// Reserve 2 lines for search bar at bottom
	fpMaxLines := innerHeight - 2
	if fpMaxLines < 3 {
		fpMaxLines = 3
	}
	fpContent := truncateHeight(m.filePicker.View(), fpMaxLines)

	// Search bar
	searchLabel := editorSearchStyle.Render("🔍 ")
	searchBar := searchLabel + m.searchInput.View()

	leftContent := fpContent + "\n" + searchBar
	leftPane := leftPaneStyle.Width(leftWidth).Height(paneHeight).Render(leftContent)

	// Right pane: editor
	rightPaneStyle := editorPaneStyle
	if m.focus == focusEditor {
		rightPaneStyle = editorActivePaneStyle
	}

	var rightContent string
	if m.fileOpen {
		fileTitle := filepath.Base(m.currentFile)
		if m.hasChanges {
			fileTitle += " [modified]"
		}
		// Truncate editor content to fit
		taContent := truncateHeight(m.textArea.View(), innerHeight-1)
		rightContent = editorTitleStyle.Render(fileTitle) + "\n" + taContent
	} else {
		rightContent = editorStatusStyle.Render("Select a file to edit...")
	}
	rightPane := rightPaneStyle.Width(rightWidth).Height(paneHeight).Render(rightContent)

	// Combine panes
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane))
	b.WriteString("\n")

	// Status
	b.WriteString(m.renderStatus())

	// Help
	help := fmt.Sprintf("tab: switch pane • /: search • %s: save • %s: close • %s: quit",
		m.config.Editor.Hotkeys.Save,
		m.config.Editor.Hotkeys.CloseFile,
		m.config.Editor.Hotkeys.Quit)
	b.WriteString(editorHelpStyle.Render(help))

	return b.String()
}

func (m EditorModel) renderStatus() string {
	if m.statusMsg == "" {
		return ""
	}

	if strings.HasPrefix(m.statusMsg, "Error") || strings.HasPrefix(m.statusMsg, "Save failed") {
		return editorStatusErrorStyle.Render(m.statusMsg)
	}
	if m.statusMsg == "Saved!" {
		return editorStatusSuccessStyle.Render("✓ " + m.statusMsg)
	}
	return editorStatusStyle.Render(m.statusMsg)
}
