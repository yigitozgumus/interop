package settings

import (
	"context"
	"errors"
	"fmt"
	"interop/internal/logging"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
)

type Alias struct {
	CommandName string `toml:"command_name"`
	Alias       string `toml:"alias,omitempty"`
}

type Project struct {
	Path        string  `toml:"path"`
	Description string  `toml:"description,omitempty"`
	Commands    []Alias `toml:"commands,omitempty"`
}

// CommandConfig represents a command that can be executed
type CommandConfig struct {
	Description  string `toml:"description,omitempty"`
	IsEnabled    bool   `toml:"is_enabled"`
	Cmd          string `toml:"cmd"`
	IsExecutable bool   `toml:"is_executable"`
}

// NewCommandConfig creates a new CommandConfig with default values
func NewCommandConfig() CommandConfig {
	return CommandConfig{
		IsEnabled:    true,
		IsExecutable: false,
	}
}

// UnmarshalTOML supports partial command definitions in the TOML settings file
// This allows having just the cmd field defined with other fields getting defaults
func (c *CommandConfig) UnmarshalTOML(data interface{}) error {
	// Set defaults first
	c.IsEnabled = true
	c.IsExecutable = false
	c.Description = ""

	// Handle different input cases
	switch v := data.(type) {
	case string:
		// If the command is specified as just a string, use it as cmd
		c.Cmd = v
	case map[string]interface{}:
		// If a field is present, use its value
		if cmd, ok := v["cmd"].(string); ok {
			c.Cmd = cmd
		}
		if desc, ok := v["description"].(string); ok {
			c.Description = desc
		}
		c.IsEnabled = getBoolWithDefault(v, "is_enabled", true)
		c.IsExecutable = getBoolWithDefault(v, "is_executable", false)
	}
	return nil
}

// Helper function to get a boolean value with a default
func getBoolWithDefault(m map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return defaultValue
}

type Settings struct {
	LogLevel              string                   `toml:"log_level"`
	Projects              map[string]Project       `toml:"projects"`
	Commands              map[string]CommandConfig `toml:"commands"`
	ExecutableSearchPaths []string                 `toml:"executable_search_paths"`
}

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

var (
	once       sync.Once
	cfg        *Settings
	err        error
	pathConfig = DefaultPathConfig
)

// SetPathConfig allows overriding the default path configuration
// Useful for testing
func SetPathConfig(config PathConfig) {
	pathConfig = config
	// Reset singleton to reload with new config
	once = sync.Once{}
	cfg = nil
	err = nil
}

// validate() guarantees ~/.settings/interop/settings.toml exists and
// returns its absolute path.
func validate() (string, error) {
	root, e := os.UserHomeDir()
	if e != nil {
		logging.Error("Failed to get user home directory: " + e.Error())
	}
	config := filepath.Join(root, pathConfig.SettingsDir)
	base := filepath.Join(config, pathConfig.AppDir)
	path := filepath.Join(base, pathConfig.CfgFile)

	if e := os.MkdirAll(base, 0o755); e != nil {
		logging.Error("Can't create the directory for settings: " + e.Error())
	} else {
		logging.Message("Settings directory is created")
	}

	// Create executables directory with executable permissions
	execDir := filepath.Join(base, pathConfig.ExecutablesDir)
	if e := os.MkdirAll(execDir, 0o755); e != nil {
		logging.Error("Can't create the directory for executables: " + e.Error())
	} else {
		logging.Message("executables directory is created")
	}

	if _, e := os.Stat(path); errors.Is(e, os.ErrNotExist) {
		def := Settings{
			LogLevel: "warning",
			Projects: map[string]Project{},
			Commands: map[string]CommandConfig{},
		}
		f, e := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
		if e != nil {
			logging.Error("Failed to create settings file: " + e.Error())
		}
		if e := toml.NewEncoder(f).Encode(def); e != nil {
			logging.Error("Failed to encode default settings: " + e.Error())
		}
		if e := f.Close(); e != nil {
			logging.Error("Failed to close settings file: " + e.Error())
		}
	}
	return path, nil
}

