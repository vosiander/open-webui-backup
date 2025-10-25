package plugins

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/backup"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/encryption"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
)

// BackupKnowledgePlugin implements the Plugin interface for backing up knowledge
type BackupKnowledgePlugin struct {
	dir              string
	encrypt          bool
	encryptRecipient []string
}

// NewBackupKnowledgePlugin creates a new BackupKnowledgePlugin
func NewBackupKnowledgePlugin() *BackupKnowledgePlugin {
	return &BackupKnowledgePlugin{}
}

// Name returns the command name
func (p *BackupKnowledgePlugin) Name() string {
	return "backup-knowledge"
}

// Description returns the command description
func (p *BackupKnowledgePlugin) Description() string {
	return "Backup knowledge base from Open WebUI"
}

// SetupFlags adds custom flags to the command
func (p *BackupKnowledgePlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&p.dir, "dir", "d", "", "Directory for backup/restore files (required)")
	cmd.MarkFlagRequired("dir")
	cmd.Flags().BoolVar(&p.encrypt, "encrypt", false, "Encrypt backup with passphrase (interactive)")
	cmd.Flags().StringSliceVar(&p.encryptRecipient, "encrypt-recipient", nil, "Encrypt backup with age public key(s)")
}

// Execute runs the backup knowledge command
func (p *BackupKnowledgePlugin) Execute(cfg *config.Config) error {
	logrus.Info("Backing up knowledge from Open WebUI...")
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
	if err := backup.BackupKnowledge(client, p.dir); err != nil {
		logrus.Fatalf("backup failed: %w", err)
	}

	// Handle encryption if requested
	if p.encrypt || len(p.encryptRecipient) > 0 {
		// Find the most recent knowledge base backup
		backupFile, err := encryption.FindLatestBackup(p.dir, "*_knowledge_base_*.zip")
		if err != nil || backupFile == "" {
			logrus.Fatalf("Failed to find backup dir for encryption")
		}

		_, err = encryption.EncryptBackupFile(backupFile, p.encrypt, p.encryptRecipient)
		if err != nil {
			logrus.Fatalf("Failed to encrypt backup: %v", err)
		}
	}

	logrus.Info("Backup completed successfully!")
	return nil
}
