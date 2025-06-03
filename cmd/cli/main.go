package main

import (
	"fmt"
	"interop/internal/command"
	"interop/internal/display"
	"interop/internal/edit"
	"interop/internal/logging"
	"interop/internal/mcp"
	projectPkg "interop/internal/project"
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

	// Define flag variable for the editor
	var editorName string

	editCmd := &cobra.Command{
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

	// Add the --editor flag to the edit command
	editCmd.Flags().StringVar(&editorName, "editor", "", "Editor to use for opening the configuration file (e.g., code, vim, nano)")

	rootCmd.AddCommand(editCmd)

	// Add MCP command group
	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP server",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Define flags for MCP commands
	var allServers bool
	var serverName string

	// MCP start command
	mcpStartCmd := &cobra.Command{
		Use:   "start [server-name]",
		Short: "Start an MCP server or all servers",
		Long:  "Start all MCP servers by default, or a specific named server if provided",
		Run: func(cmd *cobra.Command, args []string) {
			// If server name is provided as an argument, override the flag
			if len(args) > 0 {
				serverName = args[0]
				allServers = false // Single server specified, turn off all flag
			} else if serverName != "" {
				allServers = false // Single server specified, turn off all flag
			} else if !allServers {
				// No server name provided and all flag not set, default to all servers
				allServers = true
			}

			if err := mcp.StartServer(serverName, allServers); err != nil {
				logging.ErrorAndExit("Failed to start MCP server: %v", err)
			}
		},
	}
	mcpStartCmd.Flags().BoolVarP(&allServers, "all", "a", true, "Start all MCP servers (default)")
	mcpStartCmd.Flags().StringVarP(&serverName, "server", "s", "", "Specific MCP server to start")
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

			if err := mcp.StopServer(serverName, allServers); err != nil {
				logging.ErrorAndExit("Failed to stop MCP server: %v", err)
			}
		},
	}
	mcpStopCmd.Flags().BoolVarP(&allServers, "all", "a", false, "Stop all MCP servers")
	mcpStopCmd.Flags().StringVarP(&serverName, "server", "s", "", "Specific MCP server to stop")
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

			if err := mcp.RestartServer(serverName, allServers); err != nil {
				logging.ErrorAndExit("Failed to restart MCP server: %v", err)
			}
		},
	}
	mcpRestartCmd.Flags().BoolVarP(&allServers, "all", "a", false, "Restart all MCP servers")
	mcpRestartCmd.Flags().StringVarP(&serverName, "server", "s", "", "Specific MCP server to restart")
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

			status, err := mcp.GetStatus(serverName, allServers)
			if err != nil {
				logging.ErrorAndExit("Failed to get MCP server status: %v", err)
			}
			fmt.Println(status)
		},
	}
	mcpStatusCmd.Flags().BoolVarP(&allServers, "all", "a", true, "Get status of all MCP servers (default)")
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
		Run: func(cmd *cobra.Command, args []string) {
			result, err := mcp.ExportMCPConfig()
			if err != nil {
				logging.ErrorAndExit("Failed to export MCP configuration: %v", err)
			}
			fmt.Println(result)
		},
	}
	mcpCmd.AddCommand(mcpExportCmd)

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
