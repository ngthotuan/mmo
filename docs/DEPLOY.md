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
3. SSH vào server:
     - Pull new images
     - Checkout new tag
     - Run goose migration with NEW image
     - docker compose up -d (swap containers)
     - Health check 30s
     - Rollback nếu fail
```

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

## 3. Setup server lần đầu

SSH vào server, chạy:

```bash
# A. Install Docker (Ubuntu 22.04+)
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER   # logout + login lại để có quyền docker

# B. Clone repo
sudo mkdir -p /opt/mmo
sudo chown $USER:$USER /opt/mmo
cd /opt/mmo
git clone https://github.com/ngthotuan/mmo.git .

# C. Tạo .env (copy từ .env.example, fill secrets thật)
cp .env.example .env
nano .env
# Phải có DATABASE_URL, REDIS_URL, JWT_SECRET, ENCRYPTION_KEY, các API key...

# D. Login GHCR để pull image (cần GitHub PAT với scope read:packages)
echo $GHCR_PAT | docker login ghcr.io -u ngthotuan --password-stdin

# E. Khởi tạo lần đầu — dùng version mới nhất
echo "APP_VERSION=latest" >> .env
echo "IMAGE_OWNER=ngthotuan" >> .env
docker compose pull
docker compose up -d

# F. Verify
curl http://localhost:8080/health
docker compose ps
```

---

## 4. Tạo GitHub Environment `production`

Bắt buộc để có manual approval gate:

**Repo Settings → Environments → New environment → `production`**

- **Required reviewers:** thêm bạn (và team) — bắt buộc click `Approve` trước khi deploy chạy
- **Deployment branches and tags:** `Selected tags only` → pattern `v*.*.*`
- (Optional) **Wait timer:** vd 5 phút buffer trước deploy

> Secrets ở mục 1 có thể đặt ở **Environment level** thay vì Repo level để chặt chẽ hơn — chỉ workflow chạy environment này mới đọc được.

---

## 5. Tạo PAT cho server pull GHCR

GHCR là private mặc định → server cần token để pull.

**GitHub → Settings (user) → Developer settings → Personal access tokens → Tokens (classic) → Generate new**

- Note: `mmo-server-pull`
- Expiration: 1 year (hoặc no expiration nếu chấp nhận risk)
- Scopes: chỉ tick `read:packages`

Copy token, dùng làm `$GHCR_PAT` ở bước 3.D.

---

## 6. Release flow hàng ngày

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

## 7. Manual rollback (khi cần)

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

## 8. Troubleshooting

| Triệu chứng | Nguyên nhân thường gặp |
|---|---|
| Workflow stuck ở "Waiting for approval" | Reviewer chưa Approve trên tab Actions |
| `permission denied (publickey)` | SSH key chưa copy lên `~/.ssh/authorized_keys` của user trên server, hoặc `DEPLOY_USER` sai |
| `docker compose pull` 401 unauthorized | Server chưa `docker login ghcr.io` hoặc PAT hết hạn |
| Goose migration fail | `DATABASE_URL` trong `.env` server sai, hoặc DB không reachable từ container |
| Health check timeout → auto rollback | API container không start, kiểm tra `docker compose logs backend-api` |
