package settings

import (
	"context"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
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
	LogLevel string             `toml:"log_level"`
	Projects map[string]Project `toml:"projects"`
}

const (
	settingsDir    = ".config"
	appDir         = "interop"
	cfgFile        = "settings.toml"
	executablesDir = "executables"
)

var (
	once sync.Once
	cfg  *Settings
	err  error
)

// validate() guarantees ~/.settings/interop/settings.toml exists and
// returns its absolute path.
func validate() (string, error) {
	root, e := os.UserHomeDir()
	if e != nil {
		util.Error("Failed to get user home directory: " + e.Error())
	}
	config := filepath.Join(root, settingsDir)
	base := filepath.Join(config, appDir)
	path := filepath.Join(base, cfgFile)

	// ensure ~/.settings/interop
	if e := os.MkdirAll(base, 0o755); e != nil {
		util.Error("Can't create the directory for settings: " + e.Error())
	}

	// seed default file on first run
	if _, e := os.Stat(path); errors.Is(e, os.ErrNotExist) {
		def := Settings{LogLevel: "error", Projects: map[string]Project{}}
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

		// Validate project paths
		if len(c.Projects) > 0 {
			homeDir, e := os.UserHomeDir()
			if e != nil {
				err = e
				util.Error("Failed to get user home directory: " + e.Error())
			}

			for name, project := range c.Projects {
				// Check if path is absolute and outside home directory
				if filepath.IsAbs(project.Path) && !filepath.HasPrefix(project.Path, homeDir) {
					errMsg := fmt.Sprintf("project '%s' path must be inside $HOME: %s", name, project.Path)
					util.Error(errMsg)
					continue
				}

				projectPath := filepath.Join(homeDir, project.Path)
				if _, e := os.Stat(projectPath); os.IsNotExist(e) {
					errMsg := fmt.Sprintf("project '%s' path does not exist: %s", name, projectPath)
					util.Error(errMsg)
				}
			}
		}

		cfg = &c
		// Initialize the default logger with the loaded log level
		util.SetDefaultLogLevel(c.LogLevel)
	})
	return cfg, err
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
