# open-webui-backup

A secure CLI tool for backing up and restoring data from Open WebUI instances with mandatory encryption.

## Features

- **Unified Commands** - Simple backup/restore/purge with optional type filtering
- **Safe Data Deletion** - Purge command with dry-run mode and confirmation prompts
- **Mandatory Encryption** - All backups secured with age public key cryptography
- **Selective Operations** - Choose specific data types for backup/restore/purge
- **Team Workflows** - Multi-recipient encryption for shared access
- **Environment-based Configuration** - Secure credential management
- **Cross-platform** - Binaries for Linux, macOS, Windows (amd64/arm64)

## Installation

### Build from source

```bash
go build -o owuiback ./cmd/owuiback
```

### Using Task

```bash
task build
```

## Quick Start

### 1. Generate Age Key Pair (One-Time Setup)

```bash
# Generate identity (private key)
age-keygen -o ~/.age/identity.txt

# Extract recipient (public key)
age-keygen -y ~/.age/identity.txt > ~/.age/recipient.txt

# View your public key
cat ~/.age/recipient.txt
# Output: age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
```

### 2. Configure Environment Variables

```bash
export OPEN_WEBUI_URL="https://your-instance.com"
export OPEN_WEBUI_API_KEY="sk-your-api-key"
export OWUI_ENCRYPTED_RECIPIENT=$(cat ~/.age/recipient.txt)
export OWUI_DECRYPT_IDENTITY="$HOME/.age/identity.txt"
```

### 3. Create Backup

```bash
# Full backup (all data types)
./owuiback backup --out ./backups/full-backup.zip

# Selective backup (specific types)
./owuiback backup --out ./backups/kb-only.zip --knowledge
```

### 4. Restore Backup

```bash
# Full restore
./owuiback restore --file ./backups/full-backup.zip.age

# Selective restore
./owuiback restore --file ./backups/full-backup.zip.age --knowledge --models
```

## Available Commands

### backup

Create an encrypted backup of Open WebUI data.

```bash
owuiback backup --out OUTPUT_FILE [OPTIONS]
```

**Required Flags:**
- `--out`, `-o` - Output file path (`.age` extension added automatically)

**Encryption Flags (one required):**
- `--encrypt-recipient` - Age public key (repeatable for multiple recipients)
  - Or use `OWUI_ENCRYPTED_RECIPIENT` environment variable

**Selective Type Flags (optional, default: all types):**
- `--prompts` - Include only prompts
- `--tools` - Include only tools
- `--knowledge` - Include only knowledge bases
- `--models` - Include only models
- `--files` - Include only files

**Examples:**
```bash
# Full backup (all types)
./owuiback backup --out ./backups/full.zip

# Selective backup
./owuiback backup --out ./backups/kb.zip --knowledge
./owuiback backup --out ./backups/prompts-tools.zip --prompts --tools

# With explicit recipient
./owuiback backup --out ./backups/full.zip \
    --encrypt-recipient age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p

# Team backup (multiple recipients)
./owuiback backup --out ./backups/team.zip \
    --encrypt-recipient age1alice... \
    --encrypt-recipient age1bob...
```

### restore

Restore data from an encrypted backup.

```bash
owuiback restore --file INPUT_FILE [OPTIONS]
```

**Required Flags:**
- `--file`, `-f` - Input file path (encrypted .age file)

**Decryption Flags (one required):**
- `--decrypt-identity` - Path to age identity file (repeatable)
  - Or use `OWUI_DECRYPT_IDENTITY` environment variable

**Optional Flags:**
- `--overwrite` - Overwrite existing data (default: false, skips existing items)

**Selective Type Flags (optional, default: all types in backup):**
- `--prompts` - Restore only prompts
- `--tools` - Restore only tools
- `--knowledge` - Restore only knowledge bases
- `--models` - Restore only models
- `--files` - Restore only files

**Examples:**
```bash
# Full restore (all types in backup)
./owuiback restore --file ./backups/full.zip.age

# Selective restore
./owuiback restore --file ./backups/full.zip.age --knowledge
./owuiback restore --file ./backups/full.zip.age --prompts --tools

# With overwrite
./owuiback restore --file ./backups/full.zip.age --overwrite

# With explicit identity file
./owuiback restore --file ./backups/full.zip.age \
    --decrypt-identity ~/.age/identity.txt

# Team restore (any team member's identity works)
./owuiback restore --file ./backups/team.zip.age \
    --decrypt-identity ./alice_identity.txt
```

