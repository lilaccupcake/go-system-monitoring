package main

import (
	"log"
	"net/http"
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

	// Initialize WebSocket hub
	hub := NewWebSocketHub()
	go hub.Run()

	// Setup HTTP routes
	http.HandleFunc("/api/metrics", MetricsHandler(db, hub))
	http.HandleFunc("/api/history", HistoryHandler(db))
	http.HandleFunc("/health", HealthCheckHandler(db))
	http.HandleFunc("/ws", hub.WebSocketHandler)

	// Serve frontend static files
	http.Handle("/", http.FileServer(http.Dir("../frontend")))

	// Start server
	log.Printf("Server starting on port %s", config.Port)
	log.Printf("Database: %s", config.DBPath)
	log.Printf("WebSocket endpoint: ws://localhost:%s/ws", config.Port)
	log.Printf("API endpoints: http://localhost:%s/api/*", config.Port)

	if err := http.ListenAndServe(":"+config.Port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
