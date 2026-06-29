# Status

The repository contains pull request 1 through pull request 74 work, plus the current runtime notifications and demo-parity branch.

Active task: runtime notifications, persisted MCP event replay, runtime-store coverage, generated notification contracts, inbox UI/demo parity, and demo semantic-parity planning. Email/provider delivery, anonymous worker identity, per-project tokens, external wallets, and crypto integrations are out of scope.

Current implemented surface:

- Organization member provisioning has role selection; organization members can have roles updated or be deactivated through the browser and API.
- Organization reviewers can see review controls on organization-owned tasks they did not create.
- Workers see task-local own submissions with state, review notes, validation errors, response body, and submission comments.
- Organization-team reservation, organization funding, organization visibility, and organization award-recipient flows use selectors where data is available.
- Task creation has reward-kind selection for no reward, credits, collectible, and bundle rewards.
- Hosted docs and readiness/user-story docs no longer describe `/docs/` as a placeholder.
- Account lifecycle exists for guest entry, email-verification token issue/confirm, password reset/change, profile email update, and account deactivation.
- Authenticated user directory and selector-backed user/team controls are available in task creation where the app has loaded directory data.
- Collectible and bundle task creation can escrow selected collectibles immediately and task responses show the held collectible count.
- Backendless demo routes now cover current real API routes except the documented real-only health/MCP/root routes.
- Users have a persisted notification inbox for submission-created and submission-review events. The browser has an Inbox page with mark-read support, and the backendless demo mirrors the routes and seeded unread state.
- Backendless demo account lifecycle, user directory, organization/team member provisioning, and create-time collectible rewards mirror the current real-app flows closely enough for browser demo coverage.
- Admin operations status is available to platform admins at `/api/admin/operations`.
- Production `serve` wires Postgres-backed rate-limit buckets, audit events, notification inbox rows, persisted MCP HTTP session identity, and persisted MCP HTTP replay events. MCP live SSE subscribers remain process-local.
- Platform admins can view audit events at `#/admin`; audit writes cover admin default collectible awards, account deactivation, organization member provisioning/role/deactivation actions, submission review outcomes, and task refunds.
- Team detail pages load a team review queue and team work list from `/api/teams/{team_id}/work`.
- The backendless demo serves the current compiled Elm bundle, includes the admin operations/audit route, and handles `/demo/` base paths explicitly.
- Account verification/reset token issue supports API-visible local/test mode and log-delivery mode.
- Account deactivation anonymizes email, removes password credentials, and revokes active refresh/account tokens.
- User directory selectors can query `/api/users?query=...` from the browser.
- Reward scope is Sharecrop credits plus admin-minted Sharecrop collectibles only; user/org/per-project tokens, external wallets, and crypto integrations are out of scope.

Current verification:

- Current branch passed: `go test ./...`, `make check-format check-ts lint`, `make check-contracts`, `ELM_BIN=/opt/homebrew/bin/elm deno task frontend:build`, `deno task test`, `make check-policy vet check-dead-code`, `make check-copy-paste`, and targeted Playwright demo/mobile specs.
- Current branch screenshot review: `#/inbox` desktop and mobile screenshots passed visual inspection; Playwright reported no horizontal overflow.
- Current branch integration tests are present but did not run locally because `DATABASE_URL` is not set in this environment.

Blocking issues:

- None known.
