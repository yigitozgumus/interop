package path

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpand(t *testing.T) {
	// Get the real home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home directory: %v", err)
	}

	tests := []struct {
		name          string
		path          string
		expectedBase  string
		expectAbsPath bool
	}{
		{
			name:          "Tilde path",
			path:          "~/testdir",
			expectedBase:  filepath.Join(homeDir, "testdir"),
			expectAbsPath: true,
		},
		{
			name:          "Absolute path",
			path:          "/tmp/testdir",
			expectedBase:  "/tmp/testdir",
			expectAbsPath: true,
		},
		{
			name:          "Relative path",
			path:          "testdir",
			expectedBase:  filepath.Join(homeDir, "testdir"),
			expectAbsPath: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expandedPath, err := Expand(tt.path)
			if err != nil {
				t.Fatalf("Expand() error = %v", err)
			}

			if expandedPath != tt.expectedBase {
				t.Errorf("Expand() got = %v, want %v", expandedPath, tt.expectedBase)
			}

			if tt.expectAbsPath && !filepath.IsAbs(expandedPath) {
				t.Errorf("Expected absolute path, got %v", expandedPath)
			}
		})
	}
}

func TestExpandAndValidate(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "path-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file in the temp directory
	testFilePath := filepath.Join(tempDir, "testfile.txt")
	if err := os.WriteFile(testFilePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get the real home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home directory: %v", err)
	}

	tests := []struct {
		name         string
		path         string
		expectExists bool
		expectHome   bool
	}{
		{
			name:         "Existing file",
			path:         testFilePath,
			expectExists: true,
			expectHome:   false, // temp dir is typically not in home
		},
		{
			name:         "Non-existent file",
			path:         filepath.Join(tempDir, "nonexistent.txt"),
			expectExists: false,
			expectHome:   false,
		},
		{
			name:         "User's home directory",
			path:         homeDir,
			expectExists: true,
			expectHome:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ExpandAndValidate(tt.path)
			if err != nil {
				t.Fatalf("ExpandAndValidate() error = %v", err)
			}

			if info.Exists != tt.expectExists {
				t.Errorf("ExpandAndValidate() exists = %v, want %v", info.Exists, tt.expectExists)
			}

			if info.InHomeDir != tt.expectHome {
				t.Errorf("ExpandAndValidate() inHome = %v, want %v", info.InHomeDir, tt.expectHome)
			}
		})
	}
}

func TestCreateDirectories(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "path-test-create")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test path to create
	testPath := filepath.Join(tempDir, "level1", "level2", "level3")

	// Create the directories
	if err := CreateDirectories(testPath); err != nil {
		t.Fatalf("CreateDirectories() error = %v", err)
	}

	// Check if the directories were created
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Errorf("CreateDirectories() failed to create directory: %v", testPath)
	}
}
