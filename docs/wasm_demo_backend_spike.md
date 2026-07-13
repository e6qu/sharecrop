# WASM Backend Target

> **Status: completed / historical — superseded.** This describes the old
> `internal/wasmdemo` browser-storage architecture, which is deleted. The browser
> demo now runs `internal/db` over in-browser SQLite (ncruces), and production is
> the WASI guest pool — not the `js/wasm` request-adapter path described here. For
> current state see [README.md](../README.md) and [deployment.md](./deployment.md).

The deployed static demo runs a backend compiled from the Go codebase to a WASM
binary. WASM is also a first-class production execution target for the Go
backend when the host supplies explicit runtime adapters. The Go/WASM backend
path runs the same Elm app on the deployed demo site and the same app against
the server backend. Shared scenario parity tests evaluate that path against real
behavior instead of relying only on route/shape checks.

## Finding

A Go/WASM backend is viable only after the application services can run against
explicit host adapters. The WASM artifact must be produced by compiling Go
packages to `js/wasm`; a JavaScript rewrite or generated fake backend is not the
target. The current server process wires domain services through Postgres-backed
stores, auth/session stores, rate-limit buckets, audit stores, notification
stores, and MCP stores. A browser build cannot reuse pgx, migrations,
process-local server wiring, or `net/http` handlers directly. A non-browser WASM
host also needs explicit adapters instead of implicit process, filesystem,
network, or database assumptions.

Current decision: `site/demo/index.html` defaults to the compiled Go/WASM
backend, and is the only backend it supports.

## Unified architecture: one backend, two hosts

The browser demo used to be a from-scratch reimplementation of the API surface
(`internal/wasmdemo`'s old request-adapter/handler files), which let this exact
class of bug happen: an invariant enforced by the real Postgres store (e.g.
"credit reward must be funded before opening") was never ported to the demo's
independent reimplementation, so a task could reach "open" in the demo with an
unfunded reward.

The demo is now unified onto the same code the production backend runs:

```
site/demo/wasm-host.js (browser host: localStorage, JS Date, nextID counter)
        │  storageGet / storagePut / storageHas / now / nextID
        ▼
cmd/sharecrop-wasm/main_js_wasm.go
        │  receives (method, path, body, authorization) from JS
        │  constructs httptest.NewRequest + httptest.NewRecorder,
        │  replays/captures the sharecrop_refresh_token cookie
        ▼
internal/http's *http.ServeMux — the REAL mux, unmodified
        │  mux.ServeHTTP(recorder, request) — same code path cmd/sharecrop uses
        ▼
the real domain services (internal/task, internal/submission, internal/org, ...)
        │  unmodified — each only depends on its own Store interface
        ▼
internal/wasmdemo's browserstore_*.go — Store implementations backed by
     browser_storage.go's BrowserStorage primitives (a key/value Put/Get
     contract), instead of Postgres
```

Routing, validation, business rules, and error shapes are shared with the real
backend, not duplicated. Only two things are demo-specific: the browser-backed
`Store` implementations (`internal/wasmdemo/browserstore_*.go`) and the thin
JS-request → `httptest` → `mux.ServeHTTP` → response glue in
`cmd/sharecrop-wasm/main_js_wasm.go`.

### The nine browser-storage-backed stores

`internal/wasmdemo` implements a real `Store` interface for each domain package
the mux depends on, each exercised through the real, unmodified domain
`Service`:

- `browserstore_auth.go` — `auth.Store`: real password hashing
  (Argon2id via `golang.org/x/crypto/argon2`), real refresh-token rotation
  with reuse detection, signup credit grants.
- `browserstore_notification.go` — `notification.Store`.
- `browserstore_agent.go` — `agent.Store` (agent credentials).
- `browserstore_orgcred.go` — `orgcred.Store` (organization-wide credentials).
- `browserstore_org.go` — `org.Store` (organizations, teams, membership/roles).
- `browserstore_ledger.go` — `ledger.Store`: funding, refunds, accept/request-
  changes/reject submission review flows (credit payouts, tips, idempotent
  replay).
- `browserstore_task.go` / `browserstore_task_shared.go` /
  `browserstore_task_series.go` — `task.Store`: task CRUD, reservations,
  task/series comments, task series.
- `browserstore_assets.go` — `assets.Store` (collectibles, collectible reward
  funding/refunding).
- `browserstore_submission.go` — `submission.Store`, including
  `RedactSensitiveFields` (see Privacy below).

`browser_storage.go` holds the shared low-level primitives every store above is
built on: the `BrowserStorage`/`StorageKey` key-value contract, the generic
`loadStringIndex`/`appendStringIndex`/`removeFromStringIndex` list helpers, and
the `HandlerClock`/`InteractionIDSource` interfaces a host must satisfy.

### Auth: real bearer tokens, not a fake actor

The real `internal/http` mux authenticates every request via
`Authorization: Bearer <token>` (`auth.ParseAccessToken` + a real
`SubjectVerifier`), and login/register set a `sharecrop_refresh_token` HttpOnly
cookie that `/api/auth/refresh` reads back. `site/demo/wasm-host.js` forwards
whatever `Authorization` header value it has straight through to
`sharecropHandleRequest`'s 4th argument — no fake actor-id scheme. Since
`sharecropHandleRequest` builds synthetic `httptest.NewRequest`/
`httptest.NewRecorder` pairs (no real network), `main_js_wasm.go` keeps a
package-level `currentRefreshCookie *http.Cookie` that replays the refresh
cookie on every request and captures updates from the response — safe because
`js/wasm` is single-threaded, one demo session per browser tab.

### Seeding: a real seed routine, not hand-poked fixtures

`internal/wasmdemo/seed.go`'s `SeedDemoScenario` seeds the fixed demo cast (5
users, one organization, 4 tasks in various states) by calling the real
`auth.Service.Register`, `org.Service.CreateOrganization`/`ProvisionMember`, and
`task.Service.Create`/`Open`/`ledger.Service.FundTask` — the same calls a real
user would make, not raw JSON pushed into storage. It's idempotent (detected via
whether the seed admin's account already exists) and returns a ready-to-replay
refresh-token cookie so the demo's first `/api/auth/refresh` call succeeds
immediately, preserving the "already logged in" UX.

