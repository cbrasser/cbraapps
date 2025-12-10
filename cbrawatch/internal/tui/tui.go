package tui

import (
	"fmt"
	"io"
	"strings"

	"cbrawatch/internal/config"
	"cbrawatch/internal/git"
	"cbrawatch/internal/scanner"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// repoItem implements list.Item and list.DefaultItem interfaces
type repoItem struct {
	status git.RepoStatus
}

func (i repoItem) FilterValue() string {
	return i.status.Path
}

func (i repoItem) Title() string {
	indicator := getStatusIndicator(i.status)

	// Use custom name if provided, otherwise use path
	displayName := i.status.Path
	if i.status.CustomName != "" {
		displayName = i.status.CustomName
	}

	// Truncate display name if too long
	maxNameLen := 60
	if len(displayName) > maxNameLen {
		displayName = "..." + displayName[len(displayName)-maxNameLen+3:]
	}

	branch := ""
	if i.status.BranchName != "" {
		branch = fmt.Sprintf(" [%s]", i.status.BranchName)
	}

	return fmt.Sprintf("%s %s%s", indicator, displayName, branch)
}

func (i repoItem) Description() string {
	return i.status.StatusSummary()
}

// Custom delegate for repo items
type repoDelegate struct{}

func (d repoDelegate) Height() int                             { return 2 }
func (d repoDelegate) Spacing() int                            { return 0 }
func (d repoDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d repoDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(repoItem)
	if !ok {
		return
	}

	title := i.Title()
	desc := i.Description()

	// Apply status-based styling to description
	var styledDesc string
	if i.status.Error != "" {
		styledDesc = dangerStyle.Render(desc)
	} else if i.status.IsClean() {
		styledDesc = cleanStyle.Render(desc)
	} else if i.status.HasUpstreamChange {
		styledDesc = infoStyle.Render(desc)
	} else {
		styledDesc = warningStyle.Render(desc)
	}

	if index == m.Index() {
		// Selected item - add cursor indicator
		cursor := cursorStyle.Render("▶ ")
		fmt.Fprint(w, selectedItemStyle.Render(cursor+title)+"\n")
		fmt.Fprint(w, selectedItemDescStyle.Render("  "+styledDesc))
	} else {
		// Normal item - add spacing to align with cursor
		fmt.Fprint(w, listItemStyle.Render("  "+title)+"\n")
		fmt.Fprint(w, listItemDescStyle.Render("  "+styledDesc))
	}
}

// Key bindings
type keyMap struct {
	QuickPush       key.Binding
	PushWithMessage key.Binding
	Pull            key.Binding
	Refresh         key.Binding
	Quit            key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.QuickPush, k.PushWithMessage, k.Pull, k.Refresh, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.QuickPush, k.PushWithMessage, k.Pull},
		{k.Refresh, k.Quit},
	}
}

var keys = keyMap{
	QuickPush: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "quick push"),
	),
	PushWithMessage: key.NewBinding(
		key.WithKeys("P"),
		key.WithHelp("P", "push w/ message"),
	),
	Pull: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "pull"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

type viewState int

const (
	viewList viewState = iota
	viewCommitForm
)

type Model struct {
	config         *config.Config
	list           list.Model
	help           help.Model
	keys           keyMap
	spinner        spinner.Model
	repos          []git.RepoStatus
	state          viewState
	commitForm     *huh.Form
	message        string
	messageType    messageType
	spinnerMessage string
	width          int
	height         int
	isProcessing   bool
}

type messageType int

const (
	messageNone messageType = iota
	messageSuccess
	messageError
	messageInfo
)

type scanCompleteMsg struct {
	repos []git.RepoStatus
}

type gitOperationMsg struct {
	success bool
	err     error
	action  string
}

func New(cfg *config.Config) Model {
	delegate := repoDelegate{}
	l := list.New([]list.Item{}, delegate, 80, 20)
	l.Title = "🔍 Git Repository Monitor"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false) // We'll use our own help
	l.Styles.Title = titleStyle

	h := help.New()
	h.ShowAll = false

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	return Model{
		config:         cfg,
		list:           l,
		help:           h,
		keys:           keys,
		spinner:        s,
		repos:          []git.RepoStatus{},
		state:          viewList,
		message:        "",
		messageType:    messageNone,
		spinnerMessage: "Scanning repositories",
		isProcessing:   true, // Start with loading state
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		scanRepos(m.config),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h, v := listStyle.GetFrameSize()
		m.list.SetWidth(msg.Width - h)
		m.list.SetHeight(msg.Height - v - 8) // Leave space for help
		m.help.Width = msg.Width
		return m, nil

	case spinner.TickMsg:
		if m.isProcessing {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		// Don't process keys when processing
		if m.isProcessing {
			if key.Matches(msg, m.keys.Quit) {
				return m, tea.Quit
			}
			return m, nil
		}
		// Handle different views
		switch m.state {
		case viewCommitForm:
			return m.updateCommitForm(msg)
		case viewList:
			return m.updateListView(msg)
		}

	case scanCompleteMsg:
		m.repos = msg.repos
		m.isProcessing = false

		// Convert repos to list items
		items := make([]list.Item, len(m.repos))
		for i, repo := range m.repos {
			items[i] = repoItem{status: repo}
		}

		// Update list with items
		cmd := m.list.SetItems(items)

		if len(m.repos) == 0 {
			m.message = "No repositories found. Check your config paths."
			m.messageType = messageInfo
		} else {
			m.message = fmt.Sprintf("✓ Found %d repositories", len(m.repos))
			m.messageType = messageSuccess
		}
		return m, cmd

	case gitOperationMsg:
		m.isProcessing = false
		if msg.success {
			m.message = fmt.Sprintf("✓ %s completed successfully", msg.action)
			m.messageType = messageSuccess
			// Refresh repos after successful operation
			m.isProcessing = true
			m.spinnerMessage = "Refreshing repositories"
			return m, tea.Batch(m.spinner.Tick, scanRepos(m.config))
		} else {
			m.message = fmt.Sprintf("✗ %s failed: %v", msg.action, msg.err)
			m.messageType = messageError
		}
		return m, nil

	}

	return m, nil
}

