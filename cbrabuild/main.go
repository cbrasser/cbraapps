package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/BurntSushi/toml"
)

// Config represents the TOML configuration file structure
type Config struct {
	TargetDir  string          `toml:"target_dir,omitempty"`  // Single target directory (backward compatible)
	TargetDirs []string        `toml:"target_dirs,omitempty"` // Multiple target directories (takes precedence)
	SourceDir  string          `toml:"source_dir"`
	Projects   []ProjectConfig `toml:"projects,omitempty"` // Optional: explicit project list overrides auto-discovery
}

// GetTargetDirs returns the list of target directories, normalizing both single and multiple configs
func (c *Config) GetTargetDirs() []string {
	if len(c.TargetDirs) > 0 {
		return c.TargetDirs
	}
	if c.TargetDir != "" {
		return []string{c.TargetDir}
	}
	return []string{}
}

// ProjectConfig represents a single project in the config
type ProjectConfig struct {
	Path string `toml:"path"`
	Name string `toml:"name,omitempty"` // Optional, defaults to directory name
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
)

type project struct {
	name string
	path string
}

type model struct {
	projects   []project
	targetDirs []string
	cursor     int
	selected   int
	status     string
	quitting   bool
}

func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configPath := filepath.Join(homeDir, ".config", "cbraapps", "cbrabuild.toml")
	return configPath, nil
}

func loadConfig() (Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return Config{}, fmt.Errorf("failed to get config path: %w", err)
	}

	// Check if config exists, if not create default
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := createDefaultConfig(configPath); err != nil {
			return Config{}, fmt.Errorf("failed to create default config: %w", err)
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var config Config
	if err := toml.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

func createDefaultConfig(configPath string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Write default config with comments
	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer f.Close()

	configContent := fmt.Sprintf(`# cbrabuild configuration

# Target directory where built binaries will be moved to. Ideally, somewhere in your PATH
# You can specify a single directory:
target_dir = "%s"

# Or multiple directories (takes precedence over target_dir):
# target_dirs = ["%s", "/another/path"]

# Source directory, where you cloned cbraapps to.
# The build app will automatically discover and build all apps you want
# Example: "%s/Code/cbraapps"
source_dir = ""

# Optional: Explicit project list (overrides auto-discovery if specified)
# Leave empty to use auto-discovery from source_dir
# [[projects]]
# path = "/path/to/project"
# name = "binary-name"
`, filepath.Join(homeDir, ".local", "bin"), filepath.Join(homeDir, ".local", "bin"), homeDir)

	if _, err := f.WriteString(configContent); err != nil {
		return err
	}

	fmt.Printf("Created default config at: %s\n", configPath)
	fmt.Println("Please edit this file and set the 'source_dir' to your cbraapps repository path.")
	return nil
}

func initialModel() (model, error) {
	config, err := loadConfig()
	if err != nil {
		return model{}, err
	}

	var projects []project

	// If explicit projects list is provided, use it
	if len(config.Projects) > 0 {
		for _, p := range config.Projects {
			name := p.Name
			if name == "" {
				// Default to directory name if name is not specified
				name = filepath.Base(p.Path)
			}
			projects = append(projects, project{
				name: name,
				path: p.Path,
			})
		}
	} else if config.SourceDir != "" {
		// Auto-discover projects in source_dir
		discoveredProjects, err := discoverProjects(config.SourceDir)
		if err != nil {
			return model{}, fmt.Errorf("failed to discover projects: %w", err)
		}
		projects = discoveredProjects
	} else {
		return model{}, fmt.Errorf("either 'source_dir' or 'projects' must be configured")
	}

	targetDirs := config.GetTargetDirs()
	if len(targetDirs) == 0 {
		return model{}, fmt.Errorf("either 'target_dir' or 'target_dirs' must be configured")
	}

	return model{
		projects:   projects,
		targetDirs: targetDirs,
		cursor:     0,
		selected:   -1,
		status:     "",
	}, nil
}

// discoverProjects scans source_dir for subdirectories containing go.mod files
// It scans both the root level and one level deep (e.g., utils/*)
func discoverProjects(sourceDir string) ([]project, error) {
	var projects []project

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectPath := filepath.Join(sourceDir, entry.Name())
		goModPath := filepath.Join(projectPath, "go.mod")

		// Check if this directory contains a go.mod file
		if _, err := os.Stat(goModPath); err == nil {
			// This is a Go project
			projects = append(projects, project{
				name: entry.Name(),
				path: projectPath,
			})
		} else {
			// If no go.mod at this level, scan subdirectories (e.g., utils/*)
			subEntries, err := os.ReadDir(projectPath)
			if err != nil {
				// Skip if we can't read the subdirectory
				continue
			}

			for _, subEntry := range subEntries {
				if !subEntry.IsDir() {
					continue
				}

				subProjectPath := filepath.Join(projectPath, subEntry.Name())
				subGoModPath := filepath.Join(subProjectPath, "go.mod")

				if _, err := os.Stat(subGoModPath); err == nil {
					// This is a Go project in a subfolder
					projects = append(projects, project{
						name: subEntry.Name(),
						path: subProjectPath,
					})
				}
			}
		}
	}

	if len(projects) == 0 {
		return nil, fmt.Errorf("no Go projects found in %s", sourceDir)
	}

	return projects, nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.quitting {
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.projects)-1 {
				m.cursor++
			}

		case "enter", " ":
			if m.selected == -1 {
				m.selected = m.cursor
				m.status = "Building..."
				return m, m.buildProject(m.cursor)
			}

		case "esc":
			if m.selected != -1 {
				m.selected = -1
				m.status = ""
			}
		}

	case buildResultMsg:
		m.status = msg.message
		m.selected = -1 // Reset selection after build
		return m, nil
	}

	return m, nil
}

