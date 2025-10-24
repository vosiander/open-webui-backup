package plugins

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
	"github.com/vosiander/open-webui-backup/pkg/restore"
)

// RestorePromptPlugin implements the Plugin interface for restoring prompts
type RestorePromptPlugin struct {
	dir       string
	overwrite bool
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
	cmd.Flags().StringVarP(&p.dir, "dir", "d", "", "Path to prompt backup ZIP file (required)")
	cmd.Flags().BoolVar(&p.overwrite, "overwrite", false, "Overwrite existing prompt if it already exists")
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

	// Create API client
	client := openwebui.NewClient(cfg.OpenWebUIURL, cfg.OpenWebUIAPIKey)

	// Perform restore
	if err := restore.RestorePrompt(client, p.dir, p.overwrite); err != nil {
		logrus.Fatalf("restore failed: %w", err)
	}

	logrus.Info("Restore completed successfully!")
	return nil
}
