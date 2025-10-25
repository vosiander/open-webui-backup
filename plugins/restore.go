package plugins

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/encryption"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
	"github.com/vosiander/open-webui-backup/pkg/restore"
)

type RestorePlugin struct {
	file            string
	overwrite       bool
	decryptIdentity []string
	prompts         bool
	tools           bool
	knowledge       bool
	models          bool
	files           bool
	chats           bool
}

func NewRestorePlugin() *RestorePlugin {
	return &RestorePlugin{}
}

// Name returns the name of the plugin (used as command name)
func (p *RestorePlugin) Name() string {
	return "restore"
}

// Description returns a short description of the plugin
func (p *RestorePlugin) Description() string {
	return "Restore data to Open WebUI from a backup file"
}

func (p *RestorePlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&p.file, "file", "f", "", "Backup file path to restore from (required)")
	cmd.MarkFlagRequired("file")
	cmd.Flags().BoolVar(&p.overwrite, "overwrite", false, "Overwrite existing data")
	cmd.Flags().StringSliceVar(&p.decryptIdentity, "decrypt-identity", nil, "Decrypt backup with age identity file(s) (or use OWUI_DECRYPT_IDENTITY env variable)")
	cmd.Flags().BoolVar(&p.prompts, "prompts", false, "Restore only prompts")
	cmd.Flags().BoolVar(&p.tools, "tools", false, "Restore only tools")
	cmd.Flags().BoolVar(&p.knowledge, "knowledge", false, "Restore only knowledge bases")
	cmd.Flags().BoolVar(&p.models, "models", false, "Restore only models")
	cmd.Flags().BoolVar(&p.files, "files", false, "Restore only files")
	cmd.Flags().BoolVar(&p.chats, "chats", false, "Restore only chats")
}

// Execute runs the plugin with the given configuration
func (p *RestorePlugin) Execute(cfg *config.Config) error {
	logrus.Info("Starting restore...")

	if cfg.OpenWebUIAPIKey == "" {
		logrus.Fatalf("OPEN_WEBUI_API_KEY environment variable is required")
	}

	if p.file == "" {
		logrus.Fatalf("backup file is required (use --file flag)")
	}

	// Get decryption identity files (required)
	identities, err := encryption.GetDecryptIdentityFilesFromEnvOrFlag(p.decryptIdentity)
	if err != nil {
		logrus.Fatalf("Failed to get decryption identity files: %v", err)
	}

	// Create client
	client := openwebui.NewClient(cfg.OpenWebUIURL, cfg.OpenWebUIAPIKey)

	// Create temporary file for decrypted backup
	tempFile := filepath.Join(os.TempDir(), "owuiback_restore_"+filepath.Base(p.file))
	if filepath.Ext(tempFile) == ".age" {
		tempFile = tempFile[:len(tempFile)-4]
	}

	// Decrypt the backup
	logrus.Info("Decrypting backup with identity file(s)...")
	decryptOpts := &encryption.DecryptOptions{
		IdentityFiles: identities,
	}

	if err := encryption.DecryptFile(p.file, tempFile, decryptOpts); err != nil {
		logrus.Fatalf("Failed to decrypt backup: %v", err)
	}

	// Ensure temporary file is cleaned up
	defer os.Remove(tempFile)

	logrus.Info("Backup decrypted successfully")

	// Determine what to restore
	options := &restore.SelectiveRestoreOptions{
		Prompts:   p.prompts,
		Tools:     p.tools,
		Knowledge: p.knowledge,
		Models:    p.models,
		Files:     p.files,
		Chats:     p.chats,
	}

	// If no specific types are selected, restore everything
	if !options.Prompts && !options.Tools && !options.Knowledge && !options.Models && !options.Files && !options.Chats {
		logrus.Info("No specific types selected, restoring all data from backup")
		options.Prompts = true
		options.Tools = true
		options.Knowledge = true
		options.Models = true
		options.Files = true
		options.Chats = true
	}

	// Perform the restore
	if err := restore.RestoreSelective(client, tempFile, options, p.overwrite); err != nil {
		logrus.Fatalf("Failed to restore: %v", err)
	}

	logrus.Info("Restore completed successfully")
	return nil
}
