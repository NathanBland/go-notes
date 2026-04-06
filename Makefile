SHELL := /bin/sh

GO_VERSION_PKG := go
SQLC_INSTALL_PKG := github.com/sqlc-dev/sqlc/cmd/sqlc@latest
GORELEASER_PKG := github.com/goreleaser/goreleaser/v2@latest
DOCKER_COMPOSE ?= $(shell if docker compose version >/dev/null 2>&1; then printf 'docker compose'; elif command -v docker-compose >/dev/null 2>&1; then printf 'docker-compose'; fi)
COMPOSE_FILE ?= docker-compose.yml
COMPOSE_PROJECT_NAME ?= go-notes

POSTGRES_USER ?= postgres
POSTGRES_PASSWORD ?= postgres
POSTGRES_DB ?= go_notes
POSTGRES_HOST ?= localhost
POSTGRES_PORT ?= 5432
VALKEY_HOST ?= 127.0.0.1
VALKEY_PORT ?= 6379

DATABASE_URL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable
MIGRATE_DATABASE_URL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@postgres:5432/$(POSTGRES_DB)?sslmode=disable
MIGRATIONS_DIR ?= migrations
SQLC_CONFIG ?= sqlc.yaml
APP_ADDR ?= :8080
NAME ?=
CACHE_DIR ?= .cache
GO_BUILD_CACHE ?= $(abspath $(CACHE_DIR)/go-build)
GO_MOD_CACHE ?= $(abspath $(CACHE_DIR)/mod)
COVERAGE_DIR ?= $(CACHE_DIR)/coverage
COVERAGE_PROFILE ?= $(COVERAGE_DIR)/coverage.out
RAW_COVERAGE_PROFILE ?= $(COVERAGE_DIR)/coverage-raw.out
COVERAGE_THRESHOLD ?= 80.0
LOAD_LOCAL_ENV = set -a; [ ! -f ./.env ] || . ./.env; [ ! -f ./.env.local ] || . ./.env.local; set +a

.PHONY: help init check-deps deps install-go install-sqlc check-docker-tools check-docker-daemon \
	go-mod-tidy sqlc-generate docker-up docker-up-app docker-down docker-logs docker-logs-api \
	migrate-create migrate-up migrate-down fmt test test-integration coverage coverage-integration \
	coverage-check coverage-check-integration release-check-mcp release-snapshot-mcp run run-mcp

help:
	@printf '%s\n' \
		'Targets:' \
		'  make check-deps      Check whether local tools are available' \
		'  make deps            Install missing local Go/sqlc dependencies' \
		'  make docker-up       Start PostgreSQL and Valkey in Docker' \
		'  make docker-up-app   Start the full dev stack, including the hot-reloading API' \
		'  make docker-down     Stop Docker services and remove containers' \
		'  make docker-logs     Follow PostgreSQL and Valkey logs' \
		'  make docker-logs-api Follow API logs from the hot-reload container' \
		'  make go-mod-tidy     Resolve and tidy Go dependencies' \
		'  make sqlc-generate   Generate typed Go code from SQL files' \
		'  make migrate-create NAME=create_users  Create a numbered SQL migration pair' \
		'  make migrate-up      Apply all up migrations using the migrate container' \
		'  make migrate-down    Roll back one migration step using the migrate container' \
		'  make fmt             Run gofmt across the project' \
		'  make test            Run Go tests' \
		'  make test-integration Run Docker-backed integration tests' \
		'  make coverage        Generate coverage for handwritten code' \
		'  make coverage-integration Generate handwritten-code coverage with Docker-backed integration tests enabled' \
		'  make coverage-check  Enforce the minimum handwritten-code coverage threshold' \
		'  make coverage-check-integration Enforce the minimum threshold with integration-backed coverage enabled' \
		'  make release-check-mcp Validate the GoReleaser config for the MCP binary' \
		'  make release-snapshot-mcp Build local snapshot archives for the MCP binary' \
		'  make run             Start the API locally' \
		'  make run-mcp         Start the local stdio MCP server'

init:
	@mkdir -p cmd/api docs internal/app internal/auth internal/httpapi internal/notes internal/platform/cache internal/platform/db internal/platform/oidc migrations sql/queries sql/schema
	@printf 'Starter layout is ready in %s\n' "$$(pwd)"

check-deps:
	@set -eu; \
	missing=0; \
	for dep in go sqlc docker; do \
		if command -v "$$dep" >/dev/null 2>&1; then \
			printf '[ok] %s -> %s\n' "$$dep" "$$(command -v "$$dep")"; \
		else \
			printf '[missing] %s\n' "$$dep"; \
			missing=1; \
		fi; \
	done; \
	if [ -n "$(DOCKER_COMPOSE)" ]; then \
		printf '[ok] compose -> %s\n' "$(DOCKER_COMPOSE)"; \
	else \
		printf '[missing] docker compose or docker-compose\n'; \
		missing=1; \
	fi; \
	exit "$$missing"

deps: install-go install-sqlc check-docker-tools
	@$(MAKE) check-deps

