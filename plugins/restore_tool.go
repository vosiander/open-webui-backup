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

// RestoreToolPlugin implements the Plugin interface for restoring tools
type RestoreToolPlugin struct {
	dir             string
	overwrite       bool
	decrypt         bool
	decryptIdentity []string
}

// NewRestoreToolPlugin creates a new RestoreToolPlugin
func NewRestoreToolPlugin() *RestoreToolPlugin {
	return &RestoreToolPlugin{}
}

// Name returns the command name
func (p *RestoreToolPlugin) Name() string {
	return "restore-tool"
}

// Description returns the command description
func (p *RestoreToolPlugin) Description() string {
	return "Restore a tool to Open WebUI"
}

// SetupFlags adds custom flags to the command
func (p *RestoreToolPlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&p.dir, "dir", "d", "", "Path to tool backup ZIP dir (required)")
	cmd.Flags().BoolVar(&p.overwrite, "overwrite", false, "Overwrite existing tool if it already exists")
	cmd.Flags().BoolVar(&p.decrypt, "decrypt", false, "Decrypt backup with passphrase (interactive)")
	cmd.Flags().StringSliceVar(&p.decryptIdentity, "decrypt-identity", nil, "Decrypt backup with age identity dir(s)")
	cmd.MarkFlagRequired("dir")
}

// Execute runs the restore tool command
func (p *RestoreToolPlugin) Execute(cfg *config.Config) error {
	logrus.Info("Restoring tool to Open WebUI...")
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
	if err := restore.RestoreTool(client, restorePath, p.overwrite); err != nil {
		logrus.Fatalf("restore failed: %w", err)
	}

	logrus.Info("Restore completed successfully!")
	return nil
}
