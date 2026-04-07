package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func main() {
	// Khởi tạo configuration
	config := LoadConfig()

	// Khởi tạo collector
	collector := NewCollector()

	// Setup graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("Agent starting. Server: %s, Interval: %v", config.ServerURL, config.CollectInterval)
	log.Printf("Hostname: %s, OS: %s", collector.hostname, runtime.GOOS)

	// Tạo ticker cho collection loop
	ticker := time.NewTicker(config.CollectInterval)
	defer ticker.Stop()

	// Collection loop
	for {
		select {
		case <-ticker.C:
			if err := collectAndSend(collector, config.ServerURL); err != nil {
				log.Printf("Error collecting metrics: %v", err)
			}
		case <-stop:
			log.Println("Agent shutting down...")
			return
		}
	}
}

// collectAndSend thu thập và gửi metrics lên server
func collectAndSend(collector *Collector, serverURL string) error {
	// Thu thập metrics
	metric, err := collector.Collect()
	if err != nil {
		return err
	}

	// Marshal thành JSON
	jsonData, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	// Gửi HTTP POST
	resp, err := http.Post(serverURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send metric: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	log.Printf("Sent metrics - CPU: %.1f%%, RAM: %.1f%%", metric.CPUUsage, metric.RAMUsage)
	return nil
}
