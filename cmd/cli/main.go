package main

import (
	"fmt"
	"interop/internal/command"
	"interop/internal/edit"
	"interop/internal/project"
	"interop/internal/settings"
	"interop/internal/util"
	"interop/internal/validation"
	"log"
	"os"

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
	util.Message("Config is loaded")

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
			project.ListWithCommands(cfg)
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
				util.Error("Failed to run '%s': %v", commandOrAlias, err)
				os.Exit(1)
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
				util.Error(fmt.Sprintf("Failed to open settings file: %v", err))
				os.Exit(1)
			}
		},
	}

	rootCmd.AddCommand(editCmd)

	// Add validation command to check configuration
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate the configuration file",
		Run: func(cmd *cobra.Command, args []string) {
			errors := validation.ValidateCommands(cfg)
			if len(errors) == 0 {
				fmt.Println("✅ Configuration is valid!")
				return
			}

			fmt.Println("⚠️ Configuration validation issues:")
			fmt.Println("==================================")
			fmt.Println()

			severe := false
			for _, err := range errors {
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
