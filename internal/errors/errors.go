package errors

import (
	"fmt"
)

// ErrorType categorizes errors by their source and severity
type ErrorType string

const (
	// ConfigError represents configuration-related errors
	ConfigError ErrorType = "config"
	// ValidationError represents validation errors
	ValidationError ErrorType = "validation"
	// ExecutionError represents command execution errors
	ExecutionError ErrorType = "execution"
	// PathError represents path resolution errors
	PathError ErrorType = "path"
	// ProjectError represents project-related errors
	ProjectError ErrorType = "project"
	// CommandError represents command-related errors
	CommandError ErrorType = "command"
)

// AppError represents an application-specific error with context
type AppError struct {
	Type    ErrorType
	Message string
	Err     error
	Severe  bool
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the wrapped error
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewConfigError creates a new config error
func NewConfigError(message string, err error) *AppError {
	return &AppError{
		Type:    ConfigError,
		Message: message,
		Err:     err,
		Severe:  true, // Config errors are typically severe
	}
}

// NewValidationError creates a new validation error
func NewValidationError(message string, err error, severe bool) *AppError {
	return &AppError{
		Type:    ValidationError,
		Message: message,
		Err:     err,
		Severe:  severe,
	}
}

// NewExecutionError creates a new execution error
func NewExecutionError(message string, err error) *AppError {
	return &AppError{
		Type:    ExecutionError,
		Message: message,
		Err:     err,
		Severe:  true, // Execution errors are typically severe
	}
}

// NewPathError creates a new path error
func NewPathError(message string, err error) *AppError {
	return &AppError{
		Type:    PathError,
		Message: message,
		Err:     err,
		Severe:  false, // Path errors might be recoverable
	}
}

// NewProjectError creates a new project error
func NewProjectError(message string, err error, severe bool) *AppError {
	return &AppError{
		Type:    ProjectError,
		Message: message,
		Err:     err,
		Severe:  severe,
	}
}

// NewCommandError creates a new command error
func NewCommandError(message string, err error, severe bool) *AppError {
	return &AppError{
		Type:    CommandError,
		Message: message,
		Err:     err,
		Severe:  severe,
	}
}

// Is checks if the target error is of the same type as this error
func (e *AppError) Is(target error) bool {
	if t, ok := target.(*AppError); ok {
		return e.Type == t.Type
	}
	return false
}
