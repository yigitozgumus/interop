package factory

import (
	"fmt"
	"interop/internal/errors"
	"interop/internal/execution"
	"interop/internal/logging"
	"interop/internal/settings"
	"interop/internal/shell"
	"os"
	"path/filepath"
	"strings"
)

// CommandType identifies the type of command to create
type CommandType string

const (
	// ShellCommand represents a standard shell command
	ShellCommand CommandType = "shell"
	// ExecutableCommand represents a custom executable command
	ExecutableCommand CommandType = "executable"
)

// Factory creates command instances based on configuration
type Factory struct {
	Config     *settings.Settings
	Executor   *execution.Executor
	ShellInfo  *shell.Info
	SearchDirs []string
}

// NewFactory creates a new command factory
func NewFactory(config *settings.Settings, executor *execution.Executor, shellInfo *shell.Info) (*Factory, error) {
	// Get executable search paths
	searchDirs, err := settings.GetExecutableSearchPaths(config)
	if err != nil {
		return nil, errors.NewPathError("Failed to get executable search paths", err)
	}

	return &Factory{
		Config:     config,
		Executor:   executor,
		ShellInfo:  shellInfo,
		SearchDirs: searchDirs,
	}, nil
}

// Command represents a runnable command
type Command struct {
	Name        string
	Description string
	Path        string
	Args        []string
	Dir         string
	Type        CommandType
	Enabled     bool
}

// Create creates a command instance from a command configuration
func (f *Factory) Create(cmdName string, projectPath string) (*Command, error) {
	// Get command config
	cmdConfig, exists := f.Config.Commands[cmdName]
	if !exists {
		return nil, errors.NewCommandError(fmt.Sprintf("Command '%s' not found", cmdName), nil, true)
	}

	// Check if command is enabled
	if !cmdConfig.IsEnabled {
		return nil, errors.NewCommandError(fmt.Sprintf("Command '%s' is disabled", cmdName), nil, false)
	}

	// Create the appropriate command type
	if cmdConfig.IsExecutable {
		return f.createExecutableCommand(cmdName, cmdConfig, projectPath)
	}
	logging.Message("Creating shell command: %s", cmdName)

	return f.createShellCommand(cmdName, cmdConfig, projectPath)
}

// CreateFromAlias creates a command instance from an alias
func (f *Factory) CreateFromAlias(projectName string, alias string) (*Command, error) {
	// Find the project
	project, exists := f.Config.Projects[projectName]
	if !exists {
		return nil, errors.NewProjectError(fmt.Sprintf("Project '%s' not found", projectName), nil, true)
	}

	// Find the command alias
	var cmdName string
	found := false

	// Check if the alias exactly matches a command alias
	for _, cmd := range project.Commands {
		if cmd.Alias == alias {
			cmdName = cmd.CommandName
			found = true
			break
		}
	}

	// If not found as an alias, check if it directly matches a command name
	if !found {
		for _, cmd := range project.Commands {
			if cmd.CommandName == alias {
				cmdName = cmd.CommandName
				found = true
				break
			}
		}
	}

	if !found {
		return nil, errors.NewCommandError(
			fmt.Sprintf("Command or alias '%s' not found in project '%s'", alias, projectName),
			nil,
			true,
		)
	}

	// Resolve the project path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.NewPathError("Failed to get user home directory", err)
	}

	projectPath := project.Path
	if strings.HasPrefix(projectPath, "~/") {
		projectPath = filepath.Join(homeDir, projectPath[2:])
	} else if !filepath.IsAbs(projectPath) {
		projectPath = filepath.Join(homeDir, projectPath)
	}
	logging.Message("Project path: %s", projectPath)

	// Create the command
	return f.Create(cmdName, projectPath)
}

// createShellCommand creates a shell command from configuration
func (f *Factory) createShellCommand(name string, config settings.CommandConfig, workDir string) (*Command, error) {
	return &Command{
		Name:        name,
		Description: config.Description,
		Path:        f.ShellInfo.Path,
		Args:        []string{f.ShellInfo.Option, config.Cmd},
		Dir:         workDir,
		Type:        ShellCommand,
		Enabled:     config.IsEnabled,
	}, nil
}

// createExecutableCommand creates an executable command from configuration
func (f *Factory) createExecutableCommand(name string, config settings.CommandConfig, workDir string) (*Command, error) {
	// Split command and arguments
	cmdParts := strings.Fields(config.Cmd)
	if len(cmdParts) == 0 {
		return nil, errors.NewCommandError(
			"Empty command provided",
			nil,
			true,
		)
	}

	execName := cmdParts[0]
	cmdArgs := cmdParts[1:]

	// Find the executable in search paths
	var execPath string
	for _, dir := range f.SearchDirs {
		path := filepath.Join(dir, execName)
		logging.Message("Checking path: %s", path)
		if _, err := os.Stat(path); err == nil {
			execPath = path
			break
		}
	}
	logging.Message("Executable path: %s", execPath)

	if execPath == "" {
		return nil, errors.NewCommandError(
			fmt.Sprintf("Executable '%s' not found in search paths", execName),
			nil,
			true,
		)
	}

	return &Command{
		Name:        name,
		Description: config.Description,
		Path:        execPath,
		Args:        cmdArgs, // Use the parsed arguments
		Dir:         workDir,
		Type:        ExecutableCommand,
		Enabled:     config.IsEnabled,
	}, nil
}

