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
	fmt.Println()

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

		// Print project name and path
		fmt.Printf("📁 Name: %s\n", name)
		fmt.Printf("   Path: %s\n", project.Path)

		// Print status indicators
		fmt.Printf("   Status: Valid: %s  |  In $HOME: %s\n", valid, inHomeDir)

		// Print description if exists
		if project.Description != "" {
			fmt.Printf("   Description: %s\n", project.Description)
		}

		// Add separator between projects
		fmt.Println()
	}
}

// ListWithCommands prints out all configured projects with their commands
func ListWithCommands(cfg *settings.Settings) {
	if len(cfg.Projects) == 0 {
		fmt.Println("No projects found.")
		return
	}

	fmt.Println("PROJECTS:")
	fmt.Println("=========")
	fmt.Println()

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

		// Print project name and path
		fmt.Printf("📁 Name: %s\n", name)
		fmt.Printf("   Path: %s\n", project.Path)

		// Print status indicators
		fmt.Printf("   Status: Valid: %s  |  In $HOME: %s\n", valid, inHomeDir)

		// Print description if exists
		if project.Description != "" {
			fmt.Printf("   Description: %s\n", project.Description)
		}

		// Display commands for this project
		if len(project.Commands) > 0 {
			fmt.Printf("   Commands:\n")

			for _, alias := range project.Commands {
				cmd, exists := cfg.Commands[alias.CommandName]
				if !exists {
					fmt.Printf("      ⚠️ %s (referenced command not found)\n", alias.CommandName)
					continue
				}

				// Show command name and alias if present
				if alias.Alias != "" {
					fmt.Printf("      ⚡ %s (alias: %s)\n", alias.CommandName, alias.Alias)
				} else {
					fmt.Printf("      ⚡ %s\n", alias.CommandName)
				}

				// If command has a description, display it indented
				if cmd.Description != "" {
					fmt.Printf("         %s\n", cmd.Description)
				}
			}
		}

		// Add separator between projects
		fmt.Println()
	}
}
