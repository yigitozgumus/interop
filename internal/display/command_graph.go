package display

import (
	"fmt"
	"interop/internal/settings"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Command relationship types
const (
	GlobalCommandSymbol    = "🌐"
	ProjectCommandSymbol   = "📂"
	ProjectAliasSymbol     = "🔄"
	CommandDisabledSymbol  = "❌"
	CommandEnabledSymbol   = "✓"
	MCPServerSymbol        = "🔌"
	ExecutableCommandLabel = "(Executable)"
	ShellCommandLabel      = "(Shell)"
	RemoteSymbol           = "☁️"
	LocalSymbol            = "🏠"
	ConflictSymbol         = "⚠️"
)

// PrintCommandGraph displays a visual graph of commands and their relationships
func PrintCommandGraph(cfg *settings.Settings) {
	fmt.Println("Configuration Overview")
	fmt.Println("=====================")

	// Show configuration loading information
	printConfigurationSources(cfg)

	// Track which commands are associated with projects by name (no alias)
	projectBoundCommands := make(map[string][]string) // command -> []projectNames

	// Track which commands are used with aliases
	aliasedCommands := make(map[string]map[string]string) // command -> map[alias]projectName

	// Build the relationship maps
	for projectName, project := range cfg.Projects {
		for _, cmdAlias := range project.Commands {
			// Handle commands bound directly (no alias)
			if cmdAlias.Alias == "" {
				projectBoundCommands[cmdAlias.CommandName] = append(
					projectBoundCommands[cmdAlias.CommandName],
					projectName,
				)
			} else {
				// Handle aliased commands
				if _, exists := aliasedCommands[cmdAlias.CommandName]; !exists {
					aliasedCommands[cmdAlias.CommandName] = make(map[string]string)
				}
				aliasedCommands[cmdAlias.CommandName][cmdAlias.Alias] = projectName
			}
		}
	}

	// Print MCP server configuration
	printMCPServers(cfg)

	// Print the command graph with source information
	printCommands(cfg, projectBoundCommands, aliasedCommands)

	// Print legend
	printLegend()
}

// printConfigurationSources shows information about where configurations are loaded from
func printConfigurationSources(cfg *settings.Settings) {
	fmt.Println("\nConfiguration Sources:")
	fmt.Println("---------------------")

	homeDir, _ := os.UserHomeDir()
	if homeDir == "" {
		fmt.Printf("%s Unable to determine home directory\n", ConflictSymbol)
		return
	}

	configDir := filepath.Join(homeDir, ".config", "interop")

	// Show main settings file
	mainSettingsPath := filepath.Join(configDir, "settings.toml")
	if _, err := os.Stat(mainSettingsPath); err == nil {
		fmt.Printf("%s Main Settings: %s\n", LocalSymbol, mainSettingsPath)
	} else {
		fmt.Printf("%s Main Settings: %s (Not found)\n", ConflictSymbol, mainSettingsPath)
	}

	// Show command directories
	fmt.Printf("%s Command Directories:\n", LocalSymbol)

	// Check default local config directory
	localConfigDir := filepath.Join(configDir, "config.d")
	if _, err := os.Stat(localConfigDir); err == nil {
		count := countTOMLFiles(localConfigDir)
		fmt.Printf("   %s %s (%d files)\n", LocalSymbol, localConfigDir, count)
	} else {
		fmt.Printf("   %s %s (Not found)\n", ConflictSymbol, localConfigDir)
	}

	// Check remote configuration status
	remoteConfigDir := filepath.Join(configDir, "config.d.remote")
	remoteExecutablesDir := filepath.Join(configDir, "executables.remote")

	fmt.Printf("%s Remote Configuration:\n", RemoteSymbol)

	if _, err := os.Stat(remoteConfigDir); err == nil {
		count := countTOMLFiles(remoteConfigDir)
		fmt.Printf("   %s config.d.remote: Available (%d files)\n", CommandEnabledSymbol, count)
	} else {
		fmt.Printf("   %s config.d.remote: Not available\n", CommandDisabledSymbol)
	}

	if _, err := os.Stat(remoteExecutablesDir); err == nil {
		count := countFiles(remoteExecutablesDir)
		fmt.Printf("   %s executables.remote: Available (%d files)\n", CommandEnabledSymbol, count)
	} else {
		fmt.Printf("   %s executables.remote: Not available\n", CommandDisabledSymbol)
	}

	// Show remote versions file if it exists
	versionsFile := filepath.Join(configDir, "versions.toml")
	if _, err := os.Stat(versionsFile); err == nil {
		fmt.Printf("   %s Remote tracking: Active\n", CommandEnabledSymbol)
	} else {
		fmt.Printf("   %s Remote tracking: Not active\n", CommandDisabledSymbol)
	}

	// Show any potential conflicts
	showPotentialConflicts(localConfigDir, remoteConfigDir)

	fmt.Println()
}

// printMCPServers shows MCP server configuration
func printMCPServers(cfg *settings.Settings) {
	fmt.Println("MCP Servers:")
	fmt.Println("-----------")

	// Default MCP server
	fmt.Printf("%s Default MCP Server (Port: %d)\n", MCPServerSymbol, cfg.MCPPort)
	fmt.Println("   └─ Commands: (commands with no MCP field)")
	fmt.Println()

	// Named MCP servers
	if len(cfg.MCPServers) > 0 {
		for name, server := range cfg.MCPServers {
			fmt.Printf("%s %s MCP Server (Port: %d)\n", MCPServerSymbol, name, server.Port)
			if server.Description != "" {
				fmt.Printf("   └─ %s\n", server.Description)
			}

			// Count commands assigned to this server
			cmdCount := 0
			for _, cmd := range cfg.Commands {
				if cmd.MCP == name {
					cmdCount++
				}
			}
			fmt.Printf("   └─ Commands: %d\n", cmdCount)
			fmt.Println()
		}
	}
}

// printCommands shows all commands with their source and relationship information
func printCommands(cfg *settings.Settings, projectBoundCommands map[string][]string, aliasedCommands map[string]map[string]string) {
	fmt.Println("Commands:")
	fmt.Println("--------")

	for cmdName, cmdConfig := range cfg.Commands {
		// Determine command type symbol
		var typeSymbol string
		var projectList []string
		var isGlobal bool

		if projects, bound := projectBoundCommands[cmdName]; bound {
			typeSymbol = ProjectCommandSymbol
			projectList = projects
			isGlobal = false
		} else {
			typeSymbol = GlobalCommandSymbol
			isGlobal = true
		}

		// Determine enabled status
		enabledSymbol := CommandEnabledSymbol
		if !cmdConfig.IsEnabled {
			enabledSymbol = CommandDisabledSymbol
		}

		// Determine command execution type
		execType := ShellCommandLabel
		if cmdConfig.IsExecutable {
			execType = ExecutableCommandLabel
		}

		// Determine source (this is where we'd need to track where commands come from)
		sourceInfo := determineCommandSource(cmdName)

		// Print the command details with source information
		fmt.Printf("%s %s %s %s %s\n", typeSymbol, enabledSymbol, cmdName, execType, sourceInfo)

		// Print description if available
		if cmdConfig.Description != "" {
			fmt.Printf("   └─ %s\n", cmdConfig.Description)
		}

		// Print MCP server assignment if available
		if cmdConfig.MCP != "" {
			// Get server details
			if server, exists := cfg.MCPServers[cmdConfig.MCP]; exists {
				fmt.Printf("   └─ %s Assigned to MCP server: %s (Port: %d)\n", MCPServerSymbol, cmdConfig.MCP, server.Port)
			} else {
				fmt.Printf("   └─ %s Warning: Assigned to undefined MCP server: %s\n", CommandDisabledSymbol, cmdConfig.MCP)
			}
		} else {
			fmt.Printf("   └─ %s Default MCP server (Port: %d)\n", MCPServerSymbol, cfg.MCPPort)
		}

		// Print project associations
		if !isGlobal {
			fmt.Printf("   └─ Project bound: %s\n", strings.Join(projectList, ", "))
		}

		// Print aliases if any
		if aliases, hasAliases := aliasedCommands[cmdName]; hasAliases && len(aliases) > 0 {
			fmt.Printf("   └─ Aliases:\n")
			for alias, projectName := range aliases {
				fmt.Printf("      └─ %s %s (in project: %s)\n", ProjectAliasSymbol, alias, projectName)
			}
		}

		fmt.Println()
	}
}

// determineCommandSource attempts to determine where a command comes from
func determineCommandSource(cmdName string) string {
	homeDir, _ := os.UserHomeDir()
	if homeDir == "" {
		return ""
	}

	configDir := filepath.Join(homeDir, ".config", "interop")

	// Check if command might be from remote
	remoteConfigDir := filepath.Join(configDir, "config.d.remote")
	localConfigDir := filepath.Join(configDir, "config.d")

	// Check if we can find the command file in either directory
	if _, err := os.Stat(remoteConfigDir); err == nil {
		// Look for command files in remote directory
		if found := findCommandInDir(remoteConfigDir, cmdName); found {
			return fmt.Sprintf("(%s Remote)", RemoteSymbol)
		}
	}

	if _, err := os.Stat(localConfigDir); err == nil {
		// Look for command files in local directory
		if found := findCommandInDir(localConfigDir, cmdName); found {
			return fmt.Sprintf("(%s Local)", LocalSymbol)
		}
	}

	// Check main settings file
	mainSettingsPath := filepath.Join(configDir, "settings.toml")
	if found := findCommandInMainSettings(mainSettingsPath, cmdName); found {
		return fmt.Sprintf("(%s Main Settings)", LocalSymbol)
	}

	return fmt.Sprintf("(%s Unknown)", ConflictSymbol)
}

// findCommandInDir searches for a command in a directory of TOML files
func findCommandInDir(dirPath, cmdName string) bool {
	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".toml") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// Simple check if command name appears in file
		if strings.Contains(string(data), fmt.Sprintf(`[commands.%s]`, cmdName)) ||
			strings.Contains(string(data), fmt.Sprintf(`"%s"`, cmdName)) {
			return fmt.Errorf("found") // Use error to break out of walk
		}

		return nil
	})

	return err != nil && err.Error() == "found"
}

