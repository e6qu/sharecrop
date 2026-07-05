# Demo Semantic Parity

The static demo now defaults to the compiled Go/WASM backend path. The browser
loads the real Elm client, `wasm-host.js`, Go's `wasm_exec.js`, and
`sharecrop-wasm-backend.wasm`; `/api/*` requests are handled by the exported Go
WASM request adapter and explicit browser host adapters.

`site/demo/backend.js`, the hand-maintained JS mock backend the demo used
before the WASM path, is removed. It was never loaded by `site/demo/index.html`
after the WASM backend became the default, and its two Deno tests (a
route-drift check and a scenario-parity run) were retired once
`deno task check:scenario-parity:wasm` (below) was proven as CI-enforced
replacement coverage against the real, deployed WASM backend. There is no
current equivalent of `backend.js`'s route-drift check (real REST routes vs.
a mock's route table) for the WASM dispatch path — that specific coverage
is gone, a known and accepted trade-off, not an oversight.

## Current Checks

- `deno task check:scenario-parity:wasm -- --wasm site/demo/sharecrop-wasm-backend.wasm`
  loads the compiled Go/WASM artifact, configures explicit host adapters, and
  runs the shared scenario parity suite through `sharecropHandleRequest`.
- `deno task check:scenario-parity -- --origin <api-origin> --token <access-token> --refresh-token <refresh-token>`
  runs the same shared scenario against a real API. The supplied session must
  belong to a platform admin that can refresh auth, read operations status,
  issue account tokens, mint/transfer collectibles, create organizations, create
  teams, create tasks, create submissions, create task/submission comments, and
  register additional scenario actors.
- `deno task check:scenario-parity:local-real -- --origin <api-origin>` runs the
  shared scenario against a local real API by registering a scenario admin and
  granting platform-admin state through the configured `DATABASE_URL`. It fails
  if `/healthz`, registration, `psql`, the admin grant, or any scenario request
  fails.
- `deno task check:pages-routing -- --origin <pages-origin>` checks deployed
  GitHub Pages root, docs, demo entry paths, and demo assets after deployment.

## Parity Strategy

1. Keep expanding shared scenario tests that run the same scripted user flows
   against the Go HTTP API and the Go/WASM request adapter.
   - Keeps the demo static-hostable.
   - Catches behavior drift for task creation, reservations, submissions,
     reviews, notifications, collectibles, org/team flows, account tokens, agent
     credentials, and admin views.

2. Continue moving host-sensitive behavior behind explicit WASM adapters.
   - This is the active deployed-demo path and a first-class production
     execution target.
   - Higher semantic parity for code paths that run through Go handlers.
   - Requires explicit host adapters for auth, tasks, submissions, ledger,
     assets, orgs, audit, notifications, MCP sessions, rate limits, clocks,
     identity/session, randomness, and networking.
   - Does not reuse PostgreSQL, pgx, net/http server wiring, process signals, or
     production migrations directly in the browser.
   - Build size, startup cost, and request latency are measured against the
     compiled artifact by `deno task measure:wasm` (see
     [wasm_demo_backend_spike.md](./wasm_demo_backend_spike.md)). A non-browser
     reference host exists and is tested, but it is in-memory and test-only; JS
     interop, IndexedDB persistence, and a persistent-storage non-browser host
     still need production hardening.

3. Host a real demo backend with a disposable database.
   - Highest semantic parity.
   - No longer backendless and not suitable for static-only hosting.
   - Requires provisioning, reset behavior, abuse controls, and operational
     ownership.

## Recommendation

Keep expanding shared scenario parity as user-visible API surfaces change. The
suite covers selector pagination/query behavior, ledger and notification
pagination behavior, team/organization queue search/type/sort behavior,
persisted saved queue views, org/team/task/comment creation, organization member
provisioning/listing/role/deactivation, small task/submission attachments, admin
operations, platform-admin grant/revoke, account-token issue shape, privacy
request resolution, privacy retention audit shape, moderation triage audit
shape, sensitive-field redaction state, collectible catalog/mint/transfer,
agent-credential creation/revocation, submission creation/comments with
sensitive-field response metadata, notification read shape, and a multi-actor
reservation/submission-review/payout flow.

Recent parity coverage also checks admin audit pagination by requesting adjacent
one-row pages and asserting they do not collapse onto the same event when both
pages are populated. The browser admin page now exposes pagination controls for
audit events, platform-admin records, privacy requests, and moderation reports,
so the demo and real UI can exercise more than first-page-only admin lists.

Use [wasm_demo_backend_spike.md](./wasm_demo_backend_spike.md) for the current
WASM backend-target finding. The Go/WASM backend target must continue through
explicit host adapters without fallbacks or dead paths.
