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

Pull request 12 narrowed the asset economy to platform-only rewards, removed anonymous workers, and added organization credit accounts and platform collectibles:

- Anonymous wallet-based submission was removed: the anonymous submitter domain type, wallet address value, public submission route and handler, anonymous columns, and anonymous tests were deleted, and a migration dropped the `submitter_kind` and `wallet_address` columns and made submissions registered-users-only.
- Organization credit accounts were added: a migration extended `credit_accounts` to support organization owners, organizations received a credit account and grant inside the organization-creation transaction, organization-owned tasks can be funded from the organization account behind the manage-billing permission, and an organization credit balance endpoint was added.
- The ledger funding logic for user and organization funding was unified behind a shared escrow-completion helper.
- A platform collectible model was added under `internal/assets` with collectible kinds, lifecycle states, names, and transfer-policy variants, plus a reward-payout policy check.
- The collectible service and PostgreSQL repository added minting, listing, collectible task reward escrow, and refund.
- The submission-acceptance flow was generalized so accepting a submission for a collectible-reward task transfers the collectible to the worker, reported as a collectible payout.
- HTTP endpoints were added for minting and listing collectibles, funding and refunding collectible rewards, and the organization credit balance.
- Generated Elm contracts were extended with the collectible identifier, collectible kinds, states, transfer policies, and collectible responses.
- The browser app gained a collectibles panel for minting, viewing holdings, and awarding a collectible to a task, and the submission request dropped the wallet address field.

Pull request 12 scope decisions were recorded:

- Rewards were kept entirely on-platform: Sharecrop credits are the platform token and platform collectibles are the non-fungible reward. User-issued tokens, organization-issued tokens, crypto rewards, and external wallets were intentionally excluded.
- Anonymous workers were deferred until the anonymous identity and payout model is decided.

Pull request 12 test strategy was evaluated:

- Unit tests covered collectible kind, state, and transfer-policy parsing, the reward-payout policy check, and collectible minting.
- HTTP unit tests covered the collectible response wire shapes through the existing handler doubles.
- Integration tests continued to cover the ledger and series stores against PostgreSQL.
- HTTP end-to-end tests covered organization credit account funding and balance, the collectible award-on-accept flow, collectible reward refund, the issuer-controlled policy denial, and the rewritten registered-user submission tests.
- Playwright tests covered minting a collectible and awarding it to a task through the browser, while preserving the existing dashboard, agent, discovery, and acceptance tests.
- Manual screenshot review covered the collectibles panel.

Pull request 12 verification was performed:

- `make check-format`, `make check-contracts`, `make check-policy`, `make check-ts`, `make check-copy-paste`, `make check-dead-code`, `make lint`, and `make vet` passed.
- `GOCACHE=$PWD/.cache/go-build make test` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed.
- `docker compose up -d postgres` passed and the credit account and collectible migrations applied on a fresh database.
- `make test-integration` and `make test-http` passed.
- `ELM_BIN=/opt/homebrew/bin/elm ... make e2e-ui` passed the app-shell, ledger, agent, screens, and collectible Playwright tests.
- Manual screenshot review passed for `/tmp/sharecrop-pr12-collectibles.png`.

Pull request 13 fixed reward, lifecycle, requester, contract, HTTP, MCP, and session issues found during review:

- Tasks gained an explicit reward specification for no-reward and credit-reward tasks, with response fields for reward kind and credit amount.
- Credit escrow funding now requires the task to declare a matching credit reward, and credit-reward tasks cannot be opened until matching escrow is held.
- Submission acceptance stores the accept idempotency key, same-key retries return the accepted outcome without paying twice, and different-key re-accepts are rejected.
- Submission creation now requires an open visible task, and requester submission listing allows the task creator or organization reviewers.
- Domain errors now distinguish missing resources, permission denials, conflicts, and invalid states so HTTP handlers can return `404`, `403`, and `409` where applicable.
- Organization and collectible funding endpoints use the shared domain HTTP status mapping.
- Generated Elm product decoders now support records larger than eight fields, and generated task and ledger contracts include task detail and accept-submission response shapes.
- MCP task creation requires reward arguments, tool output includes reward details, raw JSON-RPC handling responds to `id:null`, client response objects are ignored as server input, and `/mcp` validates `Accept` and `MCP-Protocol-Version`.
- Browser routing moved to `Browser.application` with dashboard, discovery, and task detail routes.
- Browser auth restores sessions through refresh cookies and clears the refresh cookie through `POST /api/auth/logout`.
- The dashboard gained task creation with optional credit rewards, funding prefill for newly created credit-reward tasks, open and refund controls, task detail viewing, submission detail review, and accept controls.
- Browser task rows and detail screens show reward labels.

