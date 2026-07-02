# WASM Backend Target

The deployed static demo now defaults to a backend compiled from the Go codebase
to a WASM binary. WASM is also a first-class production execution target for the
Go backend when the host supplies explicit runtime adapters. The Go/WASM backend
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
backend. `site/demo/backend.js` is no longer loaded by the demo entrypoint; it
remains only as legacy parity/test material while the compiled WASM path is the
active static-demo backend. The shared scenario suite covers multi-actor
reservation approval, worker submission, owner acceptance, payouts/tips,
notifications, organization member provisioning/listing/role/deactivation,
privacy request resolution, sensitive-field redaction state, moderation report
projection, collectibles, account-token shapes, agent credentials, and
admin-operation shapes. That coverage is the guardrail against hidden fallback
behavior.

## Request Adapter And Storage Spike

`internal/wasmdemo` contains a request-adapter spike. It classifies the privacy,
moderation, saved-queue-view, task, notification, organization,
organization-member, team, comment, reservation, submission, and ledger route
pairs:

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
- `GET /api/tasks/{task_id}/comments`
- `POST /api/tasks/{task_id}/comments`
- `GET /api/submissions/{submission_id}/comments`
- `POST /api/submissions/{submission_id}/comments`
- `POST /api/tasks/{task_id}/reservations`
- `GET /api/tasks/{task_id}/reservations`
- `POST /api/tasks/{task_id}/reservations/{reservation_id}/approve`
- `POST /api/tasks/{task_id}/reservations/{reservation_id}/decline`
- `POST /api/tasks/{task_id}/reservations/{reservation_id}/cancel`
- `POST /api/tasks/{task_id}/submissions`
- `GET /api/tasks/{task_id}/submissions`
- `GET /api/users/{user_id}/submissions`
- `POST /api/tasks/{task_id}/submissions/{submission_id}/accept`
- `GET /api/credits/balance`
- `GET /api/credits/ledger`
- `GET /api/organizations/{organization_id}/credits/balance`
- `GET /api/organizations/{organization_id}/credits/ledger`
- `GET /api/agent-credentials`
- `POST /api/agent-credentials`
- `POST /api/agent-credentials/{credential_id}/revoke`

Unsupported methods and routes return explicit rejection results. The package
does not yet execute the full production server graph, because browser and other
WASM hosts must provide explicit storage, clock, identity/session, request,
randomness, and networking adapters.

The package now also contains explicit browser-storage boundaries for users,
account tokens, platform admins, audit events, collectibles, agent credentials,
privacy requests, moderation triage records, saved queue views, tasks, small
task/submission attachment records, actor-scoped notifications, organizations,
organization members, organization-owned teams, standalone teams, comments,
reservations, submissions, and ledger entries. The storage boundary is
caller-provided; no store is selected by default. Missing records, invalid keys,
invalid states, invalid scopes, invalid privacy request kinds, invalid task
lifecycle values, invalid notification ownership, invalid attachment parent
kinds, invalid attachment counts, invalid attachment sizes, invalid
organization/member/team ownership, invalid reservation transitions, invalid
submission states, invalid ledger owners, and storage read/write failures return
explicit rejected results.

The current request-handler steps use those storage boundaries for:

