package entity

// Roles a user can hold. Admins change settings and manage people, viewers
// only read.
const (
	RoleAdmin  = "admin"
	RoleViewer = "viewer"
)

// SessionTTL is how long a session lasts without use, in seconds. Every use
// slides it forward, so a phone that opens the dashboard monthly never signs out.
const SessionTTL = 30 * 24 * 3600

// Invite kinds: a join link creates a new user, a reset link sets a new
// password for an existing one.
const (
	InviteJoin  = "join"
	InviteReset = "reset"
)

// User is a person who can sign in to the dashboard.
type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	CreatedAt int64  `json:"created_at"`
}

// UserInfo is a user as listed on the access screen.
type UserInfo struct {
	User
	LastActive int64 `json:"last_active,omitempty"` // newest session use
	Sessions   int   `json:"sessions"`              // sessions that have not expired
}

// Session is one signed-in device.
type Session struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"-"`
	CreatedAt int64  `json:"created_at"`
	LastUsed  int64  `json:"last_used"`
	ExpiresAt int64  `json:"expires_at"`
	UserAgent string `json:"user_agent"`
	Current   bool   `json:"current"`
}

// Invite is a single-use link. Token is only set right after creation.
type Invite struct {
	ID        int64  `json:"id,omitempty"`
	Token     string `json:"token,omitempty"`
	Kind      string `json:"kind"`
	Role      string `json:"role,omitempty"`     // join only
	UserID    int64  `json:"-"`                  // reset only
	Username  string `json:"username,omitempty"` // reset only
	CreatedBy string `json:"created_by,omitempty"`
	CreatedAt int64  `json:"created_at,omitempty"`
	ExpiresAt int64  `json:"expires_at"`
}
