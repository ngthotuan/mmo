-- +goose Up

ALTER TABLE content_plans ADD COLUMN IF NOT EXISTS voice VARCHAR(100) NOT NULL DEFAULT '';

-- +goose Down

ALTER TABLE content_plans DROP COLUMN IF EXISTS voice;
