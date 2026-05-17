# API Reference

Base URL: `http://localhost:8080` (dev) / `https://yourdomain.com` (prod)

All protected endpoints require:
```
Authorization: Bearer <access_token>
Content-Type: application/json
```

---

## Authentication

### POST /api/v1/auth/register

```json
// Request
{ "name": "John", "email": "john@example.com", "password": "secret123" }

// Response 201
{ "access_token": "...", "refresh_token": "...", "expires_in": 900 }
```

### POST /api/v1/auth/login

```json
// Request
{ "email": "john@example.com", "password": "secret123" }

// Response 200
{ "access_token": "...", "refresh_token": "...", "expires_in": 900 }
```

### POST /api/v1/auth/refresh

```json
// Request
{ "refresh_token": "..." }

// Response 200
{ "access_token": "...", "refresh_token": "...", "expires_in": 900 }
```

### GET /api/v1/auth/me ★

```json
// Response 200
{
  "id": "uuid",
  "email": "john@example.com",
  "name": "John",
  "role": "owner",
  "created_at": "2025-01-01T00:00:00Z"
}
```

### PUT /api/v1/auth/profile ★

```json
// Request
{ "name": "New Name" }

// Response 200
{ "message": "profile updated" }
```

### PUT /api/v1/auth/change-password ★

```json
// Request
{ "current_password": "old", "new_password": "new-secure-pass" }

// Response 200
{ "message": "password changed" }
```

---

## Channels

### GET /api/v1/channels ★

```json
// Response 200
[
  {
    "id": "uuid",
    "platform": "tiktok",
    "platform_user_id": "123",
    "username": "@handle",
    "display_name": "My Channel",
    "avatar_url": "https://...",
    "is_active": true,
    "created_at": "..."
  }
]
```

### GET /api/v1/channels/connect/:platform ★

`:platform` = `tiktok` | `facebook`

```json
// Response 200
{ "url": "https://tiktok.com/v2/auth/authorize?..." }
```

### POST /api/v1/channels/oauth/tiktok ★

```json
// Request
{ "code": "oauth-code-from-callback" }

// Response 201
{ "data": { ...channel } }
```

### POST /api/v1/channels/oauth/facebook ★

```json
// Request
{ "code": "oauth-code", "page_id": "facebook-page-id" }

// Response 201
{ "data": { ...channel } }
```

### GET /api/v1/channels/facebook/pages ★

```json
// Response 200
{ "data": [{ "id": "pageID", "name": "My Page", "access_token": "..." }] }
```

### DELETE /api/v1/channels/:id ★

```json
// Response 200
{ "message": "deleted" }
```

### PUT /api/v1/channels/:id/toggle ★

```json
// Response 200
{ "data": { ...channel, "is_active": false } }
```

---

## Trends & Content

### GET /api/v1/trends ★

Query params: `?page=1&perPage=20&status=new`

```json
// Response 200
{
  "data": [{
    "id": "uuid",
    "source": "youtube",
    "title": "Trending Topic",
    "keywords": ["keyword1", "keyword2"],
    "trending_score": 0.87,
    "status": "new",
    "discovered_at": "..."
  }],
  "pagination": { "page": 1, "per_page": 20, "total": 42 }
}
```

### POST /api/v1/trends/discover ★

Triggers immediate trend discovery (all sources). Returns HTTP 202.

```json
// Response 202
{ "message": "discovery queued" }
```

### GET /api/v1/content ★

Query params: `?page=1&perPage=20&status=draft`

```json
// Response 200
{
  "data": [{
    "id": "uuid",
    "title": "My Video",
    "niche": "fitness",
    "script": "...",
    "script_metadata": {
      "hook": "You won't believe...",
      "cta": "Follow for more",
      "hashtags": ["fitness", "health"],
      "caption": "..."
    },
    "status": "draft",
    "auto_approve": false,
    "created_at": "..."
  }],
  "pagination": { "page": 1, "per_page": 20, "total": 10 }
}
```

### POST /api/v1/content ★

```json
// Request
{ "trend_id": "uuid" }

// Response 201
{ "data": { ...content_plan } }
```

### PUT /api/v1/content/:id ★

```json
// Request
{
  "title": "Updated Title",
  "script": "Updated script...",
  "notes": "any notes"
}

// Response 200
{ "data": { ...content_plan } }
```

### POST /api/v1/content/:id/approve ★

Marks plan as approved and enqueues video pipeline.

```json
// Response 200
{ "message": "approved" }
```

### POST /api/v1/content/:id/generate-script ★

Regenerates the AI script.

```json
// Response 200
{ "data": { ...content_plan } }
```

### DELETE /api/v1/content/:id ★

```json
// Response 200
{ "message": "deleted" }
```

---

## Videos

### GET /api/v1/videos ★

Query params: `?page=1&perPage=20&status=done`

Status values: `pending` | `media_collecting` | `tts_generating` | `assembling` | `uploading` | `done` | `failed`

```json
// Response 200
{
  "data": [{
    "id": "uuid",
    "content_plan_id": "uuid",
    "status": "done",
    "output_video_url": "https://pub-xxx.r2.dev/videos/xxx/output.mp4",
    "duration_seconds": 45,
    "file_size_bytes": 12345678,
    "retry_count": 0,
    "error_message": "",
    "created_at": "..."
  }],
  "total": 5
}
```

### GET /api/v1/videos/:id ★

```json
// Response 200
{ "data": { ...video_job } }
```

### GET /api/v1/videos/:id/download ★

Returns a presigned R2 download URL (1-hour TTL).

```json
// Response 200
{ "url": "https://r2-presigned-url?..." }
```

### POST /api/v1/videos/:id/retry ★

