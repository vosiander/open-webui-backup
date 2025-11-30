# open-webui-backup

Secure backup and restore tool for Open WebUI with mandatory age encryption.

## Features

- **Unified Commands** - Simple backup/restore/purge with selective type filtering
- **Mandatory Encryption** - All backups secured with age public key cryptography
- **Web Interface** - Built-in UI for backup/restore operations with real-time progress
- **Selective Operations** - Filter specific data types (knowledge, models, tools, prompts, files, chats)
- **Team Workflows** - Multi-recipient encryption for shared access
- **Safe Deletion** - Purge command with dry-run mode and confirmation prompts

## Installation

```bash
# Build from source
go build -o owuiback ./cmd/owuiback

# Or use Task
task build
```

## Quick Start

### Simplified Workflow (Recommended)

The new `full-backup` and `verify` commands handle age key management automatically:

```bash
# 1. Set up environment
export OPEN_WEBUI_URL="https://your-instance.com"
export OPEN_WEBUI_API_KEY="sk-your-api-key"

# 2. Create backup (generates keys automatically)
./owuiback full-backup --path ./backups

# 3. Verify backup
./owuiback verify --path ./backups

# 4. Restore backup (when needed)
./owuiback restore --file ./backups/backup-*.zip.age \
    --decrypt-identity ./backups/identity.txt
```

### Advanced Workflow (Manual Key Management)

```bash
# 1. Generate Age Keys (One-Time)
./owuiback new-identity --path ./my-keys
# Or use age-keygen:
# age-keygen -o ~/.age/identity.txt
# age-keygen -y ~/.age/identity.txt > ~/.age/recipient.txt

# 2. Configure Environment
export OPEN_WEBUI_URL="https://your-instance.com"
export OPEN_WEBUI_API_KEY="sk-your-api-key"
export OWUI_ENCRYPTED_RECIPIENT=$(cat ./my-keys/recipient.txt)
export OWUI_DECRYPT_IDENTITY="./my-keys/identity.txt"

# 3. Create backup
./owuiback backup --out ./backups/full.zip

# 4. Restore backup
./owuiback restore --file ./backups/full.zip.age

# 5. Web interface
./owuiback serve
# Open http://localhost:8080
```

## Commands

### new-identity

Generate a new age identity keypair and save to files.

```bash
# Generate new identity in specified directory
./owuiback new-identity --path ./my-keys

# Identity files created:
# - identity.txt (private key, 600 permissions)
# - recipient.txt (public key)
```

**Flags:**
- `--path` - Directory to save identity files (required)

**Notes:**
- Will not overwrite existing identity files
- Private key saved with restrictive 600 permissions
- Prints example environment variable commands

### full-backup

Create a backup with automatic age encryption and identity management.

```bash
# Full backup (generates keys if needed)
./owuiback full-backup --path ./backups

# Selective backup
./owuiback full-backup --path ./backups --knowledge --models
./owuiback full-backup --path ./backups --prompts --tools

# All files in same directory:
# - identity.txt (created if missing)
# - recipient.txt (created if missing)
# - backup-YYYYMMDD-HHMMSS.zip.age
```

**Flags:**
- `--path` - Directory for identity files and backup output (required)
- `--prompts`, `--tools`, `--knowledge`, `--models`, `--files`, `--chats`, `--users`, `--groups`, `--feedbacks` - Selective types (default: all)

**Features:**
- Auto-generates identity keypair if missing
- Reuses existing identity if present
- Creates timestamped backup files
- Automatic encryption with generated keys
- No need to manage recipients manually

### verify

Verify that a backup file can be decrypted and optionally validate its contents.

```bash
# Verify newest backup in directory
./owuiback verify --path ./backups

# Verify specific backup file
./owuiback verify --path ./backups --file backup-20240101-120000.zip.age

# Only check decryption (skip content validation)
./owuiback verify --path ./backups --only-encryption

# Verify shows:
# - Decryption success/failure
# - Backup metadata (type, timestamp, version)
# - Item counts by type
```

