const os = require("os");

/**
 * Gets the platform-specific binary information
 * @returns {Object} Object containing platform, arch, and binary name
 */
function getPlatformInfo() {
  const platform = os.platform();
  const arch = os.arch();

  let binaryName;
  let downloadName;

  switch (platform) {
    case "darwin":
      if (arch === "arm64") {
        binaryName = "interop";
        downloadName = "interop_Darwin_arm64.tar.gz";
      } else {
        binaryName = "interop";
        downloadName = "interop_Darwin_x86_64.tar.gz";
      }
      break;
    case "linux":
      if (arch === "arm64") {
        binaryName = "interop";
        downloadName = "interop_Linux_arm64.tar.gz";
      } else {
        binaryName = "interop";
        downloadName = "interop_Linux_x86_64.tar.gz";
      }
      break;
    case "win32":
      binaryName = "interop.exe";
      downloadName = "interop_Windows_x86_64.zip";
      break;
    default:
      throw new Error(`Unsupported platform: ${platform}`);
  }

  return {
    platform,
    arch,
    binaryName,
    downloadName,
    isWindows: platform === "win32",
  };
}

/**
 * Gets the download URL for the latest release
 * @param {string} version - The version to download (default: 'latest')
 * @returns {string} The download URL
 */
function getDownloadUrl(version = "latest") {
  const { downloadName } = getPlatformInfo();
  const baseUrl = "https://github.com/yigitozgumus/interop/releases";

  if (version === "latest") {
    return `${baseUrl}/latest/download/${downloadName}`;
  } else {
    return `${baseUrl}/download/${version}/${downloadName}`;
  }
}

module.exports = {
  getPlatformInfo,
  getDownloadUrl,
};
