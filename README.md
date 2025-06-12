# Interop

Interop is a powerful command-line interface tool designed to improve developer productivity by providing a unified interface for managing projects, executing commands, and integrating with AI assistants.

## What is Interop?

Interop serves as a bridge between your development projects, custom commands, and AI assistants. It allows you to:

- Organize multiple projects with metadata, commands, and validation
- Define and execute commands with project context awareness
- Configure multiple MCP (Model Context Protocol) servers for AI integration
- Streamline repetitive development tasks across different domains

## Core Features

- **Project Management**: Track and validate multiple project directories
- **Command Execution**: Run commands with project context and arguments
- **Dynamic Command Loading**: Load commands from multiple directories with precedence rules
- **Remote Configuration System**: Fetch and sync configurations from Git repositories with conflict resolution
- **AI Integration**: Multiple MCP servers to expose commands to AI assistants with enhanced metadata
- **Configuration Management**: TOML-based configuration with validation and conflict detection
- **Cross-Platform Support**: Works on Linux, macOS, and Windows

## Installation

### Using Homebrew (macOS/Linux)

```bash
brew install yigitozgumus/formulae/interop
```

### Building from Source

1. Clone the repository:
```bash
git clone https://github.com/yigitozgumus/interop.git
cd interop
```

2. Build the project:
```bash
go build -o interop ./cmd/cli
```

## Configuration

Interop uses a TOML configuration file at `~/.config/interop/settings.toml`. To edit it:

```bash
interop edit
```

### Configuration Structure

```toml
# Global settings
log_level = "verbose"  # Options: error, warning, verbose
executable_search_paths = ["~/.local/bin", "~/bin"]
command_dirs = ["~/.config/interop/commands.d", "~/projects/shared/interop-commands"]
mcp_port = 8081  # Default MCP server port

# MCP Server Configurations
[mcp_servers.domain1]
name = "domain1"
description = "Domain-specific commands"
port = 8082

[mcp_servers.domain2]
name = "domain2"
description = "Another domain for commands"
port = 8083

# Project Definitions
[projects.project1]
path = "~/projects/project1"
description = "Project 1 description"
commands = [
  { command_name = "build", alias = "b" },
  { command_name = "test" }
]

[projects.project2]
path = "~/projects/project2"
description = "Project 2 description"
commands = [
  { command_name = "deploy", alias = "d" }
]

# Command Definitions
[commands.build]
cmd = "go build ./..."
description = "Build the project"
is_enabled = true
is_executable = false
# Assign to a specific MCP server
mcp = "domain1"

[commands.test]
cmd = "go test ./..."
description = "Run tests"
is_enabled = true
is_executable = false

[commands.deploy]
cmd = "deploy.sh"
description = "Deploy the project"
is_enabled = true
is_executable = true
mcp = "domain2"

# Command with Arguments, Version, and Examples
[commands.build-app]
cmd = "go build -o ${output_file} ${package}"
description = "Build a Go application"
version = "1.1.0"
is_enabled = true
is_executable = false
arguments = [
  { name = "output_file", type = "string", description = "Output file name", required = true },
  { name = "package", type = "string", description = "Package to build", default = "./cmd/app" }
]
examples = [
  {
    description = "Build the main application",
    command = "interop run build-app output_file=my-app"
  },
  {
    description = "Build a specific package",
    command = "interop run build-app output_file=my-tool package=./cmd/tool"
  }
]
```

## Project Management

Projects are the core organizational unit in Interop.

### Listing Projects

```bash
interop projects
```

Output example:
```
PROJECTS:
=========

📁 Name: project1
   Path: ~/projects/project1
   Status: Valid: ✓  |  In $HOME: ✓
   Description: Project 1 description
   Commands:
      ⚡ build (alias: b)
         Build the project
      ⚡ test
         Run tests

📁 Name: project2
   Path: ~/projects/project2
   Status: Valid: ✓  |  In $HOME: ✓
   Description: Project 2 description
   Commands:
      ⚡ deploy (alias: d)
         Deploy the project
```

### Project Configuration

Each project includes:
- **Path**: Directory location (validated for existence)
- **Description**: Optional project description
- **Commands**: List of commands with optional aliases

## Dynamic Configuration Loading

