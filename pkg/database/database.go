package database

import (
	"bytes"
	"fmt"
	"net/url"
	"os/exec"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// DatabaseConfig holds PostgreSQL connection details
type DatabaseConfig struct {
	URL      string // Full connection string
	Host     string
	Port     int
	Database string
	User     string
	Password string
}

// DumpOptions configures pg_dump behavior
type DumpOptions struct {
	Format       string // "plain" for SQL, "custom" for pg_restore format
	Verbose      bool
	Compress     int // 0-9 compression level (for custom format)
	SchemaOnly   bool
	DataOnly     bool
	NoOwner      bool
	NoPrivileges bool
}

// RestoreOptions configures pg_restore behavior
type RestoreOptions struct {
	CreateDB     bool // Create database before restore
	NoOwner      bool
	NoPrivileges bool
	Verbose      bool
}

// DatabaseBackupMetadata tracks database backup information
type DatabaseBackupMetadata struct {
	BackupTimestamp string
	DatabaseName    string
	PostgresVersion string
	DumpFormat      string
	Compressed      bool
}

// CheckToolsAvailable verifies that pg_dump and pg_restore are installed
func CheckToolsAvailable() error {
	// Check for pg_dump
	pgDumpPath := GetPgDumpPath()
	if _, err := exec.LookPath(pgDumpPath); err != nil {
		return fmt.Errorf("pg_dump not found at %s. Set PG_DUMP_BINARY environment variable or install PostgreSQL client tools", pgDumpPath)
	}

	// Check for pg_restore
	pgRestorePath := GetPgRestorePath()
	if _, err := exec.LookPath(pgRestorePath); err != nil {
		return fmt.Errorf("pg_restore not found at %s. Set PG_RESTORE_BINARY environment variable or install PostgreSQL client tools", pgRestorePath)
	}

	logrus.Debugf("PostgreSQL tools found: pg_dump=%s, pg_restore=%s", pgDumpPath, pgRestorePath)
	return nil
}

// ParsePostgresURL parses a PostgreSQL connection string into a DatabaseConfig
// Expected format: postgres://user:password@host:port/dbname
func ParsePostgresURL(connectionURL string) (*DatabaseConfig, error) {
	if connectionURL == "" {
		return nil, fmt.Errorf("connection URL is empty")
	}

	// Parse the URL
	u, err := url.Parse(connectionURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection URL: %w", err)
	}

	// Validate scheme
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return nil, fmt.Errorf("invalid URL scheme: expected 'postgres' or 'postgresql', got '%s'", u.Scheme)
	}

	// Extract user
	user := u.User.Username()
	if user == "" {
		return nil, fmt.Errorf("username not found in connection URL")
	}

	// Extract password
	password, _ := u.User.Password()

	// Extract host
	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("host not found in connection URL")
	}

	// Extract port
	portStr := u.Port()
	port := 5432 // Default PostgreSQL port
	if portStr != "" {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid port number: %s", portStr)
		}
	}

	// Extract database name
	database := strings.TrimPrefix(u.Path, "/")
	if database == "" {
		return nil, fmt.Errorf("database name not found in connection URL")
	}

	config := &DatabaseConfig{
		URL:      connectionURL,
		Host:     host,
		Port:     port,
		Database: database,
		User:     user,
		Password: password,
	}

	logrus.Debugf("Parsed database config: host=%s, port=%d, database=%s, user=%s",
		config.Host, config.Port, config.Database, config.User)

	return config, nil
}

// TestConnection validates that the database is accessible
func TestConnection(config *DatabaseConfig) error {
	if config == nil {
		return fmt.Errorf("database config is nil")
	}

	logrus.Debug("Testing database connection...")

	// Use psql to test connection with a simple query
	cmd := exec.Command(GetPsqlPath(),
		"-h", config.Host,
		"-p", strconv.Itoa(config.Port),
		"-U", config.User,
		"-d", config.Database,
		"-c", "SELECT 1;",
		"-q", // Quiet mode
	)

	// Set PGPASSWORD environment variable
	cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%s", config.Password))

	// Run the command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("database connection test failed: %w\nOutput: %s", err, string(output))
	}

	logrus.Debug("Database connection test successful")
	return nil
}

