package util

import (
	"fmt"
	"os"
	"strings"
)

// coloured console helpers
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorGreen  = "\033[32m"
)

// LogLevel defines the minimum level of logs to output
type LogLevel int

const (
	// LevelError only shows errors
	LevelError LogLevel = iota
	// LevelWarning shows errors and info messages
	LevelWarning
	// LevelVerbose shows all messages
	LevelVerbose
)

// ParseLogLevel converts a string log level to LogLevel constant
func ParseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "verbose":
		return LevelVerbose
	case "warning":
		return LevelWarning
	default:
		return LevelError
	}
}

// Logger handles log operations with level filtering
type Logger struct {
	level     LogLevel
	useColors bool
}

// NewLogger creates a new logger with the specified log level
func NewLogger(level string) *Logger {
	return &Logger{
		level:     ParseLogLevel(level),
		useColors: true,
	}
}

// DisableColors turns off color formatting in log messages
func (l *Logger) DisableColors() {
	l.useColors = false
}

// EnableColors turns on color formatting in log messages
func (l *Logger) EnableColors() {
	l.useColors = true
}

// Error prints a red "Error: …" message to stderr.
func (l *Logger) Error(format string, args ...interface{}) {
	// Error messages are always printed regardless of log level
	if l.useColors {
		fmt.Fprintf(os.Stderr, colorRed+"Error: "+colorReset+format+"\n", args...)
	} else {
		fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	}
}

// Warning prints a yellow "Warning: …" message to stdout if log level permits.
func (l *Logger) Warning(format string, args ...interface{}) {
	if l.level >= LevelWarning {
		if l.useColors {
			fmt.Printf(colorYellow+"Warning: "+colorReset+format+"\n", args...)
		} else {
			fmt.Printf("Warning: "+format+"\n", args...)
		}
	}
}

// Message prints a green "Message: …" message to stdout if log level permits.
func (l *Logger) Message(format string, args ...interface{}) {
	if l.level >= LevelVerbose {
		if l.useColors {
			fmt.Printf(colorGreen+"Message: "+colorReset+format+"\n", args...)
		} else {
			fmt.Printf("Message: "+format+"\n", args...)
		}
	}
}

// Legacy standalone functions that use the default logger

var defaultLogger = &Logger{level: LevelError, useColors: true}

// SetDefaultLogLevel updates the log level of the default logger
func SetDefaultLogLevel(level string) {
	defaultLogger.level = ParseLogLevel(level)
}

// DisableColors turns off color formatting in the default logger
func DisableColors() {
	defaultLogger.useColors = false
}

// EnableColors turns on color formatting in the default logger
func EnableColors() {
	defaultLogger.useColors = true
}

// Error makes the program exit with a non-zero status when an error occurs
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
	// Make the program exit with a non-zero status when an error occurs
	os.Exit(1)
}

// Warning prints a yellow "Warning: …" message to stdout.
func Warning(format string, args ...interface{}) {
	defaultLogger.Warning(format, args...)
}

// Message prints a green "Message: …" message to stdout.
func Message(format string, args ...interface{}) {
	defaultLogger.Message(format, args...)
}
