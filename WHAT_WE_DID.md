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

Pull request 5 added the Go-to-Elm contract generator:

- Go-owned contract definitions were added under `internal/contracts`.
- The contract model covered aliases, product types, enums, named type references, string type references, and list type references.
- Elm generation was added for modules, type aliases, enums, decoders, and encoders.
- Generated Elm modules were added under `web/elm/src/Sharecrop/Generated/`.
- First generated contracts covered auth responses, error responses, identifiers, organization responses, organization member responses, team responses, membership statuses, subject kinds, and organization roles.
- The `sharecrop generate elm-contracts` command was added.
- The Makefile gained `contracts` and `check-contracts` targets.
- Frontend builds were changed to generate contracts before compiling Elm.
- The handwritten Elm app consumed the generated `Sharecrop.Generated.Auth.SubjectKind` type directly.

Pull request 5 test strategy was evaluated:

- Generator unit tests checked generated auth output, deterministic output, and absence of weak generated Elm shapes such as `Bool` and `Dict`.
- `check-contracts` verified generated files were current and deterministic.
- Elm compilation verified generated modules worked with Elm 0.19.1.
- The handwritten Elm app imported a generated module to ensure generated contracts were usable from normal Elm code.
- Existing HTTP end-to-end tests remained the API behavior checks for this slice.
- Playwright and manual screenshot checks were run because Elm source changed.

Pull request 5 verification was performed:

