# Do Next

Current priority from [docs/application_readiness_review.md](./docs/application_readiness_review.md):

Next recommended work:

1. Add shared scenario parity tests that execute the same flows against the Go HTTP API and `site/demo/backend.js`. Start with create task, reserve, submit, review, notifications, collectibles, organization/team visibility, account tokens, and admin operations. See [docs/demo_semantic_parity.md](./docs/demo_semantic_parity.md).
2. Add paginated/typeahead browser selectors for large user/team/org directories.
3. Add provider email delivery only if the product direction changes; current account/org setup stays admin-driven.
4. Revisit GitHub Pages hard-refresh behavior after deployment. Pull request CI cannot observe the deployed Pages routing behavior.
5. Consider a Go/WASM demo-backend spike only after scenario parity tests exist. The spike should use explicit browser storage adapters and must not add fallbacks.

Recently finished:

1. The current branch added a persisted notification inbox, generated notification contracts, browser Inbox page, demo notification routes, HTTP contract fixtures, and domain/store tests.
2. The current branch added direct integration tests for audit event listing, notification lifecycle, Postgres rate-limit buckets, MCP HTTP session counts, and persisted MCP replay events. These tests require `DATABASE_URL`.
3. The current branch persisted MCP HTTP replay events in Postgres. Live SSE subscriber channels remain process-local.
4. The current branch added [docs/demo_semantic_parity.md](./docs/demo_semantic_parity.md), recommending shared scenario parity tests before any Go/WASM demo backend spike.
5. The combined runtime/audit/team-dashboard branch wired Postgres-backed rate-limit buckets, persisted MCP HTTP session identity for production `serve`, admin audit writes/viewing, team work dashboards, generated Admin Elm contracts, and demo base-path/bundle parity.
6. The branch kept email provider delivery and anonymous worker identity out of scope. Account and organization setup remains admin/org-admin driven.
7. The branch removed a response-encoding fallback in MCP raw response handling; response encoding failures now fail loudly.

Earlier finished:

1. The combined product-readiness branch added an operations runbook, systemd service template, operations-state schema foundation, and admin operations status endpoint.
2. The browser user selector can query `/api/users?query=...`; selector Playwright coverage now exercises the search path.
3. Account verification/reset token issue supports `SHARECROP_ACCOUNT_TOKEN_DELIVERY=log`, while local/test API-token mode remains available.
4. HTTP contract fixture coverage now includes account-token sent responses, user directory responses, submission comments, and operations status.
5. Account deletion semantics are deactivation plus password credential removal, refresh/account-token revocation, and email anonymization.
6. Submission comment posting uses a real form submit button, so click and Enter behavior match.
7. Reward docs now state the current product boundary: Sharecrop credits and admin-minted Sharecrop collectibles only.
- The backendless demo route surface is checked against the real HTTP router. Intentional real-only skips are health, MCP, and root/static serving routes.
- The backendless demo has Deno contract coverage for account lifecycle, user directory, task list/detail, and create-time collectible reward response shapes.
- The demo implements account lifecycle routes, `/api/users`, profile/password/account responses, clearer 404s for unknown demo API routes, email-backed org/team member provisioning, and create-time collectible reward escrow.
- Shared Playwright scenario constants cover account lifecycle and selector-backed reward creation flows.
- Real backend coverage for the carried-over behaviors is confirmed by existing HTTP E2E and targeted Playwright account/selector specs.

Remaining after recent combined PRs:

1. Keep expanding generated/fixture-level HTTP contract coverage as the API surface grows.
2. Revisit GitHub Pages hard-refresh behavior after deployment. The demo now handles `/demo/` base paths and serves the current Elm bundle, but pull request CI cannot observe the deployed Pages routing behavior.

Recently finished:

- The combined follow-up branch added account lifecycle endpoints/UI, authenticated user directory, selector-backed user/team task creation controls, create-time collectible escrow for collectible/bundle rewards, and real-app Playwright coverage.
- The earlier combined follow-up branch added organization role management, organization reviewer browser parity, task-local worker submissions, selector-backed organization flows, reward-kind creation, demo parity, and documentation updates.
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

- UI minors: add `type_ "button"` to any remaining secondary buttons that move into forms; continue replacing raw-id fields as directory-backed selectors become available on more pages.

Other queued work:

1. Do not add anonymous worker identity unless the product direction changes. Registered-user submissions are the current model.
2. Do not add provider email delivery yet. Account and organization setup stays admin/org-admin driven.

Before starting, reread [AGENTS.md](./AGENTS.md) and update the continuity files if task scope changes.
