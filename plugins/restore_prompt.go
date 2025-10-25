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

// RestorePromptPlugin implements the Plugin interface for restoring prompts
type RestorePromptPlugin struct {
	dir             string
	overwrite       bool
	decrypt         bool
	decryptIdentity []string
}

// NewRestorePromptPlugin creates a new RestorePromptPlugin
func NewRestorePromptPlugin() *RestorePromptPlugin {
	return &RestorePromptPlugin{}
}

// Name returns the command name
func (p *RestorePromptPlugin) Name() string {
	return "restore-prompt"
}

// Description returns the command description
func (p *RestorePromptPlugin) Description() string {
	return "Restore a prompt to Open WebUI"
}

// SetupFlags adds custom flags to the command
func (p *RestorePromptPlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&p.dir, "dir", "d", "", "Path to prompt backup ZIP dir (required)")
	cmd.Flags().BoolVar(&p.overwrite, "overwrite", false, "Overwrite existing prompt if it already exists")
	cmd.Flags().BoolVar(&p.decrypt, "decrypt", false, "Decrypt backup with passphrase (interactive)")
	cmd.Flags().StringSliceVar(&p.decryptIdentity, "decrypt-identity", nil, "Decrypt backup with age identity dir(s)")
	cmd.MarkFlagRequired("dir")
}

// Execute runs the restore prompt command
func (p *RestorePromptPlugin) Execute(cfg *config.Config) error {
	logrus.Info("Restoring prompt to Open WebUI...")
	logrus.WithField("url", cfg.OpenWebUIURL).Info("Connecting to Open WebUI")

	// Validate configuration
	if cfg.OpenWebUIAPIKey == "" {
		logrus.Fatalf("OPEN_WEBUI_API_KEY environment variable is required")
	}

	if p.dir == "" {
		logrus.Fatalf("directory is required (use --dir flag)")
	}

	restorePath := p.dir
	var tempFile string

	// Handle decryption if needed
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
	if err := restore.RestorePrompt(client, restorePath, p.overwrite); err != nil {
		logrus.Fatalf("restore failed: %w", err)
	}

	logrus.Info("Restore completed successfully!")
	return nil
}
