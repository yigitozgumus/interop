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

	// Main project command
	projectCmd := &cobra.Command{
		Use:   "project",
		Short: "Project-related operations",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Project list subcommand (was "projects")
	projectListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured projects",
		Run: func(cmd *cobra.Command, args []string) {
			project.List(cfg)
		},
	}
	projectCmd.AddCommand(projectListCmd)

	// Project commands subcommand (was "project-commands")
	projectCommandsCmd := &cobra.Command{
		Use:   "commands [project-name]",
		Short: "List commands for a specific project",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			projectName := args[0]
			commands, err := settings.GetProjectCommands(cfg, projectName)
			if err != nil {
				util.Error("Failed to get commands for project '%s': %v", projectName, err)
			}

			if len(commands) == 0 {
				fmt.Printf("No commands found for project '%s'.\n", projectName)
				return
			}

			fmt.Printf("Commands for project '%s':\n", projectName)
			fmt.Println("==========================")
			fmt.Println()

			for name, cmd := range commands {
				command.PrintCommandDetails(name, cmd)
			}
		},
	}
	projectCmd.AddCommand(projectCommandsCmd)

	rootCmd.AddCommand(projectCmd)

	commandCmd := &cobra.Command{
		Use:   "command",
		Short: "Command related operations",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	commandListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured commands",
		Run: func(cmd *cobra.Command, args []string) {
			command.List(cfg.Commands)
		},
	}

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

	commandCmd.AddCommand(commandListCmd)
	rootCmd.AddCommand(commandCmd)
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