Because seeding goes through real services, the seeded credit numbers are real
ledger arithmetic, not chosen constants: the demo's fixed admin account (mara)
shows a 70-credit balance (100 signup grant − 30 allocated to a seeded task),
and "Field Operations" shows a 100-credit organization balance.

### RuntimeState: what's persistent vs. in-memory

`internal/http.DefaultRuntimeState(bootstrapAdmins)` builds the same in-memory
defaults `New()` uses for 8 `RuntimeState` fields (rate limiters, MCP sessions,
audit, saved queue views, platform admin, moderation triage, plus
privacy-request tracking). The demo overrides two of them with persistent,
browser-storage-backed implementations:

- `NotificationService` — a real `notification.Service` over
  `NotificationBrowserStore`.
- `PrivacyService` — `httpserver.NewMemoryPrivacyService(submissionStore)`. The
  in-memory privacy service only tracks privacy _requests_; it has no submission
  data of its own, so redaction is delegated through a small
  `SensitiveFieldRedactor` interface to whatever actually holds submissions —
  `SubmissionBrowserStore.RedactSensitiveFields` for the demo,
  `internal/db.PrivacyStore`'s SQL
  `update ... where retention =
  'delete_on_request'` for production.

The other 8 fields reset every page reload (acceptable: nothing in the demo
depends on their state surviving a reload; the seeded task/org/user data, which
does need to persist, lives in browser storage instead).

## Host Adapter Shape

A host must provide storage (`BrowserStorage`), a clock (`HandlerClock`), and an
id source (`InteractionIDSource`) before the browser stores can run. Two
concrete host implementations exist:

- `site/demo/wasm-host.js` is the browser host. It backs storage with
  `window.localStorage`, uses `Date.now()` for the clock, and forwards the
  `Authorization` header from every intercepted `/api/*` XHR call straight
  through to Go — it does not resolve "who is acting" itself.
- `tools/wasm_runtime_loader.ts` (`createHost`) is the non-browser reference
  host. It runs under Deno, with no browser APIs available, and is exercised by
  `deno task check:scenario-parity:wasm` and `deno task measure:wasm`.

`createHost`'s `HostFunctions` contract is `storageHas`, `storageGet`,
`storagePut`, `now`, `nextID` — no `actorID`/`setActor`/`userIDForEmail`, since
the mux resolves identity from the bearer token, not the host.

It is a test/measurement host, not a production one. Before it could back a real
non-browser deployment, it would need:

- **Persistent storage.** `createHost` backs storage with an in-memory `Map`
  discarded on process exit; a production host needs a file or real database
  behind the same `storageHas`/`storageGet`/`storagePut` contract.
- **A real clock.** `createHost` returns a fixed timestamp for deterministic
  runs; a production host needs a real system clock.
- **Real IDs.** `createHost.nextID` returns sequential, per-process predictable
  ids. The one place a browser store still calls into the host's id source
  (`NextLedgerEntryID`, for ledger entries that round-trip through
  `core.ParseLedgerEntryID`) is generated directly in Go instead
  (`core.NewLedgerEntryID()`) precisely because a sequential counter id isn't
  UUID-shaped — a production non-browser host would need the same care for
  anything it still generates itself.

