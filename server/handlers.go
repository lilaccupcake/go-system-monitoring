package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// MetricsHandler xử lý nhận metrics từ Agent
func MetricsHandler(db *sql.DB, hub *WebSocketHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse JSON từ request body
		var metric SystemMetric
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&metric); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Set timestamp nếu không có
		if metric.Timestamp.IsZero() {
			metric.Timestamp = time.Now()
		}

		// Lưu vào database
		if err := InsertMetric(db, metric); err != nil {
			log.Printf("Error inserting metric: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Broadcast đến tất cả WebSocket clients
		hub.Broadcast(metric)

		// Response thành công
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"message": "Metric received and stored",
		})
	}
}

// HistoryHandler trả về lịch sử metrics cho frontend
func HistoryHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Lấy số lượng metrics từ query param (default: 100)
		limit := 100
		if l := r.URL.Query().Get("limit"); l != "" {
			fmt.Sscanf(l, "%d", &limit)
		}

		metrics, err := GetRecentMetrics(db, limit)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metrics)
	}
}

// HealthCheckHandler trả về tình trạng server
func HealthCheckHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"database":  "connected",
		}

		// Check database
		if err := db.Ping(); err != nil {
			health["status"] = "unhealthy"
			health["database"] = "disconnected"
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(health)
	}
}
