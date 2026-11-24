package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nimbus/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupStatusPageTestDB has the tables the status page queries join
func setupStatusPageTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := setupGroupTestDB(t)
	t.Cleanup(func() { db.Close() })

	_, err := db.Exec(`
		CREATE TABLE services (
			id TEXT PRIMARY KEY,
			integration_id TEXT,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			url TEXT NOT NULL,
			icon TEXT,
			icon_type TEXT DEFAULT 'emoji',
			icon_image_path TEXT DEFAULT '',
			description TEXT,
			status TEXT NOT NULL,
			response_time INTEGER,
			position INTEGER DEFAULT 0,
			card_size TEXT DEFAULT '2x1',
			group_id TEXT,
			monitoring_enabled INTEGER NOT NULL DEFAULT 1,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);
		CREATE TABLE service_status_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			service_id TEXT NOT NULL,
			status TEXT NOT NULL,
			response_time INTEGER,
			error_message TEXT,
			checked_at TIMESTAMP NOT NULL
		);
	`)
	require.NoError(t, err)
	return db
}

// seedStatusPageService adds a service of user-1; groupID "" leaves it ungrouped
func seedStatusPageService(t *testing.T, db *sql.DB, id, groupID string, position int, monitored bool) {
	t.Helper()
	var group *string
	if groupID != "" {
		group = &groupID
	}
	createServiceDirectly(t, db, &models.Service{
		ID: id, UserID: "user-1", Name: id, URL: "http://" + id + ".internal", Status: models.StatusOnline,
		Position: position, GroupID: group, MonitoringEnabled: monitored,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
}

// seedTheirUngroupedService adds a monitored ungrouped service of user-2,
// which no group of user-1 may show
func seedTheirUngroupedService(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	createServiceDirectly(t, db, &models.Service{
		ID: id, UserID: "user-2", Name: id, URL: "http://" + id + ".internal", Status: models.StatusOnline,
		MonitoringEnabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
}

func seedStatusLog(t *testing.T, db *sql.DB, serviceID, status string, at time.Time) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO service_status_logs (service_id, status, checked_at) VALUES (?, ?, ?)`, serviceID, status, at)
	require.NoError(t, err)
}

func TestStatusLogRepository_GetHourlyUptimeByGroup(t *testing.T) {
	db := setupStatusPageTestDB(t)
	repo := NewStatusLogRepository(db)

	seedStatusPageService(t, db, "svc-a", "group-1", 0, true)
	seedStatusPageService(t, db, "svc-b", "group-1", 1, true)
	seedStatusPageService(t, db, "svc-other", "group-2", 0, true)
	seedStatusPageService(t, db, "svc-ungrouped", "", 0, true)
	seedTheirUngroupedService(t, db, "svc-theirs")

	hour0 := time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC)
	from, recentFrom := hour0.Add(30*time.Minute), hour0.Add(5*time.Hour+15*time.Minute)
	// svc-a: two checks in hour 0 after from, two in hour 5 of which one is recent
	seedStatusLog(t, db, "svc-a", models.StatusOnline, hour0.Add(31*time.Minute))
	seedStatusLog(t, db, "svc-a", models.StatusOffline, hour0.Add(59*time.Minute))
	seedStatusLog(t, db, "svc-a", models.StatusOffline, hour0.Add(5*time.Hour+10*time.Minute))
	seedStatusLog(t, db, "svc-a", models.StatusOnline, hour0.Add(5*time.Hour+30*time.Minute))
	// Before from (same clock hour), another group and ungrouped: all left out
	seedStatusLog(t, db, "svc-a", models.StatusOffline, hour0.Add(29*time.Minute))
	seedStatusLog(t, db, "svc-other", models.StatusOffline, hour0.Add(time.Hour))
	seedStatusLog(t, db, "svc-ungrouped", models.StatusOffline, hour0.Add(time.Hour))
	seedStatusLog(t, db, "svc-theirs", models.StatusOffline, hour0.Add(time.Hour))
	// svc-b: a check stored in another time zone lands in the same hour
	seedStatusLog(t, db, "svc-b", models.StatusOnline, hour0.Add(2*time.Hour+time.Minute).In(time.FixedZone("CET", 3600)))

	buckets, err := repo.GetHourlyUptimeByGroup(context.Background(), &models.Group{ID: "group-1", UserID: "user-1"}, from, recentFrom)
	require.NoError(t, err)

	got := map[string]models.UptimeBucket{}
	for _, b := range buckets {
		got[b.ServiceID+"@"+time.Duration(b.Hour*int(time.Hour)).String()] = b
	}
	assert.Len(t, got, 3)
	assert.Equal(t, models.UptimeBucket{ServiceID: "svc-a", Hour: 0, Total: 2, Online: 1}, got["svc-a@0s"])
	assert.Equal(t, models.UptimeBucket{ServiceID: "svc-a", Hour: 5, Total: 2, Online: 1, RecentTotal: 1, RecentOnline: 1}, got["svc-a@5h0m0s"])
	assert.Equal(t, models.UptimeBucket{ServiceID: "svc-b", Hour: 2, Total: 1, Online: 1}, got["svc-b@2h0m0s"])

	// A default group also counts its owner's ungrouped services, and only theirs
	buckets, err = repo.GetHourlyUptimeByGroup(context.Background(), &models.Group{ID: "group-1", UserID: "user-1", IsDefault: true}, from, recentFrom)
	require.NoError(t, err)
	assert.Len(t, buckets, 4)
	for _, b := range buckets {
		assert.NotEqual(t, "svc-theirs", b.ServiceID)
	}
}

func TestStatusLogRepository_GetHourlyUptimeByGroup_Empty(t *testing.T) {
	db := setupStatusPageTestDB(t)
	repo := NewStatusLogRepository(db)

	buckets, err := repo.GetHourlyUptimeByGroup(context.Background(), &models.Group{ID: "group-1"}, time.Now().Add(-time.Hour), time.Now())
	require.NoError(t, err)
	assert.Empty(t, buckets)
}

// The exact 24 hour and 7 day cutoffs hold for a default group's ungrouped
// services too, and another user's ungrouped checks stay out
func TestStatusLogRepository_GetHourlyUptimeByGroup_DefaultGroupWindows(t *testing.T) {
	db := setupStatusPageTestDB(t)
	repo := NewStatusLogRepository(db)

	seedStatusPageService(t, db, "svc-ungrouped", "", 0, true)
	seedTheirUngroupedService(t, db, "svc-theirs")

	now := time.Date(2026, 11, 24, 12, 34, 56, 0, time.UTC)
	from, recentFrom := now.Add(-7*24*time.Hour), now.Add(-24*time.Hour)
	seedStatusLog(t, db, "svc-ungrouped", models.StatusOffline, from.Add(-time.Second))
	seedStatusLog(t, db, "svc-ungrouped", models.StatusOnline, from.Add(time.Second))
	seedStatusLog(t, db, "svc-ungrouped", models.StatusOffline, recentFrom.Add(-time.Second))
	seedStatusLog(t, db, "svc-ungrouped", models.StatusOnline, recentFrom.Add(time.Second))
	seedStatusLog(t, db, "svc-ungrouped", models.StatusOnline, now.Add(-time.Second))
	seedStatusLog(t, db, "svc-theirs", models.StatusOffline, now.Add(-time.Second))

	buckets, err := repo.GetHourlyUptimeByGroup(context.Background(), &models.Group{ID: "group-1", UserID: "user-1", IsDefault: true}, from, recentFrom)
	require.NoError(t, err)
	assert.ElementsMatch(t, []models.UptimeBucket{
		// Hour 0 starts at 12:00, but the check at from minus a second is out
		{ServiceID: "svc-ungrouped", Hour: 0, Total: 1, Online: 1},
		// Both sides of the 24 hour cutoff share an hour; one of them is recent
		{ServiceID: "svc-ungrouped", Hour: 144, Total: 2, Online: 1, RecentTotal: 1, RecentOnline: 1},
		{ServiceID: "svc-ungrouped", Hour: 168, Total: 1, Online: 1, RecentTotal: 1, RecentOnline: 1},
	}, buckets)
}

func TestServiceRepository_GetMonitoredByGroup(t *testing.T) {
	db := setupStatusPageTestDB(t)
	repo := NewServiceRepository(db)

	now := time.Now()
	public := &models.Group{ID: "group-1", UserID: "user-1", Name: "Public", MonitoringEnabled: true, CreatedAt: now, UpdatedAt: now}
	paused := &models.Group{ID: "group-off", UserID: "user-1", Name: "Paused", MonitoringEnabled: false, CreatedAt: now, UpdatedAt: now}
	createGroupDirectly(t, db, public)
	createGroupDirectly(t, db, &models.Group{ID: "group-2", UserID: "user-1", Name: "Private", MonitoringEnabled: true, CreatedAt: now, UpdatedAt: now})
	createGroupDirectly(t, db, paused)

	seedStatusPageService(t, db, "second", "group-1", 2, true)
	seedStatusPageService(t, db, "first", "group-1", 1, true)
	seedStatusPageService(t, db, "unmonitored", "group-1", 3, false)
	seedStatusPageService(t, db, "elsewhere", "group-2", 0, true)
	seedStatusPageService(t, db, "paused", "group-off", 0, true)
	seedStatusPageService(t, db, "ungrouped", "", 0, true)
	seedStatusPageService(t, db, "ungrouped-off", "", 4, false)
	seedTheirUngroupedService(t, db, "theirs")

	services, err := repo.GetMonitoredByGroup(context.Background(), public)
	require.NoError(t, err)
	require.Len(t, services, 2, "ungrouped services belong to the default group only")
	assert.Equal(t, "first", services[0].ID)
	assert.Equal(t, "second", services[1].ID)

	none, err := repo.GetMonitoredByGroup(context.Background(), paused)
	require.NoError(t, err)
	assert.Empty(t, none)

	// The default group holds the ungrouped services too, as on the dashboard
	public.IsDefault = true
	services, err = repo.GetMonitoredByGroup(context.Background(), public)
	require.NoError(t, err)
	assert.Equal(t, []string{"ungrouped", "first", "second"}, serviceIDs(services))

	// Ungrouped services follow only their own flag, also in a paused default group
	paused.IsDefault = true
	services, err = repo.GetMonitoredByGroup(context.Background(), paused)
	require.NoError(t, err)
	assert.Equal(t, []string{"ungrouped"}, serviceIDs(services))
}

func serviceIDs(services []*models.Service) []string {
	ids := make([]string, len(services))
	for i, s := range services {
		ids[i] = s.ID
	}
	return ids
}
