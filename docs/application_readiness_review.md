# Sharecrop Application Readiness Review

This review compares the implemented application with the product thesis in
[PLAN.md](../PLAN.md): Sharecrop coordinates requested work between people,
organizations, teams, scripts, and local AI agents through a browser UI, HTTP
API, MCP interface, task discovery, response validation, submissions, scoped
tokens, escrow accounting, and payout workflow.

## Current Implemented Surface

- Email/password registration and login, JWT access tokens, rotating
  refresh-token cookies, logout revocation, guest subjects at the API level, and
  scoped agent credentials.
- Task creation, discovery, owner task list, task detail, open/cancel/unpublish,
  public/private/user/team/organization visibility, reservations, approvals,
  submissions, review outcomes, comments, task series, and user profiles.
- Sharecrop-schema parsing, validation, sensitive-field indexing, and redaction
  for receipt lookups.
- Credit ledger with signup grants, task funding escrow,
  accept/reject/request-changes settlement, refunds, partial payouts, tips, and
  organization funding.
- Platform collectibles, catalog awards, user/team/organization ownership, task
  collectible rewards, refunds, transfers, and collectible review tips.
- Organizations, organization teams, standalone teams, member
  provisioning/listing/deactivation at the API layer, team detail, and team
  member addition.
- MCP tools for task discovery/work/review, series, comments, reservations, and
  Streamable HTTP sessions with SSE replay.
- CI covers format, generated contracts, policy checks, Deno/TypeScript checks,
  lint, vet, unit tests, integration tests, HTTP end-to-end tests, shared
  scenario parity, and Playwright browser tests.

## Readiness Assessment

Sharecrop has a working core loop for a registered requester and a registered
worker:

1. A requester creates a public task.
2. The requester funds and opens it.
3. A worker discovers it.
4. The worker submits JSON.
5. The requester reviews the submission.
6. Credits and/or collectibles settle through escrow.

That loop is well covered by HTTP and Playwright tests. The application is still
short of ordinary product readiness because several flows are only API-level,
ID-driven, or prototype-level, and production account, operations, billing, and
support surfaces are absent.

## Highest Priority Gaps

1. **Team and organization work dashboards still need polish.**
   - Organization-team assignment, reservation/request-approval, and team-member
     submission eligibility exist through HTTP, MCP, browser controls, and the
     backendless demo.
   - Team detail pages split team work into review, ready-for-team, and
     assigned-to-team sections.
   - Team and organization task lists now have server-backed task search,
     task-type filters, sorting, and pagination. Organization task state filters
     are server-backed.
   - Team and organization queues have persisted saved views for repeated
     query/filter/sort combinations.
   - Result: the core organization-team task path works, but richer queue
     persistence and operator-specific queue defaults still need product polish.

2. **Worker revision and submission-discussion flows still need polish.**
   - Workers can submit and can list their own submissions from the profile
     page.
   - The task detail fetches task submissions, but the backend only allows task
     owners/reviewers to list all task submissions.
   - The worker submissions page shows review notes, response body, validation
     errors, sensitive-field metadata, and submission comments.
   - Submission-created, review, and submission-comment notifications exist.
   - The worker submissions page includes a revision inbox for requested
     changes.
   - The revision inbox can open the task detail with the previous response
     prefilled for editing.
   - Result: "request changes" is implemented, and the submissions page now
     exposes the basic revision history. Higher-volume timeline grouping still
     needs product polish if repeated rounds become common.

3. **The reward economy is internal only by product decision.**
   - Credits are signup grants and internal ledger entries.
   - Collectibles are Sharecrop platform assets minted by platform admins from
     the catalog.
   - User/org/per-project tokens, external wallets, and crypto integrations are
     out of scope.
   - Result: rewards work for internal Sharecrop incentives, not for external
     payout rails.

4. **Account lifecycle needs real delivery infrastructure.**
   - Email verification, password reset/change, profile email update,
     deactivation, and browser guest entry exist.
   - Verification/reset can be delivered through a configured log sink, while
     local/test mode can still return tokens through the API.
   - Result: product flows exist, but production email delivery needs an
     SMTP/provider adapter before public operation.

