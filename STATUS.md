# Status

The repository contains pull request 1 through pull request 73 work, merged into `main`.

Active task: combined runtime/audit/team-dashboard branch is implemented and under verification. Email/provider delivery and anonymous worker identity are out of scope.

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
- Backendless demo account lifecycle, user directory, organization/team member provisioning, and create-time collectible rewards mirror the current real-app flows closely enough for browser demo coverage.
- Admin operations status is available to platform admins at `/api/admin/operations`.
- Production `serve` wires Postgres-backed rate-limit buckets and persisted MCP HTTP session identity; MCP live SSE stream buffers remain process-local and operations reports `postgres_session_process_stream`.
- Platform admins can view audit events at `#/admin`; audit writes cover admin default collectible awards, account deactivation, organization member provisioning/role/deactivation actions, submission review outcomes, and task refunds.
- Team detail pages load a team review queue and team work list from `/api/teams/{team_id}/work`.
- The backendless demo serves the current compiled Elm bundle, includes the admin operations/audit route, and handles `/demo/` base paths explicitly.
- Account verification/reset token issue supports API-visible local/test mode and log-delivery mode.
- Account deactivation anonymizes email, removes password credentials, and revokes active refresh/account tokens.
- User directory selectors can query `/api/users?query=...` from the browser.
- Reward scope is Sharecrop credits plus admin-minted Sharecrop collectibles only; user/org/per-project tokens, external wallets, and crypto integrations are out of scope.

Current verification:

- This branch passed `go test ./...`, `make check-format`, `make check-contracts`, `deno task check:policy`, `deno check tools/*.ts tests/**/*.ts`, `deno task lint`, `deno test --allow-read tests/deno`, `go vet ./...`, `go tool deadcode -test ./...`, `ELM_BIN=/opt/homebrew/bin/elm make build`, full Playwright UI E2E, local screenshot/overflow checks for admin and team pages, integration tests with local Postgres, and HTTP E2E with local Postgres.

Blocking issues:

- None known.
