# Deployment

Sharecrop ships as one application built for two runtimes from the same source:

- **Browser demo** — `js/wasm`, storage backed by in-browser SQLite, served
  statically (GitHub Pages). Single user, entirely in-browser.
- **Backend** — the same app hosted server-side through the WASI guest pool
  (`cmd/sharecrop serve`, which embeds the `wasip1` app guest and runs it under
  the wazero runtime), with all state in Postgres. Deployed as **stateless
  replicas on ECS Fargate (arm64) behind a load balancer**.

The backend is stateless: every durable thing (tasks, submissions, ledger,
audit, notifications, MCP sessions) lives in Postgres, SSE is delivered by DB
polling, and the only per-process state (in-memory rate-limit buckets) is
defense-in-depth, not correctness. So replicas scale horizontally with no
affinity — the load balancer can round-robin freely.

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
tools/build_container.sh ghcr.io/e6qu/sharecrop:1.4.0 arm64   # -> :1.4.0-arm64
tools/build_container.sh ghcr.io/e6qu/sharecrop:1.4.0 amd64   # -> :1.4.0-amd64

# Assemble the multi-arch manifest from the per-arch images:
tools/build_container.sh ghcr.io/e6qu/sharecrop:1.4.0 manifest  # -> :1.4.0

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

1. Compute the next version from conventional commits since the last tag
   (`tools/next_version.sh`): **patch by default**, `feat` → minor, `!`/`BREAKING
   CHANGE` → major. Every merge bumps at least a patch, so every merge builds.
2. Build each arch on a native runner (arm64 on `ubuntu-24.04-arm`, amd64 on
   `ubuntu-24.04`) and push the per-arch images to the GitHub Container Registry.
3. Assemble the manifest, create the git tag and GitHub release, and prune old
   images to the newest 25 release versions (`tools/prune_ghcr_versions.sh`,
   best-effort).

Images are published to `ghcr.io/<owner>/<repo>:<version>` (e.g.
`ghcr.io/e6qu/sharecrop:v1.4.0`) with per-arch `…-arm64`/`…-amd64` tags. There is
**no `:latest`** — deployments pin an explicit version. Because merges squash to
the PR title, PR titles should follow the conventional-commit format so
`feat`/breaking changes bump the right component (anything else is a patch).
Native arm64 runners are required; without them, switch the matrix to emulated
builds.

## ECS Fargate

The whole stack — ALB, the ECS Fargate service (arm64), Aurora Serverless v2 +
RDS Proxy, secrets, and IAM — is provisioned by the Terraform in
[`deploy/terraform/`](../deploy/terraform/) (deploys into an existing VPC; see its
README). The standalone JSON task definitions in `deploy/ecs/` (arm64
`runtimePlatform`, `REPLACE_*` placeholders for account, region, roles, registry,
tag, and secret ARNs) remain as a reference for a non-Terraform deploy.

- **`sharecrop-serve.task-definition.json`** — the stateless service. Run it as
  an ECS Service with the desired replica count behind an Application Load
  Balancer. Target-group health check: `GET /healthz` (the image is distroless,
  so there is no in-container curl — health checking is external, via the ALB).
- **`sharecrop-migrate.task-definition.json`** — a one-off task that runs
  `sharecrop migrate up`. `serve` does **not** auto-migrate (so replicas never
  race on schema changes); run this task once as part of a deploy, before
  rolling the service, or as a CodeDeploy/pipeline step.

### Configuration

| Variable                        | Source                    | Notes                                             |
| ------------------------------- | ------------------------- | ------------------------------------------------- |
| `DATABASE_URL`                  | Secrets Manager           | Postgres DSN. Point at **RDS Proxy** so many replicas share a bounded connection pool. |
| `SHARECROP_ACCESS_TOKEN_SECRET` | Secrets Manager           | 32-byte access-token signing secret.              |
| `SHARECROP_HTTP_ADDR`           | image default `:8080`     | Listen address.                                   |
| `SHARECROP_MIGRATIONS_DIR`      | image default `/migrations` | Baked into the image.                           |
| `SHARECROP_WAZERO_CACHE_DIR`    | image default `/wazero-cache` | Baked AOT cache; leave as-is. Unset would make the guest compile at startup. |
| `SHARECROP_WASI_POOL_SIZE`      | optional                  | Guest pool size; defaults to GOMAXPROCS (task vCPUs). |
| `SHARECROP_INSECURE_COOKIES`    | optional                  | Leave unset in production (cookies stay Secure).  |
| `SHARECROP_ACCOUNT_TOKEN_DELIVERY` | optional               | Defaults to `log` (fail-closed).                  |

### Database

A stateless replica fleet needs a shared database, so use **Aurora
PostgreSQL** (Serverless v2 idles cheaply and can scale to zero) fronted by
**RDS Proxy** — Fargate tasks come and go and RDS Proxy keeps the backend
connection count bounded. Run the `sharecrop-migrate` task against it before the
first `serve` rollout and on every schema change.
