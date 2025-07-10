#!/usr/bin/env node

const fs = require("fs");
const path = require("path");

/**
 * Prepare script that runs before publishing
 */
function prepare() {
  console.log("Preparing interop-mcp-server package...");

  // Ensure directories exist
  const dirs = ["bin", "lib", "scripts"];
  dirs.forEach((dir) => {
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
      console.log(`Created directory: ${dir}`);
    }
  });

  // Make CLI script executable
  const cliScript = path.join("bin", "interop-mcp-server.js");
  if (fs.existsSync(cliScript)) {
    fs.chmodSync(cliScript, 0o755);
    console.log("Made CLI script executable");
  }

  // Make install script executable
  const installScript = path.join("scripts", "install-binary.js");
  if (fs.existsSync(installScript)) {
    fs.chmodSync(installScript, 0o755);
    console.log("Made install script executable");
  }

  console.log("✅ Package preparation completed!");
}

// Run preparation if this script is executed directly
if (require.main === module) {
  prepare();
}

module.exports = { prepare };
