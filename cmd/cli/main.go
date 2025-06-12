package main

import (
	"fmt"
	"interop/internal/command"
	"interop/internal/display"
	"interop/internal/edit"
	"interop/internal/logging"
	"interop/internal/mcp"
	projectPkg "interop/internal/project"
	"interop/internal/remote"
	"interop/internal/settings"
	"interop/internal/validation"
	"interop/internal/validation/project"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var (
	version    = "dev"
	isSnapshot = "false"
)

func main() {
	cfg, err := settings.Load()
	if err != nil {
		log.Fatalf("settings init: %v", err)
	}
	logging.Message("Config is loaded")

	rootCmd := &cobra.Command{
		Use:     "interop",
		Short:   "Interop - Project management CLI",
		Version: getVersionInfo(),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Projects command that shows all projects and their commands
	projectsCmd := &cobra.Command{
		Use:   "projects",
		Short: "List all configured projects with their commands",
		Run: func(cmd *cobra.Command, args []string) {
			projectPkg.ListWithCommands(cfg)
		},
	}
	rootCmd.AddCommand(projectsCmd)

	// Commands command that lists all commands
	commandsCmd := &cobra.Command{
		Use:   "commands",
		Short: "List all configured commands",
		Run: func(cmd *cobra.Command, args []string) {
			// Convert cfg.Commands to the expected format for command.ListWithProjects
			commands := make(map[string]command.Command)
			for name, cmdCfg := range cfg.Commands {
				commands[name] = command.Command{
					Description:  cmdCfg.Description,
					IsEnabled:    cmdCfg.IsEnabled,
					Cmd:          cmdCfg.Cmd,
					IsExecutable: cmdCfg.IsExecutable,
				}
			}

			// Convert project commands to the format expected by ListWithProjects
			projectCommands := make(map[string][]command.Alias)
			for projectName, project := range cfg.Projects {
				aliases := make([]command.Alias, len(project.Commands))
				for i, a := range project.Commands {
					aliases[i] = command.Alias{
						CommandName: a.CommandName,
						Alias:       a.Alias,
					}
				}
				projectCommands[projectName] = aliases
			}

			command.ListWithProjects(commands, projectCommands)
		},
	}
	rootCmd.AddCommand(commandsCmd)

	// New run command that supports both command names and aliases
	runCmd := &cobra.Command{
		Use:   "run [command-or-alias] [args...]",
		Short: "Execute a command by name or alias with optional arguments",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			commandOrAlias := args[0]
			commandArgs := args[1:]

			// Validate configuration and run the command with arguments
			err := validation.ExecuteCommandWithArgs(cfg, commandOrAlias, commandArgs)
			if err != nil {
				logging.ErrorAndExit("Failed to run '%s': %v", commandOrAlias, err)
			}
		},
	}

	rootCmd.AddCommand(runCmd) // Add run as a top-level command for easier access

	// Add Config command group
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration settings",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Define flag variable for the editor
	var editorName string

	// Config edit command (moved from root level)
	configEditCmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit the configuration file with your default editor or specified editor",
		Long:  "Edit the configuration file using the editor specified by --editor flag, $EDITOR environment variable, or nano as fallback",
		Run: func(cmd *cobra.Command, args []string) {
			err := edit.OpenSettings(editorName)
			if err != nil {
				logging.ErrorAndExit(fmt.Sprintf("Failed to open settings file: %v", err))
			}
		},
	}

	// Add the --editor flag to the config edit command
	configEditCmd.Flags().StringVar(&editorName, "editor", "", "Editor to use for opening the configuration file (e.g., code, vim, nano)")
	configCmd.AddCommand(configEditCmd)

	// Add Remote command group under config
	remoteCmd := &cobra.Command{
		Use:   "remote",
		Short: "Manage remote configuration",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Remote add command
	remoteAddCmd := &cobra.Command{
		Use:   "add [url]",
		Short: "Add a remote URL to configuration",
		Long:  "Add a remote URL that will be used for managing multiple config files and executables",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				logging.ErrorAndExit("Remote URL is required. Usage: interop config remote add <url>")
			}

			url := args[0]
			if url == "" {
				logging.ErrorAndExit("Remote URL cannot be empty")
			}

			remoteMgr := remote.NewManager()
			if err := remoteMgr.Add(url); err != nil {
				logging.ErrorAndExit("Failed to add remote URL: %v", err)
			}

			fmt.Printf("Successfully added remote URL: %s\n", url)
		},
	}
	remoteCmd.AddCommand(remoteAddCmd)

	// Remote remove command
	remoteRemoveCmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove the current remote URL",
		Long:  "Remove the currently configured remote URL from the configuration",
		Run: func(cmd *cobra.Command, args []string) {
			remoteMgr := remote.NewManager()
			if err := remoteMgr.Remove(); err != nil {
				logging.ErrorAndExit("Failed to remove remote URL: %v", err)
			}

			fmt.Println("Successfully removed remote URL")
		},
	}
	remoteCmd.AddCommand(remoteRemoveCmd)

	// Remote show command
	remoteShowCmd := &cobra.Command{
		Use:   "show",
		Short: "Show the current remote URL",
		Long:  "Display the currently configured remote URL or notify if not set",
		Run: func(cmd *cobra.Command, args []string) {
			remoteMgr := remote.NewManager()
			if err := remoteMgr.Show(); err != nil {
				logging.ErrorAndExit("Failed to show remote URL: %v", err)
			}
		},
	}
	remoteCmd.AddCommand(remoteShowCmd)

	// Remote fetch command (placeholder)
	remoteFetchCmd := &cobra.Command{
		Use:   "fetch",
		Short: "Fetch configuration from remote",
		Long:  "Fetch configuration files and executables from the configured remote Git repository. This will clone the repository, validate its structure, and sync files to local remote directories.",
		Run: func(cmd *cobra.Command, args []string) {
			remoteMgr := remote.NewManager()
			if err := remoteMgr.Fetch(); err != nil {
				logging.ErrorAndExit("Failed to fetch from remote: %v", err)
			}
		},
	}
	remoteCmd.AddCommand(remoteFetchCmd)

	// Remote clear command
	remoteClearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear all remote configuration files and reset tracking",
		Long:  "Remove all files from config.d.remote and executables.remote directories and reset the version tracking information. This provides a clean slate for remote configuration.",
		Run: func(cmd *cobra.Command, args []string) {
			remoteMgr := remote.NewManager()
			if err := remoteMgr.Clear(); err != nil {
				logging.ErrorAndExit("Failed to clear remote configuration: %v", err)
			}

			fmt.Println("Successfully cleared all remote configuration files and tracking information")
		},
	}
	remoteCmd.AddCommand(remoteClearCmd)

	// Add remote command to config command
	configCmd.AddCommand(remoteCmd)

	// Add config command group to root command
	rootCmd.AddCommand(configCmd)

	// Add MCP command group
	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP server",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Define flags for MCP commands
	var startAllServers bool
	var stopAllServers bool
	var restartAllServers bool
	var statusAllServers bool
	var serverName string
	var serverMode string

	// MCP start command
	mcpStartCmd := &cobra.Command{
		Use:   "start [server-name]",
		Short: "Start an MCP server or all servers",
		Long: `Start MCP servers in either SSE (HTTP) or stdio mode:

SSE Mode (default):
  - Runs as a daemon process in the background  
  - Communicates via HTTP on configured ports
  - Supports multiple named servers
  - Use --all flag to start all configured servers

Stdio Mode:
  - Runs in foreground and communicates via stdin/stdout
  - Used by MCP clients that spawn the server process directly
  - Supports both default and named servers
  - Does not support --all flag (single server only)
  - No HTTP ports are used
  
Examples:
  interop mcp start                    # Start all servers in SSE mode
  interop mcp start --mode stdio       # Start default server in stdio mode
  interop mcp start myserver --mode stdio # Start named server in stdio mode
  interop mcp start myserver --mode sse # Start named server in SSE mode`,
		Run: func(cmd *cobra.Command, args []string) {
			// Check for stdio mode first
			if serverMode == "stdio" && startAllServers {
				logging.ErrorAndExit("--all flag is not supported in stdio mode")
			}

			// If server name is provided as an argument, override the flag
			if len(args) > 0 {
				serverName = args[0]
				startAllServers = false // Single server specified, turn off all flag
			} else if serverName != "" {
				startAllServers = false // Single server specified, turn off all flag
			}

			// Set server mode in environment
			if serverMode != "" {
				os.Setenv("MCP_SERVER_MODE", serverMode)
			}

			// For SSE mode, default to all servers if no specific server is specified
			if serverMode != "stdio" && !startAllServers && serverName == "" {
				startAllServers = true
			}

			if err := mcp.StartServer(serverName, startAllServers); err != nil {
				logging.ErrorAndExit("Failed to start MCP server: %v", err)
			}
		},
	}
	mcpStartCmd.Flags().BoolVarP(&startAllServers, "all", "a", false, "Start all MCP servers (default, not supported in stdio mode)")
	mcpStartCmd.Flags().StringVarP(&serverName, "server", "s", "", "Specific MCP server to start")
	mcpStartCmd.Flags().StringVar(&serverMode, "mode", "sse", "Server mode (stdio or sse)")
	mcpCmd.AddCommand(mcpStartCmd)

	// MCP stop command
	mcpStopCmd := &cobra.Command{
		Use:   "stop [server-name]",
		Short: "Stop an MCP server or all servers",
		Long:  "Stop the default MCP server, a specific named server, or all servers",
		Run: func(cmd *cobra.Command, args []string) {
			// If server name is provided as an argument, override the flag
			if len(args) > 0 {
				serverName = args[0]
			}

			// Set server mode in environment
			if serverMode != "" {
				os.Setenv("MCP_SERVER_MODE", serverMode)
			}

			// In stdio mode, --all flag is not supported
			if serverMode == "stdio" && stopAllServers {
				logging.ErrorAndExit("--all flag is not supported in stdio mode")
			}

			if err := mcp.StopServer(serverName, stopAllServers); err != nil {
				logging.ErrorAndExit("Failed to stop MCP server: %v", err)
			}
		},
	}
	mcpStopCmd.Flags().BoolVarP(&stopAllServers, "all", "a", false, "Stop all MCP servers (not supported in stdio mode)")
	mcpStopCmd.Flags().StringVarP(&serverName, "server", "s", "", "Specific MCP server to stop")
	mcpStopCmd.Flags().StringVar(&serverMode, "mode", "sse", "Server mode (stdio or sse)")
	mcpCmd.AddCommand(mcpStopCmd)

	// MCP restart command
	mcpRestartCmd := &cobra.Command{
		Use:   "restart [server-name]",
		Short: "Restart an MCP server or all servers",
		Long:  "Restart the default MCP server, a specific named server, or all servers",
		Run: func(cmd *cobra.Command, args []string) {
			// If server name is provided as an argument, override the flag
			if len(args) > 0 {
				serverName = args[0]
			}

			// Set server mode in environment
			if serverMode != "" {
				os.Setenv("MCP_SERVER_MODE", serverMode)
			}

			// In stdio mode, --all flag is not supported
			if serverMode == "stdio" && restartAllServers {
				logging.ErrorAndExit("--all flag is not supported in stdio mode")
			}

			if err := mcp.RestartServer(serverName, restartAllServers); err != nil {
				logging.ErrorAndExit("Failed to restart MCP server: %v", err)
			}
		},
	}
	mcpRestartCmd.Flags().BoolVarP(&restartAllServers, "all", "a", false, "Restart all MCP servers (not supported in stdio mode)")
	mcpRestartCmd.Flags().StringVarP(&serverName, "server", "s", "", "Specific MCP server to restart")
	mcpRestartCmd.Flags().StringVar(&serverMode, "mode", "sse", "Server mode (stdio or sse)")
	mcpCmd.AddCommand(mcpRestartCmd)

	// MCP status command
	mcpStatusCmd := &cobra.Command{
		Use:   "status [server-name]",
		Short: "Get the status of an MCP server or all servers",
		Long:  "Get the status of all MCP servers by default, or a specific named server if provided",
		Run: func(cmd *cobra.Command, args []string) {
			// If server name is provided as an argument, override the flag
			if len(args) > 0 {
				serverName = args[0]
			}

			status, err := mcp.GetStatus(serverName, statusAllServers)
			if err != nil {
				logging.ErrorAndExit("Failed to get MCP server status: %v", err)
			}
			fmt.Println(status)
		},
	}
	mcpStatusCmd.Flags().BoolVarP(&statusAllServers, "all", "a", true, "Get status of all MCP servers (default)")
	mcpStatusCmd.Flags().StringVarP(&serverName, "server", "s", "", "Specific MCP server to get status for")
	mcpCmd.AddCommand(mcpStatusCmd)

	// MCP list command
	mcpListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured MCP servers and their commands",
		Run: func(cmd *cobra.Command, args []string) {
			result, err := mcp.ListMCPServers()
			if err != nil {
				logging.ErrorAndExit("Failed to list MCP servers: %v", err)
			}
			fmt.Println(result)
		},
	}
	mcpCmd.AddCommand(mcpListCmd)

	// MCP export command
	mcpExportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export MCP server configuration as JSON",
		Long: `Export MCP server configuration as JSON for use with MCP clients.

Modes:
  sse (default): Export HTTP URLs for SSE-based communication
  stdio: Export command-line configurations for stdio-based communication
  
Examples:
  interop mcp export                  # Export SSE configuration (HTTP URLs)
  interop mcp export --mode sse       # Export SSE configuration (HTTP URLs)  
  interop mcp export --mode stdio     # Export stdio configuration (command lines)`,
		Run: func(cmd *cobra.Command, args []string) {
			// Get the mode flag value, default to "sse"
			mode, _ := cmd.Flags().GetString("mode")
			if mode == "" {
				mode = "sse"
			}

			var result string
			var err error

			if mode == "stdio" || mode == "sse" {
				result, err = mcp.ExportMCPConfigWithMode(mode)
			} else {
				logging.ErrorAndExit("Invalid mode '%s'. Must be either 'stdio' or 'sse'", mode)
			}

			if err != nil {
				logging.ErrorAndExit("Failed to export MCP configuration: %v", err)
			}
			fmt.Println(result)
		},
	}
	mcpExportCmd.Flags().String("mode", "sse", "Export mode (stdio or sse)")
	mcpCmd.AddCommand(mcpExportCmd)

	// MCP prompts command
	mcpPromptsCmd := &cobra.Command{
		Use:   "prompts",
		Short: "List all configured prompts",
		Run: func(cmd *cobra.Command, args []string) {
			// Load configuration to get prompts
			cfg, err := settings.Load()
			if err != nil {
				logging.ErrorAndExit("Failed to load settings: %v", err)
			}

			if len(cfg.Prompts) == 0 {
				fmt.Println("No prompts configured.")
				return
			}

			fmt.Println("Configured Prompts:")
			fmt.Println("==================")
			fmt.Println()

			for name, prompt := range cfg.Prompts {
				fmt.Printf("Name: %s\n", name)
				fmt.Printf("Description: %s\n", prompt.Description)

				if prompt.MCP != "" {
					fmt.Printf("MCP Server: %s\n", prompt.MCP)
				} else {
					fmt.Printf("MCP Server: default\n")
				}

				fmt.Printf("Content:\n%s\n", prompt.Content)

				if len(prompt.Arguments) > 0 {
					fmt.Printf("Arguments:\n")
					for _, arg := range prompt.Arguments {
						typeStr := string(arg.Type)
						if typeStr == "" {
							typeStr = "string"
						}

						requiredStr := ""
						if arg.Required {
							requiredStr = " (required)"
						}

						defaultStr := ""
						if arg.Default != nil {
							defaultStr = fmt.Sprintf(" [default: %v]", arg.Default)
						}

						fmt.Printf("  - %s (%s): %s%s%s\n",
							arg.Name, typeStr, arg.Description, requiredStr, defaultStr)
					}
				}

				fmt.Println()
			}
		},
	}
	mcpCmd.AddCommand(mcpPromptsCmd)

	// Hidden daemon command for internal use
	mcpDaemonCmd := &cobra.Command{
		Use:    "daemon",
		Short:  "Run the MCP HTTP server (internal use only)",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := mcp.RunHTTPServer(); err != nil {
				logging.ErrorAndExit("Failed to run HTTP server: %v", err)
			}
		},
	}
	mcpCmd.AddCommand(mcpDaemonCmd)

	// MCP events command
	mcpToolsEventsCmd := &cobra.Command{
		Use:   "events [server-name]",
		Short: "Stream real-time events from an MCP server",
		Long:  "Stream real-time events from the default MCP server or a specific named server",
		Run: func(cmd *cobra.Command, args []string) {
			// If server name is provided as an argument, override the flag
			if len(args) > 0 {
				serverName = args[0]
			}

			if err := mcp.StreamServerEvents(serverName); err != nil {
				logging.ErrorAndExit("Failed to stream events: %v", err)
			}
		},
	}
	mcpToolsEventsCmd.Flags().StringVarP(&serverName, "server", "s", "", "Specific MCP server to stream events from")
	mcpCmd.AddCommand(mcpToolsEventsCmd)

	// MCP port-check command
	mcpPortCheckCmd := &cobra.Command{
		Use:   "port-check",
		Short: "Check if MCP server ports are available",
		Long:  "Check if the configured MCP server ports are available or in use by other processes",
		Run: func(cmd *cobra.Command, args []string) {
			result, err := mcp.CheckPortAvailability()
			if err != nil {
				logging.ErrorAndExit("Failed to check port availability: %v", err)
			}
			fmt.Println(result)
		},
	}
	mcpCmd.AddCommand(mcpPortCheckCmd)

	// Add MCP command group to root command
	rootCmd.AddCommand(mcpCmd)

	// Add validation command to check configuration
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate the configuration file",
		Run: func(cmd *cobra.Command, args []string) {
			// Show command graph visualization first
			display.PrintCommandGraph(cfg)

			// Validate commands using existing functionality
			cmdErrors := validation.ValidateCommands(cfg)

			// Validate projects using the new project validator
			projectValidator := project.NewValidator(cfg)
			projectResult := projectValidator.ValidateAll()

			// Combine errors from both validations
			allErrors := cmdErrors
			for _, err := range projectResult.Errors {
				// Skip project errors that are already reported by command validation
				isDuplicate := false
				for _, cmdErr := range cmdErrors {
					if cmdErr.Message == err.Error() {
						isDuplicate = true
						break
					}
				}

				if !isDuplicate {
					allErrors = append(allErrors, validation.ValidationError{
						Message: err.Error(),
						Severe:  err.Severe,
					})
				}
			}

			if len(allErrors) == 0 {
				fmt.Println("\n✅ Configuration is valid!")
				return
			}

			fmt.Println("\n⚠️ Configuration validation issues:")
			fmt.Println("==================================")
			fmt.Println()

			severe := false
			for _, err := range allErrors {
				severity := "Warning"
				if err.Severe {
					severity = "Error"
					severe = true
				}
				fmt.Printf("[%s] %s\n", severity, err.Message)
			}

			if severe {
				os.Exit(1)
			}
		},
	}

	rootCmd.AddCommand(validateCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getVersionInfo() string {
	versionInfo := version
	if isSnapshot == "true" {
		versionInfo += " (snapshot)"
	}
	return versionInfo
}

// Helper function to parse argument values with type detection
func parseArgumentValue(rawValue string) interface{} {
	// Try to detect boolean values
	if strings.EqualFold(rawValue, "true") {
		return true
	}
	if strings.EqualFold(rawValue, "false") {
		return false
	}

	// Try to detect numeric values
	if intVal, err := strconv.ParseInt(rawValue, 10, 64); err == nil {
		return intVal
	}
	if floatVal, err := strconv.ParseFloat(rawValue, 64); err == nil {
		return floatVal
	}

	// Default to string value
	return rawValue
}

// Helper function to parse argument values with a specified type
func parseArgumentValueWithType(rawValue string, argType settings.ArgumentType) interface{} {
	switch argType {
	case settings.ArgumentTypeBool:
		if strings.EqualFold(rawValue, "true") {
			return true
		}
		if strings.EqualFold(rawValue, "false") {
			return false
		}
		// If it doesn't match true/false, keep it as string
		return rawValue
	case settings.ArgumentTypeNumber:
		if intVal, err := strconv.ParseInt(rawValue, 10, 64); err == nil {
			return intVal
		}
		if floatVal, err := strconv.ParseFloat(rawValue, 64); err == nil {
			return floatVal
		}
		// If it's not a valid number, keep it as string
		return rawValue
	default: // String or any other type
		return rawValue
	}
}