Re-queues a failed video job.

```json
// Response 200
{ "message": "queued" }
```

### DELETE /api/v1/videos/:id ★

```json
// Response 200
{ "message": "deleted" }
```

### GET /api/v1/templates ★

```json
// Response 200
{
  "data": [{
    "id": "uuid",
    "name": "Slideshow",
    "type": "slideshow",
    "config": { "transition": "fade", "duration_per_slide": 3 }
  }]
}
```

---

## Publishing

### GET /api/v1/publish ★

Query params: `?page=1&perPage=20&status=scheduled`

Status values: `scheduled` | `publishing` | `published` | `failed` | `cancelled`

```json
// Response 200
{
  "data": [{
    "id": "uuid",
    "video_job_id": "uuid",
    "channel_id": "uuid",
    "platform": "tiktok",
    "caption": "Check this out!",
    "hashtags": ["viral", "tips"],
    "scheduled_at": "2025-06-01T10:00:00Z",
    "published_at": null,
    "platform_post_id": "",
    "platform_post_url": "",
    "status": "scheduled",
    "error_message": "",
    "created_at": "..."
  }],
  "total": 3
}
```

### POST /api/v1/publish ★

```json
// Request
{
  "video_job_id": "uuid",
  "channel_id": "uuid",
  "caption": "My video caption",
  "hashtags": ["tag1", "tag2"],
  "scheduled_at": "2025-06-01T10:00:00Z"  // omit for immediate
}

// Response 201
{ "data": { ...publish_job } }
```

### PUT /api/v1/publish/:id ★

Update caption, hashtags, or reschedule.

```json
// Request
{
  "caption": "New caption",
  "hashtags": ["new", "tags"],
  "scheduled_at": "2025-06-02T10:00:00Z"
}

// Response 200
{ "data": { ...publish_job } }
```

### POST /api/v1/publish/:id/publish-now ★

Triggers immediate publish regardless of `scheduled_at`.

```json
// Response 200
{ "message": "publish queued" }
```

### DELETE /api/v1/publish/:id ★

Cancels a scheduled job.

```json
// Response 200
{ "message": "cancelled" }
```

### GET /api/v1/calendar ★

Query params: `?start=2025-06-01T00:00:00Z&end=2025-06-30T23:59:59Z` (RFC3339)

```json
// Response 200
{ "data": [{ ...publish_job }, ...] }
```

---

## Products (Shop Catalog)

### GET /api/v1/products ★

Query params: `?platform=tiktok&page=1&perPage=20`

```json
// Response 200
{
  "data": [{
    "id": "uuid",
    "platform": "tiktok",
    "platform_product_id": "tiktok-shop-product-id",
    "name": "Product Name",
    "description": "...",
    "price": 29.99,
    "currency": "USD",
    "cover_image_url": "https://...",
    "product_url": "https://...",
    "status": "active",
    "synced_at": "..."
  }],
  "total": 12
}
```

### POST /api/v1/products ★

```json
// Request
{
  "platform": "tiktok",
  "platform_product_id": "shop-product-id",
  "name": "Product Name",
  "description": "Description",
  "price": 29.99,
  "currency": "USD",
  "cover_image_url": "https://...",
  "product_url": "https://..."
}

// Response 201
{ "data": { ...product } }
```

### POST /api/v1/products/sync ★

Syncs products from TikTok Shop or Facebook Catalog.

```json
// Request (TikTok)
{
  "platform": "tiktok",
  "channel_id": "uuid"
}

// Request (Facebook)
{
  "platform": "facebook",
  "channel_id": "uuid",
  "catalog_id": "facebook-catalog-id"
}

// Response 200
{ "synced": 24 }
```

### DELETE /api/v1/products/:id ★

```json
// Response 200
{ "message": "deleted" }
```

### GET /api/v1/publish/:id/products ★

```json
// Response 200
{ "data": [{ ...product }, ...] }
```

### POST /api/v1/publish/:id/products ★

Tags products on a publish job (included in TikTok `product_links` / Facebook description).

```json
// Request
{ "product_ids": ["uuid1", "uuid2"] }

// Response 200
{ "message": "products tagged" }
```

---

## Analytics

### GET /api/v1/analytics/overview ★

Query params: `?days=30`

```json
// Response 200
{
  "total_views": 125000,
  "total_likes": 4200,
  "total_comments": 380,
  "total_shares": 220,
  "post_count": 15
}
```

### GET /api/v1/analytics/posts ★

Query params: `?page=1&perPage=20`

```json
// Response 200
{
  "data": [{
    "publish_job_id": "uuid",
    "platform": "tiktok",
    "views": 15000,
    "likes": 430,
    "comments": 22,
    "shares": 18,
    "synced_at": "..."
  }],
  "pagination": { "page": 1, "per_page": 20, "total": 15 }
}
```

---

## Pipeline Status

### GET /api/v1/pipeline/status ★

```json
// Response 200
{
  "video_status_counts": {
    "pending": 1,
    "assembling": 2,
    "done": 47,
    "failed": 3
  },
  "active_jobs": [
    { "id": "uuid", "status": "assembling", "created": "..." }
  ],
  "total_videos": 53
}
```

---

## Health Check

### GET /health

No auth required.

```json
// Response 200
{ "status": "ok", "db": "ok" }
```

---

## Error Responses

All errors follow:

```json
{
  "code": 404,
  "message": "not found"
}
```

| HTTP Code | Meaning |
|---|---|
| 400 | Bad request / validation error |
| 401 | Missing or invalid JWT |
| 403 | Forbidden |
| 404 | Resource not found |
| 409 | Conflict (e.g., duplicate email) |
| 500 | Internal server error |
