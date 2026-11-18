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

func seedStatusPageService(t *testing.T, db *sql.DB, id, groupID string, position int, monitored bool) {
	t.Helper()
	createServiceDirectly(t, db, &models.Service{
		ID: id, UserID: "user-1", Name: id, URL: "http://" + id + ".internal", Status: models.StatusOnline,
		Position: position, GroupID: &groupID, MonitoringEnabled: monitored,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
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

	since := time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC)
	// svc-a: two checks in hour 0, one in hour 5
	seedStatusLog(t, db, "svc-a", models.StatusOnline, since.Add(1*time.Minute))
	seedStatusLog(t, db, "svc-a", models.StatusOffline, since.Add(59*time.Minute))
	seedStatusLog(t, db, "svc-a", models.StatusOnline, since.Add(5*time.Hour+30*time.Minute))
	// Before the range, and another group: both left out
	seedStatusLog(t, db, "svc-a", models.StatusOffline, since.Add(-time.Minute))
	seedStatusLog(t, db, "svc-other", models.StatusOffline, since.Add(time.Minute))
	// svc-b: a check stored in another time zone lands in the same hour
	seedStatusLog(t, db, "svc-b", models.StatusOnline, since.Add(2*time.Hour+time.Minute).In(time.FixedZone("CET", 3600)))

	buckets, err := repo.GetHourlyUptimeByGroup(context.Background(), "group-1", since)
	require.NoError(t, err)

	got := map[string]models.UptimeBucket{}
	for _, b := range buckets {
		got[b.ServiceID+"@"+time.Duration(b.Hour*int(time.Hour)).String()] = b
	}
	assert.Len(t, got, 3)
	assert.Equal(t, models.UptimeBucket{ServiceID: "svc-a", Hour: 0, Total: 2, Online: 1}, got["svc-a@0s"])
	assert.Equal(t, models.UptimeBucket{ServiceID: "svc-a", Hour: 5, Total: 1, Online: 1}, got["svc-a@5h0m0s"])
	assert.Equal(t, models.UptimeBucket{ServiceID: "svc-b", Hour: 2, Total: 1, Online: 1}, got["svc-b@2h0m0s"])
}

func TestStatusLogRepository_GetHourlyUptimeByGroup_Empty(t *testing.T) {
	db := setupStatusPageTestDB(t)
	repo := NewStatusLogRepository(db)

	buckets, err := repo.GetHourlyUptimeByGroup(context.Background(), "group-1", time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.Empty(t, buckets)
}

func TestServiceRepository_GetMonitoredByGroupID(t *testing.T) {
	db := setupStatusPageTestDB(t)
	repo := NewServiceRepository(db)

	now := time.Now()
	createGroupDirectly(t, db, &models.Group{ID: "group-1", UserID: "user-1", Name: "Public", MonitoringEnabled: true, CreatedAt: now, UpdatedAt: now})
	createGroupDirectly(t, db, &models.Group{ID: "group-2", UserID: "user-1", Name: "Private", MonitoringEnabled: true, CreatedAt: now, UpdatedAt: now})
	createGroupDirectly(t, db, &models.Group{ID: "group-off", UserID: "user-1", Name: "Paused", MonitoringEnabled: false, CreatedAt: now, UpdatedAt: now})

	seedStatusPageService(t, db, "second", "group-1", 2, true)
	seedStatusPageService(t, db, "first", "group-1", 1, true)
	seedStatusPageService(t, db, "unmonitored", "group-1", 3, false)
	seedStatusPageService(t, db, "elsewhere", "group-2", 0, true)
	seedStatusPageService(t, db, "paused", "group-off", 0, true)

	services, err := repo.GetMonitoredByGroupID(context.Background(), "group-1")
	require.NoError(t, err)
	require.Len(t, services, 2)
	assert.Equal(t, "first", services[0].ID)
	assert.Equal(t, "second", services[1].ID)

	paused, err := repo.GetMonitoredByGroupID(context.Background(), "group-off")
	require.NoError(t, err)
	assert.Empty(t, paused)
}
