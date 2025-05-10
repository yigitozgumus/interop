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

	projectsCmd := &cobra.Command{
		Use:   "projects",
		Short: "List all configured projects",
		Run: func(cmd *cobra.Command, args []string) {
			project.List(cfg)
		},
	}

	rootCmd.AddCommand(projectsCmd)

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
