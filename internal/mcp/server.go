package mcp

import (
	"encoding/json"
	"fmt"
	"interop/internal/logging"
	"interop/internal/settings"
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
	Name    string // Server name, empty for default
	Port    int    // Server port
}

// ServerManager manages multiple MCP servers
type ServerManager struct {
	Servers map[string]*Server // Map of server name to server instance
}

// NewServerManager creates a new MCP server manager
func NewServerManager() (*ServerManager, error) {
	cfg, err := settings.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load settings: %w", err)
	}

	manager := &ServerManager{
		Servers: make(map[string]*Server),
	}

	// Create default server
	defaultServer, err := NewServer("", cfg.MCPPort)
	if err != nil {
		return nil, err
	}
	manager.Servers["default"] = defaultServer

	// Create servers for each configured MCP server
	for name, mcpServer := range cfg.MCPServers {
		server, err := NewServer(name, mcpServer.Port)
		if err != nil {
			return nil, err
		}
		manager.Servers[name] = server
	}

	return manager, nil
}

// NewServer creates a new MCP server instance with the given name and port
func NewServer(name string, port int) (*Server, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create MCP directory if it doesn't exist
	mcpDir := filepath.Join(homeDir, ".config", "interop", "mcp")
	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create MCP directory: %w", err)
	}

	// Use name as prefix for files if not empty, otherwise use "default"
	prefix := "default"
	if name != "" {
		prefix = name
	}

	return &Server{
		PidFile: filepath.Join(mcpDir, prefix+".pid"),
		LogFile: filepath.Join(mcpDir, prefix+".log"),
		Name:    name,
		Port:    port,
	}, nil
}

// Start launches the MCP server as a daemon
func (s *Server) Start() error {
	// Check if server is already running
	if s.IsRunning() {
		serverType := "MCP server"
		if s.Name != "" {
			serverType = fmt.Sprintf("MCP server '%s'", s.Name)
		}
		return fmt.Errorf("%s is already running", serverType)
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

	// Prepare command to run server in daemon mode with port and name
	cmd := exec.Command(executable, "mcp", "daemon")

	// Add server name and port as environment variables
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("MCP_SERVER_NAME=%s", s.Name),
		fmt.Sprintf("MCP_SERVER_PORT=%d", s.Port))

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

	serverType := "MCP server"
	if s.Name != "" {
		serverType = fmt.Sprintf("MCP server '%s'", s.Name)
	}

	logging.Message("%s started with PID %d", serverType, pid)
	logging.Message("HTTP server available at http://localhost:%d", s.Port)
	return nil
}

