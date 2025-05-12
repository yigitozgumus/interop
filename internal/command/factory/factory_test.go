package factory

import (
	"interop/internal/execution"
	"interop/internal/settings"
	"interop/internal/shell"
	"os"
	"path/filepath"
	"testing"
)

func TestFactory_Create(t *testing.T) {
	// Create test shell info
	shellInfo := &shell.Info{
		Path:   "/bin/sh",
		Option: "-c",
		Name:   "sh",
	}

	// Create test settings
	testSettings := &settings.Settings{
		Commands: map[string]settings.CommandConfig{
			"enabled-shell-cmd": {
				Description:  "Enabled shell command",
				IsEnabled:    true,
				Cmd:          "echo 'shell command'",
				IsExecutable: false,
			},
			"disabled-cmd": {
				Description:  "Disabled command",
				IsEnabled:    false,
				Cmd:          "echo 'disabled'",
				IsExecutable: false,
			},
			"executable-cmd": {
				Description:  "Executable command",
				IsEnabled:    true,
				Cmd:          "test-executable",
				IsExecutable: true,
			},
		},
		ExecutableSearchPaths: []string{},
	}

	// Create executor
	executor := execution.NewExecutor()

	// Create factory
	factory, err := NewFactory(testSettings, executor, shellInfo)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	// Test creating an enabled shell command
	cmd, err := factory.Create("enabled-shell-cmd", "/test/dir")
	if err != nil {
		t.Errorf("Expected to create enabled shell command but got error: %v", err)
	} else {
		if cmd.Type != ShellCommand {
			t.Errorf("Expected ShellCommand type but got %v", cmd.Type)
		}
		if cmd.Path != "/bin/sh" {
			t.Errorf("Expected shell path /bin/sh but got %v", cmd.Path)
		}
		if len(cmd.Args) != 2 || cmd.Args[0] != "-c" || cmd.Args[1] != "echo 'shell command'" {
			t.Errorf("Unexpected args: %v", cmd.Args)
		}
	}

	// Test creating a disabled command
	_, err = factory.Create("disabled-cmd", "/test/dir")
	if err == nil {
		t.Errorf("Expected error when creating disabled command but got none")
	}

	// Test creating a non-existent command
	_, err = factory.Create("non-existent-cmd", "/test/dir")
	if err == nil {
		t.Errorf("Expected error when creating non-existent command but got none")
	}
}

func TestFactory_CreateFromAlias(t *testing.T) {
	// Create a temporary directory for testing
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home directory: %v", err)
	}

	// Create a test project directory
	testProjectDir := filepath.Join(homeDir, "test-project-factory")
	defer os.RemoveAll(testProjectDir)
	if err := os.MkdirAll(testProjectDir, 0755); err != nil {
		t.Fatalf("Failed to create test project directory: %v", err)
	}

	// Create test shell info
	shellInfo := &shell.Info{
		Path:   "/bin/sh",
		Option: "-c",
		Name:   "sh",
	}

	// Create test settings
	testSettings := &settings.Settings{
		Projects: map[string]settings.Project{
			"test-project": {
				Path:        testProjectDir,
				Description: "Test project",
				Commands: []settings.Alias{
					{CommandName: "test-cmd", Alias: "tc"},
					{CommandName: "no-alias-cmd", Alias: ""},
				},
			},
		},
		Commands: map[string]settings.CommandConfig{
			"test-cmd": {
				Description:  "Test command",
				IsEnabled:    true,
				Cmd:          "echo 'test'",
				IsExecutable: false,
			},
			"no-alias-cmd": {
				Description:  "Command without alias",
				IsEnabled:    true,
				Cmd:          "echo 'no alias'",
				IsExecutable: false,
			},
		},
	}

	// Create executor
	executor := execution.NewExecutor()

	// Create factory
	factory, err := NewFactory(testSettings, executor, shellInfo)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	// Test creating a command from alias
	cmd, err := factory.CreateFromAlias("test-project", "tc")
	if err != nil {
		t.Errorf("Expected to create command from alias but got error: %v", err)
	} else {
		if cmd.Name != "test-cmd" {
			t.Errorf("Expected command name 'test-cmd' but got %v", cmd.Name)
		}
	}

	// Test creating a command using its command name directly
	cmd, err = factory.CreateFromAlias("test-project", "no-alias-cmd")
	if err != nil {
		t.Errorf("Expected to create command using command name but got error: %v", err)
	} else {
		if cmd.Name != "no-alias-cmd" {
			t.Errorf("Expected command name 'no-alias-cmd' but got %v", cmd.Name)
		}
	}

	// Test creating a command with non-existent alias
	_, err = factory.CreateFromAlias("test-project", "non-existent")
	if err == nil {
		t.Errorf("Expected error when creating command with non-existent alias but got none")
	}

	// Test creating a command for non-existent project
	_, err = factory.CreateFromAlias("non-existent-project", "tc")
	if err == nil {
		t.Errorf("Expected error when creating command for non-existent project but got none")
	}
}
