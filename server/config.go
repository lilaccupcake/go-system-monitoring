package main

import (
	"fmt"
	"os"
)

// ServerConfig chứa cấu hình cho Server
type ServerConfig struct {
	Port                 string
	DBPath               string
	MetricsRetentionDays int
	FrontendPath         string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() ServerConfig {
	return ServerConfig{
		Port:                 getEnv("PORT", "8080"),
		DBPath:               getEnv("DB_PATH", "metrics.db"),
		MetricsRetentionDays: getIntEnv("METRICS_RETENTION_DAYS", 30),
		FrontendPath:         getEnv("FRONTEND_PATH", "frontend"),
	}
}

// getIntEnv gets integer from env or default value
func getIntEnv(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		var i int
		if _, err := fmt.Sscanf(val, "%d", &i); err == nil {
			return i
		}
	}
	return defaultVal
}

// getEnv gets environment variable or returns default value
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