- `POST /api/privacy-requests`
- `GET /api/privacy-requests`
- `GET /api/admin/privacy-requests`
- `POST /api/admin/privacy-requests/{request_id}/resolve`
- `POST /api/admin/privacy-retention/run`
- `POST /api/admin/moderation/reports/{report_id}/triage`
- `GET /api/admin/operations`
- `GET /api/admin/audit-events`
- `GET /api/admin/platform-admins`
- `POST /api/admin/platform-admins`
- `POST /api/admin/platform-admins/{user_id}/revoke`
- `GET /api/auth/refresh`
- `POST /api/auth/login`
- `POST /api/auth/register`
- `POST /api/auth/guest`
- `POST /api/auth/email-verification/confirm`
- `POST /api/auth/password-reset/request`
- `POST /api/auth/password-reset/confirm`
- `POST /api/account/email-verification`
- `PATCH /api/account/password`
- `PATCH /api/account/profile`
- `DELETE /api/account`
- `GET /api/users`
- `GET /api/users/{user_id}`
- `GET /api/collectibles/catalog`
- `GET /api/collectibles`
- `POST /api/collectibles`
- `POST /api/collectibles/award`
- `POST /api/collectibles/{collectible_id}/transfer`
- `GET /api/organizations/{organization_id}/collectibles`
- `GET /api/teams/{team_id}/collectibles`
- `GET /api/agent-credentials`
- `POST /api/agent-credentials`
- `POST /api/agent-credentials/{credential_id}/revoke`
- `POST /api/saved-queue-views`
- `GET /api/saved-queue-views`
- `GET /api/tasks`
- `POST /api/tasks`
- `GET /api/tasks/{task_id}`
- `POST /api/tasks/{task_id}/open`
- `POST /api/tasks/{task_id}/cancel`
- `POST /api/tasks/{task_id}/unpublish`
- `POST /api/tasks/{task_id}/funding`
- `POST /api/tasks/{task_id}/refund`
- `POST /api/tasks/{task_id}/collectible-refund`
- `POST /api/tasks/{task_id}/collectible-reward`
- `GET /api/teams/{team_id}/work`
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
- `GET /api/tasks/{task_id}/comments`
- `POST /api/tasks/{task_id}/comments`
- `GET /api/submissions/{submission_id}/comments`
- `POST /api/submissions/{submission_id}/comments`
- `POST /api/tasks/{task_id}/reservations`
- `GET /api/tasks/{task_id}/reservations`
- `POST /api/tasks/{task_id}/reservations/{reservation_id}/approve`
- `POST /api/tasks/{task_id}/reservations/{reservation_id}/decline`
- `POST /api/tasks/{task_id}/reservations/{reservation_id}/cancel`
- `POST /api/tasks/{task_id}/submissions`
- `GET /api/tasks/{task_id}/submissions`
- `GET /api/users/{user_id}/submissions`
- `POST /api/tasks/{task_id}/submissions/{submission_id}/accept`
- `GET /api/credits/balance`
- `GET /api/credits/ledger`
- `GET /api/organizations/{organization_id}/credits/balance`
- `GET /api/organizations/{organization_id}/credits/ledger`

The handlers reject missing storage, missing clocks, missing actor identity,
missing ID sources, unsupported routes, unsupported methods, invalid request
bodies, invalid privacy request kinds, invalid saved-queue scopes, invalid
account-token flows, invalid admin states, and invalid triage states. Guest auth
fails loudly because anonymous worker identity remains unsupported. Notification
handlers also reject invalid pagination and actor/recipient mismatches.
Organization handlers reject missing user resolvers for email-based member
provisioning rather than fabricating anonymous or placeholder identities.
Interaction handlers reject missing storage, clocks, actors, ID sources, missing
task/submission records, invalid pagination, invalid attachments, invalid
reservation transitions, invalid submission-task links, and invalid ledger
owners. They do not provide substitute stores for unimplemented routes.

## Host Adapter Shape

`internal/wasmdemo` now has an explicit host-runtime shape for the WASM target.
A host must provide storage, clock, actor/session, and interaction-ID adapters
before request execution can run. `ValidateHostRuntime` rejects a missing host
runtime or missing adapter with a clear error. This keeps the browser demo and
future production WASM host from silently substituting process state.

Two concrete host implementations exist today:

- `site/demo/wasm-host.js` is the browser host. It backs storage with
  `window.localStorage`, uses `Date.now()` for the clock, and reads the
  signed-in actor from browser-local session state.
- `tools/wasm_runtime_loader.ts` (`createHost`) is the non-browser host. It runs
  under Deno, with no browser APIs available, and is exercised by
  `deno task check:scenario-parity:wasm` and `deno task measure:wasm`.

## Non-Browser Host Adapter Reference

`createHost` in `tools/wasm_runtime_loader.ts` is the reference non-browser
implementation of the `HostFunctions` contract the compiled `js/wasm` binary
requires (`storageHas`, `storageGet`, `storagePut`, `now`, `actorID`, `nextID`,
`userIDForEmail`; see `validateJSHost` in `cmd/sharecrop-wasm/main_js_wasm.go`).
It proves the WASM backend runs outside a browser: it is loaded through Go's
`wasm_exec.js` the same way, with no DOM, `window`, or `localStorage`.

