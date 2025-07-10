#!/usr/bin/env node

const fs = require("fs");
const path = require("path");
const https = require("https");
const { getPlatformInfo, getDownloadUrl } = require("../lib/platform");

// Conditionally load optional dependencies
let tar, AdmZip;
try {
  tar = require("tar");
} catch (e) {
  // Will be loaded dynamically when needed
}
try {
  AdmZip = require("adm-zip");
} catch (e) {
  // Will be loaded dynamically when needed
}

// Use a system cache directory for the binary to avoid accidental commits
const os = require("os");
const BINARY_DIR = path.join(os.homedir(), ".cache", "interop-mcp-server", "bin");
const DOWNLOAD_TIMEOUT = 30000; // 30 seconds

/**
 * Downloads a file from a URL
 * @param {string} url - The URL to download from
 * @param {string} dest - The destination file path
 * @returns {Promise<void>}
 */
function downloadFile(url, dest) {
  return new Promise((resolve, reject) => {
    console.log(`Downloading ${url}...`);

    const file = fs.createWriteStream(dest);
    const request = https.get(url, (response) => {
      // Handle redirects
      if (response.statusCode === 302 || response.statusCode === 301) {
        file.close();
        if (fs.existsSync(dest)) {
          fs.unlinkSync(dest);
        }
        return downloadFile(response.headers.location, dest).then(resolve).catch(reject);
      }

      if (response.statusCode !== 200) {
        file.close();
        if (fs.existsSync(dest)) {
          fs.unlinkSync(dest);
        }
        return reject(
          new Error(`Failed to download: ${response.statusCode} ${response.statusMessage}`)
        );
      }

      const totalSize = parseInt(response.headers["content-length"], 10);
      let downloadedSize = 0;

      response.on("data", (chunk) => {
        downloadedSize += chunk.length;
        if (totalSize) {
          const percent = Math.round((downloadedSize / totalSize) * 100);
          process.stdout.write(`\rDownloading... ${percent}%`);
        }
      });

      response.pipe(file);

      file.on("finish", () => {
        file.close(() => {
          console.log("\nDownload completed successfully!");
          resolve();
        });
      });

      file.on("error", (err) => {
        file.close();
        if (fs.existsSync(dest)) {
          fs.unlinkSync(dest);
        }
        reject(err);
      });
    });

    request.on("error", (err) => {
      file.close();
      if (fs.existsSync(dest)) {
        fs.unlinkSync(dest);
      }
      reject(err);
    });

    request.setTimeout(DOWNLOAD_TIMEOUT, () => {
      request.abort();
      file.close();
      if (fs.existsSync(dest)) {
        fs.unlinkSync(dest);
      }
      reject(new Error("Download timeout"));
    });
  });
}

/**
 * Extracts a tar.gz file
 * @param {string} archivePath - Path to the archive
 * @param {string} extractDir - Directory to extract to
 * @returns {Promise<void>}
 */
async function extractTarGz(archivePath, extractDir) {
  if (!tar) {
    try {
      tar = require("tar");
    } catch (e) {
      throw new Error("tar dependency not available. Please install it with: npm install tar");
    }
  }

  console.log("Extracting archive...");
  await tar.extract({
    file: archivePath,
    cwd: extractDir,
    strip: 0,
  });

  console.log("Extraction completed!");
}

/**
 * Extracts a zip file
 * @param {string} archivePath - Path to the archive
 * @param {string} extractDir - Directory to extract to
 * @returns {Promise<void>}
 */
async function extractZip(archivePath, extractDir) {
  if (!AdmZip) {
    try {
      AdmZip = require("adm-zip");
    } catch (e) {
      throw new Error(
        "adm-zip dependency not available. Please install it with: npm install adm-zip"
      );
    }
  }

  console.log("Extracting archive...");
  const zip = new AdmZip(archivePath);
  zip.extractAllTo(extractDir, true);

  console.log("Extraction completed!");
}

/**
 * Makes a file executable
 * @param {string} filePath - Path to the file
 */
function makeExecutable(filePath) {
  if (process.platform !== "win32") {
    fs.chmodSync(filePath, 0o755);
  }
}

/**
 * Main installation function
 */
async function install() {
  try {
    console.log("Installing interop-mcp-server binary...");

    // Get platform info
    const platformInfo = getPlatformInfo();
    console.log(`Platform: ${platformInfo.platform} ${platformInfo.arch}`);

    // Ensure binary directory exists
    if (!fs.existsSync(BINARY_DIR)) {
      fs.mkdirSync(BINARY_DIR, { recursive: true });
    }

    // Check if binary already exists
    const binaryPath = path.join(BINARY_DIR, platformInfo.binaryName);
    if (fs.existsSync(binaryPath)) {
      console.log("Binary already exists, skipping download.");
      return;
    }

    // Download the archive
    const downloadUrl = getDownloadUrl();
    const archiveName = platformInfo.downloadName;
    const archivePath = path.join(BINARY_DIR, archiveName);

    await downloadFile(downloadUrl, archivePath);

    // Verify the archive was downloaded
    if (!fs.existsSync(archivePath)) {
      throw new Error("Archive download failed - file not found");
    }

    // Extract the archive
    if (platformInfo.isWindows) {
      await extractZip(archivePath, BINARY_DIR);
    } else {
      await extractTarGz(archivePath, BINARY_DIR);
    }

    // Verify the binary was extracted
    if (!fs.existsSync(binaryPath)) {
      throw new Error(`Binary extraction failed - ${platformInfo.binaryName} not found`);
    }

    // Make binary executable
    makeExecutable(binaryPath);

    // Clean up archive (with error handling)
    try {
      if (fs.existsSync(archivePath)) {
        fs.unlinkSync(archivePath);
      }
    } catch (cleanupError) {
      console.log("Note: Could not clean up archive file (this is not critical)");
    }

    console.log("✅ interop-mcp-server binary installed successfully!");
    console.log(`Binary location: ${binaryPath}`);
  } catch (error) {
    console.error("❌ Failed to install binary:", error.message);

    // Provide helpful error messages
    if (error.message.includes("ENOTFOUND") || error.message.includes("timeout")) {
      console.error(
        "\n💡 This might be a network issue. Please check your internet connection and try again."
      );
    } else if (error.message.includes("404")) {
      console.error("\n💡 The binary for your platform might not be available yet.");
      console.error(
        "Please check the releases page: https://github.com/yigitozgumus/interop/releases"
      );
    }

    process.exit(1);
  }
}

// Run installation if this script is executed directly
if (require.main === module) {
  install();
}

module.exports = { install };
