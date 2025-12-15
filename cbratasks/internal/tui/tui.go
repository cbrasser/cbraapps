package tui

import (
	"fmt"
	"strings"
	"time"

	"cbratasks/internal/config"
	"cbratasks/internal/storage"
	"cbratasks/internal/task"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

type viewState int

const (
	viewList viewState = iota
	viewSearch
	viewAddTask
	viewEditTask
	viewEditNote
	viewViewNote
	viewFocus
	viewArchive
)

// Messages
type syncDoneMsg struct {
	err error
}

type initialSyncDoneMsg struct {
	err error
}

type startSyncMsg struct{}

// focusKeyMap defines keybindings for focus mode
type focusKeyMap struct {
	Complete key.Binding
	Exit     key.Binding
	Up       key.Binding
	Down     key.Binding
	Filter   key.Binding
	Help     key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view.
func (k focusKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Exit, k.Help}
}

// FullHelp returns keybindings for the expanded help view.
func (k focusKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Complete, k.Filter, k.Exit},
	}
}

var focusKeys = focusKeyMap{
	Complete: key.NewBinding(
		key.WithKeys("enter", "x", " "),
		key.WithHelp("enter/x/space", "complete task"),
	),
	Exit: key.NewBinding(
		key.WithKeys("q", "esc", "f"),
		key.WithHelp("q", "quit focus mode"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "more"),
	),
}

// listKeyMap defines keybindings for main list view
type listKeyMap struct {
	Toggle      key.Binding
	Delete      key.Binding
	AddTask     key.Binding
	EditTask    key.Binding
	Search      key.Binding
	EditNote    key.Binding
	ViewNote    key.Binding
	Focus       key.Binding
	Archive     key.Binding
	ArchiveAll  key.Binding
	ViewArchive key.Binding
	Sync        key.Binding
	Quit        key.Binding
	Help        key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view.
func (k listKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Help}
}

// FullHelp returns keybindings for the expanded help view.
func (k listKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Toggle, k.AddTask, k.EditTask, k.Search, k.Focus},
		{k.Archive, k.ArchiveAll, k.ViewArchive, k.Sync},
		{k.EditNote, k.ViewNote, k.Delete, k.Quit},
	}
}

var listKeys = listKeyMap{
	Toggle: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "toggle complete"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	AddTask: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "add task"),
	),
	EditTask: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit task"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	EditNote: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "edit note"),
	),
	ViewNote: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "view note"),
	),
	Focus: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "focus mode"),
	),
	Archive: key.NewBinding(
		key.WithKeys("z"),
		key.WithHelp("z", "archive task"),
	),
	ArchiveAll: key.NewBinding(
		key.WithKeys("Z"),
		key.WithHelp("Z", "archive all"),
	),
	ViewArchive: key.NewBinding(
		key.WithKeys("A"),
		key.WithHelp("A", "view archive"),
	),
	Sync: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "sync"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "more"),
	),
}

// archiveKeyMap defines keybindings for archive view
type archiveKeyMap struct {
	ViewArchive key.Binding
	Filter      key.Binding
	Quit        key.Binding
	Help        key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view.
func (k archiveKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Help}
}

// FullHelp returns keybindings for the expanded help view.
func (k archiveKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.ViewArchive, k.Filter, k.Quit},
	}
}

var archiveKeys = archiveKeyMap{
	ViewArchive: key.NewBinding(
		key.WithKeys("A"),
		key.WithHelp("A", "back to tasks"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "more"),
	),
}

// focusItem implements list.Item for the focus mode list
type focusItem struct {
	task *task.Task
}

func (i focusItem) FilterValue() string { return i.task.Title }
func (i focusItem) Title() string       { return i.task.Title }
func (i focusItem) Description() string {
	parts := []string{}
	if i.task.DueDate != nil {
		parts = append(parts, i.task.DueString())
	}
	if len(i.task.Tags) > 0 {
		parts = append(parts, strings.Join(i.task.Tags, ", "))
	}
	return strings.Join(parts, " • ")
}

// archiveItem implements list.Item for the archive list
type archiveItem struct {
	task *task.Task
}

func (i archiveItem) FilterValue() string { return i.task.Title }
func (i archiveItem) Title() string       { return i.task.Title }
func (i archiveItem) Description() string {
	parts := []string{}
	if i.task.CompletedAt != nil {
		parts = append(parts, "Completed: "+i.task.CompletedAt.Format("Jan 02, 2006"))
	}
	if len(i.task.Tags) > 0 {
		parts = append(parts, strings.Join(i.task.Tags, ", "))
	}
	return strings.Join(parts, " • ")
}