**Flags:**
- `--path` - Directory containing identity.txt and backup files (required)
- `--file` - Specific backup file to verify (optional, auto-detects newest .age file)
- `--only-encryption` - Only verify decryption, skip content validation

**Features:**
- Auto-detects newest backup if --file not specified
- Validates ZIP structure and metadata
- Counts items by type
- Works with both encrypted and unencrypted backups
- Temporary files automatically cleaned up

### decrypt

Decrypt all .age encrypted files in a directory using identity.txt.

```bash
# Decrypt all .age files in directory
./owuiback decrypt --path ./backups

# Decrypt with force overwrite
./owuiback decrypt --path ./backups --force

# Output example:
# Decrypting 3 file(s) from ./backups
#
# ✓ backup-20240101-120000.zip.age → backup-20240101-120000.zip
# ⊘ data.zip.age → skipped (data.zip already exists, use --force)
# ✓ archive.zip.age → archive.zip
#
# ──────────────────────────────────────────────────
# Summary: 2 decrypted, 1 skipped, 0 failed
```

**Flags:**
- `--path` - Directory containing identity.txt and .age files to decrypt (required)
- `--force` - Overwrite existing decrypted files (optional)

**Features:**
- Automatically finds all .age files in directory
- Uses identity.txt from the specified path
- Validates files are actually encrypted before decrypting
- Skips files where decrypted version already exists (unless --force)
- Removes .age extension from output files
- Provides detailed progress and summary statistics
- Continues on error, reporting failures at the end

**Use Cases:**
- Bulk decryption of backup archives
- Preparing encrypted files for manual inspection
- Decrypting files for use with other tools

### backup

Create an encrypted backup of Open WebUI data.

```bash
# Full backup
./owuiback backup --out ./backups/full.zip

# Selective backup
./owuiback backup --out ./backups/kb.zip --knowledge
./owuiback backup --out ./backups/pt.zip --prompts --tools

# Team backup (multiple recipients)
./owuiback backup --out ./backups/team.zip \
    --encrypt-recipient age1alice... \
    --encrypt-recipient age1bob...
```

**Flags:**
- `--out`, `-o` - Output file path (required)
- `--encrypt-recipient` - Age public key (required, repeatable)
- `--prompts`, `--tools`, `--knowledge`, `--models`, `--files`, `--chats` - Selective types

### restore

Restore data from an encrypted backup.

```bash
# Full restore
./owuiback restore --file ./backups/full.zip.age

# Selective restore
./owuiback restore --file ./backups/full.zip.age --knowledge --models

# With overwrite
./owuiback restore --file ./backups/full.zip.age --overwrite
```

**Flags:**
- `--file`, `-f` - Input file path (required)
- `--decrypt-identity` - Path to age identity file (required, repeatable)
- `--overwrite` - Replace existing data (default: skip existing)
- `--prompts`, `--tools`, `--knowledge`, `--models`, `--files`, `--chats` - Selective types

### purge

Safely delete data with dry-run and confirmation.

```bash
# Dry-run (shows what would be deleted)
./owuiback purge
./owuiback purge --chats --files

# Force deletion (requires typing "yes")
./owuiback purge --force
./owuiback purge --chats --force
```

**Flags:**
- `--force`, `-f` - Actually perform deletion (required for deletion)
- `--chats`, `--files`, `--models`, `--knowledge`, `--prompts`, `--tools` - Selective types

### serve

Start the web interface for backup/restore operations.

```bash
# Start web server (default: http://localhost:8080)
./owuiback serve

# Custom port
./owuiback serve --port 3000
```

