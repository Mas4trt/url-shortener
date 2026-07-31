include .env
export

export PROJECT_ROOT=${CURDIR}

env-up:
	@docker compose up -d urlshortener-postgres

env-down:
	@docker compose down urlshortener-postgres

env-port-forward:
	@docker compose up -d port-forwarder

env-port-close:
	@docker compose down port-forwarder

env-cleanup:
	@read -p "Clean up all environment volume files? Risk of data loss. [Y] Yes [N] No:" ans; \
	if [ "$$ans" = "Y" ] || [ "$$ans" = "y" ]; then \
		docker compose down urlshortener-postgres && \
		rm -rf out/pqdata && \
		echo "Environment files have been cleaned"; \
	else \
		echo "Data cleanup cancelled"; \
	fi

migrate-create:
	@if [ -z "$(seq)" ]; then \
		echo "Missing required parameter seq. Example: make migrate-create seq=init"; \
		exit 1; \
	fi; \
	MSYS_NO_PATHCONV=1 docker compose run --rm urlshortener-postgres-migrate \
		create \
		-ext sql \
		-dir /migrations \
		-seq "$(seq)"

migrate-up:
	@make migrate-action action=up

migrate-down:
	@make migrate-action action=down

migrate-action:
	@if [ -z "$(action)" ]; then \
		echo "Missing required parameter action. Example: make migrate-action action=up1"; \
		exit 1; \
	fi; \
	MSYS_NO_PATHCONV=1 docker compose run --rm urlshortener-postgres-migrate \
		-path /migrations \
		-database postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@urlshortener-postgres:5432/${POSTGRES_DB}?sslmode=disable \
		"$(action)"

up:
	docker compose up -d --build

down:
	docker compose down

restart: down up

init:
	docker compose up -d urlshortener-postgres redis
	docker compose up urlshortener-postgres-migrate
	docker compose up -d app

status:
	docker compose ps

# --- previously missing: README documented these, Makefile didn't have them ---

build:
	go build -o bin/url-shortener ./cmd/url-shortener

# Unit tests only (fast, no docker). Integration tests under
# internal/tests/integration and internal/storage/postgres need a docker
# daemon (testcontainers) — run those with `make test-integration`.
test:
	go test -race -cover -short ./...

test-integration:
	go test -race -v ./internal/tests/integration/... ./internal/storage/postgres/...

lint:
	@fmt_out="$$(gofmt -l .)"; \
	if [ -n "$$fmt_out" ]; then \
		echo "not gofmt'ed:"; echo "$$fmt_out"; exit 1; \
	fi
	go vet ./...

fmt:
	gofmt -w .