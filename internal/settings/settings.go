package settings

import (
	"context"
	"errors"
	"fmt"
	"interop/internal/logging"
	"os"
	"path/filepath"
	"sort"
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
	Path        string            `toml:"path"`
	Description string            `toml:"description,omitempty"`
	Commands    []Alias           `toml:"commands,omitempty"`
	Env         map[string]string `toml:"env,omitempty"`
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
	Prefix      string       `toml:"prefix,omitempty"`      // Prefix to use for the argument (e.g. "--keys")
}

// CommandExample represents an example of how to use a command
type CommandExample struct {
	Description string `toml:"description"` // Description of what this example does
	Command     string `toml:"command"`     // Example command invocation
}

// CommandConfig represents a command that can be executed
type CommandConfig struct {
	Description  string            `toml:"description,omitempty"`
	IsEnabled    bool              `toml:"is_enabled"`
	Cmd          string            `toml:"cmd"`
	IsExecutable bool              `toml:"is_executable"`
	PreExec      []string          `toml:"pre_exec,omitempty"`  // Commands to run before the main command
	PostExec     []string          `toml:"post_exec,omitempty"` // Commands to run after the main command
	Arguments    []CommandArgument `toml:"arguments,omitempty"` // Argument definitions for the command
	MCP          string            `toml:"mcp,omitempty"`       // Optional MCP server name this command belongs to
	Version      string            `toml:"version,omitempty"`   // Version of the command
	Examples     []CommandExample  `toml:"examples,omitempty"`  // Usage examples for the command
	Env          map[string]string `toml:"env,omitempty"`       // Environment variables for the command
}

// NewCommandConfig creates a new CommandConfig with default values
func NewCommandConfig() CommandConfig {
	return CommandConfig{
		IsEnabled:    true,
		IsExecutable: false,
		PreExec:      []string{},
		PostExec:     []string{},
		Arguments:    []CommandArgument{},
		MCP:          "",
		Version:      "",
		Examples:     []CommandExample{},
		Env:          make(map[string]string),
	}
}

