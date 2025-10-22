package plugins

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
	"github.com/vosiander/open-webui-backup/pkg/restore"
)

// RestoreKnowledgePlugin implements the Plugin interface for restoring knowledge
type RestoreKnowledgePlugin struct {
	inputPath string
	overwrite bool
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
	cmd.Flags().StringVarP(&p.inputPath, "input", "i", "", "Path to backup ZIP file (required)")
	cmd.Flags().BoolVar(&p.overwrite, "overwrite", false, "Overwrite existing files in the knowledge base")

	if err := cmd.MarkFlagRequired("input"); err != nil {
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

	if p.inputPath == "" {
		logrus.Fatal("input path is required (use --input flag)")
	}

	// Create API client
	client := openwebui.NewClient(cfg.OpenWebUIURL, cfg.OpenWebUIAPIKey)

	// Perform restore with overwrite flag
	if err := restore.RestoreKnowledge(client, p.inputPath, p.overwrite); err != nil {
		logrus.Fatalf("restore failed: %v", err)
	}

	logrus.Info("Restore completed successfully!")
	return nil
}
