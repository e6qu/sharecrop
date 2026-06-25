# Do Next

Agreed multi-PR roadmap (humans + agents get full lifecycle/feature parity). PR1 is merged; do the rest in order, one PR each:

- PR1 (done): lifecycle/parity basics — MCP `open_task`/`fund_task` + participation in `create_task` + `list_tasks` state filter + `review_note` in `get_submission_status`; human UI authors a response schema + payload; docs scope names fixed.
- Series (done, `task/series-first-class`): task series promoted to a first-class domain — description + draft/published/closed lifecycle, creator-only add/remove/reorder of member tasks, a series comment thread, a stable URL, and draft-gating of member-task execution. This delivered series-level commenting; task-level and submission-level comments (the original PR3 scope) are still open below.
- PR2: pre-baked developer task TYPES — a task type/category and a catalog of templates (code review, security review, UI/UX review, product review, QA/testing) each with a description skeleton, a typed PR/URL field, and a ready-made response schema; surfaced in the create UI and `create_task`. (Net-new: type field across DB -> contract -> Elm -> MCP, plus a templates catalog.)
- PR3: comments / clarifying questions on TASKS and SUBMISSIONS — series-level comments now exist; extend the same pattern to a discussion thread on a task and on a submission so a worker can ask before submitting and a requester can answer (two-way, not just the one-way `review_note`). Reuse the `series_comments` shape (a generic `comments` target table, or per-entity tables) + endpoints + MCP tools + UI.
- PR4: scheduling / recurrence — a recurrence model and a runner that materializes task instances over time (today nothing schedules; `task-series` is only a grouping label).

Smaller parity follow-ups noticed during the review (fold into the PR they fit, or a cleanup PR): the human accept flow cannot send a collectible tip though the API supports it; no UI to cancel a task, create/list standalone teams, deactivate an org member, or refund a collectible reward; no file/image attachment anywhere (only inline JSON payload + free-text).

Other queued work:

1. Make the real-Elm demo base-path aware for GitHub Pages: pass a base (the demo's path prefix) via flags and have `pageToPath`/`pageFromUrl` honor it, plus add a Pages SPA fallback (e.g. a `404.html`), so hard-refresh and deep-links on demo sub-routes work and the URL stays under `/demo/`. Today only in-app click navigation works (see BUGS.md).
2. Out-of-process session/rate-limiter store (Postgres). Put the MCP session store and the rate limiter behind interfaces (the in-memory implementations stay the default) and add Postgres-backed implementations: a `rate_limit_buckets` table (atomic token-bucket via a row-locked upsert) and an `mcp_sessions` table for session existence/eviction. The hard part is cross-process SSE replay fan-out, which needs `LISTEN/NOTIFY` (or a polling relay) — design that explicitly. This was scoped out of the collectible-tips/arcade PR to avoid shipping fragile pubsub.
5. Redesign anonymous worker identity and payout.
6. Add user-issued or organization-issued tokens.
7. Add crypto reward metadata.
8. Move MCP HTTP sessions and SSE replay buffers out of process if multi-process deployment becomes a requirement. The in-memory store evicts idle sessions after a TTL but is still per-process.
9. Expand request and command contracts and HTTP contract fixture coverage.

Before starting, reread [AGENTS.md](./AGENTS.md) and update the continuity files if task scope changes.
