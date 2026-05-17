FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

# Install goose before copying source so it's cached independently
RUN mkdir -p /app/bin && \
    go install github.com/pressly/goose/v3/cmd/goose@latest && \
    cp "$(go env GOPATH)/bin/goose" /app/bin/goose

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/api    ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/worker ./cmd/worker

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/api     /app/bin/api
COPY --from=builder /app/bin/worker  /app/bin/worker
COPY --from=builder /app/bin/goose   /app/bin/goose
COPY --from=builder /build/internal/infrastructure/db/migrations /app/migrations
COPY --from=builder /build/config.yml /app/config.yml

EXPOSE 8080

CMD ["/app/bin/api"]