// Load parses settings.toml once.
func Load() (*Settings, error) {
	once.Do(func() {
		path, e := validate()
		if e != nil {
			err = e
			logging.Error("Failed to validate settings: " + e.Error())
		}
		var c Settings
		if _, e := toml.DecodeFile(path, &c); e != nil {
			err = e
			logging.Error("Failed to decode settings file: " + e.Error())
		}
		logging.SetDefaultLevelFromString(c.LogLevel)

		if len(c.Projects) > 0 {
			homeDir, e := os.UserHomeDir()
			if e != nil {
				err = e
				logging.Error("Failed to get user home directory: " + e.Error())
			}

			for name, project := range c.Projects {
				// Handle path with tilde expansion
				projectPath := project.Path

				// Handle tilde expansion for home directory
				if strings.HasPrefix(projectPath, "~/") && homeDir != "" {
					projectPath = filepath.Join(homeDir, projectPath[2:])
				} else if !filepath.IsAbs(projectPath) {
					projectPath = filepath.Join(homeDir, projectPath)
				}

				if filepath.IsAbs(project.Path) && !filepath.HasPrefix(project.Path, homeDir) {
					errMsg := fmt.Sprintf("project '%s' path must be inside $HOME: %s", name, project.Path)
					logging.Warning(errMsg)
					continue
				}

				if _, e := os.Stat(projectPath); os.IsNotExist(e) {
					errMsg := fmt.Sprintf("project '%s' path does not exist: %s", name, projectPath)
					logging.Warning(errMsg)
				}
			}
			logging.Message("Projects are validated")
		}

		cfg = &c
	})
	return cfg, err
}

// GetExecutablesPath returns the path to the executables directory
func GetExecutablesPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(
		homeDir,
		DefaultPathConfig.SettingsDir,
		DefaultPathConfig.AppDir,
		DefaultPathConfig.ExecutablesDir,
	), nil
}

// GetExecutableSearchPaths returns all paths to search for executables
// This includes the default executables path and any additional paths from config
func GetExecutableSearchPaths(cfg *Settings) ([]string, error) {
	// Start with the default executables path
	defaultPath, err := GetExecutablesPath()
	if err != nil {
		return nil, err
	}

	paths := []string{defaultPath}

	// Add user-configured paths
	for _, path := range cfg.ExecutableSearchPaths {
		// Handle tilde expansion for home directory
		if strings.HasPrefix(path, "~/") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				logging.Warning("Failed to get home directory for path expansion: %v", err)
				continue
			}
			path = filepath.Join(homeDir, path[2:])
		}

		// Add the path if it exists
		if _, err := os.Stat(path); err == nil {
			paths = append(paths, path)
		} else {
			logging.Warning("Executable search path not found: %s", path)
		}
	}

	return paths, nil
}

// ParseStringSlice parses a TOML value into a string slice
func ParseStringSlice(value interface{}) []string {
	if value == nil {
		return []string{}
	}

	switch v := value.(type) {
	case []interface{}:
		result := make([]string, len(v))
		for i, item := range v {
			if s, ok := item.(string); ok {
				result[i] = s
			}
		}
		return result
	case string:
		// Single string case
		return []string{v}
	}

	return []string{}
}

// GetProjectCommands returns the list of commands associated with a project
func GetProjectCommands(cfg *Settings, projectName string) (map[string]CommandConfig, error) {
	project, exists := cfg.Projects[projectName]
	if !exists {
		return nil, fmt.Errorf("project '%s' not found", projectName)
	}

	result := make(map[string]CommandConfig)

	// If no commands are defined for the project, return empty map
	if len(project.Commands) == 0 {
		return result, nil
	}

	// Collect all commands that are listed in the project
	for _, alias := range project.Commands {
		cmd, exists := cfg.Commands[alias.CommandName]
		if exists {
			// Use the alias if provided, otherwise use the original command name
			cmdKey := alias.CommandName
			if alias.Alias != "" {
				cmdKey = alias.Alias
			}
			result[cmdKey] = cmd
		}
	}

	return result, nil
}

// Get returns the settings
func Get() *Settings {
	c, e := Load()
	if e != nil {
		logging.Error("config load: " + e.Error())
	}
	return c
}

type ctxKey struct{}

func With(ctx context.Context) (context.Context, error) {
	c, e := Load()
	if e != nil {
		return ctx, e
	}
	return context.WithValue(ctx, ctxKey{}, c), nil
}

func From(ctx context.Context) *Settings {
	if v := ctx.Value(ctxKey{}); v != nil {
		if c, ok := v.(*Settings); ok {
			return c
		}
	}
	return Get()
}