- `make check-format` passed.
- `make check-contracts` passed with `GOCACHE=$PWD/.cache/go-build`.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `make check-ts` passed.
- `make lint` passed.
- `GOCACHE=$PWD/.cache/go-build make vet` passed.
- `GOCACHE=$PWD/.cache/go-build make test` passed.
- `GOCACHE=$PWD/.cache/go-build make check-dead-code` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make frontend` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed with a non-fatal global module stat-cache warning from the sandbox.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build go test -run '^$' -tags http_e2e ./tests/http_e2e` passed.
- `docker compose up -d postgres` passed.
- `SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make migrate-up` passed.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make test-http` passed.
- `ELM_BIN=/opt/homebrew/bin/elm SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build deno task e2e:ui` passed.
- Manual screenshot review passed for `/tmp/sharecrop-pr5-shell.png`.
- `docker compose down` passed.

Pull request 6 added the Sharecrop schema parser and validator:

- Local schema domain types were added under `internal/schema`.
- Schema kinds were added for object, array, string, integer, decimal string, enum, literal, union, and freeform schemas.
- Field presence was modeled as explicit `required` and `may_omit` values.
- Sensitivity categories, retention policies, and redaction policies were modeled as typed values.
- Schema JSON parsing converted boundary data into Sharecrop-owned schema types.
- Response payload JSON parsing converted payloads into Sharecrop-owned value types without using generic maps.
- Schema validation produced typed validation errors with field paths.
- Sensitive-field indexing and redaction were added for typed response values.

Pull request 7 added task series, tasks, visibility, and capability tokens:

- Task-series, task, task-visibility-scope, and task-capability-token migrations were added.
- Task-series and task-capability-token identifiers were added to the core identifier set.
- Task owner, task state, task series placement, task visibility, task payload, and task capability-token lifecycle types were added under `internal/task`.
- Opaque task capability-token generation and hashing were added without encoding task identifiers into token strings.
- The task service added task creation, opening, cancellation, listing, and capability-token creation.
- Organization-owned tasks required organization task-creation permission.
- Public organization tasks required organization public-publisher permission.
- Default task visibility mapped user-owned tasks to user visibility and organization-owned tasks to organization visibility.
- PostgreSQL task repository code was added under `internal/db`.
- HTTP task endpoints were added for creation, listing, opening, cancellation, and capability-token creation.
- Task response schemas were parsed with the local Sharecrop schema parser during task creation.
- Generated Elm contracts were extended with task identifiers, task enums, task list items, task lists, and task capability-token responses.

Pull request 7 test strategy was evaluated:

- Unit tests covered task state transitions, capability-token opacity, capability-token parsers, identifier round trips, and organization public-publishing permission behavior.
- HTTP unit tests covered task request parsing and default user visibility.
- HTTP end-to-end tests covered task creation, user-scoped listing, task opening, task cancellation, capability-token creation, and organization public-publishing permissions against PostgreSQL.
- Playwright and manual screenshot checks were run because generated Elm source changed.

Pull request 7 verification was performed:

- `GOCACHE=$PWD/.cache/go-build go test ./...` passed.
- `make check-format` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `deno task check:ts` passed.
- `deno task lint` passed.
- `GOCACHE=$PWD/.cache/go-build make vet` passed.
- `GOCACHE=$PWD/.cache/go-build make check-dead-code` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make frontend` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed with a non-fatal global module stat-cache warning from the sandbox.
- `docker compose up -d postgres` passed.
- `SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build go run ./cmd/sharecrop migrate up` passed.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build go test -tags http_e2e ./tests/http_e2e` passed.
- `ELM_BIN=/opt/homebrew/bin/elm SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build deno task e2e:ui` passed.
- Manual screenshot review passed for `/tmp/sharecrop-pr7-shell.png`.

Pull request 8 added submissions, anonymous access, and sensitive-field handling:

- Submission and submission receipt-token identifiers were added to the core identifier set.
- Submission tables were added for submissions, receipt tokens, validation errors, and sensitive-field index rows.
- Submission domain types were added for authenticated submitters, anonymous wallet submitters, submission states, validation outcomes, response JSON, wallet addresses, and receipt tokens.
- Opaque receipt-token generation and hashing were added.
- The submission service added authenticated submission, anonymous public submission, receipt lookup, and requester submission listing.
- Anonymous submissions were limited to public tasks.
- Anonymous submitters stored payout wallet addresses without linking them to user identifiers.
- Submitted response JSON was parsed and validated against the task response schema.
- Schema-invalid submissions were recorded with `invalid` state and validation-error rows.
- Sensitive submitted fields were indexed from the task response schema.
- Receipt lookup returned redacted response JSON for sensitive fields.
- PostgreSQL submission repository code was added under `internal/db`.
- HTTP endpoints were added for authenticated task submissions, anonymous public task submissions, receipt status, and requester submission listing.
- Generated Elm contracts were extended with submission identifiers, submission states, submitter kinds, validation-error responses, submission responses, submission lists, and submission-created responses.

Pull request 8 test strategy was evaluated:

- Unit tests covered anonymous/public submission permission, receipt-token creation, invalid submission recording, sensitive redaction for receipt lookup, and identifier round trips.
- HTTP unit tests covered authenticated submission request handling and receipt-token response shape.
- HTTP end-to-end tests were added for anonymous public submission, receipt redaction, invalid response recording, and requester submission listing.
- Browser user interface tests were not expanded because pull request 8 did not add visible submission screens.

Pull request 8 verification was performed:

- `GOCACHE=$PWD/.cache/go-build go test ./...` passed.
- `make check-format` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `deno task check:ts` passed.
- `deno task lint` passed.
- `GOCACHE=$PWD/.cache/go-build go vet ./...` passed.
- `GOCACHE=$PWD/.cache/go-build make check-dead-code` passed.
- `go run ./cmd/sharecrop generate elm-contracts` regenerated identical generated Elm contracts.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make frontend` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed.
- `docker compose up -d postgres` passed.
- `DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build go run ./cmd/sharecrop migrate up` applied the submission migration.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make test-http` passed, including anonymous submission, receipt redaction, invalid-response recording, and requester listing tests.
- `ELM_BIN=/opt/homebrew/bin/elm ... make e2e-ui` passed the app-shell Playwright smoke test.
- `deno task test` passed.
- `docker compose down` passed.
- Sensitive-field indexing located sensitive values in submitted payloads.
- Redaction replaced or removed sensitive fields according to schema policy.

Pull request 6 test strategy was evaluated:

- Parser tests covered typed parsing, unsupported schema kinds, freeform mode, union schemas, and enum rejection.
- Validator tests covered required field failures and valid object payloads.
- Sensitive-data tests covered sensitive path indexing, replacement redaction, and remove redaction.
- Existing HTTP end-to-end tests remained the API behavior checks for this slice because task and submission endpoints are not implemented yet.
- Browser user interface tests were not expanded because this task did not change browser user interface source.

Pull request 6 verification was performed:

- `make check-format` passed.
- `make check-contracts` passed with `GOCACHE=$PWD/.cache/go-build`.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `make check-ts` passed.
- `make lint` passed.
- `GOCACHE=$PWD/.cache/go-build make vet` passed.
- `GOCACHE=$PWD/.cache/go-build make test` passed.
- `GOCACHE=$PWD/.cache/go-build make check-dead-code` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make frontend` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed with a non-fatal global module stat-cache warning from the sandbox.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build go test -run '^$' -tags http_e2e ./tests/http_e2e` passed.
- `docker compose up -d postgres` passed.
- `SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make migrate-up` passed.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make test-http` passed.
- `docker compose down` passed.

