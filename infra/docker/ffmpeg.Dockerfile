FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/worker ./cmd/worker

FROM alpine:3.19

RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    ffmpeg \
    font-noto \
    font-noto-cjk \
    python3 \
    py3-pip && \
    pip3 install --no-cache-dir edge-tts --break-system-packages

WORKDIR /app

COPY --from=builder /app/bin/worker /app/bin/worker
COPY --from=builder /build/config.yml /app/config.yml

CMD ["/app/bin/worker", "--video-only"]
