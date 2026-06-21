# Status

The repository contained the PR 1 project skeleton.

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
- HTTP E2E smoke tests behind the `http_e2e` build tag.
- Playwright UI smoke test.
- Manual screenshot helper.
- `make` commands for build, test, serve, migration, frontend, and UI E2E.

The accepted defaults for PR 1 were:

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

- `GOCACHE=$PWD/.cache/go-build go test ./...` passed.
- `deno task test` passed.
- `deno task frontend:build` passed.
- `make build` passed earlier in PR 1.
- `deno task e2e:ui` passed earlier in PR 1.
- Manual screenshot review passed for the app shell.

See [PLAN.md](./PLAN.md) for the product and architecture plan.
See [DO_NEXT.md](./DO_NEXT.md) for the next tasks.
See [BUGS.md](./BUGS.md) for known bugs and risks.
