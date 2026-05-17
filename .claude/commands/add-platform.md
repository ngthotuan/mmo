Add support for a new social media publishing platform (e.g., Instagram, YouTube Shorts, X/Twitter).

The user will specify the platform name and API details. Follow this checklist:

## Backend

### 1. Integration client
Create `backend/internal/integration/<platform>/client.go`:
- `New(cfg config.<Platform>Config) *Client`
- `AuthURL(state string) string` — OAuth redirect
- `ExchangeCode(ctx, code) (*Tokens, error)` — code → tokens
- `RefreshToken(ctx, refreshToken) (*Tokens, error)` — token refresh
- `GetUserInfo(ctx, accessToken) (*UserProfile, error)` — profile for channel creation
- `PostVideo(ctx, accessToken, req PostVideoRequest) (string, error)` — publish
- `GetVideoStats(ctx, accessToken, videoID) (*VideoStats, error)` — analytics

### 2. Config
Add `<Platform>Config` struct to `backend/pkg/config/config.go`:
```go
type <Platform>Config struct {
    AppID       string
    AppSecret   string
    RedirectURL string
}
```
Load from env in `Load()`:
```go
<Platform>: <Platform>Config{
    AppID:       getEnv("<PLATFORM>_APP_ID", ""),
    AppSecret:   getEnv("<PLATFORM>_APP_SECRET", ""),
    RedirectURL: frontendURL + "/channels/callback/<platform>",
},
```

### 3. Channel handler
Add OAuth endpoints to `backend/internal/adapter/handler/channel_handler.go`:
- `POST /api/v1/channels/oauth/<platform>` — code exchange + channel creation
The `platform` field in the `channels` table is a free-form string — just use the platform name.

### 4. Publish worker
Add a new case to the switch in `backend/internal/adapter/worker/task_publish.go`:
```go
case "<platform>":
    postID, err = h.<platform>.PostVideo(ctx, accessToken, <platform>.PostVideoRequest{...})
```
Add the new client to `PublishHandler` struct and constructor.

### 5. Analytics sync
Add a case in `backend/internal/adapter/worker/task_sync_analytics.go` for the new platform.

### 6. Wire everything
- `cmd/api/main.go`: instantiate client, pass to channelUC, add OAuth routes
- `cmd/worker/main.go`: instantiate client, pass to publishHandler and analyticsSyncHandler

### 7. Verify
```bash
cd backend && go build ./... && go vet ./...
```

## Frontend

### 1. Channel connect button
In `frontend/src/app/(dashboard)/channels/page.tsx`, add a new connect button for the platform. Follow the same pattern as TikTok/Facebook.

### 2. Platform type
Add the platform to the `Platform` union in `frontend/src/lib/types/api.types.ts`:
```ts
export type Platform = "tiktok" | "facebook" | "<platform>";
```

### 3. Verify
```bash
cd frontend && npx tsc --noEmit
```

## Checklist before done
- [ ] OAuth flow tested end-to-end (connect → channel appears in list)
- [ ] PostVideo tested (schedule → publish queue → publish now)
- [ ] Analytics sync tested (POST /api/v1/trends/discover equivalent test)
- [ ] go build + go vet pass
- [ ] tsc --noEmit passes
