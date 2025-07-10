const { getPlatformInfo, getDownloadUrl } = require("./platform");
const { install } = require("../scripts/install-binary");

module.exports = {
  getPlatformInfo,
  getDownloadUrl,
  install,
};
