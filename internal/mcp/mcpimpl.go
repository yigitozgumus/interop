package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"interop/internal/logging"
	"interop/internal/settings"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// MCPLibServer represents the MCP server implementation using mark3labs/mcp-go
type MCPLibServer struct {
	mcpServer     *server.MCPServer
	sseServer     *server.SSEServer
	httpServer    *http.Server
	port          int
	configDir     string
	logFile       *os.File
	commandConfig map[string]settings.CommandConfig
}

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

// sanitizeOutput ensures there are no ANSI color codes in the output
// This helps prevent JSON parsing errors in the client
func sanitizeOutput(output string) string {
	// ANSI color code regex pattern
	colorPattern := regexp.MustCompile("\x1b\\[[0-9;]*m")
	return colorPattern.ReplaceAllString(output, "")
}

// NewMCPLibServer creates a new MCP server using the mark3labs/mcp-go library
func NewMCPLibServer() (*MCPLibServer, error) {
	// Disable colors in our internal logging package
	// This is essential to prevent color codes from corrupting JSON output
	logging.DisableColors()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create configuration directory
	configDir := filepath.Join(homeDir, ".config", "interop", "mcp")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create log file
	logFilePath := filepath.Join(configDir, "mcp-lib.log")
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// Redirect standard output to log file for MCP server logging
	// This is necessary because the MCP server logs to stdout
	// Save the original stdout for later restoration if needed
	originalStdout := os.Stdout
	os.Stdout = logFile

	// Make sure we restore stdout and close log file if there's an error
	cleanup := func() {
		os.Stdout = originalStdout
		logFile.Close()
	}

	// Load commands from settings
	cfg, err := settings.Load()
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to load settings: %w", err)
	}

	port := settings.GetMCPPort()

	// Create MCP server with logging disabled
	mcpServer := server.NewMCPServer(
		"Interop MCP Server",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Create HTTP server for the MCP
	httpServer := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
	}

	// Create SSE server with the MCP server
	sseServer := server.NewSSEServer(mcpServer)

	s := &MCPLibServer{
		mcpServer:     mcpServer,
		sseServer:     sseServer,
		httpServer:    httpServer,
		port:          port,
		configDir:     configDir,
		logFile:       logFile,
		commandConfig: cfg.Commands,
	}

	// Register tools based on available commands
	s.registerCommandTools()

	// Write initial log message to file only, not stdout
	s.logInfo("MCP server initialized")

	return s, nil
}

