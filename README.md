# Interop CLI

A Go command-line interface application for managing and organizing your projects.

## Features

- Project management with path validation
- Configurable logging levels
- Settings management using TOML configuration
- Support for both regular and snapshot releases
- Cross-platform support (Linux, Windows, macOS)

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

Example configuration:

```toml
log_level = "warning"

[projects]
[projects.my-project]
path = "~/projects/my-project"
description = "My awesome project"
```

## Usage

### List Projects

To list all configured projects:

```bash
interop projects
```

This will show:
- Project names
- Project paths
- Path validity status
- Whether the path is within the home directory
- Project descriptions (if provided)

### Log Levels

The following log levels are supported:
- `error`: Only shows error messages
- `warning`: Shows errors and warnings
- `verbose`: Shows all messages

## Development

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

## License

This project is licensed under the MIT License - see the LICENSE file for details.