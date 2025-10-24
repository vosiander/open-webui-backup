package plugins

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/backup"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
)

// BackupFilePlugin implements the Plugin interface for backing up files
type BackupFilePlugin struct {
	dir string
}

// NewBackupFilePlugin creates a new BackupFilePlugin
func NewBackupFilePlugin() *BackupFilePlugin {
	return &BackupFilePlugin{}
}

// Name returns the command name
func (p *BackupFilePlugin) Name() string {
	return "backup-file"
}

// Description returns the command description
func (p *BackupFilePlugin) Description() string {
	return "Backup files from Open WebUI"
}

// SetupFlags adds custom flags to the command
func (p *BackupFilePlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&p.dir, "dir", "d", "", "Directory for backup/restore files (required)")
	cmd.MarkFlagRequired("dir")
}

// Execute runs the backup file command
func (p *BackupFilePlugin) Execute(cfg *config.Config) error {
	logrus.Info("Backing up files from Open WebUI...")
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
	if err := backup.BackupFiles(client, p.dir); err != nil {
		logrus.Fatalf("backup failed: %w", err)
	}

	logrus.Info("Backup completed successfully!")
	return nil
}