5. **Operations are single-process.**
   - Runtime config includes address, database URL, migrations dir, access-token
     secret, admin ids, cookie mode, and account-token delivery mode.
   - There is Docker Compose for local Postgres, a systemd service template, and
     an operator runbook.
   - MCP sessions, SSE replay buffers, and rate-limit buckets are in memory.
   - Result: one process can be operated, but multi-process state remains design
     work.

6. **Docs are still partial.**
   - `README.md` is local-command oriented.
   - `site/docs/index.html` has a task lifecycle and MCP quickstart, and links
     to API, MCP, operator, and agent-scheduling references in the repository.
   - A guided onboarding guide exists in `docs/onboarding.md`.
   - There is no generated OpenAPI reference.
   - Result: a new user or integrator still needs some source-level context for
     edge workflows.

## Flow Review

### Visitor And Demo

Implemented:

- Static site and real-Elm demo exist.
- Demo uses an in-browser fake backend with seeded workflows.
- Demo has reset and hash routing.

Missing or partial:

- The browser uses email/password login/register plus guest entry. Provider
  email delivery and social sign-in are not implemented.
- The docs page covers the core lifecycle and MCP quickstart, and links to the
  repository API reference, MCP reference, operator runbook, and agent-side
  scheduling recipe.
- Demo semantics can still drift from Go because `site/demo/backend.js`
  re-implements the backend in JavaScript.

### Authentication And User Account

Implemented:

- Register, login, refresh, logout, browser guest entry, and API guest session
  creation.
- Refresh-token family reuse protection and logout revocation.
- Basic password-length validation and password hashing.
- Email verification, password reset/change, profile email update, and account
  deactivation.

Missing or partial:

- No SMTP/provider adapter for production email delivery.
- Account lifecycle is deactivation plus credential/session/token revocation
  and email anonymization, not row removal.
- No OAuth/social login despite earlier story text referencing mock providers.

### Requester

Implemented:

- Create tasks with title, description, template, reference URL, response
  schema, payload, reward kind, credit reward amount, owner, participation
  policy, visibility, and assignee scope.
- Fund/open/cancel/refund from browser.
- Attach collectibles from the Collectibles page.
- Review submissions with accept, reject, request changes, partial credit,
  credit tips, collectible tips, and ban.

Missing or partial:

- Collectible-only and bundle tasks can be created from the task form with
  selected collectibles escrowed at create time.
- Organization-owned funding can select an accessible organization.
- Organization visibility, organization-team reservation, default-collectible
  award, collectible transfer, and series add-task flows use selectors where
  directory data exists.
- Raw IDs remain visible in protocol surfaces, links, audit/event metadata, and
  copyable API/MCP examples. No confirmed high-traffic user-entered raw-ID flow
  is currently listed.

### Implementor

Implemented:

- Discover public tasks.
- Include reserved tasks in discovery.
- View task detail, schema, payload, reward, participation, and availability.
- Reserve/request approval for user-assignee tasks.
- Submit JSON responses.
- View task-local own submissions with state, review notes, validation errors,
  response body, and submission comments.

Missing or partial:

- Organization-team assignee tasks can be reserved through browser selectors,
  but broader browser tests are still partial.
- Notifications exist for submission-created, review outcomes, and submission
  comments.
- Team detail pages expose team work sections. User and organization queueing
  still relies on list/discovery/profile pages and organization task lists.

### Organization Operator

Implemented:

- Create organizations.
- List organizations.
- Create/list organization teams.
- Provision and list members.
- Add team members by email.
- Show organization/team collectible holdings.
- Fund organization-owned tasks from organization credits.
- Choose provisioned roles, update roles, deactivate members, and review
  organization-owned task submissions when authorized.
- Organization detail exposes an operations dashboard with loaded balance,
  ledger rows, org-scoped audit rows, member, team, collectible, and task-state
  counts.
