# Status

The repository contains pull request 1 through pull request 70 work, merged into `main`, plus the combined account/directories/rewards/Playwright branch.

Active task: final verification and PR prep for the combined account lifecycle, directory selectors, reward setup, and Playwright coverage branch.

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

Current verification:

- `go test ./...` passed.
- `go test -tags http_e2e ./tests/http_e2e` passed with local Postgres.
- `make check-contracts`, `make check-format`, `make check-policy`, `make check-ts`, `make lint`, `make test-deno`, `make check-dead-code`, and `make vet` passed.
- `make build` passed with `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_BIN=/opt/homebrew/bin/elm`.
- `deno task e2e:ui` passed: 36 Playwright tests.

Blocking issues:

- None known.
