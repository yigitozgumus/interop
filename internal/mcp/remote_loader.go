package mcp

import (
	"fmt"
	"interop/internal/logging"
	"interop/internal/settings"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// RemoteCommandLoader handles loading commands from remote repositories
type RemoteCommandLoader struct{}

// NewRemoteCommandLoader creates a new remote command loader
func NewRemoteCommandLoader() *RemoteCommandLoader {
	return &RemoteCommandLoader{}
}

// LoadCommandsFromRemote fetches commands from a remote repository and returns them
// without persisting to disk
func (r *RemoteCommandLoader) LoadCommandsFromRemote(repoURL string) (map[string]settings.CommandConfig, error) {
	logging.Message("Loading commands from remote repository: %s", repoURL)

	// Validate the Git URL
	if err := r.validateGitURL(repoURL); err != nil {
		return nil, fmt.Errorf("invalid Git repository URL: %w", err)
	}

	// Clone repository to temporary directory
	tmpDir, err := r.cloneRepository(repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Validate repository structure
	if err := r.validateRepoStructure(tmpDir); err != nil {
		return nil, fmt.Errorf("invalid repository structure: %w", err)
	}

	// Load commands from config.d directory
	commands, err := r.loadCommandsFromConfigDir(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load commands from config.d: %w", err)
	}

	// Update executable paths to point to the temporary directory
	if err := r.updateExecutablePaths(commands, tmpDir); err != nil {
		return nil, fmt.Errorf("failed to update executable paths: %w", err)
	}

	logging.Message("Successfully loaded %d commands from remote repository", len(commands))
	return commands, nil
}

// validateGitURL validates if the provided URL is a valid Git repository URL
func (r *RemoteCommandLoader) validateGitURL(gitURL string) error {
	if gitURL == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Check for SSH Git URLs (git@host:user/repo.git)
	sshPattern := regexp.MustCompile(`^git@[\w\.\-]+:[\w\.\-~]+/[\w\.\-]+\.git$`)
	if sshPattern.MatchString(gitURL) {
		return nil
	}

	// Check for HTTPS/HTTP Git URLs
	parsedURL, err := url.Parse(gitURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Must be HTTP or HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL must use http, https, or SSH format (git@host:user/repo.git)")
	}

	// Check if it's a known Git hosting service or has .git extension
	host := strings.ToLower(parsedURL.Host)
	path := parsedURL.Path

	// Known Git hosting services
	knownHosts := []string{
		"github.com",
		"gitlab.com",
		"bitbucket.org",
		"codeberg.org",
		"git.sr.ht",
	}

	isKnownHost := false
	for _, knownHost := range knownHosts {
		if host == knownHost || strings.HasSuffix(host, "."+knownHost) {
			isKnownHost = true
			break
		}
	}

	// If it's a known host, the path should look like a repository path
	if isKnownHost {
		// Path should be like /user/repo, /user/repo.git, or /~user/repo (for SourceHut)
		pathPattern := regexp.MustCompile(`^/[~]?[\w\.\-]+/[\w\.\-]+(?:\.git)?/?$`)
		if !pathPattern.MatchString(path) {
			return fmt.Errorf("invalid repository path format for %s", host)
		}
		return nil
	}

	// For unknown hosts, require .git extension
	if !strings.HasSuffix(path, ".git") {
		return fmt.Errorf("URL must end with .git or be from a known Git hosting service")
	}

	return nil
}

// cloneRepository clones the git repository to a temporary directory
func (r *RemoteCommandLoader) cloneRepository(repoURL string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "interop-mcp-remote-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	logging.Message("Cloning repository %s to %s", repoURL, tmpDir)

	_, err = r.runGitCommand("", "clone", repoURL, tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	return tmpDir, nil
}

// validateRepoStructure validates that the repository has the required folder structure
func (r *RemoteCommandLoader) validateRepoStructure(repoPath string) error {
	configDir := filepath.Join(repoPath, "config.d")
	executablesDir := filepath.Join(repoPath, "executables")

	// Check if config.d exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return fmt.Errorf("repository must contain a 'config.d' folder")
	}

	// Check if executables exists
	if _, err := os.Stat(executablesDir); os.IsNotExist(err) {
		return fmt.Errorf("repository must contain an 'executables' folder")
	}

	logging.Message("Repository structure validation passed")
	return nil
}

// loadCommandsFromConfigDir loads all TOML files from the config.d directory
func (r *RemoteCommandLoader) loadCommandsFromConfigDir(repoPath string) (map[string]settings.CommandConfig, error) {
	configDir := filepath.Join(repoPath, "config.d")
	commands := make(map[string]settings.CommandConfig)

	// Walk through all files in config.d
	err := filepath.Walk(configDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-TOML files
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".toml") {
			return nil
		}

		// Load commands from this TOML file
		fileCommands, err := r.loadCommandsFromFile(path)
		if err != nil {
			logging.Warning("Failed to load commands from %s: %v", path, err)
			return nil // Continue processing other files
		}

		// Merge commands into the main map
		for name, cmd := range fileCommands {
			if _, exists := commands[name]; exists {
				logging.Warning("Command '%s' already exists, skipping duplicate from %s", name, path)
				continue
			}
			commands[name] = cmd
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk config.d directory: %w", err)
	}

	return commands, nil
}

// loadCommandsFromFile loads commands from a single TOML file
func (r *RemoteCommandLoader) loadCommandsFromFile(filePath string) (map[string]settings.CommandConfig, error) {
	var config struct {
		Commands map[string]settings.CommandConfig `toml:"commands"`
	}

	if _, err := toml.DecodeFile(filePath, &config); err != nil {
		return nil, fmt.Errorf("failed to decode TOML file %s: %w", filePath, err)
	}

	if config.Commands == nil {
		config.Commands = make(map[string]settings.CommandConfig)
	}

	logging.Message("Loaded %d commands from %s", len(config.Commands), filePath)
	return config.Commands, nil
}

// updateExecutablePaths updates executable commands to use the temporary directory paths
func (r *RemoteCommandLoader) updateExecutablePaths(commands map[string]settings.CommandConfig, tmpDir string) error {
	executablesDir := filepath.Join(tmpDir, "executables")

	for name, cmd := range commands {
		if cmd.IsExecutable {
			// Split command to get the executable name
			cmdParts := strings.Fields(cmd.Cmd)
			if len(cmdParts) == 0 {
				continue
			}

			execName := cmdParts[0]
			execPath := filepath.Join(executablesDir, execName)

			// Check if the executable exists
			if _, err := os.Stat(execPath); err == nil {
				// Make the executable executable
				if err := os.Chmod(execPath, 0755); err != nil {
					logging.Warning("Failed to make executable %s: %v", execPath, err)
				}

				// Update the command to use the full path
				if len(cmdParts) > 1 {
					cmd.Cmd = fmt.Sprintf("%s %s", execPath, strings.Join(cmdParts[1:], " "))
				} else {
					cmd.Cmd = execPath
				}

				commands[name] = cmd
				logging.Message("Updated executable path for command '%s': %s", name, execPath)
			} else {
				logging.Warning("Executable '%s' not found for command '%s'", execName, name)
			}
		}
	}

	return nil
}

// runGitCommand runs a git command in the specified directory
func (r *RemoteCommandLoader) runGitCommand(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git command failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}
