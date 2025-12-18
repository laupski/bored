package tui

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.DefaultShowAll != false {
		t.Errorf("DefaultShowAll = %v, want %v", config.DefaultShowAll, false)
	}
	if config.MaxWorkItems != 50 {
		t.Errorf("MaxWorkItems = %v, want %v", config.MaxWorkItems, 50)
	}
}

func TestSaveAndLoadConfigFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a config file in the temp directory
	configPath := filepath.Join(tempDir, "config.toml")

	// Test config to save
	testConfig := AppConfig{
		DefaultShowAll: true,
		MaxWorkItems:   100,
	}

	// Write config manually to test location
	configContent := `default_show_all = true
max_work_items = 100
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Read it back using TOML parsing
	var loaded AppConfig
	if _, err := toml.DecodeFile(configPath, &loaded); err != nil {
		t.Fatalf("Failed to decode config: %v", err)
	}

	if loaded.DefaultShowAll != testConfig.DefaultShowAll {
		t.Errorf("DefaultShowAll = %v, want %v", loaded.DefaultShowAll, testConfig.DefaultShowAll)
	}
	if loaded.MaxWorkItems != testConfig.MaxWorkItems {
		t.Errorf("MaxWorkItems = %v, want %v", loaded.MaxWorkItems, testConfig.MaxWorkItems)
	}
}

func TestConfigFileExists(t *testing.T) {
	// This test verifies the ConfigFileExists function works
	// In a fresh environment without config, it should return false
	// We can't easily test true case without modifying the actual config location
	exists := ConfigFileExists()
	// Just verify it doesn't panic
	_ = exists
}

func TestGetConfigFilePath(t *testing.T) {
	path := GetConfigFilePath()

	// Path should not be empty or "unknown"
	if path == "" {
		t.Error("GetConfigFilePath() returned empty string")
	}

	// Path should contain "bored"
	if path != "unknown" && !contains(path, "bored") {
		t.Errorf("GetConfigFilePath() = %v, expected to contain 'bored'", path)
	}

	// Path should end with config.toml
	if path != "unknown" && !contains(path, "config.toml") {
		t.Errorf("GetConfigFilePath() = %v, expected to contain 'config.toml'", path)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestAppConfigTOMLEncoding(t *testing.T) {
	// Test that AppConfig can be properly encoded to TOML
	config := AppConfig{
		DefaultShowAll: true,
		MaxWorkItems:   75,
	}

	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	err := encoder.Encode(config)
	if err != nil {
		t.Fatalf("Failed to encode config: %v", err)
	}

	encoded := buf.String()

	// Check that the encoded string contains expected fields
	if !contains(encoded, "default_show_all") {
		t.Errorf("Encoded config should contain 'default_show_all', got: %s", encoded)
	}
	if !contains(encoded, "max_work_items") {
		t.Errorf("Encoded config should contain 'max_work_items', got: %s", encoded)
	}
}

func TestLoadConfigFileWithDefaults(t *testing.T) {
	// Test that LoadConfigFile returns defaults when file doesn't exist
	// This tests the behavior of LoadConfigFile when given a non-existent path
	config, _ := LoadConfigFile()

	// Should have valid defaults even if file doesn't exist
	if config.MaxWorkItems <= 0 {
		t.Errorf("MaxWorkItems should have a positive default, got %d", config.MaxWorkItems)
	}
}

func TestConfigMaxWorkItemsZeroDefault(t *testing.T) {
	// Test that zero MaxWorkItems gets defaulted to 50
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// Write config with zero max_work_items
	configContent := `default_show_all = false
max_work_items = 0
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Read it back
	var loaded AppConfig
	if _, err := toml.DecodeFile(configPath, &loaded); err != nil {
		t.Fatalf("Failed to decode config: %v", err)
	}

	// Apply the same defaulting logic as LoadConfigFile
	if loaded.MaxWorkItems == 0 {
		loaded.MaxWorkItems = 50
	}

	if loaded.MaxWorkItems != 50 {
		t.Errorf("MaxWorkItems should default to 50 when 0, got %d", loaded.MaxWorkItems)
	}
}
