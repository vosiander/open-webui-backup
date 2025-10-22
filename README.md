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
OPEN_WEBUI_URL="https://myinstance.com" OPEN_WEBUI_API_KEY="ey.xxx" ./owuiback backup-knowledge

# Restore knowledge base
OPEN_WEBUI_URL="https://myinstance.com" OPEN_WEBUI_API_KEY="ey.xxx" ./owuiback restore-knowledge
```

### Examples

Backup knowledge:
```bash
export OPEN_WEBUI_URL="https://myinstance.com"
export OPEN_WEBUI_API_KEY="ey.xxx"
./owuiback backup-knowledge
```

Restore knowledge:
```bash
export OPEN_WEBUI_URL="https://myinstance.com"
export OPEN_WEBUI_API_KEY="ey.xxx"
./owuiback restore-knowledge
```

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
