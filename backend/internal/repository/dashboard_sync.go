package repository

import (
	"context"
	"database/sql"
	"fmt"
)

// DashboardChanges is what a dashboard sync writes: only the rows that are
// new, changed or gone
type DashboardChanges struct {
	// Create holds the new rows, the boards to create or update, and the
	// preferences to save (nil: none)
	Create DashboardRows
	// Update holds changed rows in full; they keep their ID
	Update DashboardRows
	// Credentials are encrypted credentials to store, by integration ID; a
	// nil value clears them. Other integrations keep what they have.
	Credentials map[string][]byte
	Delete      struct {
		Groups, Integrations, Services, Widgets []string
	}
}

// Empty reports whether there is nothing to write
func (c *DashboardChanges) Empty() bool {
	rows := func(r *DashboardRows) int {
		return len(r.Groups) + len(r.Integrations) + len(r.Services) + len(r.Widgets)
	}
	return rows(&c.Create)+rows(&c.Update)+len(c.Credentials)+len(c.Create.Boards) == 0 && c.Create.Preferences == nil &&
		len(c.Delete.Groups)+len(c.Delete.Integrations)+len(c.Delete.Services)+len(c.Delete.Widgets) == 0
}

// Sync writes changes for the user in one transaction: all of it, or
// nothing. Rows of other users are never touched; an update or delete of a
// row the user no longer has fails the sync.
func (r *DashboardRepository) Sync(ctx context.Context, userID string, changes *DashboardChanges) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}
	// exec runs a statement that must change exactly the one row it names
	exec := func(what, query string, args ...any) error {
		result, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to %s: %w", what, err)
		}
		if err := requireAffected(result, sql.ErrNoRows); err != nil {
			return fmt.Errorf("failed to %s: %w", what, err)
		}
		return nil
	}

	defaultBoard, err := defaultBoardID(ctx, tx, userID)
	if err != nil {
		return err
	}
	changes.Create.own(userID, defaultBoard)
	// New boards, and the ones whose name or shared flag the document changes
	if err := upsertBoards(ctx, tx, &changes.Create, true); err != nil {
		return err
	}
	// Groups and integrations first: tiles point at them
	for _, group := range changes.Create.Groups {
		if err := insertGroup(ctx, tx, group); err != nil {
			return err
		}
	}
	for _, group := range changes.Update.Groups {
		err := exec("update group", `
			UPDATE groups SET name = $1, color = $2, position = $3, is_default = $4, monitoring_enabled = $5,
				tile_style = $6, updated_at = $7
			WHERE id = $8 AND user_id = $9`,
			group.Name, group.Color, group.Position, group.IsDefault, group.MonitoringEnabled, group.TileStyle,
			group.UpdatedAt, group.ID, userID)
		if err != nil {
			return err
		}
	}
	for _, integration := range changes.Create.Integrations {
		if err := insertIntegration(ctx, tx, integration); err != nil {
			return err
		}
	}
	for _, integration := range changes.Update.Integrations {
		err := exec("update integration", `
			UPDATE integrations SET name = $1, base_url = $2, auth_type = $3, verify_tls = $4, options = $5,
				refresh_seconds = $6, last_test_at = $7, last_test_ok = $8, last_error = $9, updated_at = CURRENT_TIMESTAMP
			WHERE id = $10 AND user_id = $11`,
			integration.Name, integration.BaseURL, integration.AuthType, integration.VerifyTLS, optionsJSON(integration.Options),
			integration.RefreshSeconds, integration.LastTestAt, integration.LastTestOK, integration.LastError, integration.ID, userID)
		if err != nil {
			return err
		}
	}
	for id, credentials := range changes.Credentials {
		err := exec("store credentials", `UPDATE integrations SET credentials_enc = $1 WHERE id = $2 AND user_id = $3`,
			nullableBytes(credentials), id, userID)
		if err != nil {
			return err
		}
	}

	for _, service := range changes.Create.Services {
		if err := insertService(ctx, tx, service); err != nil {
			return err
		}
	}
	for _, service := range changes.Update.Services {
		err := exec("update service", `
			UPDATE services SET name = $1, url = $2, icon = $3, icon_type = $4, icon_image_path = $5, description = $6,
				position = $7, card_size = $8, group_id = $9, integration_id = $10, monitoring_enabled = $11, updated_at = $12
			WHERE id = $13 AND user_id = $14`,
			service.Name, service.URL, service.Icon, service.IconType, service.IconImagePath, service.Description,
			service.Position, service.CardSize, service.GroupID, service.IntegrationID, service.MonitoringEnabled,
			service.UpdatedAt, service.ID, userID)
		if err != nil {
			return err
		}
	}
	for _, widget := range changes.Create.Widgets {
		if err := insertWidget(ctx, tx, widget); err != nil {
			return err
		}
	}
	for _, widget := range changes.Update.Widgets {
		err := exec("update widget", `
			UPDATE widgets SET title = $1, group_id = $2, integration_id = $3, config = $4, card_size = $5, position = $6,
				refresh_seconds = $7, enabled = $8, updated_at = CURRENT_TIMESTAMP
			WHERE id = $9 AND user_id = $10`,
			widget.Title, widget.GroupID, widget.IntegrationID, string(widget.Config), widget.CardSize, widget.Position,
			widget.RefreshSeconds, widget.Enabled, widget.ID, userID)
		if err != nil {
			return err
		}
	}

	// Tiles first: they point at groups and integrations
	deletes := []struct {
		table string
		ids   []string
	}{
		{"widgets", changes.Delete.Widgets},
		{"services", changes.Delete.Services},
		{"integrations", changes.Delete.Integrations},
		{"groups", changes.Delete.Groups},
	}
	for _, del := range deletes {
		for _, id := range del.ids {
			if err := exec("delete from "+del.table, `DELETE FROM `+del.table+` WHERE id = $1 AND user_id = $2`, id, userID); err != nil {
				return err
			}
		}
	}

	if changes.Create.Preferences != nil {
		if err := upsertPreferences(ctx, tx, userID, changes.Create.Preferences); err != nil {
			return fmt.Errorf("failed to save preferences: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit dashboard: %w", err)
	}
	return nil
}
