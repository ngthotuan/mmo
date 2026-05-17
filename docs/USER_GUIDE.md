# User Guide

## Overview

AutoContent automates the full TikTok/Facebook video pipeline:

```
Discover Trends → Review Script → Video Assembles → Schedule → Publish → Analyze
```

---

## 1. First-Time Setup

### 1.1 Create account

Go to the app URL → **Register** → enter name, email, password.

### 1.2 Connect your channels

**Channels → Connect**

- **TikTok:** Click "Connect TikTok" → authorize in TikTok → redirected back with channel active
- **Facebook:** Click "Connect Facebook" → authorize → select which Facebook Page to use

You can connect multiple accounts (e.g., 3 TikTok accounts + 2 Facebook pages). Toggle individual channels on/off without disconnecting them.

---

## 2. Content Pipeline

### 2.1 Discover trends

**Content → Discover Trends**

Click **Discover** to trigger an immediate scan across Google Trends, YouTube, and Reddit. Trends are also auto-discovered every 6 hours.

Each trend shows:
- Source (YouTube / Google Trends / Reddit)
- Title and keywords
- Trending score

### 2.2 Create a content plan from a trend

Click **Create Plan** on any trend. The system calls Gemini 1.5 Flash to generate:
- Video title
- Hook (opening line to grab attention)
- Full script body
- Call-to-action
- Hashtags and caption

### 2.3 Review and approve

Open a content plan to:
- Edit the title, script, or notes
- Click **Regenerate** to get a new AI script
- Click **Approve** to start video assembly
- Click **Reject** to discard

When you approve a plan, the video pipeline starts automatically (collecting media → TTS → FFmpeg → upload to R2).

---

## 3. Video Pipeline

### 3.1 Monitor progress

**Videos** page shows all jobs with live status:

| Status | Meaning |
|---|---|
| pending | Waiting to start |
| media_collecting | Downloading stock footage from Pexels/Pixabay |
| tts_generating | Generating voiceover with Edge TTS |
| assembling | FFmpeg building the 1080×1920 video |
| uploading | Uploading to Cloudflare R2 |
| done | Video ready |
| failed | Error — see error message |

The page auto-refreshes every 10 seconds.

### 3.2 Retry failed videos

If a video fails, click **Retry** to re-queue it. Check the error message first — common causes:
- Pexels/Pixabay rate limit: wait 1 hour
- FFmpeg error: check server disk space
- R2 credentials wrong: check environment variables

### 3.3 Preview and download

On a `done` video:
- Click the thumbnail to preview in a modal
- Click **Download** to get a presigned R2 URL (1-hour link)

---

## 4. Scheduling & Publishing

### 4.1 Schedule a video

From **Videos** page, click **Schedule** on a `done` video, or go to **Publish → New**.

Fill in:
- **Channel** — which TikTok/Facebook account to post to
- **Caption** — text description
- **Hashtags** — add without the # sign
- **Scheduled At** — datetime to publish (leave blank = publish immediately)

### 4.2 Tag products (optional)

After creating a publish job, click the **Products** button on the job card to open the product picker. Select products from your catalog — they will be embedded in the post:
- **TikTok:** as product links (TikTok Shop shopping bag icon)
- **Facebook:** as product URLs appended to the description

### 4.3 Publish Now

On any `scheduled` or `failed` job, click **Publish Now** to bypass the schedule and publish immediately.

### 4.4 Content calendar

**Schedule** page shows a month-view calendar with all scheduled and published posts. Click any event to see details and publish/cancel options.

---

## 5. Analytics

**Analytics** page shows performance data synced daily at 2 AM:

### Overview cards
- Total views, likes, comments, shares across all posts
- Post count in the selected period

### Per-post table
Shows each published post's metrics broken down by platform.

> Data is synced once per day. For real-time metrics, check TikTok/Facebook natively.

---

## 6. Product Catalog

Use **Products** to manage your shop catalog for video product tagging.

### Manual add

Click **Add Product** → fill in:
- Platform (TikTok / Facebook)
- Platform Product ID (your shop's product ID)
- Name, description, price, currency
- Product URL (for Facebook: full URL; for TikTok: product ID from TikTok Shop)
- Cover image URL

### Sync from shop

Click **Sync from Shop**:
- **TikTok Shop:** select the TikTok channel connected to your shop → syncs up to 100 products
- **Facebook Catalog:** select the Facebook channel + enter your Catalog ID from Meta Business Manager → syncs up to 100 products

> TikTok Shop sync requires your TikTok account to have an active TikTok Shop and shop API credentials configured by the server admin.

---

## 7. Settings

**Settings** page:

- **Profile:** Update display name
- **Change Password:** Requires current password + new password (min 8 characters)
- **Account Info:** View your role, member since date, user ID

---

## 8. Tips & Best Practices

### Maximize automation

Enable **Auto-approve** on content plans to skip the review step. Approved plans → video pipeline starts immediately.

### Multi-channel strategy

Create one content plan per trend, then schedule it to multiple channels with different captions/hashtags per channel. Each publish job is independent.

### Content cadence

Trends are discovered every 6 hours. For high-volume posting:
1. Let the system discover trends overnight
2. Bulk-approve 5–10 plans in the morning
3. Videos assemble during the day
4. Schedule them across the week from the calendar

### Product tagging best practices

- **TikTok:** Link products with valid TikTok Shop product IDs for the shop bag icon to appear
- **Facebook:** Include product URLs in the description — they become clickable links in the post
- Tag 1–3 products per video for best engagement; too many dilutes the CTA

### FFmpeg video quality

Three templates available:
- **Slideshow** — best for informational content (images + narration)
- **B-roll** — best for dynamic content (video clips + voiceover)
- **Text on Video** — best for motivational/quote content

The template is chosen automatically based on what media is found for the script topic.

---

## 9. Notifications & Monitoring

The **Dashboard** shows:
- Summary stats (channels, content plans, videos done, posts published)
- **Pipeline Status** — live active jobs with status badges, auto-refreshes every 15 seconds

If a publish job fails, the error message is shown on the job card in **Publish Queue**. Common failures:
- `video not ready: no output URL` — video assembly not complete yet
- `decrypt token: ...` — channel token may be corrupted; reconnect the channel
- `tiktok post video error` — TikTok API error; check TikTok developer console
