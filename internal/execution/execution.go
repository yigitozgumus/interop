package execution

import (
	"context"
	"fmt"
	"interop/internal/errors"
	"interop/internal/logging"
	"interop/internal/shell"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CommandInfo defines a command that can be executed
type CommandInfo struct {
	Name         string
	Description  string
	IsEnabled    bool
	Cmd          string
	IsExecutable bool
}

// Command represents a command to be executed
type Command struct {
	Path string   // Path to the executable
	Args []string // Command arguments
	Dir  string   // Working directory
	Env  []string // Environment variables
}

// Executor handles command execution
type Executor struct {
	Timeout time.Duration // Command timeout (0 means no timeout)
}

// NewExecutor creates a new command executor with default settings
func NewExecutor() *Executor {
	return &Executor{
		Timeout: 0, // No timeout by default
	}
}

// WithTimeout creates an executor with the specified timeout
func WithTimeout(timeout time.Duration) *Executor {
	return &Executor{
		Timeout: timeout,
	}
}

// Run executes a command by name
func Run(command CommandInfo, executablesPath string, projectPath ...string) error {
	return RunWithSearchPathsAndArgs(command, []string{executablesPath}, nil, projectPath...)
}

// RunWithSearchPathsAndArgs executes a command with arguments, searching for executables in multiple paths
func RunWithSearchPathsAndArgs(command CommandInfo, executableSearchPaths []string, args []string, projectPath ...string) error {
	if !command.IsEnabled {
		logging.Error("command '%s' is not enabled", command.Name)
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
			logging.Error("failed to get current working directory: %w", err)
		}

		projectDir := projectPath[0]
		// If path doesn't exist, try to report a more helpful error
		if _, err := os.Stat(projectDir); os.IsNotExist(err) {
			logging.Error("project directory doesn't exist: %s", projectDir)
		}

		// Change to project directory
		logging.Message("Changing to project directory: %s", projectDir)
		if err := os.Chdir(projectDir); err != nil {
			logging.Error("failed to change to project directory: %w", err)
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
	logging.Message("User shell: %s", userShell)

	var commandToRun *exec.Cmd

	// Check if this command should run as a shell alias
	if shell.IsAliasCommand(command.Cmd) {
		logging.Message("Running shell alias: %s", command.Cmd)
		// Run the alias using the shell package
		if args != nil && len(args) > 0 {
			cmdWithArgs := fmt.Sprintf("%s %s", command.Cmd, strings.Join(args, " "))
			logging.Message("Running shell alias with args: %s", cmdWithArgs)
			commandToRun = userShell.ExecuteAlias(cmdWithArgs)
		} else {
			logging.Message("Running shell alias: %s", command.Cmd)
			commandToRun = userShell.ExecuteAlias(command.Cmd)
		}
	} else if command.IsExecutable {
		// For executable commands, parse the command line to extract the executable name and any arguments
		cmdFields := strings.Fields(command.Cmd)
		executableName := cmdFields[0]
		cmdArgs := []string{}

		// Add command's own arguments if any
		if len(cmdFields) > 1 {
			cmdArgs = append(cmdArgs, cmdFields[1:]...)
		}

		// Add additional arguments if provided
		if args != nil && len(args) > 0 {
			cmdArgs = append(cmdArgs, args...)
		}

		// Look for the executable in all search paths
		execPath, err := FindExecutable(executableName, executableSearchPaths)
		if err != nil {
			return err
		}

		if len(cmdArgs) > 0 {
			logging.Message("Found executable '%s', executing with args: %v", execPath, cmdArgs)
			commandToRun = exec.Command(execPath, cmdArgs...)
		} else {
			logging.Message("Found executable '%s', executing", execPath)
			commandToRun = exec.Command(execPath)
		}
	} else if shell.IsLocalScriptCommand(command.Cmd) {
		// Local script that should be executed directly
		scriptPath, scriptArgs := shell.ParseLocalScript(command.Cmd)

		// Append additional arguments if provided
		if args != nil && len(args) > 0 {
			scriptArgs = append(scriptArgs, args...)
		}

		logging.Message("Running local script: %s with arguments: %v", scriptPath, scriptArgs)

		var err error
		commandToRun, err = userShell.ExecuteScript(scriptPath, scriptArgs...)
		if err != nil {
			return err
		}
	} else {
		// Standard shell command
		if args != nil && len(args) > 0 {
			cmdWithArgs := fmt.Sprintf("%s %s", command.Cmd, strings.Join(args, " "))
			logging.Message("Running shell command with args: %s", cmdWithArgs)
			commandToRun = userShell.ExecuteCommand(cmdWithArgs)
		} else {
			logging.Message("Running shell command: %s", command.Cmd)
			commandToRun = userShell.ExecuteCommand(command.Cmd)
		}
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
	// Make sure we only use the executable name, not any arguments
	executableName = strings.Fields(executableName)[0]

	// Check each search path
	for _, searchPath := range searchPaths {
		candidatePath := filepath.Join(searchPath, executableName)
		if fileInfo, err := os.Stat(candidatePath); err == nil {
			// Check if the file has executable permissions
			if fileInfo.Mode()&0100 == 0 {
				// File exists but is not executable
				return "", fmt.Errorf("file '%s' exists but doesn't have executable permissions. Run 'chmod +x %s' to fix this issue", candidatePath, candidatePath)
			}
			return candidatePath, nil
		}
	}

	// If not found in the specified search paths, try to find it in system PATH
	execPath, err := exec.LookPath(executableName)
	if err != nil {
		return "", fmt.Errorf("executable '%s' not found in any search path or system PATH: %v", executableName, err)
	}

	// Check if the found file has executable permissions
	if fileInfo, err := os.Stat(execPath); err == nil {
		if fileInfo.Mode()&0100 == 0 {
			// File exists but is not executable
			return "", fmt.Errorf("file '%s' exists but doesn't have executable permissions. Run 'chmod +x %s' to fix this issue", execPath, execPath)
		}
	}

	return execPath, nil
}

// Execute runs the command and returns an error if it fails
func (e *Executor) Execute(cmd *Command) error {
	return e.ExecuteWithContext(context.Background(), cmd)
}

// ExecuteWithContext runs the command with the provided context
func (e *Executor) ExecuteWithContext(ctx context.Context, cmd *Command) error {
	logging.Message("Executing command: %s %s", cmd.Path, strings.Join(cmd.Args, " "))

	if cmd.Dir != "" {
		logging.Message("Working directory: %s", cmd.Dir)
		// Check if directory exists
		if _, err := os.Stat(cmd.Dir); os.IsNotExist(err) {
			return errors.NewExecutionError(fmt.Sprintf("Working directory does not exist: %s", cmd.Dir), err)
		}
	}

	// Create the command with context
	execCmd := exec.CommandContext(ctx, cmd.Path, cmd.Args...)

	// Set working directory if specified
	if cmd.Dir != "" {
		execCmd.Dir = cmd.Dir
	}

	// Set environment variables if provided
	if len(cmd.Env) > 0 {
		execCmd.Env = append(os.Environ(), cmd.Env...)
	} else {
		execCmd.Env = os.Environ()
	}

	// Connect command to standard I/O
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	// Create a context with timeout if specified
	var cancel context.CancelFunc
	if e.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, e.Timeout)
		defer cancel()
	}

	// Run the command
	err := execCmd.Run()
	if err != nil {
		return errors.NewExecutionError(fmt.Sprintf("Command execution failed: %s", strings.Join(cmd.Args, " ")), err)
	}

	return nil
}

// RunInDirectory executes a command in the specified directory
func RunInDirectory(dir string, command string, args ...string) error {
	cmd := &Command{
		Path: command,
		Args: args,
		Dir:  dir,
	}

	return NewExecutor().Execute(cmd)
}
