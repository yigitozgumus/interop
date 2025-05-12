package shell

import (
	"fmt"
	"interop/internal/util"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ShellType represents a type of shell
type ShellType string

const (
	// ShellTypeBash represents the bash shell
	ShellTypeBash ShellType = "bash"
	// ShellTypeZsh represents the zsh shell
	ShellTypeZsh ShellType = "zsh"
	// ShellTypeFish represents the fish shell
	ShellTypeFish ShellType = "fish"
	// ShellTypeUnknown represents an unknown shell
	ShellTypeUnknown ShellType = "unknown"
	// ShellTypeSh represents the standard sh shell
	ShellTypeSh ShellType = "sh"
)

// Shell represents a user's shell environment
type Shell struct {
	Path string    // Path to the shell executable
	Type ShellType // Type of shell
}

// GetUserShell returns the user's shell executable path and type
func GetUserShell() Shell {
	// Get user's shell from environment
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		// Fallback to sh if SHELL is not defined
		util.Warning("SHELL environment variable not set, defaulting to /bin/sh")
		return Shell{
			Path: "/bin/sh",
			Type: ShellTypeSh,
		}
	}

	// Get shell type from path
	shellType := getShellTypeFromPath(shellPath)

	return Shell{
		Path: shellPath,
		Type: shellType,
	}
}

// getShellTypeFromPath determines the shell type from the shell path
func getShellTypeFromPath(shellPath string) ShellType {
	// Get the shell name (basename)
	shellName := filepath.Base(shellPath)

	// Determine shell type
	switch shellName {
	case "bash":
		return ShellTypeBash
	case "zsh":
		return ShellTypeZsh
	case "fish":
		return ShellTypeFish
	case "sh":
		return ShellTypeSh
	default:
		return ShellTypeUnknown
	}
}

// ExecuteCommand executes a command using the specified shell
func (s *Shell) ExecuteCommand(command string) *exec.Cmd {
	// Regular shell command
	return exec.Command(s.Path, "-c", command)
}

// ExecuteInteractiveCommand executes a command in interactive mode
func (s *Shell) ExecuteInteractiveCommand(command string) *exec.Cmd {
	// Run in interactive shell to ensure aliases are loaded
	switch s.Type {
	case ShellTypeBash:
		return exec.Command(s.Path, "-ic", command)
	case ShellTypeZsh:
		return exec.Command(s.Path, "-ic", command)
	case ShellTypeFish:
		return exec.Command(s.Path, "-ic", command)
	default:
		return exec.Command(s.Path, "-ic", command)
	}
}

// ExecuteAlias executes a shell alias
func (s *Shell) ExecuteAlias(alias string) *exec.Cmd {
	// Extract the alias name
	aliasName := strings.TrimSpace(strings.TrimPrefix(alias, "alias:"))

	// Run in interactive shell to ensure aliases are loaded
	return s.ExecuteInteractiveCommand(aliasName)
}

// ExecuteScript executes a script file
func (s *Shell) ExecuteScript(scriptPath string, args ...string) (*exec.Cmd, error) {
	// Check if the script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("script '%s' not found", scriptPath)
	}

	// Make sure the file is executable
	if err := os.Chmod(scriptPath, 0755); err != nil {
		util.Warning("Failed to set executable permissions on script '%s': %v", scriptPath, err)
	}

	// Create the command with arguments
	return exec.Command(scriptPath, args...), nil
}

// IsAliasCommand checks if a command string is an alias command
func IsAliasCommand(cmd string) bool {
	return strings.HasPrefix(cmd, "alias:")
}

// IsLocalScriptCommand checks if a command string is a local script command
func IsLocalScriptCommand(cmd string) bool {
	return strings.HasPrefix(cmd, "./")
}

// ParseLocalScript parses a local script command into script path and arguments
func ParseLocalScript(cmd string) (string, []string) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "", nil
	}

	scriptPath := parts[0]
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}

	return scriptPath, args
}
