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
- **Deployment:** a slim multi-architecture container (arm64 primary) on Amazon
  ECS Fargate in private subnets, reached by an Amazon API Gateway HTTP API
  through a VPC Link and AWS Cloud Map, with state in Sharecrop's distinct
  database inside the shared PostgreSQL service. No Application Load Balancer
  or Network Load Balancer is provisioned. The guest's machine code is baked
  into the image as a wazero AOT cache, so the server does no
  compile at startup. Every merge publishes an immutable 12-character commit-SHA
  manifest to the GitHub Container Registry, with direct arm64 and amd64 image
  tags and no mutable or semantic-version tags. The newest 20 releases are
  retained. See
  [docs/deployment.md](./docs/deployment.md).
- **Shared environment deployment:** Terraform accepts an existing Amazon
  Elastic Container Service cluster ARN, so the service can run in the shared
  `dev` cluster without creating another cluster or network path.
- **DNS integration:** Terraform configures the regional Amazon API Gateway
  custom domain and exposes its target domain and hosted-zone ID so an
  environment can create the exact Route 53 alias.
- **Monitoring integration:** Terraform exposes the actual CloudWatch Logs group
  used by serve tasks and the Amazon API Gateway access-log group. Detailed
  route metrics and bounded burst/rate throttles are enabled.
- **Provider compatibility:** The deployment module requires HashiCorp AWS
  provider 6.x, matching the shared `dev` environment and the other deployed
  service modules.
- **Health routing:** the distroless container's binary probes the real
  `/healthz` endpoint. Amazon ECS publishes task and container health to AWS
  Cloud Map, and Amazon API Gateway routes only to healthy discovered tasks.
  Terraform waits for steady state and unhealthy deployments roll back.

## State

The deployment used private Amazon ECS Fargate tasks without public IP
addresses. Amazon API Gateway reached them through a VPC Link and discovered
their address and port from AWS Cloud Map SRV registrations. The public
execute-api endpoint was disabled, the custom domain was TLS-only, and the
default route applied explicit throttles, access logs, and detailed metrics.
Security-group rules admitted the HTTP port only from the VPC Link. A policy
gate rejected any Application Load Balancer, Network Load Balancer, public task
IP, or missing private-ingress resource from the Terraform module.

The active task audited and repaired Sharecrop's complete Shauth relying-party
contract. Its acceptance boundary covered direct entry and Apps-catalog launch,
automatic SSO, external-identity provisioning, authenticated user identity,
fail-closed protected resources, app-local signed-out return, and coordinated
RP-Initiated, Front-Channel, and Back-Channel Logout against real Shauth, Ory
Hydra, and PostgreSQL.

Shauth is an additional browser identity provider. A verified OpenID Connect
issuer/subject pair is persisted independently from mutable profile claims and
receives the same rotating Sharecrop session as a local login. Local passwords
and first-party tokens remain available. Existing password accounts are never
linked to a new external identity merely because their email addresses match.
The callback uses PKCE, nonce/state validation, an authenticated short-lived
transaction cookie, and exact HTTPS issuer/public URL coordinates. It retains
the provider-signed ID token and optional `sid` in PostgreSQL for
RP-Initiated Logout without exposing them in the browser cookie. In the
production WASI deployment, Shauth authorization, callback, logout,
Back-Channel Logout, and signed-out routes run on the native host boundary
because OpenID Connect discovery and token exchange require outbound HTTPS and
logout state is shared by every replica; the rest of the application remains
hosted by the WASI guest pool. When Shauth is configured, application entry routes require the
Sharecrop refresh-session cookie and redirect a new visitor to Shauth, so an
Apps-catalog launch and a direct application URL start the same identity flow.

Shauth Back-Channel Logout validated the exact issuer, audience, signature,
expiry, standard logout event, prohibited `nonce`, freshness, and either `sid`
or `sub`. PostgreSQL atomically claimed each logout-token `jti` and revoked the
matching active refresh-token families, so replay protection survived process
and replica changes. Browser logout revoked the local refresh family before
returning the issuer-origin end-session URL with the provider-signed ID token
hint and exact `/api/auth/signed-out` redirect. The signed-out landing revoked
any residual local refresh family and did not restart authentication. The
logout verifier cached provider discovery and its remote key set while retaining
normal signing-key rotation behavior. Shauth Front-Channel Logout also revoked
the exact issuer/session-ID relationship and returned a non-cacheable,
frame-safe completion document.

When Shauth was configured, the browser hid local registration, password reset,
and token entry, while programmatic first-party credentials remained supported.
The application shell and protected browser API rejected revoked refresh-token
families even when a previously minted access token was still present. External
identity provisioning used the immutable issuer/subject pair; Shauth's optional
email-verification claim was not treated as mandatory or used to link an
existing account.

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
merge. The Shauth integration passed the frontend build, full Go suite,
WASI bridge generation checks, PostgreSQL integration and HTTP suites, and
native/WASI scenario parity. A real browser suite against Shauth commit
`15302f47330fd531536e366b2befc1d370ed7e0d`, Ory Hydra v26.2.0, PostgreSQL
17.5, and the production WASI binary passed direct entry, Apps-catalog entry,
automatic SSO, identity provisioning, account display, app-local logout,
provider-session termination, old-token rejection, and return entry without a
second login or consent. All 62 general browser cases passed with retries
disabled; the two previously timing-sensitive paths also passed ten focused
stress iterations without retries. Authentication-operation rate limits were
isolated per path and client IP so registration or recovery traffic could not
starve login traffic for users behind the same NAT.
The release publisher verified that each architecture tag was a direct OCI
image manifest and that the generic tag contained exactly Linux amd64 and Linux
arm64 before retaining the newest 20 commit-SHA releases.
The Sharecrop command suite, generation checks, policy checks, release contract,
TypeScript checks, WASI bridge checks, lint, vet, Go/Deno tests, Terraform
formatting, and provider-backed Terraform validation passed after the private
ingress replacement.

## Blocking issues

None.
