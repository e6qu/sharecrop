# Bugs

Confirmed defects: None.

Test gaps:

- Repository CI validates and policy-checks the Amazon API Gateway, VPC Link,
  AWS Cloud Map, and private Amazon ECS Terraform graph but does not create AWS
  resources. The environment repository owns the real plan/apply and live
  custom-domain, health-routing, and Shauth browser checks for each immutable
  module/image pin.
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
- Wasm is the production backend, not only a demo mechanism: `cmd/sharecrop serve`
  hosts the real `internal/http` mux + domain services as a `wasip1` guest under a
  wazero pool, bridging store calls to Postgres. The earlier risk — no persistent
  production non-browser host (the old reference host was in-memory, fixed-clock,
  sequential-id) — is resolved: production is the WASI guest pool on ECS Fargate,
  state in Postgres, with real randomness/clock and a baked AOT cache (no startup
  compile). See [docs/deployment.md](./docs/deployment.md). Ongoing work: continued
  scenario-parity expansion as API surfaces change. Known gap (still open): there
  is no route-drift-detection test (real REST routes vs. a mock's route table) for
  the WASM dispatch path, noted above.

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
