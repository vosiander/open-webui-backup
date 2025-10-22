# open-webui-backup

A small tool to backup various important information from an Open WebUI application.

## Features

- Plugin-based architecture for extensibility
- Environment variable configuration
- Support for backup and restore operations

## Installation

### Build from source

```bash
go build -o owuiback ./cmd/owuiback
```

## Configuration

Configure the tool using environment variables:

- `OPEN_WEBUI_URL`: The URL of your Open WebUI instance (default: "https://example.com")
- `OPEN_WEBUI_API_KEY`: Your Open WebUI API key (required)

## Usage

### Available Commands

```bash
# Show help
./owuiback --help

# Backup knowledge base
./owuiback backup-knowledge --help

# Restore knowledge base (not yet implemented)
./owuiback restore-knowledge
```

### Backup Knowledge Base

The `backup-knowledge` command downloads all knowledge bases from your Open WebUI instance and saves them as ZIP files.

**Required flags:**
- `--output` or `-o`: Output directory for backup files

**Example:**
```bash
export OPEN_WEBUI_URL="https://myinstance.com"
export OPEN_WEBUI_API_KEY="ey.xxx"
./owuiback backup-knowledge --output ./backups
```

**Or inline:**
```bash
OPEN_WEBUI_URL="https://myinstance.com" OPEN_WEBUI_API_KEY="ey.xxx" \
  ./owuiback backup-knowledge --output ./backups
```

#### Backup Format

Each knowledge base is backed up to a separate ZIP file with the following format:
- **Filename:** `YYYYMMDD_HHMMSS_knowledge_base_<sanitized_name>.zip`
- **Contents:**
  - `knowledge_base.json` - Complete metadata including ID, name, description, file IDs, and access control
  - `documents/` - Folder containing all document files with their original names

**Example ZIP structure:**
```
20251022_203457_knowledge_base_myknowledge.zip
├── knowledge_base.json
└── documents/
    ├── document1.md
    ├── document2.pdf
    └── document3.txt
```

**Notes:**
- The tool will fail if a backup file already exists (no overwrite)
- Each backup run creates new timestamped files, allowing multiple backups of the same knowledge base
- Document filenames are preserved from the original upload

### Restore Knowledge Base

The `restore-knowledge` command restores a knowledge base from a backup ZIP file created by the `backup-knowledge` command. The restore operation is **idempotent** - you can run it multiple times safely.

**Required flags:**
- `--input` or `-i`: Path to backup ZIP file

**Optional flags:**
- `--overwrite`: Overwrite existing files in the knowledge base (default: false)

**Example:**
```bash
export OPEN_WEBUI_URL="https://myinstance.com"
export OPEN_WEBUI_API_KEY="ey.xxx"
./owuiback restore-knowledge --input ./backups/20251022_203457_knowledge_base_myknowledge.zip
```

**Or inline:**
```bash
OPEN_WEBUI_URL="https://myinstance.com" OPEN_WEBUI_API_KEY="ey.xxx" \
  ./owuiback restore-knowledge --input ./backups/20251022_203457_knowledge_base_myknowledge.zip
```

**With overwrite flag:**
```bash
./owuiback restore-knowledge --input ./backups/20251022_203457_knowledge_base_myknowledge.zip --overwrite
```

#### Idempotent Restore Behavior

The restore operation is designed to be idempotent and safe to run multiple times:

**If knowledge base doesn't exist:**
1. Creates the knowledge base with name, description, and access control
2. Uploads all document files from the backup
3. Associates the uploaded files with the knowledge base

**If knowledge base already exists:**
1. Updates metadata (description, access control) if different from backup
2. Compares backup files with existing files (by filename)
3. For each file in backup:
   - **File doesn't exist**: Upload and add it
   - **File exists + --overwrite flag**: Remove old file, upload new one
   - **File exists + no --overwrite flag**: Skip it (keeps existing file)
4. Files in knowledge base but NOT in backup are left untouched

**Important Notes:**
- The restore is **additive** - existing files not in the backup are preserved
- File matching is done by filename only
- File IDs will differ between backups and live instances
- Metadata (description, access control) is always updated if different
- Use `--overwrite` to refresh existing files with backup versions

