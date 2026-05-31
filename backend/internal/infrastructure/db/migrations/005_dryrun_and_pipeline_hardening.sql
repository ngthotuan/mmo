-- +goose Up

-- Dry-run channels publish via a mock (no real social API call). Lets the full
-- auto-pilot loop run end-to-end without real OAuth.
ALTER TABLE channels ADD COLUMN IF NOT EXISTS dry_run BOOLEAN NOT NULL DEFAULT FALSE;

-- Exponential-backoff scheduling for publish auto-retry.
ALTER TABLE publish_jobs ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_publish_jobs_retry
    ON publish_jobs (next_retry_at)
    WHERE status = 'failed';

-- De-dupe discovered trends by (source, title) so batch INSERT ... ON CONFLICT
-- DO NOTHING is safe and we can drop the per-row existence checks. md5(title)
-- keeps the index small and is IMMUTABLE (a timestamp::date cast is not).
CREATE UNIQUE INDEX IF NOT EXISTS uq_trend_topics_source_title
    ON trend_topics (source, md5(title));

-- +goose Down
DROP INDEX IF EXISTS uq_trend_topics_source_title;
DROP INDEX IF EXISTS idx_publish_jobs_retry;
ALTER TABLE publish_jobs DROP COLUMN IF EXISTS next_retry_at;
ALTER TABLE channels DROP COLUMN IF EXISTS dry_run;