type Model struct {
	config      *config.Config
	storage     *storage.Storage
	tasks       []*task.Task
	cursor      int
	view        viewState
	searchInput textinput.Model
	addInput    textinput.Model
	noteArea    textarea.Model
	editForm    *huh.Form
	editingTask *task.Task
	viewingTask *task.Task
	spinner     spinner.Model
	syncing     bool
	width       int
	height      int
	statusMsg   string
	quitting    bool
	showArchive bool
	focusList   list.Model
	focusHelp   help.Model
	listHelp    help.Model
	archiveList list.Model
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

	// Focus list
	fl := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	fl.Title = "Focus Mode"
	fl.SetShowStatusBar(false)
	fl.SetFilteringEnabled(true)
	fl.Styles.Title = titleStyle

	// Archive list
	al := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	al.Title = "Archive"
	al.SetShowStatusBar(false)
	al.SetFilteringEnabled(true)
	al.Styles.Title = titleStyle

	// Focus help
	fh := help.New()
	fh.ShowAll = false

	// List help
	lh := help.New()
	lh.ShowAll = false

	return Model{
		config:      cfg,
		storage:     store,
		tasks:       store.GetTasks(),
		searchInput: si,
		addInput:    ai,
		noteArea:    na,
		spinner:     sp,
		focusList:   fl,
		focusHelp:   fh,
		listHelp:    lh,
		archiveList: al,
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

// getFocusTasks returns tasks due today, tomorrow, or overdue (incomplete only)
func (m Model) getFocusTasks() []*task.Task {
	var focusTasks []*task.Task
	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1)

	for _, t := range m.tasks {
		if t.Completed {
			continue
		}
		if t.DueDate == nil {
			continue
		}

		due := *t.DueDate
		// Check if overdue, due today, or due tomorrow
		if due.Before(now) ||
		   (due.Year() == now.Year() && due.YearDay() == now.YearDay()) ||
		   (due.Year() == tomorrow.Year() && due.YearDay() == tomorrow.YearDay()) {
			focusTasks = append(focusTasks, t)
		}
	}

	return focusTasks
}

// enterFocusMode sets up the focus mode view
func (m *Model) enterFocusMode() {
	focusTasks := m.getFocusTasks()
	items := make([]list.Item, len(focusTasks))
	for i, t := range focusTasks {
		items[i] = focusItem{task: t}
	}
	m.focusList.SetItems(items)
	m.focusList.SetSize(m.width, m.height-4)
	m.view = viewFocus
}

// enterArchiveMode sets up the archive view with list component
func (m *Model) enterArchiveMode() {
	archivedTasks := m.storage.GetArchivedTasks()
	items := make([]list.Item, len(archivedTasks))
	for i, t := range archivedTasks {
		items[i] = archiveItem{task: t}
	}
	m.archiveList.SetItems(items)
	m.archiveList.SetSize(m.width, m.height-4)
}

// initEditForm initializes the edit form for a task
func (m *Model) initEditForm(t *task.Task) {
	// Prepare initial values
	editTitle := t.Title
	editTags := strings.Join(t.Tags, ", ")
	editDueDate := ""
	if t.DueDate != nil {
		editDueDate = t.DueDate.Format("2006-01-02")
	}

	// Create the form
	m.editForm = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Task Title").
				Value(&editTitle).
				Key("title"),

			huh.NewInput().
				Title("Tags (comma-separated)").
				Value(&editTags).
				Placeholder("work, important").
				Key("tags"),

			huh.NewInput().
				Title("Due Date").
				Value(&editDueDate).
				Placeholder("YYYY-MM-DD, today, tomorrow, +1d, +1w").
				Key("duedate"),
		),
	)

	m.editingTask = t
}

