# Do Next

Current priority from
[docs/application_readiness_review.md](./docs/application_readiness_review.md):

Next recommended work:

1. Add deeper privacy lifecycle coverage for export contents, sensitive-field
   redaction effects, and privacy request admin resolution UI if operators need
   browser-based request handling.
2. Keep expanding shared scenario parity as new user-visible API surfaces are
   added. The current suite covers selectors, collectible
   mint/transfer/create-time refund, comments, notifications with task metadata,
   team/organization queue search/type/sort, organization reviewer acceptance,
   sensitive-field response metadata, privacy request/audit shape, and
   multi-actor reservation/submission acceptance.
3. Keep expanding generated/fixture-level HTTP contract coverage as the API
   surface grows.
4. Audit remaining raw-ID browser flows and replace high-traffic fields with
   selectors where directory data exists. No confirmed high-traffic raw-ID input
   remains after the latest audit.
5. Keep improving backendless demo semantic parity through shared scenarios
   before any WASM replacement.
6. Add provider email delivery only if the product direction changes; current
   account/org setup stays admin-driven.
7. Do not replace `site/demo/backend.js` with WASM until the adoption gates in
   [docs/wasm_demo_backend_spike.md](./docs/wasm_demo_backend_spike.md) are met.

Recently finished:

1. The persisted-ops-privacy-lifecycle branch added persisted saved queue
   views, organization ledger and org-scoped audit dashboard panels, persisted
   privacy request listing/resolution with export JSON and sensitive-field
   redaction state updates, standalone-team task assignees, persisted MCP SSE
   polling fan-out groundwork, backendless demo parity, generated contracts,
   and continuity updates. Hard deletes remained prohibited.

1. The org-ops-queues-privacy branch added in-session saved views for team work
   and organization task queues, a loaded-data organization operations
   dashboard, a worker submission revision timeline, audited privacy request
   creation, generated Privacy Elm contracts, HTTP fixture/unit coverage,
   shared scenario parity for privacy request/audit shape, backendless demo
   parity, browser assertions, and docs updates.

1. The queue-revisions-ops-privacy branch added task-list `task_type` and
   `sort` filters, browser queue controls, audit-event filters, submission
   sensitive-field metadata in responses, visible worker submission response and
   privacy summaries, shared scenario parity for queue type/sort and sensitive
   metadata, backendless demo parity, and docs updates.

1. The server-queues/revisions/parity branch added task-list search filters,
   server-side search/pagination for organization task and team work queues, a
   revision-inbox shortcut that opens the task with the prior response
   prefilled, stricter task-list pagination errors, shared scenario queue-search
   coverage, HTTP E2E queue-search coverage, and backendless demo parity for
   organization-team queue search.

1. PR 82 was merged.

1. The post-PR81 dashboards/revisions/parity branch added loaded-list
   search/filter controls for team work, organization tasks, requester tasks,
   and discovery; added a worker revision inbox; surfaced dashboard load
   failures in section-specific messages; tightened submission-comment
   notification metadata parity; added browser coverage for the new flows; and
   added [docs/onboarding.md](./docs/onboarding.md).

1. PR 81 CI passed, including `db-checks` and Playwright.

1. The readiness-dashboard-docs-parity branch updated post-PR80 continuity,
   confirmed PR 80 CI passed with `db-checks` and Playwright, repaired stale
   readiness-review gaps, added team work dashboard sections, submission-comment
   notifications, inbox task links, task/discovery pagination controls,
   API/MCP/scheduling docs, and backendless demo thread-access parity.

1. PR 80 CI passed, including `db-checks` and Playwright.

1. The parity-contract-discussion-polish branch confirmed PR 79 CI passed with
   `db-checks`, cleaned stale readiness/user-story notes, and recorded that
   current raw-ID browser exposure is limited to protocol
   surfaces/links/audit/API examples unless a new high-traffic flow is found.
1. The branch expanded shared scenario parity with organization reviewer
   acceptance of an organization-owned task funded from the organization
   balance.
1. The branch tightened backendless demo review permissions for
   organization-owned tasks, initialized balances for newly created demo
   organizations, and added a submission-comment request wire-shape fixture.
1. The branch made the browser auto-open the submission discussion after a
   worker submits and after a reviewer action succeeds, with focused Playwright
   assertions added for that behavior.

1. The runtime-parity/reward-hardening branch added `make db-checks`, a PR CI
   database-backed runtime job, deployed Pages routing verification,
   organization-scoped collectibles, in-transaction collectible tips,
   deletion-semantics documentation, collectible contract fixture expansion, and
   shared scenario parity for create-time collectible refund.
