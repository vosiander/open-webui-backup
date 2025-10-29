package plugins

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/backup"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/encryption"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
)

type BackupPlugin struct {
	out              string
	encryptRecipient []string
	prompts          bool
	tools            bool
	knowledge        bool
	models           bool
	files            bool
	chats            bool
	users            bool
	groups           bool
	feedbacks        bool
}

func NewBackupPlugin() *BackupPlugin {
	return &BackupPlugin{}
}

// Name returns the name of the plugin (used as command name)
func (p *BackupPlugin) Name() string {
	return "backup"
}

// Description returns a short description of the plugin
func (p *BackupPlugin) Description() string {
	return "Backup data from Open WebUI to a single file"
}

func (p *BackupPlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&p.out, "out", "o", "", "Output file path for the backup (required, .age extension will be appended)")
	cmd.MarkFlagRequired("out")
	cmd.Flags().StringSliceVar(&p.encryptRecipient, "encrypt-recipient", nil, "Encrypt backup with age public key(s) (or use OWUI_ENCRYPTED_RECIPIENT env variable)")
	cmd.Flags().BoolVar(&p.prompts, "prompts", false, "Include only prompts in backup")
	cmd.Flags().BoolVar(&p.tools, "tools", false, "Include only tools in backup")
	cmd.Flags().BoolVar(&p.knowledge, "knowledge", false, "Include only knowledge bases in backup")
	cmd.Flags().BoolVar(&p.models, "models", false, "Include only models in backup")
	cmd.Flags().BoolVar(&p.files, "files", false, "Include only files in backup")
	cmd.Flags().BoolVar(&p.chats, "chats", false, "Include only chats in backup")
	cmd.Flags().BoolVar(&p.groups, "groups", false, "Include only groups in backup (backed up before users)")
	cmd.Flags().BoolVar(&p.feedbacks, "feedbacks", false, "Include only feedbacks in backup (backed up before users)")
	cmd.Flags().BoolVar(&p.users, "users", false, "Include only users in backup (backed up LAST)")
}

// Execute runs the plugin with the given configuration
func (p *BackupPlugin) Execute(cfg *config.Config) error {
	logrus.Info("Starting backup...")

	if cfg.OpenWebUIAPIKey == "" {
		logrus.Fatalf("OPEN_WEBUI_API_KEY environment variable is required")
	}

	if p.out == "" {
		logrus.Fatalf("output file is required (use --out flag)")
	}

	// Get encryption recipients (required)
	recipients, err := encryption.GetEncryptRecipientsFromEnvOrFlag(p.encryptRecipient)
	if err != nil {
		logrus.Fatalf("Failed to get encryption recipients: %v", err)
	}

	// Create client
	client := openwebui.NewClient(cfg.OpenWebUIURL, cfg.OpenWebUIAPIKey)

	// Determine what to backup
	options := &backup.SelectiveBackupOptions{}

	// Check if any specific flags were provided
	anyFlagProvided := p.prompts || p.tools || p.knowledge || p.models || p.files || p.chats || p.users || p.groups || p.feedbacks

	if anyFlagProvided {
		// Selective backup based on flags
		options.Prompts = p.prompts
		options.Tools = p.tools
		options.Knowledge = p.knowledge
		options.Models = p.models
		options.Files = p.files
		options.Chats = p.chats
		options.Users = p.users
		options.Groups = p.groups
		options.Feedbacks = p.feedbacks
	} else {
		// Default: backup everything
		options.Prompts = true
		options.Tools = true
		options.Knowledge = true
		options.Models = true
		options.Files = true
		options.Chats = true
		options.Users = true
		options.Groups = true
		options.Feedbacks = true
	}

	// Prepare file paths for encryption
	// Always append .age extension to the user-specified output file
	encryptedFile := p.out
	if filepath.Ext(encryptedFile) != ".age" {
		encryptedFile = encryptedFile + ".age"
	}

	// Create temporary file for unencrypted backup
	tempFile := encryptedFile + ".tmp"

	// Perform the backup to temporary file (no progress callback for CLI)
	if err := backup.BackupSelective(client, tempFile, options, nil); err != nil {
		logrus.Fatalf("Failed to backup: %v", err)
	}

	// Encrypt the backup
	logrus.Info("Encrypting backup with public key(s)...")
	encryptOpts := &encryption.EncryptOptions{
		Recipients: recipients,
	}

	if err := encryption.EncryptFile(tempFile, encryptedFile, encryptOpts); err != nil {
		logrus.Fatalf("Failed to encrypt backup: %v", err)
	}

	// Remove unencrypted backup
	if err := os.Remove(tempFile); err != nil {
		logrus.Warnf("Failed to remove unencrypted backup: %v", err)
	}

	logrus.Infof("Backup completed successfully: %s", filepath.Base(encryptedFile))
	return nil
}
