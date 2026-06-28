# Do Next

Current priority from [docs/application_readiness_review.md](./docs/application_readiness_review.md):

1. Build account lifecycle: email verification, password reset/change, settings/profile edit, account deactivation/deletion, and browser guest entry if guest sessions remain part of the product.
2. Add searchable user/team directories so remaining recipient fields can use selectors instead of raw IDs.
3. Finish reward setup: collectible escrow during task creation, count handling, and clearer funding/open preconditions.
4. Add Playwright coverage for organization role management, worker task-local submissions, organization-team reservation, reward-kind creation, and selector flows against the real app.
5. Add operations foundation: deployment manifest, migration process, backups, logs/metrics, audit events, admin tools, and Postgres-backed MCP/rate-limit state for multi-process deployments.

Recently finished:

- The combined follow-up branch added organization role management, organization reviewer browser parity, task-local worker submissions, selector-backed organization flows, reward-kind creation, demo parity, and documentation updates.
- Organization-team assignment now has reservation/request-approval and submission eligibility through HTTP, MCP, selector-backed browser controls, and demo behavior.

Previously agreed multi-PR roadmap (humans + agents get full lifecycle/feature parity):

- PR1 (done): lifecycle/parity basics — MCP `open_task`/`fund_task` + participation in `create_task` + `list_tasks` state filter + `review_note` in `get_submission_status`; human UI authors a response schema + payload; docs scope names fixed.
- Series (done, `task/series-first-class`): task series promoted to a first-class domain — description + draft/published/closed lifecycle, creator-only add/remove/reorder of member tasks, a series comment thread, a stable URL, and draft-gating of member-task execution. This delivered series-level commenting; task-level and submission-level comments (the original PR3 scope) are still open below.
- PR2 (done, `task/dev-templates-comments`): pre-baked developer task types (code_review/security_review/product_review/ui_ux_review/qa_testing) with a template catalog in the create UI, a typed `reference_url` (the PR/resource to work on), and a per-task comment thread (HTTP + MCP + UI). This also delivered the task-level half of PR3's comments; submission-level comments remain.
- PR2 (superseded) was: pre-baked developer task TYPES — a task type/category and a catalog of templates (code review, security review, UI/UX review, product review, QA/testing) each with a description skeleton, a typed PR/URL field, and a ready-made response schema; surfaced in the create UI and `create_task`. (Net-new: type field across DB -> contract -> Elm -> MCP, plus a templates catalog.)
- PR3 (mostly done): comments now exist on SERIES and TASKS. Remaining: a comment thread on a SUBMISSION so requester and worker can discuss a specific submission (two-way, not just the one-way `review_note`). Reuse the `task_comments` shape.
- PR4 (descoped — do NOT propose a server-side scheduler): scheduling / recurrence is intentionally NOT a server feature. Decision (2026-06-25): recurring/scheduled task posting is the responsibility of a **local agent** (a client running a cron/work-loop that calls the existing MCP/API `create_task` + `open_task` + `fund_task` on its own cadence). The Sharecrop server stays request/response with no background job runner, `task_schedules` table, or recurrence model. If scheduling resurfaces, the answer is an agent-side recipe/example (e.g. a cron + MCP snippet in the docs), not a server scheduler.

Smaller parity follow-ups noticed during the review (fold into the PR they fit, or a cleanup PR): the human accept flow cannot send a collectible tip though the API supports it [DONE in task/ui-cancel-collectible-tip]; no UI to cancel a task [DONE], create/list standalone teams, deactivate an org member, or refund a collectible reward [DONE]; no file/image attachment anywhere (only inline JSON payload + free-text).

Polish follow-ups from `task/polish-bugfix-uiux-review`:

- UI minors: add `type_ "button"` to the remaining secondary buttons (latent today since none sit in a `<form>`); replace remaining free-text user/team recipient fields after directory endpoints exist.

Other queued work:

1. Make the real-Elm demo base-path aware for GitHub Pages: pass a base (the demo's path prefix) via flags and have `pageToPath`/`pageFromUrl` honor it, plus add a Pages SPA fallback (e.g. a `404.html`), so hard-refresh and deep-links on demo sub-routes work and the URL stays under `/demo/`. Today only in-app click navigation works (see BUGS.md).
2. Out-of-process session/rate-limiter store (Postgres). Put the MCP session store and the rate limiter behind interfaces (the in-memory implementations stay the default) and add Postgres-backed implementations: a `rate_limit_buckets` table (atomic token-bucket via a row-locked upsert) and an `mcp_sessions` table for session existence/eviction. The hard part is cross-process SSE replay fan-out, which needs `LISTEN/NOTIFY` (or a polling relay) — design that explicitly. This was scoped out of the collectible-tips/arcade PR to avoid shipping fragile pubsub.
5. Redesign anonymous worker identity and payout.
6. Add user-issued or organization-issued tokens.
7. Add crypto reward metadata.
8. Move MCP HTTP sessions and SSE replay buffers out of process if multi-process deployment becomes a requirement. The in-memory store evicts idle sessions after a TTL but is still per-process.
9. Expand request and command contracts and HTTP contract fixture coverage.

Before starting, reread [AGENTS.md](./AGENTS.md) and update the continuity files if task scope changes.
