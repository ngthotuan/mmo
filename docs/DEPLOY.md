# Deployment Setup

Production deploy chạy tự động khi push tag `v*.*.*`. Flow:

```
git tag v1.2.3 && git push origin v1.2.3
         ↓
release.yml triggers
         ↓
1. Build & push 3 images lên GHCR (parallel)
         ↓
2. Wait for manual approval (GitHub Environments → production)
         ↓
3. Copy `docker-compose.yml` + `infra/nginx/nginx.conf` lên server (scp)
         ↓
4. SSH vào server:
     - Pull new images
     - Run goose migration with NEW image (chạy bằng user trong DATABASE_URL)
     - docker compose up -d (swap containers)
     - Health check 30s
     - Rollback nếu fail
```

> **Server KHÔNG cần là git repo.** CI copy thẳng `docker-compose.yml` và
> `infra/nginx/nginx.conf` (file nginx được mount bởi compose) từ runner lên
> `DEPLOY_PATH` ở mỗi lần deploy — không chạy `git` trên server.

---

## 1. GitHub Secrets cần tạo

**Repo Settings → Secrets and variables → Actions → New repository secret**

| Secret name | Value | Note |
|---|---|---|
| `DEPLOY_HOST` | IP/domain server | vd `123.45.67.89` |
| `DEPLOY_USER` | SSH user | vd `deploy`, `ubuntu` |
| `DEPLOY_SSH_KEY` | Private key (full content) | Xem mục 2 để tạo |
| `DEPLOY_PORT` | SSH port (optional) | Default 22 |
| `DEPLOY_PATH` | Path repo trên server | vd `/opt/mmo` |

---

## 2. Tạo SSH key cho CI (chạy ở máy local)

```bash
ssh-keygen -t ed25519 -C "github-ci@mmo" -f ~/.ssh/mmo_deploy -N ""

# Copy public key lên server:
ssh-copy-id -i ~/.ssh/mmo_deploy.pub deploy@<DEPLOY_HOST>

# Lấy private key, paste vào GitHub Secret DEPLOY_SSH_KEY:
cat ~/.ssh/mmo_deploy
```

Test SSH thử:
```bash
ssh -i ~/.ssh/mmo_deploy deploy@<DEPLOY_HOST> 'whoami && docker --version'
```

---

## 3. Chuẩn bị Database (Postgres) — BẮT BUỘC

> ⚠️ Migration bootstrap role/grant đã bị **bỏ** khỏi codebase. Goose chạy bằng
> chính **user trong `DATABASE_URL`**, nên user đó phải **sở hữu database** (để
> `CREATE/ALTER TABLE`), và extension **`pgcrypto` phải được superuser tạo sẵn 1 lần**
> (`CREATE EXTENSION` cần quyền superuser). Làm 1 lần duy nhất khi khởi tạo.

Kết nối vào Postgres bằng tài khoản **superuser** (vd `postgres`) rồi chạy:

```sql
-- 1. Tạo role ứng dụng (đổi mật khẩu cho khớp DATABASE_URL trong .env)
CREATE ROLE mmo WITH LOGIN PASSWORD 'CHANGE_ME_STRONG';

-- 2. Tạo database do role mmo sở hữu → mọi bảng sẽ thuộc mmo, không cần GRANT
CREATE DATABASE mmo OWNER mmo;

-- 3. Cài pgcrypto trong DB mmo (chạy với superuser, kết nối vào DB mmo)
\c mmo
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
```

`DATABASE_URL` trong `.env` server phải trỏ đúng role/DB này, vd:
`postgres://mmo:CHANGE_ME_STRONG@<db-host>:5432/mmo?sslmode=disable`

Migration sẽ tự chạy ở bước deploy (job `release.yml`) bằng image mới. Muốn chạy tay:

```bash
docker compose run --rm --no-deps backend-api \
  sh -c '/app/bin/goose -dir /app/migrations postgres "$DATABASE_URL" up'
```

---

## 4. Setup server lần đầu

SSH vào server, chạy:

```bash
# A. Install Docker (Ubuntu 22.04+)
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER   # logout + login lại để có quyền docker

# B. Tạo thư mục deploy (KHÔNG cần git clone — CI sẽ scp compose + nginx vào đây)
sudo mkdir -p /opt/mmo
sudo chown $USER:$USER /opt/mmo
cd /opt/mmo

# B2. Đặt sẵn compose + nginx cho lần bring-up đầu tiên (trước deploy CI đầu tiên).
#     Cách nhanh: tải đúng 2 file từ repo tại tag/branch (hoặc scp tay từ máy bạn).
#     Cấu trúc cần có: ./docker-compose.yml và ./infra/nginx/nginx.conf
mkdir -p infra/nginx
curl -fsSL -o docker-compose.yml \
  https://raw.githubusercontent.com/ngthotuan/mmo/main/docker-compose.yml
curl -fsSL -o infra/nginx/nginx.conf \
  https://raw.githubusercontent.com/ngthotuan/mmo/main/infra/nginx/nginx.conf
# (Nếu repo private: scp 2 file này từ máy local lên, hoặc git clone 1 lần.)

# C. Tạo .env (copy từ .env.example, fill secrets thật) — xem mục 5 cho biến mới
cp .env.example .env   # hoặc tạo mới
nano .env

# D. Login GHCR để pull image (cần GitHub PAT với scope read:packages)
echo $GHCR_PAT | docker login ghcr.io -u ngthotuan --password-stdin

# E. Khởi tạo lần đầu — dùng version mới nhất
echo "APP_VERSION=latest"     >> .env
echo "IMAGE_OWNER=ngthotuan"  >> .env
docker compose pull
# Migrate trước khi up (DB đã chuẩn bị ở mục 3):
docker compose run --rm --no-deps backend-api \
  sh -c '/app/bin/goose -dir /app/migrations postgres "$DATABASE_URL" up'
docker compose up -d

# F. Verify
curl http://localhost:8080/health
docker compose ps
```

---

## 5. Biến môi trường `.env` cho production

Bắt buộc: `DATABASE_URL`, `REDIS_URL`, `JWT_SECRET`, `ENCRYPTION_KEY` (đúng 32 byte),
`APP_VERSION`, `IMAGE_OWNER`.

Quan trọng cho prod (khác với khi test local):

```env
APP_ENV=production
FRONTEND_URL=https://your-domain.com        # dùng cho CORS + OAuth redirect + nginx origin
NEXT_PUBLIC_API_URL=https://your-domain.com # browser gọi API qua đây (build-arg, xem ghi chú)

# AI: dùng Gemini thật ở prod (KHÔNG để mock)
AI_PROVIDER=gemini
AI_FALLBACK_TO_MOCK=true
GEMINI_MODEL=gemini-2.5-flash
GEMINI_API_KEY=...

# Publish THẬT ở prod — phải TẮT dry-run (để trống hoặc false)
PUBLISH_DRY_RUN=false

# Social OAuth (redirect URL được build từ FRONTEND_URL trong config.yml)
TIKTOK_CLIENT_KEY=...
TIKTOK_CLIENT_SECRET=...
FACEBOOK_APP_ID=...
FACEBOOK_APP_SECRET=...
YOUTUBE_OAUTH_CLIENT_ID=...        # YouTube Shorts publishing (Google OAuth)
YOUTUBE_OAUTH_CLIENT_SECRET=...
YOUTUBE_PRIVACY_STATUS=public      # public | unlisted | private
```

OAuth redirect URLs phải khai báo ở từng platform console, dạng:
`${FRONTEND_URL}/channels/callback/{tiktok|facebook|youtube}`.

