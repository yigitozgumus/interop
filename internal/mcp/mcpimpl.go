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
	mcpServer      *server.MCPServer
	sseServer      *server.SSEServer
	httpServer     *http.Server
	port           int
	configDir      string
	logFile        *os.File
	commandConfig  map[string]settings.CommandConfig
	commandAliases map[string]string // Maps alias -> original command name
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
		mcpServer:      mcpServer,
		sseServer:      sseServer,
		port:           port,
		configDir:      configDir,
		logFile:        logFile,
		commandConfig:  cfg.Commands,
		commandAliases: make(map[string]string),
	}

	// Register tools based on available commands
	s.registerCommandTools()

	// Write initial log message to file only, not stdout
	s.logInfo("MCP server initialized")

	return s, nil
}

// registerCommandTools converts the available commands to MCP tools
func (s *MCPLibServer) registerCommandTools() {
	// Map to track registered commands to avoid duplicates
	registeredTools := make(map[string]bool)

	// First, register all regular commands
	for name, cmd := range s.commandConfig {
		if !cmd.IsEnabled {
			continue
		}

		// Register the main command
		s.registerSingleCommandTool(name, cmd)
		registeredTools[name] = true
	}

	// Now register aliases from projects
	cfg, err := settings.Load()
	if err == nil {
		for _, project := range cfg.Projects {
			for _, cmdAlias := range project.Commands {
				// Skip if command doesn't have an alias
				if cmdAlias.Alias == "" {
					continue
				}

				// Find the original command
				cmd, exists := s.commandConfig[cmdAlias.CommandName]
				if !exists || !cmd.IsEnabled {
					s.logInfo("Skipping alias %s for command %s (command not found or disabled)",
						cmdAlias.Alias, cmdAlias.CommandName)
					continue
				}

				// Skip if this alias is already a registered command name
				if _, exists := registeredTools[cmdAlias.Alias]; exists {
					s.logInfo("Skipping alias %s for command %s (conflicts with existing command)",
						cmdAlias.Alias, cmdAlias.CommandName)
					continue
				}

				// Register the alias as a tool that points to the same command
				s.registerSingleCommandTool(cmdAlias.Alias, cmd)
				s.logInfo("Registered alias %s for command %s", cmdAlias.Alias, cmdAlias.CommandName)
				registeredTools[cmdAlias.Alias] = true

				// Store the alias mapping
				s.commandAliases[cmdAlias.Alias] = cmdAlias.CommandName
			}
		}
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

// registerSingleCommandTool registers a single command as an MCP tool
func (s *MCPLibServer) registerSingleCommandTool(name string, cmdConfig settings.CommandConfig) {
	// Create tool options
	toolOptions := []mcp.ToolOption{
		mcp.WithDescription(cmdConfig.Description),
	}

	if len(cmdConfig.Arguments) > 0 {
		for _, arg := range cmdConfig.Arguments {
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

		// Execute the command - use the actual command name from settings
		result, err := s.executeCommand(name, cmdConfig.Cmd, args)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Command execution failed: %v", err)), nil
		}

		// Return the sanitized result
		return mcp.NewToolResultText(sanitizeOutput(result)), nil
	})

	s.logInfo("Registered MCP tool for command: %s", name)
}

// executeCommand runs a command and returns its output
func (s *MCPLibServer) executeCommand(name, cmdStr string, args map[string]interface{}) (string, error) {
	// Check if the command is an alias, and if so use the original command name
	originalName := name
	if aliasTarget, isAlias := s.commandAliases[name]; isAlias {
		originalName = aliasTarget
		s.logInfo("Command %s is an alias for %s", name, originalName)
	}

	// Get the command from config using the original name
	cmdConfig, exists := s.commandConfig[originalName]
	if !exists {
		return "", fmt.Errorf("command '%s' not found", originalName)
	}

	// Check if command is enabled
	if !cmdConfig.IsEnabled {
		return "", fmt.Errorf("command '%s' is disabled", originalName)
	}

	// Validate arguments if defined
	if len(cmdConfig.Arguments) > 0 {
		if err := cmdConfig.ValidateArgs(args); err != nil {
			return "", fmt.Errorf("argument validation failed: %w", err)
		}
	}

	// Create a copy of the command string for substitution
	processedCmd := cmdStr

	// Check if command has a project context
	var projectPath string

	// If no project path in args, try to find the associated project
	if projectPath == "" {
		cfg, err := settings.Load()
		if err == nil {
			// Look through all projects to find if this command is associated with one
			for _, project := range cfg.Projects {
				for _, cmd := range project.Commands {
					if cmd.CommandName == originalName || cmd.Alias == originalName {
						// Found the project this command belongs to
						projectPath = project.Path
						s.logInfo("Found project binding for command %s: %s", originalName, projectPath)
						break
					}
				}
				if projectPath != "" {
					break
				}
			}
		}
	}

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

	s.logInfo("Executing command: %s (%s)", originalName, processedCmd)

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

	// Prepare the command based on project context
	var executeCmd string
	if projectPath != "" {
		// If project path is provided, add directory change before and after
		executeCmd = fmt.Sprintf("cd %s && %s && cd -", projectPath, processedCmd)
		s.logInfo("Running command in project directory: %s", projectPath)
	} else {
		executeCmd = processedCmd
	}

	// Execute command
	cmd := exec.Command("sh", "-c", executeCmd)
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
