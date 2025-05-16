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
- **AI Integration**: Multiple MCP servers to expose commands to AI assistants
- **Configuration Management**: TOML-based configuration with validation
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

# Command with Arguments
[commands.build-app]
cmd = "go build -o ${output_file} ${package}"
description = "Build a Go application"
is_enabled = true
is_executable = false
arguments = [
  { name = "output_file", type = "string", description = "Output file name", required = true },
  { name = "package", type = "string", description = "Package to build", default = "./cmd/app" }
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

### Using Arguments

```bash
# Named arguments
interop run generate type=component name=Button

# Positional arguments (in order of definition)
interop run generate component Button true
```

## Validation & Diagnostics

### Validate Configuration

```bash
interop validate
```

This validates:
- Project paths exist
- Command references are valid
- No conflicting aliases
- Required command arguments have definitions
- MCP server ports don't conflict

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

## License

This project is licensed under the MIT License - see the LICENSE file for details.