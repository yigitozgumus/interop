package settings

import (
	"context"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"interop/internal/command"
	"interop/internal/util"
	"os"
	"path/filepath"
	"sync"
)

type Project struct {
	Path        string `toml:"path"`
	Description string `toml:"description,omitempty"`
}

type Settings struct {
	LogLevel string                     `toml:"log_level"`
	Projects map[string]Project         `toml:"projects"`
	Commands map[string]command.Command `toml:"commands"`
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
		util.Error("Failed to get user home directory: " + e.Error())
	}
	config := filepath.Join(root, pathConfig.SettingsDir)
	base := filepath.Join(config, pathConfig.AppDir)
	path := filepath.Join(base, pathConfig.CfgFile)

	if e := os.MkdirAll(base, 0o755); e != nil {
		util.Error("Can't create the directory for settings: " + e.Error())
	} else {
		util.Message("Settings directory is created")
	}

	// Create executables directory with executable permissions
	execDir := filepath.Join(base, pathConfig.ExecutablesDir)
	if e := os.MkdirAll(execDir, 0o755); e != nil {
		util.Error("Can't create the directory for executables: " + e.Error())
	} else {
		util.Message("executables directory is created")
	}

	if _, e := os.Stat(path); errors.Is(e, os.ErrNotExist) {
		def := Settings{
			LogLevel: "warning",
			Projects: map[string]Project{},
			Commands: map[string]command.Command{},
		}
		f, e := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
		if e != nil {
			util.Error("Failed to create settings file: " + e.Error())
		}
		if e := toml.NewEncoder(f).Encode(def); e != nil {
			util.Error("Failed to encode default settings: " + e.Error())
		}
		if e := f.Close(); e != nil {
			util.Error("Failed to close settings file: " + e.Error())
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
			util.Error("Failed to validate settings: " + e.Error())
		}
		var c Settings
		if _, e := toml.DecodeFile(path, &c); e != nil {
			err = e
			util.Error("Failed to decode settings file: " + e.Error())
		}
		util.SetDefaultLogLevel(c.LogLevel)

		if len(c.Projects) > 0 {
			homeDir, e := os.UserHomeDir()
			if e != nil {
				err = e
				util.Error("Failed to get user home directory: " + e.Error())
			}

			for name, project := range c.Projects {
				if filepath.IsAbs(project.Path) && !filepath.HasPrefix(project.Path, homeDir) {
					errMsg := fmt.Sprintf("project '%s' path must be inside $HOME: %s", name, project.Path)
					util.Warning(errMsg)
					continue
				}

				projectPath := filepath.Join(homeDir, project.Path)
				if _, e := os.Stat(projectPath); os.IsNotExist(e) {
					errMsg := fmt.Sprintf("project '%s' path does not exist: %s", name, projectPath)
					util.Warning(errMsg)
				}
			}
			util.Message("Projects are validated")
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

// ------- convenience helpers ---------

func Get() *Settings {
	c, e := Load()
	if e != nil {
		util.Error("config load: " + e.Error())
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