Interop supports loading configuration definitions from multiple directories, enabling better organization and scalability for large configuration collections.

### Configuration

Add `command_dirs` to your global settings to specify directories containing configuration definition files:

```toml
command_dirs = [
  "~/.config/interop/config.d",
  "~/projects/shared/interop-configs"
]
```

### Configuration Directory Structure

Each directory can contain multiple `*.toml` files with configuration definitions:

```
~/.config/interop/config.d/
├── git-commands.toml
├── docker-commands.toml
├── dev-projects.toml
└── ai-prompts.toml
```

Example `git-commands.toml`:

```toml
[commands.git-status]
cmd = "git status"
description = "Show the working tree status"
version = "1.0.0"
is_enabled = true
mcp = "dev-tools"

[commands.git-pull]
cmd = "git pull --rebase"
description = "Fetch from and integrate with another repository"
version = "1.0.0"
is_enabled = true
mcp = "dev-tools"
```

Example `dev-projects.toml`:

```toml
[projects.my-api]
path = "~/projects/my-api"
description = "Main API project"
commands = [
  { command_name = "build", alias = "b" },
  { command_name = "test", alias = "t" }
]
```

### What Can Be Included

Configuration files in these directories can contain:
- **Commands** (`[commands.name]`) - Command definitions
- **Projects** (`[projects.name]`) - Project configurations  
- **Prompts** (`[prompts.name]`) - AI prompt templates
- **MCP Servers** (`[mcp_servers.name]`) - MCP server definitions

### Precedence Rules

When configuration names conflict, Interop follows a clear precedence order:

1. **Main `settings.toml`** (highest priority)
2. **Configuration directories** in the order specified in `command_dirs`
3. **Files within directories** in alphabetical order

This ensures predictable configuration resolution and allows for easy overriding of shared configurations.

### Benefits

- **Organization**: Group related configurations in separate files
- **Sharing**: Share configuration collections across teams or projects
- **Modularity**: Enable/disable entire configuration sets by directory

## Remote Configuration System

Interop includes a powerful remote configuration system that allows you to fetch and manage configurations from Git repositories. This enables teams to share command definitions, maintain centralized configuration libraries, and keep local setups synchronized with remote sources.

### Overview

The remote configuration system:
- Fetches configurations from Git repositories
- Maintains local copies in `config.d.remote` and `executables.remote` directories
- Tracks file changes with SHA-256 hashing for incremental updates
- Automatically integrates remote configurations with local ones
- Provides conflict detection and resolution with local configurations taking precedence

### Managing Remote Configurations

#### Adding a Remote Repository

```bash
# Add a remote Git repository
interop config remote add my-team https://github.com/myteam/interop-configs.git

# Add with SSH (recommended for private repositories)
interop config remote add my-team git@github.com:myteam/interop-configs.git
```

#### Listing Remote Repositories

```bash
interop config remote show
```

Output example:
```
Remote Configurations:
======================

🔗 my-team
   URL: git@github.com:myteam/interop-configs.git
   Status: ✓ Valid Git URL

🔗 shared-tools  
   URL: https://github.com/company/shared-tools.git
   Status: ✓ Valid Git URL
```

#### Fetching Remote Configurations

```bash
# Fetch from all configured remotes
interop config remote fetch

# Fetch from a specific remote
interop config remote fetch my-team
```

The fetch process:
1. **Clones** the repository to a temporary directory
2. **Validates** the repository structure (requires `config.d` and/or `executables` folders)
3. **Compares** file hashes to detect changes
4. **Syncs** only modified files to local remote directories
5. **Updates** version tracking with commit information
6. **Cleans up** files that were removed from the remote

#### Removing Remote Repositories

```bash
# Remove a specific remote
interop config remote remove my-team

# Clear all remote configurations and cached files
interop config remote clear
```

### Repository Structure Requirements

Remote repositories must follow this structure:

```
your-repo/
├── config.d/              # Configuration files (required)
│   ├── commands.toml      # Command definitions
│   ├── projects.toml      # Project configurations
│   └── mcp-servers.toml   # MCP server definitions
└── executables/           # Executable files (optional)
    ├── deploy.sh
    ├── build-tool
    └── scripts/
        └── helper.py
```

#### Example Remote Configuration

