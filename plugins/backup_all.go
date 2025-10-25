package plugins

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/backup"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/encryption"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
)

type BackupAllPlugin struct {
	dir              string
	encrypt          bool
	encryptRecipient []string
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
	cmd.Flags().BoolVar(&p.encrypt, "encrypt", false, "Encrypt backup with passphrase (interactive)")
	cmd.Flags().StringSliceVar(&p.encryptRecipient, "encrypt-recipient", nil, "Encrypt backup with age public key(s)")
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

	// Handle encryption if requested
	if p.encrypt || len(p.encryptRecipient) > 0 {
		// Find the most recent backup dir
		files, err := filepath.Glob(filepath.Join(p.dir, "*_owui_full_backup.zip"))
		if err != nil || len(files) == 0 {
			logrus.Fatalf("Failed to find backup dir for encryption")
		}

		// Get the most recent dir
		backupFile := files[len(files)-1]
		encryptedFile := backupFile + ".age"

		// Prepare encryption options
		var passphrase string
		var recipients []string

		if len(p.encryptRecipient) > 0 {
			// Public key mode
			recipients = p.encryptRecipient
			logrus.Info("Encrypting backup with public key(s)...")
		} else {
			// Passphrase mode - check env first, then prompt
			passphrase = encryption.GetPassphraseFromEnv()
			if passphrase == "" {
				var err error
				passphrase, err = encryption.ReadAndConfirmPassphrase()
				if err != nil {
					logrus.Fatalf("Failed to read passphrase: %v", err)
				}
			}
			logrus.Info("Encrypting backup with passphrase...")
		}

		// Encrypt the backup
		opts := &encryption.EncryptOptions{
			Passphrase: passphrase,
			Recipients: recipients,
		}

		if err := encryption.EncryptFile(backupFile, encryptedFile, opts); err != nil {
			logrus.Fatalf("Failed to encrypt backup: %v", err)
		}

		// Remove unencrypted backup
		if err := os.Remove(backupFile); err != nil {
			logrus.Warnf("Failed to remove unencrypted backup: %v", err)
		}

		logrus.Infof("Backup encrypted: %s", filepath.Base(encryptedFile))
	}

	logrus.Info("Full backup completed successfully")
	return nil
}