// RunWithArgs executes the command with additional arguments
func (c *Command) RunWithArgs(args []string) error {
	logging.Message("Running command: %s with args: %v in directory: %s", c.Name, args, c.Dir)

	// Set up command execution
	cmd := &execution.Command{
		Path: c.Path,
		Args: c.Args,
		Dir:  c.Dir,
	}

	// Get the command configuration to check for prefixed arguments
	cfg, err := settings.Load()
	if err != nil {
		logging.Warning("Failed to load settings for prefixed arguments: %v", err)
		// Continue with normal argument handling
	} else {
		// Get the command config to check for prefixed arguments
		cmdConfig, exists := cfg.Commands[c.Name]
		if exists && len(cmdConfig.Arguments) > 0 && len(args) > 0 {
			// Parse args into a map
			argsMap := make(map[string]string)
			positionalIndex := 0

			// First, collect arguments that don't have prefixes (positional arguments)
			var positionalArgDefs []settings.CommandArgument
			for _, argDef := range cmdConfig.Arguments {
				if argDef.Prefix == "" {
					positionalArgDefs = append(positionalArgDefs, argDef)
				}
			}

			// Process arguments in order
			for _, arg := range args {
				if strings.Contains(arg, "=") {
					// Handle name=value pairs
					parts := strings.SplitN(arg, "=", 2)
					if len(parts) == 2 {
						argsMap[parts[0]] = parts[1]
					}
				} else {
					// Handle positional arguments (no = sign)
					if positionalIndex < len(positionalArgDefs) {
						argDef := positionalArgDefs[positionalIndex]
						argsMap[argDef.Name] = arg
						positionalIndex++
						logging.Message("Mapped positional argument '%s' to parameter '%s'", arg, argDef.Name)
					} else {
						// If we have more positional args than expected, treat as regular args
						logging.Message("Extra positional argument: %s", arg)
					}
				}
			}

			// If we have any arguments to process
			if len(argsMap) > 0 {
				// For shell commands, we'll construct a new command string with prefixes
				if c.Type == ShellCommand && len(cmd.Args) >= 2 {
					baseCmd := cmd.Args[1]
					var prefixedArgs []string
					var positionalArgs []string

					// Process each argument definition in order
					for _, argDef := range cmdConfig.Arguments {
						if value, ok := argsMap[argDef.Name]; ok {
							if argDef.Prefix != "" {
								// For arguments with prefixes
								if argDef.Type == settings.ArgumentTypeBool {
									if value == "true" {
										prefixedArgs = append(prefixedArgs, argDef.Prefix)
									}
								} else {
									// For other types, add both prefix and value
									prefixedArgs = append(prefixedArgs, fmt.Sprintf("%s %s", argDef.Prefix, value))
								}
							} else {
								// For arguments without prefixes (positional)
								positionalArgs = append(positionalArgs, value)
							}
							// Remove from argsMap to track which ones we've processed
							delete(argsMap, argDef.Name)
						}
					}

					// Append any remaining arguments (undefined arguments)
					var standardArgs []string
					for name, value := range argsMap {
						standardArgs = append(standardArgs, fmt.Sprintf("%s=%s", name, value))
					}

					// Combine the command parts: base command + positional args + prefixed args + remaining args
					newCmd := baseCmd
					if len(positionalArgs) > 0 {
						newCmd = fmt.Sprintf("%s %s", newCmd, strings.Join(positionalArgs, " "))
					}
					if len(prefixedArgs) > 0 {
						newCmd = fmt.Sprintf("%s %s", newCmd, strings.Join(prefixedArgs, " "))
					}
					if len(standardArgs) > 0 {
						newCmd = fmt.Sprintf("%s %s", newCmd, strings.Join(standardArgs, " "))
					}

					logging.Message("Command with prefixed args: %s", newCmd)
					cmd.Args[1] = newCmd

					// We've handled the arguments, so return early
					return execution.NewExecutor().Execute(cmd)
				}
			}
		}
	}

	// If we didn't handle prefixed arguments, fall back to standard behavior
	if c.Type == ExecutableCommand && args != nil && len(args) > 0 {
		// For executable commands, add arguments directly
		cmd.Args = append(cmd.Args, args...)
	} else if c.Type == ShellCommand && args != nil && len(args) > 0 {
		// For shell commands, the command is in Args[1]
		if len(cmd.Args) >= 2 {
			// Format the command with arguments
			commandWithArgs := fmt.Sprintf("%s %s", cmd.Args[1], strings.Join(args, " "))
			cmd.Args[1] = commandWithArgs
		}
	}

	// Run the command
	return execution.NewExecutor().Execute(cmd)
}
