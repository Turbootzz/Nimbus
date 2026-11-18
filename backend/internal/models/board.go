package models

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Board constants
const (
	DefaultBoardName = "Home"
	DefaultBoardSlug = "home"
	MaxBoardNameLen  = 50
	MaxBoardSlugLen  = 40
	MaxBoardsPerUser = 20
	// BoardVisibilityAll: every user sees the shared board
	BoardVisibilityAll = "all"
)

// Board is one dashboard of a user: its own groups, services and widgets.
// A shared board is shown to every user and edited by its owner only.
type Board struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Name       string    `json:"name"`
	Slug       string    `json:"slug"`
	Position   int       `json:"position"`
	IsDefault  bool      `json:"is_default"`
	IsShared   bool      `json:"is_shared"`
	Visibility string    `json:"visibility"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// BoardRequest is the body for create and update. On update, omitted fields
// keep their value. Without a slug, a new board gets one from its name.
type BoardRequest struct {
	Name     *string `json:"name"`
	Slug     *string `json:"slug"`
	IsShared *bool   `json:"is_shared"`
}

// BoardResponse is a board as one user sees it. It never names the owner.
type BoardResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Position   int    `json:"position"`
	IsDefault  bool   `json:"is_default"`
	IsShared   bool   `json:"is_shared"`
	Visibility string `json:"visibility"`
	// Owned is false for a board someone else shares; CanEdit follows it
	Owned   bool `json:"owned"`
	CanEdit bool `json:"can_edit"`
}

// ToResponse describes the board for the given viewer. Someone else's board
// is never their default board.
func (b *Board) ToResponse(viewerID string) BoardResponse {
	owned := b.UserID == viewerID
	return BoardResponse{
		ID:         b.ID,
		Name:       b.Name,
		Slug:       b.Slug,
		Position:   b.Position,
		IsDefault:  b.IsDefault && owned,
		IsShared:   b.IsShared,
		Visibility: b.Visibility,
		Owned:      owned,
		CanEdit:    owned,
	}
}

// BoardReorderRequest lists the user's own boards in their new order
type BoardReorderRequest struct {
	Boards []string `json:"boards"`
}

// BoardIntegration is what a viewer of a shared board learns about an
// integration its tiles use: enough to label the numbers, nothing to reach it
type BoardIntegration struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	// Options holds true for every option that is set, never its value
	Options map[string]bool `json:"options"`
}

var boardSlugRegex = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Slugs the frontend may want for pages under /board
var reservedBoardSlugs = map[string]bool{
	"new": true, "edit": true, "settings": true, "default": true, "shared": true, "all": true, "kiosk": true,
}

// IsValidBoardSlug checks a slug: lowercase letters, digits and single
// dashes, not reserved, and not shaped like an ID (a URL takes either)
func IsValidBoardSlug(slug string) bool {
	if len(slug) == 0 || len(slug) > MaxBoardSlugLen || !boardSlugRegex.MatchString(slug) || reservedBoardSlugs[slug] {
		return false
	}
	_, err := uuid.Parse(slug)
	return err != nil
}

// BoardSlug makes a slug from a name; "board" when nothing usable is left
func BoardSlug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			dash = false
			b.WriteRune(r)
		} else {
			dash = true
		}
		if b.Len() >= MaxBoardSlugLen-4 { // room for a "-99" suffix
			break
		}
	}
	slug := strings.Trim(b.String(), "-")
	if !IsValidBoardSlug(slug) {
		return "board"
	}
	return slug
}
