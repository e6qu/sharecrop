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
  stateless behind a load balancer with state in single-AZ Amazon RDS for
  PostgreSQL. The guest's machine
  code is baked into the image as a wazero AOT cache, so the server does no
  compile at startup. Images publish to the GitHub Container Registry on merge,
  versioned by conventional commits (no `:latest`). See
  [docs/deployment.md](./docs/deployment.md).
- **Shared environment deployment:** Terraform accepts an existing Amazon
  Elastic Container Service cluster ARN, so the service can run in the shared
  `dev` cluster without creating another cluster or network path.
- **DNS integration:** Terraform exposes the Application Load Balancer DNS name
  and canonical hosted-zone ID so an environment can create a Route 53 alias
  record without reconstructing provider-specific values.
- **Monitoring integration:** Terraform exposes the actual CloudWatch Logs group
  used by serve tasks, including its provider-generated suffix.
- **Provider compatibility:** The deployment module requires HashiCorp AWS
  provider 6.x, matching the shared `dev` environment and the other deployed
  service modules.
- **ACM composition:** HTTPS listener creation is controlled by an explicit,
  plan-known `enable_https` input, so an environment can provision its ACM
  certificate and the Sharecrop service in one Terraform apply.

## State

Shauth is an additional browser identity provider. A verified OpenID Connect
issuer/subject pair is persisted independently from mutable profile claims and
receives the same rotating Sharecrop session as a local login. Local passwords
and first-party tokens remain available. Existing password accounts are never
linked to a new external identity merely because their email addresses match.
The callback uses PKCE, nonce/state validation, an authenticated short-lived
transaction cookie, and an HTTPS-only issuer/public URL configuration. In the
production WASI deployment, the Shauth authorization and callback routes run
on the native host boundary because OpenID Connect discovery and token exchange
require outbound HTTPS; the rest of the application remains hosted by the WASI
guest pool. When Shauth is configured, application entry routes require the
Sharecrop refresh-session cookie and redirect a new visitor to Shauth, so an
Apps-catalog launch and a direct application URL start the same identity flow.

Shauth back-channel logout validated signed logout tokens through provider
discovery, required the standard logout event and session/identity claims, and
revoked every active Sharecrop refresh-token family associated with the
verified issuer/subject. The endpoint was
`/api/auth/shauth/backchannel-logout`. PostgreSQL integration registrations
used unique addresses, so the complete suite remained repeatable on its
dedicated database instead of failing after an earlier run left real users.
The logout verifier cached provider discovery and its remote key set, while
the key set retained normal signing-key rotation behavior.

The migration command loaded only its database URL and migration directory, so
the one-off ECS migration task did not depend on HTTP or access-token runtime
configuration. The server and MCP transports verified that every migration
baked into the image had been applied before serving requests, preventing a
partially migrated database from presenting a healthy application whose
authentication callback failed later.

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
Terraform validation, and WASI bridge generation checks. Migration
configuration and current-schema checks passed unit and PostgreSQL integration
tests. Direct entry and Shauth Apps-catalog launch both completed the live
OpenID Connect callback and established a Sharecrop browser session.
Signed logout-token tests covered successful global local-session revocation,
missing `sid`, and prohibited `nonce` claims. The complete PostgreSQL
integration suite also passed on a database containing prior test data.
The HTTP end-to-end health harness implemented the complete authentication
service contract, and the tagged HTTP suite passed against PostgreSQL.

## Blocking issues

None.
