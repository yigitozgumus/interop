package remote

import (
	"crypto/sha256"
	"fmt"
	"interop/internal/config"
	"interop/internal/logging"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// RemoteConfig represents the remote configuration stored in remote.toml
type RemoteConfig struct {
	RemoteURL string `toml:"remote-url"`
}

// VersionInfo represents file version tracking information
type VersionInfo struct {
	LastCommit string            `toml:"last-commit"`
	FileSHAs   map[string]string `toml:"file-shas"`
}

// Manager handles remote configuration operations
type Manager struct {
	configManager *config.Manager
}

// NewManager creates a new remote configuration manager
func NewManager() *Manager {
	return &Manager{
		configManager: config.NewManager(),
	}
}

// validateGitURL validates if the provided URL is a valid Git repository URL
func (m *Manager) validateGitURL(gitURL string) error {
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

// GetRemoteConfigPath returns the path to the remote.toml file
func (m *Manager) GetRemoteConfigPath() (string, error) {
	root, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	settingsDir := filepath.Join(root, m.configManager.PathConfig.SettingsDir)
	appDir := filepath.Join(settingsDir, m.configManager.PathConfig.AppDir)
	remoteDir := filepath.Join(appDir, m.configManager.PathConfig.RemoteDir)

	return filepath.Join(remoteDir, "remote.toml"), nil
}

// EnsureRemoteConfig ensures the remote configuration directory and file exist
func (m *Manager) EnsureRemoteConfig() error {
	// Ensure config directories are created
	if err := m.configManager.EnsureConfigDirectories(); err != nil {
		return err
	}

	// Check if remote.toml exists, create it if it doesn't
	configPath, err := m.GetRemoteConfigPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create empty remote config
		emptyConfig := RemoteConfig{}
		if err := m.saveRemoteConfig(&emptyConfig); err != nil {
			return fmt.Errorf("failed to create remote config file: %w", err)
		}
		logging.Message("Created remote configuration file at %s", configPath)
	}

	return nil
}

// loadRemoteConfig loads the remote configuration from remote.toml
func (m *Manager) loadRemoteConfig() (*RemoteConfig, error) {
	configPath, err := m.GetRemoteConfigPath()
	if err != nil {
		return nil, err
	}

	var remoteConfig RemoteConfig
	if _, err := toml.DecodeFile(configPath, &remoteConfig); err != nil {
		return nil, fmt.Errorf("failed to decode remote config file: %w", err)
	}

	return &remoteConfig, nil
}

// saveRemoteConfig saves the remote configuration to remote.toml
func (m *Manager) saveRemoteConfig(config *RemoteConfig) error {
	configPath, err := m.GetRemoteConfigPath()
	if err != nil {
		return err
	}

	f, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create remote config file: %w", err)
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(config); err != nil {
		return fmt.Errorf("failed to encode remote config data: %w", err)
	}

	return nil
}

// Add adds a remote URL to the configuration
func (m *Manager) Add(url string) error {
	if url == "" {
		return fmt.Errorf("remote URL cannot be empty")
	}

	// Validate the URL is a valid Git repository URL
	if err := m.validateGitURL(url); err != nil {
		return fmt.Errorf("invalid Git repository URL: %w", err)
	}

	// Ensure remote config exists
	if err := m.EnsureRemoteConfig(); err != nil {
		return err
	}

	// Create config with the new URL
	config := &RemoteConfig{
		RemoteURL: url,
	}

	if err := m.saveRemoteConfig(config); err != nil {
		return err
	}

	logging.Message("Added remote URL: %s", url)
	return nil
}

// Remove removes the current remote URL from the configuration
func (m *Manager) Remove() error {
	// Ensure remote config exists
	if err := m.EnsureRemoteConfig(); err != nil {
		return err
	}

	// Load current config to check if there's a URL to remove
	currentConfig, err := m.loadRemoteConfig()
	if err != nil {
		return err
	}

	if currentConfig.RemoteURL == "" {
		return fmt.Errorf("no remote URL configured to remove")
	}

	// Create empty config
	emptyConfig := &RemoteConfig{}

	if err := m.saveRemoteConfig(emptyConfig); err != nil {
		return err
	}

	logging.Message("Removed remote URL: %s", currentConfig.RemoteURL)
	return nil
}

// Show displays the current remote URL or notifies if not set
func (m *Manager) Show() error {
	// Ensure remote config exists
	if err := m.EnsureRemoteConfig(); err != nil {
		return err
	}

	config, err := m.loadRemoteConfig()
	if err != nil {
		return err
	}

	if config.RemoteURL == "" {
		fmt.Println("No remote URL configured. Use 'interop config remote add <url>' to set one.")
		return nil
	}

	fmt.Printf("Current remote URL: %s\n", config.RemoteURL)
	return nil
}

// Fetch is a placeholder for future implementation
func (m *Manager) Fetch() error {
	// Ensure remote config exists and load it
	if err := m.EnsureRemoteConfig(); err != nil {
		return err
	}

	config, err := m.loadRemoteConfig()
	if err != nil {
		return err
	}

	if config.RemoteURL == "" {
		return fmt.Errorf("no remote URL configured. Use 'interop config remote add <url>' to set one")
	}

	logging.Message("Fetching configuration from remote: %s", config.RemoteURL)

	// Load existing version info
	versionInfo, err := m.loadVersionInfo()
	if err != nil {
		return fmt.Errorf("failed to load version info: %w", err)
	}

	// Clone repository to temporary directory
	tmpDir, err := m.cloneRepository(config.RemoteURL)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir) // Clean up temporary directory

	// Get current commit ID
	currentCommit, err := m.runGitCommand(tmpDir, "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("failed to get current commit ID: %w", err)
	}

	// Check if we need to update (compare commit IDs)
	if versionInfo.LastCommit == currentCommit {
		commitShort := currentCommit
		if len(commitShort) > 8 {
			commitShort = commitShort[:8]
		}
		logging.Message("Remote configuration is up to date (commit: %s)", commitShort)
		return nil
	}

	lastCommitShort := versionInfo.LastCommit
	if len(lastCommitShort) > 8 {
		lastCommitShort = lastCommitShort[:8]
	} else if lastCommitShort == "" {
		lastCommitShort = "none"
	}

	currentCommitShort := currentCommit
	if len(currentCommitShort) > 8 {
		currentCommitShort = currentCommitShort[:8]
	}

	logging.Message("New changes detected. Last commit: %s, Current commit: %s",
		lastCommitShort, currentCommitShort)

	// Validate repository structure
	if err := m.validateRepoStructure(tmpDir); err != nil {
		return err
	}

	// Get remote configuration directories
	remoteConfigsDir, remoteExecutablesDir, err := m.getRemoteConfigDirs()
	if err != nil {
		return err
	}

	// Prepare new version info
	newVersionInfo := &VersionInfo{
		LastCommit: currentCommit,
		FileSHAs:   make(map[string]string),
	}

	// Sync config.d directory
	configSrcDir := filepath.Join(tmpDir, "config.d")
	logging.Message("Syncing config.d to %s", remoteConfigsDir)
	if err := m.syncDirectory(configSrcDir, remoteConfigsDir, newVersionInfo.FileSHAs, "config.d"); err != nil {
		return fmt.Errorf("failed to sync config.d: %w", err)
	}

	// Clean up removed files in config.d.remote
	if err := m.cleanupRemovedFiles(remoteConfigsDir, newVersionInfo.FileSHAs, "config.d"); err != nil {
		logging.Warning("Failed to cleanup removed config files: %v", err)
	}

	// Sync executables directory
	executablesSrcDir := filepath.Join(tmpDir, "executables")
	logging.Message("Syncing executables to %s", remoteExecutablesDir)
	if err := m.syncDirectory(executablesSrcDir, remoteExecutablesDir, newVersionInfo.FileSHAs, "executables"); err != nil {
		return fmt.Errorf("failed to sync executables: %w", err)
	}

	// Clean up removed files in executables.remote
	if err := m.cleanupRemovedFiles(remoteExecutablesDir, newVersionInfo.FileSHAs, "executables"); err != nil {
		logging.Warning("Failed to cleanup removed executable files: %v", err)
	}

	// Save updated version info
	if err := m.saveVersionInfo(newVersionInfo); err != nil {
		return fmt.Errorf("failed to save version info: %w", err)
	}

	logging.Message("Successfully fetched remote configuration")
	logging.Message("Total files: %d", len(newVersionInfo.FileSHAs))
	logging.Message("Current commit: %s", currentCommit)

	return nil
}

