# @telefraudbot — development & operations targets.

.DEFAULT_GOAL := help

# --- Run ---
.PHONY: dev
dev: ## Run the bot locally (reads .env from the working directory)
	@go run ./cmd/telefraud || true

.PHONY: build
build: ## Compile a binary to ./bin/telefraud
	go build -trimpath -o bin/telefraud ./cmd/telefraud

# --- Verify ---
.PHONY: test
test: ## Run the test suite
	go test ./...

.PHONY: vet
vet: ## Vet the code
	go vet ./...

.PHONY: fmt
fmt: ## Format all Go files
	gofmt -w .

# --- Docker ---
.PHONY: docker-up
docker-up: ## Start Postgres + bot (schema auto-migrates)
	docker compose up -d --build

.PHONY: docker-db
docker-db: ## Start Postgres only (for local psql / dev against compose db)
	docker compose up -d db

.PHONY: docker-down
docker-down: ## Stop the stack
	docker compose down

.PHONY: docker-logs
docker-logs: ## Tail bot logs
	docker compose logs -f bot

# --- Help ---
.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'
