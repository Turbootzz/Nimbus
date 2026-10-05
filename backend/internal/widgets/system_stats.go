package widgets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

const maxDiskPathRunes = 200

// cpuSample is one CPU reading shared by all system stats widgets. A
// reading covers the time since the previous one, so two fetches right
// after each other would give the second one a few milliseconds (and 0%).
var cpuSample struct {
	sync.Mutex
	at      time.Time
	percent float64
}

// cpuPercent returns CPU use since the previous reading, reusing a reading
// younger than five seconds. It never waits for a sample.
func cpuPercent(ctx context.Context) (float64, error) {
	cpuSample.Lock()
	defer cpuSample.Unlock()
	if time.Since(cpuSample.at) < 5*time.Second {
		return cpuSample.percent, nil
	}
	percent, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return 0, fmt.Errorf("could not read CPU use: %w", err)
	}
	if len(percent) == 0 {
		return 0, errors.New("could not read CPU use")
	}
	cpuSample.at, cpuSample.percent = time.Now(), percent[0]
	return percent[0], nil
}

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

// systemStats shows the machine Nimbus runs on. Docker doesn't isolate CPU,
// memory and uptime in /proc, so in a container those are the host's; the
// disk is whatever the path is on. Admin only: it reads any path given.
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
		AdminOnly:             true,
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

	cpuUse, err := cpuPercent(ctx)
	if err != nil {
		return nil, err
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
		CPUPercent:    oneDecimal(cpuUse),
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
