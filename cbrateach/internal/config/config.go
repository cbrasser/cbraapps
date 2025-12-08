package config

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	DataDir        string `toml:"data_dir"`         // Hidden directory for internal app data
	CourseNotesDir string `toml:"course_notes_dir"`
	ExportDir      string `toml:"export_dir"`       // Directory for user-facing exports
	FeedbackDir    string `toml:"feedback_dir"`     // Directory for feedback files (export/email)
	SenderEmail    string `toml:"sender_email"`
	BCCEmail       string `toml:"bcc_email"`        // Email to BCC on first feedback email
}

func DefaultConfig() Config {
	homeDir, _ := os.UserHomeDir()
	configBase := filepath.Join(homeDir, ".config", "cbraapps")
	dataDir := filepath.Join(configBase, ".cbrateach") // Hidden directory

	return Config{
		DataDir:        dataDir,
		CourseNotesDir: filepath.Join(configBase, "cbrateach", "notes"),
		ExportDir:      filepath.Join(configBase, "cbrateach", "exports"),
		FeedbackDir:    filepath.Join(configBase, "cbrateach", "feedback"),
		SenderEmail:    "teacher@example.com",
		BCCEmail:       "claudio.brasser@gymneufeld.ch",
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

	// Use a temporary struct to handle migration from old config
	type OldConfig struct {
		DataDir        string `toml:"data_dir"`
		CourseNotesDir string `toml:"course_notes_dir"`
		ReviewsDir     string `toml:"reviews_dir"`     // Old field
		ExportDir      string `toml:"export_dir"`      // New field
		FeedbackDir    string `toml:"feedback_dir"`    // New field
		SenderEmail    string `toml:"sender_email"`
		CCEmail        string `toml:"cc_email"`        // Old field
		BCCEmail       string `toml:"bcc_email"`       // New field
	}

	var oldCfg OldConfig
	if err := toml.Unmarshal(data, &oldCfg); err != nil {
		return Config{}, err
	}

	// Migrate: if cc_email is set but bcc_email is not, use cc_email as bcc_email
	bccEmail := oldCfg.BCCEmail
	if bccEmail == "" && oldCfg.CCEmail != "" {
		bccEmail = oldCfg.CCEmail
	}

	// Migrate: if reviews_dir is set but export_dir is not, use reviews_dir as export_dir
	cfg := Config{
		DataDir:        oldCfg.DataDir,
		CourseNotesDir: oldCfg.CourseNotesDir,
		ExportDir:      oldCfg.ExportDir,
		FeedbackDir:    oldCfg.FeedbackDir,
		SenderEmail:    oldCfg.SenderEmail,
		BCCEmail:       bccEmail,
	}

	if cfg.ExportDir == "" && oldCfg.ReviewsDir != "" {
		cfg.ExportDir = oldCfg.ReviewsDir
		// Save migrated config
		Save(cfg)
	}

	// Ensure defaults for empty fields
	if cfg.DataDir == "" {
		cfg = DefaultConfig()
		cfg.SenderEmail = oldCfg.SenderEmail // Keep existing sender email
		Save(cfg)
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
	dirs := []string{c.DataDir, c.CourseNotesDir, c.ExportDir, c.FeedbackDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Ensure subdirectories in data_dir
	subdirs := []string{
		filepath.Join(c.DataDir, "reviews"),
		filepath.Join(c.DataDir, "mail_templates"),
	}
	for _, dir := range subdirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

// ReviewsDir returns the path to the reviews directory within data_dir
func (c Config) ReviewsDir() string {
	return filepath.Join(c.DataDir, "reviews")
}

// MailTemplatesDir returns the path to the mail templates directory within data_dir
func (c Config) MailTemplatesDir() string {
	return filepath.Join(c.DataDir, "mail_templates")
}

// EnsureDefaultEmailTemplate creates a default email template if one doesn't exist
func (c Config) EnsureDefaultEmailTemplate() error {
	templatePath := filepath.Join(c.MailTemplatesDir(), "feedback_template.txt")

	// Check if template already exists
	if _, err := os.Stat(templatePath); err == nil {
		return nil // Template already exists
	}

	// Create default template
	defaultTemplate := `Dear {{StudentName}},

Please find attached your feedback for the test "{{TestName}}" in course {{CourseName}}.

Your grade: {{Grade}}

Best regards`

	return os.WriteFile(templatePath, []byte(defaultTemplate), 0644)
}
