package validation

import (
	"fmt"
	"interop/internal/command/factory"
	"interop/internal/errors"
	"interop/internal/execution"
	"interop/internal/logging"
	"interop/internal/settings"
	"interop/internal/shell"
	"interop/internal/validation/project"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CommandType represents the type of a command
type CommandType string

const (
	GlobalCommand  CommandType = "global"
	ProjectCommand CommandType = "project"
	AliasCommand   CommandType = "alias"
)

// CommandReference contains the resolved command and its context
type CommandReference struct {
	Type        CommandType
	Command     settings.CommandConfig
	ProjectName string // Empty for global commands
	Name        string // Original command name or alias
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Message string
	Severe  bool // If true, this error should prevent operation
}

// isFileExecutable checks if a file exists and has executable permissions
func isFileExecutable(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // File doesn't exist
		}
		return false, err // Other error
	}

	// Check if it's a file and not a directory
	if fileInfo.IsDir() {
		return false, nil
	}

	// Check if file has executable permission
	// 0100 is the executable bit for owner
	return fileInfo.Mode()&0100 != 0, nil
}

// ValidateCommands validates all commands in the settings
// Returns a list of validation errors
func ValidateCommands(cfg *settings.Settings) []ValidationError {
	// First validate projects using our new project validator
	projectValidator := project.NewValidator(cfg)
	projectResult := projectValidator.ValidateAll()

	// Convert project validation errors to the old format for backward compatibility
	errors := []ValidationError{}
	for _, err := range projectResult.Errors {
		errors = append(errors, ValidationError{
			Message: err.Error(),
			Severe:  err.Severe,
		})
	}

	// Track command usage to detect conflicts (maintaining existing functionality)
	usedCommands := make(map[string]string) // command name -> project name
	usedAliases := make(map[string]string)  // alias -> project name

	// Check for command uniqueness
	for projectName, projectData := range cfg.Projects {
		for _, aliasConfig := range projectData.Commands {
			// Check if command exists - this was already checked in project validator
			if _, exists := cfg.Commands[aliasConfig.CommandName]; !exists {
				continue // Skip, already reported
			}

			// Check if command is bound to multiple projects without alias
			if aliasConfig.Alias == "" {
				if prevProject, used := usedCommands[aliasConfig.CommandName]; used {
					errors = append(errors, ValidationError{
						Message: fmt.Sprintf("Command '%s' is bound to multiple projects ('%s' and '%s') without alias",
							aliasConfig.CommandName, prevProject, projectName),
						Severe: true,
					})
				}
				usedCommands[aliasConfig.CommandName] = projectName
			} else {
				// Check if alias is unique across projects
				if prevProject, used := usedAliases[aliasConfig.Alias]; used {
					errors = append(errors, ValidationError{
						Message: fmt.Sprintf("Alias '%s' is used in multiple projects ('%s' and '%s')",
							aliasConfig.Alias, prevProject, projectName),
						Severe: true,
					})
				}
				usedAliases[aliasConfig.Alias] = projectName
			}
		}
	}

	// Validate MCP server configurations
	usedPorts := make(map[int]string) // track port -> server name mapping

	// Add default MCP port to used ports
	if cfg.MCPPort > 0 {
		usedPorts[cfg.MCPPort] = "default MCP server"
	}

	// Check for MCP server port conflicts
	for name, server := range cfg.MCPServers {
		// Validate MCP server required fields
		if server.Name == "" {
			errors = append(errors, ValidationError{
				Message: fmt.Sprintf("MCP server '%s' must have a name", name),
				Severe:  true,
			})
		} else if server.Name != name {
			errors = append(errors, ValidationError{
				Message: fmt.Sprintf("MCP server name '%s' doesn't match key '%s'", server.Name, name),
				Severe:  true,
			})
		}

		if server.Port <= 0 {
			errors = append(errors, ValidationError{
				Message: fmt.Sprintf("MCP server '%s' has invalid port: %d", name, server.Port),
				Severe:  true,
			})
		} else {
			// Check for port conflicts
			if existingServer, exists := usedPorts[server.Port]; exists {
				errors = append(errors, ValidationError{
					Message: fmt.Sprintf("MCP server '%s' has port %d which conflicts with %s",
						name, server.Port, existingServer),
					Severe: true,
				})
			} else {
				usedPorts[server.Port] = fmt.Sprintf("MCP server '%s'", name)
			}
		}

		if server.Description == "" {
			errors = append(errors, ValidationError{
				Message: fmt.Sprintf("MCP server '%s' should have a description", name),
				Severe:  false,
			})
		}
	}

	// Validate command MCP references
	for cmdName, cmd := range cfg.Commands {
		if cmd.MCP != "" {
			if _, exists := cfg.MCPServers[cmd.MCP]; !exists {
				errors = append(errors, ValidationError{
					Message: fmt.Sprintf("Command '%s' references a non-existent MCP server '%s'",
						cmdName, cmd.MCP),
					Severe: true,
				})
			}
		}
	}

	// Check executable commands for proper permissions
	for _, cmd := range cfg.Commands {
		if cmd.IsExecutable {
			// Extract just the command name (first part before whitespace)
			cmdName := strings.Fields(cmd.Cmd)[0]

			// Try to find the executable in various paths
			// First try PATH environment
			execPath, err := exec.LookPath(cmdName)
			if err != nil {
				// If not found in PATH, check in common executable locations
				execDirs := []string{"/usr/bin", "/usr/local/bin", "/bin", "/opt/bin"}
				found := false

				for _, dir := range execDirs {
					potentialPath := filepath.Join(dir, cmdName)
					isExec, err := isFileExecutable(potentialPath)
					if err == nil && isExec {
						execPath = potentialPath
						found = true
						break
					}
				}

				if !found {
					errors = append(errors, ValidationError{
						Message: fmt.Sprintf("Executable command '%s' not found in PATH or common executable directories", cmdName),
						Severe:  false,
					})
					continue
				}
			}

			// Check if found file has executable permissions
			isExec, err := isFileExecutable(execPath)
			if err != nil {
				errors = append(errors, ValidationError{
					Message: fmt.Sprintf("Error checking executable permissions for '%s': %v", cmdName, err),
					Severe:  false,
				})
			} else if !isExec {
				errors = append(errors, ValidationError{
					Message: fmt.Sprintf("Command '%s' is marked as executable but doesn't have executable permissions. Use 'chmod +x %s' to fix.", cmdName, execPath),
					Severe:  false,
				})
			}
		}
	}

	return errors
}

