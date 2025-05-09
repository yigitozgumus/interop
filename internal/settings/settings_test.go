package settings

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// testEnv provides a way to create test settings
type testEnv struct {
	tempDir        string
	settingsPath   string
	origPathConfig PathConfig
}

// setupTestEnv creates a temporary environment for testing
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	// Save original path config
	origPathConfig := pathConfig

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "settings-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Setup test path config using the same directory names
	testPathConfig := PathConfig{
		SettingsDir:    origPathConfig.SettingsDir,
		AppDir:         origPathConfig.AppDir,
		CfgFile:        origPathConfig.CfgFile,
		ExecutablesDir: origPathConfig.ExecutablesDir,
	}

	// Set the path config to use our test environment
	SetPathConfig(testPathConfig)

	// Create the directory structure in temp
	testConfigDir := filepath.Join(tempDir, testPathConfig.SettingsDir)
	testAppDir := filepath.Join(testConfigDir, testPathConfig.AppDir)
	err = os.MkdirAll(testAppDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test app dir: %v", err)
	}

	// Reset singleton for testing
	once = sync.Once{}
	cfg = nil
	err = nil

	env := &testEnv{
		tempDir:        tempDir,
		settingsPath:   filepath.Join(testAppDir, testPathConfig.CfgFile),
		origPathConfig: origPathConfig,
	}

	// Mock the UserHomeDir function using monkeypatch
	// This is done by setting a test home directory environment variable
	os.Setenv("HOME", tempDir)

	return env
}

// teardownTestEnv cleans up the test environment
func (env *testEnv) teardown(t *testing.T) {
	t.Helper()

	// Restore original path config
	SetPathConfig(env.origPathConfig)

	// Remove temp directory
	os.RemoveAll(env.tempDir)

	// Reset the HOME environment variable
	os.Unsetenv("HOME")
}

// createTestSettings creates a test settings file with provided content
func (env *testEnv) createTestSettings(t *testing.T, content string) {
	t.Helper()

	err := os.WriteFile(env.settingsPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test settings: %v", err)
	}
}

func TestValidate(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown(t)

	// Create a valid test settings file
	testContent := `log_level = "info"
[projects]
`
	env.createTestSettings(t, testContent)

	// Now run validate()
	path, err := validate()
	if err != nil {
		t.Fatalf("validate() returned error: %v", err)
	}

	if path != env.settingsPath {
		t.Errorf("validate() returned unexpected path: got %v, want %v", path, env.settingsPath)
	}

	// Check if file exists
	_, err = os.Stat(path)
	if err != nil {
		t.Errorf("Settings file does not exist at %s: %v", path, err)
	}
}

func TestLoad(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown(t)

	// Create test settings with valid content
	testContent := `log_level = "debug"
[projects]
[projects.test]
path = "test-project"
description = "Test project"
`
	env.createTestSettings(t, testContent)

	settings, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if settings == nil {
		t.Error("Load() returned nil settings")
	}

	// Verify settings content
	if settings.LogLevel != "debug" {
		t.Errorf("Expected log level debug, got %s", settings.LogLevel)
	}

	if project, ok := settings.Projects["test"]; ok {
		if project.Path != "test-project" {
			t.Errorf("Expected project path test-project, got %s", project.Path)
		}
		if project.Description != "Test project" {
			t.Errorf("Expected project description 'Test project', got %s", project.Description)
		}
	} else {
		t.Error("Test project not found in settings")
	}
}

func TestGet(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown(t)

	// Create test settings
	testContent := `log_level = "info"
[projects]
`
	env.createTestSettings(t, testContent)

	settings := Get()
	if settings == nil {
		t.Error("Get() returned nil settings")
	}
	if settings.LogLevel != "info" {
		t.Errorf("Expected log level info, got %s", settings.LogLevel)
	}
}

func TestWithAndFrom(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown(t)

	testContent := `log_level = "debug"
[projects]
`
	env.createTestSettings(t, testContent)

	// Test With function
	ctx := context.Background()
	ctxWithSettings, err := With(ctx)
	if err != nil {
		t.Fatalf("With() returned error: %v", err)
	}

	// Test From function
	settings := From(ctxWithSettings)
	if settings == nil {
		t.Error("From() returned nil settings")
	}
	if settings.LogLevel != "debug" {
		t.Errorf("Expected log level debug, got %s", settings.LogLevel)
	}

	// Test From with context that doesn't have settings
	emptyCtx := context.Background()
	fallbackSettings := From(emptyCtx)
	if fallbackSettings == nil {
		t.Error("From() with empty context returned nil instead of fallback settings")
	}
}

func TestProjectPathValidation(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown(t)

	// Create a valid project inside our temp dir (which is now HOME)
	validProjectDir := "valid-project"
	validProjectPath := filepath.Join(env.tempDir, validProjectDir)
	err := os.MkdirAll(validProjectPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create test project dir: %v", err)
	}

	testContent := `log_level = "info"
[projects]
[projects.valid]
path = "` + validProjectDir + `"
description = "Valid project"
[projects.nonexistent]
path = "nonexistent-path"
description = "Path that doesn't exist"
[projects.outside]
path = "/tmp/outside-home"
description = "Path outside home"
`
	env.createTestSettings(t, testContent)

	// Load should not fail, but it should log errors for invalid paths
	settings, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	// We can't easily test the error logging directly, but we can verify
	// that all projects are still in the settings
	if len(settings.Projects) != 3 {
		t.Errorf("Expected 3 projects, got %d", len(settings.Projects))
	}
}
