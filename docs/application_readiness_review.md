# Sharecrop Application Readiness Review

This review compares the implemented application with the product thesis in [PLAN.md](../PLAN.md): Sharecrop coordinates requested work between people, organizations, teams, scripts, and local AI agents through a browser UI, HTTP API, MCP interface, task discovery, response validation, submissions, scoped tokens, escrow accounting, and payout workflow.

## Current Implemented Surface

- Email/password registration and login, JWT access tokens, rotating refresh-token cookies, logout revocation, guest subjects at the API level, and scoped agent credentials.
- Task creation, discovery, owner task list, task detail, open/cancel/unpublish, public/private/user/team/organization visibility, reservations, approvals, submissions, review outcomes, comments, task series, and user profiles.
- Sharecrop-schema parsing, validation, sensitive-field indexing, and redaction for receipt lookups.
- Credit ledger with signup grants, task funding escrow, accept/reject/request-changes settlement, refunds, partial payouts, tips, and organization funding.
- Platform collectibles, catalog awards, user/team/organization ownership, task collectible rewards, refunds, transfers, and collectible review tips.
- Organizations, organization teams, standalone teams, member provisioning/listing/deactivation at the API layer, team detail, and team member addition.
- MCP tools for task discovery/work/review, series, comments, reservations, and Streamable HTTP sessions with SSE replay.
- CI covers format, generated contracts, policy checks, Deno/TypeScript checks, lint, vet, unit tests, integration tests, HTTP end-to-end tests, and Playwright browser tests.

## Readiness Assessment

Sharecrop has a working core loop for a registered requester and a registered worker:

1. A requester creates a public task.
2. The requester funds and opens it.
3. A worker discovers it.
4. The worker submits JSON.
5. The requester reviews the submission.
6. Credits and/or collectibles settle through escrow.

That loop is well covered by HTTP and Playwright tests. The application is still short of ordinary product readiness because several flows are only API-level, ID-driven, or prototype-level, and production account, operations, billing, and support surfaces are absent.

## Highest Priority Gaps

1. **Organization-team task assignment is not workable.**
   - The task model and UI expose `organization_team` assignee scope.
   - `task.Service.Reserve` rejects every non-user assignee task with "this task does not accept user reservations".
   - There is no HTTP, MCP, or browser command to reserve/request approval as a team.
   - Result: a requester can create a task that displays as organization-team assigned, but the intended team cannot claim it through the current product.

2. **Organization reviewer and publisher flows are not usable in the browser.**
   - The backend has organization roles and review permissions.
   - The browser decides task ownership by `detail.createdBy == subjectId`, so an organization reviewer who did not create the task does not get review controls.
   - Member provisioning in the browser hardcodes the `member` role. There is no role picker, role update, deactivate button, or permission-aware UI.
   - Result: organization workflows depend on API calls and are not self-service in the UI.

3. **Worker revision and submission-discussion flows are underexposed.**
   - Workers can submit and can list their own submissions from the profile page.
   - The task detail fetches task submissions, but the backend only allows task owners/reviewers to list all task submissions.
   - The worker submissions page shows only task id and state, with no review note, response body, validation errors, submission comments, or direct revision workflow.
   - Result: "request changes" is implemented, but the worker has no strong browser workflow for seeing requested changes, discussing a specific submission, and resubmitting from that context.

4. **The reward economy is internal only.**
   - Credits are signup grants and internal ledger entries.
   - There is no purchase/top-up flow, withdrawal/payout method, invoice/billing system, wallet reconciliation dashboard, or accounting export.
   - Collectibles are platform/internal assets; user-issued tokens, organization-issued tokens, crypto metadata, external wallets, and automated crypto payout are deferred.
   - Result: rewards work for product mechanics, but not for a real paid marketplace.

5. **Account lifecycle is minimal.**
   - There is no email verification, password reset, password change, account deletion/deactivation, user settings page, OAuth provider integration, or invite acceptance flow.
   - The API has guest subjects, but the browser has no guest entry point.
   - Result: ordinary account recovery and administrative user lifecycle are missing.

6. **Operations are local-first.**
   - Runtime config is limited to address, database URL, migrations dir, and access-token secret.
   - There is Docker Compose for local Postgres, but no deployment manifest, backup/restore process, release procedure, migration rollback plan, metrics, tracing, structured audit log, or production admin console.
   - MCP sessions, SSE replay buffers, and rate-limit buckets are in memory.
   - Result: one process can run locally, but multi-process and production operations remain design work.

7. **Docs are not product-ready.**
   - `README.md` is local-command oriented.
   - `site/docs/index.html` has a task lifecycle and MCP quickstart.
   - There is no generated API reference, complete MCP reference page, onboarding guide, or operator runbook.
   - Result: a new user or integrator still needs source-level context for many workflows.

## Flow Review

### Visitor And Demo

Implemented:

- Static site and real-Elm demo exist.
- Demo uses an in-browser fake backend with seeded workflows.
- Demo has reset and hash routing.

Missing or partial:

- `docs/user_stories.md` still mentions mock social sign-in options, but the current Elm auth view only offers email/password login and registration.
- The docs page covers the core lifecycle and MCP quickstart, but not a complete API reference or operator runbook.
- Demo semantics can still drift from Go because `site/demo/backend.js` re-implements the backend in JavaScript.

### Authentication And User Account

