# Database Migrations Guide

Migrations dùng [`goose`](https://github.com/pressly/goose) với format single-file (`-- +goose Up/Down`).

## Quy tắc vàng: Backward-Compatible 1 version

> Một migration phải để **version N-1** của code tiếp tục chạy bình thường, để auto-rollback của CI không gây crash.

Lý do: khi deploy `v1.2.3` xong mà health check fail, CI tự rollback container về `v1.2.2`. Nếu schema `v1.2.3` đã DROP column mà `v1.2.2` còn đọc column đó → container `v1.2.2` crash → outage.

## Expand-Contract pattern (3 steps, qua 3 release)

Khi cần **đổi schema không backward-compatible** (đổi tên column, đổi type, drop column…), chia thành 3 release:

### Phase 1 — EXPAND (thêm cái mới, giữ cái cũ)

```sql
-- +goose Up
-- Thêm column mới, để code cũ vẫn đọc/ghi vào column cũ
ALTER TABLE users ADD COLUMN email_v2 VARCHAR(255);

-- Backfill từ data cũ (nếu cần)
UPDATE users SET email_v2 = email WHERE email_v2 IS NULL;

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS email_v2;
```

Code release `v1.2.3` chỉ chạy migration — **chưa đụng vào code**.

### Phase 2 — MIGRATE (code dùng cái mới, cái cũ vẫn còn)

`v1.2.4`: code đọc/ghi vào `email_v2`. Cập nhật trigger để dual-write (sync giữa `email` và `email_v2`) nếu có service khác còn dùng `email`.

Tới giai đoạn này, schema chưa thay đổi gì so với `v1.2.3` → rollback an toàn.

### Phase 3 — CONTRACT (xóa cái cũ)

`v1.2.5`: code đã không còn đọc `email` (đã verify trên prod ít nhất 1 release). Bây giờ mới drop:

```sql
-- +goose Up
ALTER TABLE users DROP COLUMN email;

-- +goose Down
ALTER TABLE users ADD COLUMN email VARCHAR(255);
-- WARNING: data không thể restore — only structural rollback
```

Rollback từ `v1.2.5` về `v1.2.4` cần manual `goose down` rồi restore data từ backup.

## Các thay đổi cần Expand-Contract

| Thay đổi | Sai (1 release) | Đúng (3 release) |
|---|---|---|
| Rename column | `ALTER TABLE x RENAME COLUMN a TO b` | Add `b`, dual-write, drop `a` |
| Drop column | `ALTER TABLE x DROP COLUMN a` | Stop using → drop ở release sau |
| Change column type | `ALTER COLUMN a TYPE bigint` | Add `a_v2 bigint`, dual-write, drop `a` |
| Add NOT NULL | `ALTER COLUMN a SET NOT NULL` | Add với DEFAULT, backfill, rồi SET NOT NULL ở release sau |
| Rename table | `ALTER TABLE a RENAME TO b` | Create view `b` → `a`, đổi code, drop view & rename |

## Các thay đổi an toàn (1 release OK)

- `CREATE TABLE` (chưa có code nào dùng)
- `ADD COLUMN` (nullable hoặc có DEFAULT)
- `CREATE INDEX CONCURRENTLY` (Postgres) — tránh lock
- Update data thuần (UPDATE, INSERT)
- Add FK với `NOT VALID` rồi `VALIDATE CONSTRAINT` ở release sau

## Postgres-specific gotchas

| Operation | Vấn đề | Workaround |
|---|---|---|
| `ADD COLUMN ... NOT NULL DEFAULT 'x'` | Pre-PG11: rewrite cả table (LOCK lâu) | PG11+ OK (metadata-only). Hoặc add nullable → backfill → SET NOT NULL |
| `CREATE INDEX` | Lock table | Dùng `CREATE INDEX CONCURRENTLY` |
| `ALTER TABLE ADD CONSTRAINT FK` | Check toàn bảng (lock dài) | `ADD CONSTRAINT ... NOT VALID` → `VALIDATE CONSTRAINT` (cũ hơn) |
| Large DELETE | Bloat | Batch theo chunks |

## Lint trong CI

`pr-check.yml` chạy [`squawk`](https://squawkhq.com) trên migration files mới. Sẽ warn/fail nếu phát hiện:
- `prefer-robust-stmts` — thiếu `IF [NOT] EXISTS`
- `disallowed-unique-constraint` — `ADD CONSTRAINT UNIQUE` không dùng index concurrently
- `adding-required-field` — `NOT NULL` không có DEFAULT
- `adding-field-with-default` — pre-PG11 issue
- `renaming-column` / `renaming-table`
- `disallowed-schema` — đụng vào schema cấm
- ...

Nếu cần override (vd có lý do chính đáng để rename), thêm comment `-- squawk-ignore-next-statement: <rule>` ngay trước statement.

## Tự test backward-compat trước khi merge

```bash
# 1. Apply migration mới
make migrate-up

# 2. Chạy code version cũ (vd checkout commit trước)
git stash
git checkout HEAD~1
make dev
curl http://localhost:8080/health

# 3. Kiểm tra: API có hoạt động không?
# Nếu có lỗi liên quan column/table → migration không backward-compat → cần Expand-Contract

# Quay lại
git checkout -
git stash pop
```

## Checklist khi viết migration

- [ ] Migration backward-compat với code version trước? (test bước trên)
- [ ] DROP/RENAME/TYPE change → chia ít nhất 2 release?
- [ ] NOT NULL có DEFAULT?
- [ ] CREATE INDEX có CONCURRENTLY (nếu bảng lớn)?
- [ ] `Down` migration revert được structure?
- [ ] `squawk` pass trong PR check?
