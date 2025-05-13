package display

import (
	"fmt"
	"interop/internal/settings"
	"strings"
)

// Command relationship types
const (
	GlobalCommandSymbol    = "🌐"
	ProjectCommandSymbol   = "📂"
	ProjectAliasSymbol     = "🔄"
	CommandDisabledSymbol  = "❌"
	CommandEnabledSymbol   = "✓"
	ExecutableCommandLabel = "(Executable)"
	ShellCommandLabel      = "(Shell)"
)

// PrintCommandGraph displays a visual graph of commands and their relationships
func PrintCommandGraph(cfg *settings.Settings) {
	fmt.Println("Command Graph Overview")
	fmt.Println("=====================")

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

	// Print the command graph
	fmt.Println("\nCommands:")
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

		// Print the command details
		fmt.Printf("%s %s %s %s\n", typeSymbol, enabledSymbol, cmdName, execType)

		// Print description if available
		if cmdConfig.Description != "" {
			fmt.Printf("   └─ %s\n", cmdConfig.Description)
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

	// Legend
	fmt.Println("\nLegend:")
	fmt.Println("-------")
	fmt.Printf("%s Global Command\n", GlobalCommandSymbol)
	fmt.Printf("%s Project-bound Command\n", ProjectCommandSymbol)
	fmt.Printf("%s Command Alias\n", ProjectAliasSymbol)
	fmt.Printf("%s Enabled Command\n", CommandEnabledSymbol)
	fmt.Printf("%s Disabled Command\n", CommandDisabledSymbol)
	fmt.Println(ExecutableCommandLabel, "- Executable command")
	fmt.Println(ShellCommandLabel, "- Shell command")
}
