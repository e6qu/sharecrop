# Bugs

Confirmed defects:

- None known.

Test gaps:

- `docs/openapi.json` (`make openapi`/`make check-openapi`, `internal/openapi`)
  has an accurate method/path/operationId/bearer-auth inventory generated from
  `internal/http/server.go`'s route table and local call graph, but
  request/response bodies are generic JSON object placeholders, not typed
  per-route schemas. `internal/contracts` has no method/path/route metadata to
  drive that today; adding typed schemas needs an explicit
  route-to-contract-type mapping.
- GitHub Pages deployment cannot be observed from pull request CI because the
  Pages workflow publishes after pushes to `main` or manual dispatch. The Pages
  workflow runs the deployed routing check after deployment; manual checks can
  still run with `deno task check:pages-routing -- --origin <pages-origin>`.
- Anonymous workers are not supported. Submissions are registered-users-only.
- Some fields may still require raw IDs where the browser has no loaded
  directory data or no selector-backed flow. The latest audit found IDs still
  visible in links, protocol surfaces, metadata, audit rows, and API/MCP
  examples, but no confirmed high-traffic raw-ID input remains listed.
- Account verification and password reset support
  `SHARECROP_ACCOUNT_TOKEN_DELIVERY=log`, which logs tokens and returns a sent
  status. Provider email delivery is intentionally deferred; admins are expected
  to set up accounts and organizations directly for now.
- Account lifecycle semantics are deactivation plus credential/session/token
  revocation and email anonymization. Row removal is not part of the project
  direction because tasks, submissions, comments, ledger entries, and ownership
  rows reference users.
- Submission responses expose indexed sensitive-field metadata for authorized
  submission viewers. Privacy requests are persisted and can be resolved by
  platform admins through the API and browser admin page. Resolution stores
  export JSON or marks delete-on-request sensitive-field metadata as redacted.
  Platform admins can run retention for delete-on-request sensitive-field
  metadata, and authorized submission-list/profile reads record sensitive-field
  access events.
- The asset economy is intentionally internal-only: rewards are Sharecrop
  credits and admin-minted Sharecrop collectibles. User-issued tokens,
  organization-issued tokens, per-project tokens, crypto rewards, and external
  wallets are out of scope.
- Request/command contracts and HTTP contract fixture tests should keep
  expanding as the API grows.
- Real API shared scenario parity can run with an explicit platform-admin
  access-token plus refresh-token session, or locally through
  `check:scenario-parity:local-real` when `DATABASE_URL`, `psql`, and a local
  API are available. A long-lived deployed real API with an admin session still
  requires operator-supplied credentials.
- DB-backed checks, local real API shared scenario parity, and DB-backed
  Playwright screens passed against an isolated local PostgreSQL 15 data
  directory under `.cache` on non-default local ports: Postgres `25432`, app
  `29180`, and backendless demo `29181`.
- `tests/playwright/demo.spec.ts` and `tests/playwright/mobile.spec.ts` passed
  against the Go/WASM default demo on non-default port `29181` after the WASM
  agent-credential slice was added.

Known risks:

- Cancelling a task that holds escrow is now rejected: the store's
  `ChangeTaskState` to `cancelled` refuses with 409 "refund the task's held
  escrow before cancelling" when held credits or collectibles exist, so the
  state transition can never orphan escrow (previously Cancel left held escrow
  stranded against a cancelled task). The browser routes funded tasks to Refund;
  a rare funded-draft Cancel attempt now surfaces that 409 with the Refund
  action alongside.

- `site/demo/backend.js` remains legacy parity/test material and still
  re-implements API behavior in JS. The deployed static demo defaults to the
  compiled Go/WASM backend path and does not load `backend.js` as a fallback.
  The old Deno checks still compare its route surface with the real HTTP router
  and run shared scenario parity flows, but those checks do not prove every
  legacy JS handler has identical domain semantics.
- Go/WASM is a first-class backend execution target, not only a demo mechanism.
  Go code compiles to `js/wasm` for representative packages, the main command,
  and `cmd/sharecrop-wasm`. `internal/wasmdemo` classifies and handles current
  shared-scenario slices for auth/account, users, admin operations,
  platform-admins, audit events, collectibles, agent credentials, privacy,
  moderation, saved-queue-view, task, notification, organization,
  organization-member, team, comment, reservation, submission, and ledger
  routes. A Deno WASM runner loads a compiled Go `.wasm` artifact, verifies
  required exports, configures explicit host adapters, and runs the shared
  scenario parity suite through the exported request handler.
  `deno task measure:wasm` reports artifact size, startup time, host-process
  memory, and request latency against a compiled artifact; see
  [docs/wasm_demo_backend_spike.md](./docs/wasm_demo_backend_spike.md) for a
  baseline. `tools/wasm_runtime_loader.ts` documents and implements the
  reference non-browser host. Remaining WASM risk is a genuine production
  non-browser host: the reference host is in-memory (unpersisted across
  restarts), uses a fixed clock, lets the caller set the actor with no
  credential check, and generates IDs/secrets from a sequential counter rather
  than `crypto/rand`; none of that is safe to reuse for a production non-browser
  deployment, and no such deployment target exists yet to build a production
  host against. Continued parity expansion as API surfaces change also remains
  ongoing work. JavaScript reimplementations, generated fake backends, and
  fallback stores are not valid substitutes for the compiled Go WASM binary.

- The default test/demo HTTP constructor still uses in-memory rate-limit
  buckets, audit events, notifications, and MCP sessions. Production `serve`
  wires Postgres-backed rate-limit buckets, audit events, notifications,
  persisted MCP HTTP session identity, and persisted MCP replay events.
- MCP HTTP session identity, TTL admission, close state, active counts, and
  replay events are persisted in Postgres for production `serve`. Persisted live
  SSE subscribers poll the replay table for cross-process fan-out groundwork.
- Foreign keys use the PostgreSQL default `NO ACTION`, which blocks deletion of
  referenced rows. Application lifecycle work uses state changes, redaction,
  tombstones, and audit records rather than core-row removal.
  [docs/deletion_semantics.md](./docs/deletion_semantics.md) defines the
  lifecycle and redaction rules.
