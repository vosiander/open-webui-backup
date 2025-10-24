# open-webui-backup

A comprehensive CLI tool to backup and restore data from Open WebUI applications, including knowledge bases, models, tools, prompts, and files.

## Features

- **Plugin-based architecture** for extensibility
- **Unified backup format** - Single ZIP file containing all data types
- **Individual backups** - Backup specific items (knowledge bases, models, tools, prompts, files)
- **Metadata tracking** - Every backup includes `owui.json` with version and source information
- **Backward compatibility** - Restore from legacy separate backup files
- **Idempotent restore** - Safe to run multiple times
- **Overwrite control** - Choose whether to replace existing items

## Installation

### Build from source

```bash
go build -o owuiback ./cmd/owuiback
```

### Using Task

```bash
task build
```

## Configuration

Configure the tool using environment variables:

- `OPEN_WEBUI_URL`: The URL of your Open WebUI instance (default: "https://example.com")
- `OPEN_WEBUI_API_KEY`: Your Open WebUI API key (required)

## Available Commands

### Backup Commands

- `backup-all` - Backup all data types to a single unified ZIP file
- `backup-knowledge` - Backup knowledge bases
- `backup-model` - Backup models
- `backup-tool` - Backup tools
- `backup-prompt` - Backup prompts
- `backup-file` - Backup files

### Restore Commands

- `restore-all` - Restore from unified or legacy backup
- `restore-knowledge` - Restore knowledge bases
- `restore-model` - Restore models
- `restore-tool` - Restore tools
- `restore-prompt` - Restore prompts
- `restore-file` - Restore files

## Usage

### Unified Backup (Recommended)

The `backup-all` command creates a **single ZIP file** containing all your Open WebUI data with complete metadata tracking.

**Required flags:**
- `--dir` or `-d`: Directory for backup file

**Example:**
```bash
export OPEN_WEBUI_URL="https://myinstance.com"
export OPEN_WEBUI_API_KEY="sk-xxx"
./owuiback backup-all --dir ./backups
```

**Output:**
```
20251024_220315_owui_full_backup.zip
```

**Unified ZIP Structure:**
```
20251024_220315_owui_full_backup.zip
├── owui.json                      # Backup metadata
├── knowledge-bases/
│   ├── kb-id-1/
│   │   ├── knowledge_base.json
│   │   └── documents/
│   │       └── file.pdf
│   └── kb-id-2/
│       └── ...
├── models/
│   ├── model-id-1/
│   │   ├── model.json
│   │   ├── model-files/
│   │   └── knowledge-bases/
│   └── ...
├── tools/
│   ├── tool-id-1/
│   │   └── tool.json
│   └── ...
├── prompts/
│   ├── command-1/
│   │   └── prompt.json
│   └── ...
└── files/
    ├── file-id-1/
    │   ├── file.json
    │   └── content/
    │       └── original-filename.ext
    └── ...
```

**owui.json Metadata:**
```json
{
  "open_webui_url": "https://myinstance.com",
  "open_webui_version": "0.3.32",
  "backup_tool_version": "0.3.0",
  "backup_timestamp": "2025-10-24T22:03:15Z",
  "backup_type": "full",
  "item_count": 42,
  "unified_backup": true,
  "contained_types": ["knowledge", "model", "tool", "prompt", "file"]
}
```

### Unified Restore

The `restore-all` command automatically detects and restores from either unified or legacy backup formats.

**Required flags:**
- `--dir` or `-d`: Path to unified ZIP file or directory with legacy backups

**Optional flags:**
- `--overwrite`: Overwrite existing items (default: false)

**Example (Unified):**
```bash
./owuiback restore-all --dir ./backups/20251024_220315_owui_full_backup.zip --overwrite
```

**Example (Legacy Directory):**
```bash
./owuiback restore-all --dir ./backups/20251024_old_backup/
```

**Behavior:**
- **Unified format**: Automatically detected by presence of `owui.json` with `unified_backup: true`
- **Legacy format**: Automatically detected when directory contains separate ZIP files
- **Idempotent**: Safe to run multiple times
- **Selective restore**: Only restores data types present in backup

