package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nimbus/backend/internal/models"
)

// setupGroupTestDB creates an in-memory SQLite database for testing groups
func setupGroupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create groups table with SQLite-compatible schema
	schema := `
		CREATE TABLE IF NOT EXISTS groups (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			color TEXT DEFAULT '#6366f1',
			position INTEGER DEFAULT 0,
			is_default INTEGER DEFAULT 0,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_groups_user_id ON groups(user_id);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	return db
}

// createGroupDirectly inserts a group without using the repository's Create method
// This bypasses the RETURNING and FOR UPDATE clause issues in SQLite
func createGroupDirectly(t *testing.T, db *sql.DB, group *models.Group) {
	query := `
		INSERT INTO groups (id, user_id, name, color, position, is_default, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	isDefault := 0
	if group.IsDefault {
		isDefault = 1
	}

	_, err := db.Exec(
		query,
		group.ID,
		group.UserID,
		group.Name,
		group.Color,
		group.Position,
		isDefault,
		group.CreatedAt,
		group.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("Failed to create group directly: %v", err)
	}
}

func TestGroupRepository_GetByID(t *testing.T) {
	db := setupGroupTestDB(t)
	defer db.Close()

	repo := NewGroupRepository(db)
	ctx := context.Background()

	// Create test group
	testGroup := &models.Group{
		ID:        "group-1",
		UserID:    "user-1",
		Name:      "Test Group",
		Color:     "#ff0000",
		Position:  0,
		IsDefault: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	createGroupDirectly(t, db, testGroup)

	tests := []struct {
		name      string
		id        string
		wantErr   bool
		expectNil bool
	}{
		{
			name:      "Get existing group",
			id:        "group-1",
			wantErr:   false,
			expectNil: false,
		},
		{
			name:      "Get non-existent group",
			id:        "non-existent",
			wantErr:   true,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group, err := repo.GetByID(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.expectNil && group != nil {
				t.Errorf("GetByID() expected nil group, got %v", group)
			}
			if !tt.expectNil && group == nil {
				t.Error("GetByID() expected group, got nil")
			}
			if !tt.expectNil && group != nil {
				if group.ID != testGroup.ID {
					t.Errorf("GetByID() ID = %v, want %v", group.ID, testGroup.ID)
				}
				if group.Name != testGroup.Name {
					t.Errorf("GetByID() Name = %v, want %v", group.Name, testGroup.Name)
				}
				if group.Color != testGroup.Color {
					t.Errorf("GetByID() Color = %v, want %v", group.Color, testGroup.Color)
				}
			}
		})
	}
}

func TestGroupRepository_GetAllByUserID(t *testing.T) {
	db := setupGroupTestDB(t)
	defer db.Close()

	repo := NewGroupRepository(db)
	ctx := context.Background()

	// Create test groups for different users
	groups := []*models.Group{
		{
			ID:        "group-1",
			UserID:    "user-1",
			Name:      "User 1 Group 1",
			Color:     "#ff0000",
			Position:  0,
			IsDefault: true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "group-2",
			UserID:    "user-1",
			Name:      "User 1 Group 2",
			Color:     "#00ff00",
			Position:  1,
			IsDefault: false,
			CreatedAt: time.Now().Add(1 * time.Second),
			UpdatedAt: time.Now().Add(1 * time.Second),
		},
		{
			ID:        "group-3",
			UserID:    "user-2",
			Name:      "User 2 Group 1",
			Color:     "#0000ff",
			Position:  0,
			IsDefault: true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, g := range groups {
		createGroupDirectly(t, db, g)
	}

	tests := []struct {
		name          string
		userID        string
		expectedCount int
	}{
		{
			name:          "Get groups for user with 2 groups",
			userID:        "user-1",
			expectedCount: 2,
		},
		{
			name:          "Get groups for user with 1 group",
			userID:        "user-2",
			expectedCount: 1,
		},
		{
			name:          "Get groups for user with no groups",
			userID:        "user-3",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.GetAllByUserID(ctx, tt.userID)
			if err != nil {
				t.Errorf("GetAllByUserID() error = %v", err)
				return
			}
			if len(result) != tt.expectedCount {
				t.Errorf("GetAllByUserID() returned %d groups, want %d", len(result), tt.expectedCount)
			}
			// Verify user isolation
			for _, g := range result {
				if g.UserID != tt.userID {
					t.Errorf("GetAllByUserID() returned group with UserID %v, want %v", g.UserID, tt.userID)
				}
			}
		})
	}
}

func TestGroupRepository_GetAllByUserID_OrderedByPosition(t *testing.T) {
	db := setupGroupTestDB(t)
	defer db.Close()

	repo := NewGroupRepository(db)
	ctx := context.Background()

	groups := []*models.Group{
		{
			ID:        "group-1",
			UserID:    "user-1",
			Name:      "Third",
			Color:     "#ff0000",
			Position:  2,
			IsDefault: false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "group-2",
			UserID:    "user-1",
			Name:      "First",
			Color:     "#00ff00",
			Position:  0,
			IsDefault: true,
			CreatedAt: time.Now().Add(1 * time.Second),
			UpdatedAt: time.Now().Add(1 * time.Second),
		},
		{
			ID:        "group-3",
			UserID:    "user-1",
			Name:      "Second",
			Color:     "#0000ff",
			Position:  1,
			IsDefault: false,
			CreatedAt: time.Now().Add(2 * time.Second),
			UpdatedAt: time.Now().Add(2 * time.Second),
		},
	}

	for _, g := range groups {
		createGroupDirectly(t, db, g)
	}

	result, err := repo.GetAllByUserID(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetAllByUserID() error = %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("GetAllByUserID() returned %d groups, want 3", len(result))
	}

	expectedOrder := []string{"First", "Second", "Third"}
	for i, expected := range expectedOrder {
		if result[i].Name != expected {
			t.Errorf("GetAllByUserID() group[%d] name = %v, want %v", i, result[i].Name, expected)
		}
	}

	for i := 0; i < len(result)-1; i++ {
		if result[i].Position > result[i+1].Position {
			t.Errorf("GetAllByUserID() not ordered by position: %d > %d", result[i].Position, result[i+1].Position)
		}
	}
}

func TestGroupRepository_GetDefaultByUserID(t *testing.T) {
	db := setupGroupTestDB(t)
	defer db.Close()

	repo := NewGroupRepository(db)
	ctx := context.Background()

	// Create groups - one default, one not
	groups := []*models.Group{
		{
			ID:        "group-1",
			UserID:    "user-1",
			Name:      "Default Group",
			Color:     "#6366f1",
			Position:  0,
			IsDefault: true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "group-2",
			UserID:    "user-1",
			Name:      "Regular Group",
			Color:     "#ff0000",
			Position:  1,
			IsDefault: false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, g := range groups {
		createGroupDirectly(t, db, g)
	}

	tests := []struct {
		name    string
		userID  string
		wantErr bool
		wantID  string
	}{
		{
			name:    "Get default group for user with default",
			userID:  "user-1",
			wantErr: false,
			wantID:  "group-1",
		},
		{
			name:    "Get default group for user without default",
			userID:  "user-2",
			wantErr: true,
			wantID:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group, err := repo.GetDefaultByUserID(ctx, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDefaultByUserID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && group.ID != tt.wantID {
				t.Errorf("GetDefaultByUserID() ID = %v, want %v", group.ID, tt.wantID)
			}
			if !tt.wantErr && !group.IsDefault {
				t.Error("GetDefaultByUserID() returned non-default group")
			}
		})
	}
}

func TestGroupRepository_Update(t *testing.T) {
	db := setupGroupTestDB(t)
	defer db.Close()

	repo := NewGroupRepository(db)
	ctx := context.Background()

	// Create test group
	testGroup := &models.Group{
		ID:        "group-1",
		UserID:    "user-1",
		Name:      "Original Name",
		Color:     "#ff0000",
		Position:  0,
		IsDefault: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	createGroupDirectly(t, db, testGroup)

	tests := []struct {
		name    string
		group   *models.Group
		wantErr bool
	}{
		{
			name: "Update existing group",
			group: &models.Group{
				ID:        "group-1",
				UserID:    "user-1",
				Name:      "Updated Name",
				Color:     "#00ff00",
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "Update non-existent group",
			group: &models.Group{
				ID:        "non-existent",
				UserID:    "user-1",
				Name:      "Name",
				Color:     "#0000ff",
				UpdatedAt: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "Update with wrong user ID (user isolation check)",
			group: &models.Group{
				ID:        "group-1",
				UserID:    "wrong-user",
				Name:      "Hacked Name",
				Color:     "#000000",
				UpdatedAt: time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Update(ctx, tt.group)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Verify update was applied
	updated, err := repo.GetByID(ctx, "group-1")
	if err != nil {
		t.Fatalf("Failed to retrieve updated group: %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Errorf("Update() Name = %v, want %v", updated.Name, "Updated Name")
	}
	if updated.Color != "#00ff00" {
		t.Errorf("Update() Color = %v, want %v", updated.Color, "#00ff00")
	}
}

func TestGroupRepository_Delete(t *testing.T) {
	db := setupGroupTestDB(t)
	defer db.Close()

	repo := NewGroupRepository(db)
	ctx := context.Background()

	// Create test groups
	groups := []*models.Group{
		{
			ID:        "group-1",
			UserID:    "user-1",
			Name:      "Regular Group",
			Color:     "#ff0000",
			Position:  1,
			IsDefault: false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "group-2",
			UserID:    "user-1",
			Name:      "Default Group",
			Color:     "#6366f1",
			Position:  0,
			IsDefault: true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "group-3",
			UserID:    "user-2",
			Name:      "Other User Group",
			Color:     "#0000ff",
			Position:  0,
			IsDefault: false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, g := range groups {
		createGroupDirectly(t, db, g)
	}

	tests := []struct {
		name    string
		id      string
		userID  string
		wantErr bool
		errType error
	}{
		{
			name:    "Delete regular group with correct user",
			id:      "group-1",
			userID:  "user-1",
			wantErr: false,
		},
		{
			name:    "Delete default group (should fail)",
			id:      "group-2",
			userID:  "user-1",
			wantErr: true,
			errType: ErrCannotDeleteDefaultGroup,
		},
		{
			name:    "Delete group with wrong user (user isolation)",
			id:      "group-3",
			userID:  "user-1",
			wantErr: true,
		},
		{
			name:    "Delete non-existent group",
			id:      "non-existent",
			userID:  "user-1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Delete(ctx, tt.id, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.errType != nil && err != tt.errType {
				t.Errorf("Delete() error = %v, want %v", err, tt.errType)
			}
		})
	}

	// Verify group-1 was deleted
	_, err := repo.GetByID(ctx, "group-1")
	if err != sql.ErrNoRows {
		t.Error("Delete() did not delete the group")
	}

	// Verify group-2 (default) still exists
	group2, err := repo.GetByID(ctx, "group-2")
	if err != nil {
		t.Error("Delete() deleted default group that shouldn't have been deleted")
	}
	if group2 == nil {
		t.Error("Delete() group-2 (default) should still exist")
	}

	// Verify group-3 still exists (wrong user tried to delete)
	group3, err := repo.GetByID(ctx, "group-3")
	if err != nil {
		t.Error("Delete() deleted group that shouldn't have been deleted")
	}
	if group3 == nil {
		t.Error("Delete() group-3 should still exist")
	}
}

func TestGroupRepository_UpdatePositions(t *testing.T) {
	db := setupGroupTestDB(t)
	defer db.Close()

	repo := NewGroupRepository(db)
	ctx := context.Background()

	// Create test groups for multiple users
	groups := []*models.Group{
		{
			ID:        "group-1",
			UserID:    "user-1",
			Name:      "User 1 Group 1",
			Color:     "#ff0000",
			Position:  0,
			IsDefault: true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "group-2",
			UserID:    "user-1",
			Name:      "User 1 Group 2",
			Color:     "#00ff00",
			Position:  1,
			IsDefault: false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "group-3",
			UserID:    "user-1",
			Name:      "User 1 Group 3",
			Color:     "#0000ff",
			Position:  2,
			IsDefault: false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "group-4",
			UserID:    "user-2",
			Name:      "User 2 Group 1",
			Color:     "#ffff00",
			Position:  0,
			IsDefault: true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, g := range groups {
		createGroupDirectly(t, db, g)
	}

	tests := []struct {
		name      string
		userID    string
		positions map[string]int
		wantErr   bool
	}{
		{
			name:   "Reorder groups for user-1",
			userID: "user-1",
			positions: map[string]int{
				"group-1": 2,
				"group-2": 0,
				"group-3": 1,
			},
			wantErr: false,
		},
		{
			name:   "Update single group position",
			userID: "user-1",
			positions: map[string]int{
				"group-1": 5,
			},
			wantErr: false,
		},
		{
			name:   "Attempt to update another user's group (security check)",
			userID: "user-1",
			positions: map[string]int{
				"group-4": 10,
			},
			wantErr: true,
		},
		{
			name:   "Update non-existent group",
			userID: "user-1",
			positions: map[string]int{
				"non-existent": 0,
			},
			wantErr: true,
		},
		{
			name:      "Empty positions map",
			userID:    "user-1",
			positions: map[string]int{},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpdatePositions(ctx, tt.userID, tt.positions)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdatePositions() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && len(tt.positions) > 0 {
				for groupID, expectedPos := range tt.positions {
					group, err := repo.GetByID(ctx, groupID)
					if err != nil {
						t.Fatalf("Failed to retrieve group %s: %v", groupID, err)
					}
					if group.Position != expectedPos {
						t.Errorf("UpdatePositions() group %s position = %v, want %v", groupID, group.Position, expectedPos)
					}
				}
			}
		})
	}
}

func TestGroupRepository_UpdatePositions_Transaction(t *testing.T) {
	db := setupGroupTestDB(t)
	defer db.Close()

	repo := NewGroupRepository(db)
	ctx := context.Background()

	groups := []*models.Group{
		{
			ID:        "group-1",
			UserID:    "user-1",
			Name:      "Group 1",
			Color:     "#ff0000",
			Position:  0,
			IsDefault: true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "group-2",
			UserID:    "user-1",
			Name:      "Group 2",
			Color:     "#00ff00",
			Position:  1,
			IsDefault: false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, g := range groups {
		createGroupDirectly(t, db, g)
	}

	// Test transaction rollback
	positions := map[string]int{
		"group-1":      10,
		"non-existent": 20,
	}

	err := repo.UpdatePositions(ctx, "user-1", positions)
	if err == nil {
		t.Error("UpdatePositions() expected error for partial invalid update, got nil")
	}

	// Verify rollback
	group1, err := repo.GetByID(ctx, "group-1")
	if err != nil {
		t.Fatalf("Failed to retrieve group-1: %v", err)
	}
	if group1.Position != 0 {
		t.Errorf("UpdatePositions() transaction rollback failed: position = %v, want %v", group1.Position, 0)
	}
}

func TestIsValidHexColor(t *testing.T) {
	tests := []struct {
		name  string
		color string
		want  bool
	}{
		{
			name:  "Valid lowercase hex",
			color: "#ff0000",
			want:  true,
		},
		{
			name:  "Valid uppercase hex",
			color: "#FF0000",
			want:  true,
		},
		{
			name:  "Valid mixed case hex",
			color: "#Ff00aB",
			want:  true,
		},
		{
			name:  "Default group color",
			color: models.DefaultGroupColor,
			want:  true,
		},
		{
			name:  "Missing hash",
			color: "ff0000",
			want:  false,
		},
		{
			name:  "Too short",
			color: "#ff00",
			want:  false,
		},
		{
			name:  "Too long",
			color: "#ff00000",
			want:  false,
		},
		{
			name:  "Invalid characters",
			color: "#gggggg",
			want:  false,
		},
		{
			name:  "Empty string",
			color: "",
			want:  false,
		},
		{
			name:  "RGB format",
			color: "rgb(255,0,0)",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := models.IsValidHexColor(tt.color)
			if got != tt.want {
				t.Errorf("IsValidHexColor(%q) = %v, want %v", tt.color, got, tt.want)
			}
		})
	}
}

func TestGroup_ToResponse(t *testing.T) {
	now := time.Now()
	group := &models.Group{
		ID:        "group-1",
		UserID:    "user-1",
		Name:      "Test Group",
		Color:     "#ff0000",
		Position:  5,
		IsDefault: true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	response := group.ToResponse()

	if response.ID != group.ID {
		t.Errorf("ToResponse() ID = %v, want %v", response.ID, group.ID)
	}
	if response.Name != group.Name {
		t.Errorf("ToResponse() Name = %v, want %v", response.Name, group.Name)
	}
	if response.Color != group.Color {
		t.Errorf("ToResponse() Color = %v, want %v", response.Color, group.Color)
	}
	if response.Position != group.Position {
		t.Errorf("ToResponse() Position = %v, want %v", response.Position, group.Position)
	}
	if response.IsDefault != group.IsDefault {
		t.Errorf("ToResponse() IsDefault = %v, want %v", response.IsDefault, group.IsDefault)
	}
	if response.CreatedAt != group.CreatedAt {
		t.Errorf("ToResponse() CreatedAt = %v, want %v", response.CreatedAt, group.CreatedAt)
	}
	if response.UpdatedAt != group.UpdatedAt {
		t.Errorf("ToResponse() UpdatedAt = %v, want %v", response.UpdatedAt, group.UpdatedAt)
	}
}