1. The branch removed the separate accept-then-gift collectible tip window by
   settling collectible tips inside `LedgerStore.AcceptSubmission`.
1. The branch re-enabled `transferable_within_organization` tips with an
   explicit collectible organization scope and active-membership checks.
1. The branch kept rewards limited to Sharecrop credits and admin-minted
   Sharecrop collectibles.
1. PR 79 CI passed, including `db-checks`.

1. The multi-actor parity branch added scenario clients that carry distinct
   actor tokens and a shared scenario for approval-required reservation, owner
   approval, worker submission, owner acceptance with payout/tip, worker
   balance, and owner/worker notifications.
1. The branch made the backendless demo token-aware for local demo bearer
   tokens, so protected demo routes fail on missing/unknown tokens instead of
   silently acting as the seeded user.
1. The branch replaced the series add-task raw task-ID field with the existing
   task selector.
1. The branch added HTTP wire-shape fixtures for organization,
   organization-member, team-member, task-series-list, collectible, and
   collectibles wrappers.
1. The branch recorded the current WASM decision: keep the JavaScript demo
   backend until explicit browser storage adapters can satisfy the documented
   adoption gates without fallbacks.

1. The expanded-parity branch added admin operations, account-token issue shape,
   collectible catalog/mint/transfer, submission creation/comments, and
   notification read shape to the shared scenario parity suite.
1. The branch ran
   `deno task check:pages-routing -- --origin https://e6qu.github.io/sharecrop`
   successfully against the deployed GitHub Pages site.
1. The branch added selector-backed user/team/organization controls for admin
   default-collectible award recipients and collectible transfer recipients.
1. The branch expanded HTTP wire-shape fixtures for health/error/empty
   responses, ledger list, teams/tasks/reservations/submissions wrappers, full
   task response, agent credential request/created/list responses, and user
   profile response.
1. The scenario-parity/selectors/contracts branch added a shared scenario runner
   used by Deno demo tests and by `tools/run_scenario_parity.ts` for real API
   checks with an explicit origin/token.
1. The branch added paginated/typeahead selectors for users, organizations,
   standalone teams, and organization teams, with matching Go API query support
   and backendless demo query/pagination behavior.
1. The branch expanded HTTP wire-shape fixtures for request/command contracts
   and newer response surfaces including series, task comments, team detail,
   collectible catalog, and account-token responses.
1. The branch added `tools/check_pages_routing.ts` plus a Deno task for
   post-deploy GitHub Pages root/docs/demo/asset checks.
1. The branch documented the WASM demo backend spike in
   [docs/wasm_demo_backend_spike.md](./docs/wasm_demo_backend_spike.md),
   requiring explicit browser storage adapters and no fallback behavior before
   adoption.
1. The runtime-notifications branch added a persisted notification inbox,
   generated notification contracts, browser Inbox page, demo notification
   routes, HTTP contract fixtures, and domain/store tests.
1. The runtime-notifications branch added direct integration tests for audit
   event listing, notification lifecycle, Postgres rate-limit buckets, MCP HTTP
   session counts, and persisted MCP replay events. These tests require
   `DATABASE_URL`.
1. The runtime-notifications branch persisted MCP HTTP replay events in
   Postgres. Live SSE subscriber channels remain process-local.
1. The runtime-notifications branch added
   [docs/demo_semantic_parity.md](./docs/demo_semantic_parity.md), recommending
   shared scenario parity tests before any Go/WASM demo backend spike.
1. The combined runtime/audit/team-dashboard branch wired Postgres-backed
   rate-limit buckets, persisted MCP HTTP session identity for production
   `serve`, admin audit writes/viewing, team work dashboards, generated Admin
   Elm contracts, and demo base-path/bundle parity.
1. The branch kept email provider delivery and anonymous worker identity out of
   scope. Account and organization setup remains admin/org-admin driven.
1. The branch removed a response-encoding fallback in MCP raw response handling;
   response encoding failures now fail loudly.

Earlier finished:

1. The combined product-readiness branch added an operations runbook, systemd
   service template, operations-state schema foundation, and admin operations
   status endpoint.
2. The browser user selector can query `/api/users?query=...`; selector
   Playwright coverage now exercises the search path.
3. Account verification/reset token issue supports
   `SHARECROP_ACCOUNT_TOKEN_DELIVERY=log`, while local/test API-token mode
   remains available.
4. HTTP contract fixture coverage now includes account-token sent responses,
   user directory responses, submission comments, and operations status.
5. Account deletion semantics are deactivation plus password credential removal,
   refresh/account-token revocation, and email anonymization.
6. Submission comment posting uses a real form submit button, so click and Enter
   behavior match.
