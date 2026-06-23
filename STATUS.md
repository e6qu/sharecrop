# Status

The repository contains pull request 1 through pull request 31 work, merged into `main`.

Active task:

- Active branch `task/team-pages-and-module-split` adds team detail pages and a team-assignee selector, then decomposes the HTTP and browser monoliths. The branch is ready for review. See [WHAT_WE_DID.md](./WHAT_WE_DID.md).

Implemented in `task/team-pages-and-module-split`:

- `GET /api/teams/{id}` returns a team and its roster, gated so only the team owner, a team member, or (for an organization team) a member of the owning organization may read it. A routed `/teams/{id}` page shows the team and its members; organization team rows link to it.
- The create-task form offers an assignee scope (user or organization team) instead of always assigning to a user.
- The organization and team HTTP handlers moved into `internal/http/organizations.go`, the funding handlers into `internal/http/funding.go`, and the pure Elm label helpers into `web/elm/src/Sharecrop/Labels.elm`, shrinking `server.go` and `Main.elm` with no behavior change.

Implemented earlier in `task/org-followups` (pull request 31, merged):

- The static demo rewrites its seed tasks to be self-contained and adds hash-routed pages, including per-user profiles.
- The browser app gives each entity its own URL: routed `/organizations/{id}`, role-aware `/tasks/{id}`, `/users/{id}` profiles, `/users/{id}/work`, `/users/{id}/submissions`, `/collectibles/{id}`, and `/series/{id}`.
- `GET /api/organizations/{id}/members`, `GET /api/users/{id}`, `GET /api/users/{id}/work`, and `GET /api/users/{id}/submissions` back those pages, each with role-based access control enforced and tested against leaks.
- The create-task form offers team and organization visibility scopes, and the funding form can fund a task from organization credits.

Implemented in `task/multi-page-routing`:

- The HTTP server serves the single-page-application shell for every non-API route, so deep links and refreshes load the app. Unmatched API paths still return 404.
- The browser app has a route and URL per section: `/` overview, `/tasks`, `/tasks/new`, `/tasks/{id}`, `/discovery`, `/funding`, `/agents`, `/collectibles`, `/organizations`. The navigation bar uses real links, and each section is its own page instead of one stacked dashboard panel.
- The static demo has an always-visible reset control in the top bar, in addition to the one on the settings page.

Earlier active task:

- Branch `task/teams-org-context-collectible-ui` added standalone (user-owned) teams, organization context in the browser, and multi-collectible reward visibility. It was merged into `main`. See below.

Implemented in `task/teams-org-context-collectible-ui`:

- Team ownership is a tagged union: organization-owned or user-owned. Standalone teams are created and listed through `POST` and `GET /api/teams`, and the owner is exposed on the team contract.
- The browser switches the active organization and shows its credit balance, organization-scoped task list, and teams. The active organization panel creates teams and provisions members, and the task creation form can own a task by an organization.
- The browser surfaces the escrowed collectible count on tasks, including collectibles awarded ad hoc, and refreshes the task list after awarding.

Earlier active task:

- Branch `task/full-review-improvements` carried a multi-area review and improvements across security, backend correctness, the browser UI, the data model, and code structure. It was merged into `main`. See the implemented surface below.

Implemented in `task/full-review-improvements`:

