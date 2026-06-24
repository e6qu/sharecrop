# Bugs

Confirmed defects:

- None known.

Test gaps:

- GitHub Pages deployment cannot be observed from pull request CI because the Pages workflow publishes after pushes to `main` or manual dispatch.
- Anonymous workers were removed. The anonymous worker identity and payout model is deferred; submissions are registered-users-only.
- Browser task creation exposes public, private, and specific-user visibility, and can own a task by an organization. Organization and organization-team visibility, team assignee selection, and funding an organization-owned task from the organization credit account are not yet exposed in the browser.
- The browser provisions organization members but cannot list them; there is no `GET` members endpoint.
- Review tips are credit-only. Collectible or inventory-based tips remain deferred.
- The asset economy is platform-only: user-issued tokens, organization-issued tokens, crypto rewards, and external wallets are not implemented. Current implemented rewards are Sharecrop credits and platform collectibles, including multiple collectibles per task.
- Request/command contracts and HTTP contract fixture tests still need to expand as the API grows.

Known risks:

- No rate limiting on expensive or unauthenticated endpoints (MCP tool calls, the unauthenticated submission-receipt lookup, login/refresh). A security review rated this low (availability/defense-in-depth, not confidentiality: the receipt token is a 256-bit random bearer capability and the response is redacted). A per-subject/per-IP token-bucket limiter returning HTTP 429 is the planned follow-up.
- Tip ledger entries (`task_tip`) do not carry an `idempotency_key`, unlike payout/refund entries. A security review verified this is not exploitable: `lockTaskForReview` takes a `FOR UPDATE` row lock that serializes concurrent accept/reject and tips are only paid on the first transition out of `submitted`, so no double-tip or overdraft occurs. Adding a derived key to both tip inserts is a deferred robustness/auditing improvement.
- MCP HTTP sessions and SSE replay buffers are in-memory. Idle sessions are evicted after a TTL and are now capped per agent subject and globally, but state is still not shared across server restarts or multiple app processes.
- Standalone (user-owned) teams can be created and listed but are not yet selectable as a task assignee or visibility scope. See [DO_NEXT.md](./DO_NEXT.md).
- Foreign keys use the PostgreSQL default `NO ACTION`, which blocks deletion of referenced rows. The application has no deletion paths, so orphan rows cannot occur. Explicit `ON DELETE` behavior is not defined and should be designed alongside any future deletion feature.
