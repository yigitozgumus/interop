package mcp

import (
	"encoding/json"
	"fmt"
	"interop/internal/logging"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// StartServer starts the MCP server daemon
func StartServer() error {
	server, err := NewServer()
	if err != nil {
		logging.Error("failed to initialize MCP server: %w", err)
	}

	if server.IsRunning() {
		logging.Message("MCP server is already running")
		return nil
	}

	if err := server.Start(); err != nil {
		logging.ErrorAndExit("failed to start MCP server: %w", err)
	}

	logging.Message("MCP server started successfully")
	return nil
}

// StopServer stops the MCP server daemon
func StopServer() error {
	server, err := NewServer()
	if err != nil {
		logging.ErrorAndExit("failed to initialize MCP server: %w", err)
	}

	if !server.IsRunning() {
		logging.Message("MCP server is not running")
		return nil
	}

	if err := server.Stop(); err != nil {
		logging.ErrorAndExit("failed to stop MCP server: %w", err)
	}

	logging.Message("MCP server stopped successfully")
	return nil
}

// RestartServer restarts the MCP server daemon
func RestartServer() error {
	server, err := NewServer()
	if err != nil {
		logging.ErrorAndExit("failed to initialize MCP server: %w", err)
	}

	if err := server.Restart(); err != nil {
		logging.ErrorAndExit("failed to restart MCP server: %w", err)
	}

	logging.Message("MCP server restarted successfully")
	return nil
}

// GetStatus returns the status of the MCP server
func GetStatus() (string, error) {
	server, err := NewServer()
	if err != nil {
		return "", fmt.Errorf("failed to initialize MCP server: %w", err)
	}

	return server.Status(), nil
}

// GetServerHealth checks if the MCP server is running and healthy
func GetServerHealth() error {
	// Check if server is running
	server, err := NewServer()
	if err != nil {
		logging.ErrorAndExit("failed to initialize MCP server: %w", err)
	}

	if !server.IsRunning() {
		logging.Error("MCP server is not running")
	}

	// Check server health
	client := NewToolsClient()
	resp, err := client.GetHealth()
	if err != nil {
		return fmt.Errorf("failed to check server health: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("server health check failed: %s", resp.Message)
	}

	logging.Message("MCP server is healthy")
	return nil
}

// ListServerCommands gets all available commands from the MCP server
func ListServerCommands() error {
	// Check if server is running
	server, err := NewServer()
	if err != nil {
		return fmt.Errorf("failed to initialize MCP server: %w", err)
	}

	if !server.IsRunning() {
		return fmt.Errorf("MCP server is not running")
	}

	// Get commands
	client := NewToolsClient()
	resp, err := client.ListCommands()
	if err != nil {
		return fmt.Errorf("failed to list commands: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("failed to list commands: %s", resp.Message)
	}

	// Pretty print commands
	commandsJSON, err := json.MarshalIndent(resp.Data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format commands: %w", err)
	}

	fmt.Println("Available commands:")
	fmt.Println(string(commandsJSON))
	return nil
}

// ExecuteServerCommand executes a command on the MCP server
func ExecuteServerCommand(name string, args map[string]interface{}) error {
	// Check if server is running
	server, err := NewServer()
	if err != nil {
		return fmt.Errorf("failed to initialize MCP server: %w", err)
	}

	if !server.IsRunning() {
		return fmt.Errorf("MCP server is not running")
	}

	// Execute command
	client := NewToolsClient()
	resp, err := client.ExecuteCommand(name, args)
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	// Display response
	fmt.Printf("Command: %s\n", name)
	fmt.Printf("Status: %s\n", resp.Message)
	if resp.ExitCode != 0 {
		fmt.Printf("Exit code: %d\n", resp.ExitCode)
	}
	fmt.Println("Output:")
	fmt.Println(resp.Output)

	return nil
}

// ListServerTools gets all available tools from the MCP server
func ListServerTools() error {
	// Check if server is running
	server, err := NewServer()
	if err != nil {
		return fmt.Errorf("failed to initialize MCP server: %w", err)
	}

	if !server.IsRunning() {
		return fmt.Errorf("MCP server is not running")
	}

	// Get tools
	client := NewToolsClient()
	resp, err := client.ListTools()
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("failed to list tools: %s", resp.Message)
	}

	// Pretty print tools
	toolsJSON, err := json.MarshalIndent(resp.Data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format tools: %w", err)
	}

	fmt.Println("Available tools:")
	fmt.Println(string(toolsJSON))
	return nil
}

// StreamServerEvents subscribes to and displays events from the MCP server
func StreamServerEvents() error {
	// Check if server is running
	server, err := NewServer()
	if err != nil {
		return fmt.Errorf("failed to initialize MCP server: %w", err)
	}

	if !server.IsRunning() {
		return fmt.Errorf("MCP server is not running")
	}

	fmt.Println("Starting event stream from MCP server. Press Ctrl+C to exit.")
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
		fmt.Printf("\nEvent streaming error: %v\n", err)
		return err
	case <-doneChan:
		fmt.Println("\nEvent streaming completed.")
	}

	return nil
}

// RunHTTPServer runs the MCP HTTP server directly (not as a daemon)
// This function is called by the daemon subprocess
func (s *Server) RunHTTPServer() error {
	// Disable colors in the logger to avoid JSON parsing issues
	logging.DisableColors()
	logging.Message("Initializing server...")

	// Create a new MCP library server
	mcpLibServer, err := NewMCPLibServer()
	if err != nil {
		return fmt.Errorf("failed to create MCP library server: %w", err)
	}

	// Start the HTTP server
	if err := mcpLibServer.Start(); err != nil {
		return fmt.Errorf("failed to start MCP library server: %w", err)
	}

	logging.Message("Server started and connected successfully")

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
