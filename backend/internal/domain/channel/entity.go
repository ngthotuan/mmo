package channel

import (
	"time"

	"github.com/google/uuid"
)

type Platform string

const (
	PlatformTikTok   Platform = "tiktok"
	PlatformFacebook Platform = "facebook"
	PlatformYouTube  Platform = "youtube"
)

type Channel struct {
	ID             uuid.UUID  `db:"id"`
	UserID         uuid.UUID  `db:"user_id"`
	Platform       Platform   `db:"platform"`
	PlatformUserID string     `db:"platform_user_id"`
	Username       string     `db:"username"`
	DisplayName    string     `db:"display_name"`
	AvatarURL      string     `db:"avatar_url"`
	AccessToken    string     `db:"access_token"`  // encrypted
	RefreshToken   string     `db:"refresh_token"` // encrypted
	TokenExpiresAt *time.Time `db:"token_expires_at"`
	PageID         string     `db:"page_id"`
	IsActive       bool       `db:"is_active"`
	DryRun         bool       `db:"dry_run"` // when true, publishes are mocked (no real API call)
	Metadata       []byte     `db:"metadata"` // JSONB
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}