`config.d/team-commands.toml`:
```toml
[commands.team-deploy]
cmd = "deploy.sh"
description = "Deploy using team standards"
is_executable = true
mcp = "team-tools"
version = "2.1.0"
arguments = [
  { name = "environment", type = "string", required = true, description = "Target environment" },
  { name = "force", type = "bool", default = false, description = "Force deployment" }
]

[commands.team-test]
cmd = "run-team-tests.sh"
description = "Run standardized team tests"
is_executable = true
mcp = "team-tools"

[mcp_servers.team-tools]
name = "team-tools"
description = "Team standardized tools"
port = 8084
```

### Local Integration

Remote configurations are automatically integrated into your local setup:

#### Directory Structure

```
~/.config/interop/
├── settings.toml           # Your local settings
├── config.d/              # Local configurations
│   └── personal.toml
├── config.d.remote/       # Remote configurations (auto-managed)
│   ├── team-commands.toml
│   └── shared-tools.toml
├── executables/           # Local executables
├── executables.remote/    # Remote executables (auto-managed)
│   ├── deploy.sh
│   └── build-tool
└── versions.toml          # Remote tracking metadata (auto-managed)
```

#### Precedence Rules

When configurations conflict, Interop follows this precedence:

1. **Local configurations** (`config.d/`) - highest priority
2. **Remote configurations** (`config.d.remote/`) - lower priority
3. **Main settings.toml** - fallback for global settings

This ensures your local customizations always take precedence while still benefiting from shared remote configurations.

### Conflict Detection and Resolution

The validation system provides comprehensive conflict detection:

```bash
interop validate
```

Example output with conflicts:
```
Configuration Overview
=====================

Configuration Sources:
---------------------
🏠 Main Settings: /Users/user/.config/interop/settings.toml
🏠 Command Directories:
   🏠 /Users/user/.config/interop/config.d (2 files)
☁️ Remote Configuration:
   ✓ config.d.remote: Available (3 files)
   ✓ executables.remote: Available (5 files)
   ✓ Remote tracking: Active

⚠️ Potential Conflicts:
   ⚠️ Command 'deploy' exists in both local and remote configs
   ⚠️ Command 'test-suite' exists in both local and remote configs
   → Local configurations take precedence

Commands:
--------
🌐 ✓ deploy (Shell) (🏠 Local)
   └─ My custom deploy script
   └─ 🔌 Default MCP server (Port: 8081)

🌐 ✓ team-deploy (Shell) (☁️ Remote)
   └─ Deploy using team standards
   └─ 🔌 Assigned to MCP server: team-tools (Port: 8084)
```

### Version Tracking and Incremental Updates

Interop maintains detailed tracking of remote configurations:

#### Version Information

The system tracks:
- **File hashes** (SHA-256) for change detection
- **Last commit ID** for repository state
- **Fetch timestamps** for update history
- **File paths** for cleanup of removed files

#### Incremental Updates

Subsequent fetches are optimized:
- Only changed files are downloaded
- Removed files are cleaned up locally
- Commit history is preserved for rollback capability
- Network usage is minimized

### Git URL Validation

The system validates Git URLs to ensure compatibility:

#### Supported Formats

**SSH Format:**
```bash
git@github.com:user/repo.git
git@gitlab.com:user/repo.git
git@bitbucket.org:user/repo.git
```

**HTTPS Format:**
```bash
https://github.com/user/repo.git
https://gitlab.com/user/repo.git
https://bitbucket.org/user/repo.git
```

**Custom Git Servers:**
```bash
https://git.company.com/team/configs.git
git@git.company.com:team/configs.git
```

### Use Cases

#### Team Standardization

```bash
# Set up team-wide configurations
interop config remote add company-standards git@github.com:company/interop-standards.git
interop config remote fetch

# Now all team members have access to:
# - Standardized deployment scripts
# - Common development commands  
# - Shared MCP server configurations
# - Team-specific project templates
```

#### Multi-Environment Management

```bash
# Different configurations for different environments
interop config remote add prod-tools git@github.com:company/prod-tools.git
interop config remote add dev-tools git@github.com:company/dev-tools.git

# Fetch environment-specific tools
interop config remote fetch prod-tools
interop config remote fetch dev-tools
```

#### Open Source Tool Collections

