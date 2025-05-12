package shell

import (
	"fmt"
	"interop/internal/errors"
	"interop/internal/logging"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// Info contains information about the detected shell
type Info struct {
	Path   string // Full path to the shell
	Name   string // Shell name
	Option string // Shell option for executing commands (e.g., -c)
}

// Detector handles shell detection
type Detector struct{}

// NewDetector creates a new shell detector
func NewDetector() *Detector {
	return &Detector{}
}

// Detect detects the current shell
func (d *Detector) Detect() (*Info, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		// Default shell based on platform
		if runtime.GOOS == "windows" {
			if cmdPath, err := exec.LookPath("cmd.exe"); err == nil {
				return &Info{
					Path:   cmdPath,
					Name:   "cmd",
					Option: "/C",
				}, nil
			}
			return nil, errors.NewExecutionError("Failed to locate cmd.exe", nil)
		}

		// Default to /bin/sh on Unix systems
		return &Info{
			Path:   "/bin/sh",
			Name:   "sh",
			Option: "-c",
		}, nil
	}

	// Get shell name from path
	name := filepath.Base(shell)

	// Determine appropriate shell option
	option := "-c" // Default for most shells

	// Specific handling for windows shells
	if runtime.GOOS == "windows" {
		switch strings.ToLower(name) {
		case "cmd.exe", "cmd":
			option = "/C"
		case "powershell.exe", "powershell":
			option = "-Command"
		}
	}

	return &Info{
		Path:   shell,
		Name:   name,
		Option: option,
	}, nil
}

// IsWindows checks if the current shell is a Windows shell
func (i *Info) IsWindows() bool {
	name := strings.ToLower(i.Name)
	return name == "cmd.exe" || name == "cmd" ||
		name == "powershell.exe" || name == "powershell"
}

// DetectShell is a convenience function to detect the current shell
func DetectShell() (*Info, error) {
	return NewDetector().Detect()
}

// GetUserShell returns the user's shell executable path and type
func GetUserShell() Shell {
	// Get user's shell from environment
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		// Fallback to sh if SHELL is not defined
		logging.Warning("SHELL environment variable not set, defaulting to /bin/sh")
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
		logging.Warning("Failed to set executable permissions on script '%s': %v", scriptPath, err)
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
