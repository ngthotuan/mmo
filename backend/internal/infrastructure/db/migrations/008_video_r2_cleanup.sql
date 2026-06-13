-- +goose Up
-- Track when a published video's R2 artifacts were deleted by the retention job,
-- so the cleanup cron is idempotent and we keep an audit trail.
ALTER TABLE video_jobs ADD COLUMN r2_deleted_at TIMESTAMPTZ;

-- Partial index: the cleanup query only scans rows that still have a file on R2.
CREATE INDEX idx_video_jobs_r2_cleanup ON video_jobs(id)
    WHERE r2_deleted_at IS NULL AND output_video_key IS NOT NULL AND output_video_key <> '';

-- +goose Down
DROP INDEX IF EXISTS idx_video_jobs_r2_cleanup;
ALTER TABLE video_jobs DROP COLUMN IF EXISTS r2_deleted_at;
