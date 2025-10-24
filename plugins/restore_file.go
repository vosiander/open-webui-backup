package plugins

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
	"github.com/vosiander/open-webui-backup/pkg/restore"
)

// RestoreFilePlugin implements the Plugin interface for restoring files
type RestoreFilePlugin struct {
	dir       string
	overwrite bool
}

// NewRestoreFilePlugin creates a new RestoreFilePlugin
func NewRestoreFilePlugin() *RestoreFilePlugin {
	return &RestoreFilePlugin{}
}

// Name returns the command name
func (p *RestoreFilePlugin) Name() string {
	return "restore-file"
}

// Description returns the command description
func (p *RestoreFilePlugin) Description() string {
	return "Restore a file to Open WebUI"
}

// SetupFlags adds custom flags to the command
func (p *RestoreFilePlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&p.dir, "dir", "d", "", "Path to the backup ZIP file (required)")
	cmd.Flags().BoolVar(&p.overwrite, "overwrite", false, "Overwrite existing file if it exists")
	cmd.MarkFlagRequired("dir")
}

// Execute runs the restore file command
func (p *RestoreFilePlugin) Execute(cfg *config.Config) error {
	logrus.Info("Restoring file to Open WebUI...")
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

	// Perform restore
	if err := restore.RestoreFile(client, p.dir, p.overwrite); err != nil {
		logrus.Fatalf("restore failed: %w", err)
	}

	logrus.Info("Restore completed successfully!")
	return nil
}
