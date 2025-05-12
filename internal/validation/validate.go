package validation

import (
	"fmt"
	"interop/internal/command/factory"
	"interop/internal/errors"
	"interop/internal/execution"
	"interop/internal/settings"
	"interop/internal/shell"
	"interop/internal/validation/project"
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

	return errors
}

// ResolveCommand finds a command by name or alias
// Returns the command reference and a potential error
func ResolveCommand(cfg *settings.Settings, nameOrAlias string) (*CommandReference, error) {
	// First check if it's a global command
	if cmd, ok := cfg.Commands[nameOrAlias]; ok {
		return &CommandReference{
			Type:    GlobalCommand,
			Command: cmd,
			Name:    nameOrAlias,
		}, nil
	}

	// Then check if it's a project command or alias
	for projectName, projectData := range cfg.Projects {
		for _, alias := range projectData.Commands {
			// Check if it matches the command name
			if alias.CommandName == nameOrAlias && alias.Alias == "" {
				if cmd, ok := cfg.Commands[alias.CommandName]; ok {
					return &CommandReference{
						Type:        ProjectCommand,
						Command:     cmd,
						ProjectName: projectName,
						Name:        nameOrAlias,
					}, nil
				}
			}

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

// ExecuteCommand validates the configuration, resolves and executes a command by name or alias
func ExecuteCommand(cfg *settings.Settings, nameOrAlias string) error {
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

	// Execute the command
	return cmd.Run()
}
