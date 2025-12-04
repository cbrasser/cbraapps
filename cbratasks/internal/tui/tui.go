package tui

import (
	"fmt"
	"strings"

	"cbratasks/internal/config"
	"cbratasks/internal/storage"
	"cbratasks/internal/task"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type viewState int

const (
	viewList viewState = iota
	viewSearch
	viewAddTask
	viewEditNote
	viewViewNote
)

// Messages
type syncDoneMsg struct {
	err error
}

type initialSyncDoneMsg struct {
	err error
}

type startSyncMsg struct{}

type Model struct {
	config      *config.Config
	storage     *storage.Storage
	tasks       []*task.Task
	cursor      int
	view        viewState
	searchInput textinput.Model
	addInput    textinput.Model
	noteArea    textarea.Model
	editingTask *task.Task
	viewingTask *task.Task
	spinner     spinner.Model
	syncing     bool
	width       int
	height      int
	statusMsg   string
	quitting    bool
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF79C6")).
			Bold(true).
			MarginBottom(1)

	taskStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2"))

	completedTaskStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6272A4")).
				Strikethrough(true)

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#44475A")).
			Foreground(lipgloss.Color("#F8F8F2"))

	checkboxStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BD93F9"))

	checkboxDoneStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#50FA7B"))

	overdueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")).
			Bold(true)

	dueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB86C"))

	tagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#282A36")).
			Padding(0, 1)

	noteIndicatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8BE9FD"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			MarginTop(1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50FA7B"))

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#BD93F9")).
			Padding(0, 1)

	noteBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#8BE9FD")).
			Padding(1)

	spinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF79C6"))
)

func NewModel(cfg *config.Config, store *storage.Storage) Model {
	// Search input
	si := textinput.New()
	si.Placeholder = "Search tasks..."
	si.Width = 40

	// Add task input
	ai := textinput.New()
	ai.Placeholder = "Task title (+tag for tags, +1d for due)"
	ai.Width = 50

	// Note textarea
	na := textarea.New()
	na.Placeholder = "Add a note..."
	na.ShowLineNumbers = false
	na.SetWidth(50)
	na.SetHeight(5)

	// Spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = spinnerStyle

	return Model{
		config:      cfg,
		storage:     store,
		tasks:       store.GetTasks(),
		searchInput: si,
		addInput:    ai,
		noteArea:    na,
		spinner:     sp,
	}
}

func (m Model) Init() tea.Cmd {
	// Sync on startup if enabled
	if m.storage.IsSyncEnabled() {
		return func() tea.Msg {
			return startSyncMsg{}
		}
	}
	return nil
}

