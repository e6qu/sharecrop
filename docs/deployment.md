# Deployment

Sharecrop ships as one application built for two runtimes from the same source:

- **Browser demo** — `js/wasm`, storage backed by in-browser SQLite, served
  statically (GitHub Pages). Single user, entirely in-browser.
- **Backend** — the same app hosted server-side through the WASI guest pool
  (`cmd/sharecrop serve`, which embeds the `wasip1` app guest and runs it under
  the wazero runtime), with all state in Postgres. Deployed as **stateless
  replicas on Amazon ECS Fargate (arm64) in private subnets, reached through an
  Amazon API Gateway HTTP API private integration and AWS Cloud Map**.

The backend is stateless: every durable thing (tasks, submissions, ledger,
audit, notifications, MCP sessions) lives in Postgres, SSE is delivered by DB
polling, and the only per-process state (in-memory rate-limit buckets) is
defense-in-depth, not correctness. So replicas scale horizontally with no
affinity — Amazon API Gateway can distribute requests across the healthy task
instances returned by AWS Cloud Map.

## Container image standard

Multi-arch images follow the repo-wide naming convention:

| Reference          | What it is                                   |
| ------------------ | -------------------------------------------- |
| `sharecrop:<tag>`         | multi-arch manifest list (what deployments reference) |
| `sharecrop:<tag>-arm64`   | per-arch image, **primary** (Graviton)       |
| `sharecrop:<tag>-amd64`   | per-arch image                               |

The services run on **arm64**. The manifest (`sharecrop:<tag>`) is what task
definitions and `docker pull` reference; Fargate selects the arm64 variant from
it automatically for an arm64 task. The per-arch tags exist so a specific
architecture can be pulled or promoted directly.

### Building

