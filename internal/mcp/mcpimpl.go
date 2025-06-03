package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"interop/internal/logging"
	"interop/internal/settings"
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
	httpServer     *server.StreamableHTTPServer
	port           int
	configDir      string
	logFile        *os.File
	commandConfig  map[string]settings.CommandConfig
	promptConfig   map[string]settings.PromptConfig
	commandAliases map[string]string // Maps alias -> original command name
	serverMode     string            // "stdio" or "sse"
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

	// Get server name, port and mode from environment variables if available
	serverName := os.Getenv("MCP_SERVER_NAME")
	serverMode := os.Getenv("MCP_SERVER_MODE")
	if serverMode == "" {
		serverMode = "sse" // Default to SSE mode if not specified
	}

	// Validate server mode
	if serverMode != "stdio" && serverMode != "sse" {
		return nil, fmt.Errorf("invalid server mode: %s, must be either 'stdio' or 'sse'", serverMode)
	}

	// Determine the port to use
	var port int
	portStr := os.Getenv("MCP_SERVER_PORT")
	if portStr != "" {
		var err error
		port, err = strconv.Atoi(portStr)
		if err != nil {
			logging.Warning("Invalid MCP_SERVER_PORT environment variable: %v, using default", err)
			port = settings.GetMCPPort()
		}
	} else {
		// If no specific port is provided, check if we're running a named server
		if serverName != "" {
			// For a named server, get its port from settings
			cfg, err := settings.Load()
			if err != nil {
				return nil, fmt.Errorf("failed to load settings: %w", err)
			}

			if serverCfg, exists := cfg.MCPServers[serverName]; exists {
				port = serverCfg.Port
			} else {
				return nil, fmt.Errorf("MCP server '%s' not defined in settings", serverName)
			}
		} else {
			// Default server, use default port
			port = settings.GetMCPPort()
		}
	}

	// Use server name to create log file name
	logFileName := "mcp-lib.log"
	if serverName != "" {
		logFileName = fmt.Sprintf("mcp-lib-%s.log", serverName)
	}

	// Create log file
	logFilePath := filepath.Join(configDir, logFileName)
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

	// Create MCP server with logging disabled
	serverTitle := "Interop MCP Server"
	if serverName != "" {
		serverTitle = fmt.Sprintf("Interop MCP Server - %s", serverName)
	}

	mcpServer := server.NewMCPServer(
		serverTitle,
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithLogging(),
	)

	s := &MCPLibServer{
		mcpServer:      mcpServer,
		httpServer:     nil,
		port:           port,
		configDir:      configDir,
		logFile:        logFile,
		commandConfig:  cfg.Commands,
		promptConfig:   cfg.Prompts,
		commandAliases: make(map[string]string),
		serverMode:     serverMode,
	}

	// Register tools based on available commands for this server
	s.registerCommandTools(serverName)

	// Register prompts based on configuration for this server
	s.registerPrompts(serverName)

	// Create the appropriate server based on mode
	if serverMode == "stdio" {
		// No need to create HTTP server for stdio mode
	} else {
		// Create HTTP server for SSE mode
		s.httpServer = server.NewStreamableHTTPServer(mcpServer)
	}

	// Write initial log message to file only, not stdout
	if serverName != "" {
		s.logInfo("MCP server '%s' initialized on port %d", serverName, port)
	} else {
		s.logInfo("Default MCP server initialized on port %d", port)
	}

	return s, nil
}

