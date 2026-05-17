# Backend — Go Conventions

**Module:** `mmo`  
**Go version:** 1.25  
**Framework:** Gin (HTTP), Asynq (task queue), sqlx (DB)

---

## Verify after every change

```bash
cd backend
go build ./...   # must produce zero output
go vet ./...     # must produce zero output
```

---

## Layer rules

| Layer | Package | Imports |
|---|---|---|
| Domain | `internal/domain/*/` | stdlib only |
| Usecase | `internal/usecase/` | domain, pkg/*, infrastructure interfaces |
| Repository | `internal/adapter/repository/` | domain, sqlx, pkg/* |
| Handler | `internal/adapter/handler/` | usecase, pkg/errors, pkg/util, gin |
| Worker | `internal/adapter/worker/` | usecase, domain, integration, pkg/* |
| Integration | `internal/integration/*/` | stdlib, pkg/config |
| Infrastructure | `internal/infrastructure/*/` | stdlib, AWS SDK, asynq, etc. |

Never skip layers (e.g. handler calling repository directly).

---

## Handler conventions

Every handler file shares helpers from `channel_handler.go`:

```go
// Parse the authenticated user's UUID — panics on missing claim (should never happen with middleware)
userID := mustParseUserID(c)   // returns uuid.UUID

// Respond with typed error (apperr.AppError) or generic 500
respondErr(c, err)
```

**Pagination:** Always call `util.ParsePagination(c)` to extract `?page=` / `?perPage=`.

**Response shape:**
- Single resource: `c.JSON(200, gin.H{"data": obj})`
- List: `c.JSON(200, gin.H{"data": slice, "total": n})` — note `"total"` not `"pagination"` for simple lists, but `"pagination"` object when full pagination is needed (match what frontend expects)
- Created: `c.JSON(201, gin.H{"data": obj})`
- Action: `c.JSON(200, gin.H{"message": "done"})`

---

## Error handling

```go
import apperr "mmo/pkg/errors"

// Named errors
apperr.ErrNotFound
apperr.ErrBadRequest
apperr.ErrUnauthorized
apperr.ErrConflict
apperr.ErrInternalServer
apperr.ErrInvalidCredential
apperr.ErrInvalidToken

// Custom message
apperr.New(http.StatusBadRequest, "job is not in failed state")

// With detail (for validation)
apperr.WithDetail(http.StatusBadRequest, "validation error", err.Error())
```

`respondErr(c, err)` unwraps `*apperr.AppError` and writes the correct status. All other errors → 500.

---

## Repository patterns

```go
// Standard constructor
func NewXxxRepo(db *sqlx.DB) *XxxRepo { return &XxxRepo{db: db} }

// Parameterized queries — always $1, $2, ...
r.db.SelectContext(ctx, &dest, "SELECT * FROM table WHERE user_id=$1", userID)
r.db.GetContext(ctx, &dest, "SELECT * FROM table WHERE id=$1", id)
r.db.ExecContext(ctx, "INSERT INTO ...")

// Not found
if errors.Is(err, sql.ErrNoRows) { return nil, apperr.ErrNotFound }

// Arrays (TEXT[])
pq.Array(slice)   // for INSERT/UPDATE
// sqlx automatically scans pq arrays back into []string
```

**Pagination:**
```go
func (r *Repo) List(ctx, userID, pg util.Pagination) ([]T, int, error) {
    // count query first, then data query with LIMIT pg.Limit() OFFSET pg.Offset()
}
```

---

## Asynq task conventions

**Task name format:** `task:<verb>_<noun>` — e.g., `task:publish_now`, `task:sync_analytics`

**Constants** in `internal/infrastructure/queue/tasks.go`

**Enqueue:**
```go
payload, _ := json.Marshal(map[string]string{"id": id.String()})
t := asynq.NewTask(queue.TaskXxx, payload, asynq.Queue(queue.QueueDefault))
_, err = u.queue.EnqueueContext(ctx, t)
```

**Handler:**
```go
func (h *XxxHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
    var p struct { ID string `json:"id"` }
    if err := json.Unmarshal(task.Payload(), &p); err != nil { return err }
    ...
}
```

Register in `cmd/worker/main.go`:
```go
mux.HandleFunc(queue.TaskXxx, handler.ProcessTask)
```

---

## Config

Configuration is split across two files:
- `backend/config.yml` — all non-sensitive values (timeouts, URLs, model names, queue weights, etc.)
- `.env` / environment variables — secrets and per-environment overrides

`config.yml` supports `${VAR}` and `${VAR:-default}` placeholders expanded at startup.

**To add a new config field:**
1. Add the value to `config.yml` (use `${ENV_VAR}` if it's a secret)
2. Add the field to the matching `raw*` struct in `pkg/config/config.go`
3. Map it to the public `Config` struct inside `Load()`
4. Use `mustField()` for required strings, `mustDuration()` for durations, `mustInt()` for ints

Pass config through constructors — never call `os.Getenv` outside `pkg/config`.

---

## Crypto

OAuth tokens are encrypted before DB insert:
```go
encrypted, err := crypto.Encrypt([]byte(cfg.Auth.EncryptionKey), plaintext)
decrypted, err := crypto.Decrypt([]byte(cfg.Auth.EncryptionKey), encrypted)
```

`ENCRYPTION_KEY` must be exactly 32 bytes. Changing it after channels are connected will break all token decryption.

---

## Logging

```go
import "mmo/pkg/logger"
import "go.uber.org/zap"

logger.Info("message", zap.String("key", val), zap.Error(err))
logger.Warn(...)
logger.Error(...)
logger.Fatal(...)   // calls os.Exit(1)
```

No `fmt.Println` in production code.

---

## Adding a new migration

```bash
make migrate-create name=add_my_table
# creates: backend/internal/infrastructure/db/migrations/20060102150405_add_my_table.sql
make migrate-up
```

File format — single file per migration (goose):
```sql
-- +goose Up
CREATE TABLE ...;

-- +goose Down
DROP TABLE IF EXISTS ...;
```

---

## Common pitfalls

| Mistake | Correct approach |
|---|---|
| `channel.Platform` in string context | `string(channel.Platform)` — typed string alias |
| `uuid.UUID` → `*uuid.UUID` | `x := val; ptr = &x` — never `&val` inline |
| `R2Client.Upload()` return value | Returns only `error`; get public URL via `r2.PublicURL(key)` |
| `apperr.NewBadRequest(...)` | Doesn't exist — use `apperr.New(http.StatusBadRequest, "msg")` |
| Modifying published/publishing job | Check status first — guard with `apperr.New(400, "cannot update...")` |
| TEXT[] column | Use `pq.Array(slice)` in queries |
