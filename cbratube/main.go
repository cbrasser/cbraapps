package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kkdai/youtube/v2"
)


var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")) // Less flashy, softer white/gray

	channelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	timeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("255"))

	downloadedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("82")).
			Render(" ✓")

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)

	searchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	// Default channel colors (10 colors)
	defaultColors = []string{
		"#FF6B6B", "#4ECDC4", "#45B7D1", "#FFA07A", "#98D8C8",
		"#F7DC6F", "#BB8FCE", "#85C1E2", "#F8B739", "#52BE80",
	}
)

type Config struct {
	Channels    []string `toml:"channels"`
	MaxVideos   int      `toml:"max_videos"`   // Max videos per channel to load
	DownloadDir string   `toml:"download_dir"` // Directory to download videos to
	Colors      []string `toml:"colors"`       // Channel colors (10 colors, reused if needed)
}

type Video struct {
	ID        string
	Title     string
	Channel   string
	Published time.Time
	URL       string
}

func (v Video) FilterValue() string { return v.Title }

// videoWithStatus wraps Video with download status for display
type videoWithStatus struct {
	Video        Video
	Downloaded   bool
	ChannelColor string // Color for the channel
}

func (v videoWithStatus) FilterValue() string { return v.Video.FilterValue() }

type videoDelegate struct{}

func (d videoDelegate) Height() int                             { return 3 }
func (d videoDelegate) Spacing() int                            { return 1 }
func (d videoDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d videoDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var v Video
	var isDownloaded bool
	var channelColor string

	// Handle both Video and videoWithStatus types
	if vws, ok := item.(videoWithStatus); ok {
		v = vws.Video
		isDownloaded = vws.Downloaded
		channelColor = vws.ChannelColor // Get the channel color
	} else if vid, ok := item.(Video); ok {
		v = vid
		isDownloaded = false
		channelColor = "" // No color for plain Video
	} else {
		return
	}

	// Get channel color - we need to access the model's channelColors
	// For now, we'll use a default color if not found
	downloadedMarker := ""
	if isDownloaded {
		downloadedMarker = downloadedStyle
	}

	// Use channel color for channel name if available
	channelColorStyle := channelStyle
	if channelColor != "" {
		channelColorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(channelColor))
	}

	titleText := v.Title + downloadedMarker
	channelText := v.Channel
	timeText := "• " + v.Published.Format("2006-01-02 15:04")

	if index == m.Index() {
		str := fmt.Sprintf("%s\n%s • %s", titleText, channelText, timeText)
		fmt.Fprint(w, selectedStyle.Render(str))
	} else {
		title := titleStyle.Render(titleText)
		meta := channelColorStyle.Render(channelText) + " " + timeStyle.Render(timeText)
		fmt.Fprintf(w, "%s\n%s", title, meta)
	}
}

type model struct {
	list                 list.Model
	videos               []Video
	loading              bool
	err                  error
	config               Config
	configPath           string
	quitting             bool
	downloading          bool
	downloadURL          string
	spinner              spinner.Model
	searching            bool
	searchQuery          string
	channelColors        map[string]string // Map channel name to color
	managingChannels     bool
	channelInputActive   bool
	channelInput         string
	selectedChannelIndex int
	channelMessage       string
}

type videosLoadedMsg struct {
	videos []Video
	err    error
}

// Removed downloadProgressMsg - using spinner instead

type downloadCompleteMsg struct {
	err      error
	message  string
	useYtDlp bool // Flag to indicate we should use yt-dlp fallback
}

func (m model) Init() tea.Cmd {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	m.spinner = s

	m.channelColors = make(map[string]string)
	colors := m.config.Colors
	if len(colors) == 0 {
		colors = defaultColors
	}

	cmds := []tea.Cmd{s.Tick}
	if len(m.config.Channels) > 0 {
		cmds = append(cmds, loadVideos(m.config))
	}
	if len(cmds) == 1 {
		return s.Tick
	}
	return tea.Batch(cmds...)
}

