# Setup Guide

## Prerequisites

| Tool | Version | Notes |
|---|---|---|
| Docker | 24+ | with Docker Compose v2 |
| Go | 1.25+ | for local backend dev only |
| Node.js | 20 LTS | for local frontend dev only |
| Python | 3.11+ | for Edge TTS (inside Docker) |

---

## 1. Local Development (Docker)

### 1.1 Clone and configure

```bash
git clone <repo-url> mmo
cd mmo
make setup         # copies .env.example to .env
```

Open `.env` and fill in required values (see **Environment Variables** below).

### 1.2 Start all services

```bash
make dev           # builds images and starts everything
# or in background:
make dev-bg
```

Services started:
| Service | URL | Notes |
|---|---|---|
| Frontend | http://localhost:3000 | Next.js |
| Backend API | http://localhost:8080 | via nginx at :80 too |
| Nginx | http://localhost:80 | routes /api → backend, / → frontend |
| PostgreSQL | localhost:5432 | `mmo` / `mmo_dev_password` |
| Redis | localhost:6379 | no auth in dev |

### 1.3 Run database migrations

```bash
make migrate-up
```

Applies both migrations:
- `001_init.sql` — core schema
- `002_shop_products.sql` — product catalog tables

### 1.4 First login

1. Open http://localhost:3000
2. Register a new account
3. Connect a TikTok or Facebook channel (Settings → Channels)
4. Trigger trend discovery: **Content → Discover Trends**

---

## 2. Configuration

Configuration is split across two layers:

| Layer | File | Purpose |
|---|---|---|
| Non-sensitive defaults | `backend/config.yml` | All timeouts, URLs, model names, queue settings, etc. |
| Secrets & overrides | `.env` | API keys, DB/Redis URLs, JWT secrets |

### How it works

`config.yml` uses `${VAR}` and `${VAR:-default}` placeholders that are expanded from environment variables at startup:

```yaml
# config.yml — example entries
db:
  url: "${DATABASE_URL}"          # required, fatal if empty
app:
  env: "${APP_ENV:-development}"  # optional, falls back to "development"
gemini:
  api_key: "${GEMINI_API_KEY}"    # optional, feature degrades if empty
```

You only need to set env vars for values that differ from the defaults in `config.yml`. For **local Docker dev**, `make setup` + filling in secrets in `.env` is all that's needed.

To use a different config file: `CONFIG_FILE=/path/to/config.yml ./bin/api`

### Required secrets ★

These have no default and will cause a fatal error on startup if missing:

```env
# JWT signing key — generate: openssl rand -hex 32
JWT_SECRET=your-32-byte-hex-secret

# AES-256-GCM key for encrypting OAuth tokens — MUST be exactly 32 bytes
ENCRYPTION_KEY=your-exactly-32-char-key

# Set by docker-compose automatically; needed for local dev without Docker
DATABASE_URL=postgres://mmo:mmo_dev_password@localhost:5432/mmo?sslmode=disable
REDIS_URL=redis://localhost:6379
```

> **Generate ENCRYPTION_KEY:** `openssl rand -base64 32 | head -c 32`

In production use a strong DB password and `sslmode=require`.

### Optional overrides

These have sensible defaults in `config.yml` but can be overridden via env:

```env
APP_ENV=development          # development | production  (default: development)
APP_PORT=8080                # (default: 8080)
FRONTEND_URL=https://yourdomain.com  # (default: http://localhost:3000)
```

### Cloudflare R2 (video storage)

