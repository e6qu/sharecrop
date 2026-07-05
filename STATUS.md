# Status

The repository contains pull request 1 through pull request 132 work, merged
into `main`, plus the current `task/fund-any-reward-kind-and-open-on-create`
branch. PR 108's GitHub Pages deployment failed three times in a row after
merge for what looked like a transient GitHub-side Pages backend issue
(build/artifact steps always succeeded; only `deploy-pages` failed or hung,
with a different symptom each time); most later PRs' deployments succeeded
on the first try with no code or workflow changes, confirming it was not a
code problem — though PR 127's deployment hit the same transient failure
again and cleared on a manual retry, so this class of flakiness is still
occasionally live, not fully resolved.

The 5-phase RBAC + API-token effort (PRs 115-121), two clean-up passes
(PRs 122, 124), a docs refresh (PR 123), the WASI production-hosting spike's
plan + Phase 0/1 (PR 125), ecosystem research (PR 126), and deployment-shape
requirements (PR 132), a Go 1.26.4 upgrade (PR 127), a strengthened
"at most one open PR at a time" rule in `AGENTS.md` (PR 128), the
`site/demo/backend.js` deprecation (PR 129: replacement CI coverage; PR 130:
deletion), and a fix for the demo (WASM) backend collapsing every rejection
to HTTP 500 plus a corrected fund-panel visibility gate (PR 131) are
complete.

Active task: `task/fund-any-reward-kind-and-open-on-create` addresses a set
of related feature requests. Two are done here:

