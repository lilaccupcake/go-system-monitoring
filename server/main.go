package main

import (
	"log"
	"net/http"
	"time"
)

func main() {
	// Load configuration
	config := LoadConfig()

	// Initialize database
	db, err := InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run initial cleanup to remove old data from previous runs
	if config.MetricsRetentionDays > 0 {
		log.Printf("Running initial cleanup for metrics older than %d days...", config.MetricsRetentionDays)
		if err := CleanupOldMetrics(db, config.MetricsRetentionDays*24); err != nil {
			log.Printf("Warning: initial cleanup failed: %v", err)
		}
	}

	// Start periodic cleanup (every 24 hours)
	if config.MetricsRetentionDays > 0 {
		go func() {
			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()
			for range ticker.C {
				log.Printf("Running scheduled cleanup for metrics older than %d days...", config.MetricsRetentionDays)
				if err := CleanupOldMetrics(db, config.MetricsRetentionDays*24); err != nil {
					log.Printf("Warning: scheduled cleanup failed: %v", err)
				}
			}
		}()
	}

	// Initialize WebSocket hub
	hub := NewWebSocketHub()
	go hub.Run()

	// Setup HTTP routes
	http.HandleFunc("/api/metrics", MetricsHandler(db, hub))
	http.HandleFunc("/api/history", HistoryHandler(db))
	http.HandleFunc("/health", HealthCheckHandler(db))
	http.HandleFunc("/ws", hub.WebSocketHandler)

	// Serve frontend static files
	http.Handle("/", http.FileServer(http.Dir(config.FrontendPath)))

	// Start server
	log.Printf("Server starting on port %s", config.Port)
	log.Printf("Database: %s", config.DBPath)
	log.Printf("Metrics retention: %d days", config.MetricsRetentionDays)
	log.Printf("WebSocket endpoint: ws://localhost:%s/ws", config.Port)
	log.Printf("API endpoints: http://localhost:%s/api/*", config.Port)

	if err := http.ListenAndServe(":"+config.Port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
