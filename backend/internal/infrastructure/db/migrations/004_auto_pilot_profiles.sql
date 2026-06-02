-- +goose Up
CREATE TABLE auto_pilot_profiles (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name               TEXT         NOT NULL,
    niche              TEXT         NOT NULL DEFAULT 'general',
    voice              TEXT         NOT NULL DEFAULT '',
    target_platforms   TEXT[]       NOT NULL DEFAULT '{}',
    trend_filter       TEXT         NOT NULL DEFAULT '',
    trend_sources      TEXT[]       NOT NULL DEFAULT '{}',
    daily_count        INT          NOT NULL DEFAULT 2,
    schedule_times     TEXT[]       NOT NULL DEFAULT '{}',
    auto_approve       BOOLEAN      NOT NULL DEFAULT TRUE,
    auto_publish       BOOLEAN      NOT NULL DEFAULT TRUE,
    enabled            BOOLEAN      NOT NULL DEFAULT TRUE,
    last_run_at        TIMESTAMPTZ,
    last_run_count     INT          NOT NULL DEFAULT 0,
    total_videos       INT          NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_auto_pilot_profiles_user    ON auto_pilot_profiles(user_id);
CREATE INDEX idx_auto_pilot_profiles_enabled ON auto_pilot_profiles(enabled) WHERE enabled = TRUE;

CREATE TRIGGER trg_auto_pilot_profiles_updated_at
    BEFORE UPDATE ON auto_pilot_profiles
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Track which content_plan was created by which auto_pilot profile (for analytics).
ALTER TABLE content_plans ADD COLUMN auto_pilot_profile_id UUID
    REFERENCES auto_pilot_profiles(id) ON DELETE SET NULL;

CREATE INDEX idx_content_plans_auto_pilot ON content_plans(auto_pilot_profile_id)
    WHERE auto_pilot_profile_id IS NOT NULL;

-- +goose Down
ALTER TABLE content_plans DROP COLUMN IF EXISTS auto_pilot_profile_id;
DROP TABLE IF EXISTS auto_pilot_profiles;