### purge

Safely delete data from Open WebUI with dry-run and confirmation.

```bash
owuiback purge [OPTIONS]
```

**Optional Flags:**
- `--force`, `-f` - Actually perform deletion (required for actual deletion)

**Selective Type Flags (optional, default: all types in dry-run):**
- `--chats` - Purge only chats
- `--files` - Purge only files
- `--models` - Purge only models
- `--knowledge` - Purge only knowledge bases
- `--prompts` - Purge only prompts
- `--tools` - Purge only tools
- `--functions` - Purge only functions
- `--memories` - Purge only memories
- `--feedbacks` - Purge only feedbacks

**Default Behavior:**
- Without `--force`: Shows what would be deleted (dry-run mode)
- With `--force`: Prompts for confirmation, then deletes
- No type flags: Targets all types
- With type flags: Targets only specified types

**Examples:**
```bash
# Dry-run (shows what would be deleted)
./owuiback purge

# Dry-run for specific types
./owuiback purge --chats --files

# Force deletion (requires typing "yes" to confirm)
./owuiback purge --force

# Force deletion of specific types
./owuiback purge --chats --force
./owuiback purge --knowledge --models --force
```

**Safety Mechanisms:**
1. **Dry-run by Default** - Without `--force`, only shows counts
2. **Force Flag Required** - Must explicitly use `--force` for deletion
3. **Confirmation Prompt** - Must type "yes" to proceed
4. **Clear Warning** - Shows ⚠️  WARNING before deletion
5. **Item Counts** - Displays exactly what will be deleted

**Use Cases:**
- Clean test environments
- Remove all data before migration
- Delete specific data types (e.g., old chats)
- Prepare instance for fresh start

## Encryption

