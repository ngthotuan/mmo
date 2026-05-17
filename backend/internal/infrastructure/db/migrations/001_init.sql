-- +goose Up

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ─── Users ────────────────────────────────────────────────────────────────────

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name          VARCHAR(255) NOT NULL,
    role          VARCHAR(50)  NOT NULL DEFAULT 'owner',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ─── Social accounts / channels ───────────────────────────────────────────────

CREATE TABLE channels (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform          VARCHAR(50) NOT NULL,        -- 'tiktok' | 'facebook'
    platform_user_id  VARCHAR(255) NOT NULL,
    username          VARCHAR(255),
    display_name      VARCHAR(255),
    avatar_url        TEXT,
    access_token      TEXT        NOT NULL,         -- AES-256-GCM encrypted
    refresh_token     TEXT,                         -- AES-256-GCM encrypted
    token_expires_at  TIMESTAMPTZ,
    page_id           VARCHAR(255),                 -- Facebook pages only
    is_active         BOOLEAN     NOT NULL DEFAULT TRUE,
    metadata          JSONB       NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (platform, platform_user_id)
);

CREATE INDEX idx_channels_user_id ON channels(user_id);
CREATE INDEX idx_channels_active ON channels(user_id, is_active);

-- ─── Trend topics ─────────────────────────────────────────────────────────────

