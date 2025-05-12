package execution

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindExecutable(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "exec-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a mock executable file
	mockExecPath := filepath.Join(tempDir, "mock-exec")
	if err := os.WriteFile(mockExecPath, []byte("#!/bin/sh\necho 'Hello, World!'"), 0755); err != nil {
		t.Fatalf("Failed to create mock executable: %v", err)
	}

	tests := []struct {
		name           string
		executable     string
		searchPaths    []string
		expectError    bool
		expectedToFind bool
	}{
		{
			name:           "Find in search path",
			executable:     "mock-exec",
			searchPaths:    []string{tempDir},
			expectError:    false,
			expectedToFind: true,
		},
		{
			name:           "Not found in search path",
			executable:     "nonexistent-exec",
			searchPaths:    []string{tempDir},
			expectError:    true,
			expectedToFind: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := FindExecutable(tt.executable, tt.searchPaths)

			if tt.expectError && err == nil {
				t.Errorf("FindExecutable() expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("FindExecutable() error = %v", err)
			}

			if tt.expectedToFind && path == "" {
				t.Errorf("FindExecutable() expected to find executable, got empty path")
			}

			if tt.expectedToFind && path != filepath.Join(tempDir, tt.executable) {
				t.Errorf("FindExecutable() got = %v, want %v", path, filepath.Join(tempDir, tt.executable))
			}
		})
	}
}

// TestRunCommand is limited because we can't easily verify command execution in a test
// This is more of a smoke test to ensure the function doesn't panic
func TestRunCommand(t *testing.T) {
	// Skip actual execution in automated tests
	if os.Getenv("TEST_EXECUTION") != "1" {
		t.Skip("Skipping execution test")
	}

	// Create a command that will succeed
	cmd := CommandInfo{
		Name:         "test-echo",
		Description:  "Echo test command",
		IsEnabled:    true,
		Cmd:          "echo 'Test execution'",
		IsExecutable: false,
	}

	// Test execution - this mainly verifies that the function doesn't panic
	err := Run(cmd, "")
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
}
