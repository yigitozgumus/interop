package settings

import (
	"context"
	"errors"
	"github.com/BurntSushi/toml"
	"interop/internal/util"
	"os"
	"path/filepath"
	"sync"
)

type Settings struct {
	LogLevel string `toml:"log_level"`
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
		return "", e
	}
	config := filepath.Join(root, settingsDir)
	base := filepath.Join(config, appDir)
	path := filepath.Join(base, cfgFile)

	// ensure ~/.settings/interop
	if e := os.MkdirAll(base, 0o755); e != nil {
		util.Error("Can't create the directory for settings.")
		return "", e
	}

	// seed default file on first run
	if _, e := os.Stat(path); errors.Is(e, os.ErrNotExist) {
		def := Settings{LogLevel: "error"}
		f, e := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
		if e != nil {
			return "", e
		}
		_ = toml.NewEncoder(f).Encode(def) // ignore encode err on bootstrap
		_ = f.Close()
	}
	return path, nil
}

// Load parses settings.toml once.
func Load() (*Settings, error) {
	once.Do(func() {
		path, e := validate()
		if e != nil {
			err = e
			return
		}
		var c Settings
		if _, e := toml.DecodeFile(path, &c); e != nil {
			err = e
			return
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
