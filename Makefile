.PHONY: dev dev-build down logs migrate-up migrate-down migrate-create \
        be-run be-worker be-build be-test be-vet fe-dev fe-build fe-install \
        config setup clean help

# ─── Local dev ────────────────────────────────────────────────────────────────

dev: ## Start all services (build if needed)
	docker-compose up --build

dev-bg: ## Start all services in background
	docker-compose up -d --build

dev-up: ## Start without rebuilding (fast — use when code hasn't changed)
	docker-compose up -d

down: ## Stop all services
	docker-compose down

logs: ## Tail logs for all services
	docker-compose logs -f

logs-api: ## Tail API logs
	docker-compose logs -f backend-api

logs-worker: ## Tail worker logs
	docker-compose logs -f backend-worker backend-video-worker

# ─── Database migrations ──────────────────────────────────────────────────────

migrate-up: ## Run all pending migrations
	docker-compose exec backend-api /app/bin/goose -dir /app/migrations postgres "$$DATABASE_URL" up

migrate-down: ## Rollback last migration
	docker-compose exec backend-api /app/bin/goose -dir /app/migrations postgres "$$DATABASE_URL" down

migrate-create: ## Create a new migration: make migrate-create name=add_products
	docker-compose exec backend-api /app/bin/goose -dir /app/migrations create $(name) sql

db-shell: ## Open psql shell (uses DATABASE_URL from .env)
	@export $$(grep -v '^#' .env | grep -v '^$$' | xargs) && psql "$$DATABASE_URL"

# ─── Backend ──────────────────────────────────────────────────────────────────

be-build: ## Build backend binaries locally
	cd backend && go build -o bin/api ./cmd/api && go build -o bin/worker ./cmd/worker

be-run: ## Run API server locally (requires postgres + redis running)
	cd backend && go run ./cmd/api

be-worker: ## Run worker locally
	cd backend && go run ./cmd/worker

be-test: ## Run backend tests
	cd backend && go test ./... -v -count=1

be-vet: ## Run go vet
	cd backend && go vet ./...

be-lint: ## Run golangci-lint
	cd backend && golangci-lint run ./...

config: ## Open backend/config.yml for editing
	$${EDITOR:-vi} backend/config.yml

# ─── Frontend ─────────────────────────────────────────────────────────────────

fe-install: ## Install frontend dependencies
	cd frontend && npm install

fe-dev: ## Run frontend dev server
	cd frontend && npm run dev

fe-build: ## Build frontend for production
	cd frontend && npm run build

fe-lint: ## Run frontend linter
	cd frontend && npm run lint

# ─── Setup ────────────────────────────────────────────────────────────────────

setup: ## First-time setup: copy env file
	@if [ ! -f .env ]; then cp .env.example .env && echo ".env created from .env.example — fill in your API keys"; fi

clean: ## Remove build artifacts and docker volumes
	rm -rf backend/bin frontend/.next frontend/node_modules
	docker compose down -v

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
