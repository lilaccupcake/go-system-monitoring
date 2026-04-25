package main

import (
	"time"
)

// SystemMetric chứa thông tin metrics hệ thống tại một thời điểm
type SystemMetric struct {
	Hostname  string    `json:"hostname"`
	CPUUsage  float64   `json:"cpu_usage"`
	RAMTotal  uint64    `json:"ram_total"`
	RAMUsed   uint64    `json:"ram_used"`
	RAMUsage  float64   `json:"ram_usage_percent"`
	Timestamp time.Time `json:"timestamp"`
}

// AgentConfig chứa cấu hình cho Agent
type AgentConfig struct {
	ServerURL       string
	CollectInterval time.Duration
	MaxQueueSize    int
	MaxRetries      int
	InitialBackoff  time.Duration
	MaxBackoff      time.Duration
}
