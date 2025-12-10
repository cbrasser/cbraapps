package tui

import (
	"cbrateach/internal/config"
	"cbrateach/internal/models"
	"cbrateach/internal/storage"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

type viewState int

const (
	listView viewState = iota
	classbookView
	reviewFormView
	testListView
	testReviewView
	importTestView
	fileRenameView
	testDataView
)

type Model struct {
	cfg     config.Config
	storage *storage.Storage
	courses []models.Course
	tests   []models.Test

	// State
	state           viewState
	selectedCourse  int
	selectedStudent int
	selectedTest    int
	cursor          int

	// Test review state
	editingCell   bool
	editingGifted bool
	selectedRow   int
	selectedCol   int
	editValue     string

	// Import state
	importStep           int // 0: Select File, 1: Details, 2: Match
	importFilePickerForm *huh.Form
	importFile           string
	importData           *storage.JSONImport
	importMatches        map[string]string // key -> studentName (for matched)
	importUnmatched      []string          // keys (for unmatched)
	importCandidates     []string          // list of course students available
	importCursor         int               // Cursor for lists
	importMatchFocus     bool              // True if selecting candidate

	// Import Details
	importName   string
	importTopic  string
	importWeight string

	// File rename state
	fileRenameState fileRenameState

	// Incognito mode (hide student names)
	incognitoMode bool

	// Confirmation dialog state
	showingConfirmation  bool
	confirmationTitle    string
	confirmationMessage  string
	confirmationCallback func(Model) (Model, tea.Cmd)

	// UI dimensions
	width  int
	height int

	// Error handling
	err error
}

func NewModel(cfg config.Config) Model {
	store := storage.New(cfg)
	courses, _ := store.LoadCourses()

	return Model{
		cfg:     cfg,
		storage: store,
		courses: courses,
		state:   listView,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Pass window size to import form if active
		if m.state == importTestView && m.importFilePickerForm != nil {
			form, cmd := m.importFilePickerForm.Update(msg)
			if f, ok := form.(*huh.Form); ok {
				m.importFilePickerForm = f
			}
			return m, cmd
		}
		return m, nil

	case loadTestsMsg:
		m.tests = msg
		m.state = testListView // Ensure we are in test list view? Or stay in current if appropriate.
		return m, nil

	case testImportedMsg:
		// Test was imported, reload tests and go to test list
		m.state = testListView
		return m, m.loadTestsCmd(msg.courseID)

	case tea.KeyMsg:
		// Handle confirmation dialog if showing
		if m.showingConfirmation {
			return m.updateConfirmationDialog(msg)
		}

		switch m.state {
		case listView:
			return m.updateListView(msg)
		case classbookView:
			return m.updateClassbookView(msg)
		case testListView:
			return m.updateTestListView(msg)
		case testReviewView:
			return m.updateTestReviewView(msg)
		case importTestView:
			return m.updateImportView(msg)
		case fileRenameView:
			return m.updateFileRenameView(msg)
		case testDataView:
			return m.updateTestDataView(msg)
		}
	}

	// Pass all other non-key messages to import view if active
	// (Huh forms need mouse events, window size, etc.)
	if m.state == importTestView && m.importStep == 0 {
		return m.updateImportViewGeneric(msg)
	}

	return m, nil
}

func (m Model) View() string {
	if m.err != nil {
		return errorStyle.Render("Error: " + m.err.Error())
	}

	// Show confirmation dialog on top if active
	if m.showingConfirmation {
		return m.renderConfirmationDialog()
	}

	switch m.state {
	case listView:
		return m.renderListView()
	case classbookView:
		return m.renderClassbookView()
	case testListView:
		return m.renderTestListView()
	case importTestView:
		return m.renderImportView()
	case testReviewView:
		return m.renderTestReviewView()
	case fileRenameView:
		return m.renderFileRenameView()
	case testDataView:
		return m.renderTestDataView()
	default:
		return "Unknown view"
	}
}

type loadCoursesMsg []models.Course

func loadCourses(store *storage.Storage) tea.Cmd {
	return func() tea.Msg {
		courses, _ := store.LoadCourses()
		return loadCoursesMsg(courses)
	}
}

type saveCoursesMsg struct{}

func (m Model) saveCourses() tea.Cmd {
	return func() tea.Msg {
		m.storage.SaveCourses(m.courses)
		return saveCoursesMsg{}
	}
}

type loadTestsMsg []models.Test

func (m Model) loadTestsCmd(courseID string) tea.Cmd {
	return func() tea.Msg {
		tests, err := m.storage.LoadTests(courseID)
		if err != nil {
			return loadTestsMsg{}
		}
		return loadTestsMsg(tests)
	}
}

// Confirmation dialog helpers
func (m Model) updateConfirmationDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		// User confirmed
		m.showingConfirmation = false
		if m.confirmationCallback != nil {
			return m.confirmationCallback(m)
		}
		return m, nil

	case "n", "N", "esc":
		// User cancelled
		m.showingConfirmation = false
		m.confirmationCallback = nil
		return m, nil
	}

	return m, nil
}

func (m Model) renderConfirmationDialog() string {
	var content strings.Builder

	content.WriteString("\n\n")
	content.WriteString(titleStyle.Render(m.confirmationTitle) + "\n\n")
	content.WriteString(m.confirmationMessage + "\n\n")
	content.WriteString(helpStyle.Render("y: Yes • n: No • esc: Cancel"))

	return baseStyle.Render(content.String())
}
