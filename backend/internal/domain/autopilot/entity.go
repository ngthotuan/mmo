package autopilot

import (
	"time"

	"github.com/google/uuid"
)

// Profile is a user-defined "virtual channel" — a niche + voice + platforms + schedule
// that the auto-pilot worker uses to mass-produce content without manual review.
type Profile struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Name            string
	Niche           string
	Voice           string
	TargetPlatforms []string
	TrendFilter     string   // case-insensitive substring filter applied to trend titles/keywords
	TrendSources    []string // empty = any source; otherwise allowlist
	DailyCount      int
	ScheduleTimes   []string // HH:MM in 24h format, profile timezone = server local
	AutoApprove     bool
	AutoPublish     bool
	Enabled         bool
	LastRunAt       *time.Time
	LastRunCount    int
	TotalVideos     int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