func (m model) buildProject(selectedIdx int) tea.Cmd {
	return func() tea.Msg {
		if selectedIdx < 0 || selectedIdx >= len(m.projects) {
			return buildResultMsg{success: false, message: "Invalid project selection"}
		}

		proj := m.projects[selectedIdx]

		// Build the project
		buildCmd := exec.Command("go", "build", "-o", proj.name, ".")
		buildCmd.Dir = proj.path
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr

		if err := buildCmd.Run(); err != nil {
			return buildResultMsg{
				success: false,
				message: fmt.Sprintf("Build failed: %v", err),
			}
		}

		sourcePath := filepath.Join(proj.path, proj.name)

		// Copy to all target directories
		var successDirs []string
		for i, targetDir := range m.targetDirs {
			destPath := filepath.Join(targetDir, proj.name)

			// Create target directory if it doesn't exist
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return buildResultMsg{
					success: false,
					message: fmt.Sprintf("Failed to create target directory %s: %v", targetDir, err),
				}
			}

			// Remove old binary if it exists
			if _, err := os.Stat(destPath); err == nil {
				if err := os.Remove(destPath); err != nil {
					return buildResultMsg{
						success: false,
						message: fmt.Sprintf("Failed to remove old binary at %s: %v", destPath, err),
					}
				}
			}

			// For the first target, move the binary; for others, copy it
			if i == 0 {
				if err := os.Rename(sourcePath, destPath); err != nil {
					return buildResultMsg{
						success: false,
						message: fmt.Sprintf("Failed to move binary to %s: %v", targetDir, err),
					}
				}
				// Update source path for subsequent copies
				sourcePath = destPath
			} else {
				// Copy the file
				if err := copyFile(sourcePath, destPath); err != nil {
					return buildResultMsg{
						success: false,
						message: fmt.Sprintf("Failed to copy binary to %s: %v", targetDir, err),
					}
				}
			}

			successDirs = append(successDirs, targetDir)
		}

		// Build success message
		var message string
		if len(successDirs) == 1 {
			message = fmt.Sprintf("Successfully built and moved %s to %s", proj.name, successDirs[0])
		} else {
			message = fmt.Sprintf("Successfully built %s and copied to %d locations", proj.name, len(successDirs))
		}

		return buildResultMsg{
			success: true,
			message: message,
		}
	}
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := destFile.ReadFrom(sourceFile); err != nil {
		return err
	}

	// Copy permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, sourceInfo.Mode())
}

type buildResultMsg struct {
	success bool
	message string
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("Select project to build\n\n"))

	for i, proj := range m.projects {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		style := itemStyle
		if m.cursor == i {
			style = selectedStyle
		}

		b.WriteString(fmt.Sprintf("%s %s\n", cursor, style.Render(proj.name)))
	}

	b.WriteString("\n")

	if m.status != "" {
		if strings.Contains(m.status, "Successfully") {
			b.WriteString(successStyle.Render(m.status))
		} else if strings.Contains(m.status, "failed") || strings.Contains(m.status, "Failed") {
			b.WriteString(errorStyle.Render(m.status))
		} else {
			b.WriteString(m.status)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString("↑/↓: navigate • enter: build • q: quit\n")

	return b.String()
}

func main() {
	m, err := initialModel()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