// CreateDump creates a database dump using pg_dump (or Docker)
func CreateDump(config *DatabaseConfig, options *DumpOptions) ([]byte, error) {
	if config == nil {
		return nil, fmt.Errorf("database config is nil")
	}

	if options == nil {
		options = &DumpOptions{
			Format:       "plain",
			NoOwner:      true,
			NoPrivileges: true,
		}
	}

	// Check if Docker mode should be used
	if UseDockerPgTools() {
		return createDumpWithDocker(config, options)
	}

	logrus.Infof("Creating database dump for '%s'...", config.Database)

	// Build pg_dump command
	args := []string{
		"-h", config.Host,
		"-p", strconv.Itoa(config.Port),
		"-U", config.User,
		"-d", config.Database,
	}

	// Add format option
	if options.Format == "custom" {
		args = append(args, "-Fc") // Custom format
		if options.Compress > 0 {
			args = append(args, fmt.Sprintf("-Z%d", options.Compress))
		}
	} else {
		args = append(args, "-Fp") // Plain SQL format
	}

	// Add other options
	if options.NoOwner {
		args = append(args, "--no-owner")
	}
	if options.NoPrivileges {
		args = append(args, "--no-privileges")
	}
	if options.SchemaOnly {
		args = append(args, "--schema-only")
	}
	if options.DataOnly {
		args = append(args, "--data-only")
	}
	if options.Verbose {
		args = append(args, "-v")
	}

	cmd := exec.Command(GetPgDumpPath(), args...)

	// Set PGPASSWORD environment variable
	cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%s", config.Password))

	// Capture stdout (the dump) and stderr (logs/errors)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		// Check for version mismatch error
		stderrStr := stderr.String()
		if strings.Contains(stderrStr, "server version mismatch") {
			return nil, fmt.Errorf(`pg_dump failed due to version mismatch.

Error: %s

Solutions:
1. Enable Docker mode: export USE_DOCKER_PG_TOOLS=true
   (This will automatically use a matching PostgreSQL version via Docker)

2. Upgrade your local PostgreSQL client tools to match the server version

3. Set PG_DUMP_BINARY to point to a compatible pg_dump version:
   export PG_DUMP_BINARY=/path/to/compatible/pg_dump

4. Use Docker manually:
   docker run --rm -e PGPASSWORD=xxx postgres:<version> pg_dump ...`, stderrStr)
		}
		return nil, fmt.Errorf("pg_dump failed: %w\nError output: %s", err, stderrStr)
	}

	// Log any warnings from stderr
	if stderr.Len() > 0 {
		logrus.Debugf("pg_dump output: %s", stderr.String())
	}

	dumpData := stdout.Bytes()
	logrus.Infof("Database dump created successfully (%d bytes)", len(dumpData))

	return dumpData, nil
}

