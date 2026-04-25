package main

import (
	"fmt"
	"os"
	"time"
)

// LoadConfig loads configuration from environment variables
func LoadConfig() AgentConfig {
	return AgentConfig{
		ServerURL:       getEnv("SERVER_URL", "http://localhost:8080/api/metrics"),
		CollectInterval: getDurationEnv("COLLECT_INTERVAL", 3*time.Second),
		MaxQueueSize:    getIntEnv("MAX_QUEUE_SIZE", 1000),
		MaxRetries:      getIntEnv("MAX_RETRIES", 5),
		InitialBackoff:  getDurationEnv("INITIAL_BACKOFF", 1*time.Second),
		MaxBackoff:      getDurationEnv("MAX_BACKOFF", 30*time.Second),
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