Implemented:

- Register, login, refresh, logout, and API guest session creation.
- Refresh-token family reuse protection and logout revocation.
- Basic password-length validation and password hashing.

Missing or partial:

- No email verification or deliverability infrastructure.
- No password reset/change.
- No user settings/profile edit.
- No account deletion/deactivation.
- No browser guest flow.
- No OAuth/social login despite earlier story text referencing mock providers.

### Requester

Implemented:

- Create tasks with title, description, template, reference URL, response schema, payload, reward kind, credit reward amount, owner, participation policy, visibility, and assignee scope.
- Fund/open/cancel/refund from browser.
- Attach collectibles from the Collectibles page.
- Review submissions with accept, reject, request changes, partial credit, credit tips, collectible tips, and ban.

Missing or partial:

- Collectible-only and bundle tasks can be created from the task form, but the collectible count is still fixed to one by the HTTP parser and actual collectible escrow is still attached from the Collectibles page.
- Organization-owned funding can select an accessible organization.
- Organization visibility and organization-team reservation use selectors. User/team recipient fields still use raw IDs where no searchable directory endpoint exists.
- Series membership during task creation is not exposed; series add uses a raw task id.

### Implementor

Implemented:

- Discover public tasks.
- Include reserved tasks in discovery.
- View task detail, schema, payload, reward, participation, and availability.
- Reserve/request approval for user-assignee tasks.
- Submit JSON responses.
- View task-local own submissions with state, review notes, validation errors, response body, and submission comments.

Missing or partial:

- Organization-team assignee tasks can be reserved through browser selectors, but team-scoped submission dashboards and broader browser tests are still partial.
- There are no notifications for approval, request-changes, accept/reject, or comment events.
- There is no queue or inbox for tasks assigned to a user/team/organization beyond list/discovery/profile pages.

### Organization Operator

Implemented:

- Create organizations.
- List organizations.
- Create/list organization teams.
- Provision and list members.
- Add team members by email.
- Show organization/team collectible holdings.
- Fund organization-owned tasks from organization credits through API and raw-id browser field.

Missing or partial:

- Browser cannot choose provisioned roles; it sends `member` only.
- Browser has no deactivate-member control despite an API endpoint.
- Browser does not show permission-aware affordances for billing, reviewer, public publisher, or admin roles.
- Organization reviewers cannot review organization submissions in the browser unless they are also the task creator.
- Organization-team task assignee flow has no working reservation/submission path.

### Agent Operator

Implemented:

- Create/revoke/list scoped credentials.
- Copy MCP config and REST/MCP task examples.
- MCP supports task work, review, reservations, series, task comments, and submission comments.
- Streamable HTTP sessions support initialize, session-bound calls, SSE, replay, and delete.

Missing or partial:

- MCP HTTP session state and rate limits are process-local.
- There is no operator UI for active MCP sessions, last use, or abuse investigation.
- The task detail token helper mints broad worker tokens for the current user; there is no guided scope selection per task.
- Scheduled/recurring work is intentionally agent-side, but there is no recipe in product docs.

### Platform Admin And Economy

Implemented:

- Platform admins are configured through `SHARECROP_ADMIN_USER_IDS`.
- Admins can award catalog collectibles.

Missing or partial:

- No browser/admin page for configuring admins.
- No audit view for admin awards, ledger settlement, refunds, or disputes.
- No moderation workflow for abusive tasks/submissions/comments.
- No platform fee, billing, payout, or external wallet model.

### Data, Privacy, And Compliance

Implemented:

- Sensitive fields can be declared in schema, indexed on submission, and redacted for receipt lookup.

Missing or partial:

- No user-facing export/delete request flow.
- No retention job or deletion workflow for fields marked `delete_on_request`.
- No audit events for sensitive-field access/deletion.
- No attachment/object-storage model.

### Lists, Search, And Navigation

Implemented:

- API pagination exists for many list endpoints.
- Browser has filters for task state and discovery reserved inclusion.

Missing or partial:

- Browser list pages do not expose pagination controls.
- No search, sort, full-text filtering, or saved views.
- Several flows require copying raw UUIDs between pages.

## Documentation Drift Found

- `BUGS.md` still says review tips are credit-only, but collectible tips are implemented in the browser and backend.
- `BUGS.md` still says the browser cannot list organization members, but the organization detail page now lists members from `GET /api/organizations/{id}/members`.
- `docs/user_stories.md` still says mock social sign-in options exist, but the current real Elm auth view only has email/password login/register.
- `docs/user_stories.md` says collectible or inventory tips are deferred, but collectible tips are implemented.
- `docs/user_stories.md` has been updated for organization role management, worker task-local submissions, reward-kind creation, and selector coverage.

## Suggested Delivery Sequence

1. Finish account lifecycle: verification, reset/change password, settings, account deactivation/deletion, and browser guest entry if guests remain part of the product.
2. Add searchable user/team directories so remaining recipient fields can use selectors instead of raw IDs.
3. Finish reward setup: collectible escrow during task creation, count handling, and clearer funding/open preconditions.
4. Add Playwright coverage for organization role management, worker task-local submissions, organization-team reservation, reward-kind creation, and selector flows.
5. Add operations foundation: deployment manifest, migration process, backups, logs/metrics, audit events, admin tools, and Postgres-backed MCP/rate-limit state for multi-process deployments.
