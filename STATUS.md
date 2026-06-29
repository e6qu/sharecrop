# Status

The repository contains pull request 1 through pull request 75 work, merged into `main`.

Active task: scenario-parity, selector pagination/typeahead, contract-fixture expansion, GitHub Pages routing-check, and WASM demo-backend spike branch is implemented locally and ready for PR. Email/provider delivery, anonymous worker identity, per-project tokens, external wallets, and crypto integrations are out of scope.

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
- Backendless demo account lifecycle, user directory, organization/team member provisioning, selector pagination/query, create-time collectible rewards, and shared scenario parity flow mirror the current real-app flows closely enough for browser demo coverage.
- Admin operations status is available to platform admins at `/api/admin/operations`.
- Production `serve` wires Postgres-backed rate-limit buckets, audit events, notification inbox rows, persisted MCP HTTP session identity, and persisted MCP HTTP replay events. MCP live SSE subscribers remain process-local.
- Platform admins can view audit events at `#/admin`; audit writes cover admin default collectible awards, account deactivation, organization member provisioning/role/deactivation actions, submission review outcomes, and task refunds.
- Team detail pages load a team review queue and team work list from `/api/teams/{team_id}/work`.
- The backendless demo serves the current compiled Elm bundle, includes the admin operations/audit route, and handles `/demo/` base paths explicitly.
- Account verification/reset token issue supports API-visible local/test mode and log-delivery mode.
- Account deactivation anonymizes email, removes password credentials, and revokes active refresh/account tokens.
- Selector APIs support `query`, `limit`, and `offset` for users, organizations, standalone teams, and organization teams where those lists are exposed.
- A shared scenario parity runner covers selector pagination/query plus organization, team, task, and task-comment creation against the backendless demo and can be run against a real API with an explicit origin/token.
- A GitHub Pages routing check script verifies deployed root/docs/demo entry paths and demo assets after deployment.
- The WASM demo backend spike is documented with explicit storage-adapter gates and no fallback path.
- Reward scope is Sharecrop credits plus admin-minted Sharecrop collectibles only; user/org/per-project tokens, external wallets, and crypto integrations are out of scope.

Current verification:

- `go test ./...` passed.
- `go test ./internal/org ./internal/db ./internal/http` passed.
- `deno check tools/*.ts tests/**/*.ts` passed.
- `deno lint tools tests` passed.
- `deno run --allow-read tools/check_policy.ts` passed.
- `deno test --allow-read tests/deno` passed.
- `ELM_BIN=/opt/homebrew/bin/elm deno task frontend:build` passed.
- `deno task check:pages-routing -- --origin http://127.0.0.1:18082` passed against a local `site/` server.
- Manual screenshot review passed for the create-task selector controls on desktop and mobile demo viewports.

Blocking issues:

- None known.
