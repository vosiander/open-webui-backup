package plugins

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/backup"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
)

type BackupAllPlugin struct {
	dir string
}

func NewBackupAllPlugin() *BackupAllPlugin {
	return &BackupAllPlugin{}
}

// Name returns the name of the plugin (used as command name)
func (p *BackupAllPlugin) Name() string {
	return "backup-all"
}

// Description returns a short description of the plugin
func (p *BackupAllPlugin) Description() string {
	return "Backup all knowledge bases and models from Open WebUI"
}

func (p *BackupAllPlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&p.dir, "dir", "d", "", "Directory for backup/restore files (required)")
	cmd.MarkFlagRequired("dir")
}

// Execute runs the plugin with the given configuration
func (p *BackupAllPlugin) Execute(cfg *config.Config) error {
	logrus.Info("Starting full backup...")

	if cfg.OpenWebUIAPIKey == "" {
		logrus.Fatalf("OPEN_WEBUI_API_KEY environment variable is required")
	}

	if p.dir == "" {
		logrus.Fatalf("directory is required (use --dir flag)")
	}

	client := openwebui.NewClient(cfg.OpenWebUIURL, cfg.OpenWebUIAPIKey)

	if err := backup.BackupAll(client, p.dir); err != nil {
		logrus.Fatalf("Failed to backup: %v", err)
	}

	logrus.Info("Full backup completed successfully")
	return nil
}