It is a test/measurement host, not a production one. Before it could back a real
non-browser deployment (for example a CLI-embedded or server-embedded WASM
runtime), it would need to change in these ways:

- **Storage.** `createHost` backs storage with an in-memory `Map` that is
  discarded when the process exits. A production non-browser host needs
  persistent storage (a file or a real database) behind the same
  `storageHas`/`storageGet`/`storagePut` contract.
- **Clock.** `createHost` returns a fixed timestamp so scenario runs and
  measurements stay deterministic. A production host needs a real system clock.
- **Actor resolution.** `createHost` exposes a `setActor` test hook that lets
  the caller impersonate any seeded user with no credential check. A production
  host must resolve the actor from a verified session or token, not
  caller-supplied state.
- **IDs and secrets.** `createHost.nextID` returns sequential,
  per-process-predictable IDs (`task-1`, `task-2`, ...), and
  `NextAgentCredentialSecret` in `cmd/sharecrop-wasm/main_js_wasm.go` derives
  agent-credential secrets from that same sequential counter
  (`"scrop_agent_" + nextID("agent_secret")`). That is acceptable for
  deterministic tests and measurement, but sequential secrets are guessable and
  must not be reused for a production host. The real (non-WASM) backend
  generates agent-credential secrets with `crypto/rand`
  (`internal/agent/values.go`); a production non-browser WASM host needs the
  same cryptographically random source behind `nextID`/a dedicated secret
  adapter, not the sequential counter this reference host uses.

No `Random()` or `Network()` adapter exists in the `HostRuntime` interface
(`internal/wasmdemo/host_adapters.go`) yet, even though earlier docs on this
page describe randomness and networking as adapters a production host would
eventually need. No currently implemented route requires unpredictable secrets
or an outbound network call, so no such adapter has been added. If a future
route needs one, it should fail loudly until an explicit adapter and host
implementation exist, the same way storage/clock/actor/ID adapters do today.

## Compile Check

The current Go codebase compiles to `js/wasm` for representative packages and
the main command:

- `GOOS=js GOARCH=wasm go test -c -o /private/tmp/sharecrop-core-schema-submission.test.wasm ./internal/submission`
- `GOOS=js GOARCH=wasm go test -c -o /private/tmp/sharecrop-http.test.wasm ./internal/http`
- `GOOS=js GOARCH=wasm go build -o /private/tmp/sharecrop-cmd.wasm ./cmd/sharecrop`
- `GOOS=js GOARCH=wasm go build -o /private/tmp/sharecrop-wasm-backend.wasm ./cmd/sharecrop-wasm`

Observed artifact sizes on the local build were about 6.1 MB for the submission
test package, 12 MB for the HTTP test package, and 24 MB for the command build.
Plain `go test` produced WASM test binaries but could not execute them natively;
running those tests requires a JS/WASM test runner.

`deno task check:scenario-parity:wasm -- --wasm site/demo/sharecrop-wasm-backend.wasm`
loads the compiled Go WASM binary through Go's `wasm_exec.js`, verifies the
`sharecropWasmBackendStatus`, `sharecropConfigureHost`, and
`sharecropHandleRequest` exports, verifies that requests fail before host
configuration, configures explicit host storage/clock/actor/ID adapters, and
runs the shared scenario parity suite through the exported request handler. The
runner does not call `site/demo/backend.js` and does not emulate missing backend
behavior.

`deno task wasm:demo:build` builds `cmd/sharecrop-wasm` into
`site/demo/sharecrop-wasm-backend.wasm` and copies Go's `wasm_exec.js` into the
demo directory. The Pages workflow runs that task before uploading the static
site. The generated `.wasm` and `wasm_exec.js` files are not committed.

The deployed demo defaults to the compiled Go/WASM backend path. The entrypoint
loads `wasm-host.js`, requires the generated WASM artifacts, configures explicit
browser host functions, seeds deterministic demo data, and intercepts `/api/*`
XHR requests through `sharecropHandleRequest`. Missing artifacts, unknown
backend modes, missing host functions, missing storage keys, and invalid host
values fail loudly.

The compile check means basic Go/WASM compatibility is not the blocker. The
remaining production work is continuing to add explicit WASM storage/handler
slices when new user-visible API surfaces are added, and hardening the
non-browser host past the reference/test shape described above.