```bash
# Add community-maintained tool collections
interop config remote add awesome-dev-tools https://github.com/community/awesome-dev-tools.git
interop config remote fetch
```

### Best Practices

#### Repository Organization

1. **Separate concerns**: Use different repositories for different domains
2. **Version your configurations**: Tag releases for stable configuration sets
3. **Document commands**: Include comprehensive descriptions and examples
4. **Test configurations**: Validate configurations before pushing

#### Security Considerations

1. **Use SSH keys** for private repositories
2. **Review remote configurations** before fetching
3. **Keep sensitive data local** - don't put secrets in remote configs
4. **Audit remote sources** regularly

#### Team Workflow

1. **Centralize common tools** in team repositories
2. **Allow local overrides** for personal preferences
3. **Version control changes** to shared configurations
4. **Communicate updates** when shared configs change

### Troubleshooting

#### Common Issues

**Repository not found:**
```bash
# Check URL and access permissions
git clone <your-repo-url>  # Test manually
```

**Invalid repository structure:**
```bash
# Ensure repository has config.d/ or executables/ directory
# Check repository contents match expected structure
```

**Conflicts with local configurations:**
```bash
# Use validation to identify conflicts
interop validate

# Rename local commands if needed
# Or remove remote repository if not needed
```

**Network issues:**
```bash
# Check internet connectivity
# Verify Git credentials are set up
# Try fetching manually: git clone <repo-url>
```

## Command Management

Interop provides a flexible command system that adapts to your workflow.

### Listing Commands

```bash
interop commands
```

Output example:
```
COMMANDS:
=========

⚡ Name: build
   Status: Enabled: ✓  |  Source: Script
   Description: Build the project
   MCP Server: domain1

⚡ Name: test
   Status: Enabled: ✓  |  Source: Script
   Description: Run tests
   
⚡ Name: deploy
   Status: Enabled: ✓  |  Source: Executable
   Description: Deploy the project
   MCP Server: domain2
```

### Command Types

1. **Shell Commands**: Run through the system shell
   ```toml
   [commands.list]
   cmd = "ls -la"
   ```

2. **Executable Commands**: Run directly from configured paths
   ```toml
   [commands.deploy]
   cmd = "deploy.sh"
   is_executable = true
   ```

3. **Project-bound Commands**: Run in the context of a specific project
   ```toml
   [projects.project1]
   commands = [{ command_name = "build" }]
   ```

4. **Commands with Arguments**: Templated commands with validation
   ```toml
   [commands.build-app]
   cmd = "go build -o ${output_file} ${package}"
   arguments = [
     { name = "output_file", type = "string", required = true },
     { name = "package", type = "string", default = "./cmd/app" }
   ]
   ```

### Executing Commands

Run a command by name or alias:

```bash
# Simple command
interop run build

# Command with alias
interop run b  # Runs the build command

# Command with arguments
interop run build-app output_file=myapp.exe
```

For project-bound commands, Interop automatically:
1. Changes to the project directory
2. Executes the command
3. Returns to the original directory

## MCP Server Integration

Interop includes robust support for AI integration via MCP (Model Context Protocol) servers.

### What are MCP Servers?

MCP servers expose your commands as tools that can be invoked by AI assistants like Claude. Each server can provide a different set of commands, allowing domain-specific organization.

### Managing MCP Servers

```bash
# Start servers
interop mcp start                # Start default server
interop mcp start domain1        # Start specific server
interop mcp start --all          # Start all servers

# Check status
interop mcp status               # Default shows all servers
interop mcp status domain1       # Check specific server

# Stop servers
interop mcp stop domain1         # Stop specific server
interop mcp stop --all           # Stop all servers

# Restart servers
interop mcp restart domain1      # Restart specific server
interop mcp restart --all        # Restart all servers

# Port management
interop mcp port-check           # Check if ports are available

# Get configuration for AI tools
interop mcp export               # Export JSON configuration
```

### Multiple MCP Servers

You can organize commands by domain:

```toml
[mcp_servers.work]
name = "work"
description = "Work-related commands"
port = 8082

[mcp_servers.personal]
name = "personal"
description = "Personal project commands"
port = 8083

[commands.work-task]
cmd = "work-script.sh"
mcp = "work"  # This command is available on the work server

[commands.personal-task]
cmd = "personal-script.sh"
mcp = "personal"  # This command is available on the personal server
```

