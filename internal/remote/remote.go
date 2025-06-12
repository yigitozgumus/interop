package remote

import (
	"fmt"
	"interop/internal/config"
	"interop/internal/logging"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// RemoteConfig represents the remote configuration stored in remote.toml
type RemoteConfig struct {
	RemoteURL string `toml:"remote-url"`
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
	fmt.Println("We will implement this later")
	return nil
}
