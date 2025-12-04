package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	subtleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("78"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
)

// Setup model for first-run configuration
type SetupModel struct {
	repoInput  textinput.Model
	pathInput  textinput.Model
	focusIndex int
	done       bool
	RepoURL    string
	NotesPath  string
	defaultPath string
}

func NewSetupModel(defaultPath string) SetupModel {
	ri := textinput.New()
	ri.Placeholder = "git@github.com:user/notes.git"
	ri.Focus()
	ri.Width = 50

	pi := textinput.New()
	pi.Placeholder = defaultPath
	pi.Width = 50

	return SetupModel{
		repoInput:   ri,
		pathInput:   pi,
		defaultPath: defaultPath,
	}
}

func (m SetupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "tab", "down":
			m.focusIndex = (m.focusIndex + 1) % 2
			if m.focusIndex == 0 {
				m.repoInput.Focus()
				m.pathInput.Blur()
			} else {
				m.repoInput.Blur()
				m.pathInput.Focus()
			}
			return m, nil
		case "shift+tab", "up":
			m.focusIndex = (m.focusIndex + 1) % 2
			if m.focusIndex == 0 {
				m.repoInput.Focus()
				m.pathInput.Blur()
			} else {
				m.repoInput.Blur()
				m.pathInput.Focus()
			}
			return m, nil
		case "enter":
			if m.focusIndex == 1 || m.repoInput.Value() != "" {
				m.done = true
				m.RepoURL = m.repoInput.Value()
				m.NotesPath = m.pathInput.Value()
				if m.NotesPath == "" {
					m.NotesPath = m.defaultPath
				}
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	if m.focusIndex == 0 {
		m.repoInput, cmd = m.repoInput.Update(msg)
	} else {
		m.pathInput, cmd = m.pathInput.Update(msg)
	}
	return m, cmd
}

func (m SetupModel) View() string {
	s := titleStyle.Render("cbranotes setup") + "\n\n"

	s += "Repository URL:\n"
	s += m.repoInput.View() + "\n\n"

	s += "Notes directory:\n"
	s += m.pathInput.View() + "\n"
	s += subtleStyle.Render(fmt.Sprintf("  (default: %s)", m.defaultPath)) + "\n\n"

	s += subtleStyle.Render("tab: next field • enter: confirm • esc: quit")

	return s
}

func (m SetupModel) Done() bool {
	return m.done
}

// Spinner model for sync operations
type SpinnerModel struct {
	spinner  spinner.Model
	message  string
	done     bool
	err      error
	result   string
	action   func() error
}

type doneMsg struct {
	err error
}

func NewSpinnerModel(message string, action func() error) SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return SpinnerModel{
		spinner: s,
		message: message,
		action:  action,
	}
}

func (m SpinnerModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.runAction())
}

func (m SpinnerModel) runAction() tea.Cmd {
	return func() tea.Msg {
		err := m.action()
		return doneMsg{err: err}
	}
}

func (m SpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case doneMsg:
		m.done = true
		m.err = msg.err
		if m.err != nil {
			m.result = errorStyle.Render("✗ " + m.err.Error())
		} else {
			m.result = successStyle.Render("✓ " + m.message + " complete")
		}
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m SpinnerModel) View() string {
	if m.done {
		return m.result + "\n"
	}
	return m.spinner.View() + " " + m.message + "...\n"
}

func (m SpinnerModel) Err() error {
	return m.err
}