- Team and organization task queues have persisted saved views.

Missing or partial:

- Organization operations dashboard rollups are still oriented around loaded
  dashboard data rather than reporting queries for long time windows.

### Agent Operator

Implemented:

- Create/revoke/list scoped credentials.
- Copy MCP config and REST/MCP task examples.
- MCP supports task work, review, reservations, series, task comments, and
  submission comments.
- Streamable HTTP sessions support initialize, session-bound calls, SSE, replay,
  and delete.

Missing or partial:

- Production `serve` can use Postgres-backed MCP HTTP session identity, replay
  events, rate limits, notifications, audit events, saved views, and privacy
  requests. Persisted MCP live SSE subscribers poll the replay table for
  cross-process fan-out groundwork.
- There is no operator UI for active MCP sessions, last use, or abuse
  investigation.
- The task detail token helper mints broad worker tokens for the current user;
  there is no guided scope selection per task.
- Scheduled/recurring work is intentionally agent-side, but there is no recipe
  in product docs.

### Platform Admin And Economy

Implemented:

- Platform admins are configured through `SHARECROP_ADMIN_USER_IDS`.
- Admins can award catalog collectibles.
- Admins can inspect operations status, audit events, and privacy requests from
  the browser admin page.
- Admins can resolve queued privacy requests from the browser. Data-export
  resolution stores export JSON; sensitive-field deletion resolution marks
  delete-on-request sensitive-field metadata as redacted and records affected
  counts.
- Admins can list task moderation reports created by authenticated users. The
  current implementation persists reports as audit events.

Missing or partial:

- No browser/admin page for configuring admins.
- Moderation triage states/actions such as resolve, dismiss, annotate, and
  subject-specific links are not implemented yet.
- No platform fee, billing, payout, or external wallet model.

### Data, Privacy, And Compliance

Implemented:

- Sensitive fields can be declared in schema, indexed on submission, and
  redacted for receipt lookup.
- Submission responses expose indexed sensitive-field metadata so authorized
  users can see which response paths are governed by retention/redaction policy.
- Users can create audited privacy requests for data export or
  sensitive-field deletion.
- Platform admins can list and resolve privacy requests.
- Resolution stores data-export JSON or marks delete-on-request sensitive-field
  metadata as redacted without removing core rows.
- Sensitive-field redaction records affected counts and per-field redaction
  events.

Missing or partial:

- Retention automation remains a product job surface rather than a background
  scheduler.
- Sensitive-field access events are not recorded.
- No attachment/object-storage model.

### Lists, Search, And Navigation

Implemented:

- API pagination exists for many list endpoints.
- Browser has filters for task state and discovery reserved inclusion.
- Team and organization task queues expose search, task-type filters, sort,
  pagination, and persisted saved views.

Missing or partial:

- Browser task and discovery pages expose pagination controls. Some other list
  pages still rely on their first page or selector-local paging.
- Full-text search is not implemented.
- Raw UUIDs remain visible in links, protocol surfaces, metadata, audit rows,
  and API/MCP examples. No confirmed high-traffic user-entered raw-ID flow is
  currently listed. The current audit is in
  [raw_id_browser_flow_audit.md](./raw_id_browser_flow_audit.md).

## Documentation Drift Found

- Review tips support credits and collectibles in the browser and backend.
- User stories should continue to distinguish implemented collectible tips from
  deferred external reward systems.
- The browser uses email/password login/register plus guest entry. Provider
  email delivery and social sign-in are not implemented.

## Suggested Delivery Sequence

1. Keep expanding shared scenario parity for user-visible API surfaces and
   backendless demo behavior.
2. Keep expanding fixture-level HTTP contract coverage as request and response
   surfaces change.
3. Add Playwright coverage when browser workflows change materially.
4. Continue moving first-page-only lists to explicit pagination where
   high-volume use is expected.
5. Add provider email delivery only if account setup stops being admin-driven.
6. Do not replace the JavaScript backendless demo with WASM until the documented
   storage-adapter gates are met without fallbacks.
