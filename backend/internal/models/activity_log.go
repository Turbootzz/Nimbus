package models

import "time"

// UserActivityLog represents a log entry of user activity
type UserActivityLog struct {
	ID        string                 `json:"id" db:"id"`
	UserID    *string                `json:"user_id" db:"user_id"`       // User affected
	ActorID   *string                `json:"actor_id" db:"actor_id"`     // User who performed action
	Action    string                 `json:"action" db:"action"`         // e.g., "role_changed", "user_deleted"
	Details   map[string]interface{} `json:"details" db:"details"`       // JSON details
	IPAddress *string                `json:"ip_address" db:"ip_address"` // IP address of actor
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
}

// Activity action constants
const (
	ActionUserCreated     = "user_created"
	ActionUserDeleted     = "user_deleted"
	ActionRoleChanged     = "role_changed"
	ActionBulkRoleChanged = "bulk_role_changed"
	ActionBulkUserDeleted = "bulk_user_deleted"
	ActionPasswordChanged = "password_changed"
	ActionInvitationSent  = "invitation_sent"
	ActionInvitationUsed  = "invitation_used"
	ActionSettingChanged  = "setting_changed"
)
