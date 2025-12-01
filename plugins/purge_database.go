package plugins

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/database"
)

type PurgeDatabasePlugin struct {
	postgresURL string
	force       bool
}

func NewPurgeDatabasePlugin() *PurgeDatabasePlugin {
	return &PurgeDatabasePlugin{}
}

// Name returns the name of the plugin (used as command name)
func (p *PurgeDatabasePlugin) Name() string {
	return "purge-database"
}

// Description returns a short description of the plugin
func (p *PurgeDatabasePlugin) Description() string {
	return "Show what would be deleted (dry-run) or purge database with --force flag"
}

func (p *PurgeDatabasePlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&p.postgresURL, "postgres-url", "", "PostgreSQL connection URL (or use POSTGRES_URL env variable)")
	cmd.Flags().BoolVarP(&p.force, "force", "f", false, "Actually delete data (without this flag, only shows what would be deleted)")
}

// Execute runs the plugin with the given configuration
func (p *PurgeDatabasePlugin) Execute(cfg *config.Config) error {
	// Determine if this is a dry-run or actual deletion
	dryRun := !p.force

	if dryRun {
		logrus.Info("Starting database purge (DRY RUN - showing what would be deleted)...")
		logrus.Warn("⚠️  DRY RUN MODE - No data will be deleted")
		logrus.Info("Add --force flag to actually delete the data\n")
	} else {
		logrus.Warn("Starting database purge - THIS WILL DELETE ALL DATA!")
		logrus.Warn("⚠️  WARNING: FORCE MODE - Data will be permanently deleted!\n")
	}

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

	// Perform the purge (or dry-run)
	if err := database.PurgeDatabase(dbConfig, dryRun); err != nil {
		return fmt.Errorf("failed to purge database: %w", err)
	}

	if dryRun {
		logrus.Info("✓ Dry run completed - no changes were made")
		logrus.Info("Run with --force flag to actually delete the data")
		logrus.Info("Dry run completed")
	} else {
		logrus.Info("✓ Database purged successfully")
		logrus.Info("Database purged successfully")
	}

	return nil
}
