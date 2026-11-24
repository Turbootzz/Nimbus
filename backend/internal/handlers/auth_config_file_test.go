package handlers

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubConfigFile is a config file that manages one user's dashboard
type stubConfigFile struct {
	owner              string
	readOnly, lockPref bool
}

func (s stubConfigFile) Manages(userID string) bool          { return userID == s.owner }
func (s stubConfigFile) Locks(userID string) bool            { return s.readOnly && userID == s.owner }
func (s stubConfigFile) LocksPreferences(userID string) bool { return s.lockPref && userID == s.owner }

func TestGetMe_TellsTheOwnerAboutTheConfigFile(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-min-32-characters-long")
	db := setupAuthTestDB(t)
	defer db.Close()
	userRepo := repository.NewUserRepository(db)
	now := time.Now()
	for _, id := range []string{"owner-1", "user-2"} {
		createUserDirectly(t, db, &models.User{ID: id, Email: id + "@example.com", Name: id, Role: "admin", CreatedAt: now, UpdatedAt: now, LastActivityAt: &now})
	}
	handler := NewAuthHandler(userRepo, services.NewAuthService(), repository.NewSettingsRepository(db), repository.NewPreferencesRepository(db))

	me := func(userID string) map[string]any {
		app := fiber.New()
		app.Get("/me", func(c *fiber.Ctx) error {
			c.Locals("user_id", userID)
			return handler.GetMe(c)
		})
		status, _, body := transferRequest(t, app, http.MethodGet, "/me", "")
		require.Equal(t, http.StatusOK, status, string(body))
		var out map[string]any
		require.NoError(t, json.Unmarshal(body, &out))
		return out
	}

	assert.NotContains(t, me("owner-1"), "config_file", "no config file")

	handler.SetConfigFile(stubConfigFile{owner: "owner-1", readOnly: true})
	assert.Equal(t, map[string]any{"read_only": true, "preferences": false}, me("owner-1")["config_file"])
	assert.NotContains(t, me("user-2"), "config_file", "another user's dashboard is their own")

	handler.SetConfigFile(stubConfigFile{owner: "owner-1"})
	assert.Equal(t, map[string]any{"read_only": false, "preferences": false}, me("owner-1")["config_file"])
}
