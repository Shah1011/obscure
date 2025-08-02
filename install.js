#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const https = require('https');
const { execSync } = require('child_process');

const GITHUB_REPO = 'Shah1011/obscure';
const VERSION = require('./package.json').version;

// Platform mapping
const PLATFORM_MAPPING = {
  'darwin': 'darwin',
  'linux': 'linux', 
  'win32': 'windows'
};

const ARCH_MAPPING = {
  'x64': 'amd64',
  'arm64': 'arm64'
};

function getPlatform() {
  const platform = PLATFORM_MAPPING[process.platform];
  const arch = ARCH_MAPPING[process.arch];
  
  if (!platform || !arch) {
    throw new Error(`Unsupported platform: ${process.platform}-${process.arch}`);
  }
  
  return { platform, arch };
}

function getDownloadUrl(platform, arch) {
  const ext = platform === 'windows' ? '.exe' : '';
  const filename = `obscure-${platform}-${arch}${ext}`;
  return `https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/${filename}`;
}

function downloadFile(url, dest) {
  return new Promise((resolve, reject) => {
    console.log(`Downloading ${url}...`);
    
    const file = fs.createWriteStream(dest);
    
    https.get(url, (response) => {
      if (response.statusCode === 302 || response.statusCode === 301) {
        // Handle redirect
        return downloadFile(response.headers.location, dest).then(resolve).catch(reject);
      }
      
      if (response.statusCode !== 200) {
        reject(new Error(`Download failed: ${response.statusCode} ${response.statusMessage}`));
        return;
      }
      
      response.pipe(file);
      
      file.on('finish', () => {
        file.close();
        resolve();
      });
      
      file.on('error', (err) => {
        fs.unlink(dest, () => {}); // Delete the file on error
        reject(err);
      });
    }).on('error', reject);
  });
}

async function install() {
  try {
    const { platform, arch } = getPlatform();
    const url = getDownloadUrl(platform, arch);
    
    // Create bin directory
    const binDir = path.join(__dirname, 'bin');
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }
    
    // Download binary
    const ext = platform === 'windows' ? '.exe' : '';
    const binaryPath = path.join(binDir, `obscure${ext}`);
    
    await downloadFile(url, binaryPath);
    
    // Make executable on Unix systems
    if (platform !== 'windows') {
      fs.chmodSync(binaryPath, '755');
    }
    
    // Create cross-platform wrapper script
    const wrapperPath = path.join(binDir, 'obscure');
    const wrapperContent = platform === 'windows' 
      ? `@echo off\n"${binaryPath}" %*`
      : `#!/bin/sh\nexec "${binaryPath}" "$@"`;
    
    fs.writeFileSync(wrapperPath, wrapperContent);
    
    if (platform !== 'windows') {
      fs.chmodSync(wrapperPath, '755');
    }
    
    console.log('✅ Obscure installed successfully!');
    console.log('Run "obscure --help" to get started.');
    
  } catch (error) {
    console.error('❌ Installation failed:', error.message);
    console.error('You can download the binary manually from:');
    console.error(`https://github.com/${GITHUB_REPO}/releases`);
    process.exit(1);
  }
}

install();