# Status

The repository contains pull request 1 through pull request 86 work, merged into
`main`, plus the current `task/privacy-ops-demo-wasm-parity` branch.

Active task: deepen privacy lifecycle coverage and admin/operator handling,
expand shared scenario parity and HTTP contract fixtures, polish queue/dashboard
workflows, refresh stale readiness docs, investigate a WASM demo backend path,
and fix demo website issues found during the work. The branch now implements
admin privacy request resolution UI, richer privacy exports, sensitive-field
redaction state/counts/events, expanded parity and contract coverage, saved-view
label polish, demo CSS build copying, and WASM compile-check findings. Hard
deletes remain out of scope; use soft lifecycle states, anonymization,
redaction, tombstones, and audit records.
Email/provider delivery, anonymous worker identity, per-project tokens, external
wallets, and crypto integrations are out of scope.

Current implemented surface:

- Organization member provisioning has role selection; organization members can
  have roles updated or be deactivated through the browser and API.
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
  collectible catalog/mint/transfer, organization/team/task/task-comment
  creation, submission creation/comments, notification read shape, and a
  multi-actor reservation approval/submission acceptance/payout/notification
  flow against the backendless demo. It can be run against a real API with an
  explicit admin origin/token.
- The shared scenario parity runner also covers organization reviewer acceptance
  of an organization-owned task funded from the organization balance.
- The shared scenario parity runner covers submission-comment notifications and
  team/organization queue search/type/sort behavior plus sensitive-field
  response metadata.
- A GitHub Pages routing check script verifies deployed root/docs/demo entry
  paths and demo assets after deployment.
- The Pages workflow runs the deployed routing check after GitHub Pages
  deployment.
- `make db-checks` runs migrations plus database-backed integration and HTTP E2E
  tests when `DATABASE_URL` and `SHARECROP_MIGRATIONS_DIR` are set.
- Admin default-collectible award and collectible transfer flows use
  user/team/organization selectors where selector data exists.
- Browser task and discovery lists have explicit pagination controls.
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
  local compile-check results, bundle-size observations, and no fallback path.
- Reward scope is Sharecrop credits plus admin-minted Sharecrop collectibles
  only; user/org/per-project tokens, external wallets, and crypto integrations
  are out of scope.
- README and hosted docs link to the repository HTTP API reference, MCP
  reference, operator runbook, and agent-side scheduling recipe.
- README and hosted docs link to the onboarding guide in
  [docs/onboarding.md](./docs/onboarding.md).

Current verification:

- `go test ./...` passed.
- `deno check tools/*.ts tests/**/*.ts` passed.
- `deno lint tools tests` passed.
- `deno run --allow-read tools/check_policy.ts` passed.
- `deno test --allow-read tests/deno` passed.
- `make check-format` passed.
- `ELM_BIN=/opt/homebrew/bin/elm deno task frontend:build` passed.
- `go vet ./...` passed.
- `go tool deadcode -test ./...` passed.
- `deno run -A npm:jscpd@5.0.11 site/demo internal cmd tools web/elm/src tests`
  passed.
- `DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable
  SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations
  SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901
  SHARECROP_HTTP_ADDR=:18080 make db-checks` passed.
- `deno run -A npm:@playwright/test@1.61.0 test -c
  tests/playwright/playwright.config.ts --no-deps tests/playwright/demo.spec.ts`
  passed.
- A focused admin privacy screenshot was captured at
  `/private/tmp/sharecrop-admin-privacy.png` and inspected for layout overflow;
  export JSON wrapped inside the admin panel after the code-block and demo CSS
  build-copy fixes.
- `GOOS=js GOARCH=wasm go test -c` compile checks passed for
  `./internal/submission` and `./internal/http`; `GOOS=js GOARCH=wasm go build`
  passed for `./cmd/sharecrop`.
- PR 86 CI passed, including `db-checks` and Playwright.

Blocking issues:

- None known.
