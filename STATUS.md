# Status

The repository contains pull request 1 through pull request 69 work, merged into `main`, plus the active combined branch for organization/browser parity, worker submission UX, selectors, reward creation, and docs.

Active task: final verification, commit, push, and pull request for the combined follow-up branch.

Current implemented surface on the combined branch:

- Organization member provisioning has role selection; organization members can have roles updated or be deactivated through the browser and API.
- Organization reviewers can see review controls on organization-owned tasks they did not create.
- Workers see task-local own submissions with state, review notes, validation errors, response body, and submission comments.
- Organization-team reservation, organization funding, organization visibility, and organization award-recipient flows use selectors where data is available.
- Task creation has reward-kind selection for no reward, credits, collectible, and bundle rewards.
- Hosted docs and readiness/user-story docs no longer describe `/docs/` as a placeholder.

Current verification:

- `go test ./...` passed with `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build`.
- `deno test --allow-read tests/deno` passed.
- `deno check tools/*.ts tests/**/*.ts` passed.
- `deno lint tools tests` passed.
- `deno run --allow-read tools/check_policy.ts` passed.
- `test -z "$(gofmt -l cmd internal tests web | grep -E '\.go$')" && deno fmt --check deno.json tools tests` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build ELM_BIN=/opt/homebrew/bin/elm make frontend` passed.
- Focused Playwright demo smoke with screenshots passed against `http://127.0.0.1:18082`.

Blocking issues:

- Database-backed HTTP E2E and full real-app Playwright were not run locally because `DATABASE_URL` is not set and local server binding is restricted in the sandbox. See [BUGS.md](./BUGS.md).
