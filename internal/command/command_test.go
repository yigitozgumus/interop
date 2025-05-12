package command

import (
	"testing"
)

func TestNewCommand(t *testing.T) {
	cmd := NewCommand()

	// Test default values
	if !cmd.IsEnabled {
		t.Error("Expected IsEnabled to be true by default")
	}

	if cmd.IsExecutable {
		t.Error("Expected IsExecutable to be false by default")
	}

	if cmd.Description != "" {
		t.Error("Expected Description to be empty by default")
	}
}

func TestPrintCommandDetails(t *testing.T) {
	// This is primarily a visual test, but we ensure it doesn't panic
	cmd := Command{
		Description:  "Test command",
		IsEnabled:    true,
		Cmd:          "echo 'hello world'",
		IsExecutable: true,
	}

	// This should not panic
	PrintCommandDetails("test-command", cmd, nil)
}

func TestList(t *testing.T) {
	// Create test commands
	cmds := map[string]Command{
		"test1": {
			Description:  "Test command 1",
			IsEnabled:    true,
			Cmd:          "echo 'test1'",
			IsExecutable: false,
		},
		"test2": {
			Description:  "",
			IsEnabled:    false,
			Cmd:          "echo 'test2'",
			IsExecutable: true,
		},
	}

	// This should not panic
	List(cmds)

	// Test with empty commands
	List(map[string]Command{})
}

// TestUnmarshalTOMLWithString tests the case when command is defined as just a string
func TestUnmarshalTOMLWithString(t *testing.T) {
	var cmd Command
	err := cmd.UnmarshalTOML("echo hello")

	if err != nil {
		t.Fatalf("UnmarshalTOML failed: %v", err)
	}

	if cmd.Cmd != "echo hello" {
		t.Errorf("Expected cmd to be 'echo hello', got '%s'", cmd.Cmd)
	}

	// Check defaults
	if !cmd.IsEnabled {
		t.Error("Expected IsEnabled to default to true")
	}

	if cmd.IsExecutable {
		t.Error("Expected IsExecutable to default to false")
	}

	if cmd.Description != "" {
		t.Error("Expected Description to default to empty string")
	}
}

// TestUnmarshalTOMLWithMap tests the case when command is defined as a map with various fields
func TestUnmarshalTOMLWithMap(t *testing.T) {
	testCases := []struct {
		name     string
		input    map[string]interface{}
		expected Command
	}{
		{
			name:  "only cmd",
			input: map[string]interface{}{"cmd": "echo minimal"},
			expected: Command{
				Cmd:          "echo minimal",
				IsEnabled:    true,
				IsExecutable: false,
				Description:  "",
			},
		},
		{
			name:  "with description",
			input: map[string]interface{}{"cmd": "echo with desc", "description": "test desc"},
			expected: Command{
				Cmd:          "echo with desc",
				Description:  "test desc",
				IsEnabled:    true,
				IsExecutable: false,
			},
		},
		{
			name:  "with booleans",
			input: map[string]interface{}{"cmd": "echo flags", "is_enabled": false, "is_executable": true},
			expected: Command{
				Cmd:          "echo flags",
				IsEnabled:    false,
				IsExecutable: true,
				Description:  "",
			},
		},
		{
			name: "with everything",
			input: map[string]interface{}{
				"cmd":           "echo all",
				"description":   "full config",
				"is_enabled":    false,
				"is_executable": true,
			},
			expected: Command{
				Cmd:          "echo all",
				Description:  "full config",
				IsEnabled:    false,
				IsExecutable: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var cmd Command
			if err := cmd.UnmarshalTOML(tc.input); err != nil {
				t.Fatalf("UnmarshalTOML failed: %v", err)
			}

			if cmd.Cmd != tc.expected.Cmd {
				t.Errorf("Expected Cmd to be '%s', got '%s'", tc.expected.Cmd, cmd.Cmd)
			}

			if cmd.Description != tc.expected.Description {
				t.Errorf("Expected Description to be '%s', got '%s'", tc.expected.Description, cmd.Description)
			}

			if cmd.IsEnabled != tc.expected.IsEnabled {
				t.Errorf("Expected IsEnabled to be %v, got %v", tc.expected.IsEnabled, cmd.IsEnabled)
			}

			if cmd.IsExecutable != tc.expected.IsExecutable {
				t.Errorf("Expected IsExecutable to be %v, got %v", tc.expected.IsExecutable, cmd.IsExecutable)
			}
		})
	}
}