func loadVideos(cfg Config) tea.Cmd {
	return func() tea.Msg {
		videos, err := fetchVideos(cfg)
		return videosLoadedMsg{videos: videos, err: err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.managingChannels {
			return handleChannelManagerKey(m, msg)
		}

		// Handle search mode
		if m.searching {
			switch msg.String() {
			case "esc":
				m.searching = false
				m.searchQuery = ""
				// Reset filter
				items := make([]list.Item, len(m.videos))
				for i, v := range m.videos {
					downloaded := isVideoDownloaded(m.config.DownloadDir, v)
					channelColor := m.channelColors[v.Channel]
					items[i] = videoWithStatus{Video: v, Downloaded: downloaded, ChannelColor: channelColor}
				}
				m.list.SetItems(items)
				return m, nil
			case "backspace":
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
				}
			case "enter":
				m.searching = false
				return m, nil
			default:
				if len(msg.Runes) > 0 {
					m.searchQuery += string(msg.Runes)
				}
			}
			// Filter videos based on search query
			if m.searchQuery != "" {
				filtered := []list.Item{}
				query := strings.ToLower(m.searchQuery)
				for _, v := range m.videos {
					titleMatch := strings.Contains(strings.ToLower(v.Title), query)
					channelMatch := strings.Contains(strings.ToLower(v.Channel), query)
					if titleMatch || channelMatch {
						downloaded := isVideoDownloaded(m.config.DownloadDir, v)
						channelColor := m.channelColors[v.Channel]
						filtered = append(filtered, videoWithStatus{Video: v, Downloaded: downloaded, ChannelColor: channelColor})
					}
				}
				m.list.SetItems(filtered)
			} else {
				// Show all videos if search is empty
				items := make([]list.Item, len(m.videos))
				for i, v := range m.videos {
					downloaded := isVideoDownloaded(m.config.DownloadDir, v)
					channelColor := m.channelColors[v.Channel]
					items[i] = videoWithStatus{Video: v, Downloaded: downloaded, ChannelColor: channelColor}
				}
				m.list.SetItems(items)
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			if m.searching {
				m.searching = false
				m.searchQuery = ""
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		case "/":
			m.searching = true
			m.searchQuery = ""
			return m, nil
		case "c":
			m.managingChannels = true
			m.channelInputActive = false
			m.channelInput = ""
			m.channelMessage = ""
			return m, nil
		case "r":
			m.loading = true
			return m, loadVideos(m.config)
		case "enter":
			if len(m.videos) > 0 && !m.downloading {
				selectedItem := m.list.SelectedItem()
				var v Video
				if vws, ok := selectedItem.(videoWithStatus); ok {
					v = vws.Video
				} else if vid, ok := selectedItem.(Video); ok {
					v = vid
				} else {
					return m, nil
				}
				m.downloading = true
				m.downloadURL = v.URL
				return m, tea.Batch(
					downloadVideo(m.config.DownloadDir, v.URL),
					m.spinner.Tick,
				)
			}
		case "d":
			// Delete downloaded video
			if len(m.videos) > 0 && !m.downloading {
				selectedItem := m.list.SelectedItem()
				var v Video
				if vws, ok := selectedItem.(videoWithStatus); ok {
					v = vws.Video
					if !vws.Downloaded {
						return m, nil // Not downloaded, nothing to delete
					}
				} else if vid, ok := selectedItem.(Video); ok {
					v = vid
					if !isVideoDownloaded(m.config.DownloadDir, v) {
						return m, nil // Not downloaded, nothing to delete
					}
				} else {
					return m, nil
				}
				// Delete the file and reload
				path := getDownloadedVideoPath(m.config.DownloadDir, v)
				if path != "" {
					os.Remove(path)
					// Reload videos to update UI
					return m, loadVideos(m.config)
				}
			}
		case "o":
			// Open video (file if downloaded, URL if not)
			if len(m.videos) > 0 {
				selectedItem := m.list.SelectedItem()
				var v Video
				if vws, ok := selectedItem.(videoWithStatus); ok {
					v = vws.Video
				} else if vid, ok := selectedItem.(Video); ok {
					v = vid
				} else {
					return m, nil
				}
				return m, openVideo(m.config.DownloadDir, v)
			}
		}

	case videosLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.videos = msg.videos

		// Assign colors to channels
		colors := m.config.Colors
		if len(colors) == 0 {
			colors = defaultColors
		}
		channelSet := make(map[string]bool)
		channelIndex := 0
		for _, v := range m.videos {
			if !channelSet[v.Channel] {
				m.channelColors[v.Channel] = colors[channelIndex%len(colors)]
				channelSet[v.Channel] = true
				channelIndex++
			}
		}

		items := make([]list.Item, len(m.videos))
		for i, v := range m.videos {
			// Check if video is downloaded and wrap it
			downloaded := isVideoDownloaded(m.config.DownloadDir, v)
			channelColor := m.channelColors[v.Channel]
			items[i] = videoWithStatus{Video: v, Downloaded: downloaded, ChannelColor: channelColor}
		}
		m.list.SetItems(items)
		// Make sure the list is visible
		if len(items) > 0 {
			m.list.Select(0)
		}
		return m, nil

	case tea.WindowSizeMsg:
		// Account for border: 2 chars padding on each side = 4, plus 2 for border itself = 6 total width
		m.list.SetWidth(msg.Width - 6)
		// Reserve space: border top/bottom (2) + header (2) + spacing (1) + footer (2) + spacing (1) = 8 base
		height := msg.Height - 8
		if m.searching {
			height -= 2 // Extra space for search bar
		}
		// Set height to fill available space (list will handle pagination automatically)
		m.list.SetHeight(height)
		return m, nil

	case downloadCompleteMsg:
		if msg.useYtDlp {
			// Fallback to yt-dlp
			selectedItem := m.list.SelectedItem()
			var v Video
			if vws, ok := selectedItem.(videoWithStatus); ok {
				v = vws.Video
			} else if vid, ok := selectedItem.(Video); ok {
				v = vid
			} else {
				return m, nil
			}
			// Keep downloading state, but switch to yt-dlp
			m.err = nil // Clear any previous errors
			return m, tea.Batch(
				downloadVideoWithYtDlp(m.config.DownloadDir, v.URL),
				m.spinner.Tick,
			)
		}
		m.downloading = false
		if msg.err != nil {
			m.err = msg.err
		} else if msg.message != "" {
			// Success message - clear any previous errors
			m.err = nil
			// Reload videos to update download status
			return m, loadVideos(m.config)
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		if m.downloading || m.loading {
			m.spinner, cmd = m.spinner.Update(msg)
			if cmd != nil {
				return m, cmd
			}
			return m, nil
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func handleChannelManagerKey(m model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if m.channelInputActive {
		switch key {
		case "esc":
			m.channelInputActive = false
			m.channelInput = ""
			m.channelMessage = ""
			return m, nil
		case "backspace":
			if len(m.channelInput) > 0 {
				m.channelInput = m.channelInput[:len(m.channelInput)-1]
			}
			return m, nil
		case "enter":
			channel, err := normalizeChannelInput(m.channelInput)
			if err != nil {
				m.channelMessage = err.Error()
				return m, nil
			}
			if _, err := extractChannelID(channel); err != nil {
				m.channelMessage = fmt.Sprintf("Could not resolve channel: %v", err)
				return m, nil
			}
			if channelExists(m.config.Channels, channel) {
				m.channelMessage = "Channel already added"
				return m, nil
			}
			m.config.Channels = append(m.config.Channels, channel)
			if err := saveConfig(m.config, m.configPath); err != nil {
				m.channelMessage = fmt.Sprintf("Failed to save channel: %v", err)
				return m, nil
			}
			m.channelInputActive = false
			m.channelInput = ""
			m.channelMessage = fmt.Sprintf("Added %s", channel)
			m.selectedChannelIndex = len(m.config.Channels) - 1
			m.loading = true
			return m, loadVideos(m.config)
		default:
			if len(msg.Runes) > 0 {
				m.channelInput += string(msg.Runes)
			}
			return m, nil
		}
	}

	switch key {
	case "esc", "c":
		m.managingChannels = false
		m.channelInputActive = false
		m.channelInput = ""
		m.channelMessage = ""
		return m, nil
	case "a":
		m.channelInputActive = true
		m.channelInput = ""
		m.channelMessage = "Type a channel name, handle (@name), or URL"
		return m, nil
	case "up", "k":
		if len(m.config.Channels) > 0 && m.selectedChannelIndex > 0 {
			m.selectedChannelIndex--
		}
		return m, nil
	case "down", "j":
		if len(m.config.Channels) > 0 && m.selectedChannelIndex < len(m.config.Channels)-1 {
			m.selectedChannelIndex++
		}
		return m, nil
	case "x", "delete":
		if len(m.config.Channels) == 0 {
			return m, nil
		}
		removed := m.config.Channels[m.selectedChannelIndex]
		m.config.Channels = append(m.config.Channels[:m.selectedChannelIndex], m.config.Channels[m.selectedChannelIndex+1:]...)
		if err := saveConfig(m.config, m.configPath); err != nil {
			m.channelMessage = fmt.Sprintf("Failed to save channel list: %v", err)
			return m, nil
		}
		if len(m.config.Channels) == 0 {
			m.selectedChannelIndex = 0
			m.channelMessage = fmt.Sprintf("Removed %s", removed)
			m.list.SetItems([]list.Item{})
			m.videos = nil
			m.loading = false
			return m, nil
		}
		if m.selectedChannelIndex >= len(m.config.Channels) {
			m.selectedChannelIndex = len(m.config.Channels) - 1
		}
		m.channelMessage = fmt.Sprintf("Removed %s", removed)
		m.loading = true
		return m, loadVideos(m.config)
	default:
		return m, nil
	}
}

func channelExists(channels []string, candidate string) bool {
	for _, ch := range channels {
		if strings.EqualFold(strings.TrimSpace(ch), strings.TrimSpace(candidate)) {
			return true
		}
	}
	return false
}

func normalizeChannelInput(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("Channel cannot be empty")
	}

	// Already a URL
	if strings.Contains(trimmed, "youtube.com") || strings.Contains(trimmed, "youtu.be") {
		return trimmed, nil
	}

	// Channel ID
	if strings.HasPrefix(trimmed, "UC") && len(trimmed) == 24 {
		return trimmed, nil
	}

	// Handle with @ prefix
	if strings.HasPrefix(trimmed, "@") {
		return fmt.Sprintf("https://www.youtube.com/%s", trimmed), nil
	}

	// Plain name -> treat as slug
	handle := strings.ReplaceAll(trimmed, " ", "")
	if handle == "" {
		return "", fmt.Errorf("Channel name must include letters or numbers")
	}
	return fmt.Sprintf("https://www.youtube.com/%s", handle), nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	if m.managingChannels {
		return m.channelManagerView()
	}

	if m.loading {
		spinnerView := m.spinner.View() + " Loading videos..."
		return borderStyle.Render(spinnerView)
	}

	if m.err != nil {
		return borderStyle.Render(fmt.Sprintf("Error: %v\n\nPress q to quit", m.err))
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Render("zebratube")

	footerText := "r: refresh • enter: download • o: open • d: delete • /: search • c: channels • q: quit"
	if m.downloading {
		spinnerView := m.spinner.View()
		footerText = fmt.Sprintf("%s Downloading...", spinnerView)
	}

	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(footerText)

	// Search bar
	searchBar := ""
	if m.searching {
		searchBar = "\n" + searchStyle.Render(fmt.Sprintf("Search: %s_", m.searchQuery))
	}

	// Build content with proper spacing
	listView := m.list.View()
	// Remove any trailing newlines from list view that might break the border
	listView = strings.TrimRight(listView, "\n")
	content := fmt.Sprintf("%s\n\n%s\n\n%s%s", header, listView, footer, searchBar)
	// Render with border - ensure proper closing
	return borderStyle.Render(content)
}

func (m model) channelManagerView() string {
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Render("channels")

	var builder strings.Builder
	builder.WriteString(header)
	builder.WriteString("\n\n")

	if len(m.config.Channels) == 0 {
		builder.WriteString("No channels yet. Press a to add one.\n")
	} else {
		for i, ch := range m.config.Channels {
			line := fmt.Sprintf("%d. %s", i+1, ch)
			if i == m.selectedChannelIndex {
				builder.WriteString(selectedStyle.Render(line))
			} else {
				builder.WriteString(channelStyle.Render(line))
			}
			builder.WriteString("\n")
		}
	}

	if m.channelInputActive {
		builder.WriteString("\n")
		builder.WriteString(searchStyle.Render(fmt.Sprintf("Channel: %s_", m.channelInput)))
	}

	if m.channelMessage != "" {
		builder.WriteString("\n\n")
		builder.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(m.channelMessage))
	}

	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("a: add • enter: confirm • x: remove • esc/c: back to videos")

	builder.WriteString("\n\n")
	builder.WriteString(footer)

	content := strings.TrimRight(builder.String(), "\n")
	return borderStyle.Render(content)
}

func loadConfig() (Config, string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Config{}, "", err
	}

	configPath := filepath.Join(homeDir, ".config", "cbraapps", "cbratube.toml")

	// Try to read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Create base config if it doesn't exist
		if os.IsNotExist(err) {
			defaultDownloadDir := filepath.Join(homeDir, "Downloads")
			exampleConfig := Config{
				Channels:    []string{},
				MaxVideos:   10,
				DownloadDir: defaultDownloadDir,
				Colors:      defaultColors,
			}

			dir := filepath.Dir(configPath)
			os.MkdirAll(dir, 0755)

			f, err := os.Create(configPath)
			if err != nil {
				return Config{}, configPath, err
			}
			defer f.Close()

			encoder := toml.NewEncoder(f)
			if err := encoder.Encode(exampleConfig); err != nil {
				return Config{}, configPath, err
			}

			return exampleConfig, configPath, nil
		}
		return Config{}, configPath, err
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, configPath, err
	}

	// Set defaults if not configured
	if cfg.MaxVideos <= 0 {
		cfg.MaxVideos = 10
	}
	if cfg.DownloadDir == "" {
		cfg.DownloadDir = filepath.Join(homeDir, "Downloads")
	}
	if len(cfg.Colors) == 0 {
		cfg.Colors = defaultColors
	}

	return cfg, configPath, nil
}

func saveConfig(cfg Config, configPath string) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	return encoder.Encode(cfg)
}

func extractChannelID(input string) (string, error) {
	channelURL := strings.TrimSpace(input)

	if channelURL == "" {
		return "", fmt.Errorf("empty channel identifier")
	}

	// If it's already a channel ID (starts with UC and is 24 chars)
	if strings.HasPrefix(channelURL, "UC") && len(channelURL) == 24 {
		return channelURL, nil
	}

	// If it's a bare handle or slug without URL
	if !strings.Contains(channelURL, "youtube.com") && !strings.Contains(channelURL, "youtu.be") {
		if strings.HasPrefix(channelURL, "@") {
			channelURL = fmt.Sprintf("https://www.youtube.com/%s", channelURL)
		} else {
			channelURL = fmt.Sprintf("https://www.youtube.com/%s", channelURL)
		}
	}

	parsed, err := url.Parse(channelURL)
	if err != nil {
		return "", fmt.Errorf("invalid channel URL: %s", channelURL)
	}

	if !strings.Contains(parsed.Host, "youtube.com") {
		return "", fmt.Errorf("unsupported host in channel URL: %s", channelURL)
	}

	path := strings.Trim(parsed.Path, "/")
	if path == "" {
		return "", fmt.Errorf("invalid channel URL format: %s", channelURL)
	}

	parts := strings.Split(path, "/")

	for i, part := range parts {
		if part == "channel" && i+1 < len(parts) {
			id := strings.Split(parts[i+1], "?")[0]
			if strings.HasPrefix(id, "UC") {
				return id, nil
			}
		}
		if strings.HasPrefix(part, "@") {
			return resolveChannelPageChannelID(fmt.Sprintf("https://www.youtube.com/%s", part))
		}
		if (part == "c" || part == "user") && i+1 < len(parts) {
			target := strings.Split(parts[i+1], "?")[0]
			return resolveChannelPageChannelID(fmt.Sprintf("https://www.youtube.com/%s/%s", part, target))
		}
	}

	// Fallback: resolve the original URL (covers /slug formats)
	return resolveChannelPageChannelID(channelURL)
}

func resolveChannelPageChannelID(channelURL string) (string, error) {
	if !strings.HasPrefix(channelURL, "http") {
		channelURL = "https://www.youtube.com/" + strings.TrimLeft(channelURL, "/")
	}

	resp, err := http.Get(channelURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	html := string(body)
	searchOrder := []string{`"channelId":"`, `"externalId":"`}
	for _, marker := range searchOrder {
		idx := strings.Index(html, marker)
		if idx == -1 {
			continue
		}
		start := idx + len(marker)
		if start+24 <= len(html) {
			channelID := html[start : start+24]
			if strings.HasPrefix(channelID, "UC") {
				return channelID, nil
			}
		}
	}

	return "", fmt.Errorf("could not extract channel ID from %s", channelURL)
}

// RSS Feed structures
type Feed struct {
	XMLName xml.Name `xml:"feed"`
	Entries []Entry  `xml:"entry"`
	Author  Author   `xml:"author"`
}

type Author struct {
	Name string `xml:"name"`
}

type Entry struct {
	VideoID   string `xml:"videoId"`
	Title     string `xml:"title"`
	Published string `xml:"published"`
	Author    Author `xml:"author"`
}

func fetchVideos(cfg Config) ([]Video, error) {
	var allVideos []Video

	for _, channelURL := range cfg.Channels {
		channelID, err := extractChannelID(channelURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			continue
		}

		// Fetch RSS feed
		rssURL := fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", channelID)
		resp, err := http.Get(rssURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to fetch %s: %v\n", channelID, err)
			continue
		}

		var feed Feed
		decoder := xml.NewDecoder(resp.Body)
		if err := decoder.Decode(&feed); err != nil {
			resp.Body.Close()
			fmt.Fprintf(os.Stderr, "Warning: failed to parse RSS for %s: %v\n", channelID, err)
			continue
		}
		resp.Body.Close()

		channelName := feed.Author.Name
		if channelName == "" && len(feed.Entries) > 0 {
			channelName = feed.Entries[0].Author.Name
		}

		// Limit videos per channel
		maxVideos := cfg.MaxVideos
		if maxVideos <= 0 {
			maxVideos = 10 // Default to 10 if not configured
		}

		entriesToProcess := feed.Entries
		if len(entriesToProcess) > maxVideos {
			entriesToProcess = entriesToProcess[:maxVideos]
		}

		for _, entry := range entriesToProcess {
			publishedAt, _ := time.Parse(time.RFC3339, entry.Published)
			video := Video{
				ID:        entry.VideoID,
				Title:     entry.Title,
				Channel:   channelName,
				Published: publishedAt,
				URL:       fmt.Sprintf("https://www.youtube.com/watch?v=%s", entry.VideoID),
			}
			allVideos = append(allVideos, video)
		}
	}

	if len(allVideos) == 0 {
		return nil, fmt.Errorf("no videos found - check your channel URLs")
	}

	// Sort by publish date (newest first)
	sort.Slice(allVideos, func(i, j int) bool {
		return allVideos[i].Published.After(allVideos[j].Published)
	})

	return allVideos, nil
}

func openURL(url string) {
	// Simple cross-platform URL opener
	var cmd *exec.Cmd

	switch {
	case fileExists("/usr/bin/xdg-open"):
		cmd = exec.Command("xdg-open", url)
	case fileExists("/usr/bin/open"):
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("cmd", "/c", "start", url)
	}

	go cmd.Run()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Progress message type for the progress bar
// Removed progress-related globals - using spinner instead

// downloadVideo downloads a video using the kkdai/youtube Go library
func downloadVideo(downloadDir, url string) tea.Cmd {
	return func() tea.Msg {
		// Create download directory if it doesn't exist
		if downloadDir == "" {
			homeDir, _ := os.UserHomeDir()
			downloadDir = filepath.Join(homeDir, "Downloads")
		}
		if err := os.MkdirAll(downloadDir, 0755); err != nil {
			return downloadCompleteMsg{err: fmt.Errorf("failed to create download directory: %v", err)}
		}

		// Create YouTube client with custom HTTP client to avoid 403 errors
		client := youtube.Client{
			HTTPClient: &http.Client{
				Timeout: 30 * time.Second,
			},
		}

		// Get video information
		video, err := client.GetVideo(url)
		if err != nil {
			return downloadCompleteMsg{err: fmt.Errorf("failed to get video info: %v", err)}
		}

		// Find the best quality video format
		// Try different quality levels and format types
		var formats []youtube.Format
		var selectedFormat *youtube.Format

		// First, try to find a format that doesn't require basejs (usually video-only or audio-only formats)
		// These formats are often more reliable
		for _, f := range video.Formats {
			// Prefer formats that are video-only or have both video and audio
			// Avoid formats that require complex player extraction
			if f.MimeType != "" {
				formats = append(formats, f)
			}
		}

		// If no formats found, try quality-based selection
		if len(formats) == 0 {
			qualityLevels := []string{"medium", "high", "low", ""}
			for _, quality := range qualityLevels {
				if quality != "" {
					formats = video.Formats.Quality(quality)
				} else {
					formats = video.Formats
				}
				if len(formats) > 0 {
					break
				}
			}
		}

		if len(formats) == 0 {
			return downloadCompleteMsg{err: fmt.Errorf("no video formats available")}
		}

		// Try formats in order, starting with ones that are more likely to work
		// Prefer formats with video codec (not just audio)
		for _, f := range formats {
			if strings.Contains(f.MimeType, "video") {
				selectedFormat = &f
				break
			}
		}

		// If no video format found, use the first available
		if selectedFormat == nil {
			selectedFormat = &formats[0]
		}

		format := *selectedFormat

		// Create output file path
		// Sanitize filename
		title := sanitizeFilename(video.Title)
		if title == "" {
			title = "video"
		}
		ext := format.MimeType
		if strings.Contains(ext, "video/mp4") {
			ext = "mp4"
		} else if strings.Contains(ext, "video/webm") {
			ext = "webm"
		} else {
			ext = "mp4" // default
		}
		outputPath := filepath.Join(downloadDir, fmt.Sprintf("%s.%s", title, ext))

		// Download video
		// Try to get the stream - if it fails, try alternative formats or fallback to yt-dlp
		stream, _, err := client.GetStream(video, &format)
		if err != nil {
			errStr := err.Error()
			isBaseJSError := strings.Contains(errStr, "basejs") || strings.Contains(errStr, "playerConfig")
			// Check for 403 errors in various formats
			is403Error := strings.Contains(errStr, "403") ||
				strings.Contains(errStr, "Forbidden") ||
				strings.Contains(errStr, "status code: 403") ||
				strings.Contains(errStr, "unexpected status code: 403")

			// If we get a basejs or 403 error, try other formats first
			if isBaseJSError || is403Error {
				// Try other formats as fallback
				var fallbackErr error
				success := false
				for i, f := range formats {
					if i == 0 {
						continue // Skip the one we already tried
					}
					time.Sleep(100 * time.Millisecond) // Small delay between attempts
					stream, _, fallbackErr = client.GetStream(video, &f)
					if fallbackErr == nil {
						format = f // Use this format instead
						err = nil
						success = true
						break
					}
				}
				if !success {
					// If 403 error and all formats failed, fallback to yt-dlp
					if is403Error || strings.Contains(fallbackErr.Error(), "403") || strings.Contains(fallbackErr.Error(), "status code: 403") {
						return downloadCompleteMsg{err: nil, useYtDlp: true}
					}
					return downloadCompleteMsg{err: fmt.Errorf("failed to get video stream (tried %d formats): %v", len(formats), err)}
				}
			} else {
				return downloadCompleteMsg{err: fmt.Errorf("failed to get video stream: %v", err)}
			}
		}
		defer stream.Close()

		file, err := os.Create(outputPath)
		if err != nil {
			return downloadCompleteMsg{err: fmt.Errorf("failed to create file %s: %v", outputPath, err)}
		}
		defer file.Close()

		// Copy stream to file
		buf := make([]byte, 64*1024) // 64KB buffer for faster downloads
		var downloadErr error

		for {
			nr, er := stream.Read(buf)
			if nr > 0 {
				nw, ew := file.Write(buf[0:nr])
				if nw < 0 || nr < nw {
					nw = 0
					if ew == nil {
						ew = fmt.Errorf("invalid write result")
					}
				}
				if ew != nil {
					downloadErr = ew
					break
				}
				if nr != nw {
					downloadErr = io.ErrShortWrite
					break
				}
			}
			if er != nil {
				if er != io.EOF {
					downloadErr = er
					// Check if it's a 403 error during read - fallback immediately
					errStr := er.Error()
					if strings.Contains(errStr, "403") ||
						strings.Contains(errStr, "Forbidden") ||
						strings.Contains(errStr, "status code: 403") ||
						strings.Contains(errStr, "unexpected status code: 403") {
						file.Close()
						os.Remove(outputPath)
						return downloadCompleteMsg{err: nil, useYtDlp: true}
					}
				}
				break
			}
		}
		file.Close()

		// Send completion or error
		if downloadErr != nil {
			// Check if it's a 403 error - if so, signal fallback to yt-dlp
			errStr := downloadErr.Error()
			if strings.Contains(errStr, "403") ||
				strings.Contains(errStr, "Forbidden") ||
				strings.Contains(errStr, "status code: 403") ||
				strings.Contains(errStr, "unexpected status code: 403") {
				os.Remove(outputPath)
				return downloadCompleteMsg{err: nil, useYtDlp: true}
			}
			os.Remove(outputPath)
			return downloadCompleteMsg{err: fmt.Errorf("download failed: %v", downloadErr)}
		}

		return downloadCompleteMsg{err: nil, message: "Download completed successfully"}
	}
}

// sanitizeFilename removes invalid characters from a filename
func sanitizeFilename(name string) string {
	// Remove invalid characters for filenames
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, char := range invalidChars {
		result = strings.ReplaceAll(result, char, "_")
	}
	// Limit length
	if len(result) > 200 {
		result = result[:200]
	}
	return strings.TrimSpace(result)
}

// isVideoDownloaded checks if a video file exists in the download directory
func isVideoDownloaded(downloadDir string, v Video) bool {
	if downloadDir == "" {
		homeDir, _ := os.UserHomeDir()
		downloadDir = filepath.Join(homeDir, "Downloads")
	}

	// Check for common video extensions
	extensions := []string{".mp4", ".webm", ".mkv", ".m4a", ".mp3", ".flv", ".avi"}
	title := sanitizeFilename(v.Title)

	for _, ext := range extensions {
		path := filepath.Join(downloadDir, title+ext)
		if fileExists(path) {
			return true
		}
	}
	return false
}

// getDownloadedVideoPath returns the path to a downloaded video file, or empty string if not found
func getDownloadedVideoPath(downloadDir string, v Video) string {
	if downloadDir == "" {
		homeDir, _ := os.UserHomeDir()
		downloadDir = filepath.Join(homeDir, "Downloads")
	}

	extensions := []string{".mp4", ".webm", ".mkv", ".m4a", ".mp3", ".flv", ".avi"}
	title := sanitizeFilename(v.Title)

	for _, ext := range extensions {
		path := filepath.Join(downloadDir, title+ext)
		if fileExists(path) {
			return path
		}
	}
	return ""
}

// deleteVideo deletes a downloaded video file
// Note: This function is no longer used - deletion is handled directly in Update()

// openVideo opens a video (file if downloaded, URL if not)
func openVideo(downloadDir string, v Video) tea.Cmd {
	return func() tea.Msg {
		path := getDownloadedVideoPath(downloadDir, v)
		if path != "" {
			// Open file using system default app
			var cmd *exec.Cmd
			switch runtime.GOOS {
			case "darwin":
				cmd = exec.Command("open", path)
			case "linux":
				cmd = exec.Command("xdg-open", path)
			case "windows":
				cmd = exec.Command("cmd", "/c", "start", "", path)
			default:
				return nil
			}
			go cmd.Run()
		} else {
			// Open URL in browser
			openURL(v.URL)
		}
		return nil
	}
}

// downloadVideoWithYtDlp downloads a video using yt-dlp as fallback
func downloadVideoWithYtDlp(downloadDir, url string) tea.Cmd {
	return func() tea.Msg {
		// Find yt-dlp
		var cmdPath string
		if path, err := exec.LookPath("yt-dlp"); err == nil {
			cmdPath = path
		} else if path, err := exec.LookPath("youtube-dl"); err == nil {
			cmdPath = path
		} else {
			return downloadCompleteMsg{err: fmt.Errorf("yt-dlp not found. Please install yt-dlp for reliable downloads when the Go library fails.")}
		}

		// Create download directory if it doesn't exist
		if downloadDir == "" {
			homeDir, _ := os.UserHomeDir()
			downloadDir = filepath.Join(homeDir, "Downloads")
		}
		if err := os.MkdirAll(downloadDir, 0755); err != nil {
			return downloadCompleteMsg{err: fmt.Errorf("failed to create download directory: %v", err)}
		}

		// No progress tracking needed

		// Build command: yt-dlp -o "path/%(title)s.%(ext)s" URL
		outputTemplate := filepath.Join(downloadDir, "%(title)s.%(ext)s")
		cmd := exec.Command(cmdPath,
			"--no-playlist",
			"--quiet", // Suppress output since we're not tracking progress
			"-o", outputTemplate,
			url,
		)

		// Start the command and wait for completion (no progress tracking)
		if err := cmd.Start(); err != nil {
			return downloadCompleteMsg{err: fmt.Errorf("failed to start yt-dlp: %v", err)}
		}

		// Wait for completion
		if err := cmd.Wait(); err != nil {
			return downloadCompleteMsg{err: fmt.Errorf("yt-dlp download failed: %v", err)}
		}

		return downloadCompleteMsg{err: nil, message: "Download completed successfully (using yt-dlp)"}
	}
}

// Removed tickDownloadProgress - using spinner instead

func main() {
	cfg, cfgPath, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	// Validate download directory
	if cfg.DownloadDir != "" {
		if err := os.MkdirAll(cfg.DownloadDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not create download directory %s: %v\n", cfg.DownloadDir, err)
		}
	}

	delegate := videoDelegate{}
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = ""
	l.SetShowStatusBar(false) // Remove blue rectangle
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(true) // Enable pagination for overflow

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := model{
		list:                 l,
		config:               cfg,
		configPath:           cfgPath,
		loading:              len(cfg.Channels) > 0,
		spinner:              s,
		channelColors:        make(map[string]string),
		selectedChannelIndex: 0,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