R2 is the primary video storage. Create a bucket at [Cloudflare Dashboard](https://dash.cloudflare.com/) → R2.

```env
R2_ACCOUNT_ID=your-account-id
R2_ACCESS_KEY_ID=your-r2-access-key-id
R2_SECRET_ACCESS_KEY=your-r2-secret-key
R2_BUCKET_NAME=mmo-media          # (default: mmo-media)
R2_PUBLIC_URL=https://pub-xxxx.r2.dev
```

Enable "Public access" on the R2 bucket for video playback URLs to work.

### AI & Media APIs

```env
# Google Gemini — script generation (1,500 free requests/day)
GEMINI_API_KEY=your-gemini-key        # https://aistudio.google.com/apikey

# AI provider abstraction (script generation is behind the ai.ScriptGenerator port)
AI_PROVIDER=gemini                    # gemini | mock  (mock = deterministic, no network)
AI_FALLBACK_TO_MOCK=true              # on Gemini error/quota, fall back to mock (logged)
GEMINI_MODEL=gemini-2.5-flash         # config-driven model name

# Stock media (free tiers)
PEXELS_API_KEY=your-pexels-key        # https://www.pexels.com/api/
PIXABAY_API_KEY=your-pixabay-key      # https://pixabay.com/api/docs/
YOUTUBE_API_KEY=your-youtube-key      # https://console.cloud.google.com  (trending discovery only)
```

### Dry-run publishing (test the whole pipeline without posting)

```env
PUBLISH_DRY_RUN=true   # ALL publishes are mocked — produces real videos, no social posting
```

When `true`, the pipeline runs end-to-end (discover → script → media → TTS → FFmpeg →
R2 upload → publish) but the publish step returns a synthetic `dryrun_<platform>_<uuid>`
post id instead of calling TikTok/Facebook/YouTube. A **per-channel** `dry_run` flag does
the same for individual channels (used by the one-click "MMO channel" quick-setup). For a
fully hermetic local run, combine with `AI_PROVIDER=mock`.

### YouTube Shorts OAuth (publishing)

Separate from `YOUTUBE_API_KEY` (which is only for trending discovery). Create an OAuth 2.0
Client (type: Web) in Google Cloud Console with the `youtube.upload` scope:

```env
YOUTUBE_OAUTH_CLIENT_ID=your-google-oauth-client-id
YOUTUBE_OAUTH_CLIENT_SECRET=your-google-oauth-client-secret
YOUTUBE_PRIVACY_STATUS=unlisted       # public | unlisted | private
```

Redirect URL (in `config.yml`): `${FRONTEND_URL}/channels/callback/youtube`.
Uploaded videos get `#Shorts` appended so YouTube classifies the vertical clip as a Short.

### TikTok OAuth

Register a developer app at https://developers.tiktok.com

```env
TIKTOK_CLIENT_KEY=your-tiktok-client-key
TIKTOK_CLIENT_SECRET=your-tiktok-client-secret
```

Redirect URL is set in `config.yml`: `${FRONTEND_URL}/channels/callback/tiktok`

Required OAuth scopes: `user.info.basic`, `video.upload`, `video.publish`

**TikTok Shop** (optional — for product tagging):
```env
TIKTOK_SHOP_API_KEY=your-shop-api-key
TIKTOK_SHOP_API_SECRET=your-shop-api-secret
```

### Facebook OAuth

Register a Meta app at https://developers.facebook.com

```env
FACEBOOK_APP_ID=your-facebook-app-id
FACEBOOK_APP_SECRET=your-facebook-app-secret
```

Redirect URL is set in `config.yml`: `${FRONTEND_URL}/channels/callback/facebook`

Required permissions: `pages_manage_posts`, `pages_read_engagement`, `pages_show_list`

### Tuning non-sensitive settings

Edit `backend/config.yml` directly for:
- Server timeouts, DB connection pool sizes
- Queue concurrency and priority weights
- FFmpeg output resolution / codec settings
- Gemini model name, API base URLs
- Cron schedule expressions
- TTS default voice

---

## 3. Local Development (without Docker)

### Backend

```bash
cd backend

# Install Go dependencies
go mod download

# config.yml is already at backend/config.yml with all defaults.
# Export required secrets before running:
export DATABASE_URL=postgres://mmo:mmo_dev_password@localhost:5432/mmo?sslmode=disable
export REDIS_URL=redis://localhost:6379
export JWT_SECRET=change_me_in_production_please
export ENCRYPTION_KEY=change_me_32_bytes_key_in_prod!!

# Run API server
go run ./cmd/api

# Run general worker
go run ./cmd/worker

# Run video-only worker (separate terminal)
go run ./cmd/worker --video-only
```

To point at a custom config file:
```bash
CONFIG_FILE=/path/to/my-config.yml go run ./cmd/api
```

### Frontend

```bash
cd frontend
npm install
npm run dev      # starts on :3000
```

### Database (standalone)

Start PostgreSQL and Redis locally, then update `.env` connection strings.

Migrations:
```bash
make migrate-up
# or manually (requires goose installed locally):
goose -dir backend/internal/infrastructure/db/migrations postgres "${DATABASE_URL}" up
```

---

## 4. Makefile Reference

```bash
make setup          # copy .env.example → .env
make config         # open backend/config.yml in $EDITOR
make dev            # docker compose up --build
make dev-bg         # docker compose up -d --build
make down           # docker compose down
make logs           # follow all service logs
make migrate-up     # apply pending DB migrations
make migrate-down   # rollback one migration
make be-build       # go build ./...
make be-test        # go test ./...
make be-vet         # go vet ./...
make fe-dev         # npm run dev (frontend)
make fe-build       # npm run build (frontend)
make fe-lint        # npm run lint (frontend)
```

---

## 5. Production Deployment

### 5.1 Server requirements

Recommended: **Hetzner CX32** (~$15/month)
- 4 vCPU, 8 GB RAM, 80 GB SSD
- OS: Ubuntu 22.04 LTS

Install prerequisites:
```bash
apt update && apt upgrade -y
apt install -y docker.io docker-compose-v2 git curl
systemctl enable --now docker
```

### 5.2 Configure production env

```bash
cp .env.example .env.prod
# Edit .env.prod with production values:
# - APP_ENV=production
# - FRONTEND_URL=https://yourdomain.com
# - DATABASE_URL with strong password
# - All API keys
```

Key differences from dev:
```env
APP_ENV=production
FRONTEND_URL=https://yourdomain.com
DATABASE_URL=postgres://mmo:STRONG_PASSWORD@postgres:5432/mmo?sslmode=require
```

### 5.3 SSL / HTTPS

Use Caddy or Certbot in front of Nginx:

**Caddy (recommended):**
```
yourdomain.com {
    reverse_proxy localhost:80
}
```

### 5.4 Deploy

```bash
# Clone repo on server
git clone <repo> mmo && cd mmo

# Configure env
cp .env.example .env.prod && nano .env.prod

# Start production stack
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d

# Run migrations
docker compose exec backend-api /app/bin/goose -dir /app/migrations postgres "${DATABASE_URL}" up
```

### 5.5 Backups

PostgreSQL daily backup to R2:
```bash
# Add to crontab: 0 3 * * *
pg_dump "${DATABASE_URL}" | gzip | aws s3 cp - s3://mmo-media/backups/$(date +%Y%m%d).sql.gz \
  --endpoint-url https://${R2_ACCOUNT_ID}.r2.cloudflarestorage.com
```

---

## 6. Social Platform OAuth Setup

### TikTok

1. Go to https://developers.tiktok.com → **My Apps → Create App**
2. Add products: **Login Kit** + **Content Posting API**
3. Set redirect URI: `https://yourdomain.com/channels/callback/tiktok`
4. Copy **Client Key** and **Client Secret** to `.env`
5. Submit for review (required for `video.publish` scope)

### Facebook / Meta

1. Go to https://developers.facebook.com → **My Apps → Create App** (type: Business)
2. Add products: **Facebook Login**, **Pages API**
3. Set redirect URI: `https://yourdomain.com/channels/callback/facebook`
4. Under App Review, request: `pages_manage_posts`, `pages_read_engagement`, `pages_show_list`
5. Copy **App ID** and **App Secret** to `.env`

---

## 7. TikTok Shop Setup (Optional)

TikTok Shop API is separate from TikTok Open API.

1. Register at https://seller.tiktok.com → apply for **TikTok Shop Partner API**
2. After approval, get **Shop API Key** and **Secret**
3. Add to `.env`:
   ```env
   TIKTOK_SHOP_API_KEY=xxx
   TIKTOK_SHOP_API_SECRET=xxx
   ```
4. In the app: **Products → Sync from Shop** → select TikTok channel

---

## 8. Troubleshooting

### Videos not assembling

1. Check `backend-video-worker` logs: `docker compose logs backend-video-worker`
2. Verify ffmpeg is available: `docker compose exec backend-video-worker ffmpeg -version`
3. Ensure TTS works: `docker compose exec backend-video-worker edge-tts --version`
4. Check tmp directory has space: `df -h`

### OAuth redirect not working

- Verify `FRONTEND_URL` in `.env` matches exactly what's registered in TikTok/Facebook developer console
- Ensure no trailing slash in `FRONTEND_URL`

### Publishing fails

1. Check access token isn't expired: check `channels.token_expires_at` in DB
2. `ENCRYPTION_KEY` must not have changed since channels were connected — doing so will break all token decryption
3. Check TikTok API status at https://developers.tiktok.com/status

### Database migration errors

```bash
# Check current version
docker compose exec backend-api /app/bin/goose -dir /app/migrations postgres "${DATABASE_URL}" status

# Rollback one migration if stuck
make migrate-down
```

### R2 uploads failing

- Verify bucket name and region match the credentials
- Ensure bucket has **Public access** enabled for playback
- Check S3 endpoint: `https://${R2_ACCOUNT_ID}.r2.cloudflarestorage.com`