// ResolveCommand finds a command by name or alias
// Returns the command reference and a potential error
func ResolveCommand(cfg *settings.Settings, nameOrAlias string) (*CommandReference, error) {
	// Check if command exists in global commands
	cmd, cmdExists := cfg.Commands[nameOrAlias]
	if !cmdExists {
		// If command doesn't exist at all, check aliases and return error if not found
		for projectName, projectData := range cfg.Projects {
			for _, alias := range projectData.Commands {
				// Check if it matches an alias
				if alias.Alias == nameOrAlias {
					if cmd, ok := cfg.Commands[alias.CommandName]; ok {
						return &CommandReference{
							Type:        AliasCommand,
							Command:     cmd,
							ProjectName: projectName,
							Name:        nameOrAlias,
						}, nil
					}
				}
			}
		}
		return nil, errors.NewCommandError(fmt.Sprintf("Command or alias '%s' not found", nameOrAlias), nil, true)
	}

	// Check if command is bound to any project with its original name (no alias)
	for projectName, projectData := range cfg.Projects {
		for _, alias := range projectData.Commands {
			if alias.CommandName == nameOrAlias && alias.Alias == "" {
				// Found the command in a project with its original name, so it's a project command
				return &CommandReference{
					Type:        ProjectCommand,
					Command:     cmd,
					ProjectName: projectName,
					Name:        nameOrAlias,
				}, nil
			}
		}
	}

	// If command exists in global commands and wasn't found in any project with original name,
	// it's a global command
	return &CommandReference{
		Type:    GlobalCommand,
		Command: cmd,
		Name:    nameOrAlias,
	}, nil
}

// ExecuteCommand validates the configuration, resolves and executes a command by name or alias
func ExecuteCommand(cfg *settings.Settings, nameOrAlias string) error {
	return ExecuteCommandWithArgs(cfg, nameOrAlias, nil)
}

// ExecuteCommandWithArgs validates the configuration, resolves and executes a command by name or alias with arguments
func ExecuteCommandWithArgs(cfg *settings.Settings, nameOrAlias string, args []string) error {
	// First validate all commands
	validationErrors := ValidateCommands(cfg)
	for _, err := range validationErrors {
		if err.Severe {
			return errors.NewValidationError(fmt.Sprintf("Configuration error: %s", err.Message), nil, true)
		}
	}

	// Resolve the command using existing resolver to maintain compatibility
	cmdRef, err := ResolveCommand(cfg, nameOrAlias)
	if err != nil {
		return err
	}
	logging.Message("Command reference: %v", cmdRef)

	// Get shell info
	shellInfo, err := shell.DetectShell()
	if err != nil {
		return errors.NewExecutionError("Failed to detect shell", err)
	}

	// Create a command factory
	executor := execution.NewExecutor()
	commandFactory, err := factory.NewFactory(cfg, executor, shellInfo)
	if err != nil {
		return errors.NewExecutionError("Failed to create command factory", err)
	}

	// If it's a project command or alias, we need to create it with the project path
	var cmd *factory.Command
	logging.Message("Project name: %s", cmdRef.ProjectName)
	if cmdRef.ProjectName != "" {
		// For project commands, use CreateFromAlias
		cmd, err = commandFactory.CreateFromAlias(cmdRef.ProjectName, nameOrAlias)
	} else {
		// For global commands, use Create with empty project path
		cmd, err = commandFactory.Create(nameOrAlias, "")
	}

	if err != nil {
		return err
	}

	// Execute the command with arguments
	return cmd.RunWithArgs(args)
}
