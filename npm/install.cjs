#!/usr/bin/env node

// Postinstall script: copies the correct prebuilt binary for the user's platform.

const fs = require("fs");
const path = require("path");

const BIN_DIR = path.join(__dirname, "bin");
const BIN_NAME = process.platform === "win32" ? "construct.exe" : "construct";
const BIN_PATH = path.join(BIN_DIR, BIN_NAME);

function getPlatform() {
  const platformMap = {
    darwin: "darwin",
    linux: "linux",
    win32: "windows",
  };

  const archMap = {
    x64: "amd64",
    arm64: "arm64",
  };

  const os = platformMap[process.platform];
  const cpu = archMap[process.arch];

  if (!os || !cpu) {
    console.error(
      `Unsupported platform: ${process.platform}/${process.arch}. Build from source instead.`
    );
    process.exit(1);
  }

  return { os, cpu };
}

function isNativeBinary(filePath) {
  try {
    const buf = Buffer.alloc(4);
    const fd = fs.openSync(filePath, "r");
    fs.readSync(fd, buf, 0, 4, 0);
    fs.closeSync(fd);
    // Check for Mach-O, ELF, or PE magic bytes
    const magic = buf.toString("hex");
    return (
      magic.startsWith("cffaedfe") || // Mach-O 64-bit
      magic.startsWith("feedface") || // Mach-O 32-bit
      magic.startsWith("7f454c46") || // ELF
      magic.startsWith("4d5a")        // PE (Windows)
    );
  } catch {
    return false;
  }
}

function main() {
  // Only skip if an actual native binary exists (not the Node.js shim)
  if (fs.existsSync(BIN_PATH) && isNativeBinary(BIN_PATH)) {
    return;
  }

  const { os, cpu } = getPlatform();
  const ext = os === "windows" ? ".exe" : "";
  const filename = `construct-${os}-${cpu}${ext}`;
  const srcPath = path.join(__dirname, "binaries", filename);

  if (!fs.existsSync(srcPath)) {
    console.error(`Binary not found for ${os}/${cpu}: ${filename}`);
    console.error(
      "\nBuild from source instead:\n  git clone https://github.com/construct-space/cli\n  cd cli && go build -o construct ."
    );
    process.exit(1);
  }

  fs.mkdirSync(BIN_DIR, { recursive: true });
  fs.copyFileSync(srcPath, BIN_PATH);
  fs.chmodSync(BIN_PATH, 0o755);
  console.log(`construct installed (${os}/${cpu})`);
}

main();
