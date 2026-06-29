# Operations Runbook

Sharecrop runs as one HTTP process backed by PostgreSQL.

## Required Configuration

- `SHARECROP_HTTP_ADDR`: HTTP bind address, for example `:18080`.
- `DATABASE_URL`: PostgreSQL connection string.
- `SHARECROP_MIGRATIONS_DIR`: path to SQL migrations.
- `SHARECROP_ACCESS_TOKEN_SECRET`: at least 32 bytes.
- `SHARECROP_ADMIN_USER_IDS`: comma-separated user ids that may award catalog collectibles and read admin operations status.
- `SHARECROP_ACCOUNT_TOKEN_DELIVERY`: `api` for local/test token responses, or `log` to emit verification/reset tokens to structured logs and return only `{"status":"sent"}`.
- `SHARECROP_INSECURE_COOKIES`: set to `true` only for local plain-HTTP development.

## Deploy

1. Build the binary with `make build`.
2. Copy `bin/sharecrop`, `migrations/`, and static assets into the release directory.
3. Install an environment file at `/etc/sharecrop/sharecrop.env`.
4. Install [sharecrop.service](../deploy/systemd/sharecrop.service) or an equivalent process supervisor unit.
5. Run migrations before restarting the service.

## Migrations

Run:

```sh
sharecrop migrate up
```

Migrations are forward-only SQL files. Take a database backup before applying migrations in production. Rollback is restore-from-backup plus redeploying the prior binary.

## Backup And Restore

Use PostgreSQL-native backups. A baseline command is:

```sh
pg_dump "$DATABASE_URL" --format=custom --file=sharecrop-$(date +%Y%m%d%H%M%S).dump
```

Restore into a new database first, then point Sharecrop at the restored database after validation.

## Logs And Admin Status

The process writes structured `slog` text logs to stderr. Account verification and password reset tokens are logged only when `SHARECROP_ACCOUNT_TOKEN_DELIVERY=log`.

Platform admins can call:

```sh
GET /api/admin/operations
GET /api/admin/audit-events?action=submission_accepted&subject_kind=submission&limit=20&offset=0
```

The response reports account-token delivery mode, secure-cookie mode, active MCP session count, active rate-limit buckets, and whether MCP/rate-limit storage is process-local.

The audit endpoint supports optional `action`, `subject_kind`, and `subject_id` filters plus pagination. Use these filters when investigating task refunds, submission review outcomes, organization member changes, account deactivation, and admin collectible awards.

## Current Operational Limits

The database includes `audit_events`, `rate_limit_buckets`, and `mcp_http_sessions` tables as the operations-state schema foundation.

MCP HTTP sessions, SSE replay buffers, and rate-limit buckets are still stored in process memory by the runtime. This is acceptable for one process. Multi-process deployments need the runtime adapters to use the Postgres tables and need a cross-process SSE fan-out design before horizontal scaling.

Sharecrop rewards are internal-only: Sharecrop credits and admin-minted Sharecrop collectibles. User/org/per-project tokens, external wallets, and crypto integrations are not part of the operating model.
