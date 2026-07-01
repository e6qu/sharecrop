# Bugs

Confirmed defects:

- None known.

Test gaps:

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

Known risks:

- Cancelling a task that holds escrow is now rejected: the store's
  `ChangeTaskState` to `cancelled` refuses with 409 "refund the task's held
  escrow before cancelling" when held credits or collectibles exist, so the
  state transition can never orphan escrow (previously Cancel left held escrow
  stranded against a cancelled task). The browser routes funded tasks to Refund;
  a rare funded-draft Cancel attempt now surfaces that 409 with the Refund
  action alongside.

- `site/demo/backend.js` is a demo-only in-browser fake backend; it
  re-implements API behavior in JS and can drift from the Go backend's actual
  semantics. Deno tests compare its route surface with the real HTTP router,
  validate representative response shapes, and run shared scenario parity flows
  for selectors, admin operations, privacy request/audit/resolution shape,
  moderation report/admin-list/audit shape, sensitive-field redaction state,
  collectibles, tasks, small attachments, comments, submissions, notifications,
  and a multi-actor reservation/submission-review/payout flow, but they do not
  prove every handler has identical domain semantics.
- Go/WASM is a first-class backend execution target, not only a demo mechanism.
  Go code compiles to `js/wasm` for representative packages, the main command,
  and `cmd/sharecrop-wasm`. `internal/wasmdemo` classifies privacy, moderation,
  saved-queue-view, task, notification, organization, organization-member, team,
  comment, reservation, submission, and ledger routes. It has explicit
  browser-storage and request-handler boundaries for those slices. A Deno WASM
  runner loads a compiled Go `.wasm` artifact, verifies required exports,
  configures explicit host adapters, and runs the current task/comment/
  reservation/submission/ledger scenario through the exported request handler.
  The WASM backend target still lacks browser IndexedDB host adapters, remaining
  behavior slices for collectibles, account-token flows, admin operations,
  privacy resolution/redaction jobs, moderation projection writes, deterministic
  demo seeding/reset, the full shared scenario parity suite, and startup
  measurements. JavaScript reimplementations, generated fake backends, and
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
