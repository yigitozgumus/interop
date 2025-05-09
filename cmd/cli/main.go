package main

import (
	"fmt"
	"github.com/spf13/cobra"
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
