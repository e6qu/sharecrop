# Status

The repository contains pull request 1 through pull request 72 work, merged into `main`.

Active task: combined product-readiness foundation branch is ready for review. It adds operations runbook/deployment schema basics, searchable user directory UI, account-token log delivery mode, broader HTTP contract fixtures, concrete account deactivation/erasure semantics, submission comment form polish, and internal-only reward scope cleanup.

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
- Account verification/reset token issue supports API-visible local/test mode and log-delivery mode.
- Account deactivation anonymizes email, removes password credentials, and revokes active refresh/account tokens.
- User directory selectors can query `/api/users?query=...` from the browser.
- Reward scope is Sharecrop credits plus admin-minted Sharecrop collectibles only; user/org/per-project tokens, external wallets, and crypto integrations are out of scope.

Current verification:

- PR #72 verification passed before merge: `make check-format check-contracts check-policy check-ts lint vet test-deno frontend build`, `go tool deadcode -test ./...`, `go test ./...`, HTTP E2E with local Postgres, and targeted Playwright specs for demo, account lifecycle, and selector-backed task creation.
- Current branch verification passed: `make check-format check-contracts check-policy check-ts lint vet test-deno frontend build`, `go test ./...`, `go tool deadcode -test ./...`, HTTP E2E with local Postgres, targeted Playwright account/selector specs, and clean `make build` with workspace-local Go caches.

Blocking issues:

- None known.
