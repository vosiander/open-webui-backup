package plugins

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/backup"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/encryption"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
)

type BackupModelPlugin struct {
	dir              string
	encrypt          bool
	encryptRecipient []string
}

func NewBackupModelPlugin() *BackupModelPlugin {
	return &BackupModelPlugin{}
}

// Name returns the name of the plugin (used as command name)
func (p *BackupModelPlugin) Name() string {
	return "backup-model"
}

// Description returns a short description of the plugin
func (p *BackupModelPlugin) Description() string {
	return "Backup model from Open WebUI"
}

func (p *BackupModelPlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&p.dir, "dir", "d", "", "Directory for backup/restore files (required)")
	cmd.MarkFlagRequired("dir")
	cmd.Flags().BoolVar(&p.encrypt, "encrypt", false, "Encrypt backup with passphrase (interactive)")
	cmd.Flags().StringSliceVar(&p.encryptRecipient, "encrypt-recipient", nil, "Encrypt backup with age public key(s)")
}

// Execute runs the plugin with the given configuration
func (p *BackupModelPlugin) Execute(cfg *config.Config) error {
	logrus.Info("Backing up models from Open WebUI...")

	if cfg.OpenWebUIAPIKey == "" {
		logrus.Fatalf("OPEN_WEBUI_API_KEY environment variable is required")
	}

	if p.dir == "" {
		logrus.Fatalf("directory is required (use --dir flag)")
	}

	client := openwebui.NewClient(cfg.OpenWebUIURL, cfg.OpenWebUIAPIKey)

	if err := backup.BackupModels(client, p.dir); err != nil {
		logrus.Fatalf("Failed to backup models: %v", err)
	}

	// Handle encryption if requested
	if p.encrypt || len(p.encryptRecipient) > 0 {
		backupFile, err := encryption.FindLatestBackup(p.dir, "*_model_*.zip")
		if err != nil || backupFile == "" {
			logrus.Fatalf("Failed to find backup dir for encryption")
		}

		_, err = encryption.EncryptBackupFile(backupFile, p.encrypt, p.encryptRecipient)
		if err != nil {
			logrus.Fatalf("Failed to encrypt backup: %v", err)
		}
	}

	logrus.Info("Model backup completed successfully")
	return nil
}
