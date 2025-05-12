package path

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// HomeDirFunc defines the function type for getting the home directory
type HomeDirFunc func() (string, error)

// homeDirFunc is the function used to get the home directory
// This can be overridden for testing
var homeDirFunc HomeDirFunc = os.UserHomeDir

// SetHomeDirFunc allows overriding the home directory function for testing
func SetHomeDirFunc(fn HomeDirFunc) func() {
	old := homeDirFunc
	homeDirFunc = fn
	return func() {
		homeDirFunc = old
	}
}

// Info contains information about a path
type Info struct {
	Original  string // Original path as specified by user
	Absolute  string // Full absolute path with tilde expansion
	Exists    bool   // Whether the path exists
	InHomeDir bool   // Whether the path is inside the user's home directory
}

// HomeDir returns the user's home directory
func HomeDir() (string, error) {
	return homeDirFunc()
}

// Expand expands a path with tilde expansion and converts to absolute path
func Expand(path string) (string, error) {
	// Get user home directory
	homeDir, err := HomeDir()
	if err != nil {
		return path, fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Handle tilde expansion for home directory
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:]), nil
	}

	// If not absolute, treat as relative to home
	if !filepath.IsAbs(path) {
		return filepath.Join(homeDir, path), nil
	}

	// Already absolute
	return path, nil
}

// ExpandAndValidate expands a path and checks if it exists and is within the home directory
func ExpandAndValidate(path string) (Info, error) {
	info := Info{
		Original: path,
	}

	// Get user home directory
	homeDir, err := HomeDir()
	if err != nil {
		return info, fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Handle tilde expansion for home directory
	if strings.HasPrefix(path, "~/") {
		info.Absolute = filepath.Join(homeDir, path[2:])
	} else if !filepath.IsAbs(path) {
		// If not absolute, treat as relative to home
		info.Absolute = filepath.Join(homeDir, path)
	} else {
		// Already absolute
		info.Absolute = path
	}

	// Check if path exists
	if _, err := os.Stat(info.Absolute); err == nil {
		info.Exists = true
	}

	// Check if path is inside home directory
	if homeDir != "" && filepath.IsAbs(info.Absolute) {
		if strings.HasPrefix(info.Absolute, homeDir) {
			info.InHomeDir = true
		}
	}

	return info, nil
}

// Executable finds the path to an executable by searching in the provided directories
func Executable(executableName string, searchPaths []string) (string, error) {
	// Check each search path
	for _, dir := range searchPaths {
		// Expand the path
		expandedDir, err := Expand(dir)
		if err != nil {
			continue
		}

		candidatePath := filepath.Join(expandedDir, executableName)
		if _, err := os.Stat(candidatePath); err == nil {
			return candidatePath, nil
		}
	}

	// If not found in the search paths, try to find it in the system PATH
	if path, err := exec.LookPath(executableName); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("executable '%s' not found in any search path", executableName)
}

// CreateDirectories creates all directories in the given path
func CreateDirectories(path string) error {
	expandedPath, err := Expand(path)
	if err != nil {
		return err
	}
	return os.MkdirAll(expandedPath, 0o755)
}
