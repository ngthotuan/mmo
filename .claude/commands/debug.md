Diagnose a reported issue with the platform.

The user will describe the symptom. Follow this diagnostic process:

## 1. Identify the layer

| Symptom | Where to look |
|---|---|
| Video stuck in a status | Worker logs: `make logs-worker` |
| Publish job failing | `make logs-worker` + `publish_jobs.error_message` in DB |
| API returning wrong data | Handler + usecase logic |
| Frontend not showing data | Browser network tab + check API response shape |
| OAuth not working | Check `FRONTEND_URL` matches redirect URI in TikTok/Facebook console |
| R2 video not playing | Check `R2_PUBLIC_URL` + bucket public access enabled |

## 2. Check logs

```bash
make logs-api            # API server errors
make logs-worker         # background task errors
make db-shell            # then query relevant table
```

Useful DB queries:
```sql
-- Recent failed video jobs
SELECT id, status, error_message, updated_at FROM video_jobs
WHERE status = 'failed' ORDER BY updated_at DESC LIMIT 10;

-- Recent failed publish jobs
SELECT id, status, error_message, platform, updated_at FROM publish_jobs
WHERE status = 'failed' ORDER BY updated_at DESC LIMIT 10;

-- Stuck video jobs (in non-terminal status > 30 minutes)
SELECT id, status, created_at FROM video_jobs
WHERE status NOT IN ('done','failed') AND created_at < NOW() - INTERVAL '30 minutes';

-- Check channels with expired tokens
SELECT id, platform, username, token_expires_at FROM channels
WHERE token_expires_at < NOW();
```

## 3. Common root causes

**Videos stuck assembling:**
- Check `backend-video-worker` container is running: `docker compose ps`
- Check disk space: `df -h`
- Check ffmpeg available: `docker compose exec backend-video-worker ffmpeg -version`
- Check edge-tts: `docker compose exec backend-video-worker edge-tts --version`

**Publish jobs failing with "decrypt token":**
- `ENCRYPTION_KEY` was changed after channels were connected
- Fix: user must disconnect and reconnect the channel

**TikTok publish error:**
- Check TikTok API sandbox vs production mode in developer console
- `video.publish` scope requires app review by TikTok

**Facebook publish error:**
- Page token may be expired (60-day long-lived tokens)
- `pages_manage_posts` permission may not be approved

**Gemini returning empty scripts:**
- Check `GEMINI_API_KEY` is set
- Check free tier quota (1,500 req/day): https://aistudio.google.com

## 4. Report findings

State:
1. What the root cause is
2. Which file/line needs changing (if code bug)
3. Which config/external system needs fixing (if config issue)
4. Proposed fix
