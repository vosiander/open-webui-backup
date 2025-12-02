package plugins

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/database"
	"github.com/vosiander/open-webui-backup/pkg/encryption"
)

type BackupDatabasePlugin struct {
	postgresURL      string
	out              string
	encryptRecipient []string
	verbose          bool
}

func NewBackupDatabasePlugin() *BackupDatabasePlugin {
	return &BackupDatabasePlugin{}
}

// Name returns the name of the plugin (used as command name)
func (p *BackupDatabasePlugin) Name() string {
	return "backup-database"
}

// Description returns a short description of the plugin
func (p *BackupDatabasePlugin) Description() string {
	return "Backup PostgreSQL database to an encrypted ZIP file"
}

func (p *BackupDatabasePlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&p.postgresURL, "postgres-url", "", "PostgreSQL connection URL (or use POSTGRES_URL env variable)")
	cmd.Flags().StringVarP(&p.out, "out", "o", "", "Output file path for the backup (required, .age extension will be appended)")
	cmd.MarkFlagRequired("out")
	cmd.Flags().StringSliceVar(&p.encryptRecipient, "encrypt-recipient", nil, "Encrypt backup with age public key(s) (or use OWUI_ENCRYPTED_RECIPIENT env variable)")
	cmd.Flags().BoolVarP(&p.verbose, "verbose", "v", false, "Enable verbose output (show Docker commands and pg_dump output)")
}

// Execute runs the plugin with the given configuration
func (p *BackupDatabasePlugin) Execute(cfg *config.Config) error {
	logrus.Info("Starting database backup...")

	// Check if PostgreSQL tools are available
	if err := database.CheckToolsAvailable(); err != nil {
		return fmt.Errorf("PostgreSQL tools check failed: %w", err)
	}

	// Get PostgreSQL URL from flag or environment
	postgresURL := p.postgresURL
	if postgresURL == "" {
		postgresURL = database.GetPostgresURLFromEnv()
	}

	if postgresURL == "" {
		return fmt.Errorf("PostgreSQL connection URL is required (use --postgres-url flag or POSTGRES_URL environment variable)")
	}

	// Parse connection URL
	dbConfig, err := database.ParsePostgresURL(postgresURL)
	if err != nil {
		return fmt.Errorf("failed to parse PostgreSQL URL: %w", err)
	}

	logrus.Infof("Connecting to database: %s", database.FormatConnectionInfo(dbConfig))

	// Test connection
	if err := database.TestConnection(dbConfig); err != nil {
		return fmt.Errorf("database connection test failed: %w", err)
	}

	// Get encryption recipients (required) - supports both files and direct recipient strings
	recipients, err := encryption.GetEncryptRecipientsFromEnvOrFlag(p.encryptRecipient)
	if err != nil {
		return fmt.Errorf("failed to get encryption recipients: %w", err)
	}

	// Prepare file paths for encryption
	encryptedFile := p.out
	if filepath.Ext(encryptedFile) != ".age" {
		encryptedFile = encryptedFile + ".age"
	}

	// Create temporary file for unencrypted backup
	tempFile := encryptedFile + ".tmp"

	// Create the database backup ZIP
	if err := p.createDatabaseBackupZip(tempFile, dbConfig); err != nil {
		os.Remove(tempFile) // Clean up temp file on error
		return fmt.Errorf("failed to create database backup: %w", err)
	}

	// Encrypt the backup
	logrus.Info("Encrypting database backup with public key(s)...")
	encryptOpts := &encryption.EncryptOptions{
		Recipients: recipients,
	}

	if err := encryption.EncryptFile(tempFile, encryptedFile, encryptOpts); err != nil {
		os.Remove(tempFile) // Clean up temp file on error
		return fmt.Errorf("failed to encrypt backup: %w", err)
	}

	// Remove unencrypted backup
	if err := os.Remove(tempFile); err != nil {
		logrus.Warnf("Failed to remove unencrypted backup: %v", err)
	}

	logrus.Infof("Database backup completed successfully: %s", filepath.Base(encryptedFile))
	return nil
}

// createDatabaseBackupZip creates a ZIP file containing the database dump
func (p *BackupDatabasePlugin) createDatabaseBackupZip(outputPath string, dbConfig *database.DatabaseConfig) error {
	// Create database dump
	dumpOptions := &database.DumpOptions{
		Format:       "plain",
		NoOwner:      true,
		NoPrivileges: true,
		Verbose:      p.verbose,
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

	// Create ZIP file
	zipFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add database dump to database/dump.sql
	dumpFile, err := zipWriter.Create("database/dump.sql")
	if err != nil {
		return fmt.Errorf("failed to create dump.sql in zip: %w", err)
	}
	if _, err := dumpFile.Write(dumpData); err != nil {
		return fmt.Errorf("failed to write dump.sql: %w", err)
	}

	// Add metadata
	metadata := &database.DatabaseBackupMetadata{
		BackupTimestamp: time.Now().UTC().Format(time.RFC3339),
		DatabaseName:    dbConfig.Database,
		PostgresVersion: version,
		DumpFormat:      "plain",
		Compressed:      false,
	}

	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	metadataFile, err := zipWriter.Create("database/metadata.json")
	if err != nil {
		return fmt.Errorf("failed to create metadata.json in zip: %w", err)
	}
	if _, err := metadataFile.Write(metadataJSON); err != nil {
		return fmt.Errorf("failed to write metadata.json: %w", err)
	}

	logrus.Infof("Database backup ZIP created: %s (%d bytes)", filepath.Base(outputPath), len(dumpData))
	return nil
}
