package database

import (
	"fmt"
	"os"
)

// GetPostgresURLFromEnv retrieves the POSTGRES_URL environment variable
func GetPostgresURLFromEnv() string {
	return os.Getenv("POSTGRES_URL")
}

// IsPostgresURLSet checks if the POSTGRES_URL environment variable is set
func IsPostgresURLSet() bool {
	return GetPostgresURLFromEnv() != ""
}

// ValidateConfig performs basic validation on a DatabaseConfig
func ValidateConfig(config *DatabaseConfig) error {
	if config == nil {
		return fmt.Errorf("database config is nil")
	}

	if config.Host == "" {
		return fmt.Errorf("database host is empty")
	}

	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("database port is invalid: %d", config.Port)
	}

	if config.Database == "" {
		return fmt.Errorf("database name is empty")
	}

	if config.User == "" {
		return fmt.Errorf("database user is empty")
	}

	return nil
}

// FormatConnectionInfo returns a safe string representation of the connection
// (without password) for logging purposes
func FormatConnectionInfo(config *DatabaseConfig) string {
	if config == nil {
		return "<nil config>"
	}
	return fmt.Sprintf("%s@%s:%d/%s", config.User, config.Host, config.Port, config.Database)
}

// GetPsqlPath returns the path to the psql binary
// Checks PSQL_BINARY environment variable, defaults to /usr/local/bin/psql
func GetPsqlPath() string {
	if path := os.Getenv("PSQL_BINARY"); path != "" {
		return path
	}
	return "/usr/local/bin/psql"
}

// GetPgDumpPath returns the path to the pg_dump binary
// Checks PG_DUMP_BINARY environment variable, defaults to /usr/local/bin/pg_dump
func GetPgDumpPath() string {
	if path := os.Getenv("PG_DUMP_BINARY"); path != "" {
		return path
	}
	return "/usr/local/bin/pg_dump"
}

// GetPgRestorePath returns the path to the pg_restore binary
// Checks PG_RESTORE_BINARY environment variable, defaults to /usr/local/bin/pg_restore
func GetPgRestorePath() string {
	if path := os.Getenv("PG_RESTORE_BINARY"); path != "" {
		return path
	}
	return "/usr/local/bin/pg_restore"
}