Pull request 9 added credits, ledger, escrow, first accepted submission, the credit ledger user interface, and an expanded test pyramid:

- Credit account and ledger entry identifiers were added to the core identifier set.
- Credit account, append-only ledger entry, and task escrow tables were added, along with a single-accepted partial unique index on submissions.
- Credit domain types were added under `internal/ledger` for positive credit amounts, signed ledger amounts, derived balances, ledger entry kinds, escrow states, idempotency keys, ledger entries, and task escrows.
- Balance derivation summed the signed amounts of an account's ledger entries.
- The ledger service added task funding, submission acceptance with payout, task refund, balance lookup, and ledger listing.
- The PostgreSQL ledger repository performed funding, acceptance, and refund inside row-locked transactions.
- Each new registered user received a credit account and a `signup_grant` of 100 credits inside the user-creation transaction.
- Task funding escrowed credits from the funder's account and required sufficient balance and a draft, owner-held task.
- Submission acceptance was transactional, closed the task, enforced a single accepted submission per task, and paid the accepted authenticated worker from the escrow.
- Task refund cancelled a funded task and returned escrowed credits to the funder.
- Fund, accept, and refund used idempotency keys so retries did not double-charge or double-pay.
- HTTP endpoints were added for credit balance, ledger listing, task funding, submission acceptance, and task refund.
- The contract generator gained an integer reference type, and a single-field record decoder was changed from `Decode.mapN` to `Decode.map`.
- Generated Elm contracts were extended with credit account and ledger entry identifiers, ledger entry kinds, escrow states, balance responses, ledger entry responses, ledger responses, and task escrow responses.
- The Elm app was changed from an app shell into an interactive client with register and login, a credit balance and ledger view, and a task funding form backed by the API.
- A Postgres-backed integration test tier was added under the `integration` build tag with a `make test-integration` target.
- continuous integration was split into parallel static, unit, build, integration, HTTP end-to-end, and Playwright jobs.

Pull request 9 test strategy was evaluated:

- Unit tests covered credit amount validation, signed amount parsing, ledger entry kind and escrow state parsing, idempotency key validation, balance derivation, and ledger service delegation.
- HTTP unit tests covered the credit balance endpoint and task funding request handling with typed test doubles.
- Integration tests covered the signup grant, funding, single-escrow enforcement, acceptance payout, idempotent acceptance, and refund against PostgreSQL.
- HTTP end-to-end tests covered the signup grant, the fund-open-submit-accept-payout flow, idempotent acceptance, single-accepted enforcement, refund, insufficient-credit funding, and no-reward acceptance.
- Playwright tests covered registering through the browser to see the signup grant balance and ledger entry, and funding a task through the browser.
- Manual screenshot review covered the logged-out shell and the logged-in credit dashboard.

