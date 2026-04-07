package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/glebarez/sqlite"
)

const dbPath = "metrics.db"

// InitDB khởi tạo kết nối database và tạo bảng nếu chưa tồn tại
func InitDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Tạo bảng metrics
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hostname TEXT NOT NULL,
		cpu_usage REAL NOT NULL,
		ram_total INTEGER NOT NULL,
		ram_used INTEGER NOT NULL,
		ram_usage_percent REAL NOT NULL,
		timestamp DATETIME NOT NULL
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	// Tạo index để tối ưu query theo thời gian
	indexSQL := `CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp);`
	db.Exec(indexSQL)

	log.Println("Database initialized successfully")
	return db, nil
}

// InsertMetric lưu metric mới vào database
func InsertMetric(db *sql.DB, metric SystemMetric) error {
	insertSQL := `
	INSERT INTO metrics (hostname, cpu_usage, ram_total, ram_used, ram_usage_percent, timestamp)
	VALUES (?, ?, ?, ?, ?, ?);`

	_, err := db.Exec(insertSQL,
		metric.Hostname,
		metric.CPUUsage,
		metric.RAMTotal,
		metric.RAMUsed,
		metric.RAMUsage,
		metric.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to insert metric: %w", err)
	}

	return nil
}

// GetRecentMetrics lấy N metrics gần nhất
func GetRecentMetrics(db *sql.DB, limit int) ([]SystemMetric, error) {
	query := `
	SELECT hostname, cpu_usage, ram_total, ram_used, ram_usage_percent, timestamp
	FROM metrics
	ORDER BY timestamp DESC
	LIMIT ?;`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []SystemMetric
	for rows.Next() {
		var m SystemMetric
		err := rows.Scan(&m.Hostname, &m.CPUUsage, &m.RAMTotal, &m.RAMUsed, &m.RAMUsage, &m.Timestamp)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}

	// Reverse để có thứ tự thời gian tăng dần
	for i, j := 0, len(metrics)-1; i < j; i, j = i+1, j-1 {
		metrics[i], metrics[j] = metrics[j], metrics[i]
	}

	return metrics, nil
}

// CleanupOldMetrics xóa các metrics cũ hơn N giờ
func CleanupOldMetrics(db *sql.DB, hours int) error {
	cutoffTime := time.Now().Add(-time.Duration(hours) * time.Hour)
	deleteSQL := `DELETE FROM metrics WHERE timestamp < ?;`

	result, err := db.Exec(deleteSQL, cutoffTime)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("Cleaned up %d old metrics", rowsAffected)
	return nil
}
