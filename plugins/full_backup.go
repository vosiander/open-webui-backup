package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/backup"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/database"
	"github.com/vosiander/open-webui-backup/pkg/encryption"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
)

// FullBackupPlugin creates a backup with automatic identity management
type FullBackupPlugin struct {
	path      string
	database  bool
	prompts   bool
	tools     bool
	knowledge bool
	models    bool
	files     bool
	chats     bool
	users     bool
	groups    bool
	feedbacks bool
}

// NewFullBackupPlugin creates a new instance of the FullBackupPlugin
func NewFullBackupPlugin() *FullBackupPlugin {
	return &FullBackupPlugin{}
}

// Name returns the command name
func (p *FullBackupPlugin) Name() string {
	return "full-backup"
}

// Description returns the command description
func (p *FullBackupPlugin) Description() string {
	return "Create a full backup with automatic age encryption and identity management"
}

// SetupFlags configures the command-line flags
func (p *FullBackupPlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&p.path, "path", "", "Directory for identity files and backup output (required)")
	cmd.MarkFlagRequired("path")
	cmd.Flags().BoolVar(&p.database, "database", false, "Include database backup (requires POSTGRES_URL env variable)")
	cmd.Flags().BoolVar(&p.prompts, "prompts", false, "Include only prompts in backup")
	cmd.Flags().BoolVar(&p.tools, "tools", false, "Include only tools in backup")
	cmd.Flags().BoolVar(&p.knowledge, "knowledge", false, "Include only knowledge bases in backup")
	cmd.Flags().BoolVar(&p.models, "models", false, "Include only models in backup")
	cmd.Flags().BoolVar(&p.files, "files", false, "Include only files in backup")
	cmd.Flags().BoolVar(&p.chats, "chats", false, "Include only chats in backup")
	cmd.Flags().BoolVar(&p.users, "users", false, "Include only users in backup")
	cmd.Flags().BoolVar(&p.groups, "groups", false, "Include only groups in backup")
	cmd.Flags().BoolVar(&p.feedbacks, "feedbacks", false, "Include only feedbacks in backup")
}

// Execute creates the backup with automatic identity management
func (p *FullBackupPlugin) Execute(cfg *config.Config) error {
	log := logrus.WithField("plugin", p.Name())

	// Validate API key
	if cfg.OpenWebUIAPIKey == "" {
		return fmt.Errorf("OPEN_WEBUI_API_KEY environment variable is required")
	}

	// Create path directory if needed
	if err := os.MkdirAll(p.path, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Check for or generate identity files
	recipient, newlyGenerated, err := ensureIdentityFiles(p.path, log)
	if err != nil {
		return fmt.Errorf("failed to ensure identity files: %w", err)
	}

	if newlyGenerated {
		logrus.Info("✓ Generated new age identity keypair")
	} else {
		logrus.Info("✓ Using existing age identity keypair")
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

	// Generate timestamped filename
	timestamp := time.Now().Format("20060102-150405")
	backupFilename := fmt.Sprintf("backup-%s.zip.age", timestamp)
	backupPath := filepath.Join(p.path, backupFilename)

	log.Infof("Creating backup: %s", backupFilename)

	// Create temporary unencrypted file
	tempFile := backupPath + ".tmp"
	defer os.Remove(tempFile) // Ensure cleanup

	// Perform the backup
	if err := backup.BackupSelective(client, tempFile, options, nil); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Auto-enable database backup if POSTGRES_URL is set and flag not explicitly set
	includeDatabase := p.database
	if !includeDatabase && database.IsPostgresURLSet() {
		includeDatabase = true
		log.Info("POSTGRES_URL detected, including database backup automatically")
	}

	// Conditionally add database backup to the ZIP
	if includeDatabase {
		if err := p.addDatabaseBackupToZip(tempFile, log); err != nil {
			logrus.Warnf("Database backup skipped: %v", err)
			logrus.Warnf("⚠️  Database backup skipped: %v", err)
		} else {
			logrus.Info("✓ Database backup included")
		}
	}

	// Encrypt the backup
	log.Info("Encrypting backup...")
	encryptOpts := &encryption.EncryptOptions{
		Recipients: []string{recipient},
	}

	if err := encryption.EncryptFile(tempFile, backupPath, encryptOpts); err != nil {
		return fmt.Errorf("failed to encrypt backup: %w", err)
	}

	// Remove temporary file
	os.Remove(tempFile)

	// Print success message
	logrus.Info("✓ Backup completed successfully!\n")
	logrus.Info("Files created:")
	logrus.Infof("  Identity (private key): %s", filepath.Join(p.path, "identity.txt"))
	logrus.Infof("  Recipient (public key): %s", filepath.Join(p.path, "recipient.txt"))
	logrus.Infof("  Backup: %s", backupPath)
	logrus.Info("To verify your backup:")
	logrus.Infof("  owuiback verify --path %s", p.path)
	logrus.Info("IMPORTANT: Keep identity.txt secure - it's needed to decrypt and restore your backup!")

	return nil
}

// ensureIdentityFiles checks for existing identity files or generates new ones
// Returns the recipient public key, whether new files were created, and any error
func ensureIdentityFiles(dir string, log *logrus.Entry) (string, bool, error) {
	identityPath := filepath.Join(dir, "identity.txt")
	recipientPath := filepath.Join(dir, "recipient.txt")

	// Check if both files exist
	_, identityErr := os.Stat(identityPath)
	_, recipientErr := os.Stat(recipientPath)

	if identityErr == nil && recipientErr == nil {
		// Both files exist, read recipient
		log.Info("Using existing identity files")
		recipientData, err := os.ReadFile(recipientPath)
		if err != nil {
			return "", false, fmt.Errorf("failed to read recipient.txt: %w", err)
		}
		recipient := string(recipientData)
		// Trim whitespace including newlines
		recipient = strings.TrimSpace(recipient)
		return recipient, false, nil
	}

	// One or both files missing, generate new keypair
	log.Info("Generating new age identity keypair...")

	// Generate identity
	identity, err := encryption.GenerateIdentity()
	if err != nil {
		return "", false, fmt.Errorf("failed to generate identity: %w", err)
	}

	// Save private key (identity) with restrictive permissions
	privateKey := identity.String()
	if err := os.WriteFile(identityPath, []byte(privateKey+"\n"), 0600); err != nil {
		return "", false, fmt.Errorf("failed to write identity.txt: %w", err)
	}
	log.Infof("Saved private key to: %s", identityPath)

	// Save public key (recipient)
	publicKey := identity.Recipient().String()
	if err := os.WriteFile(recipientPath, []byte(publicKey+"\n"), 0644); err != nil {
		// Clean up identity file if recipient save fails
		os.Remove(identityPath)
		return "", false, fmt.Errorf("failed to write recipient.txt: %w", err)
	}
	log.Infof("Saved public key to: %s", recipientPath)

	return publicKey, true, nil
}

// addDatabaseBackupToZip adds database backup to an existing ZIP file
func (p *FullBackupPlugin) addDatabaseBackupToZip(zipPath string, log *logrus.Entry) error {
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

	log.Infof("Adding database backup for: %s", database.FormatConnectionInfo(dbConfig))

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
		log.Warnf("Failed to get PostgreSQL version: %v", err)
		version = "unknown"
	}

	// Add database dump and metadata to existing ZIP
	if err := backup.AddDatabaseToZip(zipPath, dumpData, dbConfig.Database, version); err != nil {
		return fmt.Errorf("failed to add database to ZIP: %w", err)
	}

	log.Infof("Database backup added successfully (%d bytes)", len(dumpData))
	return nil
}