Pull request 9 verification was performed:

- `GOCACHE=$PWD/.cache/go-build go test ./...` passed.
- `make check-format` passed.
- `make check-contracts` passed.
- `make check-policy` passed.
- `make check-ts` passed.
- `make check-copy-paste` passed.
- `GOCACHE=$PWD/.cache/go-build make check-dead-code` passed.
- `make lint` passed.
- `GOCACHE=$PWD/.cache/go-build make vet` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed.
- `docker compose up -d postgres` passed.
- `DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make migrate-up` applied the credits and ledger migration.
- `DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make test-integration` passed and was rerun to confirm idempotency safety against a persistent database.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make test-http` passed.
- `ELM_BIN=/opt/homebrew/bin/elm ... make e2e-ui` passed the app-shell, signup-grant, and browser-funding Playwright tests.
- Manual screenshot review passed for `/tmp/sharecrop-pr9-shell.png` and `/tmp/sharecrop-pr9-dashboard.png`.
- `docker compose down` was run after verification.

Pull request 10 added MCP, agent credentials, agent setup, and task discovery surfaces:

- Agent credential identifiers were added to the core identifier set.
- Agent credential and agent credential scope tables were added.
- Agent credential domain types were added under `internal/agent` for scopes, lifecycle state, labels, opaque secrets and hashes, scope sets, and scope checks.
- The agent credential service added scoped creation, verification, listing, and revocation, with PostgreSQL repository code that stored scopes in a child table.
- A local MCP JSON-RPC server was added under `internal/mcp`, implemented from the MCP specification without a Go MCP library, handling `initialize`, `ping`, `tools/list`, and `tools/call`.
- MCP tools were added for `sharecrop.list_tasks`, `sharecrop.get_task`, `sharecrop.get_task_schema`, `sharecrop.create_task`, `sharecrop.submit_response`, `sharecrop.get_submission_status`, `sharecrop.list_task_submissions`, and `sharecrop.accept_submission`, each gated by an agent scope and adapted over the existing task, submission, and ledger services.
- A task service `Get` method and a `GET /api/tasks/{task_id}` endpoint were added with a task view-permission check covering creators, public tasks, user visibility, and organization visibility.
- HTTP endpoints were added for agent credential creation, listing, and revocation, and a `POST /mcp` endpoint authenticated by agent credentials with per-tool scope enforcement.
- Generated Elm contracts were extended with the agent credential identifier, agent scopes, agent credential state, and agent credential responses.
- The browser app gained a task list panel with REST and MCP curl examples per task, and an agent setup panel for creating, viewing, and revoking scoped credentials with generated MCP client configuration and a one-time token.
- The Elm app was changed to accept an `origin` flag so the generated MCP configuration and curl examples use the live server origin.

Pull request 10 test strategy was evaluated:

- Unit tests covered agent scope parsing, scope-set de-duplication and checks, opaque secret round trips, label validation, and agent service create/verify/revoke.
- MCP unit tests covered initialize, tools/list, unknown methods, scope enforcement, tool dispatch, unknown tools, and domain rejections surfaced as tool errors.
- HTTP unit tests covered agent credential creation, unknown-scope rejection, and the MCP endpoint requiring an agent credential.
- Integration tests covered agent credential create, verify, list, and revoke against PostgreSQL.
- HTTP end-to-end tests covered the agent discover-submit-status-list-accept flow over MCP with a credit payout, MCP scope enforcement, revoked-credential rejection, and the single-task REST endpoint.
- Playwright tests covered creating an agent credential through the browser to see the token and MCP configuration, and listing the user's tasks with agent curl examples.
- Manual screenshot review covered the agent setup panel.

Pull request 10 verification was performed:

- `GOCACHE=$PWD/.cache/go-build go test ./...` passed.
- `make check-format`, `make check-contracts`, `make check-policy`, `make check-ts`, `make check-copy-paste`, `make check-dead-code`, `make lint`, and `make vet` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed.
- `docker compose up -d postgres` passed and the agent credentials migration applied.
- `make test-integration` passed and was idempotency-safe across reruns.
- `make test-http` passed, including the MCP and agent flows.
- `ELM_BIN=/opt/homebrew/bin/elm ... make e2e-ui` passed the app-shell, ledger, and agent Playwright tests.
- Manual screenshot review passed for `/tmp/sharecrop-pr10-agents.png`.

Pull request 11 added deferred backend gaps, MCP transports, and user interface polish with new screens:

- UUIDv7 generation was verified in code for version 7 and time ordering, and a parser-rejection test was added.
- HTTP contract fixture tests were added to pin the wire JSON shape of representative API responses.
- A task-series read API was added: `task` service `ListSeries` and `GetSeries` with a series view-permission check, PostgreSQL `ListSeries` and `FindSeries` repository code, `GET /api/task-series` and `GET /api/task-series/{id}` endpoints, and generated Elm task-series contracts.
- The MCP server gained `sharecrop.list_task_series` and `sharecrop.get_task_series` tools.
- The MCP server gained JSON-RPC batch handling through a shared `HandleRaw` entry point used by both transports.
- The MCP HTTP endpoint was hardened toward Streamable HTTP: a `Mcp-Session-Id` header on initialize, `Origin` validation for DNS-rebinding protection, a `405` response on `GET`, and a request body size limit.
- A stdio MCP transport was added through a `sharecrop mcp` command that authenticates with `SHARECROP_AGENT_TOKEN`, verifies the agent credential, and drives the same MCP server over newline-delimited JSON-RPC on stdin and stdout. This is the transport local agent clients launch.
- The transport surface was chosen from what Claude Code and Codex both implement as MCP clients: stdio and Streamable HTTP with a static bearer token. HTTP/1.1 and HTTP/2 are negotiated by the web server, and HTTP/3 and raw sockets were intentionally not added.
- A reusable shadcn-inspired Elm component module was added under `Sharecrop.Ui` with cards, buttons, inputs, badges, code blocks, and labels, and the app was refactored to use it.
- Browser page navigation was added with a public task discovery screen and a task detail screen that submits responses and lets task owners review and accept submissions.

Pull request 11 test strategy was evaluated:

- Unit tests covered UUIDv7 version and ordering, contract wire shapes, the series view-permission check, the MCP series tools, JSON-RPC batch and notification handling, and the stdio loop.
- Integration tests covered the task-series store list and find against PostgreSQL.
- HTTP end-to-end tests covered the series REST endpoints, the MCP series tools, MCP batch requests, the `GET` `405`, and the `Mcp-Session-Id` header.
- The stdio command was smoke-tested end to end against PostgreSQL by piping `initialize` and `tools/list` to `sharecrop mcp`.
- Playwright tests covered discovering a public task, submitting through the browser, and an owner reviewing and accepting the submission, while preserving the existing dashboard and agent-setup tests.
- Manual screenshot review covered the task detail screen.

Pull request 11 verification was performed:

- `make check-format`, `make check-contracts`, `make check-policy`, `make check-ts`, `make check-copy-paste`, `make check-dead-code`, `make lint`, and `make vet` passed.
- `GOCACHE=$PWD/.cache/go-build make test` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed.
- `docker compose up -d postgres` passed and the existing migrations applied.
- `make test-integration` passed and remained idempotency-safe across reruns.
- `make test-http` passed, including the series and MCP transport tests.
- `ELM_BIN=/opt/homebrew/bin/elm ... make e2e-ui` passed the app-shell, ledger, agent, and screens Playwright tests.
- `SHARECROP_AGENT_TOKEN=... go run ./cmd/sharecrop mcp` returned the initialize result and tool list over stdio.
- Manual screenshot review passed for `/tmp/sharecrop-pr11-detail.png`.
