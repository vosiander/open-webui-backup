package plugins

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/encryption"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
	"github.com/vosiander/open-webui-backup/pkg/restore"
)

// RestoreFilePlugin implements the Plugin interface for restoring files
type RestoreFilePlugin struct {
	dir             string
	overwrite       bool
	decrypt         bool
	decryptIdentity []string
}

// NewRestoreFilePlugin creates a new RestoreFilePlugin
func NewRestoreFilePlugin() *RestoreFilePlugin {
	return &RestoreFilePlugin{}
}

// Name returns the command name
func (p *RestoreFilePlugin) Name() string {
	return "restore-dir"
}

// Description returns the command description
func (p *RestoreFilePlugin) Description() string {
	return "Restore a dir to Open WebUI"
}

// SetupFlags adds custom flags to the command
func (p *RestoreFilePlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&p.dir, "dir", "d", "", "Path to the backup ZIP dir (required)")
	cmd.Flags().BoolVar(&p.overwrite, "overwrite", false, "Overwrite existing dir if it exists")
	cmd.Flags().BoolVar(&p.decrypt, "decrypt", false, "Decrypt the backup dir using a passphrase")
	cmd.Flags().StringSliceVar(&p.decryptIdentity, "decrypt-identity", nil, "Path to age identity dir(s) for decryption (for public key encryption)")
	cmd.MarkFlagRequired("dir")
}

// Execute runs the restore dir command
func (p *RestoreFilePlugin) Execute(cfg *config.Config) error {
	logrus.Info("Restoring dir to Open WebUI...")
	logrus.WithField("url", cfg.OpenWebUIURL).Info("Connecting to Open WebUI")

	// Validate configuration
	if cfg.OpenWebUIAPIKey == "" {
		logrus.Fatalf("OPEN_WEBUI_API_KEY environment variable is required")
	}

	if p.dir == "" {
		logrus.Fatalf("directory is required (use --dir flag)")
	}

	// Handle decryption if needed
	restorePath := p.dir
	var tempFile string
	if encryption.IsEncrypted(p.dir) || p.decrypt || len(p.decryptIdentity) > 0 {
		var isTempFile bool
		var err error
		restorePath, isTempFile, err = encryption.DecryptBackupFile(p.dir, p.decrypt, p.decryptIdentity)
		if err != nil {
			logrus.Fatalf("Failed to decrypt backup: %v", err)
		}
		if isTempFile {
			tempFile = restorePath
			defer os.Remove(tempFile)
		}
	}

	// Create API client
	client := openwebui.NewClient(cfg.OpenWebUIURL, cfg.OpenWebUIAPIKey)

	// Perform restore
	if err := restore.RestoreFile(client, restorePath, p.overwrite); err != nil {
		logrus.Fatalf("restore failed: %w", err)
	}

	logrus.Info("Restore completed successfully!")
	return nil
}
