package plugins

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
	"github.com/vosiander/open-webui-backup/pkg/restore"
)

type RestoreAllPlugin struct {
	dir       string
	overwrite bool
}

func NewRestoreAllPlugin() *RestoreAllPlugin {
	return &RestoreAllPlugin{}
}

// Name returns the name of the plugin (used as command name)
func (p *RestoreAllPlugin) Name() string {
	return "restore-all"
}

// Description returns a short description of the plugin
func (p *RestoreAllPlugin) Description() string {
	return "Restore all knowledge bases and models to Open WebUI from a backup directory"
}

func (p *RestoreAllPlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&p.dir, "dir", "d", "", "Directory containing backup ZIP files (required)")
	cmd.Flags().BoolVarP(&p.overwrite, "overwrite", "w", false, "Overwrite existing models and files")
	cmd.MarkFlagRequired("dir")
}

// Execute runs the plugin with the given configuration
func (p *RestoreAllPlugin) Execute(cfg *config.Config) error {
	logrus.Info("Starting full restore...")

	if cfg.OpenWebUIAPIKey == "" {
		logrus.Fatalf("OPEN_WEBUI_API_KEY environment variable is required")
	}

	if p.dir == "" {
		logrus.Fatalf("directory is required (use --dir flag)")
	}

	client := openwebui.NewClient(cfg.OpenWebUIURL, cfg.OpenWebUIAPIKey)

	if err := restore.RestoreAll(client, p.dir, p.overwrite); err != nil {
		logrus.Fatalf("Failed to restore: %v", err)
	}

	logrus.Info("Full restore completed successfully")
	return nil
}