func (m Model) updateListView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Refresh):
		m.message = ""
		m.messageType = messageNone
		m.spinnerMessage = "Refreshing repositories"
		m.isProcessing = true
		return m, tea.Batch(m.spinner.Tick, scanRepos(m.config))

	case key.Matches(msg, m.keys.QuickPush):
		if len(m.repos) > 0 {
			m.isProcessing = true
			m.message = ""
			m.messageType = messageNone
			m.spinnerMessage = "Pushing changes"
			return m, tea.Batch(m.spinner.Tick, performAddCommitPush(m.currentRepo(), m.config.DefaultCommitMsg))
		}

	case key.Matches(msg, m.keys.PushWithMessage):
		if len(m.repos) > 0 {
			m.state = viewCommitForm
			m.commitForm = createCommitForm()
			return m, m.commitForm.Init()
		}

	case key.Matches(msg, m.keys.Pull):
		if len(m.repos) > 0 {
			m.isProcessing = true
			m.message = ""
			m.messageType = messageNone
			m.spinnerMessage = "Pulling changes"
			return m, tea.Batch(m.spinner.Tick, performPull(m.currentRepo()))
		}

	default:
		// Let the list handle navigation, filtering, etc.
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) updateCommitForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	form, cmd := m.commitForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.commitForm = f

		if m.commitForm.State == huh.StateCompleted {
			message := m.commitForm.GetString("message")
			m.state = viewList
			m.isProcessing = true
			m.spinnerMessage = "Pushing changes"
			m.message = ""
			m.messageType = messageNone
			return m, tea.Batch(
				m.spinner.Tick,
				performAddCommitPush(m.currentRepo(), message),
			)
		}

		if m.commitForm.State == huh.StateAborted {
			m.state = viewList
			return m, nil
		}
	}

	return m, cmd
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	switch m.state {
	case viewCommitForm:
		return m.viewCommitForm()
	case viewList:
		return m.viewList()
	}

	return ""
}

func (m Model) viewList() string {
	var b strings.Builder

	// Show loading state
	if m.isProcessing && len(m.repos) == 0 {
		b.WriteString(titleStyle.Render("🔍 Git Repository Monitor"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("  %s %s...\n", m.spinner.View(), m.spinnerMessage))
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render("  Please wait while the operation completes."))
		return listStyle.Render(b.String())
	}

	// List view
	b.WriteString(m.list.View())

	// Processing indicator (when refreshing existing list)
	if m.isProcessing && len(m.repos) > 0 {
		b.WriteString("\n")
		b.WriteString(processingStyle.Render(fmt.Sprintf("  %s %s...", m.spinner.View(), m.spinnerMessage)))
	}

	// Message box
	if m.message != "" {
		b.WriteString("\n")
		msgStyle := m.getMessageStyle()
		b.WriteString(messageBoxStyle.Render(msgStyle.Render(m.message)))
	}

	// Help text
	b.WriteString("\n")
	b.WriteString(m.help.View(m.keys))

	return b.String()
}

func (m Model) viewCommitForm() string {
	var b strings.Builder

	title := titleStyle.Render("📝 Commit Message")
	b.WriteString(title)
	b.WriteString("\n\n")

	b.WriteString(m.commitForm.View())

	return baseStyle.Render(b.String())
}

func (m Model) getMessageStyle() lipgloss.Style {
	switch m.messageType {
	case messageSuccess:
		return successStyle
	case messageError:
		return errorStyle
	case messageInfo:
		return processingStyle
	default:
		return mutedStyle
	}
}

func (m Model) currentRepo() git.RepoStatus {
	if len(m.repos) == 0 {
		return git.RepoStatus{}
	}

	// Get the selected index from the list
	selectedIndex := m.list.Index()
	if selectedIndex >= 0 && selectedIndex < len(m.repos) {
		return m.repos[selectedIndex]
	}

	return git.RepoStatus{}
}

func getStatusIndicator(repo git.RepoStatus) string {
	if repo.Error != "" {
		return dangerIndicator
	}
	if repo.IsClean() {
		return cleanIndicator
	}
	if repo.HasUpstreamChange {
		return infoIndicator
	}
	return warningIndicator
}

func createCommitForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("message").
				Title("Commit Message").
				Placeholder("Your commit message...").
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("commit message cannot be empty")
					}
					return nil
				}),
		),
	)
}

// Commands

func scanRepos(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		repos := scanner.ScanRepositories(cfg)
		return scanCompleteMsg{repos: repos}
	}
}

func performAddCommitPush(repo git.RepoStatus, message string) tea.Cmd {
	return func() tea.Msg {
		err := git.AddCommitPush(repo.Path, message)
		if err != nil {
			return gitOperationMsg{
				success: false,
				err:     err,
				action:  "add/commit/push",
			}
		}
		return gitOperationMsg{
			success: true,
			action:  "add/commit/push",
		}
	}
}

func performPull(repo git.RepoStatus) tea.Cmd {
	return func() tea.Msg {
		err := git.Pull(repo.Path)
		if err != nil {
			return gitOperationMsg{
				success: false,
				err:     err,
				action:  "pull",
			}
		}
		return gitOperationMsg{
			success: true,
			action:  "pull",
		}
	}
}

func Run(cfg *config.Config) error {
	p := tea.NewProgram(New(cfg), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