check-docker-tools:
	@set -eu; \
	if ! command -v docker >/dev/null 2>&1; then \
		printf 'Docker is required for PostgreSQL and Valkey. Install Docker Desktop or Docker Engine.\n' >&2; \
		exit 1; \
	fi; \
	if [ -z "$(DOCKER_COMPOSE)" ]; then \
		printf 'Docker Compose is required. Install the Docker Compose plugin or docker-compose.\n' >&2; \
		exit 1; \
	fi

check-docker-daemon:
	@docker info >/dev/null 2>&1 || (printf 'Docker is installed, but the daemon is not running. Start Docker Desktop or dockerd and try again.\n' >&2; exit 1)

install-go:
	@set -eu; \
	if command -v go >/dev/null 2>&1; then \
		printf 'Go already installed: %s\n' "$$(command -v go)"; \
		exit 0; \
	fi; \
	if command -v brew >/dev/null 2>&1; then \
		brew install $(GO_VERSION_PKG); \
	elif command -v apt-get >/dev/null 2>&1; then \
		sudo apt-get update; \
		sudo apt-get install -y golang-go; \
	elif command -v dnf >/dev/null 2>&1; then \
		sudo dnf install -y golang; \
	elif command -v yum >/dev/null 2>&1; then \
		sudo yum install -y golang; \
	elif command -v pacman >/dev/null 2>&1; then \
		sudo pacman -Sy --noconfirm go; \
	else \
		printf 'No supported package manager found for installing Go.\n' >&2; \
		exit 1; \
	fi

install-sqlc:
	@set -eu; \
	if command -v sqlc >/dev/null 2>&1; then \
		printf 'sqlc already installed: %s\n' "$$(command -v sqlc)"; \
		exit 0; \
	fi; \
	if command -v brew >/dev/null 2>&1; then \
		brew install sqlc; \
	elif command -v go >/dev/null 2>&1; then \
		GO111MODULE=on go install "$(SQLC_INSTALL_PKG)"; \
		printf 'sqlc installed via go install. Ensure %s is on PATH if needed.\n' "$${GOBIN:-$$HOME/go/bin}"; \
	else \
		printf 'sqlc requires either Homebrew or Go on this machine.\n' >&2; \
		exit 1; \
	fi

go-mod-tidy:
	mkdir -p $(GO_BUILD_CACHE) $(GO_MOD_CACHE)
	GOCACHE=$(GO_BUILD_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go mod tidy

sqlc-generate:
	sqlc generate -f $(SQLC_CONFIG)

docker-up: check-docker-daemon
	@test -n "$(DOCKER_COMPOSE)" || (printf 'Docker Compose is required.\n' >&2; exit 1)
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE) -p $(COMPOSE_PROJECT_NAME) up -d postgres valkey

docker-up-app: check-docker-daemon
	@test -n "$(DOCKER_COMPOSE)" || (printf 'Docker Compose is required.\n' >&2; exit 1)
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE) -p $(COMPOSE_PROJECT_NAME) up -d --build --force-recreate api postgres valkey

docker-down: check-docker-daemon
	@test -n "$(DOCKER_COMPOSE)" || (printf 'Docker Compose is required.\n' >&2; exit 1)
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE) -p $(COMPOSE_PROJECT_NAME) down --remove-orphans

docker-logs: check-docker-daemon
	@test -n "$(DOCKER_COMPOSE)" || (printf 'Docker Compose is required.\n' >&2; exit 1)
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE) -p $(COMPOSE_PROJECT_NAME) logs -f postgres valkey

docker-logs-api: check-docker-daemon
	@test -n "$(DOCKER_COMPOSE)" || (printf 'Docker Compose is required.\n' >&2; exit 1)
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE) -p $(COMPOSE_PROJECT_NAME) logs -f api

migrate-create:
	@test -n "$(NAME)" || (printf 'Usage: make migrate-create NAME=create_users\n' >&2; exit 1)
	@next="$$(find $(MIGRATIONS_DIR) -maxdepth 1 -type f -name '[0-9][0-9][0-9][0-9][0-9][0-9]_*.up.sql' | sed 's#.*/##' | cut -d_ -f1 | sort | tail -n 1)"; \
	if [ -z "$$next" ]; then next=1; else next="$$(expr "$$next" + 1)"; fi; \
	seq="$$(printf '%06d' "$$next")"; \
	touch "$(MIGRATIONS_DIR)/$${seq}_$(NAME).up.sql" "$(MIGRATIONS_DIR)/$${seq}_$(NAME).down.sql"; \
	printf 'Created %s and %s\n' "$(MIGRATIONS_DIR)/$${seq}_$(NAME).up.sql" "$(MIGRATIONS_DIR)/$${seq}_$(NAME).down.sql"

migrate-up: docker-up
	@test -n "$(DOCKER_COMPOSE)" || (printf 'Docker Compose is required.\n' >&2; exit 1)
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE) -p $(COMPOSE_PROJECT_NAME) run --rm migrate -path /migrations -database "$(MIGRATE_DATABASE_URL)" up

