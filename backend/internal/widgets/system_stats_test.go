package widgets

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemStatsValidate(t *testing.T) {
	runConfigCases(t, "system_stats", []configCase{
		{"defaults to the root", `{}`, `{"disk_path":"/"}`, ""},
		{"cleaned", `{"disk_path":"/mnt/data/../data/"}`, `{"disk_path":"/mnt/data"}`, ""},
		{"relative", `{"disk_path":"data"}`, "", "absolute path"},
	})
}

// Reads the machine running the test, so only checks the values make sense
func TestSystemStatsFetch(t *testing.T) {
	w, _ := Get("system_stats")
	got, err := w.(Fetcher).Fetch(context.Background(), &FetchRequest{Config: json.RawMessage(`{"disk_path":"/"}`)})
	require.NoError(t, err)
	stats := got.(systemStatsPayload)
	assert.InDelta(t, 50, stats.CPUPercent, 50)
	assert.Positive(t, stats.MemoryTotal)
	assert.LessOrEqual(t, stats.MemoryUsed, stats.MemoryTotal)
	assert.Positive(t, stats.DiskTotal)
	assert.Positive(t, stats.UptimeSeconds)

	_, err = w.(Fetcher).Fetch(context.Background(), &FetchRequest{Config: json.RawMessage(`{"disk_path":"/does/not/exist"}`)})
	assert.EqualError(t, err, "could not read disk use of /does/not/exist")
}
