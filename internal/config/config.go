package config

import (
	"errors"
	"fmt"
	"interop/internal/logging"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// PathConfig defines the directory structure for settings
type PathConfig struct {
	SettingsDir    string
	AppDir         string
	CfgFile        string
	ExecutablesDir string
}

// DefaultPathConfig contains the default paths configuration
var DefaultPathConfig = PathConfig{
	SettingsDir:    ".config",
	AppDir:         "interop",
	CfgFile:        "settings.toml",
	ExecutablesDir: "executables",
}

// Manager handles configuration file operations
type Manager struct {
	PathConfig PathConfig
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	return &Manager{
		PathConfig: DefaultPathConfig,
	}
}

// WithCustomPath creates a new configuration manager with custom path configuration
func WithCustomPath(pathConfig PathConfig) *Manager {
	return &Manager{
		PathConfig: pathConfig,
	}
}

// GetConfigFilePath returns the path to the configuration file
func (m *Manager) GetConfigFilePath() (string, error) {
	root, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	config := filepath.Join(root, m.PathConfig.SettingsDir)
	base := filepath.Join(config, m.PathConfig.AppDir)
	path := filepath.Join(base, m.PathConfig.CfgFile)

	return path, nil
}

// EnsureConfigDirectories creates the necessary directories for the configuration
func (m *Manager) EnsureConfigDirectories() error {
	root, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	config := filepath.Join(root, m.PathConfig.SettingsDir)
	base := filepath.Join(config, m.PathConfig.AppDir)

	if err := os.MkdirAll(base, 0o755); err != nil {
		return fmt.Errorf("failed to create settings directory: %w", err)
	}
	logging.Message("Settings directory is created")

	// Create executables directory with executable permissions
	execDir := filepath.Join(base, m.PathConfig.ExecutablesDir)
	if err := os.MkdirAll(execDir, 0o755); err != nil {
		return fmt.Errorf("failed to create executables directory: %w", err)
	}
	logging.Message("Executables directory is created")

	return nil
}

// ParseFromFile parses the configuration file into the provided data structure
func (m *Manager) ParseFromFile(path string, data interface{}) error {
	_, err := toml.DecodeFile(path, data)
	if err != nil {
		return fmt.Errorf("failed to decode configuration file: %w", err)
	}
	return nil
}

// SaveToFile saves the provided data structure to the configuration file
func (m *Manager) SaveToFile(path string, data interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create configuration file: %w", err)
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(data); err != nil {
		return fmt.Errorf("failed to encode configuration data: %w", err)
	}

	return nil
}

// EnsureConfigFile creates the configuration file with default settings if it doesn't exist
func (m *Manager) EnsureConfigFile(defaultConfig interface{}) (string, error) {
	if err := m.EnsureConfigDirectories(); err != nil {
		return "", err
	}

	path, err := m.GetConfigFilePath()
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := m.SaveToFile(path, defaultConfig); err != nil {
			return "", err
		}
		logging.Message("Created default configuration file at %s", path)
	}

	return path, nil
}