All backups are **mandatory encrypted** using [age](https://age-encryption.org/) - a modern, secure file encryption tool.

### Why Age?

- **Modern Cryptography**: X25519 public key encryption
- **Simple**: Easy key generation and management
- **Team-Friendly**: Multiple recipient support
- **Secure**: Created by cryptography expert Filippo Valsorda

### Key Management

#### Generate Keys

```bash
# Generate identity (private key)
age-keygen -o ~/.age/identity.txt

# Extract recipient (public key) 
age-keygen -y ~/.age/identity.txt > ~/.age/recipient.txt

# Secure the identity file
chmod 600 ~/.age/identity.txt
```

#### Team Setup

Each team member generates their own key pair:

```bash
# Alice
age-keygen -o alice_identity.txt
ALICE_KEY=$(age-keygen -y alice_identity.txt)

# Bob
age-keygen -o bob_identity.txt
BOB_KEY=$(age-keygen -y bob_identity.txt)

# Create backup for both
./owuiback backup --out team-backup.zip \
    --encrypt-recipient $ALICE_KEY \
    --encrypt-recipient $BOB_KEY

# Either Alice or Bob can restore
./owuiback restore --file team-backup.zip.age --decrypt-identity alice_identity.txt
# OR
./owuiback restore --file team-backup.zip.age --decrypt-identity bob_identity.txt
```

### Encrypted File Format

All backups automatically get `.age` extension:

```
Input:  full-backup.zip
Output: full-backup.zip.age (encrypted)
```

The encrypted file uses age armor format (ASCII-armored) for safe storage and transmission.

## Configuration

### Environment Variables

**Required:**
- `OPEN_WEBUI_URL` - Your Open WebUI instance URL
- `OPEN_WEBUI_API_KEY` - API key for authentication
- `OWUI_ENCRYPTED_RECIPIENT` - Age public key for backup (or use `--encrypt-recipient`)
- `OWUI_DECRYPT_IDENTITY` - Path to age identity file for restore (or use `--decrypt-identity`)

**Optional:**
- `LOG_LEVEL` - Logging level (debug, info, warn, error)

### Example Configuration

```bash
# Create .env file or add to .bashrc/.zshrc
export OPEN_WEBUI_URL="https://openwebui.example.com"
export OPEN_WEBUI_API_KEY="sk-xxxxxxxxxxxxxxxxxxxxxx"
export OWUI_ENCRYPTED_RECIPIENT="age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p"
export OWUI_DECRYPT_IDENTITY="$HOME/.age/identity.txt"
```

## Usage Examples

### Basic Workflows

```bash
# Daily backup
./owuiback backup --out ./backups/daily-$(date +%Y%m%d).zip

# Backup only knowledge bases
./owuiback backup --out ./backups/knowledge-only.zip --knowledge

# Backup prompts and tools
./owuiback backup --out ./backups/prompts-tools.zip --prompts --tools

# Restore everything
./owuiback restore --file ./backups/daily-20251025.zip.age

# Restore only models
./owuiback restore --file ./backups/daily-20251025.zip.age --models

# Force overwrite existing data
./owuiback restore --file ./backups/daily-20251025.zip.age --overwrite
```

### Automated Backup Script

```bash
#!/bin/bash
# automated-backup.sh

set -e

BACKUP_DIR="/secure/backups"
RETENTION_DAYS=30
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/owui-backup-$DATE.zip"

# Create backup
./owuiback backup --out "$BACKUP_FILE"

# Verify backup was created
if [ -f "$BACKUP_FILE.age" ]; then
    echo "✓ Backup created: $BACKUP_FILE.age"
    
    # Test restore (dry-run - restore to verify, then delete test restore)
    # Note: Implement your own verification logic
    
    # Cleanup old backups
    find "$BACKUP_DIR" -name "*.age" -mtime +$RETENTION_DAYS -delete
    echo "✓ Cleaned up backups older than $RETENTION_DAYS days"
else
    echo "✗ Backup failed"
    exit 1
fi
```

### Docker Usage

```bash
# Build image
docker build -t owuiback:latest .

# Run backup
docker run --rm \
  -e OPEN_WEBUI_URL="https://instance.com" \
  -e OPEN_WEBUI_API_KEY="sk-xxx" \
  -e OWUI_ENCRYPTED_RECIPIENT="age1..." \
  -v $(pwd)/backups:/backups \
  owuiback:latest backup --out /backups/backup.zip

# Run restore
docker run --rm \
  -e OPEN_WEBUI_URL="https://instance.com" \
  -e OPEN_WEBUI_API_KEY="sk-xxx" \
  -e OWUI_DECRYPT_IDENTITY="/keys/identity.txt" \
  -v $(pwd)/backups:/backups \
  -v $(pwd)/.age:/keys \
  owuiback:latest restore --file /backups/backup.zip.age
```

## Data Types

### Knowledge Bases
- Complete metadata (name, description, access control)
- All document files with original filenames
- File relationships and IDs
- Heterogeneous types (file and collection)

### Models
- Model metadata and configuration
- Associated knowledge bases
- Model files (configurations, weights)
- Embedded KB handling with ID mapping

### Tools
- Tool definitions (name, ID, content)
- Tool metadata and access control
- Creation and update timestamps

### Prompts
- Prompt definitions (command, title, content)
- Access control settings
- Creation and update timestamps

### Files
- File metadata (filename, hash, size)
- Complete file content/data
- Access control and timestamps

## Backup Format

### Structure

```
backup.zip.age (encrypted)
  └─ backup.zip (decrypts to)
      ├── owui.json                  # Backup metadata
      ├── knowledge-bases/
      │   └── kb-id-1/
      │       ├── knowledge_base.json
      │       └── documents/
      │           └── file.pdf
      ├── models/
      │   └── model-id-1/
      │       ├── model.json
      │       ├── model-files/
      │       └── knowledge-bases/
      ├── tools/
      │   └── tool-id-1/
      │       └── tool.json
      ├── prompts/
      │   └── command-1/
      │       └── prompt.json
      └── files/
          └── file-id-1/
              ├── file.json
              └── content/
```

### Metadata

Every backup includes `owui.json` with:
- Open WebUI instance URL
- Backup timestamp
- Data types included
- Item counts
- Tool version

## Restore Behavior

### Default (Without --overwrite)
- Existing items are **skipped**
- New items are added
- Safe for incremental updates
- Preserves local modifications

### With --overwrite Flag
- Existing items are **replaced** with backup versions
- New items are added
- Use when reverting to backup state
- Overwrites all local modifications

### Idempotent Operations

All restore operations are idempotent - safe to run multiple times:

```bash
# First run: Restores all items
./owuiback restore --file backup.zip.age

# Second run: Skips existing, adds only new items
./owuiback restore --file backup.zip.age

# With overwrite: Replaces everything with backup versions
./owuiback restore --file backup.zip.age --overwrite
```

## Security Best Practices

1. **Protect Identity Files**
   ```bash
   chmod 600 ~/.age/identity.txt
   ```

2. **Use Environment Variables**
   - Don't embed keys in scripts
   - Use environment variables or secret managers

3. **Store Backups Securely**
   - Encrypted backups are safe for cloud storage
   - Still protect from unauthorized access

4. **Key Rotation**
   - Generate new keys periodically
   - Re-encrypt old backups with new keys if needed

5. **Test Restores**
   - Periodically verify you can restore from backups
   - Test with non-production instance first

6. **Multi-recipient for Teams**
   - Each team member has their own identity
   - Can revoke access by creating new backup without that recipient

## Troubleshooting

### "API key is required"
- Set `OPEN_WEBUI_API_KEY` environment variable
- Verify API key is valid and has necessary permissions

### "Failed to get encryption recipients"
- Set `OWUI_ENCRYPTED_RECIPIENT` environment variable
- Or use `--encrypt-recipient` flag with a valid age public key
- Generate keys with: `age-keygen`

### "Failed to get decryption identity files"
- Set `OWUI_DECRYPT_IDENTITY` environment variable
- Or use `--decrypt-identity` flag with path to identity file
- Verify identity file exists and is readable

### "Failed to decrypt: no identity matched"
- Ensure identity file matches one of the backup's recipients
- Check if using correct identity file
- Verify file is not corrupted

### "age command not found" (Docker)
- Rebuild Docker image with updated Dockerfile
- Image should include age package in Alpine runtime

### "Failed to connect to Open WebUI"
- Check `OPEN_WEBUI_URL` is correct
- Verify Open WebUI instance is running and accessible
- Check network connectivity

## Migration from v1.0

If you're upgrading from v1.0, note these **breaking changes**:

### Command Changes
```bash
# OLD (v1.0)
./owuiback backup-all --dir ./backups
./owuiback restore-all --dir ./backups/backup.zip

# NEW (v2.0)
./owuiback backup --out ./backups/backup.zip
./owuiback restore --file ./backups/backup.zip.age
```

### Encryption Changes
```bash
# OLD (v1.0) - Optional encryption
./owuiback backup-all --dir ./backups --encrypt

# NEW (v2.0) - Mandatory encryption
./owuiback backup --out ./backups/backup.zip
# Always creates backup.zip.age
```

### Environment Variables
```bash
# OLD (v1.0)
export AGE_PASSPHRASE="secret"

# NEW (v2.0)
export OWUI_ENCRYPTED_RECIPIENT="age1..."
export OWUI_DECRYPT_IDENTITY="/path/to/identity.txt"
```

### Key Differences

1. **Commands**: 12 commands → 3 commands (backup, restore, purge)
2. **Encryption**: Optional → Mandatory (always encrypted)
3. **Modes**: Passphrase + Public key → Public key only
4. **Flags**: `--dir` → `--out` (backup) / `--file` (restore)
5. **New**: Purge command for safe data deletion

### Compatibility

- ✅ Backup file format unchanged (still ZIP)
- ✅ v1.0 encrypted backups work with v2.0 restore
- ❌ CLI interface NOT backward compatible
- ❌ Must update scripts and automation

## Development

### Build Requirements
- Go 1.24.7 or higher
- age CLI tool
- Docker (optional, for containerization)

### Local Development

```bash
# Install dependencies
task bootstrap

# Run locally
task run -- backup --out ./test.zip

# Build all platforms
task build

# Run in Docker
task docker-run
```

### Testing

```bash
# Run integration tests
export OWUI_URL="http://localhost:8080"
export OWUI_API_KEY="your-key"
go test -v ./test/integration

# Run specific test
go test -v ./test/integration -run TestBackupCommand
```

## Architecture

- **3 Unified Commands**: backup, restore, purge
- **Selective Type Filtering**: Optional flags for specific data types
- **Mandatory Encryption**: age public key cryptography
- **Purge Safety**: Dry-run by default, force mode with confirmation
- **Plugin System**: Extensible architecture for future features
- **Stateless**: No local database, operates directly on API

## License

See LICENSE file for details.

## Contributing

Contributions welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## Support

- GitHub Issues: Report bugs and request features
- Documentation: Check this README and memory-bank/
- Examples: See usage examples above

## Version

Current: **v2.0.0** (Major architectural refactor)

See [CHANGELOG](CHANGELOG.md) for version history and release notes.