## Compile Check

The current Go codebase compiles to `js/wasm` for the main command:

- `GOOS=js GOARCH=wasm go build -o site/demo/sharecrop-wasm-backend.wasm ./cmd/sharecrop-wasm`

`deno task check:scenario-parity:wasm -- --wasm site/demo/sharecrop-wasm-backend.wasm`
loads the compiled Go WASM binary through Go's `wasm_exec.js`, verifies the
`sharecropWasmBackendStatus`, `sharecropConfigureHost`, and
`sharecropHandleRequest` exports, verifies that requests fail before host
configuration, configures the host (which seeds the demo scenario), and runs the
shared scenario parity suite through the exported request handler, forwarding a
real bearer token the same way the real-backend scenario client does.

`deno task wasm:demo:build` builds `cmd/sharecrop-wasm` into
`site/demo/sharecrop-wasm-backend.wasm` and copies Go's `wasm_exec.js` into the
demo directory. The Pages workflow runs that task before uploading the static
site. The generated `.wasm` and `wasm_exec.js` files are not committed.

## Runtime Measurements

`deno task measure:wasm -- --wasm <compiled.wasm> [--requests-per-route <n>]`
loads a compiled `cmd/sharecrop-wasm` artifact through the non-browser reference
host, configures it, and reports artifact size, startup time, host process
memory, and per-route request latency.

A local run against `site/demo/sharecrop-wasm-backend.wasm` reported:

- Artifact size: 11.73 MiB (12,295,906 bytes) — larger than earlier measurements
  from before the cutover, reflecting the real domain services and browser
  stores now compiled in.
- Startup: about 46ms from `WebAssembly.instantiate` through the first
  `sharecropWasmBackendStatus` call reporting `unconfigured`, plus about 900ms
  for `sharecropConfigureHost` — most of that is `SeedDemoScenario` making real
  service calls (register 5 users, create an organization, provision 2 members,
  create/fund/open 4 tasks), not WASM overhead itself.
- Host process memory (`Deno.memoryUsage()`, the Deno process hosting the WASM
  runtime, not WASM linear memory specifically): resident set size grew from
  about 53 MiB before loading the artifact to about 179 MiB after host
  configuration, and to about 197 MiB after 25 requests (5 routes x 5 requests
  per route).
- Request latency for `GET /api/users`, `GET /api/organizations`,
  `GET /api/tasks?scope=public`, `GET /api/tasks/{task_id}`, and
  `GET /api/credits/balance`, all authenticated with a real bearer token against
  the in-memory reference host: mean latency under 0.6ms per route, well under
  2ms even at the max observed.

These numbers describe the non-browser reference host under Deno on a
development machine, not a deployed browser or a persistent-storage production
host; they establish a measurement method and a baseline, not a production SLA.

## Required Shape

- The backend artifact is a `.wasm` binary compiled from Go code with
  `GOOS=js GOARCH=wasm`.
- The real `internal/http` mux handles every request; no parallel routing logic
  exists for the demo.
- Domain/application services are constructed with explicit host adapters
  (browser storage, a clock, an id source), not fake fallback stores.
- Production WASM hosts must provide explicit storage, clock, identity, and
  networking adapters instead of relying on process globals or hidden substitute
  behavior.
- WASM tests run the shared scenario parity suite against the real mux.

## Adoption Gates

The static demo default uses the compiled Go/WASM path because:

1. It passes the shared scenario parity suite for task creation, task series,
   comments, reservations, submission review, notifications, organizations,
   privacy/moderation, collectibles, and account-token flows — the same suite
   the real backend runs, driven through the real mux.
2. It has no fallback behavior for unimplemented stores or handlers.
3. It has a deterministic reset-and-reseed path for demo data
   (`SeedDemoScenario`, run fresh on every page load).
4. GitHub Pages deployment builds the WASM artifacts and the deployed routing
   check verifies the demo entrypoint and generated WASM runtime assets.

## Next Steps

Track A (this document) is complete: the browser demo and the production backend
now share one implementation of every domain service, differing only in which
`Store` backs them (Postgres vs. browser storage) and which host adapters supply
storage/clock/ids.

Track B — replacing `cmd/sharecrop`'s native `net/http.ListenAndServe` with a
WASI-hosted guest binary for real, horizontally-scaled production deployment —
is a separate, harder effort tracked in
`docs/wasi_production_hosting_spike_plan.md`. It is not blocked by, and does not
block, this document's work: the browser-storage `Store` implementations here
are demo-only and single-actor by nature (one JS thread per browser tab), not a
production store.
