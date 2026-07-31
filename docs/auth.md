# Authentication via sso

`url-shortener` no longer has its own user store — registration, login,
and refresh all proxy to the central [`sso`](https://github.com/Mas4trt/sso)
service over gRPC. Creating and deleting short links requires a valid sso
access token; resolving a short link (`GET /{alias}`) stays public.

## How it works

- `internal/ssoclient` — thin gRPC client around `authv1.AuthClient`,
  scoped to this app's `application_id`.
- `internal/authn` — verifies access tokens **locally**, without calling
  sso. sso signs access tokens (HS256) with this app's `apps.secret`; as
  long as `url-shortener` has a copy of that same secret, it can check
  the signature itself. Only login/register/refresh/logout actually hit
  sso over the network.
- `internal/transport/http/middleware/auth` — `RequireAuth` middleware
  wrapping the token check; applied to `POST /url` and `DELETE /{alias}`.

## Endpoints added

| Method | Path           | Description                          | Auth required |
|--------|----------------|---------------------------------------|----------------|
| POST   | `/auth/register` | Create a user via sso               | no |
| POST   | `/auth/login`     | Get access + refresh token          | no |
| POST   | `/auth/refresh`   | Rotate a refresh token              | no |
| POST   | `/auth/logout`    | Revoke a refresh token              | no |
| POST   | `/url`            | Create a short link                 | **yes** |
| DELETE | `/{alias}`        | Delete a short link                 | **yes** |
| GET    | `/{alias}`        | Resolve/redirect                    | no |

Protected requests need `Authorization: Bearer <access_token>`.

## Setup

1. In the `sso` repo, provision this service:

   ```bash
   make new-app NAME=url-shortener DB_DSN=postgres://sso:sso@<host>:5432/sso?sslmode=disable
   ```

   This prints `id`, `name`, `secret`.

2. Set in `url-shortener`'s environment (see `.env.example`):

   ```
   SSO_ADDR=sso:44044          # or localhost:44044 outside docker
   SSO_APPLICATION_ID=<id from step 1>
   SSO_APP_SECRET=<secret from step 1>
   ```

3. If running both stacks via docker-compose, `url-shortener`'s
   `docker-compose.yaml` joins sso's network (`sso_default`, from sso's
   `name: sso` compose project) so `sso:44044` resolves. If sso runs
   elsewhere, drop that network block and point `SSO_ADDR` at the real
   host.

## Access token lifetime & refresh

Access tokens expire (see sso's `token.ttl`, 1h by default). When a
request to a protected endpoint gets `401` with "invalid or expired
token", call `POST /auth/refresh` with the stored refresh token to get a
new pair — refresh tokens are single-use (sso rotates them on every
refresh), so always store the new one and discard the old.

## Dependency note

This pulls in `github.com/Mas4trt/protos` and `google.golang.org/grpc` as
new direct dependencies — run `go mod tidy` after pulling these changes
in, since I edited `go.mod` by hand without a working Go toolchain
available to verify it.