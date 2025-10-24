package plugins

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/backup"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
)

// BackupToolPlugin implements the Plugin interface for backing up tools
type BackupToolPlugin struct {
	dir string
}

// NewBackupToolPlugin creates a new BackupToolPlugin
func NewBackupToolPlugin() *BackupToolPlugin {
	return &BackupToolPlugin{}
}

// Name returns the command name
func (p *BackupToolPlugin) Name() string {
	return "backup-tool"
}

// Description returns the command description
func (p *BackupToolPlugin) Description() string {
	return "Backup tools from Open WebUI"
}

// SetupFlags adds custom flags to the command
func (p *BackupToolPlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&p.dir, "dir", "d", "", "Directory for backup/restore files (required)")
	cmd.MarkFlagRequired("dir")
}

// Execute runs the backup tool command
func (p *BackupToolPlugin) Execute(cfg *config.Config) error {
	logrus.Info("Backing up tools from Open WebUI...")
	logrus.WithField("url", cfg.OpenWebUIURL).Info("Connecting to Open WebUI")

	// Validate configuration
	if cfg.OpenWebUIAPIKey == "" {
		logrus.Fatalf("OPEN_WEBUI_API_KEY environment variable is required")
	}

	if p.dir == "" {
		logrus.Fatalf("directory is required (use --dir flag)")
	}

	// Create API client
	client := openwebui.NewClient(cfg.OpenWebUIURL, cfg.OpenWebUIAPIKey)

	// Perform backup
	if err := backup.BackupTools(client, p.dir); err != nil {
		logrus.Fatalf("backup failed: %w", err)
	}

	logrus.Info("Backup completed successfully!")
	return nil
}
