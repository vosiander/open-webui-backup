package plugins

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/backup"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
)

// BackupPromptPlugin implements the Plugin interface for backing up prompts
type BackupPromptPlugin struct {
	dir string
}

// NewBackupPromptPlugin creates a new BackupPromptPlugin
func NewBackupPromptPlugin() *BackupPromptPlugin {
	return &BackupPromptPlugin{}
}

// Name returns the command name
func (p *BackupPromptPlugin) Name() string {
	return "backup-prompt"
}

// Description returns the command description
func (p *BackupPromptPlugin) Description() string {
	return "Backup prompts from Open WebUI"
}

// SetupFlags adds custom flags to the command
func (p *BackupPromptPlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&p.dir, "dir", "d", "", "Directory for backup/restore files (required)")
	cmd.MarkFlagRequired("dir")
}

// Execute runs the backup prompt command
func (p *BackupPromptPlugin) Execute(cfg *config.Config) error {
	logrus.Info("Backing up prompts from Open WebUI...")
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
	if err := backup.BackupPrompts(client, p.dir); err != nil {
		logrus.Fatalf("backup failed: %w", err)
	}

	logrus.Info("Backup completed successfully!")
	return nil
}
