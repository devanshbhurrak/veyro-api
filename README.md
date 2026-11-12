# Veyro Backend

Go modular-monolith backend for Veyro, a business management and accounting
application for small and growing businesses in India.

## Stack

- Go, Chi, REST/OpenAPI
- PostgreSQL with pgx and golang-migrate
- `log/slog` structured logging
- Argon2id password hashing, opaque access tokens, rotating refresh tokens
- Transactional outbox for transactional email

## Layout

```
cmd/api        HTTP API entry point
cmd/worker     Outbox email worker entry point
internal/app   Dependency wiring for both binaries
internal/server Router, middleware order and route permissions
internal/modules
  identity       registration, verification, login, sessions, password recovery
  tenancy        workspaces and memberships
  authorization  roles, permissions and trusted request context
  organization   company onboarding, settings, tax registrations, readiness
  invitations    staff invitations and acceptance
  platformadmin  limited internal tenant administration
internal/platform  config, database, auth, middleware, audit, outbox, email, throttle
internal/shared    small shared helpers (errors, responses, validation, IDs)
db/migrations      versioned schema migrations
api/openapi.yaml   implemented HTTP contract
```

## Local development

Requirements: Go 1.22+, PostgreSQL 15+, golang-migrate.

```bash
createdb veyro
export DB_HOST=localhost DB_PORT=5432 DB_NAME=veyro DB_USER=postgres DB_PASSWORD=postgres
export AUTH_TOKEN_ENCRYPTION_KEY=replace-with-a-long-random-secret
make migrate-up
make run-api      # terminal 1
make run-worker   # terminal 2
```

`EMAIL_PROVIDER=log` writes rendered transactional email to
`.tmp/emails` for development. Configure `EMAIL_PROVIDER=smtp` for real
delivery. `.env.example` lists every supported variable; the application reads
configuration from the process environment only.

## API

All routes are under `/api/v1`. Permissions are enforced server-side; the
route table is in `internal/server/router.go`.

Health:

- `GET /health` liveness
- `GET /ready` database readiness

Identity:

- `POST /auth/register`, `POST /auth/verify-email`, `POST /auth/verify-email/resend`
- `POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout`
- `POST /auth/password/forgot`, `POST /auth/password/reset`
- `GET /auth/me`, `POST /auth/logout-all`, `GET /auth/sessions`,
  `DELETE /auth/sessions/{sessionID}`

Workspaces:

- `POST /workspaces`, `GET /workspaces`
- `GET|PATCH /workspaces/{tenantID}`
- members, roles, assignments, invitations under `/workspaces/{tenantID}`
- `POST /invitations/accept` (authenticated)

Company onboarding:

- `POST /workspaces/{tenantID}/companies` (idempotent with `Idempotency-Key`)
- `GET|PATCH /workspaces/{tenantID}/company`
- `GET|PATCH /workspaces/{tenantID}/company/settings`
- `GET|POST /workspaces/{tenantID}/company/tax-registrations`,
  `PATCH .../{registrationID}`
- `GET /workspaces/{tenantID}/onboarding/status`
- `GET /workspaces/{tenantID}/overview`

Platform administration (verified emails listed in `PLATFORM_ADMIN_EMAILS`):

- `GET /platform/health`
- `GET /platform/tenants`, `GET /platform/tenants/{tenantID}`
- `PATCH /platform/tenants/{tenantID}` (activate/suspend only)

The OpenAPI document is `api/openapi.yaml`. Mutating requests send JSON;
company creation accepts an `Idempotency-Key` header. Authentication uses
`Authorization: Bearer <access_token>`.

## Auth model

- Registration requires email verification before login.
- Access tokens are opaque, short-lived, and stored only as SHA-256 hashes.
- Refresh tokens rotate on every refresh. Reuse of a rotated token revokes the
  token family. A refresh may be sent in the JSON body or as the HttpOnly
  `veyro_refresh` cookie together with the `X-CSRF-Token` double-submit cookie.
- Password reset revokes all sessions.
- Tokens placed in outbox payloads are AES-256-GCM encrypted with
  `AUTH_TOKEN_ENCRYPTION_KEY`; raw tokens are never stored in PostgreSQL.

## Company onboarding

Creating a company runs one database transaction that creates the company,
optional GST registration, default branch and warehouse, chart of accounts,
cash account, financial year and 12 periods, standard units, document number
series, GST tax codes, company settings, and a default invoice template.
Readiness is computed from these records, not from client claims.

## Tests

```bash
make test
make vet
make test-race
TEST_DATABASE_URL=postgres://postgres@localhost:5432/veyro_test?sslmode=disable make test-integration
```

Integration tests skip unless `TEST_DATABASE_URL` points at an isolated,
migrated database. Never point it at production.

## Known limitations

- Login/resend throttling is in-memory and assumes a single API instance.
- SQL is handwritten with pgx; sqlc configuration exists but generated
  queries are not yet used for these modules.
- No frontend, business documents, inventory, or accounting postings are part
  of this phase.
