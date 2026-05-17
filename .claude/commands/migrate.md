Create and apply a new database migration.

The user will provide a migration name as an argument (e.g., `/migrate add_notifications`).

Steps:
1. Determine the next sequential number by listing files in `backend/internal/infrastructure/db/migrations/` and finding the highest prefix (e.g., if `002_shop_products.sql` exists, next is `003`).
2. Create the file `backend/internal/infrastructure/db/migrations/NNN_<name>.sql` with this template:
   ```sql
   -- +goose Up

   -- TODO: write your migration here

   -- +goose Down

   -- TODO: write rollback here
   ```
3. Ask the user what the migration should contain (tables, columns, indexes, constraints).
4. Write the actual SQL based on their answer, following these conventions:
   - UUIDs: `UUID PRIMARY KEY DEFAULT gen_random_uuid()`
   - Timestamps: `TIMESTAMPTZ NOT NULL DEFAULT NOW()`
   - Foreign keys: `REFERENCES table(id) ON DELETE CASCADE` (or SET NULL where appropriate)
   - Add indexes for foreign key columns and common filter columns
   - `updated_at` columns need a trigger — add: `CREATE TRIGGER trg_<table>_updated_at BEFORE UPDATE ON <table> FOR EACH ROW EXECUTE FUNCTION set_updated_at();`
   - The `set_updated_at()` function already exists from migration 001 — do not recreate it
5. Show the final SQL for review.
6. After user confirms, run `make migrate-up` to apply it.
7. Run `cd backend && go build ./...` to confirm nothing broke.