1. **A draft task is always fundable by its creator, regardless of the
   reward kind it was created with.** Previously funding required
   reward_kind credit/bundle; a none/collectible-only task now transitions
   to credit/bundle on first funding (`internal/db/ledger_store_helpers.go`'s
   new `requireFundableTask`, shared by personal and organization funding;
   mirrored in `internal/wasmdemo/request_handler.go`'s funding case). The
   Elm fund panel's visibility gate (`canFund` in `View.elm`) is now just
   "task is draft," not reward-kind-restricted.
2. **Creating a task now opens it in the UI for further editing**, with the
   browser URL updating to `#/tasks/{id}` (`Main.elm`'s `CreateTaskReceived`
   now calls `enterPage (TaskDetailPage created.id)` plus `Nav.pushUrl`,
   instead of clearing the create form and staying on it). This changed the
   assumption behind ~13 existing Playwright tests that expected to stay on
   the create page after submission — all updated to check `detail-title`
   instead of the old `create-message` text.

Still to do from the same request, not started in this branch: a creator
adding collectibles to an existing (not just brand-new) task's reward from
the task detail page UI; an admin UI for awarding collectibles to any user
(the backend route `POST /api/collectibles/award` already exists,
platform-admin-gated, catalog-only); an org admin awarding org-owned
collectibles to an org member (needs more design — the existing transfer
mechanism's permission model requires verifying it supports "org owns it,
org admin acts on the org's behalf" rather than only personal ownership).
including the two items PR 122 had flagged rather than fixed — plus any
other bug/UI/API/performance/concurrency issue found along the way,
regardless of relation to the RBAC effort. Used parallel review agents
covering concurrency/races/deadlocks, test-suite wiring integrity, and a
final dead-code sweep (Go `deadcode` + Elm + WASM-demo-specific, since
neither of the first two catch unused struct fields or Elm dead code).

Real fixes applied:
- **`internal/org/service.go`'s authz duplication (PR 122's flag #1),
  resolved**: extracted the shared "does this org-wide credential match
  this organization" actor-kind check into a new leaf package,
  `internal/orgactor`, that both `internal/authz` (task/series/submission)
  and `internal/org` (its own membership/team endpoints) now call, instead
  of `org` hand-rolling its own copy of the same logic. This was the
  correct fix for the import-cycle constraint (`authz` already imports
  `org` for `org.Permission`) — a new dependency-free package, not a
  reshuffle of the existing two.
- **A real, if narrow, TOCTOU race in the Postgres-backed rate limiter**
  (`internal/db/rate_limit_store.go`): `select ... for update` on a
  not-yet-existing bucket row has nothing to lock, so concurrent *first*
  touches of a brand-new rate-limit key could each read "no bucket yet" and
  each grant, over-admitting past capacity. Fixed by inserting the row
  (`on conflict do nothing`) before locking it, so Postgres's own
  conflicting-insert serialization protects the first touch too. Verified
  by code reasoning about Postgres's documented `INSERT ... ON CONFLICT`
  concurrency semantics, not a reliably-failing/passing test — the race
  window proved too narrow to trigger locally even at 200 concurrent
  goroutines against a fast local Postgres, so no regression test could be
  written honestly; noted rather than papered over with a test that would
  pass regardless of the fix.
- **MCP session store methods panicking on any transient database error**
  (`internal/http/mcp_sessions.go`) — crashing the *entire process*, not
  just failing one request. Especially serious for the background SSE
  replay-polling goroutine (`pollPersistedEvents`), which runs one instance
  per live subscriber: at real concurrency (the user's stated target is
  ~100 concurrent MCP SSE sessions), a single transient DB hiccup would
  have taken down every session and all other API traffic at once. Fixed
  every call site to log and fail closed (matching what "session not
  found"/"refused" already meant to callers) instead of panicking. Added a
  regression test (`TestMCPSessionStoreDegradesGracefullyOnPersistenceFailure`)
  with a fake persistence layer that always errors, proving no path panics.
- **Dead code**: an unreachable `fetchUserDirectoryQuery` wrapper in
  `Api.elm` (superseded by a direct call the real caller already uses), and
  a fully orphaned `ModerationTriageHandler`/`NewModerationTriageHandler`/
  `.Handle`/`moderationTriagePathReportID` in `internal/wasmdemo` — reachable
  only from their own dedicated tests, with the live WASM dispatch path
  routing through a separate, independently-implemented
  `AdminHandler.handleAdminModerationReports` instead. Both `go tool
  deadcode` and the Elm compiler miss this class of dead code (unused
  struct fields / functions with a same-shape live sibling), so this needed
  a dedicated pass.
- Confirmed clean (no changes needed): `go test -race` across the full
  suite (unit/integration/http_e2e) reports no data races; the test suite
  itself is genuinely wired (no skips, no orphaned build-tagged tests, both
  Deno and Playwright tiers are live in CI, no over-mocking); `go tool
  deadcode -test ./...` reports zero unreachable Go code.

**Still an open, explicit decision — not resolved in this PR**:
`site/demo/backend.js` (the orphaned JS mock backend, unreachable from the
live browser demo but still exercised directly by two Deno tests) was
flagged again as a deletion candidate, and a working WASM-based replacement
for its scenario-parity coverage was identified and verified
(`deno task check:scenario-parity:wasm`, not currently wired into CI) — but
no equivalent replacement exists yet for its route-drift-detection test.
Attempting the deletion was correctly blocked by the safety classifier as
an irreversible action needing explicit user confirmation, and a follow-up
clarifying question went unanswered, so the file and its tests remain
untouched. This is likely to resolve naturally once the separately-planned
WASM/production backend unification (see project memory) actually
supersedes it.

See `WHAT_WE_DID.md` for the full writeup.

`task/task-detail-reorder-profile-links-uiux` (PR 114, merged into `main`)
refined the task detail and profile pages for usability. Report task is now
a collapsed disclosure. Reservation status moved to the top of the task
detail page, above role-specific controls — this surfaced a real gap: task
owners previously had no way to see or act on a pending reservation request
through the browser at all (the Approve/Decline buttons existed but were
never reachable by owners). Fixed, plus added a new test since that flow
had zero prior coverage. Also scoped reservation/submission action buttons
to who's actually entitled to click them (previously any worker saw every
reservation's buttons, including other people's). People now link to their
profiles wherever their user ID appears (reservation holder, submitter, task
creator, notification actor, admin user ID). Profile pages: the
"Submissions" link now only shows on your own profile (the API 403s for
anyone else), and "Public work" is relabeled "Currently working on" with
richer per-task info, since it's current active work, not a full history.

`task/merge-tasks-nav-uiux-polish` (PR 113, merged into `main`) consolidated
the nav further at the user's explicit direction — several destinations were
"useless" as their own top-level items and should live on the Tasks page
instead. New task, Discovery, and the whole Work menu (Submissions, Series)
are gone as separate nav destinations; Tasks is now a hub showing My tasks
and Discover public tasks (both always expanded) plus collapsed
My-submissions/Series sections, with a "+ New task" button at the top.
Inbox moved into the Account menu. Nav is now just Overview/Tasks flat plus
Manage/Account menus (down from 8 items). Also added inline task funding (a
collapsed-by-default "Fund this task" panel on the task detail page, plus a
"Fund" button on task-list rows), reusing the standalone Funding page's
exact Msg plumbing.
Found and fixed two real pre-existing bugs uncovered by this refactor (the
submissions-pagination and series-refresh handlers were keyed off the
current page being an exact standalone route, so they silently no-op'd from
the new hub) — see `WHAT_WE_DID.md` for the full writeup, including a mobile
overflow bug and a Playwright strict-mode failure the test suite caught
(mara's own task is also public, so it now appears in both Tasks sections at
once).

`task/navbar-dropdown-menu-more-seed-tasks` (PR 112, merged into `main`)
followed up on PR 111's navbar grouping (still 15 buttons across 3 rows) with
real dropdown menus: Overview/Tasks/New task/Discovery/Inbox stayed flat, and
Submissions/Series ("Work"), Funding/Collectibles/Agents/Organizations
("Manage"), and Profile/Admin/Log out/Reset demo ("Account") each collapsed
into a menu — one row instead of three. The first implementation attempt
used a native `<details>`/`<summary>` (matching `Ui.disclosure`'s
no-Elm-state philosophy) but had two real bugs the Playwright suite caught:
Elm silently drops inline `onclick` attributes (a deliberate security
measure), so an attempted native-JS fix to close the menu on navigation was
a no-op, and without it the floating panel stayed open over whatever page
loaded next and intercepted clicks on it. Fixed by making the dropdown
Elm-controlled instead (`openNavMenu : Maybe String`, reset to `Nothing` in
`enterPage` on every route change). Also expanded the WASM demo's seeded
tasks from 6 to 14 for a less sparse first impression, without touching the
frozen balance/catalog-count invariants existing Playwright specs depend on.

`task/ui-navbar-declutter-a11y-seed` (PR 111, merged into `main`) was a
deliberately large, bundled UI/UX pass (explicitly requested as one PR rather
than the usual one-task-per-branch split), covering: a grouped `<nav>` navbar (replacing the
old flat 14-button row) with a fixed Profile active-state bug and a new
top-level Submissions link; further page decluttering via `Ui.disclosure` on
Create Task's ownership/access fields, the Collectibles mint/award forms, the
account-settings card, the user-submissions "all submissions" list, and the
series creator-controls/comments sections; an accessibility pass adding a
per-page `<h1>` (previously the app had exactly one static "Sharecrop" `<h1>`
that never changed per route), `aria-hidden` on decorative collectible
sprites, color-differentiated `Ui.badgeVariant` status badges, and a real
focus-visible ring on the base theme's text inputs/textareas (previously
suppressed with no visible replacement); and a richer WASM demo seed (credit
grants for the non-admin users, an organization team, more organization
members, a funded task, two more discoverable tasks, a pending
submission/reservation/inbox notification, and awarded collectibles) without
touching the seed invariants (`mara`'s 1250-credit balance, the org's 7200
balance, and the 25-entry collectible catalog) that dozens of existing
Playwright assertions depend on. Also fixed, as "boy scout rule" opportunistic
issues found along the way: `arcade.css`'s `[data-testid^="nav-"]` rule was
unconditionally overriding every nav link's background, so the active-page
nav highlight was invisible across the *entire* app in the demo skin, not just
for Profile; `Ui.dangerButtonClass` was missing the `min-h-[44px]` touch
target every other button class has; the task-detail "API & MCP" panel used a
bespoke `state.taskIntegrationOpen`/`ToggleTaskIntegration` toggle instead of
the shared `Ui.disclosure` component; and `reviewControls` hand-rolled its
label/input markup instead of reusing `Ui.fieldLabel`/`Ui.textInput`.

`task/task-series-wasm-support` (PR 110, merged into `main`) continued the
hand-testing pass onto the Task Series feature and found a fifth real bug, the
biggest gap yet: `/api/task-series` (list, create) and `/api/task-series/{id}`
(detail) were entirely unclassified in the WASM demo (a 404), so creating a
task series through the browser failed outright — the whole feature had zero
WASM demo support. Implemented `StoredTaskSeries` storage and a new
`TaskSeriesHandler` covering create/list/detail, matching `internal/http`'s
`taskSeriesResponse`/`taskSeriesDetailResponse` shapes (including that create
returns the full detail wrapper, not a bare series object — found by hitting a
second, different decode error after fixing the first 404). Series lifecycle
transitions (publish/unpublish/close/reopen), series-task membership
(add/remove/reorder), and series comments are explicitly not implemented yet:
clicking those actions shows a graceful inline error rather than crashing, but
they do not work. This is a known, larger remaining gap (see "Next recommended
work" below), analogous to the still-missing team-membership storage.

Hard deletes remain out of scope; use soft lifecycle states, anonymization,
redaction, tombstones, and audit records. Email/provider delivery, anonymous
worker identity, per-project tokens, external wallets, and crypto integrations
are out of scope.

Current implemented surface:

- Organization member provisioning has role selection; organization members can
  have roles updated or be deactivated through the browser and API.
- Organization member lists expose non-removed lifecycle rows, so managers can
  see deactivated members while deactivated members no longer satisfy active
  membership permissions.
- Organization reviewers can see review controls on organization-owned tasks
  they did not create.
- Workers see task-local own submissions with state, review notes, validation
  errors, response body, and submission comments.
- Organization-team reservation, organization funding, organization visibility,
  and organization award-recipient flows use paginated/typeahead selectors where
  data is available.
- Task creation has reward-kind selection for no reward, credits, collectible,
  and bundle rewards.
- Hosted docs and readiness/user-story docs no longer describe `/docs/` as a
  placeholder.
- Account lifecycle exists for guest entry, email-verification token
  issue/confirm, password reset/change, profile email update, and account
  deactivation.
- Authenticated user directory and selector-backed user/team/organization
  controls in task creation use query and pagination where data is available.
- Collectible and bundle task creation can escrow selected collectibles
  immediately and task responses show the held collectible count.
- The legacy `site/demo/backend.js` route checks cover current real API routes
  except the documented real-only health/MCP/root routes.
- Users have a persisted notification inbox for submission-created and
  submission-review events. The browser has an Inbox page with mark-read
  support, and the demo backend paths mirror the routes and seeded unread state.
- The Go/WASM demo backend supports account lifecycle, user directory,
  organization/team member provisioning, selector pagination/query, create-time
  collectible rewards, token-aware actor flows, and shared scenario parity flows
  closely enough for browser demo coverage.
- The Go/WASM demo backend supports agent credential creation, listing, and
  revocation for the task-detail API/MCP panel, profile agent access panel, and
  Agents page.
- The Go/WASM demo backend's `GET /api/users/{user_id}` matches the real
  backend's profile shape (`{id, tasks}`, tasks the user created) and
  `GET /api/users/{user_id}/work` lists tasks the user holds an active assignee
  reservation on, so a user's profile page (public tasks and public work tabs)
  renders correctly in the browser demo.
- Secondary sections on the busiest pages (Admin's five sections, the
  Tasks/Discovery filter panels, Create Task's advanced/optional fields, the
  organization detail page's task filters/Teams/Members/Collectibles, the team
  detail page's team-work filter panel, and the Collectibles page's admin-only
  award-recipient picker) are collapsible (`Ui.disclosure`, native
  `<details>`/`<summary>`) and collapsed by default unless already in use, so
  those pages read short at a glance and expand on demand.
- The Go/WASM demo backend's `GET /api/teams/{team_id}` returns the stored team
  plus an empty member list (no team-membership storage exists in the demo yet),
  matching `internal/http`'s `getTeam`/`teamDetailResponse` shape, so the team
  detail page loads for any standalone or organization-owned team instead of
  404ing.
- The Go/WASM demo backend supports Task Series create/list/detail
  (`POST`/`GET /api/task-series`, `GET /api/task-series/{id}`), matching
  `internal/http`'s `taskSeriesResponse`/`taskSeriesDetailResponse` shapes, so
  the browser's "Task series" page can create and open a series. Publishing,
  closing, reopening, adding/removing/reordering tasks in a series, and series
  comments are not implemented in the WASM demo yet; those actions fail with a
  graceful inline error rather than a crash.
- The Go/WASM demo backend's
  `GET /api/organizations/{organization_id}/audit-events` scopes the shared
  audit-event store to that organization's subject id, matching the real
  backend's `listOrganizationAuditEvents`, so the organization detail page's
  "Organization audit" section loads instead of 404ing.
- Admin operations status is available to platform admins at
  `/api/admin/operations`.
- Authenticated users can report tasks through the task detail page. Reports are
  persisted as `moderation_report_created` audit events and platform admins can
  list them in the Admin moderation panel and through
  `/api/admin/moderation/reports`.
- Audit record results carry the exact recorded event, so API handlers that need
  to echo newly recorded audit-backed workflow records do not reload a guessed
  latest event.
- Production `serve` wires Postgres-backed rate-limit buckets, audit events,
  notification inbox rows, persisted MCP HTTP session identity, persisted MCP
  HTTP replay events, saved queue views, and privacy requests. Persisted MCP
  live SSE subscribers poll the replay table for cross-process fan-out
  groundwork.
- Platform admins can view audit events at `#/admin`; audit writes cover admin
  default collectible awards, account deactivation, organization member
  provisioning/role/deactivation actions, submission review outcomes, and task
  refunds.
- Team detail pages load a team review queue and team work list from
  `/api/teams/{team_id}/work`.
- Team detail pages split team work into review, ready-for-team, and
  assigned-to-team sections.
- Team work and organization task queues support server-side search and
  pagination, task-type filters, and sorting. Organization task state filters
  are server-backed.
- Team work and organization task queues have persisted saved views for reusable
  query/filter/sort combinations.
- Organization detail pages expose an operations dashboard with loaded balance,
  ledger rows, org-scoped audit rows, member, team, collectible, and task-state
  counts.
- Admin audit event listing supports action, subject-kind, subject-id, and page
  filters through the API and browser controls.
- Admin audit, platform-admin, privacy-request, and moderation-report lists have
  explicit browser pagination controls.
- Platform admins are stored through explicit runtime services. Bootstrap admins
  come from `SHARECROP_ADMIN_USER_IDS`; admin-granted platform admins are
  persisted and revoked through a lifecycle state instead of row deletion.
- The admin page includes platform-admin configuration backed by the paginated
  user selector, privacy retention execution, moderation state filtering, direct
  moderation subject links where routes exist, and moderation triage actions for
  open/resolved/dismissed states with notes.
- Submission responses include indexed sensitive-field metadata, and browser
  submission history views show response bodies, validation errors, review
  notes, sensitive-field summaries, and revision shortcuts where available.
- Worker submission profile pages include a revision timeline for submission
  state, review-note, validation-error, and sensitive-field history.
- Users can create persisted audited privacy requests for data export or
  sensitive-field deletion. Platform admins can list and resolve requests.
  Resolution stores data-export JSON with owned account/submission/sensitive
  metadata, or marks delete-on-request sensitive-field metadata as redacted
  without removing core rows.
- Platform admins can run sensitive-field retention. The Postgres store redacts
  active delete-on-request sensitive-field metadata, records per-field redaction
  events, records the retention run, and writes a privacy retention audit event.
- Authorized submission-list/profile reads record sensitive-field access events
  when returned submissions include sensitive-field metadata.
- Sensitive-field response metadata includes lifecycle state and redaction time.
  Privacy sensitive-field resolution records affected counts and per-field
  redaction events.
- Requester task lists and discovery lists have loaded-list search/filter
  controls.
- Worker submission profile pages include a revision inbox for submissions in
  `changes_requested`, with a shortcut that opens the task detail and prefills
  the prior response for editing.
- Team/organization dashboard load failures surface section-specific messages
  instead of silently rendering empty lists.
- The static demo serves the current compiled Elm bundle, defaults to the
  compiled Go/WASM backend path, includes the admin operations/audit route, and
  handles `/demo/` and local root-served base paths explicitly.
- Account verification/reset token issue supports API-visible local/test mode
  and log-delivery mode.
- Account deactivation anonymizes email, removes password credentials, and
  revokes active refresh/account tokens.
- Selector APIs support `query`, `limit`, and `offset` for users, organizations,
  standalone teams, and organization teams where those lists are exposed.
- A shared scenario parity runner covers selector pagination/query, admin
  operations, account-token issue shape, privacy request/audit/resolution shape,
  moderation report/admin-list/audit shape, collectible catalog/mint/transfer,
  agent-credential creation/revocation, organization/team/task/task-comment
  creation, submission creation/comments, notification read shape, and a
  multi-actor reservation approval/submission acceptance/payout/notification
  flow against the backendless demo. It can be run against a real API with an
  explicit admin origin/token/refresh-token session.
- The shared scenario parity runner also covers organization reviewer acceptance
  of an organization-owned task funded from the organization balance.
- The shared scenario parity runner covers submission-comment notifications,
  team/organization queue search/type/sort behavior, persisted saved queue
  views, small task/submission attachments, and sensitive-field response
  metadata.
- The shared scenario parity runner covers organization member
  provisioning/listing/role-update/deactivation shape.
- The real API shared scenario parity runner probes `/healthz`, accepts token
  and refresh-token file inputs, carries refresh-cookie rotation, and reports
  invalid JSON and status errors with request context before running the shared
  scenario suite.
- A local real API shared scenario parity runner registers a scenario admin,
  grants platform-admin state through `DATABASE_URL` and `psql`, and runs the
  same shared scenario suite against a real local API without a fallback admin
  path.
- A GitHub Pages routing check script verifies deployed root/docs/demo entry
  paths and demo assets after deployment.
- The Pages workflow runs the deployed routing check after GitHub Pages
  deployment.
- `make db-checks` runs migrations plus database-backed integration and HTTP E2E
  tests when `DATABASE_URL` and `SHARECROP_MIGRATIONS_DIR` are set.
- Admin default-collectible award and collectible transfer flows use
  user/team/organization selectors where selector data exists.
- Browser task and discovery lists have explicit pagination controls.
- User submission history lists support `limit`/`offset` and browser
  previous/next controls.
- Browser ledger, organization ledger, and notification inbox lists use explicit
  `limit`/`offset` requests with previous/next controls. Demo backend paths
  honor the same pagination for those routes.
- Shared scenario parity covers adjacent one-row pages for personal ledger,
  organization ledger, and notification inbox routes.
- Task creation and submission creation support up to five small attachments
  under 500 KiB each for PNG, JPEG, GIF, WebP, plain text, JSON, and PDF.
  Attachment bytes are stored inline for this small-file path and returned as
  data URLs with metadata.
- DB-backed browser coverage verifies task attachment happy-path upload plus
  rejected type, oversized file, and five-file limit guardrails through the real
  UI.
- Inbox notification rows link to the task when notification metadata includes
  `task_id`.
- Submission comments notify the other side of the private submission discussion
  thread. Demo backend paths enforce the same submitter/reviewer thread
  visibility check.
- Collectibles carry an optional organization scope.
  `transferable_within_organization` tips require both users to be active
  members of the scoped organization.
- Submission acceptance settles credit payout, credit tip, collectible payout,
  and collectible tip in one ledger transaction.
- Series add-task management uses the loaded task selector instead of a raw
  task-ID text field.
- Standalone teams can be selected as task assignees, and standalone-team
  reservations require team membership.
- Lifecycle and redaction semantics are documented in
  [docs/deletion_semantics.md](./docs/deletion_semantics.md); core-row removal
  is not part of the project direction.
- The WASM backend target is documented with explicit storage-adapter gates,
  local compile-check results, bundle-size observations, the `internal/wasmdemo`
  request-adapter package, explicit user, account-token, platform-admin,
  audit-event, collectible, agent-credential, privacy-request,
  moderation-triage, saved-queue-view, task, attachment, notification,
  organization, organization-member, team, comment, reservation, submission, and
  ledger browser-storage boundaries and request handlers, and no fallback path.
- Go/WASM is a first-class backend execution target, not only a demo mechanism.
  The target artifact is a `.wasm` binary compiled from Go with explicit host
  adapters for storage, clock, identity/session, and request handling. No
  currently implemented route needs a randomness or networking adapter, so
  neither exists yet in the `HostRuntime` interface; one would be added, and
  fail loudly until added, if a future route needed it. JavaScript
  reimplementations, generated fake backends, and fallback stores are out of
  scope.
- `tools/wasm_runtime_loader.ts` documents and implements the reference
  non-browser `HostFunctions` host (in-memory storage, fixed clock, sequential
  IDs), used by `check:scenario-parity:wasm` and `measure:wasm`.
  `docs/wasm_demo_backend_spike.md` records what a production non-browser host
  still needs beyond that reference: persistent storage, a real clock,
  verified-session actor resolution, and cryptographically random IDs/secrets.
- `deno task measure:wasm -- --wasm <compiled.wasm>` reports artifact size,
  startup time, host-process memory, and per-route request latency for a
  compiled `cmd/sharecrop-wasm` artifact through the non-browser reference host.
  `docs/wasm_demo_backend_spike.md` records a baseline run.
- `cmd/sharecrop-wasm` builds a Go `js/wasm` artifact that exposes
  `sharecropWasmBackendStatus`, `sharecropConfigureHost`, and
  `sharecropHandleRequest`. Requests fail loudly until an explicit host is
  configured. A configured host executes auth, account, user, admin,
  collectible, agent-credential, privacy, moderation, saved-view, task, comment,
  reservation, submission, notification, organization/team, and ledger requests
  through Go handlers and caller-provided storage, clock, actor, and ID
  adapters.
- `deno task check:scenario-parity:wasm -- --wasm <compiled.wasm>` loads the
  compiled Go WASM artifact through Go's `wasm_exec.js`, verifies the
  unconfigured request failure, configures an explicit host, and runs the shared
  scenario parity suite without calling `site/demo/backend.js`.
- `site/demo/index.html` defaults to the compiled Go/WASM backend. It requires
  `wasm_exec.js` and `sharecrop-wasm-backend.wasm`, configures explicit browser
  host functions, seeds deterministic demo data, and intercepts `/api/*` XHR
  requests through `sharecropHandleRequest`. The legacy `backend.js` path is not
  loaded as a fallback.
- `deno task wasm:demo:build` builds the deployed demo WASM artifacts. The Pages
  workflow runs it before uploading `site`.
- The current raw-ID browser-flow audit is recorded in
  [docs/raw_id_browser_flow_audit.md](./docs/raw_id_browser_flow_audit.md).
- Reward scope is Sharecrop credits plus admin-minted Sharecrop collectibles
  only; user/org/per-project tokens, external wallets, and crypto integrations
  are out of scope.
- README and hosted docs link to the repository HTTP API reference, MCP
  reference, operator runbook, and agent-side scheduling recipe.
- README and hosted docs link to the onboarding guide in
  [docs/onboarding.md](./docs/onboarding.md).
- `go run ./cmd/sharecrop generate openapi` (`make openapi`) parses
  `internal/http/server.go`'s mux registrations and the `internal/http`
  package's local call graph with `go/ast` and writes `docs/openapi.json`, an
  OpenAPI 3.0 document with an accurate method/path/operationId inventory and
  bearer-auth requirement per route (a route is marked public only if its
  handler cannot reach `requireUserSubject`/`requireWorkerSubject`/
  `requireAdminSubject`/`requireOrganizationBilling`/`verifyAgent` through the
  call graph). `make check-openapi` regenerates and asserts no diff, mirroring
  `check-contracts`, and runs in PR CI (`.github/workflows/ci.yml`).
- Request/response body schemas in `docs/openapi.json` are typed JSON Schema,
  not generic placeholders, wherever `internal/openapi` can resolve the Go DTO
  struct a handler decodes/writes: it finds the `var request <Type>` +
  `.Decode(&request)` pair for requests, and either a dedicated
  `write<Foo>Response(w, status, response <Type>)` wrapper or the argument to a
  direct `writeJSON(w, status, value)` call for responses (a composite literal,
  a local variable, or a call to a single-return-value converter function),
  following the local call graph transitively so handlers that delegate to a
  shared helper (`openTask`/`cancelTask` via `changeTaskState`) still resolve.
  It also resolves a `name, ok := x.(Type)` two-value type assertion and a
  field-access expression (`response.value`), including through internal
  result-union structs that carry no JSON tags at all. 99/106 responses and
  39/61 request bodies resolve; the rest keep the generic `{"type": "object"}`
  (or empty) placeholder rather than a guess.
- `site/docs/openapi.html` shows a "Schema" column (typed vs. plain) per route
  and a typed-response-schema count in its summary, reading the same generated
  document.
- `site/docs/openapi.html` is a self-contained (no third-party viewer, no build
  step) static page on the deployed GitHub Pages site that fetches
  `site/docs/openapi.json` and renders a route table with method/path/
  operationId/auth-requirement columns and a public/protected summary count.
  `site/docs/openapi.json` is a committed copy of `docs/openapi.json`
  (`deno task site:openapi:copy`, run by `make openapi`/`make check-openapi`).
  `site/docs/index.html`'s References list links to the new page.
  `tools/check_pages_routing.ts` verifies `/docs/openapi.html` and
  `/docs/openapi.json` after deployment.
- `site/marketing.css` styles the three static pages outside the compiled Elm
  app (`site/index.html`, `site/docs/index.html`, `site/docs/openapi.html`) with
  a hand-authored "dispatch desk" paper/typewriter aesthetic (Special Elite/IBM
  Plex fonts loaded from Google Fonts, matching the demo's existing font-loading
  pattern). Previously these two pages linked a stylesheet no build step
  produced and used classes with no matching CSS anywhere, so they rendered
  fully unstyled in production; both are now fixed.
- Local test/development examples avoid the project's former common ports:
  Postgres uses `25432`, the app uses `29180`, and the backendless demo uses
  `29181`. Playwright config accepts environment overrides for those ports.

Current verification:

- Found the `internal/wasmdemo` task-series gap by actually running the compiled
  Go/WASM demo in a real Chromium browser: opened the "Task series" page, filled
  in a title/description, clicked "Create series", and read the rendered "The
  request failed with status 404." error and its network trace (patched
  `XMLHttpRequest`/`sharecropHandleRequest` to see the real request/response
  pairs, since the WASM demo intercepts XHR entirely in JS and never touches the
  real network stack, so Playwright's `page.on("response")` sees nothing). After
  adding the route, hit a second, different decode error ("Expecting an OBJECT
  with a field named `series`") the same way, which revealed that
  `createTaskSeries` returns the full detail wrapper, not a bare series object —
  confirmed by reading `internal/http/series.go`'s `writeSeriesMutation`.
- `go build ./...`, `go vet ./...`, and `go test ./...` passed, including a new
  `internal/wasmdemo` regression test covering create/list/detail.
  `GOOS=js GOARCH=wasm go build ./cmd/sharecrop-wasm/...` passed.
- `deno task check:ts`, `deno task lint`, `deno task check:policy`, and
  `deno task test` passed.
- `make check-format`, `make check-contracts`, `go tool deadcode -test ./...`,
  and `make check-copy-paste` (zero clones) passed; `git diff --check` passed.
- Ran the full Postgres-backed suite:
  `go test -tags integration
  ./tests/integration`,
  `go test -tags http_e2e ./tests/http_e2e`, and `make e2e-ui` (all 46
  Playwright specs against the real API, including the existing real-backend
  series lifecycle test, unaffected). Added a new `demo.spec.ts` test
  (`demo creates and opens a task series`) and confirmed publishing a freshly
  created series fails with a graceful inline error rather than a crash (the
  documented remaining gap).

Blocking issues:

- None.
