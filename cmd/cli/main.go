package main

import (
	"fmt"
	"interop/internal/command"
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
		Use:   "run [command-or-alias]",
		Short: "Execute a command by name or alias",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			commandOrAlias := args[0]

			// Validate configuration and run the command
			err := validation.ExecuteCommand(cfg, commandOrAlias)
			if err != nil {
				logging.ErrorAndExit("Failed to run '%s': %v", commandOrAlias, err)
			}
		},
	}

	rootCmd.AddCommand(runCmd) // Add run as a top-level command for easier access

	editCmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit the configuration file with your default editor",
		Run: func(cmd *cobra.Command, args []string) {
			err := edit.OpenSettings()
			if err != nil {
				logging.ErrorAndExit(fmt.Sprintf("Failed to open settings file: %v", err))
			}
		},
	}

	rootCmd.AddCommand(editCmd)

	// Add MCP command group
	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP server",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// MCP start command
	mcpStartCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the MCP server",
		Run: func(cmd *cobra.Command, args []string) {
			if err := mcp.StartServer(); err != nil {
				logging.ErrorAndExit("Failed to start MCP server: %v", err)
			}
		},
	}
	mcpCmd.AddCommand(mcpStartCmd)

	// MCP stop command
	mcpStopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the MCP server",
		Run: func(cmd *cobra.Command, args []string) {
			if err := mcp.StopServer(); err != nil {
				logging.ErrorAndExit("Failed to stop MCP server: %v", err)
			}
		},
	}
	mcpCmd.AddCommand(mcpStopCmd)

	// MCP restart command
	mcpRestartCmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart the MCP server",
		Run: func(cmd *cobra.Command, args []string) {
			if err := mcp.RestartServer(); err != nil {
				logging.ErrorAndExit("Failed to restart MCP server: %v", err)
			}
		},
	}
	mcpCmd.AddCommand(mcpRestartCmd)

	// MCP status command
	mcpStatusCmd := &cobra.Command{
		Use:   "status",
		Short: "Get the status of the MCP server",
		Run: func(cmd *cobra.Command, args []string) {
			status, err := mcp.GetStatus()
			if err != nil {
				logging.ErrorAndExit("Failed to get MCP server status: %v", err)
			}
			fmt.Println(status)
		},
	}
	mcpCmd.AddCommand(mcpStatusCmd)

	// Hidden daemon command for internal use
	mcpDaemonCmd := &cobra.Command{
		Use:    "daemon",
		Short:  "Run the MCP HTTP server (internal use only)",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			server, err := mcp.NewServer()
			if err != nil {
				logging.ErrorAndExit("Failed to initialize MCP server: %v", err)
			}

			if err := server.RunHTTPServer(); err != nil {
				logging.ErrorAndExit("Failed to run HTTP server: %v", err)
			}
		},
	}
	mcpCmd.AddCommand(mcpDaemonCmd)

	// MCP tools command group
	mcpToolsCmd := &cobra.Command{
		Use:   "tools",
		Short: "Interact with MCP server tools",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// MCP tools health command
	mcpToolsHealthCmd := &cobra.Command{
		Use:   "health",
		Short: "Check MCP server health",
		Run: func(cmd *cobra.Command, args []string) {
			if err := mcp.GetServerHealth(); err != nil {
				logging.ErrorAndExit("Health check failed: %v", err)
			}
		},
	}
	mcpToolsCmd.AddCommand(mcpToolsHealthCmd)

	// MCP tools list command
	mcpToolsListCmd := &cobra.Command{
		Use:   "list",
		Short: "List available MCP tools",
		Run: func(cmd *cobra.Command, args []string) {
			if err := mcp.ListServerTools(); err != nil {
				logging.ErrorAndExit("Failed to list tools: %v", err)
			}
		},
	}
	mcpToolsCmd.AddCommand(mcpToolsListCmd)

	// MCP commands list command
	mcpToolsCommandsCmd := &cobra.Command{
		Use:   "commands",
		Short: "List available MCP commands",
		Run: func(cmd *cobra.Command, args []string) {
			if err := mcp.ListServerCommands(); err != nil {
				logging.ErrorAndExit("Failed to list commands: %v", err)
			}
		},
	}
	mcpToolsCmd.AddCommand(mcpToolsCommandsCmd)

	// MCP execute command
	mcpToolsExecuteCmd := &cobra.Command{
		Use:   "execute [command] [args]",
		Short: "Execute a command through MCP server",
		Long: `Execute a command through the MCP server with arguments.
Arguments can be provided in two ways:
1. Named arguments in the format key=value
2. Positional arguments in the order they are defined in the command configuration

Supported value formats:
- Strings: value or key=value
- Numbers: 123 or key=123 (or 12.34 or key=12.34)
- Booleans: true/false or key=true/key=false

Examples:
  interop mcp tools execute build-app
  interop mcp tools execute build-app output.exe
  interop mcp tools execute build-app output.exe ./src/main.go
  interop mcp tools execute build-app version=1.0.0
  interop mcp tools execute deploy-app environment=production force=true`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			commandName := args[0]

			// Get command configuration to determine defined arguments
			cfg, err := settings.Load()
			if err != nil {
				logging.ErrorAndExit("Failed to load settings: %v", err)
			}

			// Get the command config to check for defined arguments
			cmdConfig, exists := cfg.Commands[commandName]
			if !exists {
				logging.ErrorAndExit("Command '%s' not found", commandName)
			}

			// Parse arguments
			cmdArgs := make(map[string]interface{})

			// Track which positional index we're at
			positionalIndex := 0

			// Process each provided argument
			for i := 1; i < len(args); i++ {
				// Check if it's a named argument (key=value format)
				if strings.Contains(args[i], "=") {
					parts := strings.SplitN(args[i], "=", 2)
					if len(parts) == 2 {
						key := parts[0]
						rawValue := parts[1]

						// Process the value with type detection
						parsedValue := parseArgumentValue(rawValue)
						cmdArgs[key] = parsedValue
					}
				} else {
					// It's a positional argument
					// Check if we have a corresponding argument definition
					if positionalIndex < len(cmdConfig.Arguments) {
						argDef := cmdConfig.Arguments[positionalIndex]

						// Process the value according to the defined type
						parsedValue := parseArgumentValueWithType(args[i], argDef.Type)
						cmdArgs[argDef.Name] = parsedValue

						positionalIndex++
					} else {
						// More positional arguments than defined - warn user
						logging.Warning("Ignoring extra positional argument: %s", args[i])
					}
				}
			}

			if err := mcp.ExecuteServerCommand(commandName, cmdArgs); err != nil {
				logging.ErrorAndExit("Failed to execute command: %v", err)
			}
		},
	}
	mcpToolsCmd.AddCommand(mcpToolsExecuteCmd)

	// MCP events command
	mcpToolsEventsCmd := &cobra.Command{
		Use:   "events",
		Short: "Stream real-time events from the MCP server",
		Run: func(cmd *cobra.Command, args []string) {
			if err := mcp.StreamServerEvents(); err != nil {
				logging.ErrorAndExit("Failed to stream events: %v", err)
			}
		},
	}
	mcpToolsCmd.AddCommand(mcpToolsEventsCmd)

	// Add tools command to MCP command group
	mcpCmd.AddCommand(mcpToolsCmd)

	// Add MCP command group to root command
	rootCmd.AddCommand(mcpCmd)

	// Add validation command to check configuration
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate the configuration file",
		Run: func(cmd *cobra.Command, args []string) {
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
				fmt.Println("✅ Configuration is valid!")
				return
			}

			fmt.Println("⚠️ Configuration validation issues:")
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
