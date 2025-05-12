package logging

import (
	"fmt"
	"os"
	"strings"
)

// Color codes for terminal output
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorGreen  = "\033[32m"
)

// Level defines the minimum level of logs to output
type Level int

const (
	// LevelError only shows errors
	LevelError Level = iota
	// LevelWarning shows errors and warnings
	LevelWarning
	// LevelVerbose shows all messages including informational ones
	LevelVerbose
)

// Logger handles log operations with level filtering
type Logger struct {
	level Level
}

// DefaultLogger is used by global logging functions
var DefaultLogger = NewLogger(LevelError)

// NewLogger creates a new logger with the specified log level
func NewLogger(level Level) *Logger {
	return &Logger{
		level: level,
	}
}

// ParseLevel converts a string log level to Level constant
func ParseLevel(level string) Level {
	switch strings.ToLower(level) {
	case "verbose":
		return LevelVerbose
	case "warning":
		return LevelWarning
	default:
		return LevelError
	}
}

// SetLevel updates the log level of the logger
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// SetLevelFromString updates the log level of the logger from a string
func (l *Logger) SetLevelFromString(level string) {
	l.level = ParseLevel(level)
}

// Error prints a red "Error: …" message to stderr
func (l *Logger) Error(format string, args ...interface{}) {
	// Error messages are always printed regardless of log level
	fmt.Fprintf(os.Stderr, colorRed+"Error: "+colorReset+format+"\n", args...)
}

// ErrorAndExit prints an error message and exits the program with status code 1
func (l *Logger) ErrorAndExit(format string, args ...interface{}) {
	l.Error(format, args...)
	os.Exit(1)
}

// Warning prints a yellow "Warning: …" message to stdout if log level permits
func (l *Logger) Warning(format string, args ...interface{}) {
	if l.level >= LevelWarning {
		fmt.Printf(colorYellow+"Warning: "+colorReset+format+"\n", args...)
	}
}

// Message prints a green "Message: …" message to stdout if log level permits
func (l *Logger) Message(format string, args ...interface{}) {
	if l.level >= LevelVerbose {
		fmt.Printf(colorGreen+"Message: "+colorReset+format+"\n", args...)
	}
}

// Global functions that use the default logger

// SetDefaultLevel updates the log level of the default logger
func SetDefaultLevel(level Level) {
	DefaultLogger.SetLevel(level)
}

// SetDefaultLevelFromString updates the log level of the default logger from a string
func SetDefaultLevelFromString(level string) {
	DefaultLogger.SetLevelFromString(level)
}

// Error prints an error message to stderr
func Error(format string, args ...interface{}) {
	DefaultLogger.Error(format, args...)
}

// ErrorAndExit prints an error message and exits the program with status code 1
func ErrorAndExit(format string, args ...interface{}) {
	DefaultLogger.ErrorAndExit(format, args...)
}

// Warning prints a warning message to stdout if log level permits
func Warning(format string, args ...interface{}) {
	DefaultLogger.Warning(format, args...)
}

// Message prints an informational message to stdout if log level permits
func Message(format string, args ...interface{}) {
	DefaultLogger.Message(format, args...)
}