// createDumpWithDocker creates a database dump using Docker with matching PostgreSQL version
func createDumpWithDocker(config *DatabaseConfig, options *DumpOptions) ([]byte, error) {
	// Check if Docker is available
	if !IsDockerAvailable() {
		return nil, fmt.Errorf("Docker is not available. Install Docker or set USE_DOCKER_PG_TOOLS=false")
	}

	// Get server version to determine Docker image
	serverVersion, err := GetPostgresVersion(config)
	if err != nil {
		logrus.Warnf("Failed to detect server version, using 'latest': %v", err)
		serverVersion = "latest"
	}

	majorVersion := ExtractMajorVersion(serverVersion)
	dockerImage := fmt.Sprintf("postgres:%s", majorVersion)

	// Resolve host for Docker (localhost -> host.docker.internal on Mac/Windows)
	dockerHost := ResolveDockerHost(config.Host)

	logrus.Infof("Creating database dump for '%s' using Docker image %s...", config.Database, dockerImage)
	if dockerHost != config.Host {
		logrus.Debugf("Resolved host '%s' to '%s' for Docker", config.Host, dockerHost)
	}

	// Build pg_dump arguments
	pgDumpArgs := []string{
		"-h", dockerHost,
		"-p", strconv.Itoa(config.Port),
		"-U", config.User,
		"-d", config.Database,
	}

	// Add format option
	if options.Format == "custom" {
		pgDumpArgs = append(pgDumpArgs, "-Fc")
		if options.Compress > 0 {
			pgDumpArgs = append(pgDumpArgs, fmt.Sprintf("-Z%d", options.Compress))
		}
	} else {
		pgDumpArgs = append(pgDumpArgs, "-Fp")
	}

	// Add other options
	if options.NoOwner {
		pgDumpArgs = append(pgDumpArgs, "--no-owner")
	}
	if options.NoPrivileges {
		pgDumpArgs = append(pgDumpArgs, "--no-privileges")
	}
	if options.SchemaOnly {
		pgDumpArgs = append(pgDumpArgs, "--schema-only")
	}
	if options.DataOnly {
		pgDumpArgs = append(pgDumpArgs, "--data-only")
	}
	if options.Verbose {
		pgDumpArgs = append(pgDumpArgs, "-v")
	}

	// Build Docker command
	dockerArgs := []string{"run", "--rm"}

	// Add --network=host on Linux when connecting to localhost
	if NeedsDockerHostNetwork(config.Host) {
		dockerArgs = append(dockerArgs, "--network=host")
		logrus.Debug("Using --network=host for Docker (Linux + localhost)")
	}

	dockerArgs = append(dockerArgs,
		"-e", fmt.Sprintf("PGPASSWORD=%s", config.Password),
		dockerImage,
		"pg_dump",
	)
	dockerArgs = append(dockerArgs, pgDumpArgs...)

	// Log the full command if verbose
	if options.Verbose {
		// Mask the password in the log
		maskedArgs := make([]string, len(dockerArgs))
		copy(maskedArgs, dockerArgs)
		for i, arg := range maskedArgs {
			if strings.HasPrefix(arg, "PGPASSWORD=") {
				maskedArgs[i] = "PGPASSWORD=***"
			}
		}
		logrus.Infof("Executing: docker %s", strings.Join(maskedArgs, " "))
	}

	cmd := exec.Command("docker", dockerArgs...)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("Docker pg_dump failed: %w\nError output: %s", err, stderr.String())
	}

	// Log stderr output based on verbose setting
	if stderr.Len() > 0 {
		if options.Verbose {
			logrus.Infof("pg_dump output: %s", stderr.String())
		} else {
			logrus.Debugf("Docker pg_dump output: %s", stderr.String())
		}
	}

	dumpData := stdout.Bytes()
	logrus.Infof("Database dump created successfully via Docker (%d bytes)", len(dumpData))

	return dumpData, nil
}

// RestoreDump restores a database dump using pg_restore or psql (or Docker)
func RestoreDump(config *DatabaseConfig, dumpData []byte, options *RestoreOptions) error {
	if config == nil {
		return fmt.Errorf("database config is nil")
	}

	if len(dumpData) == 0 {
		return fmt.Errorf("dump data is empty")
	}

	if options == nil {
		options = &RestoreOptions{
			NoOwner:      true,
			NoPrivileges: true,
		}
	}

	// Check if Docker mode should be used
	if UseDockerPgTools() {
		return restoreDumpWithDocker(config, dumpData, options)
	}

	logrus.Infof("Restoring database dump to '%s'...", config.Database)

	// Determine if this is a custom format or plain SQL
	isCustomFormat := len(dumpData) > 5 && string(dumpData[:5]) == "PGDMP"

	if isCustomFormat {
		// Use pg_restore for custom format
		return restoreWithPgRestore(config, dumpData, options)
	} else {
		// Use psql for plain SQL format
		return restoreWithPsql(config, dumpData, options)
	}
}

