# Interop CLI

A Go command-line interface application for managing and organizing your projects.

## Features

- Project management with path validation
- Configurable logging levels
- Settings management using TOML configuration
- Support for both regular and snapshot releases
- Cross-platform support (Linux, Windows, macOS)
- Beautiful and readable project listing
- Custom command execution with project association
- MCP (Model Context Protocol) server integration for AI assistant support

## Installation

### Using Homebrew

To install Interop using Homebrew, run the following command:

```bash
brew install yigitozgumus/formulae/interop
```

## Configuration

Interop uses a TOML configuration file located at `~/.config/interop/settings.toml`. The configuration includes:

- Log level settings
- Project configurations with paths and descriptions
- Custom commands with project associations

Example configuration:

```toml
log_level = "warning"

# Additional paths to search for executables
executable_search_paths = ["~/.local/bin", "~/bin"]

[projects]
[projects.my-project]
path = "~/projects/my-project"
description = "My awesome project"
commands = [
  {command_name = "build", alias = "b"},
  {command_name = "deploy"}
]

[commands]
[commands.build]
cmd = "go build ./..."
description = "Build the project"
is_enabled = true
is_executable = false

[commands.deploy]
cmd = "deploy.sh"
description = "Deploy the project"
is_enabled = true
is_executable = true
```

## Usage

### List Projects

To list all configured projects:

```bash
interop projects
```

This will show a beautifully formatted list of your projects with their associated commands:

```
PROJECTS:
=========

📁 Name: my-project
   Path: ~/projects/my-project
   Status: Valid: ✓  |  In $HOME: ✓
   Description: My awesome project
   Commands:
      ⚡ build
         Build the project
      ⚡ deploy (alias: dep)
         Deploy the project

📁 Name: another-project
   Path: /opt/projects/another
   Status: Valid: ✓  |  In $HOME: ✗
```

The output includes:
- Project name with a folder icon
- Project path
- Path validity status (✓ or ✗)
- Whether the path is within the home directory (✓ or ✗)
- Project descriptions (if provided)
- Commands associated with the project and their aliases

### List Commands

To list all configured commands:

```bash
interop commands
```

This will show a formatted list of your commands:

```
COMMANDS:
=========

⚡ Name: build
   Status: Enabled: ✓  |  Source: Script
   Description: Build the project

⚡ Name: deploy
   Status: Enabled: ✓  |  Source: Executables
   Description: Deploy the project
```

### Execute Commands

To execute a command or alias:

```bash
interop run <command-or-alias>
```

The system will:
1. Validate the command configuration 
2. Resolve the command or alias to determine its type
3. For project-specific commands, change to the project directory before execution
4. Execute the command and return to the original directory when done

Commands can be:
- Global commands (not tied to specific projects)
- Project-specific commands (tied to one project)
- Commands with aliases for project-specific usage
- Regular shell commands (executed via shell)
- Executable files (from the executables directory)
- Enabled/disabled as needed

### Command Types and Execution

Interop supports several ways to specify and execute commands:

1. **Shell Commands**: Regular commands executed via the user's shell (specified by the `SHELL` environment variable).
   ```toml
   [commands.list]
   cmd = "ls -la"
   ```

2. **Shell Aliases**: Run aliases defined in your shell's configuration (like .bashrc, .zshrc, or fish config).
   ```toml
   [commands.gitstat]
   cmd = "alias:gst"  # Runs the 'gst' alias from your shell
   ```

3. **Local Script Commands**: Scripts that start with `./` are executed directly from the project directory.
   ```toml
   [commands.build]
   cmd = "./gradlew :app:assembleDebug"
   ```

4. **Executable Commands**: Executables are searched for in multiple locations when `is_executable = true`:
   ```toml
   [commands.deploy]
   cmd = "deploy.sh"
   is_executable = true
   ```
   
   The search order is:
   - Interop's executables directory (`~/.config/interop/executables/`)
   - Additional paths specified in configuration (`executable_search_paths`)
   - System PATH

### MCP Server Integration

Interop includes a built-in MCP (Model Context Protocol) server for integration with AI assistants:

```bash
# Start the MCP server
interop mcp start

# Check the status of the MCP server
interop mcp status

# Stop the MCP server
interop mcp stop

# Restart the MCP server
interop mcp restart
```

The MCP server exposes your commands as tools that can be invoked by AI assistants supporting the Model Context Protocol. This allows AI assistants to:

- List and execute your configured commands
- Access project information
- Run commands in the context of specific projects

For Claude and other MCP-compatible AI assistants, the server automatically configures itself to ensure compatibility by disabling color output when needed.

### Executable Search Paths

You can specify additional directories to search for executables:

```toml
executable_search_paths = ["~/.local/bin", "~/bin"]
```

This is useful for:
- Custom scripts in your home directory
- Local development tools
- System-wide executables not in standard PATH

### Validate Configuration

To check your configuration for errors:

```bash
interop validate
```

This will validate that:
- All command references in projects exist
- No command is bound to multiple projects without an alias
- Aliases are unique across projects

### Edit Configuration

To edit the configuration file:

```bash
interop edit
```

This will open the configuration file in your default editor.

### Log Levels

The following log levels are supported:
- `error`: Only shows error messages
- `warning`: Shows errors and warnings
- `verbose`: Shows all messages

The output is colorized for better readability in terminals, but colors are automatically disabled when:
- Output is not directed to a terminal
- Integration with programmatic tools like MCP server requires plain text

## Project Structure

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

## Development

### Prerequisites

- Go 1.21 or later
- Make (optional, for using Makefile commands)

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

### Testing

Run the test suite:
```bash
go test ./...
```

### Release Process

The project uses GoReleaser for creating releases. See [README-releases.md](README-releases.md) for detailed information about the release process.

## License

This project is licensed under the MIT License - see the LICENSE file for details.