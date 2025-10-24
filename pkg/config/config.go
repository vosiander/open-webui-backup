package config

import (
	"os"
)

// BackupToolVersion is the current version of the backup tool
const BackupToolVersion = "0.3.0"

// Config holds the application configuration
type Config struct {
	OpenWebUIURL    string
	OpenWebUIAPIKey string
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		OpenWebUIURL:    getEnv("OPEN_WEBUI_URL", "https://example.com"),
		OpenWebUIAPIKey: getEnv("OPEN_WEBUI_API_KEY", ""),
	}
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
