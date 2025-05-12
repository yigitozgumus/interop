package display

import (
	"fmt"
	"strings"
)

// PrintProjectHeader prints the project header
func PrintProjectHeader() {
	fmt.Println("PROJECTS:")
	fmt.Println("=========")
	fmt.Println()
}

// PrintCommandHeader prints the command header
func PrintCommandHeader() {
	fmt.Println("COMMANDS:")
	fmt.Println("=========")
	fmt.Println()
}

// PrintProjectName prints a project name with an icon
func PrintProjectName(name string) {
	fmt.Printf("📁 Name: %s\n", name)
}

// PrintProjectPath prints a project path
func PrintProjectPath(path string) {
	fmt.Printf("   Path: %s\n", path)
}

// PrintProjectStatus prints the project status
func PrintProjectStatus(valid, inHomeDir string) {
	fmt.Printf("   Status: Valid: %s  |  In $HOME: %s\n", valid, inHomeDir)
}

// PrintProjectDescription prints a project description if present
func PrintProjectDescription(description string) {
	if description != "" {
		fmt.Printf("   Description: %s\n", description)
	}
}

// PrintCommandName prints a command name with an icon
func PrintCommandName(name string) {
	fmt.Printf("⚡ Name: %s\n", name)
}

// PrintCommandStatus prints the command status
func PrintCommandStatus(isEnabled bool, execSource string) {
	statusEnabled := "✓"
	if !isEnabled {
		statusEnabled = "✗"
	}
	fmt.Printf("   Status: Enabled: %s  |  Source: %s\n", statusEnabled, execSource)
}

// PrintCommandProjects prints the projects associated with a command
func PrintCommandProjects(projectNames []string) {
	if len(projectNames) > 0 {
		if len(projectNames) == 1 {
			fmt.Printf("   Project: %s\n", projectNames[0])
		} else {
			fmt.Printf("   Projects: %s\n", strings.Join(projectNames, ", "))
		}
	}
}

// PrintCommandDescription prints a command description if present
func PrintCommandDescription(description string) {
	if description != "" {
		fmt.Printf("   Description: %s\n", description)
	}
}

// PrintProjectCommands prints the commands for a project
func PrintProjectCommands(commandName, alias, description string) {
	if alias != "" {
		fmt.Printf("      ⚡ %s (alias: %s)\n", commandName, alias)
	} else {
		fmt.Printf("      ⚡ %s\n", commandName)
	}

	if description != "" {
		fmt.Printf("         %s\n", description)
	}
}

// PrintSeparator prints a blank line as a separator
func PrintSeparator() {
	fmt.Println()
}

// PrintUnresolvedCommand prints a warning about an unresolved command
func PrintUnresolvedCommand(commandName string) {
	fmt.Printf("      ⚠️ %s (referenced command not found)\n", commandName)
}

// PrintNoItemsFound prints a message when no items are found
func PrintNoItemsFound(itemType string) {
	fmt.Printf("No %s found.\n", itemType)
}