CREATE TABLE trend_topics (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID        REFERENCES users(id) ON DELETE SET NULL,
    source         VARCHAR(50) NOT NULL,   -- 'google_trends'|'youtube'|'reddit'|'tiktok'
    title          VARCHAR(500) NOT NULL,
    description    TEXT,
    keywords       TEXT[]      NOT NULL DEFAULT '{}',
    trending_score FLOAT,
    source_url     TEXT,
    raw_data       JSONB       NOT NULL DEFAULT '{}',
    status         VARCHAR(50) NOT NULL DEFAULT 'new',  -- 'new'|'used'|'rejected'
    discovered_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trend_topics_user_status ON trend_topics(user_id, status);
CREATE INDEX idx_trend_topics_discovered  ON trend_topics(discovered_at DESC);

-- ─── Video templates ──────────────────────────────────────────────────────────

CREATE TABLE video_templates (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        REFERENCES users(id) ON DELETE CASCADE,
    name       VARCHAR(255) NOT NULL,
    type       VARCHAR(50) NOT NULL,   -- 'slideshow'|'text_on_video'|'b_roll'
    config     JSONB       NOT NULL DEFAULT '{}',
    is_default BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── Content plans ────────────────────────────────────────────────────────────

CREATE TABLE content_plans (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    trend_topic_id    UUID        REFERENCES trend_topics(id) ON DELETE SET NULL,
    video_template_id UUID        REFERENCES video_templates(id) ON DELETE SET NULL,
    title             VARCHAR(500) NOT NULL,
    niche             VARCHAR(100),
    target_platforms  VARCHAR(50)[] NOT NULL DEFAULT '{}',
    script            TEXT,
    script_metadata   JSONB       NOT NULL DEFAULT '{}',  -- {hook, cta, hashtags[], caption}
    status            VARCHAR(50) NOT NULL DEFAULT 'draft',
    -- 'draft'|'approved'|'rejected'|'video_queued'|'video_ready'|'scheduled'|'published'
    auto_approve      BOOLEAN     NOT NULL DEFAULT FALSE,
    notes             TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_content_plans_user_status ON content_plans(user_id, status);
CREATE INDEX idx_content_plans_created     ON content_plans(created_at DESC);

-- ─── Video jobs ───────────────────────────────────────────────────────────────

CREATE TABLE video_jobs (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    content_plan_id  UUID        NOT NULL REFERENCES content_plans(id) ON DELETE CASCADE,
    template_id      UUID        REFERENCES video_templates(id) ON DELETE SET NULL,
    status           VARCHAR(50) NOT NULL DEFAULT 'pending',
    -- 'pending'|'media_collecting'|'tts_generating'|'assembling'|'uploading'|'done'|'failed'
    media_assets     JSONB       NOT NULL DEFAULT '[]',   -- [{type,url,r2_key,duration}]
    tts_audio_key    TEXT,
    subtitle_key     TEXT,
    output_video_key TEXT,
    output_video_url TEXT,
    duration_seconds FLOAT,
    file_size_bytes  BIGINT,
    ffmpeg_log       TEXT,
    retry_count      INT         NOT NULL DEFAULT 0,
    error_message    TEXT,
    started_at       TIMESTAMPTZ,
    completed_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_video_jobs_status         ON video_jobs(status);
CREATE INDEX idx_video_jobs_content_plan   ON video_jobs(content_plan_id);

-- ─── Publish jobs ─────────────────────────────────────────────────────────────

CREATE TABLE publish_jobs (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    video_job_id      UUID        NOT NULL REFERENCES video_jobs(id) ON DELETE CASCADE,
    channel_id        UUID        NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    content_plan_id   UUID        REFERENCES content_plans(id) ON DELETE SET NULL,
    platform          VARCHAR(50) NOT NULL,
    caption           TEXT,
    hashtags          TEXT[]      NOT NULL DEFAULT '{}',
    scheduled_at      TIMESTAMPTZ,
    published_at      TIMESTAMPTZ,
    platform_post_id  VARCHAR(255),
    platform_post_url TEXT,
    status            VARCHAR(50) NOT NULL DEFAULT 'scheduled',
    -- 'scheduled'|'publishing'|'published'|'failed'|'cancelled'
    retry_count       INT         NOT NULL DEFAULT 0,
    error_message     TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_publish_jobs_scheduled ON publish_jobs(scheduled_at) WHERE status = 'scheduled';
CREATE INDEX idx_publish_jobs_channel   ON publish_jobs(channel_id);
CREATE INDEX idx_publish_jobs_video     ON publish_jobs(video_job_id);

-- ─── Post analytics ───────────────────────────────────────────────────────────

CREATE TABLE post_analytics (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    publish_job_id       UUID        NOT NULL REFERENCES publish_jobs(id) ON DELETE CASCADE,
    channel_id           UUID        NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    platform             VARCHAR(50) NOT NULL,
    synced_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    views                BIGINT      NOT NULL DEFAULT 0,
    likes                BIGINT      NOT NULL DEFAULT 0,
    comments             BIGINT      NOT NULL DEFAULT 0,
    shares               BIGINT      NOT NULL DEFAULT 0,
    saves                BIGINT      NOT NULL DEFAULT 0,
    reach                BIGINT      NOT NULL DEFAULT 0,
    impressions          BIGINT      NOT NULL DEFAULT 0,
    play_time_seconds    BIGINT      NOT NULL DEFAULT 0,
    raw_data             JSONB       NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_post_analytics_publish ON post_analytics(publish_job_id);
CREATE INDEX idx_post_analytics_synced  ON post_analytics(synced_at DESC);

-- ─── Media asset library ──────────────────────────────────────────────────────

CREATE TABLE media_assets (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID        REFERENCES users(id) ON DELETE SET NULL,
    source           VARCHAR(50),                   -- 'pexels'|'pixabay'|'upload'
    source_id        VARCHAR(255),
    type             VARCHAR(20) NOT NULL,           -- 'video'|'image'|'audio'
    r2_key           TEXT        NOT NULL,
    cdn_url          TEXT,
    width            INT,
    height           INT,
    duration_seconds FLOAT,
    file_size_bytes  BIGINT,
    tags             TEXT[]      NOT NULL DEFAULT '{}',
    license          VARCHAR(100),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source, source_id)
);

CREATE INDEX idx_media_assets_type ON media_assets(type);
CREATE INDEX idx_media_assets_tags ON media_assets USING GIN(tags);

-- ─── Job audit log ────────────────────────────────────────────────────────────

CREATE TABLE job_logs (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    job_type     VARCHAR(100) NOT NULL,
    reference_id UUID,
    status       VARCHAR(50),
    message      TEXT,
    metadata     JSONB       NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_job_logs_reference ON job_logs(reference_id);
CREATE INDEX idx_job_logs_created   ON job_logs(created_at DESC);

-- ─── Triggers: updated_at ─────────────────────────────────────────────────────

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_users_updated_at           BEFORE UPDATE ON users           FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_channels_updated_at        BEFORE UPDATE ON channels        FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_content_plans_updated_at   BEFORE UPDATE ON content_plans   FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_video_templates_updated_at BEFORE UPDATE ON video_templates FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_video_jobs_updated_at      BEFORE UPDATE ON video_jobs      FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_publish_jobs_updated_at    BEFORE UPDATE ON publish_jobs    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down

DROP TRIGGER IF EXISTS trg_publish_jobs_updated_at    ON publish_jobs;
DROP TRIGGER IF EXISTS trg_video_jobs_updated_at      ON video_jobs;
DROP TRIGGER IF EXISTS trg_video_templates_updated_at ON video_templates;
DROP TRIGGER IF EXISTS trg_content_plans_updated_at   ON content_plans;
DROP TRIGGER IF EXISTS trg_channels_updated_at        ON channels;
DROP TRIGGER IF EXISTS trg_users_updated_at           ON users;
DROP FUNCTION IF EXISTS set_updated_at();

DROP TABLE IF EXISTS job_logs;
DROP TABLE IF EXISTS media_assets;
DROP TABLE IF EXISTS post_analytics;
DROP TABLE IF EXISTS publish_jobs;
DROP TABLE IF EXISTS video_jobs;
DROP TABLE IF EXISTS content_plans;
DROP TABLE IF EXISTS video_templates;
DROP TABLE IF EXISTS trend_topics;
DROP TABLE IF EXISTS channels;
DROP TABLE IF EXISTS users;
