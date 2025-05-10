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
	"path/filepath"
	"strings"
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

	// Add a single 'run' command for projects
	projectRunCmd := &cobra.Command{
		Use:   "run [project-name] [command-name]",
		Short: "Run a command for a specific project",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			projectName := args[0]
			commandName := args[1]

			// Verify project exists
			project, exists := cfg.Projects[projectName]
			if !exists {
				util.Error("Project '%s' not found", projectName)
			}

			// Verify command is associated with project
			found := false
			for _, cmdName := range project.Commands {
				if cmdName == commandName {
					found = true
					break
				}
			}

			if !found {
				util.Error("Command '%s' is not associated with project '%s'", commandName, projectName)
			}

			// Check if command exists in config
			if _, exists := cfg.Commands[commandName]; !exists {
				util.Error("Command '%s' not found in configuration", commandName)
			}

			// Get executables path and run the command
			executablesPath, err := settings.GetExecutablesPath()
			if err != nil {
				util.Error("Failed to get executables path: %v", err)
			}

			// Get project path for running commands in the correct directory
			projectPath := project.Path

			// Handle tilde expansion for home directory
			if strings.HasPrefix(projectPath, "~/") {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					util.Error("Failed to get user home directory: %v", err)
				}
				projectPath = filepath.Join(homeDir, projectPath[2:])
			} else if !filepath.IsAbs(projectPath) {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					util.Error("Failed to get user home directory: %v", err)
				}
				projectPath = filepath.Join(homeDir, projectPath)
			}

			// Run the command in the project directory
			err = command.Run(cfg.Commands, commandName, executablesPath, projectPath)
			if err != nil {
				util.Error("Failed to run command '%s' for project '%s': %v", commandName, projectName, err)
			}
		},
	}
	projectCmd.AddCommand(projectRunCmd)

	// Create project-specific subcommands for each project
	for projectName := range cfg.Projects {
		// Create a project-specific command
		projectSpecificCmd := &cobra.Command{
			Use:   projectName,
			Short: fmt.Sprintf("Operations for project '%s'", projectName),
			Run: func(cmd *cobra.Command, args []string) {
				cmd.Help()
			},
		}

		projectCmd.AddCommand(projectSpecificCmd)
	}

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
