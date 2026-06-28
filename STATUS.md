# Status

The repository contains pull request 1 through pull request 71 work, merged into `main`.

Active task: demo/real parity branch is ready for review. The branch contract-tests the backendless demo route surface and representative response shapes against the real API/client expectations, adds shared browser scenario fixtures, and fixes demo parity gaps for account lifecycle, user directory, and create-time collectible rewards.

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

Current verification:

- PR #71 verification passed before merge: `go test ./...`, HTTP E2E with local Postgres, static checks, build, and 36 Playwright tests.
- Current branch verification passed: `make check-format check-contracts check-policy check-ts lint vet test-deno frontend build`, `go tool deadcode -test ./...`, `go test ./...`, HTTP E2E with local Postgres, and targeted Playwright specs for demo, account lifecycle, and selector-backed task creation.

Blocking issues:

- None known.
