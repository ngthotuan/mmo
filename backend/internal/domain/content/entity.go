package content

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusDraft       Status = "draft"
	StatusApproved    Status = "approved"
	StatusRejected    Status = "rejected"
	StatusVideoQueued Status = "video_queued"
	StatusVideoReady  Status = "video_ready"
	StatusScheduled   Status = "scheduled"
	StatusPublished   Status = "published"
)

type TrendTopic struct {
	ID            uuid.UUID  `db:"id"`
	UserID        *uuid.UUID `db:"user_id"`
	Source        string     `db:"source"` // google_trends|youtube|reddit|tiktok
	Title         string     `db:"title"`
	Description   string     `db:"description"`
	Keywords      []string   `db:"keywords"`
	TrendingScore float64    `db:"trending_score"`
	SourceURL     string     `db:"source_url"`
	RawData       []byte     `db:"raw_data"`
	Status        string     `db:"status"` // new|used|rejected
	DiscoveredAt  time.Time  `db:"discovered_at"`
	CreatedAt     time.Time  `db:"created_at"`
}

type ScriptMetadata struct {
	Hook     string   `json:"hook"`
	CTA      string   `json:"cta"`
	Hashtags []string `json:"hashtags"`
	Caption  string   `json:"caption"`
}

type ContentPlan struct {
	ID                 uuid.UUID  `db:"id"`
	UserID             uuid.UUID  `db:"user_id"`
	TrendTopicID       *uuid.UUID `db:"trend_topic_id"`
	VideoTemplateID    *uuid.UUID `db:"video_template_id"`
	AutoPilotProfileID *uuid.UUID `db:"auto_pilot_profile_id"`
	Title              string     `db:"title"`
	Niche              string     `db:"niche"`
	TargetPlatforms    []string   `db:"target_platforms"`
	Script             string     `db:"script"`
	ScriptMetadata     []byte     `db:"script_metadata"` // JSONB
	Status             Status     `db:"status"`
	AutoApprove        bool       `db:"auto_approve"`
	Voice              string     `db:"voice"`
	Notes              string     `db:"notes"`
	CreatedAt          time.Time  `db:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at"`
}