func (m Model) doInitialSync() tea.Cmd {
	return func() tea.Msg {
		err := m.storage.Sync()
		return initialSyncDoneMsg{err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle edit form updates first if we're in edit mode
	if m.view == viewEditTask && m.editForm != nil {
		// Check for ESC to cancel before updating form
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
			m.view = viewList
			m.editForm = nil
			m.editingTask = nil
			return m, nil
		}

		form, cmd := m.editForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.editForm = f
		}

		// Check if form is complete
		if m.editForm.State == huh.StateCompleted {
			// Update task using the model field values
			if m.editingTask != nil {
				// Get values from form using Get methods
				newTitle := m.editForm.GetString("title")
				newTags := m.editForm.GetString("tags")
				newDueDate := m.editForm.GetString("duedate")

				// Get the task from storage to ensure we have the latest version
				taskToUpdate := m.storage.GetTask(m.editingTask.ID)
				if taskToUpdate == nil {
					m.statusMsg = "Error: task not found"
					m.view = viewList
					m.editForm = nil
					m.editingTask = nil
					return m, cmd
				}

				// Update the title
				taskToUpdate.Title = strings.TrimSpace(newTitle)

				// Parse and set tags
				taskToUpdate.Tags = []string{}
				if strings.TrimSpace(newTags) != "" {
					tagParts := strings.Split(newTags, ",")
					for _, tag := range tagParts {
						tag = strings.TrimSpace(tag)
						if tag != "" {
							taskToUpdate.Tags = append(taskToUpdate.Tags, strings.ToLower(tag))
						}
					}
				}

				// Parse and set due date
				dueDateTrimmed := strings.TrimSpace(newDueDate)
				if dueDateTrimmed != "" {
					if due, err := task.ParseDueDate(dueDateTrimmed); err == nil {
						taskToUpdate.DueDate = due
					} else {
						m.statusMsg = fmt.Sprintf("Invalid due date: %v", err)
						m.view = viewList
						m.editForm = nil
						m.editingTask = nil
						return m, cmd
					}
				} else {
					// Clear due date if empty
					taskToUpdate.DueDate = nil
				}

				// Save the task
				var err error
				if taskToUpdate.ListName == "radicale" && m.storage.IsSyncEnabled() {
					err = m.storage.UpdateTaskWithSync(taskToUpdate)
				} else {
					err = m.storage.UpdateTask(taskToUpdate)
				}

				if err != nil {
					m.statusMsg = fmt.Sprintf("Failed to update: %v", err)
				} else {
					m.statusMsg = fmt.Sprintf("✓ Updated: %s", taskToUpdate.Title)
				}

				// Reload tasks from storage
				m.tasks = m.storage.GetTasks()
			}

			// Return to list view and clear form state
			m.view = viewList
			m.editForm = nil
			m.editingTask = nil
			return m, cmd
		}

		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.noteArea.SetWidth(min(60, m.width-10))
		m.noteArea.SetHeight(min(10, m.height-15))
		m.focusList.SetSize(m.width, m.height-4)
		m.archiveList.SetSize(m.width, m.height-4)

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
		case viewFocus:
			return m.handleFocusMode(msg)
		case viewArchive:
			return m.handleArchiveMode(msg)
		}

		// List view keybindings
		switch key {
		case "?":
			// Toggle help
			m.listHelp.ShowAll = !m.listHelp.ShowAll
			return m, nil

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
				cursorPos := m.cursor

				// Start sync spinner if this is a radicale task
				if t.ListName == "radicale" && m.storage.IsSyncEnabled() {
					m.syncing = true
					cmds = append(cmds, m.spinner.Tick)
				}

				m.storage.ToggleCompleteWithSync(taskID)
				m.tasks = m.storage.GetTasks()

				if wasCompleted {
					// Task was completed, now it's undone - follow it to new position
					for i, tsk := range m.tasks {
						if tsk.ID == taskID {
							m.cursor = i
							break
						}
					}
					m.statusMsg = "Task reopened"
				} else {
					// Task was incomplete, now it's done - keep cursor at same position
					m.cursor = cursorPos
					if m.cursor >= len(m.tasks) && len(m.tasks) > 0 {
						m.cursor = len(m.tasks) - 1
					}
					m.statusMsg = "✓ Task completed!"
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

		case "e":
			if len(m.tasks) > 0 && m.cursor < len(m.tasks) {
				t := m.tasks[m.cursor]
				m.initEditForm(t)
				m.view = viewEditTask
				return m, m.editForm.Init()
			}

		case "s":
			// Manual sync
			if m.storage.IsSyncEnabled() && !m.syncing {
				m.syncing = true
				m.statusMsg = ""
				return m, tea.Batch(m.spinner.Tick, m.doSync())
			}

		case "z":
			// Archive single completed task
			if !m.showArchive && len(m.tasks) > 0 && m.cursor < len(m.tasks) {
				t := m.tasks[m.cursor]
				if t.Completed {
					if err := m.storage.ArchiveTask(t.ID); err == nil {
						m.tasks = m.storage.GetTasks()
						if m.cursor >= len(m.tasks) && m.cursor > 0 {
							m.cursor--
						}
						m.statusMsg = "✓ Task archived!"
					} else {
						m.statusMsg = fmt.Sprintf("Failed to archive: %v", err)
					}
				} else {
					m.statusMsg = "Only completed tasks can be archived"
				}
			}

		case "Z":
			// Archive all completed tasks
			if !m.showArchive {
				count, err := m.storage.ArchiveAllCompletedTasks()
				if err == nil {
					m.tasks = m.storage.GetTasks()
					m.cursor = 0
					m.statusMsg = fmt.Sprintf("✓ Archived %d completed task(s)", count)
				} else {
					m.statusMsg = fmt.Sprintf("Failed to archive: %v", err)
				}
			}

		case "A":
			// Toggle archive view
			m.showArchive = !m.showArchive
			if m.showArchive {
				m.enterArchiveMode()
				m.view = viewArchive
				m.statusMsg = "Viewing archive"
			} else {
				m.tasks = m.storage.GetTasks()
				m.statusMsg = "Viewing active tasks"
			}
			m.cursor = 0
			return m, nil

		case "f":
			// Enter focus mode
			m.enterFocusMode()
			return m, nil
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

func (m Model) handleFocusMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, focusKeys.Exit):
		// Exit focus mode
		m.view = viewList
		m.statusMsg = "Exited focus mode"
		return m, nil

	case key.Matches(msg, focusKeys.Help):
		// Toggle help
		m.focusHelp.ShowAll = !m.focusHelp.ShowAll
		return m, nil

	case key.Matches(msg, focusKeys.Complete):
		// Mark selected task as complete
		if selectedItem, ok := m.focusList.SelectedItem().(focusItem); ok {
			t := selectedItem.task

			// Start sync spinner if this is a radicale task
			if t.ListName == "radicale" && m.storage.IsSyncEnabled() {
				m.syncing = true
			}

			m.storage.ToggleCompleteWithSync(t.ID)
			m.tasks = m.storage.GetTasks()
			m.statusMsg = "✓ Task completed!"
			m.syncing = false

			// Refresh focus list with remaining tasks
			m.enterFocusMode()
			return m, nil
		}

	default:
		// Pass other keys to the list component for navigation
		var cmd tea.Cmd
		m.focusList, cmd = m.focusList.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleArchiveMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, archiveKeys.ViewArchive):
		// Exit archive view
		m.showArchive = false
		m.view = viewList
		m.tasks = m.storage.GetTasks()
		m.statusMsg = "Viewing active tasks"
		return m, nil

	case key.Matches(msg, archiveKeys.Help):
		// Toggle help
		m.listHelp.ShowAll = !m.listHelp.ShowAll
		return m, nil

	case key.Matches(msg, archiveKeys.Quit):
		m.quitting = true
		return m, tea.Quit

	default:
		// Pass other keys to the list component for navigation
		var cmd tea.Cmd
		m.archiveList, cmd = m.archiveList.Update(msg)
		return m, cmd
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

	// Focus mode view
	if m.view == viewFocus {
		b.WriteString(m.focusList.View() + "\n")
		if m.statusMsg != "" {
			b.WriteString(statusStyle.Render(m.statusMsg) + "\n")
		}
		b.WriteString(m.focusHelp.View(focusKeys))
		return b.String()
	}

	// Archive view
	if m.view == viewArchive {
		b.WriteString(m.archiveList.View() + "\n")
		if m.statusMsg != "" {
			b.WriteString(statusStyle.Render(m.statusMsg) + "\n")
		}
		b.WriteString(m.listHelp.View(archiveKeys))
		return b.String()
	}

	// Title
	title := "📋 Tasks"
	if m.showArchive {
		title = "📦 Archive"
	}
	b.WriteString(titleStyle.Render(title) + "\n\n")

	// Search bar (if active)
	if m.view == viewSearch {
		b.WriteString(inputStyle.Render("🔍 " + m.searchInput.View()) + "\n\n")
	}

	// Add task form (if active)
	if m.view == viewAddTask {
		b.WriteString(inputStyle.Render("➕ " + m.addInput.View()) + "\n")
		b.WriteString(helpStyle.Render("  +tag for tags, +1d/+1w/tomorrow for due") + "\n\n")
	}

	// Edit task form (if active)
	if m.view == viewEditTask && m.editForm != nil {
		b.WriteString(titleStyle.Render("✏️  Edit Task") + "\n\n")
		b.WriteString(m.editForm.View() + "\n")
		b.WriteString(helpStyle.Render("  esc: cancel") + "\n\n")
		return b.String()
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
	if m.showArchive {
		b.WriteString(m.listHelp.View(archiveKeys))
	} else {
		b.WriteString(m.listHelp.View(listKeys))
	}

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
