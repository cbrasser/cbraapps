package config

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	DataDir       string `toml:"data_dir"`
	CourseNotesDir string `toml:"course_notes_dir"`
	ReviewsDir    string `toml:"reviews_dir"`
	SenderEmail   string `toml:"sender_email"`
}

func DefaultConfig() Config {
	homeDir, _ := os.UserHomeDir()
	configBase := filepath.Join(homeDir, ".config", "cbraapps")

	return Config{
		DataDir:       filepath.Join(configBase, "cbrateach", "data"),
		CourseNotesDir: filepath.Join(configBase, "cbrateach", "notes"),
		ReviewsDir:    filepath.Join(configBase, "cbrateach", "reviews"),
		SenderEmail:   "teacher@example.com",
	}
}

func ConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".config", "cbraapps", "cbrateach.toml")
}

func Load() (Config, error) {
	path := ConfigPath()

	// If config doesn't exist, create default
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := DefaultConfig()
		if err := Save(cfg); err != nil {
			return cfg, err
		}
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func Save(cfg Config) error {
	path := ConfigPath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (c Config) EnsureDirectories() error {
	dirs := []string{c.DataDir, c.CourseNotesDir, c.ReviewsDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}