// restoreDumpWithDocker restores a database dump using Docker with matching PostgreSQL version
func restoreDumpWithDocker(config *DatabaseConfig, dumpData []byte, options *RestoreOptions) error {
	// Check if Docker is available
	if !IsDockerAvailable() {
		return fmt.Errorf("Docker is not available. Install Docker or set USE_DOCKER_PG_TOOLS=false")
	}

	// Get server version to determine Docker image
	serverVersion, err := GetPostgresVersion(config)
	if err != nil {
		logrus.Warnf("Failed to detect server version, using 'latest': %v", err)
		serverVersion = "latest"
	}

	majorVersion := ExtractMajorVersion(serverVersion)
	dockerImage := fmt.Sprintf("postgres:%s", majorVersion)

	// Resolve host for Docker (localhost -> host.docker.internal on Mac/Windows)
	dockerHost := ResolveDockerHost(config.Host)

	logrus.Infof("Restoring database dump to '%s' using Docker image %s...", config.Database, dockerImage)
	if dockerHost != config.Host {
		logrus.Debugf("Resolved host '%s' to '%s' for Docker", config.Host, dockerHost)
	}

	// Determine if this is a custom format or plain SQL
	isCustomFormat := len(dumpData) > 5 && string(dumpData[:5]) == "PGDMP"

	var tool string
	var toolArgs []string

	if isCustomFormat {
		// Use pg_restore for custom format
		tool = "pg_restore"
		toolArgs = []string{
			"-h", dockerHost,
			"-p", strconv.Itoa(config.Port),
			"-U", config.User,
			"-d", config.Database,
		}
		if options.NoOwner {
			toolArgs = append(toolArgs, "--no-owner")
		}
		if options.NoPrivileges {
			toolArgs = append(toolArgs, "--no-privileges")
		}
		if options.Verbose {
			toolArgs = append(toolArgs, "-v")
		}
	} else {
		// Use psql for plain SQL
		tool = "psql"
		toolArgs = []string{
			"-h", dockerHost,
			"-p", strconv.Itoa(config.Port),
			"-U", config.User,
			"-d", config.Database,
			"-q",
		}
		if options.Verbose {
			toolArgs = append(toolArgs, "-a")
		}
	}

	// Build Docker command
	dockerArgs := []string{"run", "--rm", "-i"} // Interactive mode for stdin

	// Add --network=host on Linux when connecting to localhost
	if NeedsDockerHostNetwork(config.Host) {
		dockerArgs = append(dockerArgs, "--network=host")
		logrus.Debug("Using --network=host for Docker (Linux + localhost)")
	}

	dockerArgs = append(dockerArgs,
		"-e", fmt.Sprintf("PGPASSWORD=%s", config.Password),
		dockerImage,
		tool,
	)
	dockerArgs = append(dockerArgs, toolArgs...)

	// Log the full command if verbose
	if options.Verbose {
		// Mask the password in the log
		maskedArgs := make([]string, len(dockerArgs))
		copy(maskedArgs, dockerArgs)
		for i, arg := range maskedArgs {
			if strings.HasPrefix(arg, "PGPASSWORD=") {
				maskedArgs[i] = "PGPASSWORD=***"
			}
		}
		logrus.Infof("Executing: docker %s", strings.Join(maskedArgs, " "))
	}

	cmd := exec.Command("docker", dockerArgs...)

	// Pipe dump data to stdin
	cmd.Stdin = bytes.NewReader(dumpData)

	// Capture stderr for logging
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Docker %s failed: %w\nError output: %s", tool, err, stderr.String())
	}

	// Log stderr output based on verbose setting
	if stderr.Len() > 0 {
		if options.Verbose {
			logrus.Infof("%s output: %s", tool, stderr.String())
		} else {
			logrus.Debugf("Docker %s output: %s", tool, stderr.String())
		}
	}

	logrus.Info("Database restored successfully via Docker")
	return nil
}

// restoreWithPgRestore restores a custom format dump using pg_restore
func restoreWithPgRestore(config *DatabaseConfig, dumpData []byte, options *RestoreOptions) error {
	logrus.Debug("Using pg_restore for custom format dump")

	// Build pg_restore command
	args := []string{
		"-h", config.Host,
		"-p", strconv.Itoa(config.Port),
		"-U", config.User,
		"-d", config.Database,
	}

	// Add options
	if options.NoOwner {
		args = append(args, "--no-owner")
	}
	if options.NoPrivileges {
		args = append(args, "--no-privileges")
	}
	if options.Verbose {
		args = append(args, "-v")
	}

	cmd := exec.Command(GetPgRestorePath(), args...)

	// Set PGPASSWORD environment variable
	cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%s", config.Password))

	// Pipe dump data to stdin
	cmd.Stdin = bytes.NewReader(dumpData)

	// Capture stderr for logging
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_restore failed: %w\nError output: %s", err, stderr.String())
	}

	// Log any output
	if stderr.Len() > 0 {
		logrus.Debugf("pg_restore output: %s", stderr.String())
	}

	logrus.Info("Database restored successfully")
	return nil
}