## Runtime Measurements

`deno task measure:wasm -- --wasm <compiled.wasm> [--requests-per-route <n>]`
loads a compiled `cmd/sharecrop-wasm` artifact through the non-browser reference
host, configures it, and reports artifact size, startup time, host process
memory, and per-route request latency. It does not measure a browser runtime;
browser memory/latency depend on the host page and have not been measured
separately.

A local run against `site/demo/sharecrop-wasm-backend.wasm` (built by
`deno task wasm:demo:build`, a `go build` artifact with no debug/test symbols,
on the machine this doc was last updated on) reported:

- Artifact size: 4.30 MiB (4,510,058 bytes).
- Startup: about 29-50ms from `WebAssembly.instantiate` through the first
  `sharecropWasmBackendStatus` call reporting `unconfigured`, plus under 1ms for
  `sharecropConfigureHost`. Startup time varied across repeated runs within that
  range on the same machine.
- Host process memory (`Deno.memoryUsage()`, the Deno process hosting the WASM
  runtime, not WASM linear memory specifically): resident set size grew from
  about 53 MiB before loading the artifact to about 95 MiB after host
  configuration, and to about 155-165 MiB after 1,000-2,500 requests (5 routes x
  200-500 requests per route across runs).
- Request latency for `GET /api/users`, `GET /api/organizations`,
  `GET /api/tasks`, `GET /api/tasks/{task_id}`, and `GET /api/credits/balance`
  against the in-memory reference host: mean latency under 0.15ms per route, p95
  under 0.4ms per route, with occasional outliers up to about 10ms attributable
  to the Deno/V8 process rather than the WASM binary itself.

These numbers describe the non-browser reference host under Deno on a
development machine, not a deployed browser or a persistent-storage production
host; they establish a measurement method and a baseline, not a production SLA.

## Required Shape

- The backend artifact is a `.wasm` binary compiled from Go code with
  `GOOS=js GOARCH=wasm`.
- A `js/wasm` request adapter receives method, path, headers, and body from
  `fetch` interception.
- Domain/application services are constructed with explicit host adapters, not
  fake fallback stores.
- Browser storage adapters are explicit packages, likely backed by IndexedDB for
  persisted data and in-memory state only where the domain already treats state
  as process-local.
- Production WASM hosts must provide explicit storage, clock, identity/session,
  request, randomness, and networking adapters instead of relying on process
  globals or hidden substitute behavior.
- WASM tests run the shared scenario parity suite against the request adapter.
- Missing browser storage features fail at build or request time with clear
  errors.

## Adoption Gates

The static demo default can use the compiled Go/WASM path because these gates
are covered by the current branch:

1. It passes the shared scenario parity suite for task creation, selectors,
   comments, reservations, submission review, notifications, collectibles, and
   account-token flows.
2. It has no fallback behavior for unimplemented stores or handlers.
3. It has a deterministic reset path for demo data.
4. GitHub Pages deployment builds the WASM artifacts and the deployed routing
   check verifies the demo entrypoint and generated WASM runtime assets.

Remaining production WASM gates:

1. Bundle size, startup time, host-process memory, and request latency are now
   measured by `deno task measure:wasm` against the compiled artifact; see
   Runtime Measurements above. Browser-specific memory/latency and a
   persistent-storage production host are not yet measured.
2. A non-browser host adapter set is documented and tested (`createHost` in
   `tools/wasm_runtime_loader.ts`, exercised by
   `deno task check:scenario-parity:wasm` and `deno task measure:wasm`), but it
   is a test/measurement host, not a production one. See Non-Browser Host
   Adapter Reference above for what a production non-browser host still needs:
   persistent storage, a real clock, verified-session actor resolution, and
   cryptographically random IDs/secrets.
3. Keep extending parity as new API surfaces are added.

## Next Spike Step

The next WASM step is closing the remaining non-browser production gate: a
persistent-storage, real-clock, verified-actor, cryptographically-random host
implementation for a genuine non-browser deployment target, built on the same
`HostRuntime`/`HostFunctions` contracts the reference host already proves out.
Keep expanding explicit storage/handler slices as new API surfaces are
introduced; a missing slice should fail loudly until it has an explicit host
adapter and handler.