migrate-down: docker-up
	@test -n "$(DOCKER_COMPOSE)" || (printf 'Docker Compose is required.\n' >&2; exit 1)
	$(DOCKER_COMPOSE) -f $(COMPOSE_FILE) -p $(COMPOSE_PROJECT_NAME) run --rm migrate -path /migrations -database "$(MIGRATE_DATABASE_URL)" down 1

fmt:
	gofmt -w $$(find . \( -path './.cache' -o -path './vendor' \) -prune -o -name '*.go' -type f -print)

test:
	mkdir -p $(GO_BUILD_CACHE) $(GO_MOD_CACHE)
	GOCACHE=$(GO_BUILD_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go test ./...

test-integration: docker-up migrate-up
	mkdir -p $(GO_BUILD_CACHE) $(GO_MOD_CACHE)
	RUN_INTEGRATION=1 DATABASE_URL="$(DATABASE_URL)" VALKEY_ADDR="$(VALKEY_HOST):$(VALKEY_PORT)" GOCACHE=$(GO_BUILD_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go test ./integration/...

coverage:
	mkdir -p $(GO_BUILD_CACHE) $(GO_MOD_CACHE) $(COVERAGE_DIR)
	HANDWRITTEN_PACKAGES="$$(GOCACHE=$(GO_BUILD_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go list ./... | grep -v '/cmd/api$$' | grep -v '/cmd/mcp$$' | grep -v '/internal/platform/db/sqlc$$')"; \
	GOCACHE=$(GO_BUILD_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go test -covermode=atomic -coverprofile="$(RAW_COVERAGE_PROFILE)" $$HANDWRITTEN_PACKAGES; \
	cp "$(RAW_COVERAGE_PROFILE)" "$(COVERAGE_PROFILE)"; \
	GOCACHE=$(GO_BUILD_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go tool cover -func="$(COVERAGE_PROFILE)"

coverage-integration: docker-up migrate-up
	mkdir -p $(GO_BUILD_CACHE) $(GO_MOD_CACHE) $(COVERAGE_DIR)
	HANDWRITTEN_PACKAGES="$$(GOCACHE=$(GO_BUILD_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go list ./... | grep -v '/cmd/api$$' | grep -v '/cmd/mcp$$' | grep -v '/internal/platform/db/sqlc$$')"; \
	RUN_INTEGRATION=1 DATABASE_URL="$(DATABASE_URL)" VALKEY_ADDR="$(VALKEY_HOST):$(VALKEY_PORT)" GOCACHE=$(GO_BUILD_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go test -covermode=atomic -coverprofile="$(RAW_COVERAGE_PROFILE)" $$HANDWRITTEN_PACKAGES; \
	cp "$(RAW_COVERAGE_PROFILE)" "$(COVERAGE_PROFILE)"; \
	GOCACHE=$(GO_BUILD_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go tool cover -func="$(COVERAGE_PROFILE)"

coverage-check: coverage
	@total="$$(GOCACHE=$(GO_BUILD_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go tool cover -func="$(COVERAGE_PROFILE)" | awk '/^total:/ { sub(/%/, "", $$3); print $$3 }')"; \
	if awk 'BEGIN { exit !('"$$total"' + 0 < $(COVERAGE_THRESHOLD)) }'; then \
		printf 'Coverage %.1f%% is below the required %.1f%% threshold.\n' "$$total" "$(COVERAGE_THRESHOLD)" >&2; \
		exit 1; \
	fi; \
	printf 'Coverage %.1f%% meets the %.1f%% threshold.\n' "$$total" "$(COVERAGE_THRESHOLD)"

coverage-check-integration: coverage-integration
	@total="$$(GOCACHE=$(GO_BUILD_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go tool cover -func="$(COVERAGE_PROFILE)" | awk '/^total:/ { sub(/%/, "", $$3); print $$3 }')"; \
	if awk 'BEGIN { exit !('"$$total"' + 0 < $(COVERAGE_THRESHOLD)) }'; then \
		printf 'Coverage %.1f%% is below the required %.1f%% threshold.\n' "$$total" "$(COVERAGE_THRESHOLD)" >&2; \
		exit 1; \
	fi; \
	printf 'Coverage %.1f%% meets the %.1f%% threshold.\n' "$$total" "$(COVERAGE_THRESHOLD)"

release-check-mcp:
	mkdir -p $(GO_BUILD_CACHE) $(GO_MOD_CACHE)
	GOCACHE=$(GO_BUILD_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go run $(GORELEASER_PKG) check --config .goreleaser.yaml

release-snapshot-mcp:
	mkdir -p $(GO_BUILD_CACHE) $(GO_MOD_CACHE)
	GOCACHE=$(GO_BUILD_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go run $(GORELEASER_PKG) release --clean --snapshot --config .goreleaser.yaml

run:
	mkdir -p $(GO_BUILD_CACHE) $(GO_MOD_CACHE)
	@$(LOAD_LOCAL_ENV); HTTP_ADDR=$(APP_ADDR) GOCACHE=$(GO_BUILD_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go run ./cmd/api

run-mcp:
	mkdir -p $(GO_BUILD_CACHE) $(GO_MOD_CACHE)
	@$(LOAD_LOCAL_ENV); GOCACHE=$(GO_BUILD_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go run ./cmd/mcp
