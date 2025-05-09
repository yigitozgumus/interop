package project

import (
	"fmt"
	"interop/internal/settings"
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
		fmt.Println("Warning: Could not determine home directory")
	}

	for name, project := range cfg.Projects {
		var fullPath string
		if filepath.IsAbs(project.Path) {
			fullPath = project.Path
		} else {
			fullPath = filepath.Join(homeDir, project.Path)
		}

		valid := "✓"
		inHomeDir := "✓"

		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			valid = "✗"
		}

		if homeDir != "" {
			if filepath.IsAbs(project.Path) {
				if !strings.HasPrefix(project.Path, homeDir) {
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
