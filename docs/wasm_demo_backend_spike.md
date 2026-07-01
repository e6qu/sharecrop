# WASM Demo Backend Spike

The backendless demo currently uses `site/demo/backend.js`, an in-browser fake backend. Shared scenario parity tests now exist, so a WASM replacement can be evaluated against the same behavior instead of relying only on route/shape checks.

## Finding

A Go/WASM demo backend is viable only after the application services can run against explicit browser storage adapters. The current production server wires domain services through Postgres-backed stores, auth/session stores, rate-limit buckets, audit stores, notification stores, and MCP stores. A browser build cannot reuse pgx, migrations, process-local server wiring, or `net/http` handlers directly.

Current decision: keep `site/demo/backend.js` as the backendless demo backend.
The shared scenario suite now covers multi-actor reservation approval, worker
submission, owner acceptance, payouts/tips, notifications, privacy request
resolution, sensitive-field redaction state, and moderation report projection.
That coverage is a better guardrail than starting a WASM replacement before
browser storage adapters exist.

## Request Adapter And Storage Spike

`internal/wasmdemo` contains a narrow request-adapter spike. It classifies only
the privacy, moderation, and saved-queue-view route pairs:

- `POST /api/privacy-requests`
- `GET /api/admin/privacy-requests`
- `POST /api/moderation/reports`
- `GET /api/admin/moderation/reports`
- `POST /api/saved-queue-views`
- `GET /api/saved-queue-views`
- `POST /api/tasks`
- `GET /api/tasks/{task_id}`

Unsupported methods and routes return explicit rejection results. The package
does not execute domain services or replace `site/demo/backend.js`.

The package now also contains explicit browser-storage boundaries for privacy
requests, moderation triage records, saved queue views, tasks, and small
task/submission attachment records. The storage boundary is caller-provided; no
in-memory store is selected by default. Missing records, invalid keys, invalid
states, invalid scopes, invalid privacy request kinds, invalid task lifecycle
values, invalid attachment parent kinds, invalid attachment counts, invalid
attachment sizes, and storage read/write failures return explicit rejected
results. This is enough to prove the next WASM path can persist five classified
slices without adding hidden fallback behavior.

The current request-handler steps use those storage boundaries for:

- `POST /api/privacy-requests`
- `GET /api/admin/privacy-requests`
- `POST /api/admin/moderation/reports/{report_id}/triage`
- `POST /api/saved-queue-views`
- `GET /api/saved-queue-views`
- `POST /api/tasks`
- `GET /api/tasks/{task_id}`

The handlers reject missing storage, missing clocks, missing actor identity,
missing ID sources, unsupported routes, unsupported methods, invalid request
bodies, invalid privacy request kinds, invalid saved-queue scopes, and invalid
triage states. They do not replace the backendless JavaScript demo and do not
provide substitute stores for unimplemented routes.

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

- A `js/wasm` request adapter receives method, path, headers, and body from `fetch` interception.
- Domain/application services are constructed with browser storage adapters, not fake fallback stores.
- Browser storage adapters are explicit packages, likely backed by IndexedDB for persisted data and in-memory state only where the domain already treats state as process-local.
- WASM tests run the shared scenario parity suite against the request adapter.
- Missing browser storage features fail at build or request time with clear errors.

## Adoption Gates

Do not replace `site/demo/backend.js` until the WASM path can satisfy these gates:

1. It passes the shared scenario parity suite for task creation, selectors, comments, reservations, submission review, notifications, collectibles, and account-token flows.
2. It has no fallback behavior for unimplemented stores or handlers.
3. It has a deterministic reset path for demo data.
4. Bundle size and startup time are measured from a production build.
5. GitHub Pages deployment still passes the deployed routing check.

## Next Spike Step

The next WASM step is to add enough explicit browser-backed stores and request
handlers for task creation, organizations, teams, submissions, ledgers,
collectibles, notifications, and account-token flows to run the shared scenario
parity suite against the WASM handler. If that requires broad store rewrites or
hidden substitute behavior, keep the JavaScript demo backend and expand shared
parity tests instead.