Each arch is built where the freshly built binary can execute (see "no build on
startup" below), so build the per-arch images first, then assemble the manifest:

```sh
# Per-arch build + push (run each on a native runner of that arch; CI does this
# in an arm64/amd64 matrix):
tools/build_container.sh ghcr.io/e6qu/sharecrop:0123456789ab arm64   # -> :0123456789ab-arm64
tools/build_container.sh ghcr.io/e6qu/sharecrop:0123456789ab amd64   # -> :0123456789ab-amd64

# Assemble the multi-arch manifest from the per-arch images:
tools/build_container.sh ghcr.io/e6qu/sharecrop:0123456789ab manifest  # -> :0123456789ab

# Local single-arch build for testing (host arch, loaded into the local docker):
PUSH=false tools/build_container.sh sharecrop:dev
```

Properties:

- **No build on startup.** Everything is baked at build time: the frontend
  (embedded in the binary), the `wasip1` guest, and — via a `wasi-precompile`
  build step — the guest's **wazero AOT machine-code cache**. `serve` loads the
  compiled guest from the cache (`SHARECROP_WAZERO_CACHE_DIR=/wazero-cache`, set
  in the image) instead of compiling the ~11 MB module on boot: startup module
  prep drops from ~1.7 s to ~0.08 s. The `guest_compile` field in the startup log
  surfaces whether the cache was hit; a slow value means it fell back to
  compiling (e.g. a stale cache). The cache key is `wazero-<version>-<arch>-linux`
  — portable across CPUs of the same arch, so an arm64 image's cache hits on any
  arm64 Fargate host.
- **Slim image.** The runtime is `distroless/static` plus one static (CGO-free)
  binary — no shell, no libc. The baked wazero cache adds ~45 MB (the machine
  code for the whole app), for a ~77 MB image; that is the deliberate cost of
  doing no compilation at startup. It still pulls quickly (registry layers are
  compressed) and has a small attack surface. For even faster task launch, enable
  [SOCI](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/container-considerations.html)
  lazy image loading on the cluster.
- **Native per arch.** Baking the cache means running the just-built binary, and
  the cache is CPU-arch-specific, so each arch is built natively (a matching
  runner) rather than cross-compiled. Building the non-native arch locally still
  works but runs the whole build under emulation.

The frontend assets embedded into the binary are the committed `web/static/*`.
Run `make frontend` before a release build if the UI changed, the same as for
`make build`.

### Releases (CI)

`.github/workflows/release.yml` builds and publishes on **every** merge to
`main`:

1. Derive the immutable tag from the first 12 lowercase hexadecimal characters
   of the merged commit SHA.
2. Build each architecture on a native runner (arm64 on `ubuntu-24.04-arm`,
   amd64 on `ubuntu-24.04`) and push direct, single-platform OCI image manifests
   to the GitHub Container Registry. Provenance and SBOM attestations are
   disabled so an architecture tag never becomes an image index.
3. Assemble the generic OCI image index, verify that it contains exactly Linux
   amd64 and Linux arm64, and prune package versions outside the newest 20
   complete commit-SHA releases (`tools/prune_ghcr_versions.sh`). The retention
   gate also deletes untagged, incomplete, mixed-tag, and unrecognized versions,
   then re-reads the package and verifies the postcondition. A shape or
   retention failure fails the workflow.

Images are published to `ghcr.io/<owner>/<repo>:<sha12>` (for example,
`ghcr.io/e6qu/sharecrop:0123456789ab`) with direct per-architecture
`<sha12>-arm64` and `<sha12>-amd64` tags. The workflow emits no `latest`, `main`,
or semantic-version tags. Deployments pin the immutable generic tag. Native
arm64 runners are required; without them, the release does not publish.

## Amazon ECS Fargate

The whole stack — an Amazon API Gateway HTTP API, its VPC Link and AWS Cloud Map
private integration, the Amazon ECS Fargate service (arm64), the tenant-specific
connection to the shared fck-rds PostgreSQL service, secrets, and IAM — is
provisioned by the Terraform in [`deploy/terraform/`](../deploy/terraform/)
(deploys into an existing VPC; see its README). It creates no Application Load
Balancer or Network Load Balancer. The standalone JSON task definitions in
`deploy/ecs/` (arm64 `runtimePlatform`, `REPLACE_*` placeholders for account,
region, roles, registry, tag, and secret ARNs) remain as a reference for a
non-Terraform deploy.

- **`sharecrop-serve.task-definition.json`** — the stateless service. Run it as
  an Amazon ECS service with the desired replica count. The Terraform module
  registers each task's address and port in AWS Cloud Map, and Amazon API
  Gateway uses `DiscoverInstances` over its VPC Link. The container health
  check runs `sharecrop healthcheck http://127.0.0.1:8080/healthz`; Amazon ECS
  publishes that health to AWS Cloud Map, so unhealthy tasks are excluded from
  request routing without an in-container shell or curl.
- **`sharecrop-migrate.task-definition.json`** — a standalone task that runs
  `sharecrop migrate up`. `serve` does **not** auto-migrate. The Terraform
  module schedules an AWS Step Functions workflow that waits for this
  standalone Amazon ECS task before it calls `UpdateService`; a failed task
  cannot roll the application service. The migration process requires only
  `DATABASE_URL` and the image-provided migrations directory; it does not
  require HTTP or token-signing configuration. PostgreSQL advisory transaction
  locking serializes duplicate cloud delivery, and the migration ledger keeps
  every SQL file at exactly one application.

### Configuration

| Variable                        | Source                    | Notes                                             |
| ------------------------------- | ------------------------- | ------------------------------------------------- |
| `DATABASE_URL`                  | Secrets Manager           | PostgreSQL DSN for Sharecrop's dedicated database. |
| `SHARECROP_ACCESS_TOKEN_SECRET` | Secrets Manager           | 32-byte access-token signing secret.              |
| `SHARECROP_HTTP_ADDR`           | image default `:8080`     | Listen address.                                   |
| `SHARECROP_MIGRATIONS_DIR`      | image default `/migrations` | Baked into the image.                           |
| `SHARECROP_WAZERO_CACHE_DIR`    | image default `/wazero-cache` | Baked AOT cache; leave as-is. Unset would make the guest compile at startup. |
| `SHARECROP_WASI_POOL_SIZE`      | optional                  | Guest pool size; defaults to GOMAXPROCS (task vCPUs). |
| `SHARECROP_INSECURE_COOKIES`    | optional                  | Leave unset in production (cookies stay Secure).  |
| `SHARECROP_ACCOUNT_TOKEN_DELIVERY` | optional               | Defaults to `log` (fail-closed).                  |
| `SHARECROP_SHAUTH_ISSUER`       | task configuration        | Exact HTTPS OpenID Connect issuer, including any trailing slash. |
| `SHARECROP_SHAUTH_CLIENT_ID`    | task configuration        | Sharecrop's confidential Shauth client ID.       |
| `SHARECROP_SHAUTH_CLIENT_SECRET` | Secrets Manager          | Sharecrop's confidential Shauth client secret.   |
| `SHARECROP_PUBLIC_URL`          | task configuration        | Exact public HTTPS origin; derives callback and logout URLs. |
| `SHARECROP_RELEASE_REVISION`    | task configuration        | Immutable 12–64 character lowercase hexadecimal commit or sha256 image digest exposed by the Shauth validation page. |

Shauth configuration is all-or-nothing. Register these exact client endpoints,
derived from `SHARECROP_PUBLIC_URL`:

- callback: `/api/auth/shauth/callback`
- Front-Channel Logout: `/api/auth/shauth/frontchannel-logout`
- Back-Channel Logout: `/api/auth/shauth/backchannel-logout`
- post-logout redirect bridge: `/auth/shauth/logout/complete`
- app-local signed-out URL: `/api/auth/signed-out`
- authenticated validation URL: `/auth/validation`

RP-Initiated Logout uses the issuer's discovered `end_session_endpoint` only
when it is on the configured issuer origin. The
provider returns through Sharecrop's fixed bridge, which ignores request input
and redirects only to Shauth's issuer-origin `/oauth/logout/complete` endpoint;
Shauth then completes its one-time correlation and returns to Sharecrop's
registered signed-out URL. The validation page exposes the verified Shauth
username, email, role, and immutable release revision for Shauth's serialized
browser acceptance checks.
The
provider-signed ID token and session identifier are retained in PostgreSQL,
not in the browser cookie. Front-Channel Logout correlates the exact issuer,
client, and provider session identifier. Back-Channel Logout replay claims and
refresh-family revocation are committed in one database transaction, so every
replica and a restarted service observe the same logout state. Shauth-managed
browser entry hides local password and reset controls; deployments without
Shauth retain those local account flows.

### Database

A stateless replica fleet needs a shared PostgreSQL database. The Terraform
module accepts the tenant-specific database URL secret managed by the shared
`fck-rds` service; another PostgreSQL deployment can supply the same coordinate.
Sharecrop must have its own database inside a shared PostgreSQL instance; it
must not share Shauth's, Ory Hydra's, or another application's migration
database.
The Terraform deployment workflow runs the `sharecrop-migrate` task before the
first `serve` rollout and every later workflow change. Serve and MCP processes
refuse to start while a migration baked into their image is absent from the
database.