Pull request 13 test strategy was evaluated:

- Unit tests covered reward parsing and service-level submission visibility/open-state behavior, organization reviewer submission listing, MCP raw handling, and logout cookie clearing.
- Integration tests covered credit reward funding, acceptance, idempotent re-accept, and refund persistence against PostgreSQL.
- HTTP end-to-end tests covered credit-reward funding and payout, no-reward acceptance, organization credit funding, collectible funding status mapping, task lifecycle status mapping, MCP reward-aware flows, and submission visibility/open-state behavior.
- Playwright tests covered browser task funding with declared rewards, task discovery, worker submission, owner review and acceptance, session switching through logout, and the existing dashboard, agent, ledger, and collectible workflows.
- Manual screenshot review covered the updated dashboard.

Pull request 13 verification was performed:

- `docker compose up -d postgres` passed.
- `GOCACHE=$PWD/.cache/go-build go test ./internal/org ./internal/task ./internal/submission ./internal/http ./internal/db ./internal/mcp` passed.
- `DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations GOCACHE=$PWD/.cache/go-build make test-integration` passed.
- `DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 GOCACHE=$PWD/.cache/go-build make test-http` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make frontend` passed.
- `ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 GOCACHE=$PWD/.cache/go-build make e2e-ui` passed.
- Manual screenshot review passed for `/tmp/sharecrop-dashboard.png`.
- `ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 GOCACHE=$PWD/.cache/go-build make ci` passed before the logout endpoint was added.
- After the logout endpoint was added, the equivalent final local check set passed in target groups: `make check-format check-contracts check-policy check-ts check-copy-paste check-dead-code lint vet test frontend build`, `make test-integration`, `make test-http`, and `make e2e-ui`.

The post-PR13 workflow plan was updated:

- Task rewards were planned as bundles that can contain credits, collectibles, both, or neither.
- Reservation-required and requester-approval-required task flows were planned.
- The default reservation expiry was set to 48 hours, with automatic release on expiry.
- Tasks were planned to allow exactly one active assignee: one user or one team.
- First implementation of team assignment was scoped to users and same-organization teams; public teams remain deferred until public-team modeling exists.
- Reserved tasks were planned to disappear from default discovery and reappear only when the viewer selects include-reserved, except for the active assignee and requester.
- Request changes was planned to require requester notes and keep the same assignee exclusive.
- Review outcomes were planned for accept, request changes, reject with optional partial reward, reject without reward, optional task-local implementor ban, and optional tips from requester balance or inventory.
- MCP work was planned to add workflow tools and full Streamable HTTP SSE with `GET /mcp`, `DELETE /mcp`, session enforcement, event IDs, and replay where practical.
- The next implementation sequence was recorded as PR 14 reservation/approval foundations, PR 15 requester ergonomics and task-page instructions, PR 16 review outcomes, PR 17 reward bundles, and PR 18 MCP workflow tools plus full SSE.

The reservation, approval, and discovery availability foundation branch added backend task assignment support:

- A task reservation identifier was added to the core identifier set.
- Task domain models gained participation policies, assignee scopes, reservation expiry, assignee variants, reservation lifecycle states, availability kinds, and viewer actions.
- Task creation commands and HTTP task creation requests gained participation policy, assignee scope, and reservation expiry values with defaults of open participation, user assignees, and 48 hours.
- PostgreSQL migrations added task participation fields, task reservations, and task-local implementor-ban storage.
- PostgreSQL task storage creates and reads participation fields, releases expired reservations, enforces one active reservation per task, rejects duplicate pending or active reservations by the same assignee, and gates submission eligibility to the active user reservation for reservation-required and approval-required tasks.
- Public task discovery hides actively reserved tasks from unrelated workers by default and shows them when `include_reserved=true`, while keeping them visible to the requester and active assignee.
- HTTP APIs were added for reserving a task, listing task reservations, approving a reservation, declining a reservation, and requester cancellation.
- Submission creation checks task reservation eligibility before validating and storing a response.
- Submission storage marks an active user reservation as submitted when that assignee submits.
- Generated Elm task contracts gained participation, assignee, availability, viewer-action, and reservation response types.
- Unit tests covered reservation service rules and submission eligibility rejection.
- HTTP end-to-end coverage was added for a reservation-required public task: reserve, unrelated submit rejection, default discovery hiding, include-reserved discovery, requester and active-assignee discovery visibility, and active-assignee submission.

The reservation foundation branch verification was performed:

- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-format check-policy check-ts check-copy-paste check-dead-code lint vet test frontend` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 go test -tags http_e2e ./tests/http_e2e -run TestReservationRequiredTaskDiscoveryAndSubmission` passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make test-integration` passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make test-http` passed with local Postgres access.

The requester ergonomics and task page instructions branch improved browser task workflows:

- The requester task list now includes tasks created by the requester even when those tasks are publicly visible.
- Browser task creation gained participation-policy controls and a reservation-expiry field.
- The funding form and collectible-award form now use task selectors sourced from the requester's task list instead of manual task identifier fields.
- Public discovery gained an include-reserved checkbox.
- Task detail pages gained reserve/request-approval actions, requester reservation review controls for approve, decline, and cancel, and task-specific REST and MCP examples.
- Generated static browser assets were rebuilt.

The requester ergonomics branch test coverage was updated:

- HTTP end-to-end coverage checks that a requester-created public task appears in that requester's task list.
- Playwright funding and collectible tests use the new task selectors.
- Playwright coverage was added for creating a reservation-required public task through the browser, opening it, reserving it as a worker, hiding it from another worker by default, and showing it with include-reserved.

The requester ergonomics branch verification was performed:

- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-format check-contracts check-policy check-ts check-copy-paste check-dead-code lint vet test frontend` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make test-integration` passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make test-http` passed with local Postgres access.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui` passed with local Postgres access.
- Manual screenshot review passed for `/tmp/sharecrop-requester-ergonomics.png`.

The review outcomes branch added requester review flows:

- A migration added the `changes_requested` submission state, stored review notes, reviewer metadata, review idempotency keys, and the `task_tip` ledger kind.
- Submission responses expose `review_note`.
- Acceptance supports full or partial credit payout and optional credit tips from the requester balance. Partial acceptance refunds withheld escrow to the funder.
- Request changes requires requester notes, stores the note, moves the submission to `changes_requested`, and reactivates a submitted user reservation for the same implementor.
- Rejection requires requester notes and supports optional partial credit payout from held escrow, optional credit tip from requester balance, and optional task-local implementor ban.
- Task-local implementor bans block direct open-task submissions as well as future reservations.
- HTTP endpoints were added for request changes and rejection, and the existing accept endpoint gained optional `payout_amount` and `tip_amount`.
- MCP tools were added for request changes and rejection, and the accept tool gained optional `payout_amount` and `tip_amount`.
- Browser task detail review controls now include review note, partial payout, tip, ban implementor, accept, request changes, and reject controls.
- Generated Elm contracts and static browser assets were rebuilt.

The review outcomes branch test coverage was updated:

- Ledger service tests cover rejection delegation and ban selection.
- Integration tests cover partial accept with tip, request-changes note storage and reservation reactivation, and reject with partial payout, tip, and implementor ban.
- HTTP contract fixture tests cover submission review notes, accept tips, and review responses.

The review outcomes branch verification was performed:

- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make test-integration` passed after the review outcome integration tests were added.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make test-http` passed after HTTP and MCP review endpoints were added.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod make check-format check-policy check-ts check-copy-paste lint vet frontend build` passed.
- `make check-dead-code` could not be rerun after final changes because the required network escalation for downloading `golang.org/x/tools` was rejected by the approval system.
- A final rerun of `make test-integration`, `make test-http`, and UI screenshot review could not be performed after the final frontend and handler refactor because escalation was rejected by the approval system.

