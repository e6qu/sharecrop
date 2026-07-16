# Status

## Implemented surface

- A Go HTTP API (`internal/http`) over domain services (`internal/task`,
  `internal/ledger`, `internal/assets`, `internal/submission`, `internal/org`,
  ...), an Elm browser client, an MCP interface at `/mcp` (Streamable HTTP with
  SSE), scoped agent and organization-wide credentials, and a generated OpenAPI
  document (`docs/openapi.json`).
- **One store implementation** (`internal/db`), engine-neutral behind a small
  `db.Querier`/`Beginner` handle abstraction, parameterised only by SQL engine:
  Postgres in production, SQLite (via ncruces) in the browser demo. There is no
  separate browser storage adapter — `internal/wasmdemo` is deleted.
- **One application, two runtimes from the same source:**
  - The **browser demo** (`cmd/sharecrop-wasm`, `js/wasm`, GitHub Pages) runs the
    real mux + domain services over in-browser SQLite.
  - The **backend** runs the same app server-side through the WASI guest pool.
    This is the production default: `cmd/sharecrop serve` embeds the `wasip1` app
    guest (`internal/wasiguest`, built by `make wasi-app-guest` as part of
    `make build`) and hosts it under a wazero runtime, dispatching its store
    calls to Postgres via `storehost`. `SHARECROP_WASI_MODE=native` runs the
    in-process mux instead; a binary built without the guest runs native.
- **Deployment:** a slim multi-arch container (arm64 primary) on AWS ECS Fargate,
  stateless behind a load balancer with state in Postgres. The guest's machine
  code is baked into the image as a wazero AOT cache, so the server does no
  compile at startup. Images publish to the GitHub Container Registry on merge,
  versioned by conventional commits (no `:latest`). See
  [docs/deployment.md](./docs/deployment.md).

## State

Shauth is an additional browser identity provider. A verified OpenID Connect
issuer/subject pair is persisted independently from mutable profile claims and
receives the same rotating Sharecrop session as a local login. Local passwords
and first-party tokens remain available. Existing password accounts are never
linked to a new external identity merely because their email addresses match.
The callback uses PKCE, nonce/state validation, an authenticated short-lived
transaction cookie, and an HTTPS-only issuer/public URL configuration.

Both the single-store-implementation program and the WASI-production-hosting
program are complete. Recent work hardened the production-default WASI path:
real randomness and clock in the guest, per-client rate limiting and MCP origin
checks (the request bridge now carries RemoteAddr and Host), fixed an MCP SSE
pool-exhaustion denial of service, forwarded request-shaping env into the guest,
and raised the bridge frame limit above the request-body limit while bounding the
host-side body read. The container/deploy work then baked the wazero AOT cache,
slimmed the image, and added the ghcr release workflow.

## Test status

PR CI runs format/contract/policy/type checks, Go unit and integration tests,
HTTP end-to-end tests, shared scenario parity against both SQL engines, and
Playwright browser tests. The Release workflow builds and publishes the image on
merge. The Shauth integration passed the frontend build, complete Go suite,
Terraform validation, and WASI bridge generation checks.

## Blocking issues

None. GitHub Pages `deploy-pages` occasionally fails transiently after a merge
and clears on retry; it is not caused by repository code.
