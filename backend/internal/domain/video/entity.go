package video

import (
	"time"

	"github.com/google/uuid"
)

type JobStatus string

const (
	JobStatusPending         JobStatus = "pending"
	JobStatusMediaCollecting JobStatus = "media_collecting"
	JobStatusTTSGenerating   JobStatus = "tts_generating"
	JobStatusAssembling      JobStatus = "assembling"
	JobStatusUploading       JobStatus = "uploading"
	JobStatusDone            JobStatus = "done"
	JobStatusFailed          JobStatus = "failed"
)

type TemplateType string

const (
	TemplateSlideshow   TemplateType = "slideshow"
	TemplateTextOnVideo TemplateType = "text_on_video"
	TemplateBRoll       TemplateType = "b_roll"
)

type Template struct {
	ID        uuid.UUID    `db:"id"`
	UserID    *uuid.UUID   `db:"user_id"`
	Name      string       `db:"name"`
	Type      TemplateType `db:"type"`
	Config    []byte       `db:"config"` // JSONB: FFmpeg params, fonts, colors, watermark
	IsDefault bool         `db:"is_default"`
	CreatedAt time.Time    `db:"created_at"`
	UpdatedAt time.Time    `db:"updated_at"`
}

type Job struct {
	ID             uuid.UUID  `db:"id"`
	ContentPlanID  uuid.UUID  `db:"content_plan_id"`
	TemplateID     *uuid.UUID `db:"template_id"`
	Status         JobStatus  `db:"status"`
	MediaAssets    []byte     `db:"media_assets"`  // JSONB
	TTSAudioKey    string     `db:"tts_audio_key"` // R2 object key
	SubtitleKey    string     `db:"subtitle_key"`
	OutputVideoKey string     `db:"output_video_key"`
	OutputVideoURL string     `db:"output_video_url"`
	DurationSeconds float64   `db:"duration_seconds"`
	FileSizeBytes  int64      `db:"file_size_bytes"`
	FFmpegLog      string     `db:"ffmpeg_log"`
	RetryCount     int        `db:"retry_count"`
	ErrorMessage   string     `db:"error_message"`
	StartedAt      *time.Time `db:"started_at"`
	CompletedAt    *time.Time `db:"completed_at"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}
