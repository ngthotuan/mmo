-- +goose Up
-- Allow 'youtube' as a publishing platform. The platform columns are plain
-- VARCHAR today (no CHECK constraint), so these statements add an explicit
-- whitelist that includes youtube. DROP IF EXISTS keeps it idempotent.
ALTER TABLE channels       DROP CONSTRAINT IF EXISTS channels_platform_check;
ALTER TABLE channels       ADD  CONSTRAINT channels_platform_check
    CHECK (platform IN ('tiktok','facebook','youtube'));

ALTER TABLE publish_jobs   DROP CONSTRAINT IF EXISTS publish_jobs_platform_check;
ALTER TABLE publish_jobs   ADD  CONSTRAINT publish_jobs_platform_check
    CHECK (platform IN ('tiktok','facebook','youtube'));

ALTER TABLE post_analytics DROP CONSTRAINT IF EXISTS post_analytics_platform_check;
ALTER TABLE post_analytics ADD  CONSTRAINT post_analytics_platform_check
    CHECK (platform IN ('tiktok','facebook','youtube'));

-- +goose Down
ALTER TABLE channels       DROP CONSTRAINT IF EXISTS channels_platform_check;
ALTER TABLE publish_jobs   DROP CONSTRAINT IF EXISTS publish_jobs_platform_check;
ALTER TABLE post_analytics DROP CONSTRAINT IF EXISTS post_analytics_platform_check;
