# Bugs

Confirmed defects:

- None known.

Test gaps:

- GitHub Pages deployment cannot be observed from pull request CI because the Pages workflow publishes after pushes to `main` or manual dispatch.
- Anonymous workers were removed. The anonymous worker identity and payout model is deferred; submissions are registered-users-only.
- Some recipient fields still require raw IDs where the browser has no loaded directory data or no typeahead/paginated selector. Task creation now uses user/team selectors for the covered visibility fields.
- Account verification and password reset support `SHARECROP_ACCOUNT_TOKEN_DELIVERY=log`, which logs tokens and returns a sent status. SMTP/provider delivery is not implemented.
- Account lifecycle deletion semantics are deactivation plus credential/session/token revocation and email anonymization. Hard row deletion is intentionally not used because tasks, submissions, comments, ledger entries, and ownership rows reference users.
- The asset economy is intentionally internal-only: rewards are Sharecrop credits and admin-minted Sharecrop collectibles. User-issued tokens, organization-issued tokens, per-project tokens, crypto rewards, and external wallets are out of scope.
- Request/command contracts and HTTP contract fixture tests still need to expand as the API grows. The backendless demo now has route-surface and representative response-shape tests, but those are not a full generated fixture suite.
- User directory browser selectors currently load the first page rather than a typeahead query result; large installations need paginated/typeahead UI.

Known risks:

- Cancelling a task that holds escrow is now rejected: the store's `ChangeTaskState` to `cancelled` refuses with 409 "refund the task's held escrow before cancelling" when held credits or collectibles exist, so the state transition can never orphan escrow (previously Cancel left held escrow stranded against a cancelled task). The browser routes funded tasks to Refund; a rare funded-draft Cancel attempt now surfaces that 409 with the Refund action alongside.

- `site/demo/backend.js` is a demo-only in-browser fake backend; it re-implements API behavior in JS and can drift from the Go backend's actual semantics. Deno tests now compare its route surface with the real HTTP router and validate representative client response shapes, but they do not prove every handler has identical domain semantics.

- A collectible review tip and the credit settle are separate per-store transactions sequenced in one accept request (credit settle first, then `GiftCollectible`). The settle is idempotent and the gift is replay-safe (already-owned-by-worker is a no-op), so a retried accept recovers; the only residual is a small window where the credit accept commits but the gift fails (e.g. the collectible changed owner concurrently), returning an error after a committed accept. Folding the tip into the ledger transaction would remove the window.
- `transferable_within_organization` collectibles cannot be tipped yet: collectibles carry no organization, so the within-org bound is unenforceable and `AllowsTip` denies it rather than allow a cross-org gift. Re-enable once collectibles carry an org and the gift checks shared membership.
- The in-memory rate limiter evicts full buckets only on request arrival (at most once per refill window), so a burst of many distinct keys can transiently grow the map until the next sweep. Key sources are bounded (client IPs / verified agent subjects) and entries are tiny, so memory pressure is low; a background sweep would remove the transient growth.

- MCP HTTP sessions, SSE replay buffers, and the rate-limiter buckets are in-memory at runtime. Idle entries are evicted (and MCP sessions are capped per subject and globally), but this state is not shared across server restarts or multiple app processes, so the rate limits are per-process rather than a global quota. The database now has operations-state tables for future runtime adapters.
- Standalone (user-owned) teams can be created, listed, and selected for task visibility. They are not yet selectable as organization-team assignees because task assignment currently models user and organization-team scopes only.
- Foreign keys use the PostgreSQL default `NO ACTION`, which blocks deletion of referenced rows. The application has no deletion paths, so orphan rows cannot occur. Explicit `ON DELETE` behavior is not defined and should be designed alongside any future deletion feature.
