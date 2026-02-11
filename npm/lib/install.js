#!/usr/bin/env node
"use strict";

// Postinstall script — downloads the prebuilt SAME binary from GitHub Releases.
// Uses only Node built-ins. ES5 syntax for Node 14 compat.

var https = require("https");
var http = require("http");
var fs = require("fs");
var os = require("os");
var path = require("path");

var VERSION = require("../package.json").version;
var BASE_URL =
  "https://github.com/sgx-labs/statelessagent/releases/download/v" + VERSION;

var PLATFORM_MAP = {
  "darwin-arm64": "darwin-arm64",
  "darwin-x64": "darwin-arm64", // Rosetta fallback
  "linux-x64": "linux-amd64",
  "win32-x64": "windows-amd64.exe",
};

function getBinarySuffix() {
  var key = os.platform() + "-" + os.arch();
  var suffix = PLATFORM_MAP[key];
  if (!suffix) {
    console.error(
      "[same] Unsupported platform: " +
        key +
        ". Supported: darwin-arm64, linux-x64, win32-x64"
    );
    return null;
  }
  return suffix;
}

function download(url, dest, cb) {
  var mod = url.indexOf("https") === 0 ? https : http;
  mod
    .get(url, function (res) {
      // Follow redirects (GitHub → CDN)
      if (
        (res.statusCode === 301 || res.statusCode === 302) &&
        res.headers.location
      ) {
        return download(res.headers.location, dest, cb);
      }
      if (res.statusCode !== 200) {
        cb(new Error("HTTP " + res.statusCode + " from " + url));
        return;
      }
      var file = fs.createWriteStream(dest);
      res.pipe(file);
      file.on("finish", function () {
        file.close(cb);
      });
      file.on("error", function (err) {
        fs.unlink(dest, function () {});
        cb(err);
      });
    })
    .on("error", function (err) {
      cb(err);
    });
}

function main() {
  var binDir = path.join(__dirname, "..", "bin");
  var isWindows = os.platform() === "win32";
  var binaryName = isWindows ? "same-binary.exe" : "same-binary";
  var dest = path.join(binDir, binaryName);

  // Idempotent — skip if binary already exists
  if (fs.existsSync(dest)) {
    console.log("[same] Binary already exists, skipping download.");
    return;
  }

  var suffix = getBinarySuffix();
  if (!suffix) {
    // Unsupported platform — exit 0 so npm install doesn't fail
    process.exit(0);
  }

  var url = BASE_URL + "/same-" + suffix;

  // Ensure bin/ exists
  try {
    fs.mkdirSync(binDir, { recursive: true });
  } catch (e) {
    // Node 8 compat: recursive may not be supported
    if (e.code !== "EEXIST") {
      try {
        fs.mkdirSync(binDir);
      } catch (e2) {
        if (e2.code !== "EEXIST") throw e2;
      }
    }
  }

  console.log("[same] Downloading SAME v" + VERSION + " for " + os.platform() + "/" + os.arch() + "...");

  download(url, dest, function (err) {
    if (err) {
      console.error("[same] Download failed: " + err.message);
      console.error("[same] The binary will be downloaded on first run.");
      // Clean up partial download
      try {
        fs.unlinkSync(dest);
      } catch (e) {}
      // Exit 0 so npm install succeeds — shim will retry on first run
      process.exit(0);
    }

    // chmod 755 on Unix
    if (!isWindows) {
      try {
        fs.chmodSync(dest, 0o755);
      } catch (e) {}
    }

    console.log("[same] Installed successfully.");
  });
}

main();
