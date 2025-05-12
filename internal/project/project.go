package project

import (
	"interop/internal/display"
	"interop/internal/path"
	"interop/internal/settings"
	"interop/internal/util"
)

// List prints out all configured projects with their name, path, and validity
func List(cfg *settings.Settings) {
	if len(cfg.Projects) == 0 {
		display.PrintNoItemsFound("projects")
		return
	}

	display.PrintProjectHeader()

	// Using path package directly instead of keeping homeDir
	_, err := path.HomeDir()
	if err != nil {
		util.Warning("Warning: Could not determine home directory")
	}

	for name, project := range cfg.Projects {
		// Use the path package to validate and expand the path
		pathInfo, err := path.ExpandAndValidate(project.Path)
		if err != nil {
			util.Warning("Failed to expand path for project %s: %v", name, err)
			continue
		}

		valid := "✓"
		if !pathInfo.Exists {
			valid = "✗"
		}

		inHomeDir := "✓"
		if !pathInfo.InHomeDir {
			inHomeDir = "✗"
		}

		// Print project details using display package
		display.PrintProjectName(name)
		display.PrintProjectPath(project.Path)
		display.PrintProjectStatus(valid, inHomeDir)
		display.PrintProjectDescription(project.Description)

		display.PrintSeparator()
	}
}

// ListWithCommands prints out all configured projects with their commands
func ListWithCommands(cfg *settings.Settings) {
	if len(cfg.Projects) == 0 {
		display.PrintNoItemsFound("projects")
		return
	}

	display.PrintProjectHeader()

	for name, project := range cfg.Projects {
		// Use the path package to validate and expand the path
		pathInfo, err := path.ExpandAndValidate(project.Path)
		if err != nil {
			util.Warning("Failed to expand path for project %s: %v", name, err)
			continue
		}

		valid := "✓"
		if !pathInfo.Exists {
			valid = "✗"
		}

		inHomeDir := "✓"
		if !pathInfo.InHomeDir {
			inHomeDir = "✗"
		}

		// Print project details using display package
		display.PrintProjectName(name)
		display.PrintProjectPath(project.Path)
		display.PrintProjectStatus(valid, inHomeDir)
		display.PrintProjectDescription(project.Description)

		// Display commands for this project
		if len(project.Commands) > 0 {
			// Print commands header
			display.PrintCommandProjects([]string{})

			for _, alias := range project.Commands {
				cmd, exists := cfg.Commands[alias.CommandName]
				if !exists {
					display.PrintUnresolvedCommand(alias.CommandName)
					continue
				}

				display.PrintProjectCommands(alias.CommandName, alias.Alias, cmd.Description)
			}
		}

		display.PrintSeparator()
	}
}

// ListWithCustomHomeDir is used for testing to allow overriding the home directory
func ListWithCustomHomeDir(cfg *settings.Settings, homeDirFunc func() (string, error)) {
	if len(cfg.Projects) == 0 {
		display.PrintNoItemsFound("projects")
		return
	}

	display.PrintProjectHeader()

	// Set up custom home directory for testing
	resetHomeDir := path.SetHomeDirFunc(homeDirFunc)
	defer resetHomeDir()

	for name, project := range cfg.Projects {
		// Now use the path package which will use our custom homeDirFunc
		pathInfo, err := path.ExpandAndValidate(project.Path)
		if err != nil {
			util.Warning("Failed to expand path for project %s: %v", name, err)
			continue
		}

		valid := "✓"
		if !pathInfo.Exists {
			valid = "✗"
		}

		inHomeDirStr := "✓"
		if !pathInfo.InHomeDir {
			inHomeDirStr = "✗"
		}

		// Print project details using display package
		display.PrintProjectName(name)
		display.PrintProjectPath(project.Path)
		display.PrintProjectStatus(valid, inHomeDirStr)
		display.PrintProjectDescription(project.Description)

		display.PrintSeparator()
	}
}
