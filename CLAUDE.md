# AutoContent — Project Guide for Claude

## What this project is

Full-stack social media automation platform: trend discovery → AI script → FFmpeg video → TikTok/Facebook publish, with analytics and e-commerce product tagging.

Monorepo layout:
```
mmo/
├── backend/      Go 1.25 — Gin API + Asynq workers
├── frontend/     Next.js 15 (App Router) — TypeScript + Tailwind
├── infra/        Docker, Nginx
├── docs/         Architecture, Setup, API, User Guide
└── .claude/      This folder — commands and project context
```

---

## Always verify before finishing any task

```bash
# Backend
cd backend && go build ./... && go vet ./...

# Frontend
cd frontend && npx tsc --noEmit
```

Both must pass with zero output. Never leave a half-broken build.

---

## Key commands

```bash
make dev              # start full stack via Docker Compose
make down             # stop everything
make migrate-up       # apply pending DB migrations
make migrate-create name=add_xyz   # scaffold new migration file
make be-build         # compile Go binaries (no Docker)
make be-test          # go test ./...
make fe-dev           # Next.js dev server (port 3000)
make db-shell         # psql into the local DB
make logs-api         # tail backend-api logs
make logs-worker      # tail worker + video-worker logs
```

---

## Architecture in one page

```
HTTP request
  → Gin handler (adapter/handler/)
    → Usecase (usecase/)          ← business logic only here
      → Repository (adapter/repository/)   ← DB via sqlx
      → Integration (integration/)         ← external APIs
      → Infrastructure (infrastructure/)   ← queue, R2, FFmpeg

Background tasks
  → Asynq worker (adapter/worker/)
    → same Usecase / Repository / Integration layers
```

**Rules:**
- Domain (`domain/`) has zero external imports — entities + repository interfaces only
- Usecases import domain interfaces, never concrete repos directly (except for now where concrete repos are used — keep this consistent)
- Handlers never contain business logic
- Workers never call HTTP handlers

---

## Database

**Go module:** `mmo`

**Migration tool:** `goose` (single file per migration with `-- +goose Up/Down` markers)

**Migration files:** `backend/internal/infrastructure/db/migrations/`
- `001_init.sql` — full base schema
- `002_shop_products.sql` — product catalog + junction table
- New migrations: `003_*.sql`, `004_*.sql` (sequential)

**ORM:** None. Raw SQL via `sqlx`. Always use parameterized queries (`$1`, `$2`, …).

**Pagination:** Use `util.Pagination` from `pkg/util`. Never roll your own `LIMIT`/`OFFSET`.

**Token encryption:** Social OAuth tokens are AES-256-GCM encrypted at rest. Use `pkg/crypto.Encrypt` / `pkg/crypto.Decrypt` with `cfg.Auth.EncryptionKey`. Never store tokens as plaintext.

---

## Adding a new domain feature

Checklist — follow this order:

1. **Migration** — new SQL file in `db/migrations/`
2. **Domain entity** — `internal/domain/<name>/entity.go`
3. **Repository interface** — in same domain package (optional but preferred)
4. **Repository implementation** — `internal/adapter/repository/<name>_repo.go`
5. **Usecase** — `internal/usecase/<name>_usecase.go`
6. **HTTP handler** — `internal/adapter/handler/<name>_handler.go`
7. **Wire** — add repo/usecase/handler to `cmd/api/main.go`, register routes
8. **Worker task** (if async) — `internal/adapter/worker/task_<name>.go`, register in `cmd/worker/main.go`
9. **Frontend API client** — `frontend/src/lib/api/<name>.ts`
10. **Frontend types** — add to `frontend/src/lib/types/api.types.ts`
11. **Frontend page** — `frontend/src/app/(dashboard)/<name>/page.tsx`
12. **Sidebar** — update `frontend/src/components/layout/Sidebar.tsx` if new page
13. **Verify** — `go build ./... && go vet ./... && tsc --noEmit`

---

## Configuration

**Non-sensitive settings** (timeouts, URLs, model names, queue weights, etc.) live in `backend/config.yml`. Edit that file directly.

**Secrets and per-environment overrides** come from environment variables. `config.yml` uses `${VAR}` placeholders that are expanded at startup. Syntax:
- `${VAR}` — required, fatal if empty
- `${VAR:-default}` — optional with fallback

Required env vars: `DATABASE_URL`, `REDIS_URL`, `JWT_SECRET`, `ENCRYPTION_KEY`

Optional (features degrade gracefully): all API keys (`GEMINI_API_KEY`, `PEXELS_API_KEY`, etc.)

Full reference: `docs/SETUP.md` or `.env.example`

---

## Documentation

| File | Purpose |
|---|---|
| `docs/ARCHITECTURE.md` | System diagram, pipeline flow, DB schema map |
| `docs/SETUP.md` | Dev + production setup, all env vars, OAuth registration |
| `docs/API.md` | Full REST API reference |
| `docs/USER_GUIDE.md` | End-user workflows |

---

## Sub-project guides

- `backend/CLAUDE.md` — Go conventions, handler patterns, error handling
- `frontend/CLAUDE.md` → `frontend/AGENTS.md` — Next.js version notes (**read before touching frontend**)
