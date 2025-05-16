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

// MCPServer represents a configured MCP server with a name, description, and port
type MCPServer struct {
	Name        string `toml:"name"`
	Description string `toml:"description"`
	Port        int    `toml:"port"`
}

type Project struct {
	Path        string  `toml:"path"`
	Description string  `toml:"description,omitempty"`
	Commands    []Alias `toml:"commands,omitempty"`
}

// ArgumentType defines the type of a command argument
type ArgumentType string

const (
	// ArgumentTypeString represents a string argument
	ArgumentTypeString ArgumentType = "string"
	// ArgumentTypeNumber represents a numeric argument
	ArgumentTypeNumber ArgumentType = "number"
	// ArgumentTypeBool represents a boolean argument
	ArgumentTypeBool ArgumentType = "bool"
)

// CommandArgument represents an argument definition for a command
type CommandArgument struct {
	Name        string       `toml:"name"`                  // Argument name
	Type        ArgumentType `toml:"type,omitempty"`        // Argument type (string, number, bool)
	Description string       `toml:"description,omitempty"` // Description of the argument
	Required    bool         `toml:"required,omitempty"`    // Whether the argument is required
	Default     interface{}  `toml:"default,omitempty"`     // Default value if not provided
}

// CommandConfig represents a command that can be executed
type CommandConfig struct {
	Description  string            `toml:"description,omitempty"`
	IsEnabled    bool              `toml:"is_enabled"`
	Cmd          string            `toml:"cmd"`
	IsExecutable bool              `toml:"is_executable"`
	Arguments    []CommandArgument `toml:"arguments,omitempty"` // Argument definitions for the command
	MCP          string            `toml:"mcp,omitempty"`       // Optional MCP server name this command belongs to
}

// NewCommandConfig creates a new CommandConfig with default values
func NewCommandConfig() CommandConfig {
	return CommandConfig{
		IsEnabled:    true,
		IsExecutable: false,
		Arguments:    []CommandArgument{},
		MCP:          "",
	}
}

// UnmarshalTOML supports partial command definitions in the TOML settings file
// This allows having just the cmd field defined with other fields getting defaults
func (c *CommandConfig) UnmarshalTOML(data interface{}) error {
	// Set defaults first
	c.IsEnabled = true
	c.IsExecutable = false
	c.Description = ""
	c.Arguments = []CommandArgument{}
	c.MCP = ""

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
		if mcp, ok := v["mcp"].(string); ok {
			c.MCP = mcp
		}

		// Parse arguments if present
		if args, ok := v["arguments"].([]interface{}); ok {
			for _, arg := range args {
				if argMap, ok := arg.(map[string]interface{}); ok {
					argument := CommandArgument{}

					// Required fields
					if name, ok := argMap["name"].(string); ok {
						argument.Name = name
					} else {
						continue // Skip if no name
					}

					// Optional fields
					if typeStr, ok := argMap["type"].(string); ok {
						argument.Type = ArgumentType(typeStr)
					} else {
						argument.Type = ArgumentTypeString // Default to string
					}

					if desc, ok := argMap["description"].(string); ok {
						argument.Description = desc
					}

					if required, ok := argMap["required"].(bool); ok {
						argument.Required = required
					}

					if def, ok := argMap["default"]; ok {
						argument.Default = def
					}

					c.Arguments = append(c.Arguments, argument)
				}
			}
		}
	}
	return nil
}

// GetArgumentValue retrieves the value for an argument from the provided arguments map
// It will use the default value if the argument is not provided and has a default
// Returns an error if a required argument is missing
func (c *CommandConfig) GetArgumentValue(argName string, providedArgs map[string]interface{}) (interface{}, error) {
	// Look for the argument definition
	var argDef *CommandArgument
	for _, arg := range c.Arguments {
		if arg.Name == argName {
			argDef = &arg
			break
		}
	}

	// If no definition found, just return the provided value or nil
	if argDef == nil {
		if value, exists := providedArgs[argName]; exists {
			return value, nil
		}
		return nil, nil
	}

	// Check if the argument is provided
	if value, exists := providedArgs[argName]; exists {
		return value, nil
	}

	// If not provided, check if it's required
	if argDef.Required {
		return nil, fmt.Errorf("required argument '%s' is missing", argName)
	}

	// If not required, return the default value
	return argDef.Default, nil
}

