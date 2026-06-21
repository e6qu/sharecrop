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
- pull request descriptions were set to be precise and timeless without reproducing code.

The remaining agent-practice questions were resolved:

- [STATUS.md](./STATUS.md) was set to stay short and cover current implemented surface, test status, active task, and blocking issues.
- Continuity updates were set to happen in the same task branch, with the final after-task update near the end of the branch.

The task workflow was updated to use one commit per task, with code, tests, and continuity-file updates included in that task commit.

Testing was set to happen throughout each task and again before finishing the task.

The task workflow was updated again so agents create one git commit at the end of each task by default.

The task workflow was updated so each task uses its own task branch and pull request.

The pull request workflow was constrained to one open pull request at a time.

New task branches were set to start from synced `origin/main` after the previous task pull request is merged.

user interface changes were set to require manual screenshot review when practical.

Playwright user interface tests were set to grow as the user interface matures and workflows stabilize.

The project repository and pull request 1 implementation defaults were recorded:

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

pull request 1 added the project skeleton and build system:

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
- HTTP end-to-end smoke tests were added behind the `http_e2e` build tag.
- Playwright user interface smoke tests were added.
- A manual screenshot helper was added.
- `make` commands were added for build, test, serve, migration, frontend, and user interface end-to-end.
- Generated local artifacts were excluded through `.gitignore`.

pull request 1 verification was performed:

- `GOCACHE=$PWD/.cache/go-build go test ./...` passed.
- `deno task test` passed.
- `deno task frontend:build` passed.
- `make build` passed.
- `deno task e2e:ui` passed earlier in the task.
- Manual screenshot review showed the app shell rendering the Sharecrop heading and skeleton text.

pull request 1 verification gaps were recorded:

- Docker Compose Postgres startup was not verified because the environment rejected Docker approval.
- `sharecrop migrate up` against live Postgres was not verified for the same reason.
- Final rerun of `deno task e2e:ui` was not performed because local-network/browser permissions had already been exhausted in this environment after an earlier successful run.
- `make build` with both `GOCACHE` and `GOMODCACHE` isolated inside the workspace could not fetch `pgx` because network access was restricted. The build had passed earlier with the existing module cache.

pull request 2 added core domain foundations and continuous integration quality gates:

- Core domain errors were added.
- Strong ID wrappers were added for users, tasks, and organizations.
- UUIDv7 generation and parsing were isolated behind `internal/core/id`.
- Lifecycle state parsing was added.
- Visibility scope variants and parsing were added.
- Per-type result variants were used instead of generic result types.
- continuous integration was added for formatting, TypeScript checks, policy checks, copy-paste detection, dead-code detection, Deno linting, Go vet, unit tests, frontend build, binary build, migrations, HTTP end-to-end, and user interface end-to-end.
- continuous integration was limited to pull requests targeting `main`, without direct `main` push runs or bare branch push runs.
- The Elm build tool was changed to require explicit `ELM_BIN`.
- Config loading was changed to require explicit environment variables instead of fallback values.
- Docker Compose was fixed for PostgreSQL 18 by mounting the volume at `/var/lib/postgresql`.

pull request 2 verification was performed:

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
- Manual screenshot review showed the app shell rendering without visible layout issues after pull request 2 changes.
- `docker compose up -d postgres` passed.
- `make migrate-up` passed against local Postgres.
- `docker compose down` passed.

pull request 2 verification gaps were recorded:

- Aggregate `make ci` was not run locally because the environment approval request timed out twice.

Pull request 3 added authentication, sessions, and guest identity:

- Guest and refresh-token identifiers were added to the core identifier set.
- Authentication value types were added for email addresses, password secrets, access-token secrets, refresh tokens, subjects, and session results.
- Password hashing was implemented with standard-library PBKDF2 and SHA-256 behind `internal/auth`.
- JSON Web Token access-token signing was implemented with standard-library HMAC SHA-256 behind `internal/auth`.
- Opaque refresh-token generation and hashing were added.
- The authentication service added registered user creation, login, guest subject creation, refresh-token rotation, and refresh-token reuse rejection.
- PostgreSQL tables were added for users, guest subjects, password credentials, and refresh tokens.
- The PostgreSQL authentication repository was added under `internal/db`.
- HTTP endpoints were added for registration, login, refresh, and guest session creation.
- Refresh tokens were returned as HttpOnly cookies.
- Config parsing was split into pure `ParseConfig` and the environment-reading `LoadConfig` shell.
- `SHARECROP_ACCESS_TOKEN_SECRET` was added as an explicit required environment variable.
- Dead-code detection was changed from `go run ...@latest` to a declared Go tool dependency invoked through `go tool deadcode`.

Pull request 3 verification was performed:

- `make check-format` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `make check-ts` passed.
- `make lint` passed.
- `GOCACHE=$PWD/.cache/go-build make vet` passed.
- `GOCACHE=$PWD/.cache/go-build make test` passed.
- `GOCACHE=$PWD/.cache/go-build make check-dead-code` passed.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build go test -run '^$' -tags http_e2e ./tests/http_e2e` passed.
- `ELM_BIN=/opt/homebrew/bin/elm make frontend` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed with a non-fatal global module stat-cache warning from the sandbox.

Pull request 3 verification gaps were recorded:

- Runtime HTTP end-to-end tests were not run locally because the environment rejected the required local listener and PostgreSQL approval after the usage limit was reached.
- Playwright browser tests were not rerun locally because the user interface was not changed and the environment could not grant further browser/listener approval.

Pull request 4 added organizations, teams, and provisioning:

- Team and organization membership identifiers were added to the core identifier set.
- Organization names, team names, organization membership statuses, organization roles, and organization permissions were added under `internal/org`.
- Organization public-publisher permission was modeled separately from reviewer and billing roles.
- Organization service methods were added for organization creation, organization listing, member provisioning, member deactivation, team creation, and team listing.
- Access-token verification was added to the authentication boundary.
- PostgreSQL tables were added for organizations, organization memberships, organization membership roles, teams, and team members.
- PostgreSQL organization repository code was added under `internal/db`.
- HTTP endpoints were added for organization creation, organization listing, organization member provisioning, organization member deactivation, organization team creation, and organization team listing.
- Organization HTTP endpoints required verified bearer access tokens and service-level permission checks.

Pull request 4 test strategy was evaluated:

- Domain constructors, enums, permissions, and service permission checks were covered by unit tests.
- HTTP handler mapping was covered with unit tests using typed test doubles.
- API and PostgreSQL behavior were covered by HTTP end-to-end tests using the real migration runner, repository, service, access tokens, and PostgreSQL.
- Browser user interface tests were not expanded because this task did not change browser user interface source.

Pull request 4 verification was performed:

- `make check-format` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `make check-ts` passed.
- `make lint` passed.
- `GOCACHE=$PWD/.cache/go-build make vet` passed.
- `GOCACHE=$PWD/.cache/go-build make test` passed.
- `GOCACHE=$PWD/.cache/go-build make check-dead-code` passed.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build go test -run '^$' -tags http_e2e ./tests/http_e2e` passed.
- `docker compose up -d postgres` passed.
- `SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make migrate-up` passed.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make test-http` passed.
- `docker compose down` passed.
- `ELM_BIN=/opt/homebrew/bin/elm make frontend` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed with a non-fatal global module stat-cache warning from the sandbox.
