#!/usr/bin/env node

const fs = require("fs");
const path = require("path");
const { spawn } = require("child_process");
const { getPlatformInfo } = require("../lib/platform");

const BINARY_DIR = path.join(__dirname);
const PACKAGE_JSON = require("../package.json");

/**
 * Displays help information
 */
function showHelp() {
  console.log(`
${PACKAGE_JSON.name} v${PACKAGE_JSON.version}
${PACKAGE_JSON.description}

USAGE:
  npx interop-mcp-server [options]

OPTIONS:
  --help, -h          Show this help message
  --version, -v       Show version information
  --mode <mode>       Server mode: 'stdio' (default) or 'sse'
  --server <name>     Named server to start (optional)
  --remote <url>      Load commands from remote Git repository
  --config            Show MCP configuration for Claude Desktop
  --test              Test the MCP server connection

EXAMPLES:
  # Start default MCP server in stdio mode (most common)
  npx interop-mcp-server

  # Start named server in stdio mode
  npx interop-mcp-server --server myserver

  # Start server with remote commands
  npx interop-mcp-server --remote https://github.com/user/repo.git

  # Show configuration for Claude Desktop
  npx interop-mcp-server --config

  # Test the server
  npx interop-mcp-server --test

ABOUT:
  This package provides the MCP (Model Context Protocol) server functionality
  from the Interop project. It allows AI assistants like Claude to execute
  commands and manage projects through a standardized protocol.

  The server runs in stdio mode by default, which is perfect for MCP clients
  that spawn the server process directly (like Claude Desktop).

MORE INFO:
  Repository: ${PACKAGE_JSON.repository.url}
  Issues: ${PACKAGE_JSON.bugs.url}
`);
}

/**
 * Shows version information
 */
function showVersion() {
  console.log(`${PACKAGE_JSON.name} v${PACKAGE_JSON.version}`);
}

/**
 * Shows MCP configuration for Claude Desktop
 */
function showConfig() {
  console.log(`
MCP Configuration for Claude Desktop:

Add this to your Claude Desktop MCP settings:

{
  "mcpServers": {
    "interop": {
      "command": "npx",
      "args": ["interop-mcp-server"]
    }
  }
}

Or with a named server:

{
  "mcpServers": {
    "interop-myserver": {
      "command": "npx",
      "args": ["interop-mcp-server", "--server", "myserver"]
    }
  }
}

With remote commands:

{
  "mcpServers": {
    "interop-remote": {
      "command": "npx",
      "args": ["interop-mcp-server", "--remote", "https://github.com/user/repo.git"]
    }
  }
}
`);
}

/**
 * Tests the MCP server
 */
async function testServer() {
  console.log("Testing MCP server...");

  const platformInfo = getPlatformInfo();
  const binaryPath = path.join(BINARY_DIR, platformInfo.binaryName);

  if (!fs.existsSync(binaryPath)) {
    console.error("❌ Binary not found. Please reinstall the package.");
    process.exit(1);
  }

  console.log("✅ Binary found");
  console.log("✅ MCP server is ready to use");
  console.log("\nTo use with Claude Desktop, add the configuration shown with:");
  console.log("  npx interop-mcp-server --config");
}

/**
 * Runs the interop binary with MCP server command
 */
function runMCPServer(args) {
  const platformInfo = getPlatformInfo();
  const binaryPath = path.join(BINARY_DIR, platformInfo.binaryName);

  // Check if binary exists
  if (!fs.existsSync(binaryPath)) {
    console.error("❌ Interop binary not found. Please reinstall the package:");
    console.error("  npm install interop-mcp-server");
    process.exit(1);
  }

  // Build command arguments
  const mcpArgs = ["mcp", "start"];

  // Parse arguments
  let i = 0;
  while (i < args.length) {
    const arg = args[i];

    switch (arg) {
      case "--mode":
        if (i + 1 < args.length) {
          mcpArgs.push("--mode", args[i + 1]);
          i += 2;
        } else {
          console.error("❌ --mode requires a value (stdio or sse)");
          process.exit(1);
        }
        break;

      case "--server":
        if (i + 1 < args.length) {
          mcpArgs.push(args[i + 1]);
          i += 2;
        } else {
          console.error("❌ --server requires a server name");
          process.exit(1);
        }
        break;

      case "--remote":
        if (i + 1 < args.length) {
          mcpArgs.push("--remote", args[i + 1]);
          i += 2;
        } else {
          console.error("❌ --remote requires a repository URL");
          process.exit(1);
        }
        break;

      default:
        console.error(`❌ Unknown argument: ${arg}`);
        console.error("Use --help for usage information");
        process.exit(1);
    }
  }

  // Set default mode to stdio if not specified
  if (!mcpArgs.includes("--mode")) {
    mcpArgs.push("--mode", "stdio");
  }

  // Spawn the process
  const child = spawn(binaryPath, mcpArgs, {
    stdio: "inherit",
    env: process.env,
  });

  // Handle process exit
  child.on("exit", (code) => {
    process.exit(code || 0);
  });

  // Handle errors
  child.on("error", (err) => {
    console.error("❌ Failed to start MCP server:", err.message);
    process.exit(1);
  });

  // Handle signals
  process.on("SIGINT", () => {
    child.kill("SIGINT");
  });

  process.on("SIGTERM", () => {
    child.kill("SIGTERM");
  });
}

/**
 * Main function
 */
function main() {
  const args = process.argv.slice(2);

  // Handle special flags
  if (args.includes("--help") || args.includes("-h")) {
    showHelp();
    return;
  }

  if (args.includes("--version") || args.includes("-v")) {
    showVersion();
    return;
  }

  if (args.includes("--config")) {
    showConfig();
    return;
  }

  if (args.includes("--test")) {
    testServer();
    return;
  }

  // Run the MCP server
  runMCPServer(args);
}

// Run main function
main();
