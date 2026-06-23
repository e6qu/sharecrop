# Bugs

Confirmed defects:

- None known.

Test gaps:

- GitHub Pages deployment cannot be observed from pull request CI because the Pages workflow publishes after pushes to `main` or manual dispatch.
- Anonymous workers were removed. The anonymous worker identity and payout model is deferred; submissions are registered-users-only.
- Browser task creation exposes public, private, and specific-user visibility. Organization and organization-team visibility and team assignee selection are not exposed in the browser because organization context in the browser is incomplete.
- Review tips are credit-only. Collectible or inventory-based tips remain deferred.
- The asset economy is platform-only: user-issued tokens, organization-issued tokens, crypto rewards, and external wallets are not implemented. Current implemented rewards are Sharecrop credits and platform collectibles, including multiple collectibles per task.
- Request/command contracts and HTTP contract fixture tests still need to expand as the API grows.

Known risks:

- MCP HTTP sessions and SSE replay buffers are in-memory. Idle sessions are evicted after a TTL, but state is still not shared across server restarts or multiple app processes.
- Standalone (user-owned) teams are not implemented. An attempt was reverted because it left the team store and HTTP layers inconsistent. See [DO_NEXT.md](./DO_NEXT.md).
- Foreign keys use the PostgreSQL default `NO ACTION`, which blocks deletion of referenced rows. The application has no deletion paths, so orphan rows cannot occur. Explicit `ON DELETE` behavior is not defined and should be designed alongside any future deletion feature.