// Stop terminates the MCP server
func (s *Server) Stop() error {
	pid, err := s.getPid()
	if err != nil {
		serverType := "MCP server"
		if s.Name != "" {
			serverType = fmt.Sprintf("MCP server '%s'", s.Name)
		}
		return fmt.Errorf("%s is not running: %w", serverType, err)
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

	serverType := "MCP server"
	if s.Name != "" {
		serverType = fmt.Sprintf("MCP server '%s'", s.Name)
	}

	logging.Message("%s stopped", serverType)
	return nil
}

// Restart restarts the MCP server
func (s *Server) Restart() error {
	if s.IsRunning() {
		if err := s.Stop(); err != nil {
			serverType := "MCP server"
			if s.Name != "" {
				serverType = fmt.Sprintf("MCP server '%s'", s.Name)
			}
			return fmt.Errorf("failed to stop %s: %w", serverType, err)
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

		serverType := "MCP server"
		if s.Name != "" {
			serverType = fmt.Sprintf("MCP server '%s'", s.Name)
		}

		return fmt.Sprintf("%s is running (PID: %d)\nHTTP server available at http://localhost:%d",
			serverType, pid, s.Port)
	}

	serverType := "MCP server"
	if s.Name != "" {
		serverType = fmt.Sprintf("MCP server '%s'", s.Name)
	}

	return fmt.Sprintf("%s is not running", serverType)
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

// StartServer starts a specific MCP server or all servers
func (m *ServerManager) StartServer(name string, all bool) error {
	if all {
		// Start all servers
		for serverName, server := range m.Servers {
			logging.Message("Starting MCP server: %s", serverName)
			if err := server.Start(); err != nil {
				logging.Warning("Failed to start MCP server '%s': %v", serverName, err)
			}
		}
		return nil
	}

	// Start a specific server by name
	if name == "" {
		// Default server
		return m.Servers["default"].Start()
	}

	server, exists := m.Servers[name]
	if !exists {
		return fmt.Errorf("MCP server '%s' not found", name)
	}

	return server.Start()
}

// StopServer stops a specific MCP server or all servers
func (m *ServerManager) StopServer(name string, all bool) error {
	if all {
		// Stop all servers
		for serverName, server := range m.Servers {
			logging.Message("Stopping MCP server: %s", serverName)
			if err := server.Stop(); err != nil {
				logging.Warning("Failed to stop MCP server '%s': %v", serverName, err)
			}
		}
		return nil
	}

	// Stop a specific server by name
	if name == "" {
		// Default server
		return m.Servers["default"].Stop()
	}

	server, exists := m.Servers[name]
	if !exists {
		return fmt.Errorf("MCP server '%s' not found", name)
	}

	return server.Stop()
}

// RestartServer restarts a specific MCP server or all servers
func (m *ServerManager) RestartServer(name string, all bool) error {
	if all {
		// Restart all servers
		for serverName, server := range m.Servers {
			logging.Message("Restarting MCP server: %s", serverName)
			if err := server.Restart(); err != nil {
				logging.Warning("Failed to restart MCP server '%s': %v", serverName, err)
			}
		}
		return nil
	}

	// Restart a specific server by name
	if name == "" {
		// Default server
		return m.Servers["default"].Restart()
	}

	server, exists := m.Servers[name]
	if !exists {
		return fmt.Errorf("MCP server '%s' not found", name)
	}

	return server.Restart()
}

// GetStatus returns the status of a specific MCP server or all servers
func (m *ServerManager) GetStatus(name string, all bool) string {
	// If a specific server is requested, only show that one
	if name != "" {
		server, exists := m.Servers[name]
		if !exists {
			return fmt.Sprintf("MCP server '%s' not found", name)
		}
		return server.Status()
	}

	// By default or if all flag is set, show all servers
	status := "MCP Servers Status:\n"
	status += "=====================\n"

	// First show default server
	status += fmt.Sprintf("\n[default]\n%s\n", m.Servers["default"].Status())

	// Then show all other servers
	for serverName, server := range m.Servers {
		if serverName != "default" {
			status += fmt.Sprintf("\n[%s]\n%s\n", serverName, server.Status())
		}
	}

	return status
}

// ListMCPServers returns a list of configured MCP servers with their details
func (m *ServerManager) ListMCPServers() string {
	cfg, err := settings.Load()
	if err != nil {
		return fmt.Sprintf("Failed to load settings: %v", err)
	}

	result := "Configured MCP Servers:\n"
	result += "=====================\n\n"

	// First show default server
	result += fmt.Sprintf("[default]\n")
	result += fmt.Sprintf("Port: %d\n", cfg.MCPPort)
	result += fmt.Sprintf("Status: %s\n\n", m.Servers["default"].Status())

	// Then show all other servers
	for name, mcpServer := range cfg.MCPServers {
		result += fmt.Sprintf("[%s]\n", name)
		result += fmt.Sprintf("Description: %s\n", mcpServer.Description)
		result += fmt.Sprintf("Port: %d\n", mcpServer.Port)

		if server, exists := m.Servers[name]; exists {
			result += fmt.Sprintf("Status: %s\n", server.Status())
		} else {
			result += "Status: Not initialized\n"
		}

		// Get commands for this server
		result += "\nCommands:\n"
		hasCommands := false

		for cmdName, cmd := range cfg.Commands {
			if cmd.MCP == name {
				result += fmt.Sprintf("- %s\n", cmdName)
				hasCommands = true
			}
		}

		if !hasCommands {
			result += "- No commands assigned\n"
		}

		result += "\n"
	}

	return result
}

// ExportMCPConfig returns a JSON representation of the MCP configuration
func (m *ServerManager) ExportMCPConfig() (string, error) {
	cfg, err := settings.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load settings: %v", err)
	}

	// Create output format with the required naming convention
	servers := make(map[string]map[string]string)

	// Add default server
	servers["default-interopMCPServer"] = map[string]string{
		"url": fmt.Sprintf("http://localhost:%d/sse", cfg.MCPPort),
	}

	// Add all configured MCP servers
	for name, mcpServer := range cfg.MCPServers {
		serverKey := fmt.Sprintf("%s-interopMCPServer", name)
		servers[serverKey] = map[string]string{
			"url": fmt.Sprintf("http://localhost:%d/sse", mcpServer.Port),
		}
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(servers, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal configuration: %v", err)
	}

	return string(jsonData), nil
}
