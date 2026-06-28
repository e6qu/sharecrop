# Do Next

Current priority from [docs/application_readiness_review.md](./docs/application_readiness_review.md):

1. Make organization-team assignment workable end to end: team reservation/request-approval commands, submission eligibility, HTTP/MCP tools, browser controls, and tests. Today the UI can mark a task as `organization_team`, but `Reserve` rejects non-user assignee tasks.
2. Add organization role management and browser reviewer parity: role picker on member provisioning, role update, deactivate-member button, permission-aware organization UI, and organization-reviewer review controls for tasks they did not personally create.
3. Add worker submission status/discussion UX: task-local "my submissions" panel, review notes, validation errors, response body, submission comments for workers, and a clear resubmit path after request-changes.
4. Replace raw id fields with selectors: organization funding, visibility scopes, team scopes, admin award recipients, collectible transfer recipients, and series task add.
5. Make reward creation coherent: reward-kind selector in task create, collectible/bundle reward setup in the same workflow, count handling, and clear funding/open preconditions.
6. Build account lifecycle: email verification, password reset/change, settings/profile edit, account deactivation/deletion, and browser guest entry if guest sessions remain part of the product.
7. Replace placeholder docs with user/API/MCP/operator docs, including the agent-side scheduling recipe.
8. Add operations foundation: deployment manifest, migration process, backups, logs/metrics, audit events, admin tools, and Postgres-backed MCP/rate-limit state for multi-process deployments.

Previously agreed multi-PR roadmap (humans + agents get full lifecycle/feature parity):

- PR1 (done): lifecycle/parity basics — MCP `open_task`/`fund_task` + participation in `create_task` + `list_tasks` state filter + `review_note` in `get_submission_status`; human UI authors a response schema + payload; docs scope names fixed.
- Series (done, `task/series-first-class`): task series promoted to a first-class domain — description + draft/published/closed lifecycle, creator-only add/remove/reorder of member tasks, a series comment thread, a stable URL, and draft-gating of member-task execution. This delivered series-level commenting; task-level and submission-level comments (the original PR3 scope) are still open below.
- PR2 (done, `task/dev-templates-comments`): pre-baked developer task types (code_review/security_review/product_review/ui_ux_review/qa_testing) with a template catalog in the create UI, a typed `reference_url` (the PR/resource to work on), and a per-task comment thread (HTTP + MCP + UI). This also delivered the task-level half of PR3's comments; submission-level comments remain.
- PR2 (superseded) was: pre-baked developer task TYPES — a task type/category and a catalog of templates (code review, security review, UI/UX review, product review, QA/testing) each with a description skeleton, a typed PR/URL field, and a ready-made response schema; surfaced in the create UI and `create_task`. (Net-new: type field across DB -> contract -> Elm -> MCP, plus a templates catalog.)
- PR3 (mostly done): comments now exist on SERIES and TASKS. Remaining: a comment thread on a SUBMISSION so requester and worker can discuss a specific submission (two-way, not just the one-way `review_note`). Reuse the `task_comments` shape.
- PR4 (descoped — do NOT propose a server-side scheduler): scheduling / recurrence is intentionally NOT a server feature. Decision (2026-06-25): recurring/scheduled task posting is the responsibility of a **local agent** (a client running a cron/work-loop that calls the existing MCP/API `create_task` + `open_task` + `fund_task` on its own cadence). The Sharecrop server stays request/response with no background job runner, `task_schedules` table, or recurrence model. If scheduling resurfaces, the answer is an agent-side recipe/example (e.g. a cron + MCP snippet in the docs), not a server scheduler.

Smaller parity follow-ups noticed during the review (fold into the PR they fit, or a cleanup PR): the human accept flow cannot send a collectible tip though the API supports it [DONE in task/ui-cancel-collectible-tip]; no UI to cancel a task [DONE], create/list standalone teams, deactivate an org member, or refund a collectible reward [DONE]; no file/image attachment anywhere (only inline JSON payload + free-text).

Polish follow-ups from `task/polish-bugfix-uiux-review`:

- UI minors: add `type_ "button"` to the remaining secondary buttons (latent today since none sit in a `<form>`); replace free-text org/team/scope id inputs with picker dropdowns.

Other queued work:

1. Make the real-Elm demo base-path aware for GitHub Pages: pass a base (the demo's path prefix) via flags and have `pageToPath`/`pageFromUrl` honor it, plus add a Pages SPA fallback (e.g. a `404.html`), so hard-refresh and deep-links on demo sub-routes work and the URL stays under `/demo/`. Today only in-app click navigation works (see BUGS.md).
2. Out-of-process session/rate-limiter store (Postgres). Put the MCP session store and the rate limiter behind interfaces (the in-memory implementations stay the default) and add Postgres-backed implementations: a `rate_limit_buckets` table (atomic token-bucket via a row-locked upsert) and an `mcp_sessions` table for session existence/eviction. The hard part is cross-process SSE replay fan-out, which needs `LISTEN/NOTIFY` (or a polling relay) — design that explicitly. This was scoped out of the collectible-tips/arcade PR to avoid shipping fragile pubsub.
5. Redesign anonymous worker identity and payout.
6. Add user-issued or organization-issued tokens.
7. Add crypto reward metadata.
8. Move MCP HTTP sessions and SSE replay buffers out of process if multi-process deployment becomes a requirement. The in-memory store evicts idle sessions after a TTL but is still per-process.
9. Expand request and command contracts and HTTP contract fixture coverage.

Before starting, reread [AGENTS.md](./AGENTS.md) and update the continuity files if task scope changes.
