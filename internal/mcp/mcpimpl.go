package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"interop/internal/logging"
	"interop/internal/settings"
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
		server.WithLogging(),
	)

	// Create SSE server with the MCP server
	sseServer := server.NewSSEServer(mcpServer, server.WithBaseURL(fmt.Sprintf("http://localhost:%d", port)))

	s := &MCPLibServer{
		mcpServer:     mcpServer,
		sseServer:     sseServer,
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

		if len(cmd.Arguments) > 0 {
			for _, arg := range cmd.Arguments {
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

	// // Add a cursor-specific tool for better integration
	// cursorTool := mcp.NewTool(
	// 	"cursor",
	// 	mcp.WithDescription("Execute operations specifically for Cursor integration"),
	// 	mcp.WithString("operation",
	// 		mcp.Description("The operation to perform (get_tools, get_status)"),
	// 	),
	// )

	// s.mcpServer.AddTool(cursorTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 	// Extract operation from request
	// 	operation := ""
	// 	if opValue, ok := request.Params.Arguments["operation"]; ok {
	// 		if opStr, ok := opValue.(string); ok {
	// 			operation = opStr
	// 		}
	// 	}

	// 	// Handle different operations
	// 	switch operation {
	// 	case "get_tools":
	// 		// Return list of tools as text format
	// 		tools := s.GetToolNames()
	// 		result := "Available tools:\n"
	// 		for _, tool := range tools {
	// 			result += "- " + tool + "\n"
	// 		}
	// 		return mcp.NewToolResultText(sanitizeOutput(result)), nil

	// 	case "get_status":
	// 		// Return server status as text
	// 		result := fmt.Sprintf("Server status: ready\nTools count: %d\nStarted at: %s",
	// 			len(s.GetToolNames()),
	// 			time.Now().Format(time.RFC3339))
	// 		return mcp.NewToolResultText(sanitizeOutput(result)), nil

	// 	default:
	// 		return mcp.NewToolResultError(fmt.Sprintf("Unknown cursor operation: %s", operation)), nil
	// 	}
	// })

	// s.logInfo("Registered cursor tool for Cursor IDE integration")

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

	port := settings.GetMCPPort()

	if err := s.sseServer.Start(fmt.Sprintf(":%d", port)); err != nil {
		return fmt.Errorf("failed to start SSE server: %w", err)
	}

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

	if err := s.sseServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	return nil
}
