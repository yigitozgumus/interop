package project

import (
	"interop/internal/settings"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidator_ValidateAll(t *testing.T) {
	// Create a temporary directory for testing
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home directory: %v", err)
	}

	// Create a test project directory
	validProjectDir := filepath.Join(homeDir, "test-valid-project")
	defer os.RemoveAll(validProjectDir)
	if err := os.MkdirAll(validProjectDir, 0755); err != nil {
		t.Fatalf("Failed to create test project directory: %v", err)
	}

	// Create test settings
	testSettings := &settings.Settings{
		Projects: map[string]settings.Project{
			"valid-project": {
				Path:        validProjectDir,
				Description: "Valid project",
				Commands: []settings.Alias{
					{CommandName: "valid-cmd", Alias: "vcmd"},
				},
			},
			"invalid-path": {
				Path:        "/invalid/path",
				Description: "Invalid path",
			},
			"invalid-command": {
				Path:        validProjectDir,
				Description: "Invalid command reference",
				Commands: []settings.Alias{
					{CommandName: "non-existent-cmd", Alias: "bad"},
				},
			},
		},
		Commands: map[string]settings.CommandConfig{
			"valid-cmd": {
				Description: "Valid command",
				IsEnabled:   true,
				Cmd:         "echo 'valid'",
			},
		},
	}

	// Create validator
	validator := NewValidator(testSettings)

	// Test validation
	result := validator.ValidateAll()

	// Should have validation errors
	if result.Valid {
		t.Errorf("Expected validation to fail but it passed")
	}

	// Should have errors for invalid path and invalid command
	if len(result.Errors) < 2 {
		t.Errorf("Expected at least 2 validation errors, got %d", len(result.Errors))
	}

	// Check for specific error types
	foundPathError := false
	foundCmdError := false
	for _, err := range result.Errors {
		errStr := err.Error()
		if strings.Contains(errStr, "Project 'invalid-path' path does not exist") {
			foundPathError = true
		}
		if strings.Contains(errStr, "Project 'invalid-command' references undefined command: non-existent-cmd") {
			foundCmdError = true
		}
	}

	if !foundPathError {
		t.Errorf("Expected path validation error but did not find it")
	}
	if !foundCmdError {
		t.Errorf("Expected command validation error but did not find it")
	}
}

func TestValidator_ValidateProject(t *testing.T) {
	// Create a temporary directory for testing
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home directory: %v", err)
	}

	// Create a test project directory
	validProjectDir := filepath.Join(homeDir, "test-valid-project-single")
	defer os.RemoveAll(validProjectDir)
	if err := os.MkdirAll(validProjectDir, 0755); err != nil {
		t.Fatalf("Failed to create test project directory: %v", err)
	}

	// Create test settings
	testSettings := &settings.Settings{
		Projects: map[string]settings.Project{
			"valid-project": {
				Path:        validProjectDir,
				Description: "Valid project",
				Commands: []settings.Alias{
					{CommandName: "valid-cmd", Alias: "vcmd"},
				},
			},
			"invalid-command": {
				Path:        validProjectDir,
				Description: "Invalid command reference",
				Commands: []settings.Alias{
					{CommandName: "non-existent-cmd", Alias: "bad"},
				},
			},
		},
		Commands: map[string]settings.CommandConfig{
			"valid-cmd": {
				Description: "Valid command",
				IsEnabled:   true,
				Cmd:         "echo 'valid'",
			},
		},
	}

	// Create validator
	validator := NewValidator(testSettings)

	// Test validation with valid project
	validResult := validator.ValidateProject("valid-project")
	if !validResult.Valid {
		t.Errorf("Expected valid project to pass validation but it failed: %v", validResult.Errors)
	}

	// Test validation with invalid command reference
	invalidResult := validator.ValidateProject("invalid-command")
	if invalidResult.Valid {
		t.Errorf("Expected project with invalid command to fail validation but it passed")
	}

	// Test validation with non-existent project
	nonExistentResult := validator.ValidateProject("non-existent")
	if nonExistentResult.Valid {
		t.Errorf("Expected non-existent project to fail validation but it passed")
	}
}
