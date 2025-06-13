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

// RemoteEntry represents a single remote repository configuration
type RemoteEntry struct {
	Name string `toml:"name"`
	URL  string `toml:"url"`
}

// RemoteConfig represents the remote configuration stored in remote.toml
type RemoteConfig struct {
	Remotes []RemoteEntry `toml:"remotes"`
}

// VersionInfo represents file version tracking information
type VersionInfo struct {
	LastCommit string            `toml:"last-commit"`
	FileSHAs   map[string]string `toml:"file-shas"`
	RemoteName string            `toml:"remote-name"` // Track which remote this version info belongs to
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
		emptyConfig := RemoteConfig{
			Remotes: []RemoteEntry{},
		}
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

	// Initialize remotes slice if nil
	if remoteConfig.Remotes == nil {
		remoteConfig.Remotes = []RemoteEntry{}
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

// findRemoteByName finds a remote entry by name
func (m *Manager) findRemoteByName(config *RemoteConfig, name string) (*RemoteEntry, int) {
	for i, remote := range config.Remotes {
		if remote.Name == name {
			return &remote, i
		}
	}
	return nil, -1
}

// Add adds a named remote URL to the configuration
func (m *Manager) Add(name, url string) error {
	if name == "" {
		return fmt.Errorf("remote name cannot be empty")
	}
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

	// Load existing config
	config, err := m.loadRemoteConfig()
	if err != nil {
		return err
	}

	// Check if remote name already exists
	if existing, _ := m.findRemoteByName(config, name); existing != nil {
		return fmt.Errorf("remote '%s' already exists with URL: %s", name, existing.URL)
	}

	// Add new remote
	config.Remotes = append(config.Remotes, RemoteEntry{
		Name: name,
		URL:  url,
	})

	if err := m.saveRemoteConfig(config); err != nil {
		return err
	}

	logging.Info("Added remote '%s' with URL: %s", name, url)
	return nil
}

// Remove removes a named remote from the configuration
func (m *Manager) Remove(name string) error {
	if name == "" {
		return fmt.Errorf("remote name cannot be empty")
	}

	// Ensure remote config exists
	if err := m.EnsureRemoteConfig(); err != nil {
		return err
	}

	// Load existing config
	config, err := m.loadRemoteConfig()
	if err != nil {
		return err
	}

	// Find and remove the remote
	_, index := m.findRemoteByName(config, name)
	if index == -1 {
		return fmt.Errorf("remote '%s' not found", name)
	}

	// Remove from slice
	config.Remotes = append(config.Remotes[:index], config.Remotes[index+1:]...)

	if err := m.saveRemoteConfig(config); err != nil {
		return err
	}

	// Also remove the version tracking file for this remote
	if err := m.removeVersionInfo(name); err != nil {
		logging.Warning("Failed to remove version info for remote '%s': %v", name, err)
	}

	logging.Info("Removed remote '%s'", name)
	return nil
}

// Show displays all configured remotes
func (m *Manager) Show() error {
	// Ensure remote config exists
	if err := m.EnsureRemoteConfig(); err != nil {
		return err
	}

	config, err := m.loadRemoteConfig()
	if err != nil {
		return err
	}

	fmt.Println("Remote Configurations:")
	fmt.Println("======================")
	fmt.Println()

	if len(config.Remotes) == 0 {
		fmt.Println("No remote repositories configured.")
		fmt.Println()
		fmt.Println("Add a remote with:")
		fmt.Println("  interop config remote add <name> <git-url>")
		logging.Info("No remote repositories configured.")
		return nil
	}

	for _, remote := range config.Remotes {
		fmt.Printf("🔗 %s\n", remote.Name)
		fmt.Printf("   URL: %s\n", remote.URL)

		// Validate URL and show status
		if err := m.validateGitURL(remote.URL); err != nil {
			fmt.Printf("   Status: ❌ Invalid Git URL: %v\n", err)
		} else {
			fmt.Printf("   Status: ✓ Valid Git URL\n")
		}
		fmt.Println()
	}

	return nil
}

// Fetch fetches configurations from remotes (all or specific named remote)
func (m *Manager) Fetch(remoteName string) error {
	// Ensure remote config exists
	if err := m.EnsureRemoteConfig(); err != nil {
		return err
	}

	config, err := m.loadRemoteConfig()
	if err != nil {
		return err
	}

	if len(config.Remotes) == 0 {
		return fmt.Errorf("no remote repositories configured")
	}

	var remotesToFetch []RemoteEntry

	if remoteName != "" {
		// Fetch specific remote
		remote, _ := m.findRemoteByName(config, remoteName)
		if remote == nil {
			return fmt.Errorf("remote '%s' not found", remoteName)
		}
		remotesToFetch = []RemoteEntry{*remote}
	} else {
		// Fetch all remotes
		remotesToFetch = config.Remotes
	}

	for _, remote := range remotesToFetch {
		logging.Message("Fetching from remote '%s' (%s)...", remote.Name, remote.URL)
		if err := m.fetchFromRemote(remote); err != nil {
			logging.Error("Failed to fetch from remote '%s': %v", remote.Name, err)
			continue
		}
		logging.Message("Successfully fetched from remote '%s'", remote.Name)
	}

	logging.Info("Fetch operation completed.")
	return nil
}

// fetchFromRemote fetches from a specific remote
func (m *Manager) fetchFromRemote(remote RemoteEntry) error {
	// Clone repository to temporary directory
	tmpDir, err := m.cloneRepository(remote.URL)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Validate repository structure
	if err := m.validateRepoStructure(tmpDir); err != nil {
		return fmt.Errorf("invalid repository structure: %w", err)
	}

	// Get current commit ID
	currentCommit, err := m.runGitCommand(tmpDir, "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("failed to get current commit: %w", err)
	}
	currentCommit = strings.TrimSpace(currentCommit)

	// Load existing version info for this remote
	versionInfo, err := m.loadVersionInfoForRemote(remote.Name)
	if err != nil {
		// If version info doesn't exist, create new one
		versionInfo = &VersionInfo{
			FileSHAs:   make(map[string]string),
			RemoteName: remote.Name,
		}
	}

	// Check if we need to update (commit changed or no previous version info)
	if versionInfo.LastCommit == currentCommit && len(versionInfo.FileSHAs) > 0 {
		logging.Message("Remote '%s' is already up to date (commit: %s)", remote.Name, currentCommit[:8])
		return nil
	}

	logging.Message("Updating from remote '%s' (commit: %s)", remote.Name, currentCommit[:8])

	// Get remote directories
	remoteConfigDir, remoteExecutablesDir, err := m.getRemoteConfigDirs()
	if err != nil {
		return err
	}

	// Track all current SHAs for cleanup
	allCurrentSHAs := make(map[string]string)

	// Sync config.d directory if it exists
	srcConfigDir := filepath.Join(tmpDir, "config.d")
	if _, err := os.Stat(srcConfigDir); err == nil {
		if err := os.MkdirAll(remoteConfigDir, 0755); err != nil {
			return fmt.Errorf("failed to create remote config directory: %w", err)
		}

		newSHAs := make(map[string]string)
		if err := m.syncDirectory(srcConfigDir, remoteConfigDir, versionInfo.FileSHAs, "config.d"); err != nil {
			return fmt.Errorf("failed to sync config directory: %w", err)
		}

		if err := m.updateSHAsForDirectory(remoteConfigDir, newSHAs, "config.d"); err != nil {
			return fmt.Errorf("failed to update SHAs for config directory: %w", err)
		}

		for path, sha := range newSHAs {
			versionInfo.FileSHAs[path] = sha
			allCurrentSHAs[path] = sha
		}
	}

	// Sync executables directory if it exists
	srcExecutablesDir := filepath.Join(tmpDir, "executables")
	if _, err := os.Stat(srcExecutablesDir); err == nil {
		if err := os.MkdirAll(remoteExecutablesDir, 0755); err != nil {
			return fmt.Errorf("failed to create remote executables directory: %w", err)
		}

		newSHAs := make(map[string]string)
		if err := m.syncDirectory(srcExecutablesDir, remoteExecutablesDir, versionInfo.FileSHAs, "executables"); err != nil {
			return fmt.Errorf("failed to sync executables directory: %w", err)
		}

		if err := m.updateSHAsForDirectory(remoteExecutablesDir, newSHAs, "executables"); err != nil {
			return fmt.Errorf("failed to update SHAs for executables directory: %w", err)
		}

		for path, sha := range newSHAs {
			versionInfo.FileSHAs[path] = sha
			allCurrentSHAs[path] = sha
		}
	}

	// Clean up files that were removed from remote
	if err := m.cleanupRemovedFiles(remoteConfigDir, allCurrentSHAs, "config.d"); err != nil {
		logging.Warning("Failed to cleanup removed config files: %v", err)
	}
	if err := m.cleanupRemovedFiles(remoteExecutablesDir, allCurrentSHAs, "executables"); err != nil {
		logging.Warning("Failed to cleanup removed executable files: %v", err)
	}

	// Remove stale SHAs for files that no longer exist
	for path := range versionInfo.FileSHAs {
		if _, exists := allCurrentSHAs[path]; !exists {
			delete(versionInfo.FileSHAs, path)
		}
	}

	// Update version info
	versionInfo.LastCommit = currentCommit
	if err := m.saveVersionInfoForRemote(remote.Name, versionInfo); err != nil {
		return fmt.Errorf("failed to save version info: %w", err)
	}

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

// Clear removes all remote configuration files and resets tracking information
func (m *Manager) Clear() error {
	// Get remote configuration directories
	remoteConfigsDir, remoteExecutablesDir, err := m.getRemoteConfigDirs()
	if err != nil {
		return err
	}

	removedItems := 0

	// Remove config.d.remote directory
	if _, err := os.Stat(remoteConfigsDir); err == nil {
		if err := os.RemoveAll(remoteConfigsDir); err != nil {
			return fmt.Errorf("failed to remove remote config directory: %w", err)
		}
		logging.Message("Removed remote config directory: %s", remoteConfigsDir)
		removedItems++
	}

	// Remove executables.remote directory
	if _, err := os.Stat(remoteExecutablesDir); err == nil {
		if err := os.RemoveAll(remoteExecutablesDir); err != nil {
			return fmt.Errorf("failed to remove remote executables directory: %w", err)
		}
		logging.Message("Removed remote executables directory: %s", remoteExecutablesDir)
		removedItems++
	}

	// Remove all version tracking files for named remotes
	root, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	settingsDir := filepath.Join(root, m.configManager.PathConfig.SettingsDir)
	appDir := filepath.Join(settingsDir, m.configManager.PathConfig.AppDir)
	remoteDir := filepath.Join(appDir, m.configManager.PathConfig.RemoteDir)

	// Remove all versions-*.toml files
	if _, err := os.Stat(remoteDir); err == nil {
		entries, err := os.ReadDir(remoteDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasPrefix(entry.Name(), "versions-") && strings.HasSuffix(entry.Name(), ".toml") {
					versionFilePath := filepath.Join(remoteDir, entry.Name())
					if err := os.Remove(versionFilePath); err != nil {
						logging.Warning("Failed to remove version file %s: %v", versionFilePath, err)
					} else {
						logging.Message("Removed version tracking file: %s", versionFilePath)
						removedItems++
					}
				}
			}
		}
	}

	// Remove legacy versions.toml file if it exists
	legacyVersionsPath, err := m.getVersionsPath()
	if err == nil {
		if _, err := os.Stat(legacyVersionsPath); err == nil {
			if err := os.Remove(legacyVersionsPath); err != nil {
				logging.Warning("Failed to remove legacy versions file: %v", err)
			} else {
				logging.Message("Removed legacy versions tracking file: %s", legacyVersionsPath)
				removedItems++
			}
		}
	}

	if removedItems == 0 {
		logging.Message("No remote files to clear")
	} else {
		logging.Info("Successfully cleared %d remote items", removedItems)
	}

	return nil
}

// getVersionsPathForRemote returns the path to the versions file for a specific remote
func (m *Manager) getVersionsPathForRemote(remoteName string) (string, error) {
	root, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	settingsDir := filepath.Join(root, m.configManager.PathConfig.SettingsDir)
	appDir := filepath.Join(settingsDir, m.configManager.PathConfig.AppDir)
	remoteDir := filepath.Join(appDir, m.configManager.PathConfig.RemoteDir)

	return filepath.Join(remoteDir, fmt.Sprintf("versions-%s.toml", remoteName)), nil
}

// loadVersionInfoForRemote loads the version information for a specific remote
func (m *Manager) loadVersionInfoForRemote(remoteName string) (*VersionInfo, error) {
	versionsPath, err := m.getVersionsPathForRemote(remoteName)
	if err != nil {
		return nil, err
	}

	// If file doesn't exist, return empty version info
	if _, err := os.Stat(versionsPath); os.IsNotExist(err) {
		return &VersionInfo{
			FileSHAs:   make(map[string]string),
			RemoteName: remoteName,
		}, nil
	}

	var versionInfo VersionInfo
	if _, err := toml.DecodeFile(versionsPath, &versionInfo); err != nil {
		return nil, fmt.Errorf("failed to decode versions file for remote '%s': %w", remoteName, err)
	}

	if versionInfo.FileSHAs == nil {
		versionInfo.FileSHAs = make(map[string]string)
	}
	versionInfo.RemoteName = remoteName

	return &versionInfo, nil
}

// saveVersionInfoForRemote saves the version information for a specific remote
func (m *Manager) saveVersionInfoForRemote(remoteName string, versionInfo *VersionInfo) error {
	versionsPath, err := m.getVersionsPathForRemote(remoteName)
	if err != nil {
		return err
	}

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(versionsPath), 0o755); err != nil {
		return fmt.Errorf("failed to create versions directory: %w", err)
	}

	f, err := os.Create(versionsPath)
	if err != nil {
		return fmt.Errorf("failed to create versions file for remote '%s': %w", remoteName, err)
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(versionInfo); err != nil {
		return fmt.Errorf("failed to encode versions data for remote '%s': %w", remoteName, err)
	}

	return nil
}

// removeVersionInfo removes the version tracking file for a specific remote
func (m *Manager) removeVersionInfo(remoteName string) error {
	versionsPath, err := m.getVersionsPathForRemote(remoteName)
	if err != nil {
		return err
	}

	if _, err := os.Stat(versionsPath); os.IsNotExist(err) {
		// File doesn't exist, nothing to remove
		return nil
	}

	if err := os.Remove(versionsPath); err != nil {
		return fmt.Errorf("failed to remove versions file for remote '%s': %w", remoteName, err)
	}

	return nil
}

// updateSHAsForDirectory calculates and updates SHAs for all files in a directory
func (m *Manager) updateSHAsForDirectory(dirPath string, shas map[string]string, relativePath string) error {
	return filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Calculate relative path from the base directory
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Create the key for the SHA map
		key := filepath.Join(relativePath, relPath)
		key = filepath.ToSlash(key) // Normalize path separators

		// Calculate SHA for the file
		sha, err := m.calculateFileSHA(path)
		if err != nil {
			return fmt.Errorf("failed to calculate SHA for %s: %w", path, err)
		}

		shas[key] = sha
		return nil
	})
}
