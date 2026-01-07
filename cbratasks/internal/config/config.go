package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	DefaultList string            `toml:"default_list"` // Default list for new tasks
	Sync        SyncConfig        `toml:"sync"`
	GitHub      GitHubConfig      `toml:"github"`
	Lists       map[string]string `toml:"lists"`   // list name -> color
	Tags        map[string]string `toml:"tags"`    // Deprecated: kept for migration only
	Hotkeys     HotkeyConfig      `toml:"hotkeys"`
}

type SyncConfig struct {
	Enabled  bool   `toml:"enabled"`
	Backend  string `toml:"backend"`  // "local", "radicale", or "nextcloud"
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
		DefaultList: "inbox",
		Sync: SyncConfig{
			Enabled:  false,
			Backend:  "radicale",
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
		Lists: map[string]string{
			"inbox":    "#95E1D3", // mint
			"work":     "#FF6B6B", // red
			"personal": "#4ECDC4", // teal
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
# Supported sync backends: "radicale", "nextcloud", or "local" (default)
# For NextCloud, set backend = "nextcloud" and URL to your NextCloud instance (e.g., https://nextcloud.example.com)
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

// GetListColor returns the color for a list, or a default gray if not found
func (c *Config) GetListColor(list string) string {
	if color, ok := c.Lists[list]; ok {
		return color
	}
	return "#888888"
}

// ListExists checks if a list is defined in the config
func (c *Config) ListExists(list string) bool {
	_, ok := c.Lists[list]
	return ok
}

// AddList adds a new list with a color
func (c *Config) AddList(name, color string) {
	if c.Lists == nil {
		c.Lists = make(map[string]string)
	}
	c.Lists[name] = color
}

// RemoveList removes a list from the config
func (c *Config) RemoveList(name string) {
	delete(c.Lists, name)
}

// GetAllLists returns all list names
func (c *Config) GetAllLists() []string {
	lists := make([]string, 0, len(c.Lists))
	for name := range c.Lists {
		lists = append(lists, name)
	}
	return lists
}

// MigrateTagsToLists migrates old tag configuration to lists
func (c *Config) MigrateTagsToLists() {
	// If we already have lists, migration already done
	if len(c.Lists) > 0 {
		return
	}

	// Migrate tags to lists
	if len(c.Tags) > 0 {
		c.Lists = make(map[string]string)
		for tag, color := range c.Tags {
			c.Lists[tag] = color
		}
		c.Tags = nil // Clear tags after migration
	}

	// Ensure we have a default list
	if c.DefaultList == "" || c.DefaultList == "local" || c.DefaultList == "radicale" {
		c.DefaultList = "inbox"
	}

	// Ensure default list exists in lists
	if !c.ListExists(c.DefaultList) {
		c.AddList(c.DefaultList, "#95E1D3")
	}
}

// GetTagColor returns the color for a tag (deprecated - kept for backward compatibility)
func (c *Config) GetTagColor(tag string) string {
	if color, ok := c.Tags[tag]; ok {
		return color
	}
	// Fall back to list colors
	return c.GetListColor(tag)
}

func (c *Config) String() string {
	return fmt.Sprintf("Config{DefaultList: %s, SyncEnabled: %v, Lists: %d}", c.DefaultList, c.Sync.Enabled, len(c.Lists))
}
