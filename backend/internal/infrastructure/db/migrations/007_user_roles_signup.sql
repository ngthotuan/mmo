-- +goose Up
-- Role model:
--   admin  — full access + user management
--   member — full access to all features (granted by an admin)
--   viewer — read-only; default for new sign-ups until an admin grants access
-- New sign-ups default to 'viewer'. Existing 'owner' accounts become 'admin' so
-- that user management is not locked out after this migration.
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'viewer';
UPDATE users SET role = 'admin' WHERE role = 'owner';

-- +goose Down
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'owner';
UPDATE users SET role = 'owner' WHERE role IN ('admin', 'member', 'viewer');
