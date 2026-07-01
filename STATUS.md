# Status

The repository contains pull request 1 through pull request 97 work, merged into
`main`, plus the current
`task/wasm-host-adapters-scenario-parity` branch.

Active task: `task/wasm-host-adapters-scenario-parity` wires explicit host
configuration into the Go `js/wasm` command, runs request execution through the
configured WASM host for the existing task/comment/reservation/submission/ledger
slices, updates the WASM scenario runner to exercise that path, records remaining
WASM behavior slices, and includes deployed Pages routing verification. The
branch must keep fail-loud behavior and must not add fallback stores or
JavaScript backend reimplementations as the target.
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
- Backendless demo routes now cover current real API routes except the
  documented real-only health/MCP/root routes.
- Users have a persisted notification inbox for submission-created and
  submission-review events. The browser has an Inbox page with mark-read
  support, and the backendless demo mirrors the routes and seeded unread state.
- Backendless demo account lifecycle, user directory, organization/team member
  provisioning, selector pagination/query, create-time collectible rewards,
  token-aware actor flows, and shared scenario parity flows mirror the current
  real-app flows closely enough for browser demo coverage.
- Admin operations status is available to platform admins at
  `/api/admin/operations`.
- Authenticated users can report tasks through the task detail page. Reports
  are persisted as `moderation_report_created` audit events and platform admins
  can list them in the Admin moderation panel and through
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
- Team work and organization task queues have persisted saved views for
  reusable query/filter/sort combinations.
- Organization detail pages expose an operations dashboard with loaded balance,
  ledger rows, org-scoped audit rows, member, team, collectible, and task-state
  counts.
- Admin audit event listing supports action, subject-kind, subject-id, and page
  filters through the API and browser controls.
- Admin audit, platform-admin, privacy-request, and moderation-report lists
  have explicit browser pagination controls.
- Platform admins are stored through explicit runtime services. Bootstrap
  admins come from `SHARECROP_ADMIN_USER_IDS`; admin-granted platform admins are
  persisted and revoked through a lifecycle state instead of row deletion.
- The admin page includes platform-admin configuration backed by the paginated
  user selector, privacy retention execution, moderation state filtering,
  direct moderation subject links where routes exist, and moderation triage
  actions for open/resolved/dismissed states with notes.
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
  active delete-on-request sensitive-field metadata, records per-field
  redaction events, records the retention run, and writes a privacy retention
  audit event.
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
- The backendless demo serves the current compiled Elm bundle, includes the
  admin operations/audit route, and handles `/demo/` base paths explicitly.
- Account verification/reset token issue supports API-visible local/test mode
  and log-delivery mode.
- Account deactivation anonymizes email, removes password credentials, and
  revokes active refresh/account tokens.
- Selector APIs support `query`, `limit`, and `offset` for users, organizations,
  standalone teams, and organization teams where those lists are exposed.
- A shared scenario parity runner covers selector pagination/query, admin
  operations, account-token issue shape, privacy request/audit/resolution shape,
  moderation report/admin-list/audit shape, collectible catalog/mint/transfer,
  organization/team/task/task-comment creation, submission creation/comments,
  notification read shape, and a multi-actor reservation
  approval/submission acceptance/payout/notification flow against the
  backendless demo. It can be run against a real API with an explicit admin
  origin/token/refresh-token session.
- The shared scenario parity runner also covers organization reviewer acceptance
  of an organization-owned task funded from the organization balance.
- The shared scenario parity runner covers submission-comment notifications,
  team/organization queue search/type/sort behavior, persisted saved queue
  views, small task/submission attachments, and sensitive-field response
  metadata.
- The shared scenario parity runner covers organization member
  provisioning/listing/role-update/deactivation shape.
- The real API shared scenario parity runner probes `/healthz`, accepts
  token and refresh-token file inputs, carries refresh-cookie rotation, and
  reports invalid JSON and status errors with request context before running the
  shared scenario suite.
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
- Browser ledger, organization ledger, and notification inbox lists use
  explicit `limit`/`offset` requests with previous/next controls. The
  backendless demo honors the same pagination for those routes.
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
  thread. The backendless demo enforces the same submitter/reviewer thread
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
- The WASM demo backend spike is documented with explicit storage-adapter gates,
  local compile-check results, bundle-size observations, a narrow
  `internal/wasmdemo` request-adapter package, explicit privacy-request,
  moderation-triage, saved-queue-view, task, attachment, notification,
  organization, organization-member, team, comment, reservation, submission,
  and ledger browser-storage boundaries,
  narrow privacy-request, moderation-triage, saved-queue-view, task,
  notification, organization, organization-member, team, comment, reservation,
  submission, and ledger request handlers, and no fallback path.
- Go/WASM is a first-class backend execution target, not only a demo mechanism.
  The target artifact is a `.wasm` binary compiled from Go with explicit host
  adapters for storage, clock, identity/session, request handling, randomness,
  and networking. JavaScript reimplementations, generated fake backends, and
  fallback stores are out of scope.
- `cmd/sharecrop-wasm` builds a Go `js/wasm` artifact that exposes
  `sharecropWasmBackendStatus`, `sharecropConfigureHost`, and
  `sharecropHandleRequest`. Requests fail loudly until an explicit host is
  configured. A configured host executes task/comment/reservation/submission and
  ledger requests through Go handlers and caller-provided storage, clock, actor,
  and ID adapters.
- `deno task check:scenario-parity:wasm -- --wasm <compiled.wasm>` loads the
  compiled Go WASM artifact through Go's `wasm_exec.js`, verifies the
  unconfigured request failure, configures an explicit host, and runs a
  task/comment/reservation/submission acceptance and ledger scenario without
  calling `site/demo/backend.js`.
- The current raw-ID browser-flow audit is recorded in
  [docs/raw_id_browser_flow_audit.md](./docs/raw_id_browser_flow_audit.md).
- Reward scope is Sharecrop credits plus admin-minted Sharecrop collectibles
  only; user/org/per-project tokens, external wallets, and crypto integrations
  are out of scope.
- README and hosted docs link to the repository HTTP API reference, MCP
  reference, operator runbook, and agent-side scheduling recipe.
- README and hosted docs link to the onboarding guide in
  [docs/onboarding.md](./docs/onboarding.md).
- Local test/development examples avoid the project's former common ports:
  Postgres uses `25432`, the app uses `29180`, and the backendless demo uses
  `29181`. Playwright config accepts environment overrides for those ports.

Current verification:

- `go test ./...` passed.
- `deno task check:ts` passed.
- `deno task lint` passed.
- `deno task check:policy` passed.
- `deno task test` passed.
- `deno fmt --check deno.json tools tests site/demo/backend.js` passed.
- `make check-contracts` passed.
- `go tool deadcode -test ./...` passed.
- `GOOS=js GOARCH=wasm go build -o
  /private/tmp/sharecrop-wasm-backend.wasm ./cmd/sharecrop-wasm` passed.
- `deno task check:scenario-parity:wasm -- --wasm
  /private/tmp/sharecrop-wasm-backend.wasm` passed.
- `deno task check:pages-routing -- --origin https://e6qu.github.io/sharecrop`
  passed.
- `git diff --check` passed.

Blocking issues:

- None.
