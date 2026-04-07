package main

import "os"

// ServerConfig chứa cấu hình cho Server
type ServerConfig struct {
	Port   string
	DBPath string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() ServerConfig {
	return ServerConfig{
		Port:   getEnv("PORT", "8080"),
		DBPath: getEnv("DB_PATH", "metrics.db"),
	}
}

// getEnv gets environment variable or returns default value
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
