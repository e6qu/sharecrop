# Bugs

Confirmed defects:

- None known.

Test gaps:

- GitHub Pages deployment cannot be observed from pull request CI because the Pages workflow publishes after pushes to `main` or manual dispatch. The Pages workflow runs the deployed routing check after deployment; manual checks can still run with `deno task check:pages-routing -- --origin <pages-origin>`.
- Anonymous workers are not supported. Submissions are registered-users-only.
- Some fields may still require raw IDs where the browser has no loaded directory data or no selector-backed flow. No confirmed high-traffic raw-ID flow is currently listed.
- Account verification and password reset support `SHARECROP_ACCOUNT_TOKEN_DELIVERY=log`, which logs tokens and returns a sent status. Provider email delivery is intentionally deferred; admins are expected to set up accounts and organizations directly for now.
- Account lifecycle deletion semantics are deactivation plus credential/session/token revocation and email anonymization. Hard row deletion is intentionally not used because tasks, submissions, comments, ledger entries, and ownership rows reference users.
- The asset economy is intentionally internal-only: rewards are Sharecrop credits and admin-minted Sharecrop collectibles. User-issued tokens, organization-issued tokens, per-project tokens, crypto rewards, and external wallets are out of scope.
- Request/command contracts and HTTP contract fixture tests should keep expanding as the API grows.
- Local database-backed integration verification through `make db-checks` requires `DATABASE_URL` and `SHARECROP_MIGRATIONS_DIR`; PR 79 CI passed this job.
- A focused local Playwright run for submission-discussion auto-open behavior did not complete because the sandbox blocked binding the local server port and escalation was unavailable after the environment usage limit was reached. The Playwright assertions are present and should run in PR CI.

Known risks:

- Cancelling a task that holds escrow is now rejected: the store's `ChangeTaskState` to `cancelled` refuses with 409 "refund the task's held escrow before cancelling" when held credits or collectibles exist, so the state transition can never orphan escrow (previously Cancel left held escrow stranded against a cancelled task). The browser routes funded tasks to Refund; a rare funded-draft Cancel attempt now surfaces that 409 with the Refund action alongside.

- `site/demo/backend.js` is a demo-only in-browser fake backend; it re-implements API behavior in JS and can drift from the Go backend's actual semantics. Deno tests compare its route surface with the real HTTP router, validate representative response shapes, and run shared scenario parity flows for selectors, admin operations, collectibles, tasks, comments, submissions, notifications, and a multi-actor reservation/submission-review/payout flow, but they do not prove every handler has identical domain semantics.

- The default test/demo HTTP constructor still uses in-memory rate-limit buckets, audit events, notifications, and MCP sessions. Production `serve` wires Postgres-backed rate-limit buckets, audit events, notifications, persisted MCP HTTP session identity, and persisted MCP replay events.
- MCP HTTP session identity, TTL admission, close state, active counts, and replay events are persisted in Postgres for production `serve`, but live SSE subscriber channels remain process-local.
- Standalone (user-owned) teams can be created, listed, and selected for task visibility. They are not yet selectable as organization-team assignees because task assignment currently models user and organization-team scopes only.
- Foreign keys use the PostgreSQL default `NO ACTION`, which blocks deletion of referenced rows. The application has no core hard-delete paths, so orphan rows cannot occur. [docs/deletion_semantics.md](./docs/deletion_semantics.md) defines the design gate for future deletion features.