> **Lưu ý `NEXT_PUBLIC_API_URL`:** biến này được inline lúc **build image frontend**
> (qua build-arg trong `docker-compose.yml`). Image GHCR phát hành bởi `release.yml`
> build **không** truyền domain của bạn → mặc định `http://localhost`. Để frontend
> gọi đúng API domain prod, hoặc (a) build image frontend riêng với
> `--build-arg NEXT_PUBLIC_API_URL=https://your-domain.com`, hoặc (b) để frontend
> gọi API **same-origin qua nginx** (giữ default `http://localhost` → đổi thành
> domain qua reverse proxy/nginx serve cùng host). Cách đơn giản nhất: truy cập app
> qua nginx (:80/:443) cùng domain, API ở `/api` → không cần sửa gì.

---

## 6. Tạo GitHub Environment `production`

Bắt buộc để có manual approval gate:

**Repo Settings → Environments → New environment → `production`**

- **Required reviewers:** thêm bạn (và team) — bắt buộc click `Approve` trước khi deploy chạy
- **Deployment branches and tags:** `Selected tags only` → pattern `v*.*.*`
- (Optional) **Wait timer:** vd 5 phút buffer trước deploy

> Secrets ở mục 1 có thể đặt ở **Environment level** thay vì Repo level để chặt chẽ hơn — chỉ workflow chạy environment này mới đọc được.

---

## 7. Tạo PAT cho server pull GHCR

GHCR là private mặc định → server cần token để pull.

**GitHub → Settings (user) → Developer settings → Personal access tokens → Tokens (classic) → Generate new**

- Note: `mmo-server-pull`
- Expiration: 1 year (hoặc no expiration nếu chấp nhận risk)
- Scopes: chỉ tick `read:packages`

Copy token, dùng làm `$GHCR_PAT` ở bước 4.D.

---

## 8. Release flow hàng ngày

```bash
# Sau khi merge PR vào main, đảm bảo CI passes
git checkout main
git pull

# Tạo tag mới
git tag v1.2.3
git push origin v1.2.3

# Mở https://github.com/<owner>/mmo/actions
# → Release workflow → Click `Review deployments` → Approve
# → Deploy job tiếp tục
# → Health check sau 30s → done
```

---

## 9. Manual rollback (khi cần)

Nếu auto-rollback của workflow không kick in (vd deploy chạy xong, sau đó vài giờ mới phát hiện bug), SSH vào server:

```bash
cd /opt/mmo

# Pin lại version cũ
sed -i '/^APP_VERSION=/d' .env
echo "APP_VERSION=1.2.2" >> .env

# Pull và swap
docker compose pull
docker compose up -d

# Health check
curl http://localhost:8080/health
```

Lưu ý: rollback **không tự revert migration**. Nếu migration của v1.2.3 đã chạy mà schema không backward-compatible, cần `goose down` manual.

---

## 10. Troubleshooting

| Triệu chứng | Nguyên nhân thường gặp |
|---|---|
| Workflow stuck ở "Waiting for approval" | Reviewer chưa Approve trên tab Actions |
| `permission denied (publickey)` | SSH key chưa copy lên `~/.ssh/authorized_keys` của user trên server, hoặc `DEPLOY_USER` sai |
| `docker compose pull` 401 unauthorized | Server chưa `docker login ghcr.io` hoặc PAT hết hạn |
| `fatal: not a git repository` ở bước deploy | Phiên bản workflow cũ chạy `git` trên server. Đã fix: CI scp compose + nginx, không dùng git. Đảm bảo `release.yml` đã cập nhật. |
| Goose migration fail | `DATABASE_URL` trong `.env` server sai, hoặc DB không reachable từ container |
| `permission denied for table ...` / `must be owner of table` khi migrate | DB **không** do user trong `DATABASE_URL` sở hữu. Tạo lại theo mục 3 (`CREATE DATABASE mmo OWNER mmo`). |
| `extension "pgcrypto" ... permission denied` | Chưa tạo `pgcrypto` bằng superuser. Xem mục 3 bước 3. |
| nginx container exit / `host not found` | Thiếu `infra/nginx/nginx.conf` ở `DEPLOY_PATH`. CI scp tự copy; lần đầu xem mục 4 B2. |
| Health check timeout → auto rollback | API container không start, kiểm tra `docker compose logs backend-api` |
