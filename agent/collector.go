package main

import (
	"fmt"
	"os"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// Collector chịu trách nhiệm thu thập metrics
type Collector struct {
	hostname string
}

// NewCollector tạo collector mới
func NewCollector() *Collector {
	hostname, _ := os.Hostname()
	return &Collector{hostname: hostname}
}

// Collect thu thập tất cả metrics
func (c *Collector) Collect() (*SystemMetric, error) {
	// Lấy CPU usage
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU usage: %w", err)
	}

	// Lấy RAM info
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}

	metric := &SystemMetric{
		Hostname:  c.hostname,
		CPUUsage:  cpuPercent[0],
		RAMTotal:  memInfo.Total,
		RAMUsed:   memInfo.Used,
		RAMUsage:  memInfo.UsedPercent,
		Timestamp: time.Now(),
	}

	return metric, nil
}

// FormatBytes chuyển bytes sang định dạng dễ đọc
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