// registerCommandTools converts the available commands to MCP tools
func (s *MCPLibServer) registerCommandTools() {
	for name, cmd := range s.commandConfig {
		if !cmd.IsEnabled {
			continue
		}

		// Create tool options
		toolOptions := []mcp.ToolOption{
			mcp.WithDescription(cmd.Description),
		}

		// If the command has defined arguments, add them as separate parameters
		if len(cmd.Arguments) > 0 {
			for _, arg := range cmd.Arguments {
				// Add all arguments as strings for simplicity
				// The MCP library seems to only support strings and objects
				description := arg.Description
				if arg.Type != settings.ArgumentTypeString {
					description = fmt.Sprintf("%s (type: %s)", description, arg.Type)
				}

				toolOptions = append(toolOptions,
					mcp.WithString(arg.Name, mcp.Description(description)),
				)
			}
		} else {
			// For backward compatibility, keep the old 'args' parameter
			toolOptions = append(toolOptions,
				mcp.WithObject("args", mcp.Description("Optional arguments for the command")),
			)
		}

		// Create the tool with all options
		tool := mcp.NewTool(name, toolOptions...)

		// Store the command in a local variable to avoid closure issues
		cmdConfig := cmd
		cmdName := name

		// Add the tool handler
		s.mcpServer.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Handle arguments according to how they were defined
			var args map[string]interface{}
			if len(cmdConfig.Arguments) > 0 {
				// For commands with defined arguments, extract each from the request
				args = make(map[string]interface{})
				for _, arg := range cmdConfig.Arguments {
					if value, ok := request.Params.Arguments[arg.Name]; ok {
						// Convert values based on the expected type
						switch arg.Type {
						case settings.ArgumentTypeNumber:
							// Try to convert string to number if needed
							if strVal, ok := value.(string); ok {
								if numVal, err := strconv.ParseFloat(strVal, 64); err == nil {
									args[arg.Name] = numVal
								} else {
									args[arg.Name] = value
								}
							} else {
								args[arg.Name] = value
							}
						case settings.ArgumentTypeBool:
							// Try to convert string to bool if needed
							if strVal, ok := value.(string); ok {
								if boolVal, err := strconv.ParseBool(strVal); err == nil {
									args[arg.Name] = boolVal
								} else {
									args[arg.Name] = value
								}
							} else {
								args[arg.Name] = value
							}
						default:
							args[arg.Name] = value
						}
					}
				}
			} else {
				// For legacy commands, use the 'args' object if provided
				if rawArgs, ok := request.Params.Arguments["args"]; ok {
					if argsMap, ok := rawArgs.(map[string]interface{}); ok {
						args = argsMap
					}
				}
			}

			// Execute the command
			result, err := s.executeCommand(cmdName, cmdConfig.Cmd, args)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Command execution failed: %v", err)), nil
			}

			// Return the sanitized result
			return mcp.NewToolResultText(sanitizeOutput(result)), nil
		})

		s.logInfo("Registered MCP tool for command: %s", name)
	}

	// Add a special commands tool that lists available commands
	listCommandsTool := mcp.NewTool(
		"commands",
		mcp.WithDescription("List all available commands"),
	)

	s.mcpServer.AddTool(listCommandsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		commands := make(map[string]interface{})
		for name, cmd := range s.commandConfig {
			if cmd.IsEnabled {
				commands[name] = map[string]interface{}{
					"description": cmd.Description,
					"cmd":         cmd.Cmd,
				}
			}
		}

		// Format the output as JSON text
		cmdJSON, _ := json.MarshalIndent(commands, "", "  ")
		return mcp.NewToolResultText(sanitizeOutput(string(cmdJSON))), nil
	})

	s.logInfo("Registered MCP commands tool")

	// Add a cursor-specific tool for better integration
	cursorTool := mcp.NewTool(
		"cursor",
		mcp.WithDescription("Execute operations specifically for Cursor integration"),
		mcp.WithString("operation",
			mcp.Description("The operation to perform (get_tools, get_status)"),
		),
	)

	s.mcpServer.AddTool(cursorTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract operation from request
		operation := ""
		if opValue, ok := request.Params.Arguments["operation"]; ok {
			if opStr, ok := opValue.(string); ok {
				operation = opStr
			}
		}

		// Handle different operations
		switch operation {
		case "get_tools":
			// Return list of tools as text format
			tools := s.GetToolNames()
			result := "Available tools:\n"
			for _, tool := range tools {
				result += "- " + tool + "\n"
			}
			return mcp.NewToolResultText(sanitizeOutput(result)), nil

		case "get_status":
			// Return server status as text
			result := fmt.Sprintf("Server status: ready\nTools count: %d\nStarted at: %s",
				len(s.GetToolNames()),
				time.Now().Format(time.RFC3339))
			return mcp.NewToolResultText(sanitizeOutput(result)), nil

		default:
			return mcp.NewToolResultError(fmt.Sprintf("Unknown cursor operation: %s", operation)), nil
		}
	})

	s.logInfo("Registered cursor tool for Cursor IDE integration")

	// Add a simple ping tool that responds with pong - often expected by clients
	pingTool := mcp.NewTool(
		"ping",
		mcp.WithDescription("Simple ping/pong tool to check server responsiveness"),
	)

	s.mcpServer.AddTool(pingTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("pong"), nil
	})

	s.logInfo("Registered ping tool")

	// Add an echo tool - standard for MCP servers
	echoTool := mcp.NewTool(
		"echo",
		mcp.WithDescription("Echo back the input message"),
		mcp.WithString("message",
			mcp.Description("Message to echo back"),
		),
	)

	s.mcpServer.AddTool(echoTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		message := "Hello, world!"
		if msgValue, ok := request.Params.Arguments["message"]; ok {
			if msgStr, ok := msgValue.(string); ok {
				message = msgStr
			}
		}

		return mcp.NewToolResultText(sanitizeOutput(message)), nil
	})

	s.logInfo("Registered echo tool")
}

