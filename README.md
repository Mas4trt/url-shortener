# URL Shortener Service

A high-performance URL shortening service built in Go, designed for reliable link creation, low-latency redirects, and extensible authentication flows.

![Go](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-316192?logo=postgresql)
![Redis](https://img.shields.io/badge/Redis-7-DC382D?logo=redis)

## Overview

This project implements a compact URL shortener with a clean separation between transport, application, domain, and infrastructure layers. It supports:

- public redirect resolution for short aliases
- authenticated creation and deletion of short links
- PostgreSQL-backed persistence for durable storage
- Redis-backed caching for fast reads
- structured logging, request tracing, and rate limiting
- containerized local development with Docker Compose

## Architecture

The service follows a layered architecture that keeps business logic independent from delivery and storage concerns.

```text
Client -> HTTP handlers -> application services -> domain models
                           |                     |
                           |                     +-> storage adapters
                           +-> auth middleware / SSO client
```

### Core components

- Transport: HTTP routing, validation, middleware, response shaping
- Application: orchestration of business workflows and dependency wiring
- Domain: URL models, validation rules, and core errors
- Storage: PostgreSQL repository and Redis cache implementations
- Auth: SSO-based token verification for protected operations

## Tech stack

- Language: Go 1.26+
- Web framework: chi
- Validation: go-playground/validator
- Database: PostgreSQL
- Cache: Redis
- Migrations: golang-migrate
- Dependency injection: google/wire
- Testing: testify, testcontainers-go
- Containerization: Docker Compose

## Project structure

```text
cmd/                    # application entrypoints
internal/
  app/                  # lifecycle and bootstrap orchestration
  authn/                # token validation logic
  bootstrap/            # dependency injection and initialization
  config/               # configuration loading and validation
  domain/               # business models and shared errors
  service/              # URL service logic
  ssoclient/            # SSO gRPC client integration
  storage/              # PostgreSQL and Redis adapters
  tests/                # integration test suites
  transport/            # HTTP handlers, middleware, routing
migrations/             # SQL migration files
pkg/                    # shared utilities
```

## Getting started

### Prerequisites

- Docker and Docker Compose
- Go 1.26 or newer
- Make

### 1. Clone the repository

```bash
git clone <repository-url>
cd url-shortener
```

### 2. Configure environment

The application uses environment variables and the local config file at [config/local.yaml](config/local.yaml). A minimal setup looks like this:

```bash
export POSTGRES_USER=urlshortener
export POSTGRES_PASSWORD=urlshortener
export POSTGRES_DB=urlshortener
export SSO_ADDR=sso:44044
export SSO_APPLICATION_ID=<your-sso-application-id>
export SSO_APP_SECRET=<your-sso-app-secret>
```

> For full authenticated flows, configure the SSO variables as described in [docs/auth.md](docs/auth.md). Public redirect and health endpoints remain available without them.

### 3. Start the local stack

```bash
make up
```

This boots:
- PostgreSQL
- Redis
- the application service
- database migrations

### 4. Verify the deployment

```bash
curl http://localhost:8080/healthz
```

Expected response: a healthy status payload from the service.

## API reference

### Public endpoints

| Method | Path | Description |
| --- | --- | --- |
| GET | /healthz | Health check endpoint |
| GET | /{alias} | Resolve a short alias and redirect to the original URL |

### Authenticated endpoints

Protected routes require a bearer token in the Authorization header.

| Method | Path | Description |
| --- | --- | --- |
| POST | /auth/register | Register a user through the SSO service |
| POST | /auth/login | Authenticate and receive access and refresh tokens |
| POST | /auth/refresh | Rotate a refresh token |
| POST | /auth/logout | Revoke a refresh token |
| POST | /url | Create a short link |
| DELETE | /{alias} | Delete a short link |

### Example: create a short link

```bash
curl -X POST http://localhost:8080/url \
  -H "Authorization: Bearer <access-token>" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com","alias":"example"}'
```

## Configuration

Configuration is loaded from environment variables and the local YAML config in [config/local.yaml](config/local.yaml).

Key settings include:

- DATABASE_URL: PostgreSQL connection string
- REDIS_URL: Redis endpoint
- CONFIG_PATH: path to the configuration file
- SSO_ADDR: address of the SSO service
- SSO_APPLICATION_ID: SSO application identifier
- SSO_APP_SECRET: SSO shared secret
- alias_length: generated alias length
- max_retries: collision retry budget
- ttl: cache TTL for Redis entries

## Development workflow

### Useful commands

```bash
make build               # build the binary
make test                # run unit tests with race detection and coverage
make test-integration   # run integration tests (requires Docker)
make lint                # enforce formatting and run go vet
make migrate-up          # apply database migrations
make migrate-down        # roll back database migrations
make down                # stop the local stack
```

### Testing

The repository includes both unit and integration tests.

- Unit tests run locally without external services.
- Integration tests use testcontainers-go to provision isolated PostgreSQL and Redis instances.

Run the full test suite:

```bash
make test
```

## Authentication

Write operations are protected by an SSO-backed authentication flow. The service validates access tokens locally and delegates account lifecycle operations to the SSO service. See [docs/auth.md](docs/auth.md) for the full design and setup details.

## Security and operations

The service includes several production-minded safeguards:

- request ID propagation for structured tracing
- request logging middleware
- basic rate limiting for HTTP traffic
- dependency injection for predictable wiring and testing
- database migrations for schema evolution

## Contributing

Contributions are welcome. Keep changes focused, update tests when behavior changes, and follow the existing project conventions.

## License

This project is distributed under the MIT License.

