package execution

import (
	"fmt"
	"interop/internal/logging"
	"interop/internal/shell"
	"os"
	"os/exec"
	"path/filepath"
)

// CommandInfo defines a command that can be executed
type CommandInfo struct {
	Name         string
	Description  string
	IsEnabled    bool
	Cmd          string
	IsExecutable bool
}

// Run executes a command by name
func Run(command CommandInfo, executablesPath string, projectPath ...string) error {
	return RunWithSearchPaths(command, []string{executablesPath}, projectPath...)
}

// RunWithSearchPaths executes a command, searching for executables in multiple paths
func RunWithSearchPaths(command CommandInfo, executableSearchPaths []string, projectPath ...string) error {
	if !command.IsEnabled {
		return fmt.Errorf("command '%s' is not enabled", command.Name)
	}

	logging.Message("Command '%s' is enabled, proceeding with execution", command.Name)

	// Store current working directory if we need to change to project directory
	var currentDir string
	var err error

	// If project path is provided, change to that directory before running the command
	if len(projectPath) > 0 && projectPath[0] != "" {
		// Save current directory to return to after command execution
		currentDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}

		projectDir := projectPath[0]
		// If path doesn't exist, try to report a more helpful error
		if _, err := os.Stat(projectDir); os.IsNotExist(err) {
			return fmt.Errorf("project directory doesn't exist: %s", projectDir)
		}

		// Change to project directory
		logging.Message("Changing to project directory: %s", projectDir)
		if err := os.Chdir(projectDir); err != nil {
			return fmt.Errorf("failed to change to project directory: %w", err)
		}

		// Ensure we change back to original directory when done
		defer func() {
			logging.Message("Changing back to original directory: %s", currentDir)
			if err := os.Chdir(currentDir); err != nil {
				logging.Error("Failed to change back to original directory: %v", err)
			}
		}()
	}

	// Get user's shell
	userShell := shell.GetUserShell()

	var commandToRun *exec.Cmd

	// Check if this command should run as a shell alias
	if shell.IsAliasCommand(command.Cmd) {
		// Run the alias using the shell package
		logging.Message("Running shell alias: %s", command.Cmd)
		commandToRun = userShell.ExecuteAlias(command.Cmd)
	} else if command.IsExecutable {
		// For executable commands, look for the executable in all search paths
		execPath, err := FindExecutable(command.Cmd, executableSearchPaths)
		if err != nil {
			return err
		}

		logging.Message("Found executable '%s', executing", execPath)
		commandToRun = exec.Command(execPath)
	} else if shell.IsLocalScriptCommand(command.Cmd) {
		// Local script that should be executed directly
		scriptPath, args := shell.ParseLocalScript(command.Cmd)
		logging.Message("Running local script: %s with arguments: %v", scriptPath, args)

		var err error
		commandToRun, err = userShell.ExecuteScript(scriptPath, args...)
		if err != nil {
			return err
		}
	} else {
		// Standard shell command
		logging.Message("Running shell command: %s", command.Cmd)
		commandToRun = userShell.ExecuteCommand(command.Cmd)
	}

	// Set up the command to use the current terminal
	commandToRun.Stdin = os.Stdin
	commandToRun.Stdout = os.Stdout
	commandToRun.Stderr = os.Stderr

	// Run the command
	return commandToRun.Run()
}

// FindExecutable searches for an executable in the provided search paths
func FindExecutable(executableName string, searchPaths []string) (string, error) {
	// Check each search path
	for _, searchPath := range searchPaths {
		candidatePath := filepath.Join(searchPath, executableName)
		if _, err := os.Stat(candidatePath); err == nil {
			// Make sure the file is executable
			if err := os.Chmod(candidatePath, 0755); err != nil {
				return "", fmt.Errorf("failed to set executable permissions: %w", err)
			}
			return candidatePath, nil
		}
	}

	// If not found in the specified search paths, try to find it in system PATH
	execPath, err := exec.LookPath(executableName)
	if err != nil {
		return "", fmt.Errorf("executable '%s' not found in any search path or system PATH: %v", executableName, err)
	}

	return execPath, nil
}
