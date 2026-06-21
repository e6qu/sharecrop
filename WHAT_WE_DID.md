# What We Did

The project plan was written in [PLAN.md](./PLAN.md).

The agent workflow was documented in [AGENTS.md](./AGENTS.md).

The Claude pointer file was added in [CLAUDE.md](./CLAUDE.md).

The continuity-file policy was clarified:

- Continuity files were set to update before and after each task.
- [STATUS.md](./STATUS.md) was set to summarize implementation status precisely and factually.
- [WHAT_WE_DID.md](./WHAT_WE_DID.md) was set to remain append-oriented while allowing old or irrelevant parts to be compressed.
- [DO_NEXT.md](./DO_NEXT.md) was set to hold a prioritized queue.
- [BUGS.md](./BUGS.md) was set to include confirmed defects, test gaps, and open risks.
- PR descriptions were set to be precise and timeless without reproducing code.

The remaining agent-practice questions were resolved:

- [STATUS.md](./STATUS.md) was set to stay short and cover current implemented surface, test status, active task, and blocking issues.
- Continuity updates were set to happen in the same task branch, with the final after-task update near the end of the branch.

The task workflow was updated to use one commit per task, with code, tests, and continuity-file updates included in that task commit.

Testing was set to happen throughout each task and again before finishing the task.

The task workflow was updated again so agents create one git commit at the end of each task by default.

The task workflow was updated so each task uses its own task branch and pull request.

The pull request workflow was constrained to one open pull request at a time.

New task branches were set to start from synced `origin/main` after the previous task pull request is merged.

UI changes were set to require manual screenshot review when practical.

Playwright UI tests were set to grow as the UI matures and workflows stabilize.

The project repository and PR 1 implementation defaults were recorded:

- GitHub project URL: `https://github.com/e6qu/sharecrop`.
- Canonical SSH remote: `git@github.com:e6qu/sharecrop.git`.
- Go module path: `github.com/e6qu/sharecrop`.
- Local development was set to use Docker Compose for Postgres.
- App config was set to use `DATABASE_URL`.
- The task runner was set to `make`.
- The frontend tool runner was changed to Deno.
- npm was excluded from the frontend toolchain.
- Elm and Tailwind were set to run through Deno-managed tooling or pinned local tooling without npm.
- The first test database strategy was set to one resettable test database per test run.
- The first migration command was set to `sharecrop migrate up`.
- The default app port was set to `18080`.
- The default local Postgres port was set to `15432`.
- Common development ports such as `3000`, `5432`, `8000`, and `8080` were avoided.

The MCP implementation direction was changed:

- No Go MCP library was selected.
- MCP protocol handling was set to be implemented locally from the official MCP specification.

Vitest was considered and not selected.

Deno's built-in test runner was selected for Deno tooling unless a TypeScript/Vite layer is introduced later.

PR 1 added the project skeleton and build system:

- The Go module `github.com/e6qu/sharecrop` was created.
- The `cmd/sharecrop` binary entry point was added.
- A `net/http` server was added with `/healthz` and an embedded static app shell.
- Config loading was added for HTTP address, `DATABASE_URL`, and migrations directory.
- PostgreSQL access was isolated in `internal/db` with `pgx`.
- A plain SQL migration runner was added with `sharecrop migrate up`.
- An initial migration file was added.
- Docker Compose configuration was added for Postgres on local port `15432`.
- An Elm app shell was added.
- Tailwind was wired through Deno-managed tooling.
- Deno smoke tests were added.
- Go HTTP unit tests were added.
- HTTP E2E smoke tests were added behind the `http_e2e` build tag.
- Playwright UI smoke tests were added.
- A manual screenshot helper was added.
- `make` commands were added for build, test, serve, migration, frontend, and UI E2E.
- Generated local artifacts were excluded through `.gitignore`.

PR 1 verification was performed:

- `GOCACHE=$PWD/.cache/go-build go test ./...` passed.
- `deno task test` passed.
- `deno task frontend:build` passed.
- `make build` passed.
- `deno task e2e:ui` passed earlier in the task.
- Manual screenshot review showed the app shell rendering the Sharecrop heading and skeleton text.

PR 1 verification gaps were recorded:

- Docker Compose Postgres startup was not verified because the environment rejected Docker approval.
- `sharecrop migrate up` against live Postgres was not verified for the same reason.
- Final rerun of `deno task e2e:ui` was not performed because local-network/browser permissions had already been exhausted in this environment after an earlier successful run.
- `make build` with both `GOCACHE` and `GOMODCACHE` isolated inside the workspace could not fetch `pgx` because network access was restricted. The build had passed earlier with the existing module cache.

PR 2 added core domain foundations and CI quality gates:

- Core domain errors were added.
- Strong ID wrappers were added for users, tasks, and organizations.
- UUIDv7 generation and parsing were isolated behind `internal/core/id`.
- Lifecycle state parsing was added.
- Visibility scope variants and parsing were added.
- Per-type result variants were used instead of generic result types.
- CI was added for formatting, TypeScript checks, policy checks, copy-paste detection, dead-code detection, Deno linting, Go vet, unit tests, frontend build, binary build, migrations, HTTP E2E, and UI E2E.
- CI was limited to pull requests targeting `main`, without direct `main` push runs or bare branch push runs.
- The Elm build tool was changed to require explicit `ELM_BIN`.
- Config loading was changed to require explicit environment variables instead of fallback values.
- Docker Compose was fixed for PostgreSQL 18 by mounting the volume at `/var/lib/postgresql`.

PR 2 verification was performed:

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
- Manual screenshot review showed the app shell rendering without visible layout issues after PR 2 changes.
- `docker compose up -d postgres` passed.
- `make migrate-up` passed against local Postgres.
- `docker compose down` passed.

PR 2 verification gaps were recorded:

- Aggregate `make ci` was not run locally because the environment approval request timed out twice.
