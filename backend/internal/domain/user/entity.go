package user

import (
	"time"

	"github.com/google/uuid"
)

// Roles control what a user may do.
//
//	admin  — full access to all features + user management
//	member — full access to all features (granted by an admin)
//	viewer — read-only; the default for new sign-ups until an admin grants access
const (
	RoleAdmin  = "admin"
	RoleMember = "member"
	RoleViewer = "viewer"
)

// HasFullAccess reports whether a role may use mutating (write) endpoints.
func HasFullAccess(role string) bool {
	return role == RoleAdmin || role == RoleMember
}

// IsAssignableRole reports whether a role may be assigned by an admin.
func IsAssignableRole(role string) bool {
	return role == RoleAdmin || role == RoleMember || role == RoleViewer
}

type User struct {
	ID        uuid.UUID `db:"id"         json:"id"`
	Email     string    `db:"email"      json:"email"`
	Name      string    `db:"name"       json:"name"`
	Role      string    `db:"role"       json:"role"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
