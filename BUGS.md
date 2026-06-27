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
- Detail-page load-vs-error distinction: the TaskDetail page now renders an error on a failed/forbidden fetch, but TeamDetail, SeriesDetail, and UserProfile still use `Result.toMaybe`, so an HTTP/decode error leaves them on a perpetual "Loading…" with no error shown.
- The demo (`site/demo/backend.js`) `reservationChange` and `reserve` handlers skip the ownership and `assignee_scope == user` guards the real backend enforces, so the demo will approve an already-declined reservation, approve a reservation on someone else's task, or let a user reserve an `organization_team`-scoped task. Low impact (showcase-only) but a fidelity gap.

Known risks:

- The demo runs the real Elm client (path-routed `Browser.application`) under the `/sharecrop/demo/` GitHub Pages base, but the client builds root-absolute URLs (`/tasks/{id}`). In-app click navigation works (pushState, no reload), but a hard-refresh or deep-link on a demo sub-route 404s on Pages, and the URL bar leaves the `/demo/` base. Fixing it cleanly needs base-path awareness in the Elm router (a base from flags prefixing `pageToPath`/`pageFromUrl`) plus a Pages SPA fallback; deferred (see DO_NEXT).
- `site/demo/backend.js` is a demo-only in-browser fake backend; it re-implements API behavior in JS and can drift from the Go backend's actual semantics. It is not used by the shipped app and is not contract-tested against the Go DTOs (only the demo smoke test exercises it).

- A collectible review tip and the credit settle are separate per-store transactions sequenced in one accept request (credit settle first, then `GiftCollectible`). The settle is idempotent and the gift is replay-safe (already-owned-by-worker is a no-op), so a retried accept recovers; the only residual is a small window where the credit accept commits but the gift fails (e.g. the collectible changed owner concurrently), returning an error after a committed accept. Folding the tip into the ledger transaction would remove the window.
- `transferable_within_organization` collectibles cannot be tipped yet: collectibles carry no organization, so the within-org bound is unenforceable and `AllowsTip` denies it rather than allow a cross-org gift. Re-enable once collectibles carry an org and the gift checks shared membership.
- The in-memory rate limiter evicts full buckets only on request arrival (at most once per refill window), so a burst of many distinct keys can transiently grow the map until the next sweep. Key sources are bounded (client IPs / verified agent subjects) and entries are tiny, so memory pressure is low; a background sweep would remove the transient growth.

- MCP HTTP sessions, SSE replay buffers, and the rate-limiter buckets are in-memory. Idle entries are evicted (and MCP sessions are capped per subject and globally), but this state is not shared across server restarts or multiple app processes, so the rate limits are per-process rather than a global quota.
- Standalone (user-owned) teams can be created and listed but are not yet selectable as a task assignee or visibility scope. See [DO_NEXT.md](./DO_NEXT.md).
- Foreign keys use the PostgreSQL default `NO ACTION`, which blocks deletion of referenced rows. The application has no deletion paths, so orphan rows cannot occur. Explicit `ON DELETE` behavior is not defined and should be designed alongside any future deletion feature.
