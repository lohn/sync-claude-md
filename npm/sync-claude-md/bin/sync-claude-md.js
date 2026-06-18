#!/usr/bin/env node
const { spawnSync } = require('child_process');
const path = require('path');

const platformPackages = {
  darwin: {
    arm64: '@sync-claude-md/cli-darwin-arm64',
    x64: '@sync-claude-md/cli-darwin-x64',
  },
  linux: {
    arm64: '@sync-claude-md/cli-linux-arm64',
    x64: '@sync-claude-md/cli-linux-x64',
  },
  win32: {
    arm64: '@sync-claude-md/cli-win32-arm64',
    x64: '@sync-claude-md/cli-win32-x64',
  },
};

const platform = process.platform;
const arch = process.arch;
const pkg = platformPackages[platform]?.[arch];

if (!pkg) {
  console.error(`Error: Unsupported platform: ${platform}-${arch}`);
  console.error('Please install sync-claude-md manually:');
  console.error('  https://github.com/lohn/sync-claude-md/releases');
  process.exit(1);
}

const binaryName =
  platform === 'win32' ? 'sync-claude-md.exe' : 'sync-claude-md';

let binaryPath;
try {
  binaryPath = require.resolve(path.join(pkg, binaryName));
} catch (e) {
  console.error(`Error: Could not find binary for ${pkg}.`);
  console.error(
    'Your platform may not be supported. Please install sync-claude-md manually:',
  );
  console.error('  https://github.com/lohn/sync-claude-md/releases');
  process.exit(1);
}

const result = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: 'inherit',
});
process.exit(result.status ?? 1);
