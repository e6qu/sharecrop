# Bugs

Confirmed defects:

- None known.

Test gaps:

- `docs/openapi.json` (`make openapi`/`make check-openapi`, `internal/openapi`)
  has an accurate method/path/operationId/bearer-auth inventory generated from
  `internal/http/server.go`'s route table and local call graph. Request/
  response body schemas are typed (derived from the actual Go DTO struct a
  handler decodes/writes) for 102/109 responses and 41/63 request bodies; the
  rest are genuinely out of scope (MCP JSON-RPC passthrough, bodyless routes,
  static/index/healthz), not generator gaps.
- GitHub Pages deployment cannot be observed from pull request CI because the
  Pages workflow publishes after pushes to `main` or manual dispatch. The Pages
  workflow runs the deployed routing check after deployment; manual checks can
  still run with `deno task check:pages-routing -- --origin <pages-origin>`.
- Anonymous workers are not supported. Submissions are registered-users-only.
- Some fields may still require raw IDs where the browser has no loaded
  directory data or no selector-backed flow. The latest audit found IDs still
  visible in links, protocol surfaces, metadata, audit rows, and API/MCP
  examples, but no confirmed high-traffic raw-ID input remains listed.
- Account verification and password reset default to
  `SHARECROP_ACCOUNT_TOKEN_DELIVERY=log` (fail closed), which logs the token
  and returns a sent status; production `serve` never returns the token in the
  HTTP response. `api` mode (token returned in the response body) is opt-in and
  used by the browser demo and tests only. Password-reset returns the same
  neutral response for unknown and known emails. Provider email delivery is
  intentionally deferred; admins are expected to set up accounts and
  organizations directly for now.
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
  `29180`, and the WASM demo `29181`.
- `tests/playwright/demo.spec.ts` and `tests/playwright/mobile.spec.ts` passed
  against the Go/WASM default demo on non-default port `29181`.

Known risks:

- Credit funding uses a two-section wallet, not an escrow state machine.
  Funding moves credits from the account's spendable section (still
  `sum(ledger_entries)`) into its allocated section (a stateless
  `task_funds` row); allocated credits are not spendable, so they cannot be
  double-spent. `ChangeTaskState` runs its invariant checks and the state
  UPDATE in one transaction with an expected-prior-state predicate, so
  concurrent cancel/fund/refund cannot interleave to orphan allocated
  credits or reopen a cancelled task. Cancelling a funded un-awarded task
  returns its allocated credits to the funder's spendable balance
  automatically.

- Refunds are auto-granted by default: the task owner (requester) or the
  user holding the active reservation can refund while the task is not yet
  awarded (no accepted submission). The allocated credits and any held
  collectible return to the funder, and the task is cancelled. This is a
  deliberate, permissive policy — a requester can refund even while a
  submission is pending review (the task is cancelled and the worker is not
  paid). Self-deactivation is still rejected while the user owns tasks
  holding allocated credits or held collectibles, so funds that only their
  owner can refund are not stranded.

- The task detail, create, and state-change responses report the live
  allocated reward a task holds (`allocated_credits` and the individual
  `allocated_collectible_ids`), distinct from the declared reward. The
  browser gates the Refund button on it (no Refund on an unfunded declared
  reward) and shows a per-task funding line; the Overview/org pages show the
  account's allocated total.

- Refunding or cancelling a task releases the worker's reservation (to
  `cancelled_by_requester`) through both storage adapters, so a reservation no longer
  dangles in an active/submitted state on a cancelled task. The refund/cancel
  paths (credit refund, collectible refund, and the cancel state transition)
  all go through the shared release helper.

- `site/demo/backend.js` (the legacy JS mock backend) and its Deno tests have
  been removed; `deno task check:scenario-parity:wasm` is now CI-enforced
  replacement coverage against the real, deployed WASM backend. No equivalent
  exists for the removed route-drift-detection test (real REST routes vs. a
  mock's route table) against the WASM dispatch path — a known, accepted gap.
- Go/WASM is a first-class backend execution target, not only a demo mechanism.
  `cmd/sharecrop-wasm` compiles the real `internal/http` mux and the real
  domain services to `js/wasm`; `internal/wasmdemo` provides
  browser-storage-backed `Store` implementations for all 9 domain packages
  (`browserstore_*.go`) plus the seed routine — it no longer classifies or
  handles requests itself. A Deno WASM runner
  (`tools/wasm_runtime_loader.ts`) loads a compiled Go `.wasm` artifact,
  verifies required exports, configures explicit host adapters (storage,
  clock, id source), and runs the shared scenario parity suite through the
  exported request handler with real bearer tokens; the mux resolves
  identity from the bearer token, not the host. `deno task measure:wasm`
  reports artifact size, startup time, host-process memory, and request
  latency against a compiled artifact; see
  [docs/wasm_demo_backend_spike.md](./docs/wasm_demo_backend_spike.md) for a
  baseline. Remaining WASM risk is a genuine production non-browser host:
  the reference host is in-memory (unpersisted across restarts), uses a
  fixed clock, and its `nextID` counter yields sequential
  submission/comment/reservation ids (ledger entry ids are generated in Go
  as real UUIDs); none of that is safe to reuse for a production non-browser
  deployment, and no such deployment target exists yet to build a production
  host against. Continued parity expansion as API surfaces change also
  remains ongoing work. JavaScript reimplementations, generated fake
  backends, and fallback stores are not valid substitutes for the compiled
  Go WASM binary.

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
