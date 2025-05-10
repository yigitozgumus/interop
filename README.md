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

[projects]
[projects.my-project]
path = "~/projects/my-project"
description = "My awesome project"

[commands]
[commands.build]
cmd = "go build ./..."
description = "Build the project"
is_enabled = true
is_executable = false
projects = ["my-project"]

[commands.deploy]
cmd = "deploy.sh"
description = "Deploy the project"
is_enabled = true
is_executable = true
projects = ["my-project"]
```

## Usage

### List Projects

To list all configured projects:

```bash
interop projects
```

This will show a beautifully formatted list of your projects:

```
PROJECTS:
=========

📁 Name: my-project
   Path: ~/projects/my-project
   Status: Valid: ✓  |  In $HOME: ✓
   Description: My awesome project

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
   Projects: [my-project]
   Description: Build the project

⚡ Name: deploy
   Status: Enabled: ✓  |  Source: Executables
   Projects: [my-project]
   Description: Deploy the project
```

### Execute Commands

To execute a configured command:

```bash
interop command run <command-name>
```

Commands can be:
- Regular shell commands (executed via shell)
- Executable files (from the executables directory)
- Associated with specific projects
- Enabled/disabled as needed

### Log Levels

The following log levels are supported:
- `error`: Only shows error messages
- `warning`: Shows errors and warnings
- `verbose`: Shows all messages

## Project Structure

```
.
├── cmd/
│   └── cli/          # Main application entry point
├── internal/
│   ├── command/      # CLI command implementations
│   ├── edit/         # Project editing functionality
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