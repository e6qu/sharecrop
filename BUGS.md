# Bugs

Confirmed defects:

- None known.

Test gaps:

- GitHub Pages deployment cannot be observed from pull request CI because the Pages workflow publishes after pushes to `main` or manual dispatch.
- Anonymous workers are not supported. Submissions are registered-users-only.
- Some recipient fields still require raw IDs where the browser has no loaded directory data or no typeahead/paginated selector. Task creation now uses user/team selectors for the covered visibility fields.
- Account verification and password reset support `SHARECROP_ACCOUNT_TOKEN_DELIVERY=log`, which logs tokens and returns a sent status. Provider email delivery is intentionally deferred; admins are expected to set up accounts and organizations directly for now.
- Account lifecycle deletion semantics are deactivation plus credential/session/token revocation and email anonymization. Hard row deletion is intentionally not used because tasks, submissions, comments, ledger entries, and ownership rows reference users.
- The asset economy is intentionally internal-only: rewards are Sharecrop credits and admin-minted Sharecrop collectibles. User-issued tokens, organization-issued tokens, per-project tokens, crypto rewards, and external wallets are out of scope.
- Request/command contracts and HTTP contract fixture tests still need to expand as the API grows. The backendless demo has route-surface and representative response-shape tests, but those are not a full generated fixture suite.
- User directory browser selectors currently load the first page rather than a typeahead query result; large installations need paginated/typeahead UI.
- Local integration verification requires `DATABASE_URL` and `SHARECROP_MIGRATIONS_DIR`. The current branch adds integration tests for notifications, audit listing, persisted MCP sessions/replay events, and rate-limit buckets, but they were not executable in this environment because `DATABASE_URL` is not set.

Known risks:

- Cancelling a task that holds escrow is now rejected: the store's `ChangeTaskState` to `cancelled` refuses with 409 "refund the task's held escrow before cancelling" when held credits or collectibles exist, so the state transition can never orphan escrow (previously Cancel left held escrow stranded against a cancelled task). The browser routes funded tasks to Refund; a rare funded-draft Cancel attempt now surfaces that 409 with the Refund action alongside.

- `site/demo/backend.js` is a demo-only in-browser fake backend; it re-implements API behavior in JS and can drift from the Go backend's actual semantics. Deno tests now compare its route surface with the real HTTP router and validate representative client response shapes, but they do not prove every handler has identical domain semantics.

- A collectible review tip and the credit settle are separate per-store transactions sequenced in one accept request (credit settle first, then `GiftCollectible`). The settle is idempotent and the gift is replay-safe (already-owned-by-worker is a no-op), so a retried accept recovers; the only residual is a small window where the credit accept commits but the gift fails (e.g. the collectible changed owner concurrently), returning an error after a committed accept. Folding the tip into the ledger transaction would remove the window.
- `transferable_within_organization` collectibles cannot be tipped yet: collectibles carry no organization, so the within-org bound is unenforceable and `AllowsTip` denies it rather than allow a cross-org gift. Re-enable once collectibles carry an org and the gift checks shared membership.
- The default test/demo HTTP constructor still uses in-memory rate-limit buckets, audit events, notifications, and MCP sessions. Production `serve` wires Postgres-backed rate-limit buckets, audit events, notifications, persisted MCP HTTP session identity, and persisted MCP replay events.
- MCP HTTP session identity, TTL admission, close state, active counts, and replay events are persisted in Postgres for production `serve`, but live SSE subscriber channels remain process-local.
- Standalone (user-owned) teams can be created, listed, and selected for task visibility. They are not yet selectable as organization-team assignees because task assignment currently models user and organization-team scopes only.
- Foreign keys use the PostgreSQL default `NO ACTION`, which blocks deletion of referenced rows. The application has no deletion paths, so orphan rows cannot occur. Explicit `ON DELETE` behavior is not defined and should be designed alongside any future deletion feature.
