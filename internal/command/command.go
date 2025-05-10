package command

import (
	"fmt"
	"interop/internal/util"
	"os"
	"os/exec"
)

// Command defines a command that can be executed
type Command struct {
	Description  string   `toml:"description,omitempty"`
	IsEnabled    bool     `toml:"is_enabled"`
	Projects     []string `toml:"projects"`
	Cmd          string   `toml:"cmd"`
	IsExecutable bool     `toml:"is_executable"`
}

// NewCommand creates a new Command with default values
func NewCommand() Command {
	return Command{
		IsEnabled:    true,
		Projects:     []string{},
		IsExecutable: false,
	}
}

// UnmarshalTOML supports partial command definitions in the TOML settings file
// This allows having just the cmd field defined with other fields getting defaults
func (c *Command) UnmarshalTOML(data interface{}) error {
	// Set defaults first
	c.IsEnabled = true
	c.Projects = []string{}
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
		c.Projects = getStringSliceWithDefault(v, "projects", []string{})
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

	// Print associated projects if any
	if len(cmd.Projects) > 0 {
		fmt.Printf("   Projects: %v\n", cmd.Projects)
	}

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
func Run(commands map[string]Command, commandName string) error {
	cmd, exists := commands[commandName]
	if !exists {
		util.Error("command '%s' not found", commandName)
	}

	if !cmd.IsEnabled {
		util.Error("command '%s' is not enabled", commandName)
	}

	util.Message("Command '%s' is enabled, proceeding with execution", commandName)

	if len(cmd.Projects) == 0 {
		util.Message("Command '%s' is not associated with any projects", commandName)
	}

	// Execute the command
	command := exec.Command("sh", "-c", cmd.Cmd)
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

// Helper function to get a string slice with a default
func getStringSliceWithDefault(m map[string]interface{}, key string, defaultValue []string) []string {
	if val, ok := m[key]; ok {
		// Handle both array and single string cases
		switch v := val.(type) {
		case []interface{}:
			result := make([]string, len(v))
			for i, item := range v {
				if s, ok := item.(string); ok {
					result[i] = s
				}
			}
			return result
		}
	}
	return defaultValue
}