### Individual Backups

Each data type can be backed up individually. All individual backups include `owui.json` metadata.

#### Backup Knowledge Bases

```bash
./owuiback backup-knowledge --dir ./backups
```

**Output:** `20251024_220315_knowledge_base_name.zip`

#### Backup Models

```bash
./owuiback backup-model --dir ./backups
```

**Output:** `20251024_220315_model_name.zip`

#### Backup Tools

```bash
./owuiback backup-tool --dir ./backups
```

**Output:** `20251024_220315_tools.zip`

#### Backup Prompts

```bash
./owuiback backup-prompt --dir ./backups
```

**Output:** `20251024_220315_prompts.zip`

#### Backup Files

```bash
./owuiback backup-file --dir ./backups
```

**Output:** `20251024_220315_files.zip`

### Individual Restores

Each backup type can be restored individually with the same `--overwrite` flag behavior.

#### Restore Knowledge Base

```bash
./owuiback restore-knowledge --dir ./backups/20251024_220315_knowledge_base_name.zip --overwrite
```

#### Restore Model

```bash
./owuiback restore-model --dir ./backups/20251024_220315_model_name.zip --overwrite
```

#### Restore Tool

```bash
./owuiback restore-tool --dir ./backups/20251024_220315_tools.zip --overwrite
```

#### Restore Prompt

```bash
./owuiback restore-prompt --dir ./backups/20251024_220315_prompts.zip --overwrite
```

#### Restore File

```bash
./owuiback restore-file --dir ./backups/20251024_220315_files.zip --overwrite
```

## Backup Metadata

Every backup ZIP file (both unified and individual) includes an `owui.json` metadata file at the root. This enables:

- **Version compatibility checking** (future feature)
- **Source tracking** - Know which Open WebUI instance created the backup
- **Timestamp tracking** - When the backup was created
- **Content identification** - What data types are included
- **Format detection** - Unified vs individual backup

**Individual Backup Metadata Example:**
```json
{
  "open_webui_url": "https://myinstance.com",
  "open_webui_version": "0.3.32",
  "backup_tool_version": "0.3.0",
  "backup_timestamp": "2025-10-24T22:03:15Z",
  "backup_type": "tool",
  "item_count": 5,
  "unified_backup": false
}
```

## Restore Behavior

### Overwrite Flag

The `--overwrite` flag controls behavior when an item already exists:

**Without `--overwrite` (default):**
- Existing items are **skipped**
- New items are added
- Safe for incremental backups
- Preserves existing data

**With `--overwrite`:**
- Existing items are **replaced** with backup version
- New items are added
- Use when you want to revert to backup state
- Overwrites local modifications

### Idempotent Restore

All restore operations are idempotent - you can safely run them multiple times:

1. First run: Creates all items from backup
2. Subsequent runs: 
   - Updates metadata if changed
   - Adds missing items
   - Skips or overwrites existing items based on `--overwrite` flag

**Example workflow:**
```bash
# Initial restore
./owuiback restore-all --dir backup.zip

# Fails partway through due to network issue
# Simply run again - already-restored items will be skipped
./owuiback restore-all --dir backup.zip

# Want to reset everything to backup state
./owuiback restore-all --dir backup.zip --overwrite
```

## Data Types

### Knowledge Bases

- Complete knowledge base metadata (name, description, access control)
- All associated document files with original filenames
- File relationships and IDs

### Models

- Model metadata and configuration
- Associated knowledge bases
- Model files (configurations, weights, etc.)

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

## Testing

### Integration Tests

Comprehensive integration tests are available in `test/integration/`:

```bash
# Run all integration tests
export OWUI_URL="http://localhost:8080"
export OWUI_API_KEY="your-api-key"
go test -v ./test/integration

# Run specific test
go test -v ./test/integration -run TestUnifiedBackupAll

# Run benchmarks
go test -v -bench=. ./test/integration
```

See `test/integration/README.md` for detailed testing documentation.

## Plugin Architecture

This tool uses a plugin-based architecture that allows for easy extension. Each command is implemented as a plugin that adheres to the `Plugin` interface defined in `pkg/plugin/plugin.go`.