7. Reward docs now state the current product boundary: Sharecrop credits and
   admin-minted Sharecrop collectibles only.

- The backendless demo route surface is checked against the real HTTP router.
  Intentional real-only skips are health, MCP, and root/static serving routes.
- The backendless demo has Deno contract coverage for account lifecycle, user
  directory, task list/detail, and create-time collectible reward response
  shapes.
- The demo implements account lifecycle routes, `/api/users`,
  profile/password/account responses, clearer 404s for unknown demo API routes,
  email-backed org/team member provisioning, and create-time collectible reward
  escrow.
- Shared Playwright scenario constants cover account lifecycle and
  selector-backed reward creation flows.
- Real backend coverage for the carried-over behaviors is confirmed by existing
  HTTP E2E and targeted Playwright account/selector specs.

Recently finished:

- The combined follow-up branch added account lifecycle endpoints/UI,
  authenticated user directory, selector-backed user/team task creation
  controls, create-time collectible escrow for collectible/bundle rewards, and
  real-app Playwright coverage.
- The earlier combined follow-up branch added organization role management,
  organization reviewer browser parity, task-local worker submissions,
  selector-backed organization flows, reward-kind creation, demo parity, and
  documentation updates.
- Organization-team assignment now has reservation/request-approval and
  submission eligibility through HTTP, MCP, selector-backed browser controls,
  and demo behavior.

Previously agreed multi-PR roadmap (humans + agents get full lifecycle/feature
parity):

- PR1 (done): lifecycle/parity basics — MCP `open_task`/`fund_task` +
  participation in `create_task` + `list_tasks` state filter + `review_note` in
  `get_submission_status`; human UI authors a response schema + payload; docs
  scope names fixed.
- Series (done, `task/series-first-class`): task series promoted to a
  first-class domain — description + draft/published/closed lifecycle,
  creator-only add/remove/reorder of member tasks, a series comment thread, a
  stable URL, and draft-gating of member-task execution. This delivered
  series-level commenting; task-level and submission-level comments (the
  original PR3 scope) are still open below.
- PR2 (done, `task/dev-templates-comments`): pre-baked developer task types
  (code_review/security_review/product_review/ui_ux_review/qa_testing) with a
  template catalog in the create UI, a typed `reference_url` (the PR/resource to
  work on), and a per-task comment thread (HTTP + MCP + UI). This also delivered
  the task-level half of PR3's comments; submission-level comments remain.
- PR2 (superseded) was: pre-baked developer task TYPES — a task type/category
  and a catalog of templates (code review, security review, UI/UX review,
  product review, QA/testing) each with a description skeleton, a typed PR/URL
  field, and a ready-made response schema; surfaced in the create UI and
  `create_task`. (Net-new: type field across DB -> contract -> Elm -> MCP, plus
  a templates catalog.)
- PR3 (mostly done): comments now exist on SERIES and TASKS. Remaining: a
  comment thread on a SUBMISSION so requester and worker can discuss a specific
  submission (two-way, not just the one-way `review_note`). Reuse the
  `task_comments` shape.
- PR4 (descoped — do NOT propose a server-side scheduler): scheduling /
  recurrence is intentionally NOT a server feature. Decision (2026-06-25):
  recurring/scheduled task posting is the responsibility of a **local agent** (a
  client running a cron/work-loop that calls the existing MCP/API
  `create_task` + `open_task` + `fund_task` on its own cadence). The Sharecrop
  server stays request/response with no background job runner, `task_schedules`
  table, or recurrence model. If scheduling resurfaces, the answer is an
  agent-side recipe/example (e.g. a cron + MCP snippet in the docs), not a
  server scheduler.

Smaller parity follow-ups noticed during the review (fold into the PR they fit,
or a cleanup PR): the human accept flow cannot send a collectible tip though the
API supports it [DONE in task/ui-cancel-collectible-tip]; no UI to cancel a task
[DONE], create/list standalone teams, deactivate an org member, or refund a
collectible reward [DONE]; no file/image attachment anywhere (only inline JSON
payload + free-text).

Polish follow-ups from `task/polish-bugfix-uiux-review`:

- UI minors: add `type_ "button"` to any remaining secondary buttons that move
  into forms; continue replacing raw-id fields as directory-backed selectors
  become available on more pages.

Other queued work:

1. Do not add anonymous worker identity unless the product direction changes.
   Registered-user submissions are the current model.
2. Do not add provider email delivery yet. Account and organization setup stays
   admin/org-admin driven.

Before starting, reread [AGENTS.md](./AGENTS.md) and update the continuity files
if task scope changes.