// restoreWithPsql restores a plain SQL dump using psql
func restoreWithPsql(config *DatabaseConfig, dumpData []byte, options *RestoreOptions) error {
	logrus.Debug("Using psql for plain SQL dump")

	// Build psql command
	args := []string{
		"-h", config.Host,
		"-p", strconv.Itoa(config.Port),
		"-U", config.User,
		"-d", config.Database,
		"-q", // Quiet mode
	}

	if options.Verbose {
		args = append(args, "-a") // Echo all
	}

	cmd := exec.Command(GetPsqlPath(), args...)

	// Set PGPASSWORD environment variable
	cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%s", config.Password))

	// Pipe SQL data to stdin
	cmd.Stdin = bytes.NewReader(dumpData)

	// Capture stderr for logging
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("psql restore failed: %w\nError output: %s", err, stderr.String())
	}

	// Log any output
	if stderr.Len() > 0 {
		logrus.Debugf("psql output: %s", stderr.String())
	}

	logrus.Info("Database restored successfully")
	return nil
}

// PurgeDatabase drops all tables and schemas in the database
func PurgeDatabase(config *DatabaseConfig, dryRun bool) error {
	if config == nil {
		return fmt.Errorf("database config is nil")
	}

	if dryRun {
		logrus.Info("DRY RUN: Showing what would be deleted without actually deleting...")
	} else {
		logrus.Warnf("PURGING database '%s' - this will drop all tables and data!", config.Database)
	}

	// SQL to drop all tables in public schema
	dropSQL := `
DO $$ DECLARE
    r RECORD;
BEGIN
    -- Drop all tables
    FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') LOOP
        RAISE NOTICE 'Dropping table: %', r.tablename;
        EXECUTE 'DROP TABLE IF EXISTS public.' || quote_ident(r.tablename) || ' CASCADE';
    END LOOP;
    
    -- Drop all sequences
    FOR r IN (SELECT sequence_name FROM information_schema.sequences WHERE sequence_schema = 'public') LOOP
        RAISE NOTICE 'Dropping sequence: %', r.sequence_name;
        EXECUTE 'DROP SEQUENCE IF EXISTS public.' || quote_ident(r.sequence_name) || ' CASCADE';
    END LOOP;
    
    -- Drop all views
    FOR r IN (SELECT table_name FROM information_schema.views WHERE table_schema = 'public') LOOP
        RAISE NOTICE 'Dropping view: %', r.table_name;
        EXECUTE 'DROP VIEW IF EXISTS public.' || quote_ident(r.table_name) || ' CASCADE';
    END LOOP;
END $$;
`

	if dryRun {
		// Just list what would be dropped
		listSQL := `
SELECT 'TABLE' as type, tablename as name FROM pg_tables WHERE schemaname = 'public'
UNION ALL
SELECT 'SEQUENCE', sequence_name FROM information_schema.sequences WHERE sequence_schema = 'public'
UNION ALL
SELECT 'VIEW', table_name FROM information_schema.views WHERE table_schema = 'public'
ORDER BY type, name;
`
		cmd := exec.Command(GetPsqlPath(),
			"-h", config.Host,
			"-p", strconv.Itoa(config.Port),
			"-U", config.User,
			"-d", config.Database,
			"-c", listSQL,
		)
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%s", config.Password))

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to list database objects: %w\nOutput: %s", err, string(output))
		}

		logrus.Info("Objects that would be deleted:")
		logrus.Info(string(output))
		return nil
	}

	// Execute the drop SQL
	cmd := exec.Command(GetPsqlPath(),
		"-h", config.Host,
		"-p", strconv.Itoa(config.Port),
		"-U", config.User,
		"-d", config.Database,
		"-c", dropSQL,
	)
	cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%s", config.Password))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to purge database: %w\nOutput: %s", err, string(output))
	}

	logrus.Infof("Database '%s' purged successfully", config.Database)
	logrus.Debug(string(output))
	return nil
}

// GetPostgresVersion retrieves the PostgreSQL server version
func GetPostgresVersion(config *DatabaseConfig) (string, error) {
	if config == nil {
		return "", fmt.Errorf("database config is nil")
	}

	cmd := exec.Command(GetPsqlPath(),
		"-h", config.Host,
		"-p", strconv.Itoa(config.Port),
		"-U", config.User,
		"-d", config.Database,
		"-t", // Tuples only
		"-c", "SELECT version();",
	)
	cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%s", config.Password))

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get PostgreSQL version: %w", err)
	}

	version := strings.TrimSpace(string(output))
	logrus.Debugf("PostgreSQL version: %s", version)
	return version, nil
}