// registerCommandTools converts the available commands to MCP tools
func (s *MCPLibServer) registerCommandTools(serverName string) {
	// Map to track registered commands to avoid duplicates
	registeredTools := make(map[string]bool)

	// First, register all commands for this server
	for name, cmd := range s.commandConfig {
		if !cmd.IsEnabled {
			continue
		}

		// If we have a specific server name, only add commands belonging to this server
		if serverName != "" {
			// Only add commands assigned to this server or with no MCP field
			if cmd.MCP != "" && cmd.MCP != serverName {
				// Skip commands assigned to a different server
				continue
			}
		} else {
			// For default server, only add commands with no MCP field
			if cmd.MCP != "" {
				// Skip commands assigned to a specific server
				continue
			}
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

				// Filter by server name
				if serverName != "" {
					// For a named server, only include commands for this server
					if cmd.MCP != "" && cmd.MCP != serverName {
						// Skip commands assigned to a different server
						continue
					}
				} else {
					// For default server, only include commands with no MCP field
					if cmd.MCP != "" {
						// Skip commands assigned to a specific server
						continue
					}
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

		// Show only commands for this server
		for name, cmd := range s.commandConfig {
			if cmd.IsEnabled {
				// Filter by server name
				if serverName != "" {
					// For a named server, only include commands for this server
					if cmd.MCP != "" && cmd.MCP != serverName {
						// Skip commands assigned to a different server
						continue
					}
				} else {
					// For default server, only include commands with no MCP field
					if cmd.MCP != "" {
						// Skip commands assigned to a specific server
						continue
					}
				}

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
}

// registerPrompts registers prompts from configuration as MCP prompts
func (s *MCPLibServer) registerPrompts(serverName string) {
	// Register prompts for this server
	for name, promptConfig := range s.promptConfig {
		// Filter by server name similar to commands
		if serverName != "" {
			// For a named server, only add prompts assigned to this server or with no MCP field
			if promptConfig.MCP != "" && promptConfig.MCP != serverName {
				// Skip prompts assigned to a different server
				continue
			}
		} else {
			// For default server, only add prompts with no MCP field
			if promptConfig.MCP != "" {
				// Skip prompts assigned to a specific server
				continue
			}
		}

		// Create prompt options starting with description
		promptOptions := []mcp.PromptOption{
			mcp.WithPromptDescription(promptConfig.Description),
		}

		// Add arguments to the prompt if defined
		if len(promptConfig.Arguments) > 0 {
			for _, arg := range promptConfig.Arguments {
				description := arg.Description
				if arg.Type != settings.ArgumentTypeString {
					description = fmt.Sprintf("%s (type: %s)", description, arg.Type)
				}

				// Create argument options
				argOptions := []mcp.ArgumentOption{
					mcp.ArgumentDescription(description),
				}

				// Add required option if argument is required
				if arg.Required {
					argOptions = append(argOptions, mcp.RequiredArgument())
				}

				// Add argument as a prompt argument
				promptOptions = append(promptOptions,
					mcp.WithArgument(arg.Name, argOptions...),
				)
			}
		}

		// Create the prompt using the mcp-go library with all options
		prompt := mcp.NewPrompt(promptConfig.Name, promptOptions...)

		// Add the prompt handler
		s.mcpServer.AddPrompt(prompt, func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			// Process arguments if the prompt has them defined
			var processedArgs map[string]interface{}
			if len(promptConfig.Arguments) > 0 {
				processedArgs = make(map[string]interface{})

				// Validate and process each argument
				for _, argDef := range promptConfig.Arguments {
					var value interface{}

					// Get the value from the request arguments
					if request.Params.Arguments != nil {
						if argValue, exists := request.Params.Arguments[argDef.Name]; exists {
							// Arguments come as strings from the request, convert based on expected type
							switch argDef.Type {
							case settings.ArgumentTypeNumber:
								// Convert string to number
								if numVal, err := strconv.ParseFloat(argValue, 64); err == nil {
									value = numVal
								} else {
									value = argValue
								}
							case settings.ArgumentTypeBool:
								// Convert string to bool
								if boolVal, err := strconv.ParseBool(argValue); err == nil {
									value = boolVal
								} else {
									value = argValue
								}
							default:
								value = argValue
							}
						}
					}

					// If no value was provided, check if it's required
					if value == nil {
						if argDef.Required {
							if argDef.Default == nil {
								return nil, fmt.Errorf("required argument '%s' is missing", argDef.Name)
							}
							value = argDef.Default
						} else {
							// Use default value if available
							value = argDef.Default
						}
					}

					// Store the processed value
					if value != nil {
						processedArgs[argDef.Name] = value
					}
				}
			}

			// Create the prompt content based on configuration and arguments
			promptText := promptConfig.Content

			// Perform template substitution if arguments are provided
			if len(processedArgs) > 0 {
				// Replace argument placeholders in the content
				for key, value := range processedArgs {
					placeholder := "{" + key + "}"
					replacement := fmt.Sprintf("%v", value)
					promptText = strings.ReplaceAll(promptText, placeholder, replacement)
				}
			}

			// Create the prompt result with the configured description and processed content
			messages := []mcp.PromptMessage{
				mcp.NewPromptMessage(
					mcp.RoleUser,
					mcp.NewTextContent(promptText),
				),
			}

			return mcp.NewGetPromptResult(promptConfig.Description, messages), nil
		})

		s.logInfo("Registered MCP prompt: %s", name)
	}

	if len(s.promptConfig) > 0 {
		s.logInfo("Registered %d MCP prompts", len(s.promptConfig))
	} else {
		s.logInfo("No prompts configured for this server")
	}
}

// registerSingleCommandTool registers a single command as an MCP tool
func (s *MCPLibServer) registerSingleCommandTool(name string, cmdConfig settings.CommandConfig) {
	// Determine if this command is global (not bound to any project)
	isGlobalCommand := s.isGlobalCommand(name)

	// Create tool options
	toolOptions := []mcp.ToolOption{
		mcp.WithDescription(cmdConfig.Description),
	}

	// Add project_path parameter for global commands
	if isGlobalCommand {
		toolOptions = append(toolOptions,
			mcp.WithString("project_path", mcp.Description("Optional path to project directory where the command should be executed. If provided, the command will run in this directory context, similar to project-bound commands.")),
		)
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
		// Convert request parameters to map
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("Invalid arguments format"), nil
		}

		// For global commands, extract project_path separately (don't add to args)
		var providedProjectPath string
		if isGlobalCommand {
			if pathValue, ok := args["project_path"]; ok {
				if pathStr, ok := pathValue.(string); ok && pathStr != "" {
					providedProjectPath = pathStr
				}
			}
		}

		// Handle arguments according to how they were defined
		var processedArgs map[string]interface{}
		if len(cmdConfig.Arguments) > 0 {
			// For commands with defined arguments, extract each from the request
			processedArgs = make(map[string]interface{})
			for _, arg := range cmdConfig.Arguments {
				if value, ok := args[arg.Name]; ok {
					// Convert values based on the expected type
					switch arg.Type {
					case settings.ArgumentTypeNumber:
						// Convert to number if needed
						switch v := value.(type) {
						case string:
							if numVal, err := strconv.ParseFloat(v, 64); err == nil {
								processedArgs[arg.Name] = numVal
							} else {
								processedArgs[arg.Name] = value
							}
						case float64:
							processedArgs[arg.Name] = v
						case int:
							processedArgs[arg.Name] = float64(v)
						default:
							processedArgs[arg.Name] = value
						}
					case settings.ArgumentTypeBool:
						// Convert to bool if needed
						switch v := value.(type) {
						case string:
							if boolVal, err := strconv.ParseBool(v); err == nil {
								processedArgs[arg.Name] = boolVal
							} else {
								processedArgs[arg.Name] = value
							}
						case bool:
							processedArgs[arg.Name] = v
						default:
							processedArgs[arg.Name] = value
						}
					default:
						processedArgs[arg.Name] = value
					}
				}
			}
		} else {
			// For legacy commands, use the 'args' object if provided
			// But exclude project_path from it
			if rawArgs, ok := args["args"]; ok {
				if argsMap, ok := rawArgs.(map[string]interface{}); ok {
					processedArgs = make(map[string]interface{})
					for key, value := range argsMap {
						// Don't include project_path in args - it's handled separately
						if key != "project_path" {
							processedArgs[key] = value
						}
					}
				}
			}
		}

		// Execute the command - pass project_path separately
		result, err := s.executeCommandWithPath(name, cmdConfig.Cmd, processedArgs, providedProjectPath)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Command execution failed: %v", err)), nil
		}

		// Return the sanitized result
		return mcp.NewToolResultText(sanitizeOutput(result)), nil
	})

	s.logInfo("Registered MCP tool for command: %s", name)
}

// isGlobalCommand checks if a command is global (not bound to any project)
// A command is considered project-bound only if it's referenced in a project WITHOUT an alias
// Commands with aliases remain global, only the alias becomes project-specific
func (s *MCPLibServer) isGlobalCommand(commandName string) bool {
	cfg, err := settings.Load()
	if err != nil {
		return true // Assume global if we can't load config
	}

	// Check all projects to see if this command is bound to any without an alias
	for _, project := range cfg.Projects {
		for _, cmd := range project.Commands {
			// Only consider it project-bound if the command is referenced directly (no alias)
			if cmd.CommandName == commandName && cmd.Alias == "" {
				return false // Command is bound to a project without alias
			}
		}
	}
	return true // Command not found in any project without alias, so it's global
}

// executeCommandWithPath runs a command and returns its output, with project_path handled separately
func (s *MCPLibServer) executeCommandWithPath(name, cmdStr string, args map[string]interface{}, projectPath string) (string, error) {
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
	var projectPathUsed string

	// If project_path is provided, use it
	if projectPath != "" {
		// Expand tilde if needed
		if strings.HasPrefix(projectPath, "~/") {
			if homeDir, err := os.UserHomeDir(); err == nil {
				projectPathUsed = filepath.Join(homeDir, projectPath[2:])
			} else {
				projectPathUsed = projectPath
			}
		} else {
			projectPathUsed = projectPath
		}
		s.logInfo("Using provided project path for command %s: %s", originalName, projectPathUsed)
	} else {
		// If no project_path is provided, try to find the associated project
		cfg, err := settings.Load()
		if err == nil {
			// Look through all projects to find if this command is associated with one
			for _, project := range cfg.Projects {
				for _, cmd := range project.Commands {
					if cmd.CommandName == originalName || cmd.Alias == originalName {
						// Found the project this command belongs to
						projectPathUsed = project.Path
						s.logInfo("Found project binding for command %s: %s", originalName, projectPathUsed)
						break
					}
				}
				if projectPathUsed != "" {
					break
				}
			}
		}
	}

	// Create a slice for arguments that use prefixes
	var prefixedArgs []string
	// Create a slice for positional arguments (no prefix)
	var positionalArgs []string

	// Process arguments in the order they are defined
	for _, argDef := range cmdConfig.Arguments {
		// Get the value (using default if not provided)
		value, err := cmdConfig.GetArgumentValue(argDef.Name, args)
		if err != nil {
			return "", fmt.Errorf("error getting argument value: %w", err)
		}

		// If the value is nil (not provided and no default), skip
		if value == nil {
			continue
		}
		logging.Message("Processing argument: %s", argDef.Name)

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

		// Check if this argument has a prefix
		if argDef.Prefix != "" {
			logging.Message("Adding prefixed argument: %s %s", argDef.Prefix, valueStr)
			// Add to prefixed arguments list
			if argDef.Type == settings.ArgumentTypeBool {
				// For boolean arguments, only add the flag if true
				if valueStr == "true" {
					prefixedArgs = append(prefixedArgs, argDef.Prefix)
				}
			} else {
				// For other types, add both prefix and value
				prefixedArgs = append(prefixedArgs, fmt.Sprintf("%s %s", argDef.Prefix, valueStr))
			}
		} else {
			// For non-prefixed arguments, first try placeholder replacement
			placeholder := "${" + argDef.Name + "}"
			if strings.Contains(processedCmd, placeholder) {
				// If the command contains a placeholder, replace it
				processedCmd = strings.ReplaceAll(processedCmd, placeholder, valueStr)
				logging.Message("Replaced placeholder %s with value: %s", placeholder, valueStr)
			} else {
				// If no placeholder, treat as positional argument
				positionalArgs = append(positionalArgs, valueStr)
				logging.Message("Added positional argument: %s", valueStr)
			}
		}
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

	// Combine command parts: base command + positional args + prefixed args
	if len(positionalArgs) > 0 {
		processedCmd = fmt.Sprintf("%s %s", processedCmd, strings.Join(positionalArgs, " "))
	}
	if len(prefixedArgs) > 0 {
		processedCmd = fmt.Sprintf("%s %s", processedCmd, strings.Join(prefixedArgs, " "))
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
	if projectPathUsed != "" {
		// If project path is provided, add directory change before and after
		executeCmd = fmt.Sprintf("cd %s && %s && cd -", projectPathUsed, processedCmd)
		s.logInfo("Running command in project directory: %s", projectPathUsed)
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

// Start starts the MCP server in either stdio or SSE mode
func (s *MCPLibServer) Start() error {
	s.logInfo("Starting MCP server in %s mode", s.serverMode)

	// Ensure colors are disabled again just before starting server
	logging.DisableColors()

	if s.serverMode == "stdio" {
		// In stdio mode, just start the server directly
		return server.ServeStdio(s.mcpServer)
	}

	// In SSE mode, start the HTTP server
	if err := s.httpServer.Start(fmt.Sprintf("127.0.0.1:%d", s.port)); err != nil {
		err = fmt.Errorf("failed to start HTTP server: %w", err)
		logging.Error("%v", err)
		return err
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

	if s.serverMode == "sse" && s.httpServer != nil {
		// Gracefully shutdown the HTTP server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(ctx); err != nil {
			err = fmt.Errorf("failed to shutdown HTTP server: %w", err)
			logging.Error("%v", err)
			return err
		}
	}

	return nil
}
