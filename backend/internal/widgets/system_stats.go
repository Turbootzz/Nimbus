package widgets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

const maxDiskPathRunes = 200

type systemStatsConfig struct {
	// DiskPath is the mount whose usage is shown
	DiskPath string `json:"disk_path"`
}

type systemStatsPayload struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	MemoryUsed    uint64  `json:"memory_used"` // bytes
	MemoryTotal   uint64  `json:"memory_total"`
	DiskPercent   float64 `json:"disk_percent"`
	DiskUsed      uint64  `json:"disk_used"`
	DiskTotal     uint64  `json:"disk_total"`
	UptimeSeconds uint64  `json:"uptime_seconds"`
}

// systemStats shows the machine Nimbus runs on. Inside a container that is
// the container's view unless the host's /proc is mounted and HOST_PROC
// points at it (gopsutil reads HOST_PROC, HOST_SYS and HOST_ROOT).
type systemStats struct{}

func init() { Register(systemStats{}) }

func (systemStats) Type() string { return "system_stats" }

func (systemStats) Meta() Meta {
	return Meta{
		Name:                  "System stats",
		Category:              CategoryInfo,
		DefaultSize:           "2x1",
		AllowedSizes:          allSizes,
		DefaultRefreshSeconds: 30,
	}
}

func (systemStats) Validate(config json.RawMessage) (json.RawMessage, error) {
	cfg, err := decodeConfig[systemStatsConfig](config)
	if err != nil {
		return nil, err
	}
	if cfg.DiskPath == "" {
		cfg.DiskPath = "/"
	}
	if !filepath.IsAbs(cfg.DiskPath) || len([]rune(cfg.DiskPath)) > maxDiskPathRunes {
		return nil, fmt.Errorf("disk path must be an absolute path of at most %d characters", maxDiskPathRunes)
	}
	cfg.DiskPath = filepath.Clean(cfg.DiskPath)
	return encodeConfig(cfg)
}

func (systemStats) Fetch(ctx context.Context, req *FetchRequest) (any, error) {
	cfg, err := decodeConfig[systemStatsConfig](req.Config)
	if err != nil || cfg.DiskPath == "" {
		return nil, errors.New("widget has no disk path")
	}

	// CPU use is measured over one second
	cpuPercent, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err != nil || len(cpuPercent) == 0 {
		return nil, fmt.Errorf("could not read CPU use: %v", err)
	}
	memory, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not read memory use: %w", err)
	}
	usage, err := disk.UsageWithContext(ctx, cfg.DiskPath)
	if err != nil {
		return nil, fmt.Errorf("could not read disk use of %s", cfg.DiskPath)
	}
	uptime, err := host.UptimeWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not read uptime: %w", err)
	}

	return systemStatsPayload{
		CPUPercent:    oneDecimal(cpuPercent[0]),
		MemoryPercent: oneDecimal(memory.UsedPercent),
		MemoryUsed:    memory.Used,
		MemoryTotal:   memory.Total,
		DiskPercent:   oneDecimal(usage.UsedPercent),
		DiskUsed:      usage.Used,
		DiskTotal:     usage.Total,
		UptimeSeconds: uptime,
	}, nil
}

func oneDecimal(v float64) float64 {
	return math.Round(v*10) / 10
}