The reward bundles branch added combined reward modeling:

- Task reward specs gained collectible-only and bundled credit-plus-collectible variants.
- A migration allowed `collectible` and `bundle` task reward kinds while keeping credit amounts required only for credit-bearing rewards.
- Task creation through HTTP and MCP accepts reward kinds `none`, `credit`, `collectible`, and `bundle`.
- Task list, detail, generated Elm contracts, MCP summaries, and MCP detail outputs expose `reward_collectible_count` alongside reward kind and credit amount.
- Opening a task now requires held credit escrow for credit-bearing rewards and a held collectible reward for collectible-bearing rewards.
- Credit funding can coexist with a collectible reward on bundled tasks.
- Accepting a bundled task pays the credit escrow and transfers the collectible in one accepted payout outcome.
- Same-key accept retries reconstruct bundled payout responses without paying twice.
- Refunding a bundled task through the credit refund endpoint returns both the held credits and the held collectible; the collectible-only refund endpoint rejects declared bundles so it cannot strand credit escrow.
- Browser reward labels show credits, collectibles, or both.

The reward bundles branch test coverage was updated:

- HTTP end-to-end coverage verifies that bundled tasks cannot open until both reward components are funded, acceptance pays both components, same-key accept retries remain idempotent, and bundled refunds return both credits and the collectible.
- HTTP end-to-end helper response shapes include reward kind, credit amount, and collectible count.

