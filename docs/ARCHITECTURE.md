# Architecture

## System Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                            Docker Network                           │
│                                                                     │
│  ┌──────────┐    ┌─────────────────┐    ┌───────────────────────┐  │
│  │  Nginx   │───▶│  Next.js (FE)   │    │    Backend API (Go)   │  │
│  │  :80     │    │  :3000          │    │    Gin + JWT  :8080   │  │
│  └──────────┘    └─────────────────┘    └──────────┬────────────┘  │
│       │                                            │               │
│       └────────────── /api/* ──────────────────────┘               │
│                                                    │               │
│  ┌─────────────────┐  ┌────────────────┐           │               │
│  │  General Worker │  │  Video Worker  │           │               │
│  │  (Asynq)        │  │  (FFmpeg+TTS)  │           │               │
│  └────────┬────────┘  └───────┬────────┘           │               │
│           │                   │                    │               │
│           └───────────────────┴────────────────────┘               │
│                               │                                    │
│              ┌────────────────┼────────────────┐                   │
│              ▼                ▼                ▼                   │
│         ┌─────────┐    ┌──────────┐    ┌────────────┐             │
│         │ Redis 7 │    │ Postgres │    │   Nginx    │             │
│         │ (queue) │    │    16    │    │ (static,   │             │
│         └─────────┘    └──────────┘    │  reverse   │             │
│                                        │  proxy)    │             │
│                                        └────────────┘             │
└─────────────────────────────────────────────────────────────────────┘

External:
  ┌─────────────────┐  ┌──────────────┐  ┌──────────────┐
  │ Cloudflare R2   │  │ TikTok API   │  │ Meta API     │
  │ (video storage) │  │ (publish +   │  │ (publish +   │
  └─────────────────┘  │  shop)       │  │  catalog)    │
                       └──────────────┘  └──────────────┘
  ┌─────────────────┐  ┌──────────────┐
  │ Gemini Flash    │  │ Pexels +     │
  │ (script gen)    │  │ Pixabay      │
  └─────────────────┘  └──────────────┘
```

---

## Backend — Clean Architecture

```
backend/
├── cmd/
│   ├── api/main.go          HTTP server, dependency wiring
│   └── worker/main.go       Asynq worker, cron schedules
├── internal/
│   ├── domain/              Entities + repository interfaces
│   │   ├── channel/
│   │   ├── content/
│   │   ├── video/
│   │   ├── publish/
│   │   └── product/
│   ├── usecase/             Business logic (no framework deps)
│   ├── adapter/
│   │   ├── handler/         Gin HTTP handlers
│   │   ├── repository/      PostgreSQL implementations (sqlx)
│   │   └── worker/          Asynq task handlers
│   ├── infrastructure/
│   │   ├── db/              Connection pool + migrations
│   │   ├── queue/           Asynq client/server/scheduler
│   │   ├── ffmpeg/          Video assembler
│   │   └── storage/         Cloudflare R2 client
│   └── integration/         Third-party API clients
│       ├── tiktok/          Open API + Shop API
│       ├── facebook/        Graph API + Catalog
│       ├── gemini/          Script generation
│       ├── pexels/          Stock video/photos
│       ├── pixabay/         Stock video/photos (fallback)
│       ├── edgetts/         TTS via Python subprocess
│       ├── googletrends/    Trend discovery
│       ├── youtube/         Trending videos
│       └── reddit/          Trending posts
└── pkg/
    ├── config/              YAML + env-var configuration (config.yml + ${VAR} expansion)
    ├── crypto/              AES-256-GCM token encryption
    ├── errors/              Typed app errors
    ├── logger/              Zap structured logging
    ├── middleware/           JWT auth, CORS
    └── util/                Pagination helpers
```

### Dependency flow

```
handler → usecase → repository (DB)
                 → integration (external APIs)
                 → infrastructure (queue, storage)
worker  → usecase → repository
                 → integration
```

No layer imports from a higher layer. Domain has zero external dependencies.

---

## Pipeline — Full Automation Flow

```
[Cron 6h]
    │
    ▼
TaskDiscoverTrends
  ├─ Google Trends scraper
  ├─ YouTube Data API v3
  └─ Reddit API
    │
    │ (enqueues TaskGenerateScript per topic)
    ▼
TaskGenerateScript
  └─ Gemini 1.5 Flash → ScriptResult{title, hook, body, cta, hashtags, caption}
     └─ content_plan saved (status: draft)
        │
        │ [Human review or auto-approve]
        ▼
      approved
        │ (enqueues TaskCollectMedia)
        ▼
TaskCollectMedia
  ├─ Pexels API: SearchVideos / SearchImages
  └─ Pixabay API: SearchVideos / SearchImages (fallback)
     └─ files downloaded to /tmp, uploaded to R2 (media_assets)
        │ (enqueues TaskGenerateTTS)
        ▼
TaskGenerateTTS
  └─ edge-tts CLI → {script}.mp3, {script}.srt
     └─ uploaded to R2
        │ (enqueues TaskAssembleVideo)
        ▼
TaskAssembleVideo [video worker]
  ├─ Slideshow: images + ken-burns + TTS + subtitles
  ├─ B-roll:    video clips + TTS sync + subtitles
  └─ TextOnVideo: bg video + drawtext + TTS
     └─ output: 1080×1920 H.264+AAC .mp4
        │ (enqueues TaskUploadToR2)
        ▼
TaskUploadToR2
  └─ videos/{jobID}/output.mp4 → public R2 URL
     └─ video_job.status = done
        │
        │ [User schedules via UI → POST /publish]
        ▼
publish_job (status: scheduled)
        │
[Cron 1min]
        ▼
TaskCheckPublish
  └─ SELECT due jobs → enqueues TaskPublishNow (critical queue)
        │
        ▼
TaskPublishNow
  ├─ TikTok: POST /v2/post/publish/video/init/ + optional product_links
  └─ Facebook: POST /{pageID}/videos + optional product URLs in description
     └─ publish_job.status = published

[Cron 2AM]
TaskSyncAnalytics
  └─ Fetch per-post stats → upsert post_analytics
```

---

## Queue Architecture

Three queues with weighted concurrency:

| Queue | Weight | Used for |
|---|---|---|
| `critical` | 6 | `task:publish_now` — time-sensitive |
| `default` | 3 | Pipeline steps (media, TTS, assemble, upload) |
| `low` | 1 | Cron jobs (trends, analytics, token refresh) |

Two worker processes:
- **backend-worker**: handles all queues except video assembly
- **backend-video-worker**: handles only `task:assemble_video` + `task:upload_to_r2` (CPU-isolated)

---

## Database Schema

```
users
  └──< channels (platform, encrypted_tokens, page_id)
  └──< trend_topics (source, keywords, score)
  └──< content_plans (script, status: draft→approved→video_queued→video_ready→published)
         └──< video_jobs (status: pending→media_collecting→tts_generating→assembling→uploading→done/failed)
                └──< publish_jobs (status: scheduled→publishing→published/failed)
                       └──< post_analytics (views, likes, comments, shares, reach)
                       └──< publish_job_products (junction)
  └──< products (platform_product_id, name, price, currency, image, url)
  video_templates (type: slideshow/b_roll/text_on_video, config JSONB)
  media_assets (r2_key, type: video/image, source: pexels/pixabay)
  job_logs (task, status, message, timestamps)
```

Migrations:
- `001_init.sql` — full initial schema, auto-updated_at triggers
- `002_shop_products.sql` — products + publish_job_products

---

## Security

| Concern | Implementation |
|---|---|
| API auth | JWT (HS256), 15-min access token, 7-day refresh token |
| Social tokens | AES-256-GCM encryption at rest (`ENCRYPTION_KEY`) |
| Password | bcrypt (default cost) |
| CORS | Allowlist: `FRONTEND_URL` env var only |
| SQL injection | sqlx parameterized queries throughout |

---

## Frontend Architecture

```
frontend/src/
├── app/
│   ├── (auth)/           Login, register pages
│   └── (dashboard)/      Protected layout
│       ├── page.tsx      Dashboard overview
│       ├── channels/     OAuth connect + manage
│       ├── content/      Trend review + script editing
│       ├── videos/       Video library + status
│       ├── publish/      Publish queue + product tagging
│       ├── schedule/     Calendar view
│       ├── analytics/    Stats dashboard
│       ├── products/     Shop product catalog
│       └── settings/     Profile + password
├── components/
│   ├── layout/           Sidebar, Header
│   └── ui/               shadcn/ui primitives
└── lib/
    ├── api/              Axios clients per domain
    ├── types/            Shared TypeScript interfaces
    └── store/            Zustand UI state
```

**Data fetching:** TanStack Query with cache invalidation on mutations. Auto-refetch intervals on live-updating pages (pipeline: 15s, videos: 10s).

**Auth:** JWT stored in `localStorage`, Axios interceptor attaches `Authorization: Bearer` header, auto-refreshes on 401.
