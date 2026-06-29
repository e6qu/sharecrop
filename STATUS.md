# Status

The repository contains pull request 1 through pull request 79 work, merged into `main`.

Active task: continuity cleanup, database-backed check confirmation, expanded scenario parity, fixture coverage, raw-ID audit, backendless demo semantic parity, and submission-discussion polish are implemented on `task/parity-contract-discussion-polish`. Email/provider delivery, anonymous worker identity, per-project tokens, external wallets, and crypto integrations are out of scope.

Current implemented surface:

- Organization member provisioning has role selection; organization members can have roles updated or be deactivated through the browser and API.
- Organization reviewers can see review controls on organization-owned tasks they did not create.
- Workers see task-local own submissions with state, review notes, validation errors, response body, and submission comments.
- Organization-team reservation, organization funding, organization visibility, and organization award-recipient flows use paginated/typeahead selectors where data is available.
- Task creation has reward-kind selection for no reward, credits, collectible, and bundle rewards.
- Hosted docs and readiness/user-story docs no longer describe `/docs/` as a placeholder.
- Account lifecycle exists for guest entry, email-verification token issue/confirm, password reset/change, profile email update, and account deactivation.
- Authenticated user directory and selector-backed user/team/organization controls in task creation use query and pagination where data is available.
- Collectible and bundle task creation can escrow selected collectibles immediately and task responses show the held collectible count.
- Backendless demo routes now cover current real API routes except the documented real-only health/MCP/root routes.
- Users have a persisted notification inbox for submission-created and submission-review events. The browser has an Inbox page with mark-read support, and the backendless demo mirrors the routes and seeded unread state.
- Backendless demo account lifecycle, user directory, organization/team member provisioning, selector pagination/query, create-time collectible rewards, token-aware actor flows, and shared scenario parity flows mirror the current real-app flows closely enough for browser demo coverage.
- Admin operations status is available to platform admins at `/api/admin/operations`.
- Production `serve` wires Postgres-backed rate-limit buckets, audit events, notification inbox rows, persisted MCP HTTP session identity, and persisted MCP HTTP replay events. MCP live SSE subscribers remain process-local.
- Platform admins can view audit events at `#/admin`; audit writes cover admin default collectible awards, account deactivation, organization member provisioning/role/deactivation actions, submission review outcomes, and task refunds.
- Team detail pages load a team review queue and team work list from `/api/teams/{team_id}/work`.
- The backendless demo serves the current compiled Elm bundle, includes the admin operations/audit route, and handles `/demo/` base paths explicitly.
- Account verification/reset token issue supports API-visible local/test mode and log-delivery mode.
- Account deactivation anonymizes email, removes password credentials, and revokes active refresh/account tokens.
- Selector APIs support `query`, `limit`, and `offset` for users, organizations, standalone teams, and organization teams where those lists are exposed.
- A shared scenario parity runner covers selector pagination/query, admin operations, account-token issue shape, collectible catalog/mint/transfer, organization/team/task/task-comment creation, submission creation/comments, notification read shape, and a multi-actor reservation approval/submission acceptance/payout/notification flow against the backendless demo. It can be run against a real API with an explicit admin origin/token.
- The shared scenario parity runner also covers organization reviewer acceptance of an organization-owned task funded from the organization balance.
- A GitHub Pages routing check script verifies deployed root/docs/demo entry paths and demo assets after deployment.
- The Pages workflow runs the deployed routing check after GitHub Pages deployment.
- `make db-checks` runs migrations plus database-backed integration and HTTP E2E tests when `DATABASE_URL` and `SHARECROP_MIGRATIONS_DIR` are set.
- Admin default-collectible award and collectible transfer flows use user/team/organization selectors where selector data exists.
- Collectibles carry an optional organization scope. `transferable_within_organization` tips require both users to be active members of the scoped organization.
- Submission acceptance settles credit payout, credit tip, collectible payout, and collectible tip in one ledger transaction.
- Series add-task management uses the loaded task selector instead of a raw task-ID text field.
- Deletion semantics are documented in [docs/deletion_semantics.md](./docs/deletion_semantics.md); core rows are not hard-deleted without an explicit lifecycle design.
- The WASM demo backend spike is documented with explicit storage-adapter gates and no fallback path.
- Reward scope is Sharecrop credits plus admin-minted Sharecrop collectibles only; user/org/per-project tokens, external wallets, and crypto integrations are out of scope.

Current verification:

- `go test ./...` passed.
- `go test ./internal/http ./internal/db ./internal/assets ./internal/ledger` passed.
- `deno check tools/*.ts tests/**/*.ts` passed.
- `deno lint tools tests` passed.
- `deno run --allow-read tools/check_policy.ts` passed.
- `deno test --allow-read tests/deno` passed.
- `make check-format` passed.
- `go vet ./...` passed.
- `go tool deadcode -test ./...` passed.
- `deno run -A npm:jscpd@5.0.11 site/demo internal cmd tools web/elm/src tests` passed.
- `ELM_BIN=/opt/homebrew/bin/elm deno task frontend:build` passed.
- PR 79 CI passed, including `db-checks`.
- `make check-contracts` passed after PR 79 was merged.
- A focused Playwright submission-discussion check was attempted locally. The sandbox blocked local server binding, and escalation was unavailable because the environment usage limit was reached. The updated Playwright assertions should run in PR CI.

Blocking issues:

- None known.
