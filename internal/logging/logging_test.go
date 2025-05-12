package logging

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	testCases := []struct {
		input    string
		expected Level
	}{
		{"verbose", LevelVerbose},
		{"VERBOSE", LevelVerbose},
		{"warning", LevelWarning},
		{"WARNING", LevelWarning},
		{"error", LevelError},
		{"ERROR", LevelError},
		{"", LevelError},        // Default case
		{"invalid", LevelError}, // Default case
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := ParseLevel(tc.input)
			if result != tc.expected {
				t.Errorf("ParseLevel(%q) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}

func TestLoggerSetLevel(t *testing.T) {
	logger := NewLogger(LevelError)
	if logger.level != LevelError {
		t.Errorf("Expected initial level to be LevelError, got %v", logger.level)
	}

	logger.SetLevel(LevelWarning)
	if logger.level != LevelWarning {
		t.Errorf("Expected level to be LevelWarning after SetLevel, got %v", logger.level)
	}

	logger.SetLevelFromString("verbose")
	if logger.level != LevelVerbose {
		t.Errorf("Expected level to be LevelVerbose after SetLevelFromString, got %v", logger.level)
	}
}

func captureOutput(f func()) string {
	// Redirect stdout to capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call the function that generates output
	f()

	// Restore stdout and read the captured output
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func captureStderr(f func()) string {
	// Redirect stderr to capture output
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Call the function that generates output
	f()

	// Restore stderr and read the captured output
	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestLoggerMessage(t *testing.T) {
	testCases := []struct {
		level    Level
		expected bool // Whether message should be printed
	}{
		{LevelError, false},
		{LevelWarning, false},
		{LevelVerbose, true},
	}

	for _, tc := range testCases {
		t.Run(levelToString(tc.level), func(t *testing.T) {
			logger := NewLogger(tc.level)
			output := captureOutput(func() {
				logger.Message("test message")
			})

			if tc.expected && !strings.Contains(output, "test message") {
				t.Errorf("Expected message to be printed at level %v, but it wasn't", tc.level)
			} else if !tc.expected && strings.Contains(output, "test message") {
				t.Errorf("Expected message not to be printed at level %v, but it was", tc.level)
			}
		})
	}
}

func TestLoggerWarning(t *testing.T) {
	testCases := []struct {
		level    Level
		expected bool // Whether warning should be printed
	}{
		{LevelError, false},
		{LevelWarning, true},
		{LevelVerbose, true},
	}

	for _, tc := range testCases {
		t.Run(levelToString(tc.level), func(t *testing.T) {
			logger := NewLogger(tc.level)
			output := captureOutput(func() {
				logger.Warning("test warning")
			})

			if tc.expected && !strings.Contains(output, "test warning") {
				t.Errorf("Expected warning to be printed at level %v, but it wasn't", tc.level)
			} else if !tc.expected && strings.Contains(output, "test warning") {
				t.Errorf("Expected warning not to be printed at level %v, but it was", tc.level)
			}
		})
	}
}

func TestLoggerError(t *testing.T) {
	logger := NewLogger(LevelError)
	output := captureStderr(func() {
		logger.Error("test error")
	})

	if !strings.Contains(output, "test error") {
		t.Error("Expected error to be printed, but it wasn't")
	}
}

func TestDefaultLoggerFunctions(t *testing.T) {
	// Test SetDefaultLevel
	originalLevel := DefaultLogger.level
	defer func() {
		DefaultLogger.level = originalLevel
	}()

	SetDefaultLevel(LevelVerbose)
	if DefaultLogger.level != LevelVerbose {
		t.Errorf("Expected DefaultLogger level to be LevelVerbose after SetDefaultLevel, got %v", DefaultLogger.level)
	}

	// Test SetDefaultLevelFromString
	SetDefaultLevelFromString("warning")
	if DefaultLogger.level != LevelWarning {
		t.Errorf("Expected DefaultLogger level to be LevelWarning after SetDefaultLevelFromString, got %v", DefaultLogger.level)
	}

	// Test Message with verbose level
	SetDefaultLevel(LevelVerbose)
	output := captureOutput(func() {
		Message("test message")
	})
	if !strings.Contains(output, "test message") {
		t.Error("Expected Message to print with LevelVerbose, but it didn't")
	}

	// Test Warning with warning level
	SetDefaultLevel(LevelWarning)
	output = captureOutput(func() {
		Warning("test warning")
	})
	if !strings.Contains(output, "test warning") {
		t.Error("Expected Warning to print with LevelWarning, but it didn't")
	}

	// Test Error
	output = captureStderr(func() {
		Error("test error")
	})
	if !strings.Contains(output, "test error") {
		t.Error("Expected Error to print, but it didn't")
	}
}

// Helper function to convert level to string for test naming
func levelToString(level Level) string {
	switch level {
	case LevelVerbose:
		return "LevelVerbose"
	case LevelWarning:
		return "LevelWarning"
	case LevelError:
		return "LevelError"
	default:
		return "Unknown"
	}
}