Each server exposes only the commands assigned to it, creating a clean separation between different domains.

### AI Assistant Integration

When an AI assistant connects to an MCP server, it can:
1. See available commands and their descriptions
2. Execute commands with arguments
3. Receive command outputs and errors

This creates a powerful interface where the AI can help you execute tasks based on natural language instructions.

## Command Arguments

Commands can have typed arguments with validation:

```toml
[commands.generate]
cmd = "generate.sh ${type} ${name} ${force}"
description = "Generate a new component"
arguments = [
  { name = "type", type = "string", description = "Component type", required = true },
  { name = "name", type = "string", description = "Component name", required = true },
  { name = "force", type = "bool", description = "Overwrite if exists", default = false }
]
```

### Argument Types

- **string**: Text values
- **number**: Numeric values (integers or decimals)
- **bool**: Boolean values (true/false)

### Argument Features

- **Required**: Mark arguments that must be provided
- **Default Values**: Set fallback values for optional arguments
- **Descriptions**: Document the purpose of each argument
- **Prefix**: Specify command-line flags to use for arguments (e.g., `--key`)

### Using Arguments

```bash
# Named arguments
interop run generate type=component name=Button

# Positional arguments (in order of definition)
interop run generate component Button true
```

### Prefixed Arguments

Prefixed arguments allow you to define command-line arguments with specific prefixes (such as `--keys` or `-f`). This is especially useful when working with scripts or tools that expect arguments in a specific format:

```toml
[commands.update-strings]
cmd="python3 scripts/update_strings.py"
description="Update localization strings"
arguments=[
  {name = "keys", type="string", required = false, description = "Keys to update", prefix = "--keys"},
  {name = "language", type="string", required = false, description = "Language code", prefix = "--language"},
  {name = "verbose", type="bool", required = false, description = "Verbose output", prefix = "--verbose"}
]
```

When executing:
```bash
interop run update-strings --keys "key1 key2" --language en --verbose true
```

The actual command executed will be:
```bash
python3 scripts/update_strings.py --keys key1 key2 --language en --verbose
```

#### How Prefixed Arguments Work

1. For arguments with prefixes: Interop appends them to the command with their prefixes
2. For arguments without prefixes: Interop substitutes them in the command string using `${arg_name}` placeholders
3. Boolean arguments with prefixes: If the value is `true`, only the prefix is added; otherwise, the argument is omitted
4. Non-boolean arguments with prefixes: The prefix and value are added together

#### Benefits of Prefixed Arguments

- Works consistently across all shells (bash, fish, zsh, etc.)
- Arguments can be provided in any order
- No need to escape special characters in argument values
- Compatible with tools that require specific argument formats

## Enhanced Command Metadata

Interop supports rich metadata for commands to improve AI assistant integration and documentation.

### Version Information

Add version tracking to your commands:

```toml
[commands.deploy]
cmd = "deploy.sh"
description = "Deploy the application"
version = "2.1.0"
is_enabled = true
```

### Usage Examples

Provide concrete examples of how to use commands:

```toml
[commands.create-component]
cmd = "generate.sh ${type} ${name}"
description = "Generate a new component"
version = "1.0.0"
arguments = [
  { name = "type", type = "string", description = "Component type", required = true },
  { name = "name", type = "string", description = "Component name", required = true }
]
examples = [
  {
    description = "Create a React component",
    command = "interop run create-component type=react name=Button"
  },
  {
    description = "Create a Vue component", 
    command = "interop run create-component type=vue name=Header"
  }
]
```

### Benefits for AI Integration

When commands include version and examples:

- **AI assistants** can provide more accurate suggestions
- **Documentation** is generated automatically
- **Team onboarding** becomes easier with concrete examples
- **Version tracking** helps with compatibility and updates

## Validation & Diagnostics

### Enhanced Configuration Validation

```bash
interop validate
```

The validation system provides comprehensive analysis of your configuration:

#### Configuration Sources Analysis
- **Main Settings**: Validates `settings.toml` exists and is accessible
- **Local Directories**: Shows file counts and accessibility of `config.d/`
- **Remote Directories**: Shows status of `config.d.remote/` and `executables.remote/`
- **Remote Tracking**: Indicates if remote version tracking is active

