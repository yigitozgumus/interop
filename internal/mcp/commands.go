package mcp

import (
	"encoding/json"
	"fmt"
	"interop/internal/logging"
	"interop/internal/settings"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// StartServer starts the MCP server daemon with support for multiple servers
func StartServer(serverName string, all bool) error {
	manager, err := NewServerManager()
	if err != nil {
		logging.Error("failed to initialize MCP server manager: %v", err)
		return err
	}

	if err := manager.StartServer(serverName, all); err != nil {
		logging.ErrorAndExit("failed to start MCP server: %v", err)
		return err
	}

	if all {
		logging.Message("All MCP servers started successfully")
	} else if serverName == "" {
		logging.Message("Default MCP server started successfully")
	} else {
		logging.Message("MCP server '%s' started successfully", serverName)
	}

	return nil
}

// StopServer stops the MCP server daemon with support for multiple servers
func StopServer(serverName string, all bool) error {
	manager, err := NewServerManager()
	if err != nil {
		logging.Error("failed to initialize MCP server manager: %v", err)
		return err
	}

	if err := manager.StopServer(serverName, all); err != nil {
		logging.ErrorAndExit("failed to stop MCP server: %v", err)
		return err
	}

	if all {
		logging.Message("All MCP servers stopped successfully")
	} else if serverName == "" {
		logging.Message("Default MCP server stopped successfully")
	} else {
		logging.Message("MCP server '%s' stopped successfully", serverName)
	}

	return nil
}

// RestartServer restarts the MCP server daemon with support for multiple servers
func RestartServer(serverName string, all bool) error {
	manager, err := NewServerManager()
	if err != nil {
		logging.Error("failed to initialize MCP server manager: %v", err)
		return err
	}

	if err := manager.RestartServer(serverName, all); err != nil {
		logging.ErrorAndExit("failed to restart MCP server: %v", err)
		return err
	}

	if all {
		logging.Message("All MCP servers restarted successfully")
	} else if serverName == "" {
		logging.Message("Default MCP server restarted successfully")
	} else {
		logging.Message("MCP server '%s' restarted successfully", serverName)
	}

	return nil
}

// GetStatus returns the status of the MCP server with support for multiple servers
// By default it shows status for all servers
func GetStatus(serverName string, all bool) (string, error) {
	manager, err := NewServerManager()
	if err != nil {
		return "", fmt.Errorf("failed to initialize MCP server manager: %v", err)
	}

	return manager.GetStatus(serverName, all), nil
}

// ListMCPServers lists all configured MCP servers
func ListMCPServers() (string, error) {
	manager, err := NewServerManager()
	if err != nil {
		return "", fmt.Errorf("failed to initialize MCP server manager: %v", err)
	}

	return manager.ListMCPServers(), nil
}

// ExportMCPConfig exports the MCP configuration as JSON
func ExportMCPConfig() (string, error) {
	manager, err := NewServerManager()
	if err != nil {
		return "", fmt.Errorf("failed to initialize MCP server manager: %v", err)
	}

	return manager.ExportMCPConfig()
}

// StreamServerEvents subscribes to and displays events from the MCP server
func StreamServerEvents(serverName string) error {
	// Get server info to check if it's running
	manager, err := NewServerManager()
	if err != nil {
		err = fmt.Errorf("failed to initialize MCP server manager: %v", err)
		logging.Error("%v", err)
		return err
	}

	var server *Server
	if serverName == "" {
		// Use default server
		server = manager.Servers["default"]
	} else {
		// Use named server
		var exists bool
		server, exists = manager.Servers[serverName]
		if !exists {
			err = fmt.Errorf("MCP server '%s' not found", serverName)
			logging.Error("%v", err)
			return err
		}
	}

	if !server.IsRunning() {
		serverDesc := "MCP server"
		if serverName != "" {
			serverDesc = fmt.Sprintf("MCP server '%s'", serverName)
		}
		err = fmt.Errorf("%s is not running", serverDesc)
		logging.Error("%v", err)
		return err
	}

	port := server.Port
	serverDesc := "MCP server"
	if serverName != "" {
		serverDesc = fmt.Sprintf("MCP server '%s'", serverName)
	}

	fmt.Printf("Starting event stream from %s. Press Ctrl+C to exit.\n", serverDesc)
	fmt.Println("─────────────────────────────────────────────────────────────")

	// Set up signal handling for graceful exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create done channel
	doneChan := make(chan struct{})
	errChan := make(chan error, 1)

	// Start event streaming in a goroutine
	go func() {
		client := NewToolsClient()
		client.SetPort(port) // Use the correct port

		err := client.SubscribeToEvents(func(event string, data string) {
			// Detect and ignore heartbeat events unless in verbose mode
			if event == "heartbeat" {
				fmt.Printf("❤ Heartbeat received at %s\n", time.Now().Format(time.RFC3339))
				return
			}

			// Print a divider for each non-heartbeat event
			fmt.Println("─────────────────────────────────────────────────────────────")

			// For other events, pretty print the JSON
			fmt.Printf("📌 EVENT: %s\n", event)

			// Try to unmarshal and pretty print the data
			var prettyData interface{}
			if err := json.Unmarshal([]byte(data), &prettyData); err == nil {
				// Successfully parsed JSON
				prettyJSON, _ := json.MarshalIndent(prettyData, "", "  ")
				fmt.Printf("%s\n", string(prettyJSON))
			} else {
				// Not valid JSON, print raw data
				fmt.Printf("DATA: %s\n", data)
			}

			fmt.Println("─────────────────────────────────────────────────────────────")
		})

		if err != nil {
			errChan <- err
			return
		}
		close(doneChan)
	}()

	// Wait for signal, error, or completion
	select {
	case <-sigChan:
		fmt.Println("\nEvent streaming stopped by user.")
	case err := <-errChan:
		logging.Error("Event streaming error: %v", err)
		fmt.Printf("\nEvent streaming error: %v\n", err)
		return err
	case <-doneChan:
		fmt.Println("\nEvent streaming completed.")
	}

	return nil
}

// RunHTTPServer runs the MCP HTTP server directly (not as a daemon)
// This function is called by the daemon subprocess
func RunHTTPServer() error {
	// Disable colors in the logger to avoid JSON parsing issues
	logging.DisableColors()

	// Get server name from environment variable
	serverName := os.Getenv("MCP_SERVER_NAME")
	if serverName != "" {
		logging.Message("Initializing MCP server '%s'...", serverName)
	} else {
		logging.Message("Initializing default MCP server...")
	}

	// Create a new MCP library server with name and port from env variables
	mcpLibServer, err := NewMCPLibServer()
	if err != nil {
		err = fmt.Errorf("failed to create MCP library server: %w", err)
		logging.Error("%v", err)
		return err
	}

	// Start the HTTP server
	if err := mcpLibServer.Start(); err != nil {
		err = fmt.Errorf("failed to start MCP library server: %w", err)
		logging.Error("%v", err)
		return err
	}

	if serverName != "" {
		logging.Message("MCP server '%s' started and connected successfully", serverName)
	} else {
		logging.Message("Default MCP server started and connected successfully")
	}

	// Handle OS signals for graceful shutdown
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-signals

	// Stop the server gracefully when signal received
	if err := mcpLibServer.Stop(); err != nil {
		logging.Error("Error stopping MCP server: %v", err)
	}

	return nil
}

// CheckPortAvailability checks if the configured MCP server ports are available
func CheckPortAvailability() (string, error) {
	cfg, err := settings.Load()
	if err != nil {
		err = fmt.Errorf("failed to load settings: %v", err)
		logging.Error("%v", err)
		return "", err
	}

	result := "MCP Ports Availability Check:\n"
	result += "============================\n\n"

	// Check default port
	result += fmt.Sprintf("Default port %d: ", cfg.MCPPort)
	if IsPortAvailable(cfg.MCPPort) {
		result += "Available\n"
	} else {
		result += "In use\n"
		// Add process info
		processInfo := GetProcessUsingPort(cfg.MCPPort)
		result += fmt.Sprintf("Process using this port:\n%s\n", processInfo)
	}

	// Check configured server ports
	for name, server := range cfg.MCPServers {
		result += fmt.Sprintf("\nServer '%s' port %d: ", name, server.Port)
		if IsPortAvailable(server.Port) {
			result += "Available\n"
		} else {
			result += "In use\n"
			// Add process info
			processInfo := GetProcessUsingPort(server.Port)
			result += fmt.Sprintf("Process using this port:\n%s\n", processInfo)
		}
	}

	return result, nil
}
