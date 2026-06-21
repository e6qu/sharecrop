# Status

The repository contained the pull request 1 project skeleton, pull request 2 core domain and quality-gate work, and pull request 3 authentication work.

Pull request 4 was active.

Active task:

- Add organizations, teams, and provisioning.
- Add organization and team domain types.
- Add organization role and permission domain types.
- Add PostgreSQL organization/team/provisioning tables and repository code.
- Add HTTP endpoints for organization creation, listing, members, and teams.
- Re-evaluate tests for the organization provisioning slice.

Implemented surface:

- Go module `github.com/e6qu/sharecrop`.
- Go binary entry point at `cmd/sharecrop`.
- `net/http` server with `/healthz` and embedded static app shell.
- Config loading for HTTP address, `DATABASE_URL`, and migrations directory.
- PostgreSQL connection boundary using `pgx`.
- Plain SQL migration runner with `sharecrop migrate up`.
- Initial migration file.
- Docker Compose Postgres service on local port `15432`.
- Elm app shell.
- Tailwind build through Deno-managed tooling.
- Deno smoke test harness.
- Go HTTP unit tests.
- HTTP end-to-end smoke tests behind the `http_e2e` build tag.
- Playwright user interface smoke test.
- Manual screenshot helper.
- `make` commands for build, test, serve, migration, frontend, and user interface end-to-end.
- Core domain foundations for errors, IDs, lifecycle states, and visibility scopes.
- continuous integration workflow for pull requests targeting `main`, covering formatting, type checks, policy checks, copy-paste detection, dead-code detection, linting, vet, tests, frontend build, binary build, migrations, HTTP end-to-end, and user interface end-to-end.
- Explicit configuration loading without fallback values.
- Authentication domain and boundary code under `internal/auth`.
- Registered user creation and login endpoints.
- Guest subject creation endpoint.
- Refresh endpoint using opaque rotating refresh-token cookies.
- JSON Web Token access tokens signed by local standard-library code.
- PostgreSQL authentication tables and repository code.
- Organization and team identifiers.
- Organization names, team names, membership statuses, roles, and permissions.
- Organization service methods for creation, listing, member provisioning, member deactivation, team creation, and team listing.
- PostgreSQL organization, membership, role, team, and team-member tables.
- HTTP organization endpoints protected by verified bearer access tokens.

The accepted defaults for pull request 1 were:

- Go module path: `github.com/e6qu/sharecrop`.
- Local Postgres through Docker Compose.
- App config through `DATABASE_URL`.
- `make` as the task runner.
- Deno as the frontend tool runner, not npm.
- Elm and Tailwind invoked through Deno-managed tooling or pinned local tooling without npm.
- One resettable test database per test run at first.
- Initial migration command: `sharecrop migrate up`.
- Default app port: `18080`.
- Default local Postgres port: `15432`.

Last observed checks:

- `make check-format` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `make check-ts` passed.
- `make lint` passed.
- `GOCACHE=$PWD/.cache/go-build make vet` passed.
- `GOCACHE=$PWD/.cache/go-build make test` passed.
- `GOCACHE=$PWD/.cache/go-build make check-dead-code` passed.
- `ELM_BIN=/opt/homebrew/bin/elm make frontend` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod make build` passed.
- `make test-http` passed with local listener permission.
- `deno task e2e:ui` passed with local browser permission.
- Manual screenshot review passed for the app shell after pull request 2 changes.
- `docker compose up -d postgres` passed.
- `make migrate-up` passed against local Postgres.
- `docker compose down` passed.
- Pull request 2 continuous integration passed before merge.
- Pull request 3 continuous integration passed before merge.
- Pull request 4 local unit, formatting, type, lint, policy, copy-paste, dead-code, build, migration, and HTTP end-to-end checks passed.

See [PLAN.md](./PLAN.md) for the product and architecture plan.
See [DO_NEXT.md](./DO_NEXT.md) for the next tasks.
See [BUGS.md](./BUGS.md) for known bugs and risks.
