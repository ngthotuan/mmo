Guide the implementation of a new end-to-end feature following the project's Clean Architecture conventions.

The user will describe the feature. Walk through each step below, implementing and verifying as you go.

## Step-by-step checklist

### 1. Plan
- Identify what domain entity/entities are needed
- Identify what API endpoints are needed
- Identify if any async worker tasks are needed
- Confirm the plan with the user before writing any code

### 2. Backend — Domain
- Create `backend/internal/domain/<name>/entity.go` with Go structs and typed status constants
- No external imports in domain — only stdlib and `github.com/google/uuid`

### 3. Backend — Migration
- Add `backend/internal/infrastructure/db/migrations/NNN_<name>.sql`
- Run `make migrate-up`

### 4. Backend — Repository
- Create `backend/internal/adapter/repository/<name>_repo.go`
- Implement: Create, GetByID, List (with `util.Pagination`), Update, Delete at minimum
- Use `pq.Array()` for TEXT[] columns
- Return `apperr.ErrNotFound` when `sql.ErrNoRows`

### 5. Backend — Usecase
- Create `backend/internal/usecase/<name>_usecase.go`
- Constructor takes only concrete repo types and clients needed
- Define request structs inline (e.g., `CreateXxxRequest`)
- Business rules and validation live here — not in handlers

### 6. Backend — Handler
- Create `backend/internal/adapter/handler/<name>_handler.go`
- Use `mustParseUserID(c)` and `respondErr(c, err)` helpers
- Parse UUIDs with `uuid.Parse(c.Param("id"))` and respond 400 on error
- Use `util.ParsePagination(c)` for paginated endpoints

### 7. Backend — Wire
- Add repo to `cmd/api/main.go` repositories section
- Add usecase wiring
- Add handler wiring
- Add routes in the appropriate group

### 8. Backend — Worker (if async)
- Create `backend/internal/adapter/worker/task_<name>.go`
- Add task constant to `internal/infrastructure/queue/tasks.go`
- Register handler in `cmd/worker/main.go`
- Add cron schedule if needed

### 9. Backend — Verify
```bash
cd backend && go build ./... && go vet ./...
```
Both must pass with zero output before moving to frontend.

### 10. Frontend — Types
- Add new interface(s) to `frontend/src/lib/types/api.types.ts`

### 11. Frontend — API client
- Create `frontend/src/lib/api/<name>.ts`
- Export a `<name>Api` object with typed functions using `apiClient`

### 12. Frontend — Page
- Create `frontend/src/app/(dashboard)/<name>/page.tsx`
- Use `"use client"` directive
- Use `useQuery` for data fetching, `useMutation` for writes
- Use `toast.success(...)` / `toast.error(...)` from `sonner` for feedback
- Show `<Skeleton>` while loading, empty state when no data

### 13. Frontend — Sidebar
- If a new top-level page, add entry to `frontend/src/components/layout/Sidebar.tsx`
- Import icon from `lucide-react`

### 14. Frontend — Verify
```bash
cd frontend && npx tsc --noEmit
```
Must pass with zero output.

## Conventions to follow

- No business logic in handlers — move to usecase
- No raw SQL in usecases — use repository methods
- No `fmt.Println` — use `logger.Info/Warn/Error`
- Social tokens: always `crypto.Encrypt/Decrypt` — never store plaintext
- `platform` field from `channel.Platform`: cast with `string(channel.Platform)`
- `uuid.UUID` to `*uuid.UUID`: `x := val; ptr = &x`
