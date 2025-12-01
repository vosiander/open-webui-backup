package plugins

import (
	"archive/zip"
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/database"
	"github.com/vosiander/open-webui-backup/pkg/encryption"
)

type RestoreDatabasePlugin struct {
	postgresURL     string
	file            string
	decryptIdentity []string
	clean           bool
	createDB        bool
}

func NewRestoreDatabasePlugin() *RestoreDatabasePlugin {
	return &RestoreDatabasePlugin{}
}

// Name returns the name of the plugin (used as command name)
func (p *RestoreDatabasePlugin) Name() string {
	return "restore-database"
}

// Description returns a short description of the plugin
func (p *RestoreDatabasePlugin) Description() string {
	return "Restore PostgreSQL database from an encrypted backup"
}

func (p *RestoreDatabasePlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&p.postgresURL, "postgres-url", "", "PostgreSQL connection URL (or use POSTGRES_URL env variable)")
	cmd.Flags().StringVarP(&p.file, "file", "f", "", "Encrypted backup file to restore from (required)")
	cmd.MarkFlagRequired("file")
	cmd.Flags().StringSliceVar(&p.decryptIdentity, "decrypt-identity", nil, "Age identity file(s) for decryption (or use OWUI_DECRYPTION_IDENTITY env variable)")
	cmd.Flags().BoolVar(&p.clean, "clean", false, "Drop existing objects before restoring (use with caution)")
	cmd.Flags().BoolVar(&p.createDB, "create-db", false, "Create database if it doesn't exist")
}

// Execute runs the plugin with the given configuration
func (p *RestoreDatabasePlugin) Execute(cfg *config.Config) error {
	logrus.Info("Starting database restore...")

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

	logrus.Infof("Target database: %s", database.FormatConnectionInfo(dbConfig))

	// Test connection
	if err := database.TestConnection(dbConfig); err != nil {
		return fmt.Errorf("database connection test failed: %w", err)
	}

	// Get decryption identities
	identities, err := encryption.GetDecryptIdentitiesFromEnvOrFlag(p.decryptIdentity)
	if err != nil {
		return fmt.Errorf("failed to get decryption identities: %w", err)
	}

	// Decrypt the backup file
	logrus.Info("Decrypting backup file...")
	tempDecrypted := p.file + ".decrypted.tmp"
	defer os.Remove(tempDecrypted) // Clean up temp file

	decryptOpts := &encryption.DecryptOptions{
		Identities: identities,
	}

	if err := encryption.DecryptFile(p.file, tempDecrypted, decryptOpts); err != nil {
		return fmt.Errorf("failed to decrypt backup: %w", err)
	}

	// Extract database dump from ZIP
	dumpData, err := p.extractDatabaseDump(tempDecrypted)
	if err != nil {
		return fmt.Errorf("failed to extract database dump: %w", err)
	}

	// Restore the database
	restoreOptions := &database.RestoreOptions{
		Clean:        p.clean,
		CreateDB:     p.createDB,
		NoOwner:      true,
		NoPrivileges: true,
	}

	if err := database.RestoreDump(dbConfig, dumpData, restoreOptions); err != nil {
		return fmt.Errorf("failed to restore database: %w", err)
	}

	logrus.Info("Database restored successfully")
	return nil
}

// extractDatabaseDump extracts the database dump from a ZIP file
func (p *RestoreDatabasePlugin) extractDatabaseDump(zipPath string) ([]byte, error) {
	// Open ZIP file
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer zipReader.Close()

	// Find database/dump.sql in the ZIP
	var dumpFile *zip.File
	for _, f := range zipReader.File {
		if f.Name == "database/dump.sql" {
			dumpFile = f
			break
		}
	}

	if dumpFile == nil {
		return nil, fmt.Errorf("database/dump.sql not found in backup ZIP")
	}

	// Read the dump file
	rc, err := dumpFile.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open dump.sql: %w", err)
	}
	defer rc.Close()

	dumpData, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to read dump.sql: %w", err)
	}

	logrus.Infof("Extracted database dump from backup (%d bytes)", len(dumpData))
	return dumpData, nil
}
