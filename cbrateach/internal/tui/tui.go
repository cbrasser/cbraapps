package tui

import (
	"cbrateach/internal/config"
	"cbrateach/internal/models"
	"cbrateach/internal/storage"

	tea "github.com/charmbracelet/bubbletea"
)

type viewState int

const (
	listView viewState = iota
	classbookView
	reviewFormView
	testListView
	testReviewView
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
	editingCell     bool
	editingGifted   bool
	selectedRow     int
	selectedCol     int
	editValue       string

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
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case listView:
			return m.updateListView(msg)
		case classbookView:
			return m.updateClassbookView(msg)
		case testListView:
			return m.updateTestListView(msg)
		case testReviewView:
			return m.updateTestReviewView(msg)
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.err != nil {
		return errorStyle.Render("Error: " + m.err.Error())
	}

	switch m.state {
	case listView:
		return m.renderListView()
	case classbookView:
		return m.renderClassbookView()
	case testListView:
		return m.renderTestListView()
	case testReviewView:
		return m.renderTestReviewView()
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
