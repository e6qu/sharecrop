# WASM Backend Target

The backendless demo currently uses `site/demo/backend.js`, an in-browser fake backend. The target direction is to replace that JavaScript fake with a backend compiled from the Go codebase to a WASM binary. WASM is also a first-class production execution target for the Go backend when the host supplies explicit runtime adapters. The Go/WASM backend path must run the same Elm app on the deployed demo site and the same app against the server backend. Shared scenario parity tests now exist, so the WASM target can be evaluated against real behavior instead of relying only on route/shape checks.

## Finding

A Go/WASM backend is viable only after the application services can run against explicit host adapters. The WASM artifact must be produced by compiling Go packages to `js/wasm`; a JavaScript rewrite or generated fake backend is not the target. The current server process wires domain services through Postgres-backed stores, auth/session stores, rate-limit buckets, audit stores, notification stores, and MCP stores. A browser build cannot reuse pgx, migrations, process-local server wiring, or `net/http` handlers directly. A non-browser WASM host also needs explicit adapters instead of implicit process, filesystem, network, or database assumptions.

Current decision: keep `site/demo/backend.js` as the temporary backendless demo backend while building the Go/WASM backend target.
The shared scenario suite now covers multi-actor reservation approval, worker
submission, owner acceptance, payouts/tips, notifications, organization member
provisioning/listing/role/deactivation, privacy request resolution,
sensitive-field redaction state, and moderation report projection. That
coverage is the guardrail for replacing the fake backend without adding hidden
fallback behavior.

## Request Adapter And Storage Spike

`internal/wasmdemo` contains a narrow request-adapter spike. It classifies only
the privacy, moderation, saved-queue-view, task, notification, organization,
organization-member, and team route pairs:

- `POST /api/privacy-requests`
- `GET /api/admin/privacy-requests`
- `POST /api/moderation/reports`
- `GET /api/admin/moderation/reports`
- `POST /api/saved-queue-views`
- `GET /api/saved-queue-views`
- `POST /api/tasks`
- `GET /api/tasks/{task_id}`
- `GET /api/notifications`
- `POST /api/notifications/{notification_id}/read`
- `POST /api/organizations`
- `GET /api/organizations`
- `POST /api/organizations/{organization_id}/members`
- `GET /api/organizations/{organization_id}/members`
- `PATCH /api/organizations/{organization_id}/members/{user_id}/roles`
- `PATCH /api/organizations/{organization_id}/members/{user_id}/deactivate`
- `POST /api/organizations/{organization_id}/teams`
- `GET /api/organizations/{organization_id}/teams`
- `POST /api/teams`
- `GET /api/teams`

Unsupported methods and routes return explicit rejection results. The package
does not yet execute the full domain service graph or replace
`site/demo/backend.js`.

The package now also contains explicit browser-storage boundaries for privacy
requests, moderation triage records, saved queue views, tasks, small
task/submission attachment records, actor-scoped notifications, organizations,
organization members, organization-owned teams, and standalone teams. The
storage boundary is caller-provided; no in-memory store is selected by default.
Missing records, invalid keys, invalid states, invalid scopes, invalid privacy
request kinds, invalid task lifecycle values, invalid notification ownership,
invalid attachment parent kinds, invalid attachment counts, invalid attachment
sizes, invalid organization/member/team ownership, and storage read/write
failures return explicit rejected results. This is enough to prove the next
WASM path can persist the currently classified slices without adding hidden
fallback behavior.

The current request-handler steps use those storage boundaries for:

- `POST /api/privacy-requests`
- `GET /api/admin/privacy-requests`
- `POST /api/admin/moderation/reports/{report_id}/triage`
- `POST /api/saved-queue-views`
- `GET /api/saved-queue-views`
- `POST /api/tasks`
- `GET /api/tasks/{task_id}`
- `GET /api/notifications`
- `POST /api/notifications/{notification_id}/read`
- `POST /api/organizations`
- `GET /api/organizations`
- `POST /api/organizations/{organization_id}/members`
- `GET /api/organizations/{organization_id}/members`
- `PATCH /api/organizations/{organization_id}/members/{user_id}/roles`
- `PATCH /api/organizations/{organization_id}/members/{user_id}/deactivate`
- `POST /api/organizations/{organization_id}/teams`
- `GET /api/organizations/{organization_id}/teams`
- `POST /api/teams`
- `GET /api/teams`

The handlers reject missing storage, missing clocks, missing actor identity,
missing ID sources, unsupported routes, unsupported methods, invalid request
bodies, invalid privacy request kinds, invalid saved-queue scopes, and invalid
triage states. Notification handlers also reject invalid pagination and
actor/recipient mismatches. Organization handlers reject missing user resolvers
for email-based member provisioning rather than fabricating anonymous or
placeholder identities. They do not provide substitute stores for unimplemented
routes.

## Compile Check

The current Go codebase compiles to `js/wasm` for representative packages and
the main command:

- `GOOS=js GOARCH=wasm go test -c -o /private/tmp/sharecrop-core-schema-submission.test.wasm ./internal/submission`
- `GOOS=js GOARCH=wasm go test -c -o /private/tmp/sharecrop-http.test.wasm ./internal/http`
- `GOOS=js GOARCH=wasm go build -o /private/tmp/sharecrop-cmd.wasm ./cmd/sharecrop`

Observed artifact sizes on the local build were about 6.1 MB for the submission
test package, 12 MB for the HTTP test package, and 24 MB for the command build.
Plain `go test` produced WASM test binaries but could not execute them natively;
running those tests requires a JS/WASM test runner.

The compile check means basic Go/WASM compatibility is not the blocker. The
blockers are broader request adaptation, enough browser storage adapters,
deterministic seed and reset, startup size, and running the shared scenario
parity suite against a WASM request handler.

## Required Shape

- The backend artifact is a `.wasm` binary compiled from Go code with
  `GOOS=js GOARCH=wasm`.
- A `js/wasm` request adapter receives method, path, headers, and body from `fetch` interception.
- Domain/application services are constructed with explicit host adapters, not fake fallback stores.
- Browser storage adapters are explicit packages, likely backed by IndexedDB for persisted data and in-memory state only where the domain already treats state as process-local.
- Production WASM hosts must provide explicit storage, clock, identity/session,
  request, randomness, and networking adapters instead of relying on process
  globals or hidden substitute behavior.
- WASM tests run the shared scenario parity suite against the request adapter.
- Missing browser storage features fail at build or request time with clear errors.

## Adoption Gates

Replace `site/demo/backend.js` only after the WASM path can satisfy these gates:

1. It passes the shared scenario parity suite for task creation, selectors, comments, reservations, submission review, notifications, collectibles, and account-token flows.
2. It has no fallback behavior for unimplemented stores or handlers.
3. It has a deterministic reset path for demo data.
4. Bundle size and startup time are measured from a production build.
5. GitHub Pages deployment still passes the deployed routing check.

## Next Spike Step

The next WASM step is to add enough explicit browser-backed stores and request
handlers for submissions, ledgers, collectibles, account-token flows,
reservation/approval, and comments to run the shared scenario parity suite
against the WASM handler. If a missing slice is discovered, it should fail
loudly until that slice has an explicit browser storage adapter and handler.
