package remote

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateGitURL(t *testing.T) {
	manager := NewManager()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		// Valid HTTPS URLs
		{"GitHub HTTPS with .git", "https://github.com/user/repo.git", false},
		{"GitHub HTTPS without .git", "https://github.com/user/repo", false},
		{"GitLab HTTPS with .git", "https://gitlab.com/user/repo.git", false},
		{"GitLab HTTPS without .git", "https://gitlab.com/user/repo", false},
		{"Bitbucket HTTPS", "https://bitbucket.org/user/repo.git", false},
		{"Codeberg HTTPS", "https://codeberg.org/user/repo.git", false},
		{"SourceHut HTTPS", "https://git.sr.ht/~user/repo", false},

		// Valid SSH URLs
		{"GitHub SSH", "git@github.com:user/repo.git", false},
		{"GitLab SSH", "git@gitlab.com:user/repo.git", false},
		{"Custom host SSH", "git@git.example.com:user/repo.git", false},

		// Valid unknown host with .git
		{"Unknown host with .git", "https://git.example.com/user/repo.git", false},

		// Invalid URLs
		{"Empty URL", "", true},
		{"Invalid scheme", "ftp://github.com/user/repo.git", true},
		{"No protocol", "github.com/user/repo.git", true},
		{"Known host without proper path", "https://github.com/invalid", true},
		{"Known host with too many path segments", "https://github.com/user/repo/extra/path", true},
		{"Unknown host without .git", "https://git.example.com/user/repo", true},
		{"Invalid SSH format", "git@github.com/user/repo.git", true},
		{"Malformed URL", "https://[invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.validateGitURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateGitURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMakeExecutablesExecutable(t *testing.T) {
	manager := NewManager()

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "interop-executable-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files with different permissions
	testFiles := []struct {
		name         string
		initialMode  os.FileMode
		expectedMode os.FileMode
	}{
		{"script.sh", 0644, 0755},          // rw-r--r-- -> rwxr-xr-x
		{"binary", 0600, 0700},             // rw------- -> rwx------
		{"readonly", 0444, 0555},           // r--r--r-- -> r-xr-xr-x
		{"no-permissions", 0000, 0000},     // --------- -> --------- (no change)
		{"already-executable", 0755, 0755}, // rwxr-xr-x -> rwxr-xr-x (no change)
	}

	// Create test files
	for _, tf := range testFiles {
		filePath := filepath.Join(tmpDir, tf.name)

		// Create the file
		file, err := os.Create(filePath)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", tf.name, err)
		}
		file.WriteString("#!/bin/bash\necho 'test'\n")
		file.Close()

		// Set initial permissions
		if err := os.Chmod(filePath, tf.initialMode); err != nil {
			t.Fatalf("Failed to set initial permissions for %s: %v", tf.name, err)
		}
	}

	// Create a subdirectory with a file to test recursive behavior
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	subFile := filepath.Join(subDir, "nested-script.py")
	file, err := os.Create(subFile)
	if err != nil {
		t.Fatalf("Failed to create nested file: %v", err)
	}
	file.WriteString("#!/usr/bin/env python3\nprint('test')\n")
	file.Close()

	if err := os.Chmod(subFile, 0644); err != nil {
		t.Fatalf("Failed to set permissions for nested file: %v", err)
	}

	// Run makeExecutablesExecutable
	if err := manager.makeExecutablesExecutable(tmpDir); err != nil {
		t.Fatalf("makeExecutablesExecutable failed: %v", err)
	}

	// Verify permissions were set correctly
	for _, tf := range testFiles {
		filePath := filepath.Join(tmpDir, tf.name)

		info, err := os.Stat(filePath)
		if err != nil {
			t.Fatalf("Failed to stat file %s: %v", tf.name, err)
		}

		actualMode := info.Mode().Perm()
		if actualMode != tf.expectedMode {
			t.Errorf("File %s: expected permissions %o, got %o", tf.name, tf.expectedMode, actualMode)
		}
	}

	// Verify nested file permissions
	info, err := os.Stat(subFile)
	if err != nil {
		t.Fatalf("Failed to stat nested file: %v", err)
	}

	expectedNestedMode := os.FileMode(0755)
	actualNestedMode := info.Mode().Perm()
	if actualNestedMode != expectedNestedMode {
		t.Errorf("Nested file: expected permissions %o, got %o", expectedNestedMode, actualNestedMode)
	}

	// Verify directory permissions weren't changed
	dirInfo, err := os.Stat(subDir)
	if err != nil {
		t.Fatalf("Failed to stat subdirectory: %v", err)
	}

	if !dirInfo.IsDir() {
		t.Error("Subdirectory should still be a directory")
	}
}
