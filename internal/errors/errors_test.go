package errors

import (
	"errors"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name   string
		appErr *AppError
		want   string
	}{
		{
			name: "with wrapped error",
			appErr: &AppError{
				Type:    ConfigError,
				Message: "failed to load config",
				Err:     errors.New("file not found"),
				Severe:  true,
			},
			want: "config: failed to load config (file not found)",
		},
		{
			name: "without wrapped error",
			appErr: &AppError{
				Type:    ValidationError,
				Message: "invalid project name",
				Err:     nil,
				Severe:  false,
			},
			want: "validation: invalid project name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.appErr.Error(); got != tt.want {
				t.Errorf("AppError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	appErr := &AppError{
		Type:    ExecutionError,
		Message: "execution failed",
		Err:     innerErr,
		Severe:  true,
	}

	if got := appErr.Unwrap(); got != innerErr {
		t.Errorf("AppError.Unwrap() = %v, want %v", got, innerErr)
	}
}

func TestAppError_Is(t *testing.T) {
	err1 := &AppError{Type: ConfigError, Message: "config error"}
	err2 := &AppError{Type: ConfigError, Message: "another config error"}
	err3 := &AppError{Type: ValidationError, Message: "validation error"}

	// Same error type should match
	if !err1.Is(err2) {
		t.Errorf("AppError.Is() expected %v to match %v", err1, err2)
	}

	// Different error types should not match
	if err1.Is(err3) {
		t.Errorf("AppError.Is() expected %v not to match %v", err1, err3)
	}
}

func TestErrorFactoryFunctions(t *testing.T) {
	innerErr := errors.New("inner error")

	tests := []struct {
		name       string
		errFunc    func() *AppError
		wantType   ErrorType
		wantErr    error
		wantSevere bool
	}{
		{
			name: "NewConfigError",
			errFunc: func() *AppError {
				return NewConfigError("config error", innerErr)
			},
			wantType:   ConfigError,
			wantErr:    innerErr,
			wantSevere: true,
		},
		{
			name: "NewValidationError",
			errFunc: func() *AppError {
				return NewValidationError("validation error", innerErr, false)
			},
			wantType:   ValidationError,
			wantErr:    innerErr,
			wantSevere: false,
		},
		{
			name: "NewExecutionError",
			errFunc: func() *AppError {
				return NewExecutionError("execution error", innerErr)
			},
			wantType:   ExecutionError,
			wantErr:    innerErr,
			wantSevere: true,
		},
		{
			name: "NewPathError",
			errFunc: func() *AppError {
				return NewPathError("path error", innerErr)
			},
			wantType:   PathError,
			wantErr:    innerErr,
			wantSevere: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.errFunc()
			if got.Type != tt.wantType {
				t.Errorf("Error type = %v, want %v", got.Type, tt.wantType)
			}
			if got.Err != tt.wantErr {
				t.Errorf("Inner error = %v, want %v", got.Err, tt.wantErr)
			}
			if got.Severe != tt.wantSevere {
				t.Errorf("Severe = %v, want %v", got.Severe, tt.wantSevere)
			}
		})
	}
}
