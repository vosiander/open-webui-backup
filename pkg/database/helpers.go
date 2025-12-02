package database

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
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

// UseDockerPgTools checks if Docker should be used for PostgreSQL tools
// Checks USE_DOCKER_PG_TOOLS environment variable
func UseDockerPgTools() bool {
	val := os.Getenv("USE_DOCKER_PG_TOOLS")
	return val == "true" || val == "1" || val == "yes"
}

// IsDockerAvailable checks if Docker is installed and accessible
func IsDockerAvailable() bool {
	cmd := exec.Command("docker", "--version")
	return cmd.Run() == nil
}

// ExtractMajorVersion extracts the major version from a PostgreSQL version string
// Example: "PostgreSQL 17.4 (Debian 17.4-1.pgdg120+1)" -> "17"
func ExtractMajorVersion(versionStr string) string {
	// Look for pattern like "PostgreSQL X.Y" or just "X.Y"
	parts := strings.Fields(versionStr)
	for i, part := range parts {
		if part == "PostgreSQL" && i+1 < len(parts) {
			// Next field should be version
			version := parts[i+1]
			// Extract major version (before first dot)
			if idx := strings.Index(version, "."); idx > 0 {
				return version[:idx]
			}
			return version
		}
	}

	// Fallback: look for first number.number pattern
	for _, part := range parts {
		if strings.Contains(part, ".") && len(part) > 0 && part[0] >= '0' && part[0] <= '9' {
			if idx := strings.Index(part, "."); idx > 0 {
				return part[:idx]
			}
		}
	}

	return "latest"
}

// ResolveDockerHost resolves localhost addresses for Docker containers
// On Mac/Windows: localhost -> host.docker.internal
// On Linux: returns original host (will use --network=host flag instead)
func ResolveDockerHost(host string) string {
	// Check if host is localhost or 127.0.0.1
	isLocalhost := host == "localhost" || host == "127.0.0.1" || host == "::1"

	if !isLocalhost {
		return host
	}

	// On Mac and Windows, replace localhost with host.docker.internal
	// On Linux, keep localhost (we'll use --network=host flag instead)
	goos := os.Getenv("GOOS")
	if goos == "" {
		// Runtime detection
		goos = "unknown"
		// We'll use a simple check: if /proc exists, it's likely Linux
		if _, err := os.Stat("/proc"); err == nil {
			goos = "linux"
		} else if _, err := os.Stat("/Applications"); err == nil {
			goos = "darwin"
		}
	}

	if goos == "linux" {
		return host // Keep localhost, will use --network=host
	}

	// Mac (darwin) or Windows
	return "host.docker.internal"
}

// NeedsDockerHostNetwork returns true if Docker should use --network=host
// This is needed on Linux when connecting to localhost
func NeedsDockerHostNetwork(host string) bool {
	isLocalhost := host == "localhost" || host == "127.0.0.1" || host == "::1"
	if !isLocalhost {
		return false
	}

	// Check if running on Linux
	goos := os.Getenv("GOOS")
	if goos == "" {
		// Runtime detection
		if _, err := os.Stat("/proc"); err == nil {
			return true
		}
		return false
	}

	return goos == "linux"
}
