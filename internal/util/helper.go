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
	// LevelInfo shows errors and info messages
	LevelInfo
	// LevelVerbose shows all messages
	LevelVerbose
)

// ParseLogLevel converts a string log level to LogLevel constant
func ParseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "verbose":
		return LevelVerbose
	case "info":
		return LevelInfo
	default:
		return LevelError
	}
}

// Logger handles log operations with level filtering
type Logger struct {
	level LogLevel
}

// NewLogger creates a new logger with the specified log level
func NewLogger(level string) *Logger {
	return &Logger{
		level: ParseLogLevel(level),
	}
}

// Error prints a red "Error: …" message to stderr.
func (l *Logger) Error(format string, args ...interface{}) {
	// Error messages are always printed regardless of log level
	fmt.Fprintf(os.Stderr, colorRed+"Error: "+colorReset+format+"\n", args...)
}

// Info prints a yellow "Info: …" message to stdout if log level permits.
func (l *Logger) Info(format string, args ...interface{}) {
	if l.level >= LevelInfo {
		fmt.Printf(colorYellow+"Info: "+colorReset+format+"\n", args...)
	}
}

// Message prints a green "Message: …" message to stdout if log level permits.
func (l *Logger) Message(format string, args ...interface{}) {
	if l.level >= LevelVerbose {
		fmt.Printf(colorGreen+"Message: "+colorReset+format+"\n", args...)
	}
}

// Legacy standalone functions that use the default logger

var defaultLogger = &Logger{level: LevelError}

// SetDefaultLogLevel updates the log level of the default logger
func SetDefaultLogLevel(level string) {
	defaultLogger.level = ParseLogLevel(level)
}

// Error makes the program exit with a non-zero status when an error occurs
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
	// Make the program exit with a non-zero status when an error occurs
	os.Exit(1)
}

// Info prints a yellow "Info: …" message to stdout.
func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// Message prints a green "Message: …" message to stdout.
func Message(format string, args ...interface{}) {
	defaultLogger.Message(format, args...)
}
