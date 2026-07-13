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

```sh
# Push both per-arch images and assemble the manifest:
tools/build_container.sh sharecrop:1.4.0
#   -> sharecrop:1.4.0-arm64, sharecrop:1.4.0-amd64, sharecrop:1.4.0 (manifest)

# Local single-arch build for testing (arm64, loaded into the local docker):
PUSH=false tools/build_container.sh sharecrop:dev
```

The build is fast to load and fast to build:

- **Small image.** The runtime is `distroless/static` plus one static
  (CGO-free) binary — no shell, no libc, ~30 MB total. Quick to pull, small
  attack surface.
- **No emulation.** The `Dockerfile` builds the arch-independent parts (the
  `wasip1` guest, which is wasm; the committed `web/static` frontend) once on the
  native build platform and only cross-compiles the final host binary per arch,
  so a multi-arch build does not pay QEMU costs.
- **Slow-cold-start note.** wazero compiles the ~12 MB embedded guest at process
  start, so a fresh task takes a moment to become ready (a one-time cost per task
  on long-lived Fargate tasks). For faster task launch, enable
  [SOCI](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/container-considerations.html)
  lazy image loading on the cluster.

The frontend assets embedded into the binary are the committed `web/static/*`.
Run `make frontend` before a release build if the UI changed, the same as for
`make build`.

## ECS Fargate

Task definitions are in `deploy/ecs/` (arm64 `runtimePlatform`). Replace the
`REPLACE_*` placeholders (account, region, roles, registry, tag, secret ARNs).

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
| `SHARECROP_WASI_POOL_SIZE`      | optional                  | Guest pool size; defaults to GOMAXPROCS (task vCPUs). |
| `SHARECROP_INSECURE_COOKIES`    | optional                  | Leave unset in production (cookies stay Secure).  |
| `SHARECROP_ACCOUNT_TOKEN_DELIVERY` | optional               | Defaults to `log` (fail-closed).                  |

### Database

A stateless replica fleet needs a shared database, so use **Aurora
PostgreSQL** (Serverless v2 idles cheaply and can scale to zero) fronted by
**RDS Proxy** — Fargate tasks come and go and RDS Proxy keeps the backend
connection count bounded. Run the `sharecrop-migrate` task against it before the
first `serve` rollout and on every schema change.
