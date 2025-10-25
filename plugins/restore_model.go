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

type RestoreModelPlugin struct {
	dir             string
	overwrite       bool
	decrypt         bool
	decryptIdentity []string
}

func NewRestoreModelPlugin() *RestoreModelPlugin {
	return &RestoreModelPlugin{}
}

// Name returns the name of the plugin (used as command name)
func (p *RestoreModelPlugin) Name() string {
	return "restore-model"
}

// Description returns a short description of the plugin
func (p *RestoreModelPlugin) Description() string {
	return "Restore model to Open WebUI from a backup ZIP dir"
}

func (p *RestoreModelPlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&p.dir, "dir", "d", "", "Path to backup ZIP dir (required)")
	cmd.Flags().BoolVarP(&p.overwrite, "overwrite", "w", false, "Overwrite existing model and files")
	cmd.Flags().BoolVar(&p.decrypt, "decrypt", false, "Decrypt backup with passphrase (interactive)")
	cmd.Flags().StringSliceVar(&p.decryptIdentity, "decrypt-identity", nil, "Decrypt backup with age identity dir(s)")
	cmd.MarkFlagRequired("dir")
}

// Execute runs the plugin with the given configuration
func (p *RestoreModelPlugin) Execute(cfg *config.Config) error {
	logrus.Info("Restoring model to Open WebUI...")

	if cfg.OpenWebUIAPIKey == "" {
		logrus.Fatalf("OPEN_WEBUI_API_KEY environment variable is required")
	}

	if p.dir == "" {
		logrus.Fatalf("directory path is required (use --dir flag)")
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

	client := openwebui.NewClient(cfg.OpenWebUIURL, cfg.OpenWebUIAPIKey)

	if err := restore.RestoreModel(client, restorePath, p.overwrite); err != nil {
		logrus.Fatalf("Failed to restore model: %v", err)
	}

	logrus.Info("Model restore completed successfully")
	return nil
}
