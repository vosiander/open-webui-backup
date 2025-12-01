package config

import (
	"os"
	"strconv"
)

// BackupToolVersion is the current version of the backup tool
const BackupToolVersion = "0.3.0"

// Config holds the application configuration
type Config struct {
	OpenWebUIURL    string
	OpenWebUIAPIKey string
	PostgresURL     string
	ServerPort      int
	BackupsDir      string
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		OpenWebUIURL:    getEnv("OPEN_WEBUI_URL", "https://example.com"),
		OpenWebUIAPIKey: getEnv("OPEN_WEBUI_API_KEY", ""),
		PostgresURL:     getEnv("POSTGRES_URL", ""),
		ServerPort:      getEnvInt("OWUI_SERVER_PORT", 3000),
		BackupsDir:      getEnv("OWUI_BACKUPS_DIR", "./backups"),
	}
}

// Update updates the configuration with new values
func (c *Config) Update(url, apiKey *string) {
	if url != nil && *url != "" {
		c.OpenWebUIURL = *url
	}
	if apiKey != nil {
		c.OpenWebUIAPIKey = *apiKey
	}
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt retrieves an integer environment variable or returns a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
