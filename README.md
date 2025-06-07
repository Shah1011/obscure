# Obscure

A secure, encrypted backup tool that supports multiple cloud storage providers. Obscure allows you to create encrypted backups of your data and store them in your own cloud storage buckets (BYOB - Bring Your Own Bucket).

## Features

- 🔐 **End-to-End Encryption**: All backups are encrypted using AES-GCM before being uploaded
- ☁️ **Multiple Cloud Providers**: Support for Amazon S3 and Google Cloud Storage
- 🔄 **Version Control**: Tag and version your backups for easy organization
- 🔍 **Easy Management**: List, restore, and delete backups with simple commands
- 🔒 **Secure**: No cloud provider credentials stored in the cloud - you control your data
- 🚀 **Fast**: Uses efficient compression and streaming for large files

## Installation

1. Download the latest release from the [releases page](https://github.com/Shah1011/obscure/releases)
2. Extract the binary to a location in your PATH
3. Run `obscure signup` to create your account

## Quick Start

1. **Sign up and configure your cloud provider**:
   ```bash
   obscure signup
   ```

2. **Create your first backup**:
   ```bash
   obscure backup --tag=myproject --version=1.0
   ```

3. **List your backups**:
   ```bash
   obscure ls
   ```

4. **Restore a backup**:
   ```bash
   obscure restore myproject/1.0_myproject.obscure
   ```

## Available Commands

### Authentication
- `obscure signup` - Create a new account
- `obscure login` - Log in to your account
- `obscure logout` - Log out of your account
- `obscure whoami` - Show current user info

### Backup Management
- `obscure backup [--tag TAG] [--version VERSION] [--direct]` - Create a new backup
- `obscure restore [backup_path]` - Restore a backup
- `obscure ls` - List all backups
- `obscure rm <filename>` - Delete a specific backup
- `obscure rmdir <tag>` - Delete all backups under a tag

### Cloud Provider Management
- `obscure provider add [s3|gcs]` - Add a new cloud provider
- `obscure provider remove [s3|gcs]` - Remove a cloud provider
- `obscure provider list` - List configured providers
- `obscure switch-provider [s3|gcs]` - Switch active provider
- `obscure which-provider` - Show current provider

### Utility
- `obscure debug` - Show debug information about your session

## Backup Formats

Backups can be created in two formats:
1. **Encrypted** (default): Files are encrypted and compressed (`.obscure` extension)
2. **Direct**: Files are stored as a tar archive without encryption (`.tar` extension)

## Security

- All backups are encrypted using AES-GCM
- Encryption keys are derived from your password using PBKDF2
- Cloud provider credentials are stored locally only
- No sensitive data is stored in the cloud

## Environment Variables

- `AWS_ACCESS_KEY_ID`: Your AWS access key
- `AWS_SECRET_ACCESS_KEY`: Your AWS secret key
- `AWS_REGION`: Your AWS region
- `GOOGLE_APPLICATION_CREDENTIALS`: Path to your GCP service account key

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
