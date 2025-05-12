package project

import (
	"fmt"
	"interop/internal/errors"
	"interop/internal/logging"
	"interop/internal/settings"
	"os"
	"path/filepath"
	"strings"
)

// Validator handles project validation operations
type Validator struct {
	settings *settings.Settings
}

// NewValidator creates a new project validator
func NewValidator(settings *settings.Settings) *Validator {
	return &Validator{
		settings: settings,
	}
}

// ValidationResult contains the result of a validation operation
type ValidationResult struct {
	Errors []errors.AppError
	Valid  bool
}

// ValidateAll checks all projects in the settings
func (v *Validator) ValidateAll() ValidationResult {
	var validationErrors []errors.AppError

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ValidationResult{
			Errors: []errors.AppError{*errors.NewPathError("Failed to get user home directory", err)},
			Valid:  false,
		}
	}

	for name, project := range v.settings.Projects {
		// Validate project path
		projectPath := project.Path

		// Handle tilde expansion for home directory
		if strings.HasPrefix(projectPath, "~/") && homeDir != "" {
			projectPath = filepath.Join(homeDir, projectPath[2:])
		} else if !filepath.IsAbs(projectPath) {
			projectPath = filepath.Join(homeDir, projectPath)
		}

		if filepath.IsAbs(project.Path) && !filepath.HasPrefix(project.Path, homeDir) {
			message := fmt.Sprintf("Project '%s' path must be inside $HOME: %s", name, project.Path)
			validationErrors = append(validationErrors, *errors.NewProjectError(message, nil, false))
		}

		if _, err := os.Stat(projectPath); os.IsNotExist(err) {
			message := fmt.Sprintf("Project '%s' path does not exist: %s", name, projectPath)
			validationErrors = append(validationErrors, *errors.NewProjectError(message, err, true))
		}

		// Validate project commands
		for _, alias := range project.Commands {
			if _, ok := v.settings.Commands[alias.CommandName]; !ok {
				message := fmt.Sprintf("Project '%s' references undefined command: %s", name, alias.CommandName)
				validationErrors = append(validationErrors, *errors.NewProjectError(message, nil, true))
			}
		}
	}

	return ValidationResult{
		Errors: validationErrors,
		Valid:  len(validationErrors) == 0,
	}
}

// ValidateProject checks if a specific project is valid
func (v *Validator) ValidateProject(projectName string) ValidationResult {
	var validationErrors []errors.AppError

	project, exists := v.settings.Projects[projectName]
	if !exists {
		return ValidationResult{
			Errors: []errors.AppError{*errors.NewProjectError(fmt.Sprintf("Project '%s' not found", projectName), nil, true)},
			Valid:  false,
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ValidationResult{
			Errors: []errors.AppError{*errors.NewPathError("Failed to get user home directory", err)},
			Valid:  false,
		}
	}

	// Validate project path
	projectPath := project.Path

	// Handle tilde expansion for home directory
	if strings.HasPrefix(projectPath, "~/") && homeDir != "" {
		projectPath = filepath.Join(homeDir, projectPath[2:])
	} else if !filepath.IsAbs(projectPath) {
		projectPath = filepath.Join(homeDir, projectPath)
	}

	if filepath.IsAbs(project.Path) && !filepath.HasPrefix(project.Path, homeDir) {
		message := fmt.Sprintf("Project '%s' path must be inside $HOME: %s", projectName, project.Path)
		validationErrors = append(validationErrors, *errors.NewProjectError(message, nil, false))
	}

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		message := fmt.Sprintf("Project '%s' path does not exist: %s", projectName, projectPath)
		validationErrors = append(validationErrors, *errors.NewProjectError(message, err, true))
	}

	// Validate project commands
	for _, alias := range project.Commands {
		if _, ok := v.settings.Commands[alias.CommandName]; !ok {
			message := fmt.Sprintf("Project '%s' references undefined command: %s", projectName, alias.CommandName)
			validationErrors = append(validationErrors, *errors.NewProjectError(message, nil, true))
		}
	}

	return ValidationResult{
		Errors: validationErrors,
		Valid:  len(validationErrors) == 0,
	}
}

// LogValidationErrors logs validation errors with appropriate severity levels
func LogValidationErrors(result ValidationResult) {
	if result.Valid {
		logging.Message("Project validation successful")
		return
	}

	for _, err := range result.Errors {
		if err.Severe {
			logging.Error("%s", err.Error())
		} else {
			logging.Warning("%s", err.Error())
		}
	}
}
