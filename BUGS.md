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
- Full Playwright mobile/screens coverage was not run locally for this branch.
  Focused demo Playwright coverage passed, including admin privacy request
  resolution and task moderation reporting. Focused admin desktop/mobile
  screenshots were inspected.
- Database-backed integration and HTTP E2E checks were not completed locally for
  this branch because `DATABASE_URL` is not set in the current environment.
  Postgres-backed coverage should be run before merge or by CI.

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
  collectibles, tasks, comments, submissions, notifications, and a multi-actor
  reservation/submission-review/payout flow, but they do not prove every handler
  has identical domain semantics.
- Go code compiles to `js/wasm` for representative packages and the main
  command. A narrow `internal/wasmdemo` request-adapter spike exists for privacy
  and moderation route classification, and an explicit moderation-triage browser
  storage boundary exists. A WASM demo backend still lacks a request handler
  wired to domain-service execution, deterministic reset, startup measurements,
  and a JS/WASM scenario test runner.

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
