# Status

The repository contains pull request 1 through pull request 68 work, merged into `main`.

Active task: `task/org-team-assignment`.

Current branch implements organization-team task assignment:

- Organization-team scoped tasks can be reserved or requested through HTTP, MCP, and the browser by a user who belongs to the selected organization-owned team.
- Submission eligibility accepts a submitter who belongs to the team on an active organization-team reservation.
- The demo backend and generated browser bundles were updated for the same flow.

Current verification:

- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build ELM_BIN=/opt/homebrew/bin/elm make frontend` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build go test -tags http_e2e ./tests/http_e2e -run TestOrganizationTeamReservationAndSubmission` could not run locally because `DATABASE_URL` is not set.

Blocking issues:

- None known for the current branch.
