package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nimbus/backend/internal/repository"

	"github.com/nimbus/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverallStatus(t *testing.T) {
	assert.Equal(t, models.StatusPageUnknown, overallStatus(0, 0))
	assert.Equal(t, models.StatusPageOperational, overallStatus(3, 0))
	assert.Equal(t, models.StatusPageDegraded, overallStatus(2, 1))
	assert.Equal(t, models.StatusPageOutage, overallStatus(0, 2))
}

func TestUptimeBars(t *testing.T) {
	hours := make([]uptimeCount, 12)
	hours[0] = uptimeCount{total: 4, online: 4}
	hours[1] = uptimeCount{total: 4, online: 2}
	hours[11] = uptimeCount{total: 3, online: 1}

	perHour := uptimeBars(hours, 1)
	require.Len(t, perHour, 12)
	assert.Equal(t, 100.0, *perHour[0])
	assert.Equal(t, 50.0, *perHour[1])
	assert.Nil(t, perHour[2], "an hour without checks has no value")
	assert.Equal(t, 33.33, *perHour[11])

	perSix := uptimeBars(hours, 6)
	require.Len(t, perSix, 2)
	assert.Equal(t, 75.0, *perSix[0], "bars weigh every check the same")
	assert.Equal(t, 33.33, *perSix[1])

	assert.Nil(t, sumUptime(nil).percentage())
}

func TestPublicStatusService_Whitelist(t *testing.T) {
	service := &models.Service{
		ID: "id", UserID: "user", Name: "Jellyfin", URL: "http://10.0.0.5:8096", Description: "secret",
		Icon: "🎬", IconType: models.IconTypeImageURL, IconImagePath: "http://10.0.0.5:8096/favicon.ico",
		Status: models.StatusOnline,
	}

	hidden := publicStatusService(service, nil, false)
	assert.Empty(t, hidden.URL)
	assert.Equal(t, models.IconTypeEmoji, hidden.IconType, "a remote icon would leak the address")
	assert.Empty(t, hidden.IconImagePath)
	assert.Equal(t, "🎬", hidden.Icon)
	assert.Nil(t, hidden.Uptime24h)
	assert.Len(t, hidden.Bars24h, 24)
	assert.Len(t, hidden.Bars7d, 28)

	shown := publicStatusService(service, nil, true)
	assert.Equal(t, service.URL, shown.URL)
	assert.Equal(t, models.IconTypeImageURL, shown.IconType)
	assert.Equal(t, service.IconImagePath, shown.IconImagePath)

	service.IconType, service.IconImagePath = models.IconTypeImageUpload, "abc.png"
	uploaded := publicStatusService(service, nil, false)
	assert.Equal(t, "abc.png", uploaded.IconImagePath, "uploaded icons are served by Nimbus itself")
}

func TestStatusPageService_CacheExpiresAfter30Seconds(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE system_settings (key TEXT PRIMARY KEY, value TEXT NOT NULL, updated_at TIMESTAMP, updated_by TEXT)`)
	require.NoError(t, err)

	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	service := NewStatusPageService(repository.NewSettingsRepository(db), nil, nil, nil, nil, t.TempDir())
	service.now = func() time.Time { return now }

	page, err := service.Public(context.Background())
	require.NoError(t, err)
	require.Nil(t, page, "off by default")

	// Without the table every rebuild fails, so an answer proves the cache
	_, err = db.Exec(`DROP TABLE system_settings`)
	require.NoError(t, err)

	now = now.Add(statusPageCacheTTL - time.Second)
	_, err = service.Public(context.Background())
	assert.NoError(t, err, "still served from the cache")

	now = now.Add(time.Second)
	_, err = service.Public(context.Background())
	require.Error(t, err, "rebuilt once the copy is 30 seconds old")

	// A failed build is kept for a moment, so a broken database gets one
	// attempt per period and not one per visitor
	_, err = db.Exec(`CREATE TABLE system_settings (key TEXT PRIMARY KEY, value TEXT NOT NULL, updated_at TIMESTAMP, updated_by TEXT)`)
	require.NoError(t, err)
	now = now.Add(statusPageErrorTTL - time.Second)
	_, err = service.Public(context.Background())
	assert.Error(t, err, "the failure is still cached")

	now = now.Add(time.Second)
	_, err = service.Public(context.Background())
	assert.NoError(t, err, "and retried after 5 seconds")
}

func TestPublicURL(t *testing.T) {
	assert.Equal(t, "https://example.com/a?b=c", publicURL("https://example.com/a?b=c"))
	assert.Equal(t, "http://10.0.0.5:8096", publicURL("http://user:secret@10.0.0.5:8096"))
	for _, hidden := range []string{"", "javascript://x/%0aalert(1)", "ssh://host", "http://", "::"} {
		assert.Empty(t, publicURL(hidden), hidden)
	}
}

func TestStatusPageService_UseGroupKeepsTheOtherSettings(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE system_settings (key TEXT PRIMARY KEY, value TEXT NOT NULL, updated_at TIMESTAMP, updated_by TEXT)`)
	require.NoError(t, err)
	settingsRepo := repository.NewSettingsRepository(db)
	require.NoError(t, settingsRepo.UpdateBatch(context.Background(), map[string]string{
		statusPageEnabledKey: "true", statusPageGroupKey: "old", statusPageTitleKey: "Home lab",
	}, nil))
	service := NewStatusPageService(settingsRepo, nil, nil, nil, nil, t.TempDir())
	service.hasCache = true

	require.NoError(t, service.UseGroup(context.Background(), "user", "new"))
	settings, err := service.Settings(context.Background())
	require.NoError(t, err)
	assert.Equal(t, models.StatusPageSettings{Enabled: true, GroupID: "new", Title: "Home lab"}, settings)
	assert.False(t, service.hasCache, "the cached page is dropped")
}
