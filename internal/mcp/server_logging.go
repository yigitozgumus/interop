package mcp

import (
	"fmt"
	"time"
)

// logToFile logs a message to the log file with a timestamp
func (s *MCPLibServer) logToFile(level, format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(s.logFile, "[%s] [%s] %s\n", timestamp, level, message)
}

// logInfo logs an informational message to the log file
func (s *MCPLibServer) logInfo(format string, args ...interface{}) {
	s.logToFile("INFO", format, args...)
}

// logWarning logs a warning message to the log file
func (s *MCPLibServer) logWarning(format string, args ...interface{}) {
	s.logToFile("WARNING", format, args...)
}

// logError logs an error message to the log file
func (s *MCPLibServer) logError(format string, args ...interface{}) {
	s.logToFile("ERROR", format, args...)
}