func (m Model) doInitialSync() tea.Cmd {
	return func() tea.Msg {
		err := m.storage.Sync()
		return initialSyncDoneMsg{err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.noteArea.SetWidth(min(60, m.width-10))
		m.noteArea.SetHeight(min(10, m.height-15))

	case spinner.TickMsg:
		if m.syncing {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case syncDoneMsg:
		m.syncing = false
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Sync failed: %v", msg.err)
		} else {
			m.tasks = m.storage.GetTasks()
			m.statusMsg = "✓ Sync complete!"
		}

	case initialSyncDoneMsg:
		m.syncing = false
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Sync failed: %v", msg.err)
		} else {
			m.tasks = m.storage.GetTasks()
			m.statusMsg = "✓ Synced from server"
		}

	case startSyncMsg:
		m.syncing = true
		m.statusMsg = ""
		return m, tea.Batch(m.spinner.Tick, m.doInitialSync())

	case tea.KeyMsg:
		key := msg.String()

		// Handle different views
		switch m.view {
		case viewSearch:
			return m.handleSearchInput(msg)
		case viewAddTask:
			return m.handleAddInput(msg)
		case viewEditNote:
			return m.handleNoteInput(msg)
		case viewViewNote:
			return m.handleViewNote(msg)
		}

		// List view keybindings
		switch key {
		case m.config.Hotkeys.Quit, "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
			}

		case "x":
			if len(m.tasks) > 0 && m.cursor < len(m.tasks) {
				t := m.tasks[m.cursor]
				taskID := t.ID
				wasCompleted := t.Completed
				
				// Start sync spinner if this is a radicale task
				if t.ListName == "radicale" && m.storage.IsSyncEnabled() {
					m.syncing = true
					cmds = append(cmds, m.spinner.Tick)
				}
				
				m.storage.ToggleCompleteWithSync(taskID)
				m.tasks = m.storage.GetTasks()

				// Find the task's new position after re-sorting
				for i, tsk := range m.tasks {
					if tsk.ID == taskID {
						m.cursor = i
						break
					}
				}

				if !wasCompleted {
					m.statusMsg = "✓ Task completed!"
				} else {
					m.statusMsg = "Task reopened"
				}
				
				m.syncing = false
			}

		case m.config.Hotkeys.Delete:
			if len(m.tasks) > 0 && m.cursor < len(m.tasks) {
				t := m.tasks[m.cursor]
				m.storage.DeleteTaskWithSync(t.ID)
				m.tasks = m.storage.GetTasks()
				if m.cursor >= len(m.tasks) && m.cursor > 0 {
					m.cursor--
				}
				m.statusMsg = "Task deleted"
			}

		case m.config.Hotkeys.EditNote:
			if len(m.tasks) > 0 && m.cursor < len(m.tasks) {
				t := m.tasks[m.cursor]
				m.editingTask = t
				m.noteArea.SetValue(t.Note)
				m.noteArea.Focus()
				m.view = viewEditNote
				return m, textarea.Blink
			}

		case m.config.Hotkeys.ViewNote:
			// Tab - view note if task has one
			if len(m.tasks) > 0 && m.cursor < len(m.tasks) {
				t := m.tasks[m.cursor]
				if t.HasNote() {
					m.viewingTask = t
					m.view = viewViewNote
				}
			}

		case m.config.Hotkeys.Search:
			m.view = viewSearch
			m.searchInput.Focus()
			return m, textinput.Blink

		case m.config.Hotkeys.AddTask:
			m.view = viewAddTask
			m.addInput.SetValue("")
			m.addInput.Focus()
			return m, textinput.Blink

		case "s":
			// Manual sync
			if m.storage.IsSyncEnabled() && !m.syncing {
				m.syncing = true
				m.statusMsg = ""
				return m, tea.Batch(m.spinner.Tick, m.doSync())
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) doSync() tea.Cmd {
	return func() tea.Msg {
		err := m.storage.Sync()
		return syncDoneMsg{err: err}
	}
}

func (m Model) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		m.view = viewList
		m.searchInput.SetValue("")
		m.searchInput.Blur()
		m.tasks = m.storage.GetTasks()
		return m, nil

	case "enter":
		// Keep search results, just close input
		m.view = viewList
		m.searchInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)

	// Live search
	m.tasks = m.storage.Search(m.searchInput.Value())
	m.cursor = 0

	return m, cmd
}

func (m Model) handleAddInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		m.view = viewList
		m.addInput.SetValue("")
		m.addInput.Blur()
		return m, nil

	case "enter":
		input := strings.TrimSpace(m.addInput.Value())
		if input == "" {
			m.view = viewList
			m.addInput.Blur()
			return m, nil
		}

		// Parse the input for title, tags, and due date
		newTask := m.parseTaskInput(input)
		m.storage.AddTaskWithSync(newTask)
		m.tasks = m.storage.GetTasks()
		m.statusMsg = fmt.Sprintf("Added: %s", newTask.Title)

		m.view = viewList
		m.addInput.SetValue("")
		m.addInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.addInput, cmd = m.addInput.Update(msg)
	return m, cmd
}

func (m Model) handleNoteInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		// Save note and exit
		if m.editingTask != nil {
			m.editingTask.SetNote(m.noteArea.Value())
			m.storage.UpdateTask(m.editingTask)
			if m.editingTask.ListName == "radicale" {
				m.storage.PushTask(m.editingTask)
			}
			m.tasks = m.storage.GetTasks()
			m.statusMsg = "Note saved"
		}
		m.view = viewList
		m.editingTask = nil
		m.noteArea.Blur()
		return m, nil

	case "ctrl+s":
		// Save note explicitly
		if m.editingTask != nil {
			m.editingTask.SetNote(m.noteArea.Value())
			m.storage.UpdateTask(m.editingTask)
			if m.editingTask.ListName == "radicale" {
				m.storage.PushTask(m.editingTask)
			}
			m.tasks = m.storage.GetTasks()
			m.statusMsg = "Note saved"
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.noteArea, cmd = m.noteArea.Update(msg)
	return m, cmd
}

func (m Model) handleViewNote(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc", "tab", "enter", "q":
		m.view = viewList
		m.viewingTask = nil
		return m, nil

	case m.config.Hotkeys.EditNote:
		// Switch to edit mode
		if m.viewingTask != nil {
			m.editingTask = m.viewingTask
			m.noteArea.SetValue(m.viewingTask.Note)
			m.noteArea.Focus()
			m.viewingTask = nil
			m.view = viewEditNote
			return m, textarea.Blink
		}
	}

	return m, nil
}

