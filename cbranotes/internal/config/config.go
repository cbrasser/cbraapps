package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	RepoURL   string       `toml:"repo_url"`
	NotesPath string       `toml:"notes_path"`
	Editor    EditorConfig `toml:"editor"`
}

type EditorConfig struct {
	UseSystemEditor    bool      `toml:"use_system_editor"`
	EditorInMainWindow bool      `toml:"editor_in_main_window"`
	Hotkeys            HotkeyMap `toml:"hotkeys"`
}

type HotkeyMap struct {
	Save               string `toml:"save"`
	CloseFile          string `toml:"close_file"`
	SwitchToFilePicker string `toml:"switch_to_filepicker"`
	Quit               string `toml:"quit"`
}

// DefaultEditorConfig returns sensible defaults for the editor
func DefaultEditorConfig() EditorConfig {
	return EditorConfig{
		UseSystemEditor:    false,
		EditorInMainWindow: false,
		Hotkeys: HotkeyMap{
			Save:               "ctrl+s",
			CloseFile:          "ctrl+w",
			SwitchToFilePicker: "ctrl+p",
			Quit:               "ctrl+q",
		},
	}
}

func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "cbraapps", "cbranotes")
}

func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "cbraapps", "cbranotes.toml")
}

func Exists() bool {
	_, err := os.Stat(ConfigPath())
	return err == nil
}

func Load() (*Config, error) {
	var cfg Config
	_, err := toml.DecodeFile(ConfigPath(), &cfg)
	if err != nil {
		return nil, err
	}

	// Apply defaults for missing editor config
	defaults := DefaultEditorConfig()
	if cfg.Editor.Hotkeys.Save == "" {
		cfg.Editor.Hotkeys.Save = defaults.Hotkeys.Save
	}
	if cfg.Editor.Hotkeys.CloseFile == "" {
		cfg.Editor.Hotkeys.CloseFile = defaults.Hotkeys.CloseFile
	}
	if cfg.Editor.Hotkeys.SwitchToFilePicker == "" {
		cfg.Editor.Hotkeys.SwitchToFilePicker = defaults.Hotkeys.SwitchToFilePicker
	}
	if cfg.Editor.Hotkeys.Quit == "" {
		cfg.Editor.Hotkeys.Quit = defaults.Hotkeys.Quit
	}

	return &cfg, nil
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

