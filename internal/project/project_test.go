package project

import (
	"bytes"
	"interop/internal/settings"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestList tests the standard List function
func TestList(t *testing.T) {
	// Create a basic test with a single project to verify the List function works
	cfg := &settings.Settings{
		Projects: map[string]settings.Project{
			"test": {
				Path:        "test-project",
				Description: "Test project",
			},
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call function
	List(cfg)

	// Get output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Restore stdout
	os.Stdout = oldStdout

	// Basic verification
	if !strings.Contains(output, "test: test-project") {
		t.Errorf("Expected output to contain project information, got %q", output)
	}
}

func TestListWithCustomHomeDir(t *testing.T) {
	// Create a temporary home directory for testing
	tempHomeDir := t.TempDir()

	// Create a valid project directory inside the temp home
	validProjectDir := filepath.Join(tempHomeDir, "valid-project")
	err := os.MkdirAll(validProjectDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Define test cases
	tests := []struct {
		name           string
		projects       map[string]settings.Project
		homeDir        string
		expectedOutput []string
		notExpected    []string
	}{
		{
			name:           "No projects",
			projects:       map[string]settings.Project{},
			homeDir:        tempHomeDir,
			expectedOutput: []string{"No projects found."},
		},
		{
			name: "Valid project inside home",
			projects: map[string]settings.Project{
				"valid": {
					Path:        filepath.Join(tempHomeDir, "valid-project"),
					Description: "A valid project",
				},
			},
			homeDir: tempHomeDir,
			expectedOutput: []string{
				"PROJECTS:",
				"valid:",
				"[Valid: ✓]",
				"[In $HOME: ✓]",
				"Description: A valid project",
			},
		},
		{
			name: "Invalid path project",
			projects: map[string]settings.Project{
				"invalid": {
					Path:        filepath.Join(tempHomeDir, "non-existent"),
					Description: "A non-existent project",
				},
			},
			homeDir: tempHomeDir,
			expectedOutput: []string{
				"PROJECTS:",
				"invalid:",
				"[Valid: ✗]",
				"[In $HOME: ✓]",
				"Description: A non-existent project",
			},
		},
		{
			name: "Project outside home",
			projects: map[string]settings.Project{
				"outside": {
					Path:        "/tmp/outside-home",
					Description: "A project outside home",
				},
			},
			homeDir: tempHomeDir,
			expectedOutput: []string{
				"PROJECTS:",
				"outside:",
				"[In $HOME: ✗]",
				"Description: A project outside home",
			},
		},
		{
			name: "Relative path project",
			projects: map[string]settings.Project{
				"relative": {
					Path:        "valid-project",
					Description: "A project with relative path",
				},
			},
			homeDir: tempHomeDir,
			expectedOutput: []string{
				"PROJECTS:",
				"relative:",
				"valid-project",
				"[Valid: ✓]",
				"[In $HOME: ✓]",
				"Description: A project with relative path",
			},
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test settings
			cfg := &settings.Settings{
				Projects: tt.projects,
			}

			// Create a mock home directory function
			mockHomeDir := func() (string, error) {
				return tt.homeDir, nil
			}

			// Save stdout to restore it later
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Call function being tested with mock home dir
			ListWithCustomHomeDir(cfg, mockHomeDir)

			// Close writer to get output
			w.Close()
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			// Restore stdout
			os.Stdout = oldStdout

			// Check expected outputs
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, got %q", expected, output)
				}
			}

			// Check not expected outputs
			for _, notExpected := range tt.notExpected {
				if strings.Contains(output, notExpected) {
					t.Errorf("Expected output to NOT contain %q, got %q", notExpected, output)
				}
			}
		})
	}
}
