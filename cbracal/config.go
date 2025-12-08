package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Default configuration template
const defaultConfigTOML = `# cbracal - Calendar Configuration
#
# This is the default configuration file. Customize it according to your needs.

# Radicale CalDAV server configuration (optional)
# Uncomment and fill in your details to sync with a Radicale server
# [radicale]
# server_url = "https://your-radicale-server.com"
# username = "your-username"
# password = "your-password"

# Additional calendars from URLs or local files
# [[calendars]]
# name = "Public Holidays"
# url = "https://example.com/holidays.ics"
# type = "url"
#
# [[calendars]]
# name = "Personal"
# file = "personal.ics"
# type = "file"

# Local .ics files in the config directory
# local_calendars = ["work.ics", "personal.ics"]

# Notification daemon settings (for cbracal --daemon mode)
[notifications]
enabled = true
check_interval = 60          # seconds between checking for upcoming events
advance_notice = [15, 5, 1]  # minutes before event to send notifications
reload_interval = 5          # minutes between full calendar reloads
`

func getConfigDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(usr.HomeDir, ".config", "cbraapps")
	return configDir, nil
}

func getDataDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	dataDir := filepath.Join(usr.HomeDir, ".config", "cbraapps", "cbracal")
	return dataDir, nil
}

// createDefaultConfig creates the config directory and default config file if they don't exist
func createDefaultConfig() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %v", err)
	}

	// Also create data directory for local .ics files
	dataDir, err := getDataDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create data directory: %v", err)
	}

	configPath := filepath.Join(configDir, "cbracal.toml")

	// Check if config file already exists
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil // Config exists, don't overwrite
	}

	// Create default config file
	if err := os.WriteFile(configPath, []byte(defaultConfigTOML), 0644); err != nil {
		return "", fmt.Errorf("failed to create default config: %v", err)
	}

	return configPath, nil
}

func loadConfig() (*Config, error) {
	// Try current directory first (dev mode) - support TOML
	localConfigs := []string{"config.toml"}
	for _, localConfig := range localConfigs {
		if _, err := os.Stat(localConfig); err == nil {
			var config Config
			if _, err := toml.DecodeFile(localConfig, &config); err != nil {
				return nil, err
			}
			return &config, nil
		}
	}

	// Fall back to standard config directory (build version)
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %v", err)
	}

	configPath := filepath.Join(configDir, "cbracal.toml")
	if _, err := os.Stat(configPath); err == nil {
		var config Config
		if _, err := toml.DecodeFile(configPath, &config); err != nil {
			return nil, err
		}
		return &config, nil
	}

	// No config found - create default config
	configPath, err = createDefaultConfig()
	if err != nil {
		return nil, err
	}

	// Load the newly created default config
	var config Config
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