// parseTaskInput parses input like "Buy milk +shopping +1d"
func (m Model) parseTaskInput(input string) *task.Task {
	parts := strings.Fields(input)
	var titleParts []string
	var tags []string
	var dueStr string

	for _, part := range parts {
		if strings.HasPrefix(part, "+") {
			suffix := part[1:]
			// Check if it's a date pattern
			if _, err := task.ParseDueDate(suffix); err == nil {
				dueStr = suffix
			} else {
				// It's a tag
				tags = append(tags, suffix)
			}
		} else {
			titleParts = append(titleParts, part)
		}
	}

	title := strings.Join(titleParts, " ")
	newTask := task.NewTask(title, m.config.DefaultList)

	for _, tag := range tags {
		newTask.AddTag(tag)
	}

	if dueStr != "" {
		if due, err := task.ParseDueDate(dueStr); err == nil {
			newTask.SetDueDate(*due)
		}
	}

	return newTask
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("📋 Tasks") + "\n\n")

	// Search bar (if active)
	if m.view == viewSearch {
		b.WriteString(inputStyle.Render("🔍 " + m.searchInput.View()) + "\n\n")
	}

	// Add task form (if active)
	if m.view == viewAddTask {
		b.WriteString(inputStyle.Render("➕ " + m.addInput.View()) + "\n")
		b.WriteString(helpStyle.Render("  +tag for tags, +1d/+1w/tomorrow for due") + "\n\n")
	}

	// Note editor (if active)
	if m.view == viewEditNote && m.editingTask != nil {
		b.WriteString(titleStyle.Render("📝 Note for: " + m.editingTask.Title) + "\n")
		b.WriteString(noteBoxStyle.Render(m.noteArea.View()) + "\n")
		b.WriteString(helpStyle.Render("  esc: save & close • ctrl+s: save") + "\n\n")
		return b.String()
	}

	// Note viewer (if active)
	if m.view == viewViewNote && m.viewingTask != nil {
		b.WriteString(titleStyle.Render("📝 Note for: " + m.viewingTask.Title) + "\n")
		b.WriteString(noteBoxStyle.Render(m.viewingTask.Note) + "\n")
		b.WriteString(helpStyle.Render(fmt.Sprintf("  esc/tab: close • %s: edit", m.config.Hotkeys.EditNote)) + "\n\n")
		return b.String()
	}

	// Task list
	if len(m.tasks) == 0 {
		b.WriteString(helpStyle.Render("  No tasks. Press 'a' to add one.") + "\n")
	} else {
		for i, t := range m.tasks {
			line := m.renderTask(t, i == m.cursor)
			b.WriteString(line + "\n")
		}
	}

	b.WriteString("\n")

	// Syncing spinner
	if m.syncing {
		b.WriteString(m.spinner.View() + " Syncing...\n")
	}

	// Status message
	if m.statusMsg != "" && !m.syncing {
		b.WriteString(statusStyle.Render(m.statusMsg) + "\n")
	}

	// Help
	help := fmt.Sprintf("x: toggle • %s: note • %s: view note • %s: add • %s: search • s: sync • %s: quit",
		m.config.Hotkeys.EditNote,
		m.config.Hotkeys.ViewNote,
		m.config.Hotkeys.AddTask,
		m.config.Hotkeys.Search,
		m.config.Hotkeys.Quit)
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func (m Model) renderTask(t *task.Task, selected bool) string {
	// Markdown-style checkbox
	var checkbox string
	if t.Completed {
		checkbox = checkboxDoneStyle.Render("[x]")
	} else {
		checkbox = checkboxStyle.Render("[ ]")
	}

	// Title
	title := t.Title
	var titleRendered string
	if t.Completed {
		titleRendered = completedTaskStyle.Render(title)
	} else {
		titleRendered = taskStyle.Render(title)
	}

	// Note indicator
	noteIndicator := ""
	if t.HasNote() {
		noteIndicator = noteIndicatorStyle.Render(" 📝")
	}

	// Due date
	dueStr := ""
	if t.DueDate != nil && !t.Completed {
		if t.IsOverdue() {
			dueStr = overdueStyle.Render(" ⚠ " + t.DueString())
		} else {
			dueStr = dueStyle.Render(" · " + t.DueString())
		}
	}

	// Tags
	var tagParts []string
	for _, tag := range t.Tags {
		color := m.config.GetTagColor(tag)
		tagParts = append(tagParts, tagStyle.Background(lipgloss.Color(color)).Render(tag))
	}
	tags := ""
	if len(tagParts) > 0 {
		tags = " " + strings.Join(tagParts, " ")
	}

	// Combine
	line := fmt.Sprintf("  %s %s%s%s%s", checkbox, titleRendered, noteIndicator, dueStr, tags)

	if selected {
		// Highlight the whole line
		line = selectedStyle.Render(line)
	}

	return line
}

func Run(cfg *config.Config, store *storage.Storage) error {
	m := NewModel(cfg, store)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