// executeCommand runs a command and returns its output
func (s *MCPLibServer) executeCommand(name, cmdStr string, args map[string]interface{}) (string, error) {
	// Get the command from config
	cmdConfig, exists := s.commandConfig[name]
	if !exists {
		return "", fmt.Errorf("command '%s' not found", name)
	}

	// Check if command is enabled
	if !cmdConfig.IsEnabled {
		return "", fmt.Errorf("command '%s' is disabled", name)
	}

	// Validate arguments if defined
	if len(cmdConfig.Arguments) > 0 {
		if err := cmdConfig.ValidateArgs(args); err != nil {
			return "", fmt.Errorf("argument validation failed: %w", err)
		}
	}

	// Create a copy of the command string for substitution
	processedCmd := cmdStr

	// First pass: replace argument placeholders with their values
	for _, argDef := range cmdConfig.Arguments {
		// Get the value (using default if not provided)
		value, err := cmdConfig.GetArgumentValue(argDef.Name, args)
		if err != nil {
			return "", fmt.Errorf("error getting argument value: %w", err)
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
		processedCmd = strings.ReplaceAll(processedCmd, placeholder, valueStr)
	}

	// Second pass: handle any non-defined arguments (for backward compatibility)
	for key, value := range args {
		// Skip arguments that were already processed
		alreadyProcessed := false
		for _, argDef := range cmdConfig.Arguments {
			if key == argDef.Name {
				alreadyProcessed = true
				break
			}
		}
		if alreadyProcessed {
			continue
		}

		// Replace the placeholder with the value
		placeholder := "${" + key + "}"
		valueStr := fmt.Sprintf("%v", value)
		processedCmd = strings.ReplaceAll(processedCmd, placeholder, valueStr)
	}

	s.logInfo("Executing command: %s (%s)", name, processedCmd)

	// Create a temporary file for output
	tmpDir, err := os.MkdirTemp(s.configDir, "cmd-output-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	outputFile := filepath.Join(tmpDir, "output.txt")
	outFile, err := os.Create(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Execute command
	cmd := exec.Command("sh", "-c", processedCmd)
	cmd.Stdout = outFile
	cmd.Stderr = outFile

	err = cmd.Run()
	if err != nil {
		// Still read output even if command failed
		outFile.Seek(0, 0)
		output, _ := os.ReadFile(outputFile)

		// Make sure to sanitize the output to remove any ANSI color codes
		return sanitizeOutput(fmt.Sprintf("Command failed: %v\nOutput:\n%s", err, string(output))), err
	}

	// Read command output
	outFile.Seek(0, 0)
	output, err := os.ReadFile(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read command output: %w", err)
	}

	// Return sanitized output
	return sanitizeOutput(string(output)), nil
}

// Start starts the MCP server with HTTP and SSE
func (s *MCPLibServer) Start() error {
	s.logInfo("Starting MCP server with HTTP on port %d", s.port)

	// Ensure colors are disabled again just before starting server
	logging.DisableColors()

	// Set up HTTP handlers
	mux := http.NewServeMux()

	// Root handler
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		toolNames := s.GetToolNames()
		response := map[string]interface{}{
			"status":       "ok",
			"message":      "MCP Server is running",
			"tools_count":  len(toolNames),
			"tools":        toolNames,
			"mcp":          true,
			"version":      "1.0.0",
			"name":         "Interop MCP Server",
			"capabilities": []string{"jsonrpc", "sse", "tools"},
		}
		json.NewEncoder(w).Encode(response)
	})

	// Register SSE handler for events
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		// Set headers for SSE
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache, no-transform")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("X-Accel-Buffering", "no") // For Nginx

		// Log that a client connected
		clientIP := r.RemoteAddr
		s.logInfo("SSE client connected from %s", clientIP)

		// Use context to detect when client disconnects
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		// Create flusher for streaming
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		// Send initial connection event
		initialEvent := fmt.Sprintf("event: connected\ndata: {\"time\":\"%s\"}\n\n", time.Now().Format(time.RFC3339))
		fmt.Fprint(w, initialEvent)
		flusher.Flush()

		// Start heartbeat ticker
		heartbeatTicker := time.NewTicker(10 * time.Second)
		defer heartbeatTicker.Stop()

		// Listen for disconnect
		go func() {
			<-ctx.Done()
			heartbeatTicker.Stop()
			s.logInfo("SSE client %s disconnected", clientIP)
		}()

		// Keep the connection open and send heartbeats
		for {
			select {
			case <-ctx.Done():
				return
			case <-heartbeatTicker.C:
				heartbeat := fmt.Sprintf("event: heartbeat\ndata: {\"time\":\"%s\"}\n\n", time.Now().Format(time.RFC3339))
				if _, err := fmt.Fprint(w, heartbeat); err != nil {
					s.logWarning("Failed to send heartbeat: %v", err)
					return
				}
				flusher.Flush()
			}
		}
	})

	// Add compatibility route for SSE for clients that might be using different paths
	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		// Redirect to the standard /events endpoint
		http.Redirect(w, r, "/events", http.StatusPermanentRedirect)
	})

	// Handle MCP JSON-RPC requests
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		// For SSE connection requests, redirect to SSE handler
		if r.Header.Get("Accept") == "text/event-stream" {
			s.logInfo("SSE connection attempt on /mcp - redirecting to /events")
			http.Redirect(w, r, "/events", http.StatusTemporaryRedirect)
			return
		}

		// Set common headers
		w.Header().Set("Content-Type", "application/json")

		// Check if this is a JSON-RPC request
		if r.Method == http.MethodPost {
			var rpcRequest struct {
				JsonRpc string          `json:"jsonrpc"`
				Id      interface{}     `json:"id"`
				Method  string          `json:"method"`
				Params  json.RawMessage `json:"params"`
			}

			// Read the body
			body, err := io.ReadAll(r.Body)
			if err != nil {
				s.logError("Error reading request body: %v", err)
				errorResponse := map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      nil,
					"error": map[string]interface{}{
						"code":    -32700,
						"message": "Parse error: invalid JSON was received by the server",
					},
				}
				json.NewEncoder(w).Encode(errorResponse)
				return
			}

			// Log raw request for debugging
			s.logInfo("Received JSON-RPC request: %s", string(body))

			// Parse request
			if err := json.Unmarshal(body, &rpcRequest); err != nil {
				s.logError("Error parsing JSON-RPC request: %v", err)
				errorResponse := map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      nil,
					"error": map[string]interface{}{
						"code":    -32700,
						"message": "Parse error: invalid JSON was received by the server",
					},
				}
				json.NewEncoder(w).Encode(errorResponse)
				return
			}

			// Handle different methods directly for better compatibility
			if rpcRequest.Method == "mcpGetTools" {
				// Return list of available tools directly
				tools := s.GetToolNames()
				toolsObjects := make([]map[string]interface{}, 0, len(tools))

				for _, name := range tools {
					description := "Tool for executing commands"
					if name == "echo" {
						description = "Echo back the input message"
					} else if name == "ping" {
						description = "Simple ping/pong tool"
					} else if name == "commands" {
						description = "List all available commands"
					} else if name == "cursor" {
						description = "Cursor integration operations"
					}

					toolsObjects = append(toolsObjects, map[string]interface{}{
						"name":        name,
						"description": description,
					})
				}

				response := map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      rpcRequest.Id,
					"result":  toolsObjects,
				}

				json.NewEncoder(w).Encode(response)
				return
			} else if rpcRequest.Method == "mcpCallTool" {
				// Parse params
				var params struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments"`
				}

				if err := json.Unmarshal(rpcRequest.Params, &params); err != nil {
					s.logError("Error parsing mcpCallTool params: %v", err)
					errorResponse := map[string]interface{}{
						"jsonrpc": "2.0",
						"id":      rpcRequest.Id,
						"error": map[string]interface{}{
							"code":    -32602,
							"message": "Invalid params for mcpCallTool",
						},
					}
					json.NewEncoder(w).Encode(errorResponse)
					return
				}

				s.logInfo("Tool call request: %s with args: %v", params.Name, params.Arguments)

				var result string
				var resultErr error

				// Handle different tools directly
				switch params.Name {
				case "echo":
					message := "Hello, World!"
					if msgVal, ok := params.Arguments["message"]; ok {
						if msgStr, ok := msgVal.(string); ok {
							message = msgStr
						}
					}
					result = message

				case "ping":
					result = "pong"

				case "commands":
					commands := make(map[string]interface{})
					for name, cmd := range s.commandConfig {
						if cmd.IsEnabled {
							commands[name] = map[string]interface{}{
								"description": cmd.Description,
								"cmd":         cmd.Cmd,
							}
						}
					}
					commandsJSON, _ := json.MarshalIndent(commands, "", "  ")
					result = string(commandsJSON)

				case "cursor":
					operation := ""
					if opVal, ok := params.Arguments["operation"]; ok {
						if opStr, ok := opVal.(string); ok {
							operation = opStr
						}
					}

					switch operation {
					case "get_tools":
						tools := s.GetToolNames()
						resultStr := "Available tools:\n"
						for _, tool := range tools {
							resultStr += "- " + tool + "\n"
						}
						result = resultStr
					case "get_status":
						result = fmt.Sprintf("Server status: ready\nTools count: %d\nStarted at: %s",
							len(s.GetToolNames()),
							time.Now().Format(time.RFC3339))
					default:
						resultErr = fmt.Errorf("Unknown cursor operation: %s", operation)
					}

				default:
					// Try to execute as a command if it exists in our config
					if cmd, exists := s.commandConfig[params.Name]; exists && cmd.IsEnabled {
						result, resultErr = s.executeCommand(params.Name, cmd.Cmd, params.Arguments)
					} else {
						resultErr = fmt.Errorf("Tool not found: %s", params.Name)
					}
				}

				// Return response
				if resultErr != nil {
					errorResponse := map[string]interface{}{
						"jsonrpc": "2.0",
						"id":      rpcRequest.Id,
						"error": map[string]interface{}{
							"code":    -32603,
							"message": resultErr.Error(),
						},
					}
					json.NewEncoder(w).Encode(errorResponse)
				} else {
					response := map[string]interface{}{
						"jsonrpc": "2.0",
						"id":      rpcRequest.Id,
						"result": map[string]interface{}{
							"kind":  "text",
							"value": sanitizeOutput(result),
						},
					}
					json.NewEncoder(w).Encode(response)
				}
				return
			}
		}

		// For other requests, delegate to the standard handler
		s.sseServer.ServeHTTP(w, r)
	})

	// Add handler for health checks - for backward compatibility
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := ToolResponse{
			Success: true,
			Message: "Server is healthy",
		}
		json.NewEncoder(w).Encode(resp)
	})

	// Add backward-compatible handler for listing commands
	mux.HandleFunc("/commands", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		commands := make(map[string]interface{})
		for name, cmd := range s.commandConfig {
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

		json.NewEncoder(w).Encode(resp)
	})

	// Add backward-compatible handler for executing commands
	mux.HandleFunc("/commands/execute", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Only allow POST
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse request
		var req CommandRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			http.Error(w, "Command name is required", http.StatusBadRequest)
			return
		}

		// Check if command exists
		cmd, exists := s.commandConfig[req.Name]
		if !exists {
			http.Error(w, fmt.Sprintf("Command '%s' not found", req.Name), http.StatusBadRequest)
			return
		}

		if !cmd.IsEnabled {
			http.Error(w, fmt.Sprintf("Command '%s' is disabled", req.Name), http.StatusBadRequest)
			return
		}

		// Execute command
		output, err := s.executeCommand(req.Name, cmd.Cmd, req.Args)

		// Prepare response
		resp := CommandResponse{
			Success: err == nil,
			Message: "Command executed successfully",
			Output:  output,
		}

		if err != nil {
			resp.Success = false
			resp.Message = fmt.Sprintf("Command failed: %v", err)
			resp.ExitCode = 1
		}

		json.NewEncoder(w).Encode(resp)
	})

	// Add backward-compatible handler for listing tools
	mux.HandleFunc("/tools/list", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		tools := []string{
			"echo",
			"commands",
			"execute",
			"events",
		}

		resp := ToolResponse{
			Success: true,
			Message: "Available tools",
			Data:    tools,
		}

		json.NewEncoder(w).Encode(resp)
	})

	// Add a debug endpoint for diagnostics
	mux.HandleFunc("/diagnostics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Build comprehensive diagnostic information
		diagnostics := map[string]interface{}{
			"status": "ok",
			"server_info": map[string]interface{}{
				"port":        s.port,
				"config_dir":  s.configDir,
				"started_at":  time.Now().Format(time.RFC3339),
				"api_version": "1.0.0",
			},
			"tools": map[string]interface{}{
				"names":  s.GetToolNames(),
				"count":  len(s.GetToolNames()),
				"source": "command_config",
			},
			"commands": func() map[string]interface{} {
				commands := make(map[string]interface{})
				for name, cmd := range s.commandConfig {
					if cmd.IsEnabled {
						commands[name] = map[string]interface{}{
							"description":        cmd.Description,
							"cmd":                cmd.Cmd,
							"registered_as_tool": true,
						}
					}
				}
				return commands
			}(),
		}

		json.NewEncoder(w).Encode(diagnostics)
	})

	// Add a capabilities endpoint specifically for client detection
	mux.HandleFunc("/capabilities", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		capabilities := map[string]interface{}{
			"jsonrpc":         true,
			"sse":             true,
			"tools":           true,
			"version":         "1.0.0",
			"name":            "Interop MCP Server",
			"description":     "Management Control Panel for Interop CLI",
			"available_tools": s.GetToolNames(),
		}

		json.NewEncoder(w).Encode(capabilities)
	})

	// Set the handler
	s.httpServer.Handler = mux

	// Start HTTP server
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logError("HTTP server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the MCP server
func (s *MCPLibServer) Stop() error {
	s.logInfo("Stopping MCP server")

	// Restore stdout before closing the log file
	os.Stdout = os.Stderr

	// Close log file
	if s.logFile != nil {
		s.logFile.Close()
	}

	// Gracefully shutdown the HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	return nil
}

// GetToolNames returns a list of all registered tool names
func (s *MCPLibServer) GetToolNames() []string {
	// Get the tool names directly from the MCP server if possible
	// Otherwise return a list of command names since they're registered as tools
	names := make([]string, 0)

	// Add the special tools
	names = append(names, "commands")

	// Add all enabled commands
	for name, cmd := range s.commandConfig {
		if cmd.IsEnabled {
			names = append(names, name)
		}
	}

	return names
}
