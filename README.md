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

### 1. Generate Age Keys (One-Time)

```bash
age-keygen -o ~/.age/identity.txt
age-keygen -y ~/.age/identity.txt > ~/.age/recipient.txt
chmod 600 ~/.age/identity.txt
```

### 2. Configure Environment

```bash
export OPEN_WEBUI_URL="https://your-instance.com"
export OPEN_WEBUI_API_KEY="sk-your-api-key"
export OWUI_ENCRYPTED_RECIPIENT=$(cat ~/.age/recipient.txt)
export OWUI_DECRYPT_IDENTITY="$HOME/.age/identity.txt"
```

### 3. Backup & Restore

```bash
# Create backup
./owuiback backup --out ./backups/full.zip

# Restore backup
./owuiback restore --file ./backups/full.zip.age

# Web interface
./owuiback serve
# Open http://localhost:8080
```

## Commands

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