// findCommandInMainSettings checks if a command is defined in the main settings file
func findCommandInMainSettings(filePath, cmdName string) bool {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	return strings.Contains(string(data), fmt.Sprintf(`[commands.%s]`, cmdName)) ||
		strings.Contains(string(data), fmt.Sprintf(`"%s"`, cmdName))
}

// printLegend shows the legend for all symbols used
func printLegend() {
	fmt.Println("Legend:")
	fmt.Println("-------")
	fmt.Printf("%s Global Command\n", GlobalCommandSymbol)
	fmt.Printf("%s Project-bound Command\n", ProjectCommandSymbol)
	fmt.Printf("%s Command Alias\n", ProjectAliasSymbol)
	fmt.Printf("%s Enabled Command\n", CommandEnabledSymbol)
	fmt.Printf("%s Disabled Command\n", CommandDisabledSymbol)
	fmt.Printf("%s MCP Server Association\n", MCPServerSymbol)
	fmt.Printf("%s Local Configuration\n", LocalSymbol)
	fmt.Printf("%s Remote Configuration\n", RemoteSymbol)
	fmt.Printf("%s Warning/Conflict\n", ConflictSymbol)
	fmt.Println(ExecutableCommandLabel, "- Executable command")
	fmt.Println(ShellCommandLabel, "- Shell command")
}