## Configuration

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `OPEN_WEBUI_URL` | Open WebUI instance URL | ✅ |
| `OPEN_WEBUI_API_KEY` | API key for authentication | ✅ |
| `OWUI_ENCRYPTED_RECIPIENT` | Age public key for backup | ✅ (or use flag) |
| `OWUI_DECRYPT_IDENTITY` | Path to age identity file | ✅ (or use flag) |

### Example .env

```bash
export OPEN_WEBUI_URL="https://openwebui.example.com"
export OPEN_WEBUI_API_KEY="sk-xxxxxxxxxxxxxxxxxxxxxx"
export OWUI_ENCRYPTED_RECIPIENT="age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p"
export OWUI_DECRYPT_IDENTITY="$HOME/.age/identity.txt"
```

## Encryption

All backups use [age](https://age-encryption.org/) encryption with X25519 public key cryptography.

### Team Setup

```bash
# Alice generates keys
age-keygen -o alice_identity.txt
ALICE_KEY=$(age-keygen -y alice_identity.txt)

# Bob generates keys
age-keygen -o bob_identity.txt
BOB_KEY=$(age-keygen -y bob_identity.txt)

# Create backup for both
./owuiback backup --out team.zip \
    --encrypt-recipient $ALICE_KEY \
    --encrypt-recipient $BOB_KEY

# Either can restore
./owuiback restore --file team.zip.age --decrypt-identity alice_identity.txt
```

## Docker

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

# Web interface
docker run -p 8080:8080 \
  -e OPEN_WEBUI_URL="https://instance.com" \
  -e OPEN_WEBUI_API_KEY="sk-xxx" \
  owuiback:latest serve
```

## Development

```bash
# Install dependencies
task bootstrap

# Build
task build

# Run locally
task run -- backup --out test.zip

# Run tests
go test -v ./...
```

## Releases

The project uses GitHub Actions to automatically build cross-platform binaries when a tag is pushed.

### Creating a Release

1. **Tag the release:**
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **Workflow builds:**
   - Linux (x64, ARM64)
   - macOS (x64, ARM64)
   - Windows (x64, ARM64)

3. **GitHub Release created with:**
   - Compressed binaries (`.tar.gz` for Unix, `.zip` for Windows)
   - SHA256 checksums
   - Auto-generated release notes
   - Version info embedded in binaries

### Testing a Release

```bash
# Create and push a test tag
git tag v0.0.1-test
git push origin v0.0.1-test

# Watch the workflow at:
# https://github.com/vosiander/open-webui-backup/actions

# Download and verify
wget https://github.com/vosiander/open-webui-backup/releases/download/v0.0.1-test/owuiback-v0.0.1-test-linux-amd64.tar.gz
tar -xzf owuiback-v0.0.1-test-linux-amd64.tar.gz
./owuiback-linux-amd64 --version

# Verify checksum
wget https://github.com/vosiander/open-webui-backup/releases/download/v0.0.1-test/checksums.txt
sha256sum -c checksums.txt --ignore-missing
```

### Version Information

Binaries include embedded version information accessible via `--version` flag:
- Git tag version
- Git commit hash
- Build date
- Go version used

## Architecture

- **Go Backend** - CLI and web server
- **Vue 3 Frontend** - Modern reactive UI
- **Age Encryption** - Public key cryptography
- **WebSocket** - Real-time operation progress
- **Plugin System** - Extensible command architecture

## Migration from v1.0

v2.0 introduces breaking changes:

```bash
# OLD (v1.0)
./owuiback backup-all --dir ./backups
./owuiback restore-all --dir ./backups/backup.zip

# NEW (v2.0)
./owuiback backup --out ./backups/backup.zip
./owuiback restore --file ./backups/backup.zip.age
```

**Key Changes:**
- 12 commands → 3 commands (backup, restore, purge)
- Optional encryption → Mandatory encryption
- Passphrase mode removed (public key only)
- `--dir` flag → `--out`/`--file` flags

## License

See LICENSE file.

## Contributing

Contributions welcome! Fork, create feature branch, add tests, submit PR.
