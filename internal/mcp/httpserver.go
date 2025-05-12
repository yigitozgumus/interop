package mcp

import (
	"encoding/json"
	"fmt"
	"interop/internal/logging"
	"interop/internal/settings"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// MCPServer is the HTTP server for MCP
type MCPServer struct {
	Port     int
	DataDir  string
	Commands map[string]settings.CommandConfig
	handlers map[string]http.HandlerFunc
	// SSE related fields
	clients   map[chan string]bool
	clientsMu sync.Mutex
}

// CommandRequest represents a request to execute a command
type CommandRequest struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args,omitempty"`
}

// CommandResponse represents the result of a command execution
type CommandResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Output   string `json:"output,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
}

// ToolResponse represents a response from a tool
type ToolResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// SSEEvent represents an event for SSE
type SSEEvent struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

// NewMCPServer creates a new MCP HTTP server
func NewMCPServer() (*MCPServer, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create MCP data directory if it doesn't exist
	dataDir := filepath.Join(homeDir, ".config", "interop", "mcp", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create MCP data directory: %w", err)
	}

	// Load commands from settings
	cfg, err := settings.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load settings: %w", err)
	}

	server := &MCPServer{
		Port:     8080,
		DataDir:  dataDir,
		Commands: cfg.Commands,
		handlers: make(map[string]http.HandlerFunc),
		clients:  make(map[chan string]bool),
	}

	// Register default handlers
	server.registerHandlers()

	return server, nil
}

// Start starts the HTTP server
func (s *MCPServer) Start() error {
	// Register the main mux with all handlers
	mux := http.NewServeMux()

	// Create middleware to add CORS headers to all responses
	corsMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers for all responses
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")

			// Handle OPTIONS requests (preflight)
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			// Call the original handler
			next(w, r)
		}
	}

	// Add all registered handlers with CORS middleware
	for pattern, handler := range s.handlers {
		logging.Message("Registering handler for pattern: %s", pattern)
		mux.HandleFunc(pattern, corsMiddleware(handler))
	}

	// Add default handler for root
	mux.HandleFunc("/", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		resp := ToolResponse{
			Success: true,
			Message: "MCP Server is running",
			Data: map[string]interface{}{
				"version": "1.0.0",
				"time":    time.Now().Format(time.RFC3339),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))

	// Start the HTTP server
	addr := fmt.Sprintf(":%d", s.Port)
	logging.Message("Starting MCP HTTP server on %s", addr)

	// Run in goroutine to not block
	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			logging.Error("HTTP server error: %v", err)
		}
	}()

	// Send a test event after 3 seconds to verify broadcasting works
	go func() {
		time.Sleep(3 * time.Second)
		s.broadcastEvent("server_ready", map[string]interface{}{
			"message": "MCP Server is fully initialized and ready",
			"time":    time.Now().Format(time.RFC3339),
			"port":    s.Port,
		})
	}()

	return nil
}

// broadcastEvent sends an event to all connected SSE clients
func (s *MCPServer) broadcastEvent(event string, data interface{}) {
	// Create the event payload
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		logging.Error("Failed to marshal SSE event data: %v", err)
		return
	}

	// Format as SSE data (proper SSE format)
	message := fmt.Sprintf("event: %s\ndata: %s\n\n", event, string(payloadBytes))

	logging.Message("Broadcasting event: %s with %d bytes of data", event, len(payloadBytes))

	// Send to all clients
	s.clientsMu.Lock()
	clientCount := len(s.clients)
	sentCount := 0

	for client := range s.clients {
		select {
		case client <- message:
			// Message sent successfully
			sentCount++
		default:
			// Client is not receiving, remove it
			delete(s.clients, client)
			close(client)
		}
	}
	s.clientsMu.Unlock()

	logging.Message("Event broadcast complete: sent to %d/%d clients", sentCount, clientCount)
}

// executeCommand runs a command and returns its output
func (s *MCPServer) executeCommand(cmdName string, args map[string]interface{}) (CommandResponse, error) {
	response := CommandResponse{}

	// Check if command exists
	cmd, exists := s.Commands[cmdName]
	if !exists {
		return response, fmt.Errorf("command '%s' not found", cmdName)
	}

	// Check if command is enabled
	if !cmd.IsEnabled {
		return response, fmt.Errorf("command '%s' is disabled", cmdName)
	}

	// Validate arguments if defined
	if len(cmd.Arguments) > 0 {
		if err := cmd.ValidateArgs(args); err != nil {
			return response, fmt.Errorf("argument validation failed: %w", err)
		}
	}

	// Prepare command string with arguments
	cmdString := cmd.Cmd

	// First pass: replace argument placeholders with their values from definitions
	for _, argDef := range cmd.Arguments {
		// Get the value (using default if not provided)
		value, err := cmd.GetArgumentValue(argDef.Name, args)
		if err != nil {
			return response, fmt.Errorf("error getting argument value: %w", err)
		}

		// If the value is nil (not provided and no default), skip replacement
		if value == nil {
			continue
		}

		// Convert value to string based on type
		var valueStr string
		switch argDef.Type {
		case settings.ArgumentTypeBool:
			if boolVal, ok := value.(bool); ok {
				valueStr = fmt.Sprintf("%v", boolVal)
			} else {
				valueStr = fmt.Sprintf("%v", value)
			}
		case settings.ArgumentTypeNumber:
			valueStr = fmt.Sprintf("%v", value)
		default: // string or any other type
			valueStr = fmt.Sprintf("%v", value)
		}

		// Replace placeholder
		placeholder := "${" + argDef.Name + "}"
		cmdString = strings.ReplaceAll(cmdString, placeholder, valueStr)
	}

	// Second pass: handle any non-defined arguments (for backward compatibility)
	for key, value := range args {
		// Skip arguments that were already processed
		alreadyProcessed := false
		for _, argDef := range cmd.Arguments {
			if key == argDef.Name {
				alreadyProcessed = true
				break
			}
		}
		if alreadyProcessed {
			continue
		}

		// Replace ${key} with value in cmdString
		placeholder := "${" + key + "}"
		valStr := fmt.Sprintf("%v", value)
		cmdString = strings.ReplaceAll(cmdString, placeholder, valStr)
	}

	// Create a temporary directory for output
	outputDir, err := os.MkdirTemp(s.DataDir, "cmd-output-*")
	if err != nil {
		return response, fmt.Errorf("failed to create output directory: %w", err)
	}
	defer os.RemoveAll(outputDir)

	// Create output file
	outputFile := filepath.Join(outputDir, "output.txt")
	outFile, err := os.Create(outputFile)
	if err != nil {
		return response, fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Broadcast command start event
	s.broadcastEvent("command_start", map[string]interface{}{
		"name":    cmdName,
		"command": cmdString,
		"time":    time.Now().Format(time.RFC3339),
	})

	// Execute the command
	logging.Message("Executing command: %s", cmdString)
	execCmd := exec.Command("sh", "-c", cmdString)
	execCmd.Stdout = outFile
	execCmd.Stderr = outFile

	// Run the command
	err = execCmd.Run()
	if err != nil {
		var exitCode int
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}

		// Read output even if the command failed
		outFile.Seek(0, 0)
		output, _ := io.ReadAll(outFile)

		response.Success = false
		response.Message = fmt.Sprintf("Command failed with exit code %d", exitCode)
		response.Output = string(output)
		response.ExitCode = exitCode

		// Broadcast command failure event
		s.broadcastEvent("command_failed", map[string]interface{}{
			"name":      cmdName,
			"exit_code": exitCode,
			"error":     err.Error(),
			"time":      time.Now().Format(time.RFC3339),
		})

		return response, nil
	}

	// Read the output
	outFile.Seek(0, 0)
	output, err := io.ReadAll(outFile)
	if err != nil {
		return response, fmt.Errorf("failed to read command output: %w", err)
	}

	response.Success = true
	response.Message = "Command executed successfully"
	response.Output = string(output)
	response.ExitCode = 0

	// Broadcast command success event
	s.broadcastEvent("command_success", map[string]interface{}{
		"name":   cmdName,
		"output": string(output),
		"time":   time.Now().Format(time.RFC3339),
	})

	return response, nil
}

// registerHandlers registers all HTTP handlers
func (s *MCPServer) registerHandlers() {
	// Health check endpoint
	s.handlers["/health"] = func(w http.ResponseWriter, r *http.Request) {
		resp := ToolResponse{
			Success: true,
			Message: "Server is healthy",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	// SSE endpoint for event streaming
	s.handlers["/events"] = func(w http.ResponseWriter, r *http.Request) {
		// Set headers for SSE
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache, no-transform")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("X-Accel-Buffering", "no") // For Nginx

		// Log that a client connected
		clientIP := r.RemoteAddr
		logging.Message("SSE client connected from %s", clientIP)

		// Create a channel for this client
		messageChan := make(chan string, 10)

		// Register client
		s.clientsMu.Lock()
		s.clients[messageChan] = true
		s.clientsMu.Unlock()

		// Send initial connection event
		initialEvent := fmt.Sprintf("event: connected\ndata: {\"time\":\"%s\"}\n\n", time.Now().Format(time.RFC3339))
		fmt.Fprint(w, initialEvent)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		} else {
			logging.Warning("Streaming not supported by the underlying HTTP connection")
		}

		// Use request context to detect client disconnection
		ctx := r.Context()

		// Keep connection open until client disconnects
		heartbeatTicker := time.NewTicker(10 * time.Second)
		defer heartbeatTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				// Client disconnected, remove from clients
				logging.Message("SSE client %s disconnected", clientIP)
				s.clientsMu.Lock()
				delete(s.clients, messageChan)
				close(messageChan)
				s.clientsMu.Unlock()
				return
			case msg, ok := <-messageChan:
				if !ok {
					// Channel was closed
					return
				}
				// Send message to client
				if _, err := fmt.Fprint(w, msg); err != nil {
					logging.Warning("Failed to send event to client %s: %v", clientIP, err)
					return
				}
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			case <-heartbeatTicker.C:
				// Send a heartbeat to keep the connection alive
				heartbeat := fmt.Sprintf("event: heartbeat\ndata: {\"time\":\"%s\"}\n\n", time.Now().Format(time.RFC3339))
				if _, err := fmt.Fprint(w, heartbeat); err != nil {
					logging.Warning("Failed to send heartbeat to client %s: %v", clientIP, err)
					return
				}
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			}
		}
	}

	// List available commands
	s.handlers["/commands"] = func(w http.ResponseWriter, r *http.Request) {
		commands := make(map[string]interface{})

		for name, cmd := range s.Commands {
			if cmd.IsEnabled {
				commands[name] = map[string]interface{}{
					"description": cmd.Description,
					"cmd":         cmd.Cmd,
				}
			}
		}

		resp := ToolResponse{
			Success: true,
			Message: "Available commands",
			Data:    commands,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	// Execute a command
	s.handlers["/commands/execute"] = func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req CommandRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			http.Error(w, "Command name is required", http.StatusBadRequest)
			return
		}

		response, err := s.executeCommand(req.Name, req.Args)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}

	// Example tool endpoint
	s.handlers["/tools/echo"] = func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var input map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		resp := ToolResponse{
			Success: true,
			Message: "Echo successful",
			Data:    input,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	// Add more tool endpoints as needed
	s.handlers["/tools/list"] = func(w http.ResponseWriter, r *http.Request) {
		tools := []string{
			"echo",
			"commands",
			"execute",
			"events", // Add SSE endpoint to the list
			// Add more tools here
		}

		resp := ToolResponse{
			Success: true,
			Message: "Available tools",
			Data:    tools,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
