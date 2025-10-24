package plugins

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
	"github.com/vosiander/open-webui-backup/pkg/restore"
)

// RestoreToolPlugin implements the Plugin interface for restoring tools
type RestoreToolPlugin struct {
	dir       string
	overwrite bool
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
	cmd.Flags().StringVarP(&p.dir, "dir", "d", "", "Path to tool backup ZIP file (required)")
	cmd.Flags().BoolVar(&p.overwrite, "overwrite", false, "Overwrite existing tool if it already exists")
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

	// Create API client
	client := openwebui.NewClient(cfg.OpenWebUIURL, cfg.OpenWebUIAPIKey)

	// Perform restore
	if err := restore.RestoreTool(client, p.dir, p.overwrite); err != nil {
		logrus.Fatalf("restore failed: %w", err)
	}

	logrus.Info("Restore completed successfully!")
	return nil
}