**Example Output:**
```
INFO[0000] Restoring knowledge to Open WebUI...
INFO[0000] Connecting to Open WebUI                     url="https://myinstance.com"
INFO[0000] Starting restore from: ./backups/20251022_203457_knowledge_base_myknowledge.zip
INFO[0001] Extracted knowledge base: My Knowledge
INFO[0001] Found 3 files to restore
INFO[0001] Creating knowledge base: My Knowledge
INFO[0002] Knowledge base created with ID: abc123
INFO[0002] Uploading file: document1.md (2048 bytes)
INFO[0003] File uploaded successfully: document1.md (ID: file123)
INFO[0003] Uploading file: document2.pdf (51200 bytes)
INFO[0004] File uploaded successfully: document2.pdf (ID: file456)
INFO[0004] Uploading file: document3.txt (1024 bytes)
INFO[0005] File uploaded successfully: document3.txt (ID: file789)
INFO[0005] Linking 3 files to knowledge base
INFO[0005] Linking file 1/3 (ID: file123)
INFO[0006] Linking file 2/3 (ID: file456)
INFO[0007] Linking file 3/3 (ID: file789)
INFO[0007] All files linked successfully
INFO[0007] Successfully restored knowledge base: My Knowledge
INFO[0007] Restore completed successfully!
```

**Example Output (existing KB, no --overwrite):**
```
INFO[0000] Restoring knowledge to Open WebUI...
INFO[0000] Overwrite mode disabled - existing files will be skipped
INFO[0000] Knowledge base 'My Knowledge' already exists (ID: abc123)
INFO[0000] Found 2 existing files in knowledge base
INFO[0001] Uploading new file: document3.md (1024 bytes)
INFO[0002] New file added: document3.md (ID: file999)
INFO[0002] File already exists, skipping: document1.md
INFO[0002] File already exists, skipping: document2.pdf
INFO[0002] File sync completed: 1 new, 0 overwritten, 2 skipped
INFO[0002] Successfully restored knowledge base: My Knowledge
```

**Example Output (existing KB, with --overwrite):**
```
INFO[0000] Restoring knowledge to Open WebUI...
INFO[0000] Overwrite mode enabled - existing files will be replaced
INFO[0000] Knowledge base 'My Knowledge' already exists (ID: abc123)
INFO[0000] Found 2 existing files in knowledge base
INFO[0001] Overwriting existing file: document1.md (2048 bytes)
INFO[0002] File overwritten: document1.md (new ID: file888)
INFO[0002] Overwriting existing file: document2.pdf (51200 bytes)
INFO[0003] File overwritten: document2.pdf (new ID: file777)
INFO[0003] File sync completed: 0 new, 2 overwritten, 0 skipped
INFO[0003] Successfully restored knowledge base: My Knowledge
```

**Common Errors:**
- `ZIP file not found` - Check the file path
- `Invalid ZIP file` - The backup file may be corrupted
- `knowledge_base.json not found in ZIP file` - Not a valid backup file
- `Failed to upload file` - Network or API error
- `Failed to create knowledge base` - API error or permission issue
- `Failed to link file` - File processing may not have completed

## Plugin Architecture

This tool uses a plugin-based architecture that allows for easy extension. Each command is implemented as a plugin that adheres to the `Plugin` interface defined in `pkg/plugin/plugin.go`.

### Adding New Plugins

1. Create a new file in the `plugins/` directory
2. Implement the `Plugin` interface:
   - `Name() string` - Command name
   - `Description() string` - Command description
   - `Execute(config *config.Config) error` - Command logic
3. Register the plugin in `cmd/owuiback/main.go`

Example:
```go
type MyPlugin struct{}

func (p *MyPlugin) Name() string {
    return "my-command"
}

func (p *MyPlugin) Description() string {
    return "My command description"
}

func (p *MyPlugin) Execute(cfg *config.Config) error {
    // Command implementation
    return nil
}
```

## License

See LICENSE file for details.
