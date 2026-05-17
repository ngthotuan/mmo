# AutoContent — Social Media Automation Platform

Automate the entire short-form video pipeline: trend discovery → AI script → video assembly → multi-channel publishing (TikTok, Facebook), with analytics and e-commerce product tagging.

## Features

| Feature | Description |
|---|---|
| Trend discovery | Google Trends, YouTube Data API, Reddit — every 6 hours |
| AI script generation | Google Gemini 1.5 Flash — hook, script, CTA, hashtags |
| Video assembly | FFmpeg (1080×1920 9:16) — Slideshow / B-roll / Text-on-video |
| TTS narration | Edge TTS (Microsoft, free) — .mp3 + .srt subtitles |
| Stock media | Pexels + Pixabay (free tiers) |
| Cloud storage | Cloudflare R2 — zero egress cost |
| Publishing | TikTok Content Posting API v2, Meta Graph API v20 |
| Scheduler | Cron-based publish queue (1-minute resolution) |
| Analytics | Daily sync of views, likes, comments, shares |
| Product tagging | TikTok Shop product links + Facebook catalog URLs |
| Multi-channel | Multiple TikTok and Facebook accounts per user |

## Quick Start

```bash
git clone <repo>
cd mmo
make setup        # copies .env.example → .env
# Edit .env with your credentials (see docs/SETUP.md)
make dev          # starts all services via Docker Compose
```

Open http://localhost:3000 → register → connect channels → discover trends → approve scripts → videos are assembled automatically.

## Documentation

| Document | Description |
|---|---|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | System design, data flow, component overview |
| [docs/SETUP.md](docs/SETUP.md) | Developer setup + production deployment guide |
| [docs/API.md](docs/API.md) | REST API reference |

## Tech Stack

**Backend:** Go 1.23, Gin, Asynq + Redis, sqlx + PostgreSQL  
**Frontend:** Next.js 15 (App Router), TypeScript, Tailwind CSS, shadcn/ui, TanStack Query  
**Infrastructure:** Docker Compose, Nginx, Cloudflare R2, FFmpeg

## Estimated Monthly Cost

| Component | Cost |
|---|---|
| VPS (Hetzner CX32 — 4 vCPU, 8 GB) | ~$15 |
| Cloudflare R2 storage | $0.015/GB (10 GB free) |
| Gemini 1.5 Flash | $0.075/1M tokens (1,500 req/day free) |
| TTS, stock media, social APIs | Free |
| **Total** | **~$15–30/month** |
