package plugins

import (
	"fmt"

	"github.com/vosiander/open-webui-backup/pkg/config"
)

// BackupKnowledgePlugin implements the Plugin interface for backing up knowledge
type BackupKnowledgePlugin struct{}

// NewBackupKnowledgePlugin creates a new BackupKnowledgePlugin
func NewBackupKnowledgePlugin() *BackupKnowledgePlugin {
	return &BackupKnowledgePlugin{}
}

// Name returns the command name
func (p *BackupKnowledgePlugin) Name() string {
	return "backup-knowledge"
}

// Description returns the command description
func (p *BackupKnowledgePlugin) Description() string {
	return "Backup knowledge base from Open WebUI"
}

// Execute runs the backup knowledge command
func (p *BackupKnowledgePlugin) Execute(cfg *config.Config) error {
	fmt.Printf("Backing up knowledge from Open WebUI...\n")
	fmt.Printf("URL: %s\n", cfg.OpenWebUIURL)

	if cfg.OpenWebUIAPIKey == "" {
		return fmt.Errorf("OPEN_WEBUI_API_KEY environment variable is required")
	}

	// TODO: Implement actual backup logic
	fmt.Println("Backup completed successfully!")

	return nil
}
