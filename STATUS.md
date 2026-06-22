# Status

The repository contains pull request 1 through pull request 18 work. Pull request 18 was merged into `main`.

Active task:

- Active branch `task/mcp-workflow-sse` implemented remaining MCP workflow tools and Streamable HTTP SSE support and is awaiting pull request creation.

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
- continuous integration workflow for pull requests targeting `main`, split into parallel jobs: static checks (formatting, contracts, policy, type checks, copy-paste, dead-code, lint, vet), unit tests, frontend and binary build, Postgres integration tests, HTTP end-to-end tests, and Playwright user interface end-to-end tests.
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
- Go-owned contract definitions under `internal/contracts`.
- Deterministic Elm contract generation for auth, errors, identifiers, organizations, and teams.
- Generated Elm modules under `web/elm/src/Sharecrop/Generated/`.
- The handwritten Elm app directly consumes a generated auth contract type.
- Makefile contract generation and deterministic contract checks.
- Local Sharecrop schema domain types under `internal/schema`.
- Schema JSON parsing for object, array, string, integer, decimal string, enum, literal, union, and freeform schemas.
- Response payload parsing into Sharecrop-owned value types.
- Schema validation with typed validation errors and field paths.
- Sensitive-field indexing and redaction for typed response values.
- Task series, tasks, task visibility scopes, and task capability token tables.
- Task owner, task state, task series placement, visibility, payload, and capability-token domain types.
- Task service methods for creation, opening, cancellation, listing, and capability-token creation.
- Task PostgreSQL repository code.
- HTTP task endpoints for creation, listing, opening, cancellation, and capability-token creation.
- Organization-owned tasks default to organization-scoped visibility when the request uses default visibility.
- Public organization tasks require the organization public-publisher permission.
- Task response schemas are parsed by the local Sharecrop schema parser during task creation.
- Generated Elm task contract enums and list-item read models.
- Submission and submission receipt-token identifiers.
- Submission, receipt-token, validation-error, and sensitive-field-index tables.
- Submission domain types for authenticated submitters, anonymous wallet submitters, lifecycle states, validation outcomes, and receipt tokens.
- Submission service methods for authenticated submission, anonymous public submission, receipt lookup, and requester submission listing.
- Submission PostgreSQL repository code.
- HTTP submission endpoints for authenticated submission, anonymous public submission, receipt lookup, and task submission listing.
- Anonymous submissions require public tasks and store payout wallet addresses without user linkage.
- Submission responses are validated against the task response schema and invalid responses are recorded with validation errors.
- Receipt lookup redacts sensitive fields from response JSON.
- Generated Elm submission contracts.
- Credit account and ledger identifiers.
- Credit account, append-only ledger entry, and task escrow tables.
- A single-accepted partial unique index on submissions.
- Credit domain types for positive credit amounts, signed ledger amounts, derived balances, ledger entry kinds, escrow states, and idempotency keys.
- A ledger service for task funding, submission acceptance with payout, task refund, balance lookup, and ledger listing.
- Each new registered user receives a credit account and a `signup_grant` of 100 credits in the user-creation transaction.
- Task funding escrows credits from the funder and requires sufficient balance.
- Submission acceptance is transactional, enforces a single accepted submission per task, and pays the accepted authenticated worker from the escrow.
- Task refund cancels a funded task and returns escrowed credits to the funder.
- Fund, accept, and refund use idempotency keys.
- HTTP endpoints for credit balance, ledger listing, task funding, submission acceptance, and task refund.
- An integer reference type in the contract generator and a single-field record decoder fix.
- Generated Elm ledger contracts.
- An interactive Elm app with register/login, a credit balance and ledger view, and a task funding form backed by the API.
- A separate Postgres-backed integration test tier under the `integration` build tag.
- continuous integration split into parallel static, unit, build, integration, HTTP end-to-end, and Playwright jobs.
- Agent credential identifiers, tables, and a child scope table.
- Agent credential domain types for scopes, lifecycle state, labels, opaque secrets, and scope sets under `internal/agent`.
- An agent credential service for scoped creation, verification, listing, and revocation, with PostgreSQL repository code.
- A local MCP JSON-RPC server under `internal/mcp`, implemented from the specification without a Go MCP library, exposing `initialize`, `tools/list`, and `tools/call`.
- MCP tools for listing tasks, getting a task, getting a task schema, creating a task, submitting a response, getting submission status, listing task submissions, and accepting a submission, each gated by an agent scope.
- A single-task read endpoint with a task view-permission check.
- HTTP endpoints for agent credential creation, listing, and revocation, and a `POST /mcp` endpoint authenticated by agent credentials.
- Generated Elm agent contracts.
- Browser screens for listing the user's tasks with REST and MCP curl examples, and for creating, viewing, and revoking scoped agent credentials with generated MCP client configuration.
- Task-series read API with `GET /api/task-series` and `GET /api/task-series/{id}` and a series view-permission check.
- `list_task_series` and `get_task_series` MCP tools.
- JSON-RPC batch handling in the MCP server.
- A stdio MCP transport through the `sharecrop mcp` command, authenticated by `SHARECROP_AGENT_TOKEN`, driving the same MCP server as the HTTP endpoint.
- Streamable HTTP hardening on `/mcp`: `Mcp-Session-Id` on initialize, `Origin` validation, a `405` on `GET`, and a request body limit.
- UUIDv7 generation verified in code for version and time ordering.
- HTTP contract fixture tests pinning response wire shapes.
- A reusable shadcn-inspired Elm component module under `Sharecrop.Ui`.
- Browser page navigation with a public task discovery screen and a task detail screen with response submission and owner submission review and acceptance.
- Submissions are restricted to registered users; anonymous wallet-based submission was removed.
- Organization credit accounts: organizations receive a credit account and grant on creation, fund organization-owned tasks from the organization account behind the billing permission, and expose an organization credit balance.
- Platform collectibles under `internal/assets` with kinds, lifecycle states, transfer-policy variants, and a policy check for reward payout.
- Collectible minting, collectible task rewards with escrow, transfer to the accepted worker on acceptance, and refund.
- HTTP endpoints for minting and listing collectibles, funding and refunding collectible rewards, and an organization credit balance.
- Generated Elm collectible contracts and a browser collectibles panel for minting, holdings, and awarding a collectible to a task.
- Task reward specifications for no-reward, credit-reward, collectible-reward, and bundled credit-plus-collectible tasks.
- Task create, list, detail, MCP summary, and MCP detail responses expose reward kind, credit amount, and collectible reward count.
- Credit escrow funding requires a matching declared credit reward, and credit or bundled tasks require matching held escrow before opening.
- Collectible-reward and bundled tasks require a held collectible reward before opening.
- Submission acceptance can pay no reward, credits, a collectible, or a bundled credit-plus-collectible payout.
- Bundled reward refunds return held credits and the held collectible together.
- Accept-submission idempotency is stored per accepted submission and same-key retries do not pay twice.
- Submission creation requires an open visible task, and requester submission listing allows the creator or organization reviewers.
- Domain HTTP errors map missing resources, permission denials, invalid state, and conflicts to `404`, `403`, and `409` where applicable.
- Generated Elm contract decoding supports products larger than eight fields.
- MCP create-task arguments include reward fields, MCP raw handling responds to `id:null`, client response messages are not dispatched as server requests, and `/mcp` validates `Accept` and `MCP-Protocol-Version`.
- Browser routing uses `Browser.application` with dashboard, discovery, and task detail routes.
- Browser auth restores sessions through the refresh cookie on load and clears the refresh cookie on logout.
- The browser dashboard can create tasks with optional credit rewards, prefill funding for newly created credit-reward tasks, open and refund tasks, and review submission details before accepting.
- Task participation policies: open submissions, reservation required, and requester approval required.
- Task assignee scopes for users and organization teams.
- Task reservation identifiers, reservation lifecycle states, and reservation expiry values with a 48-hour default.
- PostgreSQL task reservation storage, one-active-reservation enforcement, expired-reservation release, and task-local implementor-ban storage.
- HTTP reservation APIs for reserve, approve, decline, cancel, and list reservations.
- Submission creation checks reservation eligibility before storing a response.
- Public discovery hides actively reserved tasks from unrelated workers by default and shows them with `include_reserved=true`, while keeping them visible to the requester and active assignee.
- Task create, list, and detail responses expose participation policy, assignee scope, reservation expiry, availability kind, and viewer action.
- Browser task creation exposes participation policy and reservation expiry controls.
- Browser funding and collectible-award forms select from the requester's task list instead of requiring manual task identifiers.
- Browser discovery exposes an include-reserved control.
- Browser task detail pages expose reserve/request-approval actions, requester reservation review controls, and task-specific REST and MCP examples.
- Requester task lists include tasks created by the requester even when the task is publicly visible.
- Submission review stores requester notes and exposes them on submission responses.
- Requesters can request changes for submitted work; the submission moves to `changes_requested` and a submitted reservation is reactivated for the same implementor.
- Requesters can reject submitted work with required notes, optional partial credit payout from held escrow, optional credit tip from current requester balance, and optional task-local implementor ban.
- Requesters can accept submitted work with full or partial credit payout and optional credit tip; partial acceptance refunds withheld escrow to the funder.
- Task-local implementor bans block later direct submissions as well as later reservations for the same task.
- HTTP review endpoints exist for accept, request changes, and reject.
- MCP review tools support accept with optional payout/tip, request changes, and reject with optional partial payout/tip/ban.
- MCP reservation tools support reserve/request-approval, list task reservations, approve reservation, decline reservation, and cancel reservation.
- Streamable HTTP MCP supports initialized `Mcp-Session-Id` sessions, session enforcement on later POST requests, `GET /mcp` server-sent event streams, response event IDs, `Last-Event-ID` replay for recent session events, live delivery of later POST responses to open SSE streams, and `DELETE /mcp` session termination.
- Browser task detail submission review controls expose review note, partial payout, tip, ban, accept, request-changes, and reject actions.
- Browser task detail MCP curl examples show initialize plus session-aware tool calls.

Planned defaults:

- Public-team assignment is deferred unless public teams already exist; first implementation supports users and same-organization teams.

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

Last observed checks on `task/mcp-workflow-sse`:

- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 go test -tags http_e2e ./tests/http_e2e` passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 go test -tags integration ./tests/integration` passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui` passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-format check-contracts check-policy check-ts check-copy-paste check-dead-code lint vet test-deno` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build` passed.

Blocking issues:

- None known.

See [PLAN.md](./PLAN.md) for the product and architecture plan.
See [DO_NEXT.md](./DO_NEXT.md) for the next tasks.
See [BUGS.md](./BUGS.md) for known bugs and risks.