// UnmarshalTOML supports partial command definitions in the TOML settings file
// This allows having just the cmd field defined with other fields getting defaults
func (c *CommandConfig) UnmarshalTOML(data interface{}) error {
	// Set defaults first
	c.IsEnabled = true
	c.IsExecutable = false
	c.Description = ""
	c.PreExec = []string{}
	c.PostExec = []string{}
	c.Arguments = []CommandArgument{}
	c.MCP = ""
	c.Version = ""
	c.Examples = []CommandExample{}
	c.Env = make(map[string]string)

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
		if version, ok := v["version"].(string); ok {
			c.Version = version
		}

		// Parse pre_exec commands if present
		if preExec, ok := v["pre_exec"].([]interface{}); ok {
			for _, cmd := range preExec {
				if cmdStr, ok := cmd.(string); ok {
					c.PreExec = append(c.PreExec, cmdStr)
				}
			}
		}

		// Parse post_exec commands if present
		if postExec, ok := v["post_exec"].([]interface{}); ok {
			for _, cmd := range postExec {
				if cmdStr, ok := cmd.(string); ok {
					c.PostExec = append(c.PostExec, cmdStr)
				}
			}
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

					// Add prefix handling
					if prefix, ok := argMap["prefix"].(string); ok {
						argument.Prefix = prefix
					}

					c.Arguments = append(c.Arguments, argument)
				}
			}
		}

		// Parse examples if present
		if examples, ok := v["examples"].([]interface{}); ok {
			for _, ex := range examples {
				if exMap, ok := ex.(map[string]interface{}); ok {
					example := CommandExample{}

					if desc, ok := exMap["description"].(string); ok {
						example.Description = desc
					}
					if cmd, ok := exMap["command"].(string); ok {
						example.Command = cmd
					}

					// Only add if both fields are present
					if example.Description != "" && example.Command != "" {
						c.Examples = append(c.Examples, example)
					}
				}
			}
		}

		// Parse environment variables if present
		if env, ok := v["env"].(map[string]interface{}); ok {
			for key, value := range env {
				if strValue, ok := value.(string); ok {
					c.Env[key] = strValue
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

// PromptConfig represents a configured prompt that can be exposed via MCP
type PromptConfig struct {
	Name        string            `toml:"name"`                // Name of the prompt
	Description string            `toml:"description"`         // Description of what the prompt does
	Content     string            `toml:"content"`             // The actual prompt content/template
	MCP         string            `toml:"mcp,omitempty"`       // Optional MCP server name this prompt belongs to
	Arguments   []CommandArgument `toml:"arguments,omitempty"` // Argument definitions for the prompt
}

type Settings struct {
	LogLevel              string                   `toml:"log_level"`
	Env                   map[string]string        `toml:"env,omitempty"`
	Projects              map[string]Project       `toml:"projects"`
	Commands              map[string]CommandConfig `toml:"commands"`
	Prompts               map[string]PromptConfig  `toml:"prompts"` // Add prompts configuration
	ExecutableSearchPaths []string                 `toml:"executable_search_paths"`
	CommandDirs           []string                 `toml:"command_dirs"` // Directories to load additional command files from
	MCPPort               int                      `toml:"mcp_port"`
	MCPServers            map[string]MCPServer     `toml:"mcp_servers"`
}

// PathConfig defines the directory structure for settings
type PathConfig struct {
	SettingsDir    string
	AppDir         string
	CfgFile        string
	ExecutablesDir string
	CommandsDir    string
}

// DefaultPathConfig contains the default paths configuration
var DefaultPathConfig = PathConfig{
	SettingsDir:    ".config",
	AppDir:         "interop",
	CfgFile:        "settings.toml",
	ExecutablesDir: "executables",
	CommandsDir:    "commands.d",
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

// defaultSettingsTemplate is the embedded template for the settings file.
// This avoids issues with missing template files and makes the binary self-contained.
var defaultSettingsTemplate = `# Interop Settings Template
# This file documents all available configuration options for Interop.
# Uncomment and edit the fields you wish to configure.

# =====================
# GLOBAL SETTINGS
# =====================

# log_level = "warning"         # Options: error, warning, verbose
# executable_search_paths = [   # Additional directories to search for executables
#   "~/.local/bin",
#   "~/bin"
# ]
# command_dirs = [              # Directories to load additional command definitions from
#   "~/.config/interop/commands.d"  # Default: if not specified, this directory is automatically used
#   "~/projects/shared/interop-commands"
# ]
# mcp_port = 8081               # Default port for the main MCP server

# =====================
# MCP SERVER CONFIGURATION
# =====================

#[mcp_servers.example]
#name = "example"               # Unique name for this MCP server (must match the key)
#description = "Example domain-specific server"
#port = 8082                    # Port for this MCP server

# =====================
# MCP PROMPTS
# =====================
# Define reusable prompts that MCP clients can access. Prompts are templates
# that help LLMs interact with your server effectively.
#
# Each prompt can be assigned to a specific MCP server using the 'mcp' field.
# If no 'mcp' field is specified, the prompt will be available on the default server.
#
# Prompts can also define arguments that allow customization when the prompt is used.

#[prompts.create_merge_request]
#name = "create_merge_request"
#description = "Complete MR creation workflow: analyzes branch changes, generates MR description, and creates the merge request"
#content = """
#You are helping create a merge request. Follow this workflow:
#
#1. **Analyze Branch Changes**: First, run the generate-cursor-prompt-for-mr command with target branch: {target_branch}
#2. **Review the Analysis**: Read the generated analysis and create an appropriate MR title: {mr_title}
#3. **Generate MR Description**: Based on the analysis, create a detailed MR description
#4. **Create the MR**: Run the create-mr command with the temp directory from step 1
#
#Include detailed changes: {include_detailed_changes}
#
#Make sure to:
#- Use clear, descriptive titles
#- Include context about what changed and why
#- Reference any related issues or tickets
#- Follow the team's MR guidelines
#"""
#arguments = [
#  { name = "target_branch", type = "string", description = "The branch you want to merge into", required = true },
#  { name = "mr_title", type = "string", description = "Title for the merge request", default = "" },
#  { name = "include_detailed_changes", type = "bool", description = "Include detailed file changes in description", default = true }
#]
# This prompt orchestrates multiple MCP commands in a workflow

#[prompts.code_review]
#name = "code_review"           # Name of the prompt (must match the key)
#description = "Code review assistance prompt"
#content = "Please review the following {language} code, focusing on {focus_area}. Look for potential issues, improvements, and best practices."
#mcp = "example"                # (Optional) Assign this prompt to a specific MCP server
#arguments = [                  # (Optional) Arguments for prompt customization
#  { name = "language", type = "string", description = "Programming language", required = true },
#  { name = "focus_area", type = "string", description = "Area to focus on", default = "general" }
#]

#[prompts.documentation]
#name = "documentation"         # Name of the prompt (must match the key)  
#description = "Generate technical documentation"
#content = """
#Generate comprehensive technical documentation for {topic}.
#
#Include examples: {include_examples}
#Detail level: {detail_level}/5
#
#Structure the documentation with:
#1. Overview and purpose
#2. Key concepts and terminology  
#3. Implementation details
#4. Usage examples (if requested)
#5. Best practices and recommendations
#"""
#arguments = [                  # Example with different argument types
#  { name = "topic", type = "string", description = "Documentation topic", required = true },
#  { name = "include_examples", type = "bool", description = "Include code examples", default = true },
#  { name = "detail_level", type = "number", description = "Detail level (1-5)", default = 3 }
#]
# No 'mcp' field means this prompt is available on the default server

# =====================
# MCP TOOLS & GLOBAL COMMANDS
# =====================
# Global commands automatically receive an optional "project_path" parameter when exposed as MCP tools.
# This allows AI assistants to specify a working directory for the command.
#
# A command is considered global unless it's bound to a project WITHOUT an alias.
# Commands with aliases remain global - only the alias becomes project-specific.
#
# Examples:
# - Command "build" with alias "b" in a project: "build" stays global, "b" is project-specific
# - Command "test" without alias in a project: "test" becomes project-specific
# - Command "deploy" not in any project: "deploy" is global
#
# Global commands can be run in any project directory by providing the project_path parameter.

# =====================
# PROJECT DEFINITIONS
# =====================

#[projects.sample_project]
#path = "~/projects/sample"     # Path to the project directory (must be inside $HOME)
#description = "Sample project for demonstration"
#commands = [                   # List of commands for this project (with optional aliases)
#  { command_name = "build", alias = "b" },
#  { command_name = "test" }
#]

# =====================
# COMMAND DEFINITIONS
# =====================
# Commands can be defined in the main settings.toml file or in separate files
# in directories specified by command_dirs. Commands from main settings.toml
# take precedence over those in external directories.

#[commands.build]
#cmd = "go build ./..."         # The shell command or executable to run
#description = "Build the project"
#version = "1.0.0"              # (Optional) Version of the command
#is_enabled = true              # Enable or disable this command
#is_executable = false          # If true, run as an executable; if false, run in shell
#mcp = "example"                # (Optional) Assign this command to a specific MCP server
#arguments = [                  # (Optional) List of arguments for this command
#  { name = "output_file", type = "string", description = "Output file name", required = true },
#  { name = "package", type = "string", description = "Package to build", default = "./cmd/app" }
#]
#examples = [                   # (Optional) Usage examples for the command
#  {
#    description = "Build the main application",
#    command = "interop run build output_file=my-app"
#  },
#  {
#    description = "Build a specific package",
#    command = "interop run build output_file=my-tool package=./cmd/tool"
#  }
#]

#[commands.test]
#cmd = "go test ./..."
#description = "Run tests"
#is_enabled = true
#is_executable = false

#[commands.deploy]
#cmd = "deploy.sh"
#description = "Deploy the project"
#is_enabled = true
#is_executable = true
#mcp = "example"

# Example command with prefixed arguments
#[commands.script]
#cmd = "python scripts/myscript.py"
#description = "Run a Python script with prefixed arguments"
#arguments = [
#  { name = "keys", type = "string", description = "Keys to process", required = false, prefix = "--keys" },
#  { name = "language", type = "string", description = "Language code", required = false, prefix = "--language" }
#]

# =====================
# COMMAND ARGUMENT TYPES
# =====================
# type: string | number | bool
# Example:
# arguments = [
#   { name = "type", type = "string", description = "Component type", required = true },
#   { name = "force", type = "bool", description = "Overwrite if exists", default = false }
# ]

# =====================
# PREFIX ARGUMENTS
# =====================
# Use the 'prefix' field to specify command-line prefixes for arguments.
# For example:
# arguments = [
#   { name = "verbose", type = "bool", description = "Enable verbose output", prefix = "--verbose" },
#   { name = "keys", type = "string", description = "Keys to process", prefix = "--keys" }
# ]
# This will generate commands like: my-command --verbose --keys value

# =====================
# END OF TEMPLATE
# =====================
`

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

	// Create commands.d directory for command definitions
	commandsDir := filepath.Join(base, pathConfig.CommandsDir)
	if e := os.MkdirAll(commandsDir, 0o755); e != nil {
		logging.Error("Can't create the directory for commands: " + e.Error())
	} else {
		logging.Message("commands.d directory is created")
	}

	if _, e := os.Stat(path); errors.Is(e, os.ErrNotExist) {
		// Use the embedded template instead of reading from a file
		// This avoids issues with missing template files
		f, e := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
		if e != nil {
			logging.Error("Failed to create settings file: " + e.Error())
		} else {
			if _, writeErr := f.Write([]byte(defaultSettingsTemplate)); writeErr != nil {
				logging.Error("Failed to write template to settings file: " + writeErr.Error())
			}
			if e := f.Close(); e != nil {
				logging.Error("Failed to close settings file: " + e.Error())
			}
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

	// Check prompt configurations
	for promptName, prompt := range cfg.Prompts {
		// Validate required fields
		if prompt.Name == "" {
			return fmt.Errorf("prompt '%s' must have a name", promptName)
		}

		if prompt.Description == "" {
			return fmt.Errorf("prompt '%s' must have a description", promptName)
		}

		if prompt.Content == "" {
			return fmt.Errorf("prompt '%s' must have content", promptName)
		}

		// Ensure prompt.Name matches the key
		if prompt.Name != promptName {
			return fmt.Errorf("prompt name '%s' doesn't match key '%s'", prompt.Name, promptName)
		}

		// Check prompt MCP references
		if prompt.MCP != "" {
			if _, exists := cfg.MCPServers[prompt.MCP]; !exists {
				return fmt.Errorf("prompt '%s' references non-existent MCP server '%s'",
					promptName, prompt.MCP)
			}
		}

		// Validate prompt arguments
		for _, arg := range prompt.Arguments {
			if arg.Name == "" {
				return fmt.Errorf("prompt '%s' has an argument without a name", promptName)
			}
			if arg.Description == "" {
				return fmt.Errorf("prompt '%s' argument '%s' must have a description", promptName, arg.Name)
			}
			// Validate argument types
			switch arg.Type {
			case ArgumentTypeString, ArgumentTypeNumber, ArgumentTypeBool:
				// Valid types
			case "":
				// Default to string if not specified
			default:
				return fmt.Errorf("prompt '%s' argument '%s' has invalid type '%s'", promptName, arg.Name, arg.Type)
			}
		}
	}

	return nil
}

// loadCommandsFromDirectory loads command definitions from TOML files in a directory
func loadCommandsFromDirectory(dirPath string) (map[string]CommandConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Handle tilde expansion
	if strings.HasPrefix(dirPath, "~/") {
		dirPath = filepath.Join(homeDir, dirPath[2:])
	} else if !filepath.IsAbs(dirPath) {
		dirPath = filepath.Join(homeDir, dirPath)
	}

	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		logging.Warning("Command directory does not exist: %s", dirPath)
		return map[string]CommandConfig{}, nil
	}

	commands := make(map[string]CommandConfig)

	// Read all .toml files in the directory
	files, err := filepath.Glob(filepath.Join(dirPath, "*.toml"))
	if err != nil {
		return nil, fmt.Errorf("failed to list TOML files in %s: %w", dirPath, err)
	}

	// Sort files alphabetically for consistent loading order
	sort.Strings(files)

	for _, file := range files {
		var fileCommands struct {
			Commands map[string]CommandConfig `toml:"commands"`
		}

		if _, err := toml.DecodeFile(file, &fileCommands); err != nil {
			logging.Warning("Failed to parse command file %s: %v", file, err)
			continue
		}

		// Merge commands from this file
		for name, cmd := range fileCommands.Commands {
			if _, exists := commands[name]; exists {
				logging.Warning("Duplicate command '%s' found in %s, keeping first occurrence", name, file)
				continue
			}
			commands[name] = cmd
			logging.Message("Loaded command '%s' from %s", name, file)
		}
	}

	return commands, nil
}

// mergeCommands merges commands from multiple sources with precedence rules
// Priority order: main settings.toml > command_dirs (in order) > within dir (alphabetical)
func mergeCommands(mainCommands map[string]CommandConfig, commandDirs []string) (map[string]CommandConfig, []string) {
	result := make(map[string]CommandConfig)
	var conflicts []string

	// Start with main commands (highest priority)
	for name, cmd := range mainCommands {
		result[name] = cmd
	}

	// Load commands from each directory in order
	for _, dir := range commandDirs {
		dirCommands, err := loadCommandsFromDirectory(dir)
		if err != nil {
			logging.Warning("Failed to load commands from directory %s: %v", dir, err)
			continue
		}

		// Merge directory commands
		for name, cmd := range dirCommands {
			if _, exists := result[name]; exists {
				conflicts = append(conflicts, fmt.Sprintf("Command '%s' conflicts between main settings and %s", name, dir))
				continue // Keep existing (higher priority)
			}
			result[name] = cmd
		}
	}

	return result, conflicts
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

		// Initialize empty collections if nil
		if c.Projects == nil {
			c.Projects = make(map[string]Project)
		}
		if c.Commands == nil {
			c.Commands = make(map[string]CommandConfig)
		}
		if c.Prompts == nil {
			c.Prompts = make(map[string]PromptConfig)
		}
		if c.MCPServers == nil {
			c.MCPServers = make(map[string]MCPServer)
		}

		// Handle command directories with backwards compatibility
		commandDirs := c.CommandDirs

		// If no command_dirs are explicitly configured, add the default commands.d directory
		if len(commandDirs) == 0 {
			defaultCommandsPath, err := GetCommandsPath()
			if err == nil {
				// Only add if the directory exists to avoid warnings
				if _, err := os.Stat(defaultCommandsPath); err == nil {
					commandDirs = []string{defaultCommandsPath}
					logging.Message("Using default commands directory: %s", defaultCommandsPath)
				}
			}
		}

		// Load commands from command directories
		if len(commandDirs) > 0 {
			mergedCommands, conflicts := mergeCommands(c.Commands, commandDirs)
			c.Commands = mergedCommands

			// Log conflicts for visibility
			for _, conflict := range conflicts {
				logging.Warning(conflict)
			}

			if len(conflicts) > 0 {
				logging.Message("Found %d command name conflicts. Main settings.toml takes precedence.", len(conflicts))
			}

			logging.Message("Loaded commands from %d directories", len(commandDirs))
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

	logging.Message("Executable search paths: %v", paths)
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

// MergeEnvironmentVariables merges environment variables with the specified precedence:
// 1. Command-level env (highest priority)
// 2. Project-level env (if executed in a project context)
// 3. Global-level env
// 4. The shell's existing environment variables (lowest priority)
func MergeEnvironmentVariables(cfg *Settings, commandName string, projectName string) []string {
	// Start with the current environment
	envMap := make(map[string]string)

	// Copy all existing environment variables (lowest priority)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Apply global environment variables (3rd priority)
	if cfg.Env != nil {
		for key, value := range cfg.Env {
			envMap[key] = value
		}
	}

	// Apply project-level environment variables if in project context (2nd priority)
	if projectName != "" {
		if project, exists := cfg.Projects[projectName]; exists && project.Env != nil {
			for key, value := range project.Env {
				envMap[key] = value
			}
		}
	}

	// Apply command-level environment variables (highest priority)
	if command, exists := cfg.Commands[commandName]; exists && command.Env != nil {
		for key, value := range command.Env {
			envMap[key] = value
		}
	}

	// Convert map back to slice format expected by exec.Cmd
	env := make([]string, 0, len(envMap))
	for key, value := range envMap {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env
}

// GetCommandsPath returns the path to the default commands directory
func GetCommandsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(
		homeDir,
		DefaultPathConfig.SettingsDir,
		DefaultPathConfig.AppDir,
		DefaultPathConfig.CommandsDir,
	), nil
}
