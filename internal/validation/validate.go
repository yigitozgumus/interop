package validation

import (
	"fmt"
	"interop/internal/command"
	"interop/internal/settings"
	"os"
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

// ValidateCommands validates all commands in the settings
// Returns a list of validation errors
func ValidateCommands(cfg *settings.Settings) []ValidationError {
	errors := []ValidationError{}

	// Track command usage to detect conflicts
	usedCommands := make(map[string]string) // command name -> project name
	usedAliases := make(map[string]string)  // alias -> project name

	// First check for command uniqueness
	for projectName, project := range cfg.Projects {
		for _, aliasConfig := range project.Commands {
			// Check if command exists
			if _, exists := cfg.Commands[aliasConfig.CommandName]; !exists {
				errors = append(errors, ValidationError{
					Message: fmt.Sprintf("Project '%s' references non-existent command '%s'",
						projectName, aliasConfig.CommandName),
					Severe: true,
				})
				continue
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

	return errors
}

// ResolveCommand finds a command by name or alias
// Returns the command reference and a potential error
func ResolveCommand(cfg *settings.Settings, nameOrAlias string) (*CommandReference, error) {
	// First, check if it's a direct command name
	if cmd, exists := cfg.Commands[nameOrAlias]; exists {
		// Is this command project-specific?
		isProjectSpecific := false
		projectName := ""

		// Check if command is bound to a project
		for pName, project := range cfg.Projects {
			for _, aliasConfig := range project.Commands {
				if aliasConfig.CommandName == nameOrAlias && aliasConfig.Alias == "" {
					isProjectSpecific = true
					projectName = pName
					break
				}
			}
			if isProjectSpecific {
				break
			}
		}

		if isProjectSpecific {
			return &CommandReference{
				Type:        ProjectCommand,
				Command:     cmd,
				ProjectName: projectName,
				Name:        nameOrAlias,
			}, nil
		}

		// It's a global command
		return &CommandReference{
			Type:    GlobalCommand,
			Command: cmd,
			Name:    nameOrAlias,
		}, nil
	}

	// Not a direct command, check if it's an alias
	for projectName, project := range cfg.Projects {
		for _, aliasConfig := range project.Commands {
			if aliasConfig.Alias == nameOrAlias {
				// Found the alias
				cmd, exists := cfg.Commands[aliasConfig.CommandName]
				if !exists {
					return nil, fmt.Errorf("alias '%s' references non-existent command '%s'",
						nameOrAlias, aliasConfig.CommandName)
				}

				return &CommandReference{
					Type:        AliasCommand,
					Command:     cmd,
					ProjectName: projectName,
					Name:        nameOrAlias,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("command or alias '%s' not found", nameOrAlias)
}

// resolveProjectPath handles tilde expansion and resolves relative paths
// to absolute paths based on the user's home directory
func resolveProjectPath(path string) (string, error) {
	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Handle tilde expansion for home directory
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:]), nil
	}

	// If not absolute, treat as relative to home
	if !filepath.IsAbs(path) {
		return filepath.Join(homeDir, path), nil
	}

	// Already absolute
	return path, nil
}

// ExecuteCommand runs a command by name or alias
func ExecuteCommand(cfg *settings.Settings, nameOrAlias string) error {
	// First, validate all commands
	validationErrors := ValidateCommands(cfg)
	for _, err := range validationErrors {
		if err.Severe {
			return fmt.Errorf("configuration error: %s", err.Message)
		}
	}

	// Resolve the command
	cmdRef, err := ResolveCommand(cfg, nameOrAlias)
	if err != nil {
		return err
	}

	// Get all executable search paths
	searchPaths, err := settings.GetExecutableSearchPaths(cfg)
	if err != nil {
		return fmt.Errorf("failed to get executable search paths: %w", err)
	}

	// If it's a project command or alias, we need the project path
	projectPath := ""
	if cmdRef.ProjectName != "" {
		project, exists := cfg.Projects[cmdRef.ProjectName]
		if !exists {
			return fmt.Errorf("project '%s' not found", cmdRef.ProjectName)
		}

		// Resolve the project path (handle tilde and relative paths)
		resolvedPath, err := resolveProjectPath(project.Path)
		if err != nil {
			return err
		}

		projectPath = resolvedPath
	}

	// Convert the settings.CommandConfig to command.Command
	cmdToRun := command.Command{
		Description:  cmdRef.Command.Description,
		IsEnabled:    cmdRef.Command.IsEnabled,
		Cmd:          cmdRef.Command.Cmd,
		IsExecutable: cmdRef.Command.IsExecutable,
	}

	// Create a new map with just the command we want to run
	commandToRun := map[string]command.Command{
		cmdRef.Name: cmdToRun,
	}

	// Run the command with all search paths
	if projectPath != "" {
		return command.RunWithSearchPaths(commandToRun, cmdRef.Name, searchPaths, projectPath)
	}

	return command.RunWithSearchPaths(commandToRun, cmdRef.Name, searchPaths)
}
