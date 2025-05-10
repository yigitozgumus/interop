package project

import (
	"fmt"
	"interop/internal/settings"
	"interop/internal/util"
	"os"
	"path/filepath"
	"strings"
)

// List prints out all configured projects with their name, path, and validity
func List(cfg *settings.Settings) {
	if len(cfg.Projects) == 0 {
		fmt.Println("No projects found.")
		return
	}

	fmt.Println("PROJECTS:")
	fmt.Println("=========")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
		util.Warning("Warning: Could not determine home directory")
	}

	for name, project := range cfg.Projects {
		path := project.Path

		// Handle tilde expansion for home directory
		if strings.HasPrefix(path, "~/") && homeDir != "" {
			path = filepath.Join(homeDir, path[2:])
		}

		var fullPath string
		if filepath.IsAbs(path) {
			fullPath = path
		} else {
			fullPath = filepath.Join(homeDir, path)
		}

		valid := "✓"
		inHomeDir := "✓"

		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			valid = "✗"
		}

		if homeDir != "" {
			if filepath.IsAbs(path) {
				if !strings.HasPrefix(path, homeDir) {
					inHomeDir = "✗"
				}
			}
		}

		fmt.Printf("%s: %s [Valid: %s] [In $HOME: %s]\n", name, project.Path, valid, inHomeDir)
		if project.Description != "" {
			fmt.Printf("  Description: %s\n", project.Description)
		}
	}
}
