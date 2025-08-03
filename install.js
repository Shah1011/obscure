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
    
    // Create bin directory
    const binDir = path.join(__dirname, 'bin');
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }
    
    // Check if pre-built binary exists (from GitHub Actions)
    const ext = platform === 'windows' ? '.exe' : '';
    const preBuiltBinaryName = `obscure-${platform}-${arch}${ext}`;
    const preBuiltBinaryPath = path.join(__dirname, preBuiltBinaryName);
    const binaryPath = path.join(binDir, `obscure${ext}`);
    
    if (fs.existsSync(preBuiltBinaryPath)) {
      // Use pre-built binary (from GitHub Actions publish)
      console.log(`Using pre-built binary: ${preBuiltBinaryName}`);
      fs.copyFileSync(preBuiltBinaryPath, binaryPath);
    } else {
      // Download binary (for local development)
      const url = getDownloadUrl(platform, arch);
      console.log(`Downloading ${url}...`);
      await downloadFile(url, binaryPath);
    }
    
    // Make executable on Unix systems
    if (platform !== 'windows') {
      fs.chmodSync(binaryPath, '755');
    }
    
    // Create cross-platform wrapper script
    const wrapperPath = path.join(binDir, 'obscure');
    
    if (platform === 'windows') {
      // Create a Node.js wrapper script that NPM can handle properly
      const nodeWrapperContent = `#!/usr/bin/env node
const { spawn } = require('child_process');
const path = require('path');

const binaryPath = path.join(__dirname, 'obscure.exe');
const child = spawn(binaryPath, process.argv.slice(2), { stdio: 'inherit' });

child.on('exit', (code) => {
  process.exit(code);
});
`;
      fs.writeFileSync(wrapperPath, nodeWrapperContent);
      
      // Also create a .cmd file for direct execution
      const cmdPath = wrapperPath + '.cmd';
      const cmdContent = `@echo off\nnode "%~dp0obscure" %*`;
      fs.writeFileSync(cmdPath, cmdContent);
      
      // Try to manually create NPM symlinks (workaround for NPM Windows issue)
      try {
        const npmPrefix = execSync('npm config get prefix', { encoding: 'utf8' }).trim();
        const globalBinPath = path.join(npmPrefix, 'obscure');
        const globalCmdPath = path.join(npmPrefix, 'obscure.cmd');
        
        // Create the global shell script (for Git Bash, WSL, etc.)
        const globalShellScript = `#!/bin/sh
basedir=$(dirname "$(echo "$0" | sed -e 's,\\\\\\\\,/,g')")
case \`uname\` in
    *CYGWIN*|*MINGW*|*MSYS*) basedir=\`cygpath -w "$basedir"\`;;
esac

if [ -x "$basedir/node" ]; then
  "$basedir/node"  "$basedir/node_modules/obscure-backup/bin/obscure" "$@"
  ret=$?
else 
  node  "$basedir/node_modules/obscure-backup/bin/obscure" "$@"
  ret=$?
fi
exit $ret`;

        // Create the global .cmd script (for Windows Command Prompt and PowerShell)
        const globalCmdScript = `@ECHO off
GOTO start
:find_dp0
SET dp0=%~dp0
EXIT /b
:start
SETLOCAL
CALL :find_dp0

IF EXIST "%dp0%\\node.exe" (
  SET "_prog=%dp0%\\node.exe"
) ELSE (
  SET "_prog=node"
  SET PATHEXT=%PATHEXT:;.JS;=;%
)

"%_prog%"  "%dp0%\\node_modules\\obscure-backup\\bin\\obscure" %*
ENDLOCAL
EXIT /b %errorlevel%`;

        fs.writeFileSync(globalBinPath, globalShellScript);
        fs.writeFileSync(globalCmdPath, globalCmdScript);
        
        console.log('✅ Created global NPM symlinks automatically');
      } catch (error) {
        console.log('⚠️  Could not create global symlinks automatically:', error.message);
        console.log('   This is a known NPM issue on Windows.');
      }
    } else {
      // Unix systems
      const wrapperContent = `#!/bin/sh\nexec "${binaryPath}" "$@"`;
      fs.writeFileSync(wrapperPath, wrapperContent);
      fs.chmodSync(wrapperPath, '755');
    }
    
    console.log('✅ Obscure installed successfully!');
    console.log('');
    
    // Test if the global command works
    try {
      require('child_process').execSync('obscure --help', { stdio: 'ignore', timeout: 5000 });
      console.log('🎉 Global "obscure" command is ready!');
      console.log('Run "obscure --help" to get started.');
    } catch (error) {
      // Check if we're on Windows and symlinks were created
      if (platform === 'windows') {
        try {
          const npmPrefix = execSync('npm config get prefix', { encoding: 'utf8' }).trim();
          const globalCmdPath = path.join(npmPrefix, 'obscure.cmd');
          if (fs.existsSync(globalCmdPath)) {
            console.log('🎉 Global "obscure" command should be ready!');
            console.log('Run "obscure --help" to get started.');
            console.log('(If command not found, try restarting your terminal)');
            return;
          }
        } catch (e) {
          // Fall through to workarounds
        }
      }
      
      console.log('⚠️  Global "obscure" command not found. This is a known NPM issue on Windows.');
      console.log('');
      console.log('🔧 Workarounds:');
      console.log('   1. Try: npm link obscure-backup');
      console.log('   2. Or run directly: npx obscure-backup');
      console.log('   3. Or use full path: node "' + path.join(__dirname, 'bin', 'obscure') + '"');
      console.log('');
      console.log('📖 More info: https://github.com/Shah1011/obscure#installation');
    }
    
  } catch (error) {
    console.error('❌ Installation failed:', error.message);
    console.error('You can download the binary manually from:');
    console.error(`https://github.com/${GITHUB_REPO}/releases`);
    process.exit(1);
  }
}

install();