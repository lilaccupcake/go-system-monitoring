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
	"sync"
	"syscall"
	"time"
)

// Agent quản lý việc thu thập và gửi metrics với retry logic
type Agent struct {
	config    AgentConfig
	collector *Collector
	queue     []*SystemMetric
	mu        sync.Mutex
	stop      chan struct{}
}

// NewAgent tạo agent mới
func NewAgent(config AgentConfig, collector *Collector) *Agent {
	return &Agent{
		config:    config,
		collector: collector,
		queue:     make([]*SystemMetric, 0, config.MaxQueueSize),
		stop:      make(chan struct{}),
	}
}

// Run khởi chạy agent
func (a *Agent) Run() {
	log.Printf("Agent starting. Server: %s, Interval: %v", a.config.ServerURL, a.config.CollectInterval)
	log.Printf("Hostname: %s, OS: %s", a.collector.hostname, runtime.GOOS)
	log.Printf("Retry config: MaxRetries=%d, InitialBackoff=%v, MaxBackoff=%v, MaxQueueSize=%d",
		a.config.MaxRetries, a.config.InitialBackoff, a.config.MaxBackoff, a.config.MaxQueueSize)

	// Khởi chạy worker để xử lý queue
	go a.worker()

	// Tạo ticker cho collection loop
	ticker := time.NewTicker(a.config.CollectInterval)
	defer ticker.Stop()

	// Collection loop
	for {
		select {
		case <-ticker.C:
			a.collectAndEnqueue()
		case <-a.stop:
			log.Println("Agent shutting down...")
			return
		case <-a.configDone():
			// Xử lý khi config reload (nếu có)
		}
	}
}

// configDone trả về channel đóng khi nhận signal tái cấu hình (placeholder)
func (a *Agent) configDone() <-chan struct{} {
	return a.stop
}

// collectAndEnqueue thu thập metrics và thêm vào queue
func (a *Agent) collectAndEnqueue() {
	metric, err := a.collector.Collect()
	if err != nil {
		log.Printf("Error collecting metrics: %v", err)
		return
	}

	a.mu.Lock()
	a.queue = append(a.queue, metric)

	// Nếu queue vượt quá max size, loại bỏ metrics cũ nhất
	if len(a.queue) > a.config.MaxQueueSize {
		removeCount := len(a.queue) - a.config.MaxQueueSize
		a.queue = a.queue[removeCount:]
		log.Printf("Queue full, dropped %d oldest metrics", removeCount)
	}
	a.mu.Unlock()

	log.Printf("Collected metrics - CPU: %.1f%%, RAM: %.1f%% (queue size: %d)",
		metric.CPUUsage, metric.RAMUsage, len(a.queue))
}

// worker xử lý queue với retry/backoff logic
func (a *Agent) worker() {
	for {
		// Lấy metric từ queue
		metric := a.dequeue()
		if metric == nil {
			time.Sleep(100 * time.Millisecond) // Không có gì, chờ chút
			continue
		}

		// Gửi với retry logic
		if err := a.sendWithRetry(metric); err != nil {
			log.Printf("Failed to send metric after retries: %v", err)
			// Metrics đã hết retry, bỏ đi (có thể log hoặc metrics)
		} else {
			log.Printf("Successfully sent metric - CPU: %.1f%%, RAM: %.1f%%",
				metric.CPUUsage, metric.RAMUsage)
		}
	}
}

// dequeue lấy metric cũ nhất từ queue
func (a *Agent) dequeue() *SystemMetric {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.queue) == 0 {
		return nil
	}

	metric := a.queue[0]
	a.queue = a.queue[1:]
	return metric
}

// sendWithRetry gửi metric với exponential backoff retry
func (a *Agent) sendWithRetry(metric *SystemMetric) error {
	var lastErr error

	for attempt := 0; attempt <= a.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Tính backoff với jitter
			backoff := a.calculateBackoff(attempt)
			jitter := time.Duration(float64(backoff) * (0.5 + 0.5*float64(time.Now().UnixNano()%1000)/1000.0))
			log.Printf("Retry attempt %d/%d after %v...", attempt, a.config.MaxRetries, jitter)
			time.Sleep(jitter)
		}

		if err := a.sendMetric(metric); err != nil {
			lastErr = err
			log.Printf("Send failed (attempt %d/%d): %v", attempt+1, a.config.MaxRetries+1, err)
			continue
		}

		return nil // Success
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// calculateBackoff tính exponential backoff duration
func (a *Agent) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: initial * 2^attempt
	backoff := a.config.InitialBackoff * (1 << uint(attempt))
	if backoff > a.config.MaxBackoff {
		backoff = a.config.MaxBackoff
	}
	return backoff
}

// sendMetric gửi một metric lên server
func (a *Agent) sendMetric(metric *SystemMetric) error {
	// Marshal thành JSON
	jsonData, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	// Gửi HTTP POST với timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Post(a.config.ServerURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send metric: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	return nil
}

// Stop dừng agent
func (a *Agent) Stop() {
	close(a.stop)
}

// Graceful shutdown handling
func runAgentWithGracefulShutdown(agent *Agent) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	agent.Stop()
}

func main() {
	// Khởi tạo configuration
	config := LoadConfig()

	// Khởi tạo collector
	collector := NewCollector()

	// Khởi tạo agent
	agent := NewAgent(config, collector)

	// Setup graceful shutdown
	go runAgentWithGracefulShutdown(agent)

	// Chạy agent
	agent.Run()
}
