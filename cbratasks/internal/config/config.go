package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	DefaultList string            `toml:"default_list"`
	Sync        SyncConfig        `toml:"sync"`
	GitHub      GitHubConfig      `toml:"github"`
	Tags        map[string]string `toml:"tags"` // tag name -> color
	Hotkeys     HotkeyConfig      `toml:"hotkeys"`
}

type SyncConfig struct {
	Enabled  bool   `toml:"enabled"`
	URL      string `toml:"url"`
	Username string `toml:"username"`
	Password string `toml:"password"`
}

type GitHubConfig struct {
	Enabled  bool     `toml:"enabled"`
	Username string   `toml:"username"`
	Token    string   `toml:"token"`
	Repos    []string `toml:"repos"` // List of repos for creating issues
}

type HotkeyConfig struct {
	MarkComplete string `toml:"mark_complete"`
	Delete       string `toml:"delete"`
	EditNote     string `toml:"edit_note"`
	ViewNote     string `toml:"view_note"`
	AddTask      string `toml:"add_task"`
	Search       string `toml:"search"`
	Quit         string `toml:"quit"`
}

func DefaultConfig() Config {
	return Config{
		DefaultList: "local",
		Sync: SyncConfig{
			Enabled:  false,
			URL:      "https://radicale.example.com",
			Username: "",
			Password: "",
		},
		GitHub: GitHubConfig{
			Enabled:  false,
			Username: "",
			Token:    "",
			Repos:    []string{},
		},
		Tags: map[string]string{
			"work":     "#FF6B6B", // red
			"home":     "#4ECDC4", // teal
			"personal": "#95E1D3", // mint
			"urgent":   "#F38181", // coral
			"shopping": "#AA96DA", // purple
		},
		Hotkeys: HotkeyConfig{
			MarkComplete: "x",
			Delete:       "d",
			EditNote:     "n",
			ViewNote:     "tab",
			AddTask:      "a",
			Search:       "/",
			Quit:         "q",
		},
	}
}

func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "cbraapps", "cbratasks")
}

func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "cbraapps", "cbratasks.toml")
}

func DataDir() string {
	return filepath.Join(ConfigDir(), "data")
}

func Exists() bool {
	_, err := os.Stat(ConfigPath())
	return err == nil
}

func Load() (*Config, error) {
	if !Exists() {
		// Create default config
		if err := createDefaultConfig(); err != nil {
			return nil, err
		}
	}

	var cfg Config
	_, err := toml.DecodeFile(ConfigPath(), &cfg)
	if err != nil {
		return nil, err
	}

	// Apply defaults for missing values
	defaults := DefaultConfig()
	if cfg.Hotkeys.MarkComplete == "" {
		cfg.Hotkeys.MarkComplete = defaults.Hotkeys.MarkComplete
	}
	if cfg.Hotkeys.Delete == "" {
		cfg.Hotkeys.Delete = defaults.Hotkeys.Delete
	}
	if cfg.Hotkeys.EditNote == "" {
		cfg.Hotkeys.EditNote = defaults.Hotkeys.EditNote
	}
	if cfg.Hotkeys.ViewNote == "" {
		cfg.Hotkeys.ViewNote = defaults.Hotkeys.ViewNote
	}
	if cfg.Hotkeys.AddTask == "" {
		cfg.Hotkeys.AddTask = defaults.Hotkeys.AddTask
	}
	if cfg.Hotkeys.Search == "" {
		cfg.Hotkeys.Search = defaults.Hotkeys.Search
	}
	if cfg.Hotkeys.Quit == "" {
		cfg.Hotkeys.Quit = defaults.Hotkeys.Quit
	}
	if cfg.Tags == nil {
		cfg.Tags = defaults.Tags
	}

	return &cfg, nil
}

func createDefaultConfig() error {
	// Create parent config directory
	configPath := ConfigPath()
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Create app-specific directory for data
	if err := os.MkdirAll(ConfigDir(), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(DataDir(), 0755); err != nil {
		return err
	}

	cfg := DefaultConfig()

	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write header comment
	header := `# cbratasks configuration
# Auto-generated on first run

# Default task list: "local" or "radicale" (if sync enabled)
`
	f.WriteString(header)

	return toml.NewEncoder(f).Encode(cfg)
}

func Save(cfg *Config) error {
	configPath := ConfigPath()
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return toml.NewEncoder(f).Encode(cfg)
}

// GetTagColor returns the color for a tag, or a default gray if not found
func (c *Config) GetTagColor(tag string) string {
	if color, ok := c.Tags[tag]; ok {
		return color
	}
	return "#888888"
}

func (c *Config) String() string {
	return fmt.Sprintf("Config{DefaultList: %s, SyncEnabled: %v}", c.DefaultList, c.Sync.Enabled)
}
