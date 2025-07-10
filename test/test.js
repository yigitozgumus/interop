#!/usr/bin/env node

const fs = require("fs");
const path = require("path");
const { getPlatformInfo } = require("../lib/platform");

/**
 * Simple test suite for interop-mcp-server
 */
function runTests() {
  console.log("Running interop-mcp-server tests...\n");

  let passed = 0;
  let failed = 0;

  // Test 1: Platform detection
  try {
    const platformInfo = getPlatformInfo();
    console.log("✅ Platform detection test passed");
    console.log(`   Platform: ${platformInfo.platform}`);
    console.log(`   Architecture: ${platformInfo.arch}`);
    console.log(`   Binary: ${platformInfo.binaryName}`);
    console.log(`   Download: ${platformInfo.downloadName}`);
    passed++;
  } catch (error) {
    console.log("❌ Platform detection test failed:", error.message);
    failed++;
  }

  // Test 2: Package.json validation
  try {
    const packageJson = require("../package.json");
    if (packageJson.name && packageJson.version && packageJson.bin) {
      console.log("✅ Package.json validation passed");
      console.log(`   Name: ${packageJson.name}`);
      console.log(`   Version: ${packageJson.version}`);
      passed++;
    } else {
      throw new Error("Missing required fields in package.json");
    }
  } catch (error) {
    console.log("❌ Package.json validation failed:", error.message);
    failed++;
  }

  // Test 3: CLI script exists
  try {
    const cliPath = path.join(__dirname, "..", "bin", "interop-mcp-server.js");
    if (fs.existsSync(cliPath)) {
      console.log("✅ CLI script exists");
      console.log(`   Path: ${cliPath}`);
      passed++;
    } else {
      throw new Error("CLI script not found");
    }
  } catch (error) {
    console.log("❌ CLI script test failed:", error.message);
    failed++;
  }

  // Test 4: Install script exists
  try {
    const installPath = path.join(__dirname, "..", "scripts", "install-binary.js");
    if (fs.existsSync(installPath)) {
      console.log("✅ Install script exists");
      console.log(`   Path: ${installPath}`);
      passed++;
    } else {
      throw new Error("Install script not found");
    }
  } catch (error) {
    console.log("❌ Install script test failed:", error.message);
    failed++;
  }

  // Test 5: Library exports
  try {
    const lib = require("../lib/index");
    if (lib.getPlatformInfo && lib.getDownloadUrl && lib.install) {
      console.log("✅ Library exports test passed");
      passed++;
    } else {
      throw new Error("Missing required exports from library");
    }
  } catch (error) {
    console.log("❌ Library exports test failed:", error.message);
    failed++;
  }

  // Summary
  console.log("\n" + "=".repeat(50));
  console.log(`Test Results: ${passed} passed, ${failed} failed`);

  if (failed === 0) {
    console.log("🎉 All tests passed!");
    process.exit(0);
  } else {
    console.log("💥 Some tests failed!");
    process.exit(1);
  }
}

// Run tests if this script is executed directly
if (require.main === module) {
  runTests();
}

module.exports = { runTests };