#### Conflict Detection
- **Command Conflicts**: Identifies commands defined in multiple sources
- **Precedence Rules**: Shows which configuration takes precedence
- **Visual Indicators**: Uses symbols to highlight conflicts and warnings
- **Source Attribution**: Shows whether each command comes from local or remote sources

#### Validation Checks
- Project paths exist and are accessible
- Command references are valid
- No conflicting aliases within the same scope
- Required command arguments have proper definitions
- MCP server ports don't conflict
- Command directory accessibility and TOML syntax
- Remote repository structure compliance
- Git URL format validation

#### Example Validation Output

```
Configuration Overview
=====================

Configuration Sources:
---------------------
🏠 Main Settings: /Users/user/.config/interop/settings.toml
🏠 Command Directories:
   🏠 /Users/user/.config/interop/config.d (2 files)
☁️ Remote Configuration:
   ✓ config.d.remote: Available (3 files)
   ✓ executables.remote: Available (5 files)
   ✓ Remote tracking: Active

⚠️ Potential Conflicts:
   ⚠️ Command 'deploy' exists in both local and remote configs
   → Local configurations take precedence

MCP Servers:
-----------
🔌 Default MCP Server (Port: 8081)
   └─ Commands: (commands with no MCP field)

🔌 team-tools MCP Server (Port: 8084)
   └─ Team standardized tools
   └─ Commands: 5

Commands:
--------
🌐 ✓ deploy (Shell) (🏠 Local)
   └─ My custom deploy script
   └─ 🔌 Default MCP server (Port: 8081)

🌐 ✓ team-deploy (Shell) (☁️ Remote)
   └─ Deploy using team standards
   └─ 🔌 Assigned to MCP server: team-tools (Port: 8084)

Legend:
-------
🌐 Global Command        🏠 Local Configuration
📂 Project-bound Command ☁️ Remote Configuration  
🔄 Command Alias         ⚠️ Warning/Conflict
✓ Enabled Command        🔌 MCP Server Association
❌ Disabled Command

✅ Configuration is valid!
```

### Port Checking

```bash
interop mcp port-check
```

This shows:
- Which ports are available
- Which ports are in use
- Which processes are using ports

## Logging Levels

Configure verbosity in settings:

```toml
log_level = "verbose"  # Options: error, warning, verbose
```

## Advanced Features

### Executable Search Paths

Interop searches for executables in:
1. Configuration directory (`~/.config/interop/executables/`)
2. Additional paths specified in configuration
3. System PATH

```toml
executable_search_paths = ["~/.local/bin", "~/bin"]
```

## Development

### Project Structure

```
.
├── cmd/
│   └── cli/          # Main application entry point
├── internal/
│   ├── command/      # CLI command implementations
│   ├── display/      # Output formatting utilities
│   ├── edit/         # Project editing functionality
│   ├── logging/      # Logging with color control
│   ├── mcp/          # MCP server implementation
│   ├── project/      # Project management core
│   ├── settings/     # Configuration management
│   └── util/         # Shared utilities
├── dist/             # Distribution files
└── .github/          # GitHub workflows and templates
```

### Testing

Run the test suite:
```bash
go test ./...
```

## Quick Reference

### Remote Configuration Commands

```bash
# Remote repository management
interop config remote add <name> <git-url>     # Add remote repository
interop config remote remove <name>            # Remove remote repository
interop config remote show                     # List all remotes
interop config remote clear                    # Remove all remotes and cached files

# Fetching configurations
interop config remote fetch                    # Fetch from all remotes
interop config remote fetch <name>             # Fetch from specific remote

# Validation and diagnostics
interop validate                               # Comprehensive configuration validation
interop mcp port-check                         # Check MCP server port availability
```

### Configuration File Locations

```bash
~/.config/interop/settings.toml               # Main configuration file
~/.config/interop/config.d/                   # Local configuration directory
~/.config/interop/config.d.remote/            # Remote configurations (auto-managed)
~/.config/interop/executables/                # Local executable files
~/.config/interop/executables.remote/         # Remote executables (auto-managed)
~/.config/interop/versions.toml               # Remote tracking metadata (auto-managed)
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.