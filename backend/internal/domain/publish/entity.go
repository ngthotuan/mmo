package publish

import (
	"time"

	"github.com/google/uuid"
)

type JobStatus string

const (
	JobStatusScheduled  JobStatus = "scheduled"
	JobStatusPublishing JobStatus = "publishing"
	JobStatusPublished  JobStatus = "published"
	JobStatusFailed     JobStatus = "failed"
	JobStatusCancelled  JobStatus = "cancelled"
)

type Job struct {
	ID              uuid.UUID  `db:"id"`
	VideoJobID      uuid.UUID  `db:"video_job_id"`
	ChannelID       uuid.UUID  `db:"channel_id"`
	ContentPlanID   *uuid.UUID `db:"content_plan_id"`
	Platform        string     `db:"platform"`
	Caption         string     `db:"caption"`
	Hashtags        []string   `db:"hashtags"`
	ScheduledAt     *time.Time `db:"scheduled_at"`
	PublishedAt     *time.Time `db:"published_at"`
	PlatformPostID  string     `db:"platform_post_id"`
	PlatformPostURL string     `db:"platform_post_url"`
	Status          JobStatus  `db:"status"`
	RetryCount      int        `db:"retry_count"`
	NextRetryAt     *time.Time `db:"next_retry_at"`
	ErrorMessage    string     `db:"error_message"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
}

type Analytics struct {
	ID               uuid.UUID `db:"id"`
	PublishJobID     uuid.UUID `db:"publish_job_id"`
	ChannelID        uuid.UUID `db:"channel_id"`
	Platform         string    `db:"platform"`
	SyncedAt         time.Time `db:"synced_at"`
	Views            int64     `db:"views"`
	Likes            int64     `db:"likes"`
	Comments         int64     `db:"comments"`
	Shares           int64     `db:"shares"`
	Saves            int64     `db:"saves"`
	Reach            int64     `db:"reach"`
	Impressions      int64     `db:"impressions"`
	PlayTimeSeconds  int64     `db:"play_time_seconds"`
	RawData          []byte    `db:"raw_data"`
}
