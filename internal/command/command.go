package command

import (
	"fmt"
	"interop/internal/display"
	"interop/internal/execution"
)

// Command defines a command that can be executed
type Command struct {
	Description  string `toml:"description,omitempty"`
	IsEnabled    bool   `toml:"is_enabled"`
	Cmd          string `toml:"cmd"`
	IsExecutable bool   `toml:"is_executable"`
}

// Alias represents a command alias in a project
type Alias struct {
	CommandName string
	Alias       string
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
func PrintCommandDetails(name string, cmd Command, projectAssociations map[string][]string) {
	// Print command details using display package
	display.PrintCommandName(name)

	execSource := "Script"
	if cmd.IsExecutable {
		execSource = "Executables"
	}

	display.PrintCommandStatus(cmd.IsEnabled, execSource)

	// Print source information
	display.PrintCommandSource(name)

	// Print project associations if any
	projectNames, hasProjects := projectAssociations[name]
	if hasProjects && len(projectNames) > 0 {
		display.PrintCommandProjects(projectNames)
	}

	// Print description if exists
	display.PrintCommandDescription(cmd.Description)

	// Add separator
	display.PrintSeparator()
}

// List prints out all configured commands with their properties
func List(commands map[string]Command) {
	if len(commands) == 0 {
		display.PrintNoItemsFound("commands")
		return
	}

	display.PrintCommandHeader()

	for name, cmd := range commands {
		PrintCommandDetails(name, cmd, nil)
	}
}

// ListWithProjects prints all commands with their project associations
func ListWithProjects(commands map[string]Command, projectCommands map[string][]Alias) {
	if len(commands) == 0 {
		display.PrintNoItemsFound("commands")
		return
	}

	display.PrintCommandHeader()

	// Build a map of command name to associated projects
	commandProjects := make(map[string][]string)

	// For each project
	for projectName, aliases := range projectCommands {
		// For each command in project
		for _, aliasConfig := range aliases {
			// Add to the map
			commandProjects[aliasConfig.CommandName] = append(
				commandProjects[aliasConfig.CommandName],
				projectName+(func() string {
					if aliasConfig.Alias != "" {
						return " (alias: " + aliasConfig.Alias + ")"
					}
					return ""
				})())
		}
	}

	for name, cmd := range commands {
		PrintCommandDetails(name, cmd, commandProjects)
	}
}

// RunWithSearchPathsAndArgs executes a command by name with arguments, searching for executables in multiple paths
func RunWithSearchPathsAndArgs(commands map[string]Command, commandName string, executableSearchPaths []string, args []string, projectPath ...string) error {
	cmd, exists := commands[commandName]
	if !exists {
		return fmt.Errorf("command '%s' not found", commandName)
	}

	// Create execution.CommandInfo from Command
	execInfo := execution.CommandInfo{
		Name:         commandName,
		Description:  cmd.Description,
		IsEnabled:    cmd.IsEnabled,
		Cmd:          cmd.Cmd,
		IsExecutable: cmd.IsExecutable,
	}

	// Use the execution package to run the command
	return execution.RunWithSearchPathsAndArgs(execInfo, executableSearchPaths, args, projectPath...)
}

// Helper function to get a boolean value with a default
func getBoolWithDefault(m map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return defaultValue
}
