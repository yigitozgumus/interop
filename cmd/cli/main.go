package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"interop/internal/command"
	"interop/internal/edit"
	"interop/internal/project"
	"interop/internal/settings"
	"interop/internal/util"
	"log"
	"os"
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

	commandRunCmd := &cobra.Command{
		Use:   "run [command-name]",
		Short: "Execute a configured command",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			commandName := args[0]

			// Get the executables path
			executablesPath, err := settings.GetExecutablesPath()
			if err != nil {
				util.Error("Failed to get executables path: %v", err)
			}

			err = command.Run(cfg.Commands, commandName, executablesPath)
			if err != nil {
				util.Error("Failed to run command '%s': %v", commandName, err)
			}
		},
	}

	commandCmd.AddCommand(commandListCmd)
	commandCmd.AddCommand(commandRunCmd)
	rootCmd.AddCommand(commandCmd)

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
