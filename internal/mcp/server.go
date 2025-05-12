package mcp

import (
	"fmt"
	"interop/internal/logging"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Server represents the MCP server
type Server struct {
	PidFile string
	LogFile string
}

// NewServer creates a new MCP server instance
func NewServer() (*Server, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create MCP directory if it doesn't exist
	mcpDir := filepath.Join(homeDir, ".config", "interop", "mcp")
	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create MCP directory: %w", err)
	}

	return &Server{
		PidFile: filepath.Join(mcpDir, "mcp.pid"),
		LogFile: filepath.Join(mcpDir, "mcp.log"),
	}, nil
}

// Start launches the MCP server as a daemon
func (s *Server) Start() error {
	// Check if server is already running
	if s.IsRunning() {
		return fmt.Errorf("MCP server is already running")
	}

	// Create log file
	logFile, err := os.OpenFile(s.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	// Get the path to the current executable
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Prepare command to run server in daemon mode
	// We use the current executable with a special flag to run the HTTP server
	cmd := exec.Command(executable, "mcp", "daemon")
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	// Write PID to file
	pid := cmd.Process.Pid
	if err := os.WriteFile(s.PidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
		// Try to kill the process if we couldn't write the PID file
		cmd.Process.Kill()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	logging.Message("MCP server started with PID %d", pid)
	logging.Message("HTTP server available at http://localhost:4567")
	return nil
}

// Stop terminates the MCP server
func (s *Server) Stop() error {
	pid, err := s.getPid()
	if err != nil {
		return fmt.Errorf("MCP server is not running: %w", err)
	}

	// Find the process
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Send SIGTERM to gracefully terminate
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// If SIGTERM fails, try SIGKILL
		if err := process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	// Wait for process to exit
	for i := 0; i < 10; i++ {
		if !s.IsRunning() {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Remove PID file
	if err := os.Remove(s.PidFile); err != nil {
		logging.Warning("Failed to remove PID file: %v", err)
	}

	logging.Message("MCP server stopped")
	return nil
}

// Restart restarts the MCP server
func (s *Server) Restart() error {
	if s.IsRunning() {
		if err := s.Stop(); err != nil {
			return fmt.Errorf("failed to stop MCP server: %w", err)
		}
	}

	// Wait a moment to ensure the previous process has completely terminated
	time.Sleep(1 * time.Second)

	return s.Start()
}

// IsRunning checks if the MCP server is running
func (s *Server) IsRunning() bool {
	pid, err := s.getPid()
	if err != nil {
		return false
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// Status returns the current status of the MCP server
func (s *Server) Status() string {
	if s.IsRunning() {
		pid, _ := s.getPid()
		return fmt.Sprintf("MCP server is running (PID: %d)\nHTTP server available at http://localhost:4567", pid)
	}
	return "MCP server is not running"
}

// getPid reads the PID from the PID file
func (s *Server) getPid() (int, error) {
	if _, err := os.Stat(s.PidFile); os.IsNotExist(err) {
		return 0, fmt.Errorf("PID file not found")
	}

	pidBytes, err := os.ReadFile(s.PidFile)
	if err != nil {
		return 0, fmt.Errorf("failed to read PID file: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}

	return pid, nil
}
