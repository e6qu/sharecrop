# Operations Runbook

Sharecrop runs as stateless, multi-replica containers in private subnets, with
all state in PostgreSQL. Production is deployed on Amazon ECS Fargate behind an
Amazon API Gateway HTTP API private integration and AWS Cloud Map; the full
deployment reference is [deployment.md](./deployment.md).

## Required Configuration

- `SHARECROP_HTTP_ADDR`: HTTP bind address, for example `:29180`. The container
  image defaults it to `:8080`.
- `DATABASE_URL`: PostgreSQL connection string. In production, point it at an RDS
  Proxy endpoint so many replicas share a bounded connection pool.
- `SHARECROP_MIGRATIONS_DIR`: path to SQL migrations (baked into the image at
  `/migrations`).
- `SHARECROP_ACCESS_TOKEN_SECRET`: at least 32 bytes.
- `SHARECROP_WASI_MODE`: unset (the default) hosts the app through the embedded
  WASI guest pool; set to `native` to run the in-process mux instead.
- `SHARECROP_WAZERO_CACHE_DIR`: the baked wazero AOT cache (image default
  `/wazero-cache`) so the guest is loaded, not compiled, at startup. Leave as-is.
- `SHARECROP_ADMIN_USER_IDS`: comma-separated bootstrap platform-admin user ids. Platform admins can grant and revoke other platform admins (bootstrap admins cannot be revoked), award catalog collectibles, read admin operations status, list platform-wide audit events, list and triage moderation reports, list and resolve privacy requests, and run sensitive-field retention.
- `SHARECROP_ACCOUNT_TOKEN_DELIVERY`: `api` for local/test token responses, or `log` to emit verification/reset tokens to structured logs and return only `{"status":"sent"}`.
- `SHARECROP_INSECURE_COOKIES`: set to `true` only for local plain-HTTP development.

## Deploy

Production runs as a container on ECS Fargate. The full procedure — the slim
multi-arch (arm64) image, the ghcr release workflow, the ECS task definitions,
and the database setup — is in [deployment.md](./deployment.md). In short:

1. A merge to `main` builds and publishes the immutable 12-character commit-SHA
   image to the GitHub Container Registry (`.github/workflows/release.yml`).
2. Run the one-off `sharecrop migrate up` task against the database.
3. Roll the `sharecrop-serve` ECS service to the new image tag.

For a single-host / non-container deployment, the binary from `make build` plus
[sharecrop.service](../deploy/systemd/sharecrop.service) still works (copy the
binary, `migrations/`, and an env file, run migrations, start the unit), but it
is no longer the production path.

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

The response reports account-token delivery mode, secure-cookie mode, active MCP session count, active rate-limit buckets, and whether MCP/rate-limit storage is process-local or Postgres-backed.

The audit endpoint supports optional `action`, `subject_kind`, and `subject_id` filters plus pagination. Use these filters when investigating task refunds, submission review outcomes, organization member changes, account deactivation, privacy requests, and admin collectible awards. Privacy requests use `action=privacy_request_created` and `subject_kind=privacy_request`.

## Current Operational Limits

The database includes `audit_events`, `rate_limit_buckets`, and `mcp_http_sessions` tables as the operations-state schema foundation.

Production `serve` (`cmd/sharecrop`) wires Postgres-backed rate-limit buckets, persisted MCP HTTP session identity, and persisted MCP replay events against those tables (migrations `000024_operations_foundation.sql` and `000026_mcp_http_events.sql`), along with Postgres-backed audit events, notifications, saved queue views, privacy requests, platform admins, and moderation triage. This makes `serve` stateless, so it scales horizontally across the task instances discovered through AWS Cloud Map: MCP sessions and replay events are shared through Postgres, and SSE subscribers deliver by polling the replay table (the WASI-hosted backend returns a bounded response rather than holding a stream open). Real-time cross-replica SSE push would still want an HTTP/2 streaming transport or pub/sub (see DO_NEXT.md); polling covers correctness today. The only process-local state is the per-process rate-limit buckets, which are defense-in-depth, not correctness. The in-memory defaults apply only to the test/demo HTTP constructor, not to `serve`.

Sharecrop rewards are internal-only: Sharecrop credits and admin-minted Sharecrop collectibles. User/org/per-project tokens, external wallets, and crypto integrations are not part of the operating model.
