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