- Submission read paths keep redaction for unauthorized viewers (receipt token holders) and return unredacted data to authorized requesters and organization reviewers; tests pin both behaviors and confirm non-reviewers receive `403`.
- Submission accept, reject, and request-changes authorize the task creator or an organization member with the review-submissions permission, resolved inside the review transaction.
- The Sharecrop schema parser and value parser bound nesting depth and reject overly nested input.
- A request body size limit applies to all JSON HTTP endpoints.
- Refresh tokens belong to a family; reusing a consumed or revoked token revokes the whole family.
- MCP HTTP sessions are evicted after an idle timeout.
- A concurrency test confirms the transactional acceptance path keeps at most one accepted submission per task.
- A task can escrow multiple collectible rewards; acceptance transfers all held collectibles to the worker and refund returns them all.
- List endpoints accept `limit` and `offset` pagination through a `core.Page` value type, defaulting to a bounded page.
- The task list accepts `state` and `participation_policy` filters, and task list items expose the active reservation assignee.
- The browser task creation form exposes public, private, and specific-user visibility; the dashboard shows task-state guidance, a task-state filter, the active assignee on task rows, and an organizations panel that lists and creates organizations.
- Checkbox and label accessibility and badge and label contrast were improved through shared `Sharecrop.Ui` helpers.
- Auth HTTP handlers were moved from `internal/http/server.go` into `internal/http/auth_handlers.go`.

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
- [docs/user_stories.md](./docs/user_stories.md) maps the current browser, demo, HTTP API, MCP, requester, implementor, organization, and agent-operator stories.
- The GitHub Pages static site under `site/` has a root landing page, `/demo/` interactive localStorage-backed demo, and `/docs/` placeholder.
- The static demo supports light and dark modes plus corporate, rustic, blocky, and showcase themes.
- The static demo supports demo user selection, mock provider sign-in buttons, local task workflow edits, and a visible clear-state control.
- GitHub Actions has a Pages workflow that publishes `site/` on pushes to `main` and manual dispatch.
- The static demo uses separate pages for overview, discovery, requester workflow, review queue, API/MCP instructions, and demo settings.
- Demo login starts from a Viewing as account control in the top-right corner and opens a discrete login panel with demo users and mock provider choices.
- The static demo uses delegated event handlers, debounced localStorage writes for text input, bounded locally-created task/reservation/submission state, and an opaque non-sticky top bar.
- The static demo requester flow includes title, description, reward, visibility, participation policy, and reservation expiry fields.
- The static demo review flow exposes per-reservation approve, decline, and release controls plus per-submission request-changes, reject, accept, partial payout, tip, and ban controls.
- The Elm build helper rejects the recursive npm Elm wrapper when `ELM_BIN` points to it, so local builds fail fast instead of hanging or flooding warnings.
- The static demo uses Command, Mission Board, Post Mission, Review Queue, Uplink, and Settings pages.
- The static demo mission board groups seeded and local missions into Available, Reserved, Awaiting approval, Submitted, and Settled lanes.
- The static demo seeds varied public and organization missions with open submissions, reservations, approval, submitted work, changes requested, rejection, acceptance, credit rewards, collectible rewards, and bundled rewards.
- The static demo persona picker changes the active persona, default page, selected mission, and available actions.
- The static demo tracks local activity, balances, collectible inventory, mission timelines, review drafts, and mission transitions in localStorage.
- The static demo review queue is scoped to requesters and organization reviewers; implementor personas see an empty permission-appropriate queue.
- The static demo mission board and review pages use the visible filtered task for actions so controls affect the briefing shown to the user.
- Static demo request-changes decisions no longer transfer payout or tip credits.
- Static demo cards expose persona-specific next-action chips, requester or assignee context, readable schema blocks, and accessible pressed/current states.
- Static demo localStorage reads and writes are guarded, normalized, and bounded before merging browser state into the seed state.
- The static demo uses Dashboard, Tasks, Post Task, Reviews, Agent/API, and Settings labels.
- The static demo Tasks page renders a scannable task list with status, reward, requester or assignee context, and row-level action widgets.
- Opening a task from the demo task list navigates to a separate Task Detail page instead of changing an inline detail pane.
- The static demo screenshot helper captures task-list and task-detail desktop and mobile states.

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

Last observed checks for pull request 19:

- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 go test -tags http_e2e ./tests/http_e2e` passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 go test -tags integration ./tests/integration` passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui` passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-format check-contracts check-policy check-ts check-copy-paste check-dead-code lint vet test-deno` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build` passed.
- GitHub CI passed before pull request 19 was merged: static checks, unit tests, build, integration tests, HTTP end-to-end tests, and Playwright user interface end-to-end tests.

Last observed checks for pull request 21:

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
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys tools/capture_demo_screenshots.ts` captured desktop and mobile screenshots for corporate light, blocky dark, rustic light, and showcase dark demo states.
- GitHub CI passed before pull request 21 was merged: static checks, unit tests, build, integration tests, HTTP end-to-end tests, and Playwright user interface end-to-end tests.

Last observed checks on `task/demo-ui-ux-repair`:

- `make check-format` passed.
- `make check-ts` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `make test-deno` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-contracts check-dead-code lint vet` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui` passed with local Postgres access.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys tools/audit_demo_ui.ts` passed for deployed and local demo pages with no console warnings, console errors, page errors, failed requests, or horizontal overflow.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys tools/capture_demo_screenshots.ts` captured desktop and mobile screenshots for corporate light, blocky dark, rustic light, and showcase dark demo states.

Last observed checks on `task/demo-performance-flow-review`:

- `make check-format` passed.
- `make check-ts` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `make test-deno` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build go test ./...` passed.
- `make vet` passed.
- `make check-contracts` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build make check-dead-code` passed.
- `make lint` passed.
- `ELM_BIN=/opt/homebrew/bin/elm make build` passed with normal Go module cache access.
- `make e2e-ui` passed.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys tools/audit_demo_ui.ts` passed for deployed and local demo pages with no console warnings, console errors, page errors, failed requests, or horizontal overflow.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys tools/capture_demo_screenshots.ts` captured desktop and mobile screenshots for overview, discovery, requester create, review, API/MCP, settings, blocky dark, rustic light mobile, and showcase dark mobile states.
- `docker compose up -d postgres` confirmed local Postgres was running.
- `SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=migrations make migrate-up` passed.
- `SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build make test-integration` passed.
- `SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build make test-http` passed.

Last observed checks on `task/demo-game-like-personas`:

- `node --check site/demo/app.js` passed.
- `make check-format` passed.
- `make check-ts` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `make test-deno` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build go test ./...` passed.
- `make vet` passed.
- `make lint` passed.
- `make check-contracts` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build make check-dead-code` passed.
- `ELM_BIN=/opt/homebrew/bin/elm make build` passed.
- `make e2e-ui` passed.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys tools/audit_demo_ui.ts` passed for deployed and local demo pages with no reported console warnings, console errors, page errors, failed requests, or horizontal overflow.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys tools/capture_demo_screenshots.ts` captured desktop and mobile screenshots for Command, Mission Board, Post Mission, Review Queue, Uplink, Settings, blocky dark, rustic light mobile, and showcase dark mobile states.
- `SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build make test-integration` passed.
- `SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build make test-http` passed.

Last observed checks on `task/demo-ui-polish-pass`:

- `node --check site/demo/app.js` passed.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys npm:@playwright/test@1.61.0 test -c tests/playwright/playwright.config.ts tests/playwright/demo_static.spec.ts` passed.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys tools/capture_demo_screenshots.ts` captured desktop and mobile demo screenshots.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys tools/audit_demo_ui.ts` passed for deployed and local demo pages with no reported console warnings, console errors, page errors, failed requests, or horizontal overflow.
- `make check-format` passed.
- `make check-ts` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `make test-deno` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-contracts check-dead-code lint vet` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui` passed.

Last observed checks on `task/demo-list-detail-navigation`:

- `node --check site/demo/app.js` passed.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys npm:@playwright/test@1.61.0 test -c tests/playwright/playwright.config.ts tests/playwright/demo_static.spec.ts` passed.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys tools/capture_demo_screenshots.ts` captured desktop and mobile screenshots, including task-list and task-detail states.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys tools/audit_demo_ui.ts` passed for deployed and local demo pages with no reported console warnings, console errors, page errors, failed requests, or horizontal overflow.
- `make check-format` passed.
- `make check-ts` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `make test-deno` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-contracts check-dead-code lint vet` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui` passed.

Last observed checks on `task/multi-page-routing`:

- `make check-format`, `make check-contracts`, `make check-policy`, `make check-ts`, `make check-copy-paste`, `make check-dead-code`, `make lint`, `make vet`, `make test`, and `make test-deno` passed.
- `make build` and `make frontend` passed.
- `make test-integration`, `make test-http`, and `make e2e-ui` passed with local Postgres access.

Last observed checks on `task/teams-org-context-collectible-ui`:

- `make check-format`, `make check-contracts`, `make check-policy`, `make check-ts`, `make check-copy-paste`, `make check-dead-code`, `make lint`, `make vet`, `make test`, and `make test-deno` passed.
- `make build` and `make frontend` passed.
- `make test-integration`, `make test-http`, and `make e2e-ui` passed with local Postgres access.

Last observed checks on `task/full-review-improvements`:

- `make check-format`, `make check-contracts`, `make check-policy`, `make check-ts`, `make check-copy-paste`, `make check-dead-code`, `make lint`, `make vet`, `make test`, and `make test-deno` passed.
- `make build` and `make frontend` passed.
- `make test-integration`, `make test-http`, and `make e2e-ui` passed with local Postgres access.

Blocking issues:

- None known.

See [PLAN.md](./PLAN.md) for the product and architecture plan.
See [DO_NEXT.md](./DO_NEXT.md) for the next tasks.
See [BUGS.md](./BUGS.md) for known bugs and risks.
