package command

import (
	"fmt"
	"interop/internal/util"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	// Print project associations if any
	projectNames, hasProjects := projectAssociations[name]
	if hasProjects && len(projectNames) > 0 {
		if len(projectNames) == 1 {
			fmt.Printf("   Project: %s\n", projectNames[0])
		} else {
			fmt.Printf("   Projects: %s\n", strings.Join(projectNames, ", "))
		}
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
		PrintCommandDetails(name, cmd, nil)
	}
}

// ListWithProjects prints all commands with their project associations
func ListWithProjects(commands map[string]Command, projectCommands map[string][]Alias) {
	if len(commands) == 0 {
		fmt.Println("No commands found.")
		return
	}

	fmt.Println("COMMANDS:")
	fmt.Println("=========")
	fmt.Println()

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

// Run executes a command by name
func Run(commands map[string]Command, commandName string, executablesPath string, projectPath ...string) error {
	return RunWithSearchPaths(commands, commandName, []string{executablesPath}, projectPath...)
}

// RunWithSearchPaths executes a command by name, searching for executables in multiple paths
func RunWithSearchPaths(commands map[string]Command, commandName string, executableSearchPaths []string, projectPath ...string) error {
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

		projectDir := projectPath[0]
		// If path doesn't exist, try to report a more helpful error
		if _, err := os.Stat(projectDir); os.IsNotExist(err) {
			return fmt.Errorf("project directory doesn't exist: %s", projectDir)
		}

		// Change to project directory
		util.Message("Changing to project directory: %s", projectDir)
		if err := os.Chdir(projectDir); err != nil {
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

	// Get user's shell
	userShell := os.Getenv("SHELL")
	if userShell == "" {
		// Fallback to sh if SHELL is not defined
		userShell = "/bin/sh"
		util.Warning("SHELL environment variable not set, defaulting to /bin/sh")
	}

	// Get the shell name (basename)
	shellName := filepath.Base(userShell)

	// Check if this command should run as a shell alias
	if strings.HasPrefix(cmd.Cmd, "alias:") {
		// Extract the alias name
		aliasCmd := strings.TrimSpace(strings.TrimPrefix(cmd.Cmd, "alias:"))
		util.Message("Running shell alias: %s", aliasCmd)

		// Run in interactive shell to ensure aliases are loaded
		switch shellName {
		case "bash":
			command = exec.Command(userShell, "-ic", aliasCmd)
		case "zsh":
			command = exec.Command(userShell, "-ic", aliasCmd)
		case "fish":
			command = exec.Command(userShell, "-ic", aliasCmd)
		default:
			command = exec.Command(userShell, "-ic", aliasCmd)
		}
	} else if cmd.IsExecutable {
		// For executable commands, look for the executable in all search paths
		execFound := false
		var execPath string
		var execErr error

		for _, searchPath := range executableSearchPaths {
			candidatePath := filepath.Join(searchPath, cmd.Cmd)
			if _, err := os.Stat(candidatePath); err == nil {
				// Found the executable
				execPath = candidatePath
				execFound = true

				// Make sure the file is executable
				if err := os.Chmod(execPath, 0755); err != nil {
					execErr = fmt.Errorf("failed to set executable permissions: %w", err)
					continue
				}

				util.Message("Found executable '%s' in %s, executing", cmd.Cmd, searchPath)
				command = exec.Command(execPath)
				break
			}
		}

		// Check the PATH environment if not found in search paths
		if !execFound {
			// Try to find the executable in the system PATH
			if lookPath, err := exec.LookPath(cmd.Cmd); err == nil {
				execPath = lookPath
				execFound = true

				util.Message("Found executable '%s' in system PATH: %s", cmd.Cmd, execPath)
				command = exec.Command(execPath)
			}
		}

		if !execFound {
			if execErr != nil {
				return execErr
			}
			return fmt.Errorf("executable '%s' not found in any of the search paths", cmd.Cmd)
		}
	} else if strings.HasPrefix(cmd.Cmd, "./") {
		// Special handling for shell scripts that start with ./
		// Split the command into the script path and arguments
		parts := strings.Fields(cmd.Cmd)
		if len(parts) == 0 {
			return fmt.Errorf("invalid command: empty command")
		}

		scriptPath := parts[0][2:] // Remove the ./ prefix

		// Check if the script exists
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			return fmt.Errorf("script '%s' not found in current directory", scriptPath)
		}

		// Make sure the file is executable
		if err := os.Chmod(scriptPath, 0755); err != nil {
			util.Warning("Failed to set executable permissions on script '%s': %v", scriptPath, err)
		}

		// Create the command with arguments
		if len(parts) > 1 {
			util.Message("Executing script: %s with arguments: %v", scriptPath, parts[1:])
			command = exec.Command("./"+scriptPath, parts[1:]...)
		} else {
			util.Message("Executing script: %s", scriptPath)
			command = exec.Command("./" + scriptPath)
		}
	} else {
		// For regular commands, execute with the shell
		util.Message("Executing command with shell: %s", shellName)
		command = exec.Command(userShell, "-c", cmd.Cmd)
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