### Plugin Interface

```go
type Plugin interface {
    Name() string                          // Command name
    Description() string                   // Command description
    SetupFlags(cmd *cobra.Command)        // Add custom flags
    Execute(cfg *config.Config) error     // Execute command
}
```

### Adding New Plugins

1. Create a new file in the `plugins/` directory
2. Implement the `Plugin` interface
3. Register the plugin in `cmd/owuiback/main.go`

Example:
```go
package plugins

type MyPlugin struct {
    dir string
}

func NewMyPlugin() *MyPlugin {
    return &MyPlugin{}
}

func (p *MyPlugin) Name() string {
    return "my-command"
}

func (p *MyPlugin) Description() string {
    return "My command description"
}

func (p *MyPlugin) SetupFlags(cmd *cobra.Command) {
    cmd.Flags().StringVarP(&p.dir, "dir", "d", "", "Directory (required)")
    cmd.MarkFlagRequired("dir")
}

func (p *MyPlugin) Execute(cfg *config.Config) error {
    // Command implementation
    client := openwebui.NewClient(cfg.OpenWebUIURL, cfg.OpenWebUIAPIKey)
    // ... implementation
    return nil
}
```

Then register in `main.go`:
```go
registry.Register(plugins.NewMyPlugin())
```

## Architecture

### Package Structure

- `cmd/owuiback/` - Main application entry point
- `pkg/openwebui/` - Open WebUI API client and data models
- `pkg/backup/` - Backup logic and ZIP creation
- `pkg/restore/` - Restore logic and ZIP extraction
- `pkg/config/` - Configuration management
- `pkg/plugin/` - Plugin system interface
- `plugins/` - Individual command plugins
- `test/integration/` - Integration tests

### Key Components

**API Client** (`pkg/openwebui/client.go`):
- HTTP client for Open WebUI API
- Authentication handling
- Request/response processing
- Error handling

**Backup Engine** (`pkg/backup/backup.go`):
- ZIP file creation
- Metadata generation
- File organization
- Individual and unified backup logic

**Restore Engine** (`pkg/restore/restore.go`):
- ZIP file extraction
- Format detection (unified vs legacy)
- Item creation/update
- Overwrite logic
- Idempotent operation handling

**Data Models** (`pkg/openwebui/models.go`):
- Go structs for all Open WebUI data types
- JSON serialization/deserialization
- Metadata structures

## Troubleshooting

### Common Issues

**"API key is required"**
- Set `OPEN_WEBUI_API_KEY` environment variable
- Verify the API key is valid

**"Failed to connect to Open WebUI"**
- Check `OPEN_WEBUI_URL` is correct
- Verify Open WebUI is running
- Check network connectivity

**"File already exists"**
- Backup files are not overwritten by default
- Use different output directory or remove old backups
- Restore operations use `--overwrite` flag

**"Knowledge base already exists" (without --overwrite)**
- This is expected behavior - existing items are skipped
- Use `--overwrite` flag to replace existing items

**"Failed to upload file"**
- Check file size limits in Open WebUI
- Verify available storage space
- Check network stability

### Debug Mode

Set log level for detailed output:
```bash
export LOG_LEVEL=debug
./owuiback backup-all --dir ./backups
```

## Version History

### v0.3.0 (Current)
- ✅ Added unified backup format (single ZIP for all data)
- ✅ Added backup metadata (`owui.json`) to all backups
- ✅ Added support for tools, prompts, and files
- ✅ Refactored backup-all to create unified ZIP
- ✅ Added format detection to restore-all (unified vs legacy)
- ✅ Added comprehensive integration tests
- ✅ Backward compatibility with legacy separate ZIPs

### v0.2.0
- ✅ Added model backup and restore
- ✅ Improved knowledge base handling
- ✅ Added plugin architecture

### v0.1.0
- ✅ Initial release
- ✅ Knowledge base backup and restore

## License

See LICENSE file for details.

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## Support

For issues and questions:
- Open an issue on GitHub
- Check existing issues for solutions
- Refer to integration test examples in `test/integration/`