// ValidateArgs checks if all required arguments are provided and all provided arguments are defined
// Returns an error if validation fails
func (c *CommandConfig) ValidateArgs(args map[string]interface{}) error {
	// Check if all required arguments are provided
	for _, arg := range c.Arguments {
		if arg.Required {
			if _, exists := args[arg.Name]; !exists {
				if arg.Default == nil {
					return fmt.Errorf("required argument '%s' is missing", arg.Name)
				}
			}
		}
	}

	// Check if all provided arguments are defined (if Arguments is not empty)
	if len(c.Arguments) > 0 {
		for name := range args {
			found := false
			for _, arg := range c.Arguments {
				if arg.Name == name {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("unknown argument '%s' provided", name)
			}
		}
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
	MCPPort               int                      `toml:"mcp_port"`
	MCPServers            map[string]MCPServer     `toml:"mcp_servers"`
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
			MCPPort:  cfg.MCPPort,
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

// ValidateMCPConfig validates the MCP configuration
// It checks:
// - top level mcp_port can't be the same as any MCP server port
// - can't have MCP servers with the same port or name
func ValidateMCPConfig(cfg *Settings) error {
	if cfg.MCPServers == nil {
		cfg.MCPServers = make(map[string]MCPServer)
		return nil
	}

	// Check for duplicates or conflicts with default port
	usedPorts := make(map[int]string)

	// First put the default port in the map
	if cfg.MCPPort > 0 {
		usedPorts[cfg.MCPPort] = "default MCP server"
	}

	// Now check all defined MCP servers
	for name, server := range cfg.MCPServers {
		// Validate required fields
		if server.Name == "" {
			return fmt.Errorf("MCP server '%s' must have a name", name)
		}

		if server.Port <= 0 {
			return fmt.Errorf("MCP server '%s' must have a valid port", name)
		}

		if server.Description == "" {
			return fmt.Errorf("MCP server '%s' must have a description", name)
		}

		// Check for port conflicts
		if existingServer, exists := usedPorts[server.Port]; exists {
			return fmt.Errorf("MCP server '%s' has port %d which conflicts with %s",
				name, server.Port, existingServer)
		}

		usedPorts[server.Port] = fmt.Sprintf("MCP server '%s'", name)

		// Ensure server.Name matches the key
		if server.Name != name {
			return fmt.Errorf("MCP server name '%s' doesn't match key '%s'", server.Name, name)
		}
	}

	// Check command MCP references
	for cmdName, cmd := range cfg.Commands {
		if cmd.MCP != "" {
			if _, exists := cfg.MCPServers[cmd.MCP]; !exists {
				return fmt.Errorf("command '%s' references non-existent MCP server '%s'",
					cmdName, cmd.MCP)
			}
		}
	}

	return nil
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

		// Provide default values for fields that might not be in the file
		if c.Projects == nil {
			c.Projects = make(map[string]Project)
		}
		if c.Commands == nil {
			c.Commands = make(map[string]CommandConfig)
		}
		if c.MCPServers == nil {
			c.MCPServers = make(map[string]MCPServer)
		}

		// Validate MCP configuration
		if err := ValidateMCPConfig(&c); err != nil {
			err = err
			logging.Error("Failed to validate MCP configuration: " + err.Error())
		}

		cfg = &c
	})
	return cfg, err
}

func GetMCPPort() int {
	cfg, err := Load()
	if err != nil {
		logging.Error("Failed to load settings: " + err.Error())
	}
	return cfg.MCPPort
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