The reward bundles branch verification was performed:

- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make frontend` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 go test -tags http_e2e ./tests/http_e2e` passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 go test -tags integration ./tests/integration` passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-format check-policy check-ts check-copy-paste lint vet test-deno check-dead-code frontend` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui` passed with local Postgres access.
- `make check-contracts` regenerated the intended Elm contract changes and failed before commit because the generated files differed from `HEAD`; it should be rerun after the reward bundles commit.
- Manual screenshot review was skipped; Playwright UI coverage passed.

The MCP workflow and Streamable HTTP SSE branch added the remaining MCP workflow surface:

- MCP services and tools gained task reservation support: reserve/request approval, list task reservations, approve reservation, decline reservation, and cancel reservation.
- Reservation tool results return reservation identifiers, task identifiers, assignee kind and identifier, state, and requester identifier.
- Streamable HTTP MCP now stores initialized HTTP sessions and requires `Mcp-Session-Id` on later non-initialize POST requests.
- `GET /mcp` now serves `text/event-stream`, replays recent session response events after `Last-Event-ID`, stays open, and streams later POST responses to connected clients with event IDs.
- `DELETE /mcp` terminates the current session and later requests with that session ID fail.
- MCP sessions and recent response events are kept in the app process memory.
- Browser task detail MCP curl examples now show initialize first, then session-aware `submit_response` and `get_task_schema` tool calls.

The MCP workflow and Streamable HTTP SSE branch test coverage was updated:

- MCP unit tests cover the new reservation tool dispatch path.
- HTTP end-to-end MCP tests now initialize sessions, include `Mcp-Session-Id` on tool calls, cover reserve/list/approve reservation tools, cover SSE replay, cover live SSE delivery after a later POST, and cover session deletion.
- Existing MCP series tool HTTP tests now use initialized sessions.

The MCP workflow and Streamable HTTP SSE branch verification was performed:

- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 go test -tags http_e2e ./tests/http_e2e` passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 go test -tags integration ./tests/integration` passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui` passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-format check-contracts check-policy check-ts check-copy-paste check-dead-code lint vet test-deno` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build` passed.
- Manual screenshot review was skipped; Playwright UI coverage passed.

The UI themes and GitHub Pages demo branch added static demo and documentation surfaces:

- [docs/user_stories.md](./docs/user_stories.md) was added to map demo visitor, requester, implementor, organization operator, agent operator, platform reviewer, and deferred stories.
- A GitHub Pages static site was added under `site/`.
- The Pages root serves the project landing page.
- `/demo/` serves an interactive static demo with localStorage-backed state.
- `/docs/` serves a documentation placeholder.
- The demo supports light and dark mode selection.
- The demo supports corporate, rustic, blocky, and showcase themes.
- The demo supports demo user selection for requester, implementor, organization reviewer, and agent operator perspectives.
- The demo includes mock Google, Apple, Microsoft, Facebook, and X.com sign-in buttons without implementing provider authentication.
- The demo includes a visible clear-state control.
- The demo maps requester creation, discovery, reservation, approval, submission, review, partial payout, tip, ban, REST instruction, and MCP instruction stories into one static workflow surface.
- A GitHub Actions Pages workflow was added to publish `site/` after pushes to `main` or manual dispatch.
- Playwright coverage was added for static demo theme switching, user switching, local state persistence, and state reset.
- A screenshot helper was added under `tools/` for repeatable demo screenshots.

The UI themes and GitHub Pages demo branch verification was performed:

- `make check-format` passed.
- `make check-ts` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `make test-deno` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make frontend` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-contracts check-dead-code lint vet` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui` passed with local Postgres access.
- Screenshot review was performed for `/tmp/sharecrop-screens/desktop-corporate-light.png`, `/tmp/sharecrop-screens/desktop-blocky-dark.png`, `/tmp/sharecrop-screens/mobile-rustic-light.png`, and `/tmp/sharecrop-screens/mobile-showcase-dark.png`.

The demo UI/UX repair branch refined the GitHub Pages demo:

- The demo was split into separate pages for overview, discovery, requester workflow, review queue, API/MCP instructions, and demo settings.
- The previous all-in-one scrolling demo was replaced with a focused app shell, page tabs, task tables, detail panels, and review panels.
- Demo login was moved into a discrete top-right account control that starts as Guest and opens a login panel.
- The login panel supports selecting a demo user and choosing mock Google, Apple, Microsoft, Facebook, and X.com provider buttons.
- The login panel closes after user selection, provider selection, or page navigation so it does not block other controls.
- Dark theme overrides were fixed for all visual themes.
- A demo audit helper was added to check deployed and local demo pages for console warnings, console errors, page errors, failed requests, and horizontal overflow.
- Playwright static demo coverage was updated for the separated navigation and login flow.

The demo UI/UX repair branch verification was performed:

- `make check-format` passed.
- `make check-ts` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `make test-deno` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-contracts check-dead-code lint vet` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui` passed with local Postgres access.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys tools/audit_demo_ui.ts` passed for deployed and local demo pages.
- Screenshot review was performed for `/tmp/sharecrop-screens/desktop-corporate-light.png` and `/tmp/sharecrop-screens/mobile-showcase-dark.png`.

The `task/demo-performance-flow-review` branch repaired the static demo after performance and flow review:

- The demo runtime was changed from per-render event binding to document-level delegated event handling.
- Text input now updates in-memory state and debounces localStorage writes instead of re-rendering the page on every keystroke.
- Locally created demo tasks, reservations, and submissions are bounded so localStorage state cannot grow without limit.
- localStorage quota failures clear the demo state and return the user to settings.
- The expensive sticky translucent top bar with backdrop blur was replaced with an opaque non-sticky top bar.
- The top-level demo pages were kept separate for overview, discovery, requester create, review, API/MCP instructions, and settings.
- Demo clear-state controls were moved out of the header and into settings.
- Requester task creation now shows title, description, reward, visibility, participation policy, and reservation expiry fields.
- Discovery actions now match the selected task policy: reserve, request approval, submit, or no action.
- Review controls are per reservation and per submission, with approve, decline, release, request changes, reject, accept, partial payout, tip, and ban controls.
- API/MCP instructions separate worker REST, worker MCP, requester MCP, and credential setup placeholders.
- The static demo Playwright test covers theme selection, user selection, typed task creation, local state persistence, and clear state through the settings page.
- The screenshot capture helper targets exact page-tab labels and captures overview, discovery, create, review, API/MCP, settings, desktop theme, and mobile theme states.
- The Elm build helper rejects the recursive npm Elm wrapper when `ELM_BIN` points to it, preventing local builds from hanging and flooding Node warnings.
