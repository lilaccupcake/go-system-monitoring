package main

import (
	"os"
	"time"
)

// LoadConfig loads configuration from environment variables
func LoadConfig() AgentConfig {
	return AgentConfig{
		ServerURL:       getEnv("SERVER_URL", "http://localhost:8080/api/metrics"),
		CollectInterval: getDurationEnv("COLLECT_INTERVAL", 3*time.Second),
	}
}

// getEnv gets environment variable or returns default value
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// getDurationEnv gets duration from env or default
func getDurationEnv(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}
