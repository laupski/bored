package tui

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AppConfig represents the application configuration stored in a file
type AppConfig struct {
	// General settings
	DefaultShowAll      bool `toml:"default_show_all"`     // Default value for "show all" toggle on board
	EnableNotifications bool `toml:"enable_notifications"` // Enable sound notifications for work item changes

	// Display settings
	MaxWorkItems int `toml:"max_work_items"` // Maximum work items to fetch (default 50)
}

// DefaultConfig returns a new AppConfig with default values
func DefaultConfig() AppConfig {
	return AppConfig{
		DefaultShowAll:      false,
		EnableNotifications: true, // Enable by default
		MaxWorkItems:        50,
	}
}

// getConfigDir returns the appropriate config directory for the current OS
// - Windows: %APPDATA%\bored
// - macOS: ~/Library/Application Support/bored
// - Linux/other: ~/.config/bored
func getConfigDir() (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "windows":
		// Use APPDATA on Windows
		appData := os.Getenv("APPDATA")
		if appData == "" {
			// Fallback to home directory
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		configDir = filepath.Join(appData, "bored")
	case "darwin":
		// Use ~/Library/Application Support on macOS
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, "Library", "Application Support", "bored")
	default:
		// Use XDG_CONFIG_HOME or ~/.config on Linux and other Unix-like systems
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			xdgConfig = filepath.Join(home, ".config")
		}
		configDir = filepath.Join(xdgConfig, "bored")
	}

	return configDir, nil
}

// getConfigFilePath returns the full path to the config file
func getConfigFilePath() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.toml"), nil
}

// LoadConfigFile loads the application configuration from the config file
func LoadConfigFile() (AppConfig, error) {
	configPath, err := getConfigFilePath()
	if err != nil {
		return DefaultConfig(), err
	}

	var config AppConfig
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return DefaultConfig(), nil
		}
		return DefaultConfig(), err
	}

	// Apply defaults for any zero values (in case config file is from older version)
	if config.MaxWorkItems == 0 {
		config.MaxWorkItems = 50
	}

	return config, nil
}

// SaveConfigFile saves the application configuration to the config file
func SaveConfigFile(config AppConfig) error {
	configPath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	// Ensure the config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return err
	}

	// Encode config to TOML
	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(config); err != nil {
		return err
	}

	return os.WriteFile(configPath, buf.Bytes(), 0600)
}

// GetConfigFilePath returns the config file path for display purposes
func GetConfigFilePath() string {
	path, err := getConfigFilePath()
	if err != nil {
		return "unknown"
	}
	return path
}

// ConfigFileExists checks if the config file exists
func ConfigFileExists() bool {
	configPath, err := getConfigFilePath()
	if err != nil {
		return false
	}
	_, err = os.Stat(configPath)
	return err == nil
}

// updateConfigFile handles input for the config file screen
func (m Model) updateConfigFile(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Return to credential screen without saving
			m.view = ViewConfig
			m.appConfigMessage = ""
			return m, nil
		case "tab", "down":
			m.configFileFocus = (m.configFileFocus + 1) % 3
			return m, m.updateConfigFileFocus()
		case "shift+tab", "up":
			m.configFileFocus--
			if m.configFileFocus < 0 {
				m.configFileFocus = 2
			}
			return m, m.updateConfigFileFocus()
		case "enter", " ":
			// Toggle boolean options
			if m.configFileFocus == 0 { // DefaultShowAll
				m.appConfig.DefaultShowAll = !m.appConfig.DefaultShowAll
				return m, nil
			}
			if m.configFileFocus == 1 { // EnableNotifications
				m.appConfig.EnableNotifications = !m.appConfig.EnableNotifications
				return m, nil
			}
		case "ctrl+s":
			// Save config
			return m.saveConfigFile()
		}
	}

	// Handle text input for MaxWorkItems field
	if m.configFileFocus == 2 {
		cmd := m.updateConfigFileInputs(msg)
		return m, cmd
	}

	return m, nil
}

// updateConfigFileFocus updates focus state for config file inputs
func (m *Model) updateConfigFileFocus() tea.Cmd {
	if m.configFileFocus == 2 { // MaxWorkItems
		return m.configFileInputs[0].Focus()
	}
	m.configFileInputs[0].Blur()
	return nil
}

// updateConfigFileInputs updates the text inputs for config file
func (m *Model) updateConfigFileInputs(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.configFileInputs[0], cmd = m.configFileInputs[0].Update(msg)
	return cmd
}

// saveConfigFile saves the current config to file
func (m Model) saveConfigFile() (tea.Model, tea.Cmd) {
	// Parse MaxWorkItems from input
	maxItemsStr := strings.TrimSpace(m.configFileInputs[0].Value())
	if maxItemsStr != "" {
		if val, err := strconv.Atoi(maxItemsStr); err == nil && val > 0 {
			m.appConfig.MaxWorkItems = val
		}
	}

	// Save to file
	if err := SaveConfigFile(m.appConfig); err != nil {
		m.appConfigMessage = fmt.Sprintf("Error saving config: %v", err)
	} else {
		m.appConfigMessage = "Configuration saved successfully"
		// Apply settings immediately
		m.showAll = m.appConfig.DefaultShowAll
	}

	return m, nil
}

// viewConfigFile renders the config file screen
func (m Model) viewConfigFile() string {
	var b strings.Builder

	title := titleStyle.Render("Application Settings")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Config file path
	configPath := GetConfigFilePath()
	pathStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
	b.WriteString(pathStyle.Render(fmt.Sprintf("Config file: %s", configPath)))
	b.WriteString("\n\n")

	// Settings
	settings := []struct {
		label       string
		description string
	}{
		{"Default Show All", "Show all work items by default (not just yours)"},
		{"Enable Notifications", "Play sound when assigned work items change"},
		{"Max Work Items", "Maximum number of work items to fetch"},
	}

	for i, setting := range settings {
		style := labelStyle
		if i == m.configFileFocus {
			style = style.Foreground(lipgloss.Color("229"))
		}

		b.WriteString(style.Render(setting.label))
		b.WriteString("\n")

		// Render the control based on type
		switch i {
		case 0: // DefaultShowAll (checkbox)
			checkbox := "[ ]"
			if m.appConfig.DefaultShowAll {
				checkbox = "[x]"
			}
			if i == m.configFileFocus {
				b.WriteString(selectedStyle.Render(checkbox))
			} else {
				b.WriteString(normalStyle.Render(checkbox))
			}
		case 1: // EnableNotifications (checkbox)
			checkbox := "[ ]"
			if m.appConfig.EnableNotifications {
				checkbox = "[x]"
			}
			if i == m.configFileFocus {
				b.WriteString(selectedStyle.Render(checkbox))
			} else {
				b.WriteString(normalStyle.Render(checkbox))
			}
		case 2: // MaxWorkItems (text input)
			b.WriteString(m.configFileInputs[0].View())
		}

		b.WriteString("\n")
		descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		b.WriteString(descStyle.Render(setting.description))
		b.WriteString("\n\n")
	}

	// Messages
	if m.appConfigMessage != "" {
		if strings.Contains(m.appConfigMessage, "Error") {
			b.WriteString(errorStyle.Render("❌ " + m.appConfigMessage))
		} else {
			b.WriteString(successStyle.Render("✓ " + m.appConfigMessage))
		}
		b.WriteString("\n\n")
	}

	// Help
	b.WriteString(helpStyle.Render("tab/↑↓: navigate • space/enter: toggle • ctrl+s: save • esc: back"))

	return boxStyle.Render(b.String())
}