// expandPath expands tilde and relative paths
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if homeDir, err := os.UserHomeDir(); err == nil {
			return filepath.Join(homeDir, path[2:])
		}
	}
	return path
}

// countTOMLFiles counts the number of .toml files in a directory
func countTOMLFiles(dirPath string) int {
	count := 0
	filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(path, ".toml") {
			count++
		}
		return nil
	})
	return count
}

// countFiles counts the number of files in a directory
func countFiles(dirPath string) int {
	count := 0
	filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			count++
		}
		return nil
	})
	return count
}

// showPotentialConflicts identifies potential conflicts between local and remote configs
func showPotentialConflicts(localDir, remoteDir string) {
	if _, err := os.Stat(localDir); os.IsNotExist(err) {
		return
	}
	if _, err := os.Stat(remoteDir); os.IsNotExist(err) {
		return
	}

	localCommands := getCommandsFromDir(localDir)
	remoteCommands := getCommandsFromDir(remoteDir)

	conflicts := []string{}
	for cmd := range localCommands {
		if _, exists := remoteCommands[cmd]; exists {
			conflicts = append(conflicts, cmd)
		}
	}

	if len(conflicts) > 0 {
		fmt.Printf("%s Potential Conflicts:\n", ConflictSymbol)
		for _, cmd := range conflicts {
			fmt.Printf("   %s Command '%s' exists in both local and remote configs\n", ConflictSymbol, cmd)
		}
		fmt.Printf("   → Local configurations take precedence\n")
	}
}

// getCommandsFromDir extracts command names from TOML files in a directory
func getCommandsFromDir(dirPath string) map[string]bool {
	commands := make(map[string]bool)

	filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(path, ".toml") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// Simple parsing to find command definitions
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "[commands.") && strings.HasSuffix(line, "]") {
				// Extract command name from [commands.cmdname]
				cmdName := strings.TrimPrefix(line, "[commands.")
				cmdName = strings.TrimSuffix(cmdName, "]")
				commands[cmdName] = true
			}
		}

		return nil
	})

	return commands
}
