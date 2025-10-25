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

// RestoreKnowledgePlugin implements the Plugin interface for restoring knowledge
type RestoreKnowledgePlugin struct {
	dir             string
	overwrite       bool
	decrypt         bool
	decryptIdentity []string
}

// NewRestoreKnowledgePlugin creates a new RestoreKnowledgePlugin
func NewRestoreKnowledgePlugin() *RestoreKnowledgePlugin {
	return &RestoreKnowledgePlugin{}
}

// Name returns the command name
func (p *RestoreKnowledgePlugin) Name() string {
	return "restore-knowledge"
}

// Description returns the command description
func (p *RestoreKnowledgePlugin) Description() string {
	return "Restore knowledge base to Open WebUI"
}

// SetupFlags adds custom flags to the command
func (p *RestoreKnowledgePlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&p.dir, "dir", "d", "", "Path to backup ZIP dir (required)")
	cmd.Flags().BoolVar(&p.overwrite, "overwrite", false, "Overwrite existing files in the knowledge base")
	cmd.Flags().BoolVar(&p.decrypt, "decrypt", false, "Decrypt backup with passphrase (interactive)")
	cmd.Flags().StringSliceVar(&p.decryptIdentity, "decrypt-identity", nil, "Decrypt backup with age identity dir(s)")

	if err := cmd.MarkFlagRequired("dir"); err != nil {
		logrus.Fatalf("marking required flag failed: %v", err)
	}
}

// Execute runs the restore knowledge command
func (p *RestoreKnowledgePlugin) Execute(cfg *config.Config) error {
	logrus.Info("Restoring knowledge to Open WebUI...")
	logrus.WithField("url", cfg.OpenWebUIURL).Info("Connecting to Open WebUI")

	if p.overwrite {
		logrus.Info("Overwrite mode enabled - existing files will be replaced")
	} else {
		logrus.Info("Overwrite mode disabled - existing files will be skipped")
	}

	// Validate configuration
	if cfg.OpenWebUIAPIKey == "" {
		logrus.Fatal("OPEN_WEBUI_API_KEY environment variable is required")
	}

	if p.dir == "" {
		logrus.Fatal("directory path is required (use --dir flag)")
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

	// Perform restore with overwrite flag
	if err := restore.RestoreKnowledge(client, restorePath, p.overwrite); err != nil {
		logrus.Fatalf("restore failed: %v", err)
	}

	logrus.Info("Restore completed successfully!")
	return nil
}
