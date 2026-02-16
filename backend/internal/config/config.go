package config

import (
	"os"
)

// Config holds the application configuration
type Config struct {
	Port         string
	DatabasePath string
	JWTSecret    string
}

// Load reads configuration from environment variables or sets default values
func Load() *Config {
	return &Config{
		Port:         getEnv("PORT", "8080"),
		DatabasePath: getEnv("DATABASE_PATH", "./data/backend.db"),
		JWTSecret:    getEnv("JWT_SECRET", "default-secret-change-me"),
	}
}

// getEnv retrieves the value of the environment variable named by the key.
// It returns the value, which will be the default value if the variable is not present.
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
