package command

import (
	"fmt"
	"interop/internal/util"
	"os"
	"os/exec"
	"path/filepath"
)

// Command defines a command that can be executed
type Command struct {
	Description  string `toml:"description,omitempty"`
	IsEnabled    bool   `toml:"is_enabled"`
	Cmd          string `toml:"cmd"`
	IsExecutable bool   `toml:"is_executable"`
}

// NewCommand creates a new Command with default values
func NewCommand() Command {
	return Command{
		IsEnabled:    true,
		IsExecutable: false,
	}
}

// UnmarshalTOML supports partial command definitions in the TOML settings file
// This allows having just the cmd field defined with other fields getting defaults
func (c *Command) UnmarshalTOML(data interface{}) error {
	// Set defaults first
	c.IsEnabled = true
	c.IsExecutable = false
	c.Description = ""

	// Handle different input cases
	switch v := data.(type) {
	case string:
		// If the command is specified as just a string, use it as cmd
		c.Cmd = v
	case map[string]interface{}:
		// If a field is present, use its value
		if cmd, ok := v["cmd"].(string); ok {
			c.Cmd = cmd
		}
		if desc, ok := v["description"].(string); ok {
			c.Description = desc
		}
		c.IsEnabled = getBoolWithDefault(v, "is_enabled", true)
		c.IsExecutable = getBoolWithDefault(v, "is_executable", false)
	}
	return nil
}

// PrintCommandDetails prints detailed information about a single command
func PrintCommandDetails(name string, cmd Command) {
	// Print command name
	fmt.Printf("⚡ Name: %s\n", name)

	// Print status indicators
	statusEnabled := "✓"
	if !cmd.IsEnabled {
		statusEnabled = "✗"
	}

	execSource := "Script"
	if cmd.IsExecutable {
		execSource = "Executables"
	}

	fmt.Printf("   Status: Enabled: %s  |  Source: %s\n", statusEnabled, execSource)

	// Print description if exists
	if cmd.Description != "" {
		fmt.Printf("   Description: %s\n", cmd.Description)
	}

	// Add separator
	fmt.Println()
}

// List prints out all configured commands with their properties
func List(commands map[string]Command) {
	if len(commands) == 0 {
		fmt.Println("No commands found.")
		return
	}

	fmt.Println("COMMANDS:")
	fmt.Println("=========")
	fmt.Println()

	for name, cmd := range commands {
		PrintCommandDetails(name, cmd)
	}
}

// Run executes a command by name
func Run(commands map[string]Command, commandName string, executablesPath string, projectPath ...string) error {
	cmd, exists := commands[commandName]
	if !exists {
		return fmt.Errorf("command '%s' not found", commandName)
	}

	if !cmd.IsEnabled {
		return fmt.Errorf("command '%s' is not enabled", commandName)
	}

	util.Message("Command '%s' is enabled, proceeding with execution", commandName)

	// Store current working directory if we need to change to project directory
	var currentDir string
	var err error

	// If project path is provided, change to that directory before running the command
	if len(projectPath) > 0 && projectPath[0] != "" {
		// Save current directory to return to after command execution
		currentDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}

		// Change to project directory
		util.Message("Changing to project directory: %s", projectPath[0])
		if err := os.Chdir(projectPath[0]); err != nil {
			return fmt.Errorf("failed to change to project directory: %w", err)
		}

		// Ensure we change back to original directory when done
		defer func() {
			util.Message("Changing back to original directory: %s", currentDir)
			if err := os.Chdir(currentDir); err != nil {
				util.Error("Failed to change back to original directory: %v", err)
			}
		}()
	}

	var command *exec.Cmd

	if cmd.IsExecutable {
		// For executable commands, look for the executable in the executables directory
		execPath := filepath.Join(executablesPath, cmd.Cmd)
		if _, err := os.Stat(execPath); os.IsNotExist(err) {
			return fmt.Errorf("executable '%s' not found in executables directory", cmd.Cmd)
		}

		// Make sure the file is executable
		if err := os.Chmod(execPath, 0755); err != nil {
			return fmt.Errorf("failed to set executable permissions: %w", err)
		}

		util.Message("Found executable '%s', executing", cmd.Cmd)
		command = exec.Command(execPath)
	} else {
		// For regular commands, execute them with the shell
		command = exec.Command("sh", "-c", cmd.Cmd)
	}

	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin

	return command.Run()
}

// Helper function to get a boolean value with a default
func getBoolWithDefault(m map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return defaultValue
}