// getRemoteConfigDirs returns the paths to remote configuration directories
func (m *Manager) getRemoteConfigDirs() (string, string, error) {
	root, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	settingsDir := filepath.Join(root, m.configManager.PathConfig.SettingsDir)
	appDir := filepath.Join(settingsDir, m.configManager.PathConfig.AppDir)

	remoteConfigsDir := filepath.Join(appDir, "config.d.remote")
	remoteExecutablesDir := filepath.Join(appDir, "executables.remote")

	return remoteConfigsDir, remoteExecutablesDir, nil
}

// getVersionsPath returns the path to the versions.toml file
func (m *Manager) getVersionsPath() (string, error) {
	root, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	settingsDir := filepath.Join(root, m.configManager.PathConfig.SettingsDir)
	appDir := filepath.Join(settingsDir, m.configManager.PathConfig.AppDir)
	remoteDir := filepath.Join(appDir, m.configManager.PathConfig.RemoteDir)

	return filepath.Join(remoteDir, "versions.toml"), nil
}

// calculateFileSHA calculates the SHA256 hash of a file
func (m *Manager) calculateFileSHA(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to calculate SHA for %s: %w", filePath, err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// runGitCommand runs a git command in the specified directory
func (m *Manager) runGitCommand(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git command failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// cloneRepository clones the git repository to a temporary directory
func (m *Manager) cloneRepository(repoURL string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "interop-remote-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	logging.Message("Cloning repository %s to %s", repoURL, tmpDir)

	_, err = m.runGitCommand("", "clone", repoURL, tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	return tmpDir, nil
}

// validateRepoStructure validates that the repository has the required folder structure
func (m *Manager) validateRepoStructure(repoPath string) error {
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

// loadVersionInfo loads the version information from versions.toml
func (m *Manager) loadVersionInfo() (*VersionInfo, error) {
	versionsPath, err := m.getVersionsPath()
	if err != nil {
		return nil, err
	}

	// If file doesn't exist, return empty version info
	if _, err := os.Stat(versionsPath); os.IsNotExist(err) {
		return &VersionInfo{
			FileSHAs: make(map[string]string),
		}, nil
	}

	var versionInfo VersionInfo
	if _, err := toml.DecodeFile(versionsPath, &versionInfo); err != nil {
		return nil, fmt.Errorf("failed to decode versions file: %w", err)
	}

	if versionInfo.FileSHAs == nil {
		versionInfo.FileSHAs = make(map[string]string)
	}

	return &versionInfo, nil
}

// saveVersionInfo saves the version information to versions.toml
func (m *Manager) saveVersionInfo(versionInfo *VersionInfo) error {
	versionsPath, err := m.getVersionsPath()
	if err != nil {
		return err
	}

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(versionsPath), 0o755); err != nil {
		return fmt.Errorf("failed to create versions directory: %w", err)
	}

	f, err := os.Create(versionsPath)
	if err != nil {
		return fmt.Errorf("failed to create versions file: %w", err)
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(versionInfo); err != nil {
		return fmt.Errorf("failed to encode versions data: %w", err)
	}

	return nil
}

// copyFile copies a file from src to dst, preserving permissions
func (m *Manager) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer sourceFile.Close()

	// Get source file info for permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Preserve permissions
	if err := os.Chmod(dst, sourceInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}

// syncDirectory recursively syncs files from source to destination directory
func (m *Manager) syncDirectory(srcDir, dstDir string, currentSHAs map[string]string, relativePath string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dstDir, err)
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", srcDir, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())
		relativeFilePath := filepath.Join(relativePath, entry.Name())

		if entry.IsDir() {
			// Recursively sync subdirectories
			if err := m.syncDirectory(srcPath, dstPath, currentSHAs, relativeFilePath); err != nil {
				return err
			}
		} else {
			// Calculate SHA of source file
			srcSHA, err := m.calculateFileSHA(srcPath)
			if err != nil {
				return fmt.Errorf("failed to calculate SHA for %s: %w", srcPath, err)
			}

			// Check if file needs to be updated
			if existingSHA, exists := currentSHAs[relativeFilePath]; !exists || existingSHA != srcSHA {
				if err := m.copyFile(srcPath, dstPath); err != nil {
					return err
				}
				logging.Message("Updated file: %s", relativeFilePath)
			} else {
				logging.Message("File unchanged: %s", relativeFilePath)
			}

			// Update SHA in map
			currentSHAs[relativeFilePath] = srcSHA
		}
	}

	return nil
}

// cleanupRemovedFiles removes files that no longer exist in the source
func (m *Manager) cleanupRemovedFiles(dstDir string, newSHAs map[string]string, relativePath string) error {
	entries, err := os.ReadDir(dstDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist, nothing to clean
		}
		return fmt.Errorf("failed to read directory %s: %w", dstDir, err)
	}

	for _, entry := range entries {
		dstPath := filepath.Join(dstDir, entry.Name())
		relativeFilePath := filepath.Join(relativePath, entry.Name())

		if entry.IsDir() {
			// Recursively clean subdirectories
			if err := m.cleanupRemovedFiles(dstPath, newSHAs, relativeFilePath); err != nil {
				return err
			}

			// Remove directory if it's empty
			if isEmpty, err := m.isDirEmpty(dstPath); err == nil && isEmpty {
				if err := os.Remove(dstPath); err != nil {
					logging.Warning("Failed to remove empty directory %s: %v", dstPath, err)
				} else {
					logging.Message("Removed empty directory: %s", relativeFilePath)
				}
			}
		} else {
			// Remove file if it doesn't exist in new SHAs
			if _, exists := newSHAs[relativeFilePath]; !exists {
				if err := os.Remove(dstPath); err != nil {
					logging.Warning("Failed to remove file %s: %v", dstPath, err)
				} else {
					logging.Message("Removed file: %s", relativeFilePath)
				}
			}
		}
	}

	return nil
}

// isDirEmpty checks if a directory is empty
func (m *Manager) isDirEmpty(dirPath string) (bool, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}
