package plugins

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/backup"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/database"
	"github.com/vosiander/open-webui-backup/pkg/encryption"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
)

type BackupPlugin struct {
	out              string
	encryptRecipient []string
	database         bool
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
	cmd.Flags().BoolVar(&p.database, "database", false, "Include database backup (auto-enabled if POSTGRES_URL is set)")
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

	// Auto-enable database backup if POSTGRES_URL is set and flag not explicitly set
	includeDatabase := p.database
	if !includeDatabase && database.IsPostgresURLSet() {
		includeDatabase = true
		logrus.Info("POSTGRES_URL detected, including database backup automatically")
	}

	// Conditionally add database backup to the ZIP
	if includeDatabase {
		if err := p.addDatabaseBackupToZip(tempFile); err != nil {
			logrus.Warnf("Database backup skipped: %v", err)
			logrus.Warnf("⚠️  Database backup skipped: %v", err)
		} else {
			logrus.Info("Database backup included")
			logrus.Info("✓ Database backup included")
		}
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

// addDatabaseBackupToZip adds database backup to an existing ZIP file
func (p *BackupPlugin) addDatabaseBackupToZip(zipPath string) error {
	// Check if POSTGRES_URL is set
	postgresURL := database.GetPostgresURLFromEnv()
	if postgresURL == "" {
		return fmt.Errorf("POSTGRES_URL environment variable not set")
	}

	// Check if PostgreSQL tools are available
	if err := database.CheckToolsAvailable(); err != nil {
		return fmt.Errorf("PostgreSQL tools not available: %w", err)
	}

	// Parse connection URL
	dbConfig, err := database.ParsePostgresURL(postgresURL)
	if err != nil {
		return fmt.Errorf("failed to parse POSTGRES_URL: %w", err)
	}

	logrus.Infof("Adding database backup for: %s", database.FormatConnectionInfo(dbConfig))

	// Test connection
	if err := database.TestConnection(dbConfig); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	// Create database dump
	dumpOptions := &database.DumpOptions{
		Format:       "plain",
		NoOwner:      true,
		NoPrivileges: true,
	}

	dumpData, err := database.CreateDump(dbConfig, dumpOptions)
	if err != nil {
		return fmt.Errorf("failed to create database dump: %w", err)
	}

	// Get PostgreSQL version for metadata
	version, err := database.GetPostgresVersion(dbConfig)
	if err != nil {
		logrus.Warnf("Failed to get PostgreSQL version: %v", err)
		version = "unknown"
	}

	// Add database dump and metadata to existing ZIP
	if err := backup.AddDatabaseToZip(zipPath, dumpData, dbConfig.Database, version); err != nil {
		return fmt.Errorf("failed to add database to ZIP: %w", err)
	}

	logrus.Infof("Database backup added successfully (%d bytes)", len(dumpData))
	return nil
}
