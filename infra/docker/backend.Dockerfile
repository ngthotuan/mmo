# ─── Builder ──────────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Module cache layer — only invalidated when go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

# Pin goose so the layer is cached and we don't hit the network every build.
RUN mkdir -p /app/bin && \
    go install github.com/pressly/goose/v3/cmd/goose@v3.22.0 && \
    cp "$(go env GOPATH)/bin/goose" /app/bin/goose

COPY . .

# Build api + worker in parallel; both reuse the same module cache.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/api    ./cmd/api    & \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/worker ./cmd/worker & \
    wait

# ─── Shared runtime base ──────────────────────────────────────────────────────
FROM alpine:3.19 AS runtime-base

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/api     /app/bin/api
COPY --from=builder /app/bin/worker  /app/bin/worker
COPY --from=builder /app/bin/goose   /app/bin/goose
COPY --from=builder /build/internal/infrastructure/db/migrations /app/migrations
COPY --from=builder /build/config.yml /app/config.yml

# ─── API + general worker (no ffmpeg/python) ──────────────────────────────────
FROM runtime-base AS runtime-api

EXPOSE 8080
CMD ["/app/bin/api"]

# ─── Video worker (adds ffmpeg + edge-tts) ────────────────────────────────────
FROM runtime-base AS runtime-video

RUN apk add --no-cache \
    ffmpeg \
    font-noto \
    font-noto-cjk \
    python3 \
    py3-pip && \
    pip3 install --no-cache-dir edge-tts --break-system-packages

CMD ["/app/bin/worker", "--video-only"]
