package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestStruct is a test configuration structure
type TestStruct struct {
	Name        string            `toml:"name"`
	Value       int               `toml:"value"`
	IsEnabled   bool              `toml:"is_enabled"`
	StringMap   map[string]string `toml:"string_map"`
	StringSlice []string          `toml:"string_slice"`
}

func TestManager_GetConfigFilePath(t *testing.T) {
	// Create a test manager with custom path config
	manager := &Manager{
		PathConfig: PathConfig{
			SettingsDir:    ".test-config",
			AppDir:         "test-app",
			CfgFile:        "test-settings.toml",
			ExecutablesDir: "test-executables",
		},
	}

	// Get the config file path
	path, err := manager.GetConfigFilePath()
	if err != nil {
		t.Fatalf("GetConfigFilePath() error = %v", err)
	}

	// Check that the path contains the expected components
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home directory: %v", err)
	}

	expected := filepath.Join(homeDir, ".test-config", "test-app", "test-settings.toml")
	if path != expected {
		t.Errorf("GetConfigFilePath() got = %v, want %v", path, expected)
	}
}

func TestManager_SaveAndParseFromFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file path
	testPath := filepath.Join(tempDir, "test-config.toml")

	// Create a test manager
	manager := NewManager()

	// Create test data
	testData := TestStruct{
		Name:      "Test",
		Value:     42,
		IsEnabled: true,
		StringMap: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		StringSlice: []string{"item1", "item2", "item3"},
	}

	// Save the test data to file
	err = manager.SaveToFile(testPath, testData)
	if err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	// Parse the file back into a new structure
	var parsedData TestStruct
	err = manager.ParseFromFile(testPath, &parsedData)
	if err != nil {
		t.Fatalf("ParseFromFile() error = %v", err)
	}

	// Check that the parsed data matches the original
	if parsedData.Name != testData.Name {
		t.Errorf("Name: got = %v, want %v", parsedData.Name, testData.Name)
	}
	if parsedData.Value != testData.Value {
		t.Errorf("Value: got = %v, want %v", parsedData.Value, testData.Value)
	}
	if parsedData.IsEnabled != testData.IsEnabled {
		t.Errorf("IsEnabled: got = %v, want %v", parsedData.IsEnabled, testData.IsEnabled)
	}
	if len(parsedData.StringMap) != len(testData.StringMap) {
		t.Errorf("StringMap length: got = %v, want %v", len(parsedData.StringMap), len(testData.StringMap))
	}
	if len(parsedData.StringSlice) != len(testData.StringSlice) {
		t.Errorf("StringSlice length: got = %v, want %v", len(parsedData.StringSlice), len(testData.StringSlice))
	}
}

func TestManager_EnsureConfigFile(t *testing.T) {
	// Override the user home directory for testing
	originalUserHomeDir := os.UserHomeDir
	tmpDir, err := os.MkdirTemp("", "config-home-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Replace the UserHomeDir function with a mock
	os.UserHomeDir = func() (string, error) {
		return tmpDir, nil
	}
	// Restore the original function when the test is done
	defer func() {
		os.UserHomeDir = originalUserHomeDir
	}()

	// Create a test manager
	manager := NewManager()

	// Default config for testing
	defaultConfig := TestStruct{
		Name:      "Default",
		Value:     100,
		IsEnabled: true,
	}

	// Ensure config file exists
	path, err := manager.EnsureConfigFile(defaultConfig)
	if err != nil {
		t.Fatalf("EnsureConfigFile() error = %v", err)
	}

	// Check that the file was created
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Config file was not created at %s", path)
	}

	// Parse the file to check its content
	var parsedData TestStruct
	err = manager.ParseFromFile(path, &parsedData)
	if err != nil {
		t.Fatalf("ParseFromFile() error = %v", err)
	}

	// Verify default values were saved
	if parsedData.Name != defaultConfig.Name {
		t.Errorf("Name: got = %v, want %v", parsedData.Name, defaultConfig.Name)
	}
}
