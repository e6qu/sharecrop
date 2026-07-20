# Sharecrop Project Plan

## Product Thesis

Sharecrop is a coordination layer where people, organizations, teams, scripts, and local AI agents can discover requested work, reserve or request approval for work when required, submit responses, and receive rewards when a requester accepts or partially rewards a result.

The platform does not execute tasks itself. It provides the web UI, HTTP API, MCP interface, task discovery, response validation, submission tracking, scoped access tokens, escrow accounting, and payout workflow. Work happens outside the platform.

## Core Product Decisions

- Sharecrop serves both a public marketplace and private organization/team workflows.
- Sharecrop is not a managed task runner.
- REST/curl and MCP are both first-class interfaces.
- A task may receive any number of submissions when its participation policy allows them.
- Some tasks require an exclusive reservation or requester approval before submission.
- A task may have at most one active assignee: one user or one team.
- Only one submission can become the accepted result, but rejected work may receive a requester-selected partial reward.
- Requesters can manually accept submissions.
- Requesters can request changes, reject work, decline or cancel reservations, ban an implementor from a task, pay partial rewards, and add tips from their own balance or inventory.
- Auto-accept can exist later, but must be constrained by validation, escrow, limits, and owner policy.
- Anonymous workers are deferred until the anonymous worker identity and payout model is redesigned.
- Task responses can be Sharecrop-schema validated or freeform.
- Task series are ordered lists of tasks.
- Tasks can be public or scoped to specific users, teams, organizations, organization users, or organization teams.
- Tasks posted inside an organization are organization-scoped by default and hidden from the public marketplace unless an authorized organization role publishes them.
- Users and organizations can hold Sharecrop credit accounts.
- Each new registered user receives 100 Sharecrop credits.
- Sharecrop credits are represented through an append-only ledger, not as a mutable balance field alone.
- Task rewards are bundles that may contain Sharecrop credits, collectibles, both, or neither.
- Sharecrop may sell platform-issued collectibles such as special emojis, graphics, badges, or other reward items.
- Sharecrop collectibles are minted by platform admins only. User/org/per-project tokens, external wallets, and crypto integrations are out of scope.

## Software Stack

### Runtime

- Go backend.
- One deployable Go binary for the website, API, static assets, and, if practical, MCP server.
- A Go/WASM backend build is a first-class production execution target. It must
  be compiled from Go code with explicit host adapters for storage, clock,
  identity/session, networking/request handling, and other runtime boundaries.
  JavaScript reimplementations, generated fake backends, and fallback stores are
  not acceptable substitutes. This is realized: production hosts the app as a
  `wasip1` guest under a wazero pool, and the browser demo runs the same app as
  `js/wasm`. See [docs/deployment.md](./docs/deployment.md).
- Standard library `net/http`; no `chi` or larger web framework initially.
- Go `embed` for compiled frontend assets and static files.
- No type-ignore comments or intentional lint/type-safety escapes in application code.

### Data

- PostgreSQL is the server source of truth.
- Plain SQL migrations checked into the repo.
- No migration framework initially.
- `pgx` for PostgreSQL access.
- Use `sqlc` for typed query generation where it does not compromise the domain model.
- Domain models remain separate from generated DB row types.
- SQLite (via ncruces) is the browser demo's store engine; PostgreSQL remains the
  central server source of truth. SQLite is not used for the server.
- DuckDB may be useful later for analytics/export workflows, not OLTP.

### Frontend

- Elm for the browser frontend.
- Elm compiles to static JavaScript served by the Go binary.
- Tailwind CSS for styling from the start.
- No React, Next.js, htmx, or JavaScript component framework at the start.
- shadcn/ui's official ready-made component templates do not include Elm, but the shadcn registry/distribution model can work with any framework.
- Sharecrop may use shadcn CLI/registry ideas for distributing local Elm/Tailwind component code if that proves useful.
- Sharecrop should not import React components while Elm is the frontend.
- Sharecrop may use shadcn visual conventions, spacing, color tokens, and component structure implemented in Elm/Tailwind.
- Elm API/domain contract types are generated from Go contract definitions.
- Handwritten Elm view/form models are still used for UI-only and partially-valid states.
- The UI is an API client, not a privileged application layer.
- The frontend must be swappable without changing domain services or core API contracts.

### Auth And Tokens

- JWT access tokens.
- Opaque rotating refresh tokens stored server-side as hashes.
- Refresh token reuse should revoke the token family.
- Browser auth should use secure, HttpOnly cookies.
- Shauth OpenID Connect is an additional browser sign-in provider. External
  identities are keyed by verified issuer and subject; new external identities
  never attach themselves to an existing password account based only on email.
- Shauth sessions retain the provider-signed ID token and optional session ID
  server-side for RP-Initiated Logout. Signed Back-Channel Logout revokes
  matching refresh-token families atomically and stores replay claims durably.
- Relying-party logout returns to an app-local, reload-safe signed-out page.
  That page never starts authentication automatically and exposes one explicit
  same-origin `Sign in with Shauth` recovery control in accessible light and
  dark presentations. Provider-initiated logout invalidates the same local
  session and access-token family.
- Agent credentials are separate from user session auth.
- Task/capability tokens are opaque random tokens linked server-side to tasks, scopes, and optional subjects.

### Validation

- Use a local Sharecrop schema parser and validator built from Sharecrop domain types.
- Do not depend on an external JSON Schema validator.
- App/domain validation happens through explicit constructors and typed command objects.
- API DTOs are raw boundary types and must be converted into domain types before business logic runs.
- Task response schemas may include Sharecrop sensitivity metadata for fields containing PII or other sensitive data.
- Sensitive-field metadata must be persisted so those fields can be located, redacted, censored, exported selectively, or deleted later.

### Dependency Segregation

Third-party dependencies must be contained behind small boundary packages.

Rules:

- Third-party types should not leak into domain packages.
- Weak third-party APIs that expose `any`, maps, booleans, nils, or stringly values must be wrapped immediately.
- Boundary packages convert dependency values into Sharecrop strong types.
- Domain packages depend on Sharecrop types, not dependency types.

Segregation boundaries:

- `internal/db`: the single engine-neutral store implementation; `pgx` is confined
  here (in `handle.go`), serving PostgreSQL in production and SQLite in the demo.
- `internal/auth`: contains JWT and Argon2id library usage.
- `internal/core/id`: contains UUID library usage.
- `internal/schema`: contains local Sharecrop schema parsing and validation.
- `internal/http`: contains raw HTTP and JSON decoding/encoding.

### Contract Generation

- Use Go contract definitions as the source of truth for Elm generation.
- Prefer an explicit Go contract DSL over reflection over arbitrary structs.
- The contract DSL must preserve strong product types, strong enums, and tagged unions.
- OpenAPI may be exported later from the same contract definitions, but there is no API versioning requirement yet.

### API-First UI Boundary

Sharecrop is API-first. The Elm UI must consume the same HTTP API and generated contracts that external clients use.

Rules:

- Domain services must not depend on Elm or UI-specific state.
- HTTP API behavior must be complete without the Elm UI.
- UI actions must map to API commands.
- UI read models must come from API responses.
- Browser-only state belongs in handwritten Elm view/form models.
- The UI may be replaced by another frontend without rewriting task, ledger, auth, organization, schema, submission, or MCP services.
- No server-side shortcut should exist only for the current UI.
- API and contract tests are the source of truth for behavior; UI tests verify that the UI correctly exercises that behavior.

Contract rules:

- No bool fields for meaningful domain decisions.
- No untyped string fields for values that have domain meaning.
- No generic object/dict/map fields for known business concepts.
- Use tagged unions for variants.
- Use strong enums for closed sets.
- Use product types for grouped fields.
- Use explicit lifecycle states instead of `deleted_at`, `disabled_at`, `activated_at`, or similar nullable timestamp flags.
- Raw arbitrary JSON is allowed only as an intentionally bounded payload type, such as a submitted response body or task input payload.
- Raw arbitrary JSON must be wrapped in a named domain/contract type and validated before use.

### Local Sharecrop Schema

Sharecrop response schemas should be parsed into local strong domain types instead of being handled as generic JSON Schema maps.

The schema language can begin as a JSON-compatible subset inspired by JSON Schema, but the internal representation must be Sharecrop-owned.

Schema concepts:

- Object schema.
- Array schema.
- String schema.
- Integer schema.
- Decimal-as-string schema.
- Enum schema.
- Literal schema.
- Union schema.
- Optional field as explicit field presence policy.
- Freeform schema.
- Sensitive-field annotation.
- Retention policy.
- Redaction behavior.

Schema parser rules:

- Parse untrusted schema JSON into raw boundary DTOs.
- Convert DTOs into Sharecrop schema product types and tagged unions.
- Reject unsupported schema constructs explicitly.
- Preserve field paths for validation errors and sensitive-field indexing.
- Do not pass schema data around as `map[string]any`.
- Do not model schema booleans as domain booleans.

### Deferred

- No Redis until Postgres-backed jobs/rate limits are insufficient.
- No object storage in the MVP. Small task/submission attachments are stored
  inline with a five-file request limit and a 500 KiB per-file limit.
- No external search service; use Postgres full-text search first.
- No hosted auth library.
- External wallets and crypto payouts are out of scope.
- No Kubernetes requirement.

### Primary Dependencies

Runtime/build:

- Go 1.26.4 or newer compatible patch release.
- PostgreSQL 18.4 or newer compatible minor release.
- Elm 0.19.1.
- `sqlc` v1.31.1 as a build-time generator.
- Tailwind CSS as a frontend build-time styling dependency.
- Playwright as a browser E2E test dependency.
- Use Deno's built-in test runner for Deno tooling.

Go libraries:

- `github.com/jackc/pgx/v5` v5.10.0 for PostgreSQL access.
- `github.com/golang-jwt/jwt/v5` v5.3.1 for JWT handling, contained in `internal/auth`.
- `golang.org/x/crypto` v0.54.0 for Argon2id, contained in `internal/auth`.
- `github.com/google/uuid` v1.6.0 for UUIDv7/UUID parsing, contained in `internal/core/id`.

Not included:

- No external JSON Schema library.
- No web router dependency.
- No ORM.
- No migration framework.
- No Vitest unless a meaningful TypeScript/Vite layer is introduced later.
- No direct React shadcn/ui components unless the frontend stack changes to React.
- shadcn registry/CLI may be used later for Sharecrop-owned Elm/Tailwind component distribution.

## Modeling Philosophy

Sharecrop should model the domain aggressively with strong types so that large classes of bugs are impossible or difficult to represent.

### Principles

- Domain models are the center of the application.
- DB rows, API DTOs, and domain objects are separate.
- Avoid magic strings and magic numbers.
- Avoid relying on Go zero values as meaningful defaults.
- Prefer explicit constructors over exported mutable structs.
- Prefer unexported fields with validated constructors.
- Prefer product types for required grouped values.
- Prefer tagged unions/sum types for real variants.
- Prefer strong enums over raw strings.
- Limit optional fields; absence should be explicit and domain-meaningful.
- Avoid `nil` in domain code.
- Do not use `any`.
- Do not use type ignores, intentional lint escapes, or intentionally weak typing.
- Do not use stringly typed domain values.
- Do not use dictly typed domain values such as generic maps for meaningful business concepts.
- Avoid booleans in domain and contract models; use strong enums, tagged unions, explicit commands, or separate methods for separate intentions.
- Avoid nullable timestamp fields that secretly encode lifecycle state such as activated, deleted, revoked, or disabled.
- Model lifecycle through explicit states and transition events.
- Timestamps should record facts or event times, not act as hidden status flags.
- Do not add fallback behavior unless it is explicitly required for reliability behavior such as retry/backoff.
- Do not add workarounds, fake behavior, false behavior, disabled tests, or quick fixes instead of fixing the underlying issue.
- Ask before introducing fallback behavior.
- Prefer explicit `Result`-style construction and service outcomes, even though this is not idiomatic Go.

Low-level dependencies may expose booleans, nils, or generic data. Those must be contained at boundaries and converted into Sharecrop domain types before application logic runs.

### Strong Product Types

Use structs with required, validated fields for domain values:

- `TaskTitle`
- `TaskInstructions`
- `CreditAmount`
- `WalletAddress`
- `JSONSchemaDocument`
- `TaskInputPayload`
- `SubmissionPayload`
- `AccessToken`
- `RefreshToken`
- `TaskCapabilityToken`
- `SubmissionReceiptToken`

No domain code should pass plain strings for these concepts once parsed.

### Result And Option Types

Because `any` is not allowed, Sharecrop should not use generic `Result[T]` or `Option[T]`.

Instead:

- Use explicit result types for domain constructors and operations.
- Use explicit optional/sum types where absence is meaningful.
- Prefer tagged unions that name the absence case over a generic optional value.
- Keep result/option helpers handwritten or generated per type if repetition becomes a problem.

Examples:

- `TaskTitleResult`
- `CreateTaskResult`
- `AcceptSubmissionResult`
- `TaskSeriesPlacement`: no series or placed in series.
- `ExpirationPolicy`: no expiration or expires at instant.

### Strong Enums

Use unexported-value wrappers or equivalent strong enum patterns for:

- `TaskStatus`
- `SubmissionStatus`
- `RewardStatus`
- `TaskVisibility`
- `VisibilityScope`
- `OrganizationRole`
- `TeamRole`
- `LedgerEntryKind`
- `CapabilityScope`
- `ResponseMode`
- `WalletNetwork`
- `RewardAsset`
- `CollectibleKind`
- `TokenIssuerKind`
- `TransferPolicy`
- `DataSensitivity`

Raw database/API strings are parsed into strong enums at the boundary.

### Tagged Unions

Use sealed interfaces with unexported marker methods for variants such as:

- `OwnerRef`
- `SubmitterRef`
- `VisibilityScope`
- `RewardBundle`
- `ParticipationPolicy`
- `AssigneeScope`
- `ReservationState`
- `ReviewOutcome`
- `ResponseSpec`
- `PayoutTarget`
- `TaskState`
- `CapabilitySubject`
- `LedgerReference`

Examples of intended variants:

- `OwnerRef`: user, team, organization, organization team.
- `SubmitterRef`: registered user, agent credential.
- `VisibilityScope`: public, scoped users, scoped teams, scoped organizations, scoped organization users, scoped organization teams.
- `RewardBundle`: empty reward, Sharecrop credits, Sharecrop collectibles, or Sharecrop credits plus collectibles.
- `RewardAsset`: Sharecrop credits and admin-minted Sharecrop collectibles.
- `ParticipationPolicy`: open submissions, reservation required, requester approval required.
- `AssigneeScope`: one user, one public Sharecrop team, one organization team in the same organization.
- `ReservationState`: requested, active, declined, cancelled by requester, cancelled by worker, expired, submitted.
- `ReviewOutcome`: accept, request changes, reject with partial reward, reject without reward.
- `ResponseSpec`: Sharecrop schema response, freeform response.
- `PayoutTarget`: user credit account, organization credit account, team payout target where supported.

### State Modeling

Avoid one partially-filled `Task` struct that represents every possible state.

Prefer explicit states where useful:

- `DraftTask`
- `OpenTask`
- `ClosedTask`
- `CancelledTask`
- `ExpiredTask`

Only valid state transitions should be available as methods/functions. For example, accepting a submission should require an open task and an acceptable submission.

## Go-To-Elm Contract Generation

Go is the source of truth for the API/domain contract consumed by Elm.

The generator should not inspect arbitrary domain structs. Instead, the server defines an explicit Go contract layer that maps from domain objects into wire-safe DTOs. That contract layer is then used to generate Elm types, decoders, and encoders.

Flow:

```text
Go domain model
  -> explicit Go contract/DTO mapping
  -> generated Elm contract modules
  -> handwritten Elm view/form models where needed
```

Generated Elm lives under:

```text
web/elm/src/Sharecrop/Generated/
```

Rules:

- Generated Elm is read-only.
- Generated Elm should be idiomatic Elm, not stringly typed records.
- Generated Elm should use custom types for tagged unions.
- Generated Elm should use records for product types.
- Generated Elm should include JSON decoders.
- Generated Elm should include JSON encoders where the client sends that type back to the server.
- The Elm app can use generated API/read-model types directly.
- JSON uses explicit tagged-union objects.
- JSON tags and fields should use snake_case.
- Elm constructors should use normal Elm-style PascalCase.
- Elm fields should use normal Elm-style camelCase.
- OpenAPI may be exported later from the same contract layer, but there is no API versioning requirement yet and Elm quality takes priority.

Direct generated-type usage is appropriate for:

- API responses.
- API request payloads that are already valid.
- Task read models.
- Submission read models.
- Ledger read models.
- Organization/team read models.
- Capability-token read models.

Handwritten Elm models are still required for:

- Form fields that may temporarily contain invalid input.
- Draft task creation flows.
- Schema builder state.
- Field focus, dirty flags, expanded panels, selected tabs, and loading states.
- Optimistic UI state.
- View-specific grouping such as task cards, ledger rows, or review panels.

This preserves strong server/client symmetry without compromising Elm's ability to model interactive UI state cleanly.

## Main Actors

### Requesters

Requesters create tasks, fund rewards, review submissions, and accept one response.

A requester may act as:

- An individual user.
- A team.
- An organization.
- An organization-owned team.

### Workers

Workers discover tasks and submit responses.

A worker may be:

- An authenticated user.
- A local AI agent acting through MCP/API credentials.
- A script or curl client.
- A public Sharecrop team or an organization team acting through an eligible member where the task allows team assignment.

### Organizations And Teams

Organizations own users, organization teams, credit balances, tasks, task series, and policy settings.

Teams group users. A team may be standalone or owned by an organization.

## Domain Areas

### Users

Users can authenticate, create tasks, submit work, join teams and organizations, and hold a Sharecrop credit account.

Signup behavior:

- Create user.
- Create user credit account.
- Insert ledger entry: `signup_grant +100`.

### Organizations

Organizations can own:

- Organization users.
- Organization teams.
- Tasks.
- Task series.
- Credit accounts.
- Collectibles.
- Agent credentials.
- Policy settings.

Organization users and organization teams are under the control and provisioning of that organization. The organization can invite, provision, deactivate, assign roles, and remove organization-scoped access without relying on the personal account owner to manage those permissions manually.

Initial organization roles:

- Owner.
- Admin.
- Member.
- Billing.
- Reviewer.
- Public publisher.

Organization permissions should include:

- Create organization task.
- Review organization submissions.
- Manage organization billing/credits.
- Publish organization task publicly.
- Switch organization task visibility.

### Teams

Teams can be standalone or organization-owned.

Organization teams are teams whose owner is an organization. They should still be modeled explicitly because permissions and task ownership depend on whether the team belongs to an organization.

### Credit Accounts And Ledger

Users and organizations can own Sharecrop credit accounts.

Ledger entries should be append-only and typed:

- Signup grant.
- Task escrow.
- Task refund.
- Task payout.
- Manual adjustment.

Balances are derived from ledger entries.

Credit amounts must be integer base units. No floats.

### User And Organization Tokens

User/org/per-project reward tokens are out of scope. Sharecrop collectibles are minted by platform admins only.

Collectible ownership:

- Users can hold Sharecrop platform collectibles.
- Organizations and teams can hold Sharecrop platform collectibles awarded by platform admins.

Token rules:

- Minting policy is controlled by the issuer.
- Organization token minting is controlled by organization permissions.
- Tokens use integer base units.
- Token balances are ledger-backed.
- Tokens can be attached as task rewards when the task owner has sufficient balance and permission.
- Token transferability varies by explicit token policy.
- Token policy is modeled as variants, not booleans.

Initial token transfer policies:

- Non-transferable except task payout.
- Transferable between users.
- Transferable within organization.
- Issuer-controlled transfer.

### Platform Collectibles

Sharecrop may sell or issue platform collectibles.

Examples:

- Special emojis.
- Graphics.
- Badges.
- Seasonal items.
- Limited-edition reward artifacts.

Collectible rules:

- Collectibles are ownable assets.
- Collectibles may be unique or fungible editions.
- Collectibles can be hoarded, traded, or attached as task rewards depending on policy.
- Collectible policy is explicit and strongly typed.
- The platform tracks ownership and transfer history.
- Collectibles can later be sold by Sharecrop for credits or other non-crypto platform payment methods.

Collectible states:

- Draft.
- Available.
- Held.
- Listed for trade.
- Escrowed as task reward.
- Awarded.
- Retired.

### Task Series

A task series is an ordered list of related tasks.

Series ownership can be:

- User.
- Team.
- Organization.
- Organization team.

The first version treats a series as a list, not a dependency graph or workflow engine.

### Tasks

Tasks contain:

- Owner.
- Optional series placement.
- Visibility scope.
- Instructions.
- Optional input payload.
- Response specification.
- Sensitive-field policy.
- Reward bundle.
- Acceptance policy.
- Participation policy.
- Assignee scope.
- Reservation expiry policy.
- Expiration policy.

Visibility variants:

- Public.
- Scoped users.
- Scoped teams.
- Scoped organizations.
- Scoped organization users.
- Scoped organization teams.

Visibility rules:

- Tasks posted by an individual user may be public or scoped to selected users, teams, organizations, organization users, or organization teams where the requester has permission.
- Tasks posted by a standalone team may be public or scoped to that team by default.
- Tasks posted inside an organization default to organization-only visibility.
- Organization tasks are hidden from the public marketplace unless explicitly published.
- Publishing an organization task publicly requires an organization role with public-post permission.
- Switching an organization task from organization-scoped to public requires the same permission.
- Organization tasks may also be scoped to specific organization users or organization teams.

Response variants:

- Sharecrop schema response.
- Freeform response.

Response schemas may mark fields with Sharecrop-specific sensitivity metadata. This supports fields such as:

- PII.
- Credentials.
- Payment details.
- Wallet-adjacent identity data.
- Private organization data.
- User-defined sensitive categories.

Sensitivity metadata should support:

- Field path.
- Sensitivity category.
- Retention policy.
- Redaction behavior.
- Deletion eligibility.
- Viewer permissions.

The server should build and store a sensitive-field index for submitted responses when possible. That index allows the platform to retrieve, redact, censor, export, or delete sensitive fields without relying on ad hoc payload inspection later.

Reward bundles:

- Empty bundle: no declared reward.
- Sharecrop credit amount.
- One or more platform collectibles.
- Sharecrop credits plus platform collectibles.

Participation policies:

- Open submissions: any visible registered user can submit while the task is open.
- Reservation required: a worker or eligible team must reserve the task before submitting.
- Requester approval required: a worker or eligible team requests approval, and the requester approves exactly one active assignee before submission.

Assignee scope:

- One user.
- One public Sharecrop team.
- One organization team in the same organization as an organization-owned or organization-visible task.

Reservation defaults and rules:

- Default reservation expiry is 48 hours.
- Requesters may choose a different expiry duration.
- Expiry automatically releases the task for other workers.
- A task may have at most one active reservation.
- A reservation keeps the task exclusive for the assignee.
- Requesting changes keeps the reservation active and exclusive.
- Requester cancellation or rejection without a continuing revision path releases the task unless the task is closed.
- Requesters may ban a user or team from the same task after rejection or cancellation.
- Task-local bans do not ban the worker globally.

Discovery rules:

- Reserved tasks are hidden from default discovery for other workers.
- Discovery can include reserved tasks when the client passes an explicit include-reserved option.
- Reserved tasks remain visible and actionable to the active assignee.
- Requesters can see their own reserved and pending-approval tasks.

Task lifecycle:

- Draft.
- Open.
- Closed.
- Cancelled.
- Expired.

### Submissions

A task may receive any number of submissions.

Submission submitter variants:

- Registered user.
- Agent credential.

Submission lifecycle:

- Received.
- Schema invalid.
- Valid.
- Changes requested.
- Rejected.
- Accepted.
- Superseded.

Rules:

- Only one submission can become accepted.
- Acceptance must be transactional.
- Accepting one submission closes reward-bearing acceptance for the task.
- Later submissions may exist when the requester requests changes from the same assignee.
- Requesting changes requires requester notes.
- Rejected work can receive a requester-selected partial reward only after a submitted response exists.
- Requesters can tip from current balance and inventory at review time; tips are not pre-funded escrow.
- Partial collectible rewards may award any subset of escrowed collectibles plus optional extra collectibles from requester inventory.
- Extra credit tips are paid from the requester balance at review time.
- Submitted responses should record any detected sensitive field paths based on the task response schema.

### Sensitive Data Handling

Sharecrop must support schemas that identify sensitive fields, including PII fields.

Goals:

- Find sensitive fields in submitted responses.
- Redact sensitive fields for viewers who lack permission.
- Censor sensitive fields in UI/API output when required.
- Delete sensitive fields without necessarily deleting the entire submission.
- Support retention policies by sensitivity category.
- Audit sensitive-data access and deletion.

Schema metadata:

- Use the local Sharecrop schema parser and validator.
- The wire representation may be JSON-compatible and inspired by JSON Schema.
- Add Sharecrop-specific sensitivity metadata as first-class parsed schema data.

Example extension shape:

```json
{
  "type": "string",
  "x-sharecrop-sensitive": {
    "category": "pii",
    "redaction": "mask",
    "retention": "delete_on_request"
  }
}
```

Sensitive data categories should be strong enums in the domain model, not raw strings.

Initial categories:

- PII.
- Credential.
- Payment.
- Wallet identity.
- Organization private.
- User defined.

Deletion/redaction behavior should be explicit:

- Mask.
- Omit.
- Hash.
- Delete.
- Reveal only to owner/reviewer.

For freeform responses, sensitivity handling is weaker because field paths are unavailable. Freeform tasks may still require manual requester tagging or whole-response sensitivity classification.

### Capability Tokens

Tasks can have infinitely many opaque capability tokens.

The raw token must not encode the task ID or other sensitive data. It should be high-entropy random data. The server stores only a hash and links the token row to the task.

Capability token fields:

- Token hash.
- Task ID.
- Task series placement.
- Scope.
- Subject scope.
- Label policy.
- Usage policy.
- Use count.
- Expiration policy.
- Lifecycle state.
- Created event time.
- Last-used event time.

Capability token lifecycle states:

- Active.
- Exhausted.
- Expired.
- Revoked.

Revocation is modeled as a lifecycle transition plus an audit event, not as a nullable timestamp field.

Capability scopes include:

- Task view.
- Task submit.
- Task review.
- Submission accept.
- Task series continue.
- Submission status.
- Agent install.

Use cases:

- Public task submit links.
- Private task invite links.
- Deferred anonymous submission receipts.
- Task-series continuation.
- MCP agent task access.
- Limited-use review links.

## Reward And Escrow Flow

### Sharecrop Credits

Sharecrop credits are the preferred MVP reward mechanism.

Flow:

1. Requester creates a task with a reward bundle that may include credits.
2. Requester funds the credit portion from a user or organization credit account when credits are declared.
3. The platform inserts task escrow ledger entries.
4. Task becomes open.
5. Workers submit directly, reserve, or request approval depending on the task participation policy.
6. Requester accepts one valid submission, requests changes, rejects it, or rejects it with a partial reward.
7. Platform inserts task payout ledger entries for accepted, partial, or tip credit payouts.
8. Task reward status records paid, partially paid, refunded, or retained reward outcomes.

Cancellation:

- If an escrowed task is cancelled before acceptance, insert refund ledger entries.

### Sharecrop Collectibles As Rewards

Sharecrop platform collectibles and Sharecrop credits are the product-scope reward assets. Collectibles are minted by platform admins only.

Reward flow should mirror credits:

1. Requester selects a Sharecrop credit or collectible reward item as part of the reward bundle.
2. Platform verifies ownership, transfer policy, and requester permission.
3. Platform escrows the asset.
4. Task becomes open.
5. Requester accepts one submission or rejects with a partial reward.
6. Platform transfers the requester-selected subset of escrowed assets, and optional extra assets from requester inventory, to the worker or team holding account.

Collectible rewards may be:

- A specific unique item.
- One edition from a fungible collection.
- A bundle of collectibles.

Collectible transfers must be ledger/event-backed and auditable.

### Auto-Accept

Auto-accept should require:

- Escrowed reward.
- Passing schema validation for Sharecrop-schema tasks.
- Owner-level enablement.
- Maximum reward threshold.
- Optional delay window.

## Anonymous Access

Anonymous workers are deferred.

Before anonymous submission returns, Sharecrop needs a design for:

- Anonymous worker identity.
- Receipt-token status access.
- Abuse controls such as rate limits and optional CAPTCHA.

Guest visitors may later receive an opaque browser token for non-account continuity. That guest token is not a wallet or payout identity.

## API Surface

### Authentication

Supported modes:

- Browser session using JWT access token and opaque refresh token cookies.
- API token or agent credential for MCP/local agents.
- Deferred: anonymous task capability tokens.
- Deferred: anonymous submission receipt tokens.

### Tasks

- `GET /api/tasks`
- `GET /api/tasks/{task_id}`
- `POST /api/tasks`
- `PATCH /api/tasks/{task_id}`
- `POST /api/tasks/{task_id}/open`
- `POST /api/tasks/{task_id}/cancel`

Filters:

- Visibility.
- Visibility scope.
- Owner.
- Series.
- Status.
- Reward asset.
- Reward type.
- Response mode.
- Availability.
- Include reserved tasks.
- Participation policy.

### Task Series

- `GET /api/task-series`
- `GET /api/task-series/{series_id}`
- `POST /api/task-series`
- `PATCH /api/task-series/{series_id}`
- `POST /api/task-series/{series_id}/tasks`

### Submissions

- `GET /api/tasks/{task_id}/submissions`
- `POST /api/tasks/{task_id}/submissions`
- `GET /api/submissions/{submission_id}`
- `POST /api/submissions/{submission_id}/accept`
- `POST /api/submissions/{submission_id}/reject`
- `POST /api/submissions/{submission_id}/request-changes`

Deferred anonymous:

- `POST /api/public/tasks/{task_id}/submissions`
- `GET /api/public/submissions/status/{receipt_token}`

### Reservations And Approval

- `POST /api/tasks/{task_id}/reservations`
- `GET /api/tasks/{task_id}/reservations`
- `POST /api/tasks/{task_id}/reservations/{reservation_id}/approve`
- `POST /api/tasks/{task_id}/reservations/{reservation_id}/decline`
- `POST /api/tasks/{task_id}/reservations/{reservation_id}/cancel`
- `POST /api/tasks/{task_id}/bans`

### Rewards And Ledger

- `GET /api/credit-accounts/{account_id}/balance`
- `GET /api/credit-accounts/{account_id}/ledger`
- `POST /api/tasks/{task_id}/fund`
- `POST /api/tasks/{task_id}/refund`
- `GET /api/collectibles`
- `POST /api/collectibles`
- `POST /api/collectibles/{collectible_id}/trade`

### Organizations And Teams

- `GET /api/organizations`
- `POST /api/organizations`
- `GET /api/organizations/{organization_id}`
- `PATCH /api/organizations/{organization_id}`
- `GET /api/organizations/{organization_id}/users`
- `POST /api/organizations/{organization_id}/users`
- `GET /api/organizations/{organization_id}/teams`
- `POST /api/organizations/{organization_id}/teams`
- `GET /api/teams/{team_id}`
- `PATCH /api/teams/{team_id}`
- `POST /api/teams/{team_id}/users`

## MCP Surface

The Sharecrop MCP server should expose task discovery and submission tools for local agents.

Initial MCP tools:

- `sharecrop.list_tasks`
- `sharecrop.get_task`
- `sharecrop.get_task_schema`
- `sharecrop.reserve_task`
- `sharecrop.request_task_approval`
- `sharecrop.cancel_my_reservation`
- `sharecrop.submit_response`
- `sharecrop.get_submission_status`
- `sharecrop.list_task_series`
- `sharecrop.get_task_series`

Authenticated/requester tools:

- `sharecrop.list_my_tasks`
- `sharecrop.list_org_tasks`
- `sharecrop.create_task`
- `sharecrop.create_task_series`
- `sharecrop.list_task_reservations`
- `sharecrop.approve_task_reservation`
- `sharecrop.decline_task_reservation`
- `sharecrop.request_submission_changes`
- `sharecrop.accept_submission`
- `sharecrop.reject_submission`

MCP should be an adapter over the same domain services as the HTTP API.

Do not use a Go MCP library. Implement the MCP protocol handling locally and use the official MCP specification as the source of truth.

Streamable HTTP completion:

- `POST /mcp` requires an `Accept` header that supports `application/json` and `text/event-stream`.
- `POST /mcp` may return a single `application/json` JSON-RPC response or a `text/event-stream` response.
- `POST /mcp` returns `202 Accepted` with no body for accepted JSON-RPC notifications or client responses.
- `GET /mcp` opens a server-sent events stream when the client accepts `text/event-stream`.
- `DELETE /mcp` terminates the current MCP session when the client provides `Mcp-Session-Id`.
- Initialize returns `Mcp-Session-Id`; later HTTP MCP requests must include that session header.
- Unknown or expired MCP sessions return `404`.
- SSE events have event IDs.
- `Last-Event-ID` supports replay for retained events where practical.
- SSE notifications can include task, reservation, and submission changes.
- The server must not broadcast the same MCP message across multiple concurrent streams.

## UI Plan

Primary pages:

- Pending Work.
- My Tasks.
- Task Series.
- Organizations.
- Teams.
- Submissions.
- Credits.
- Wallets.
- Agent Setup.

Important flows:

- Browse pending public and scoped work.
- View task detail.
- View task availability and reservation status.
- Include reserved tasks in discovery with an explicit checkbox.
- Reserve a task or request approval where required.
- Submit response.
- Create task.
- Configure reward bundles containing credits, collectibles, both, or neither.
- Configure participation policy, assignee scope, and reservation expiry.
- Scope task visibility to users, teams, organizations, organization users, or organization teams.
- Publish organization task publicly when permitted.
- Edit Sharecrop schema directly.
- Mark schema fields as sensitive/PII.
- Later: schema builder UI.
- Review submissions.
- Review reservations and approval requests.
- Accept one submission.
- Request changes with required notes.
- Reject work with or without a requester-selected partial reward.
- Ban an implementor from the same task.
- Fund task with credits.
- Fund task with collectibles.
- Fund task with user/org tokens later.
- View ledger.
- Create shareable task capability links.
- Configure local agent/MCP.
- Copy curl examples.
- Show REST and MCP instructions on every task detail page for worker result submission.

## Security And Correctness Requirements

Required early:

- Token hashing for refresh tokens and capability tokens.
- Rotating refresh tokens.
- Reuse detection for refresh token families.
- Payload size limits.
- Sharecrop schema validation timeouts.
- API/agent credential scopes.
- Authorization checks for visibility scope changes.
- Organization-role checks before public publishing of organization tasks.
- Audit events for task creation, funding, acceptance, rejection, and payout.
- Idempotency keys for funding, acceptance, and payout.
- Transactional acceptance so only one submission can win.
- Row-level transaction discipline for ledger and escrow operations.
- No floats for credits or money-like values.
- Sensitive-field indexing for schema-based submissions.
- Redaction before returning sensitive submission data to unauthorized viewers.
- Audit events for sensitive-field access and deletion.

Important later:

- CAPTCHA for public web submissions.
- Abuse reporting.
- Organization policy controls.

## Testing Strategy

Sharecrop should be built in a TDD style with a complete test pyramid.

Testing layers:

- Unit tests for domain constructors, strong enums, tagged unions, lifecycle transitions, schema parsing, validation, redaction, token generation, and ledger arithmetic.
- Integration tests for repositories, migrations, Postgres transactions, auth session storage, escrow, acceptance, visibility checks, and sensitive-field indexing.
- HTTP end-to-end tests for the public API, authenticated API, receipt-status API where enabled, and agent/MCP-facing flows.
- Playwright end-to-end tests for browser UI workflows.

TDD expectations:

- Domain behavior starts with tests before implementation where practical.
- New state transitions require tests for allowed and rejected transitions.
- Ledger and escrow changes require invariant tests.
- Visibility and permission changes require positive and negative tests.
- Schema parser changes require parser, validator, and redaction tests.
- Bug fixes should include a failing regression test first.

Test pyramid guidance:

- Most tests should be fast unit tests.
- Integration tests should cover DB and transaction correctness, not every formatting branch.
- HTTP E2E tests should cover complete API workflows and serve as client contract tests.
- Playwright tests should cover critical user workflows, not every visual variant.

Required early test suites:

- `go test ./...` for Go unit and integration tests.
- Elm test suite for non-generated UI logic where practical.
- HTTP E2E test runner against a real test server and test Postgres database.
- Playwright test suite against the compiled Go binary serving the Elm UI.

API-first testing rule:

- Any behavior required by the UI must have API-level coverage before or alongside Playwright coverage.
- Playwright should prove the UI wiring and browser behavior, not be the only proof of domain behavior.

## Repository Shape

Initial shape:

```text
cmd/sharecrop/
  main.go

internal/
  app/
  auth/
  assets/
  capability/
  core/
  db/
  http/
  ledger/
  mcp/
  orgs/
  schema/
  submissions/
  tasks/
  validation/

migrations/

sql/
  queries/

tests/
  http_e2e/
  playwright/

web/
  elm/
    src/
  static/
  styles/
```

Package intent:

- `internal/core`: foundational strong types, result/option helpers, domain errors.
- `internal/db`: the engine-neutral store implementation (PostgreSQL and SQLite), transactions, and query execution behind a small handle abstraction.
- `internal/tasks`: task domain model and services.
- `internal/submissions`: submission domain model and services.
- `internal/ledger`: credit accounts, ledger entries, escrow, payout.
- `internal/assets`: user tokens, organization tokens, collectibles, ownership, transfer policy.
- `internal/schema`: Sharecrop schema parser, schema domain types, response validation, sensitive-field extraction.
- `internal/capability`: opaque scoped token generation and verification.
- `internal/http`: HTTP handlers, DTO parsing, response rendering.
- `internal/mcp`: MCP adapter.

## Implementation Milestones

### Milestone 1: Skeleton And Domain Foundations

- Go binary with `net/http`.
- Elm build embedded into Go binary.
- Tailwind CSS.
- Postgres connection.
- Plain SQL migration runner or manual migration command.
- Strong ID/value types.
- Strong enums and tagged-union patterns.
- Domain error and result patterns.

### Milestone 2: Auth And Accounts

- User registration/login.
- JWT access tokens.
- Opaque rotating refresh tokens.
- User credit account creation.
- Signup grant of 100 credits.
- Basic organization/team models.

### Milestone 3: Tasks And Capability Tokens

- Create task.
- Open/cancel task.
- Public and scoped task listing.
- Scoped task visibility.
- Organization-default-hidden task behavior.
- Organization public-publisher permission.
- Task capability token generation.
- Shareable task links.
- Task series as ordered lists.

### Milestone 4: Submissions And Validation

- Sharecrop schema response mode.
- Sensitive-field schema metadata.
- Sensitive-field indexing for submitted responses.
- Freeform response mode.
- Authenticated submissions.
- Deferred anonymous submissions.
- Receipt-token status checks.
- Submission review.

### Milestone 5: Credits, Escrow, And Acceptance

- Fund task with Sharecrop credits.
- Escrow ledger entries.
- Transactional first-accepted submission enforcement.
- Payout ledger entries.
- Refund cancelled escrowed tasks.
- Ledger UI.

### Milestone 6: MCP And Agent Setup

- MCP task browsing.
- MCP schema retrieval.
- MCP response submission.
- Agent credentials.
- Agent setup page.
- curl examples throughout UI.

### Milestone 7: Organization Workflows

- Organization-owned credit accounts.
- Organization teams.
- Organization task filters.
- Reviewer/billing/admin permissions.

### Milestone 8: Advanced Acceptance

- Auto-accept policies.
- Schema builder UI.
- Schema builder support for sensitive/PII fields.
- Submission caps.

### Milestone 9: Platform Collectibles

- Platform collectible catalog.
- Collectible ownership and trade history.
- Attach Sharecrop collectibles as task rewards.
- Escrow and award collectible rewards.

## Open Questions

Two of the original open questions are settled by the implementation:

- Organization task default visibility: when task creation asks for the
  `default` visibility kind, visibility follows the owner
  (`defaultVisibilityForOwner` in `internal/http/tasks.go`). An
  organization-owned task defaults to organization-wide visibility (all
  organization members); an organization-team-owned task defaults to that
  team. Narrower scoping uses an explicit `organization_team`, `team`, or
  `user` visibility.
- Collectible transferability: transferability is a per-collectible transfer
  policy chosen at mint time (`internal/assets/policy.go`):
  `non_transferable_except_payout` (account-bound outside payout),
  `transferable_between_users`, `transferable_within_organization`, and
  `issuer_controlled`.

Still open, and dormant by product decision:

- What anonymous worker identity and payout model should be used if
  anonymous submissions return. Submissions currently require registered
  users.

## Chosen Implementation Defaults

- GitHub project URL: `https://github.com/e6qu/sharecrop`.
- Canonical SSH remote: `git@github.com:e6qu/sharecrop.git`.
- Go module path: `github.com/e6qu/sharecrop`.
- Local development uses Docker Compose for PostgreSQL.
- Application config is driven by `DATABASE_URL`.
- Amazon ECS service rollouts are controlled by an AWS Step Functions workflow
  that waits for the standalone database migration task before updating the
  service; Terraform never rolls the application service ahead of migrations.
- Shared environments may set a plan-known ownership boolean to reuse an
  existing Amazon API Gateway VPC Link and its paired security group, including
  resource-derived IDs that remain unknown until apply. Standalone deployments
  create a dedicated link by default.
- Task runner is `make`.
- Frontend tool runner is Deno, not npm.
- Elm compiler and Tailwind are invoked through Deno-managed tooling or pinned local tooling without npm.
- Integration tests use one resettable test database per test run at first.
- Initial migration command is `sharecrop migrate up`.
- Local app examples use port `29180`.
- Local PostgreSQL examples use port `25432`.
- Avoid common development ports such as `3000`, `5432`, `8000`, `8080`,
  `15432`, `18080`, and `18081`.
- PR task branches use names such as `task/pr-01-skeleton`.
- Create one pull request for each task.
- Keep at most one pull request open at any time.
- After a task pull request is merged, sync local `main` with `origin/main` before starting the next task.
- CI runs only for pull requests targeting `main`.
- CI does not run on direct pushes to `main` or bare branch pushes.
- Create each new task branch from synced `origin/main`.
- Create one git commit at the end of each task.
- Use Go contract definitions and a Go-based generator for Elm.
- Use `sqlc` from the start where practical.
- Do not use generic `Option[T]` or `Result[T]`.
- Do not use `any`.
- Use handwritten or generated per-type result/option/sum types.
- Use UUIDv7 IDs unless implementation friction is high enough to justify UUIDv4.
- Sharecrop credits are not transferable in the MVP except through accepted task payouts, refunds, signup grants, and manual admin adjustments.
- No-reward tasks may be opened without escrow.
- Reward-bearing tasks must be escrowed before opening.
- User/org tokens and collectibles are not MVP reward assets; design them now and implement after credit escrow is stable.
- Public marketplace browsing does not require a task capability token.
- Shareable scoped links use explicit task capability tokens.
- Guest browser tokens are minted only when needed, not for every passive visitor.
- Sensitive field deletion mutates/redacts stored submission payloads and records audit/tombstone events without preserving the original sensitive value.

## Recommended MVP Scope

Build the MVP around the central task/reward loop and Sharecrop credits:

- Go single binary.
- Elm frontend.
- Postgres.
- Plain SQL migrations.
- Strong domain modeling.
- Users, organizations, teams, organization users, and organization teams.
- 100-credit signup grant.
- Credit ledger and escrow.
- Future user/org token and collectible reward model.
- Public and scoped tasks.
- Task series as ordered lists.
- Sharecrop schema and freeform responses.
- Submissions governed by participation policy.
- One accepted submission receives the final accepted reward.
- Rejected submitted work may receive a requester-selected partial reward.
- Authenticated submissions.
- Deferred anonymous submission identity and payout model.
- Receipt-token status checks where anonymous or capability-token flows require them.
- Opaque task capability tokens.
- REST/curl support.
- MCP support for listing tasks, reading schemas, and submitting responses.

Crypto payout automation and user/org token reward execution should come after the core task, validation, reservation, escrow, review, asset, and ledger models are reliable.

## PR Roadmap

Sharecrop will use large implementation PRs, up to roughly 10,000 lines each. This is intentionally larger than traditional review guidance. To keep those PRs manageable, each PR should have:

- One architectural theme.
- A clear acceptance checklist.
- Generated code separated from handwritten code.
- Domain tests for state transitions and constructors.
- Migration files grouped with the code that uses them.
- No unrelated refactors.

### PR 1: Project Skeleton And Build System

Goal:

- Establish the Go/Elm/Postgres project structure and build loop.

Tasks:

- Create Go module.
- Add `cmd/sharecrop/main.go`.
- Add `internal/app`, `internal/http`, `internal/db`, and `internal/core` packages.
- Add `net/http` server with health endpoint.
- Add config loading from environment.
- Add Postgres connection setup.
- Add plain SQL migration runner command.
- Add initial `migrations/` directory.
- Add Elm project under `web/elm`.
- Add Tailwind build setup.
- Embed compiled frontend/static assets into the Go binary.
- Add Go test conventions for unit and integration tests.
- Add HTTP E2E test harness against a real test server.
- Add Playwright test setup for UI E2E.
- Add `Makefile` or equivalent task runner.
- Add basic CI-style commands: format, unit test, integration test, HTTP E2E, Playwright E2E, build.

Acceptance checks:

- `go test ./...` passes.
- HTTP E2E harness can run a smoke test.
- Playwright can load the app shell.
- Go binary starts locally.
- Health endpoint responds.
- Elm compiles.
- Tailwind output is generated.
- Static assets are embedded and served.

### PR 2: Core Domain Type System

Goal:

- Establish Sharecrop's strong modeling foundation.

Tasks:

- Add strong ID wrappers.
- Add UUIDv7 generation behind `internal/core/id`.
- Add domain error types.
- Add per-type result patterns.
- Add strong enum pattern.
- Add tagged-union pattern examples.
- Add clock/time value wrappers.
- Add lifecycle state/event conventions.
- Add no-bool/no-any/no-dictly domain guidelines as code comments or package docs.
- Add tests for ID parsing, enum parsing, result handling, and lifecycle transitions.
- Write tests before or alongside each domain primitive.

Acceptance checks:

- No `any` in handwritten application/domain code.
- No generic `Result[T]` or `Option[T]`.
- No domain package exposes third-party types.
- Package docs explain modeling conventions.

### PR 3: Auth, Sessions, And Guest Identity

Goal:

- Implement registered user auth, JWT access tokens, opaque refresh tokens, and guest continuity tokens.

Tasks:

- Add users migration.
- Add refresh session migration.
- Add guest subject migration.
- Add password hashing with Argon2id behind `internal/auth`.
- Add JWT creation/verification behind `internal/auth`.
- Add opaque rotating refresh token storage with token hashes.
- Add refresh token family reuse detection.
- Add signup/login/logout/refresh endpoints.
- Add guest token minting only when needed.
- Add browser cookie handling.
- Add user domain types and auth command/result types.
- Add auth middleware that converts boundary auth into domain subject types.
- Add unit, integration, and HTTP E2E tests for auth flows.

Acceptance checks:

- User can register, login, refresh, and logout.
- Refresh token rotation works.
- Reuse of a rotated refresh token revokes the family.
- Guest token is not created for passive health/static requests.
- Sensitive routes require current DB permission checks, not only JWT claims.

### PR 4: Organizations, Teams, And Provisioning

Goal:

- Implement users, teams, organizations, organization users, organization teams, roles, and provisioning rules.

Tasks:

- Add organization migrations.
- Add team migrations.
- Add organization user/team migrations.
- Add organization role and permission domain types.
- Add organization provisioning services.
- Add invite/provision/deactivate/remove flows.
- Add organization public-publisher permission.
- Add API endpoints for orgs, org users, org teams, and teams.
- Add Elm generated contracts and basic UI pages for org/team lists.
- Add unit, integration, HTTP E2E, and Playwright coverage for organization provisioning.

Acceptance checks:

- Organizations can provision and deactivate org users.
- Organizations can create and manage org teams.
- Organization-scoped access is controlled by organization permissions.
- Public publishing permission is represented separately from reviewer/billing roles.

### PR 5: Go-To-Elm Contract Generator

Goal:

- Generate Elm API contract types from Go-owned contract definitions.

Tasks:

- Add `internal/contracts`.
- Add explicit Go contract DSL for product types, enums, and tagged unions.
- Generate Elm modules under `web/elm/src/Sharecrop/Generated/`.
- Generate decoders.
- Generate encoders for client-submitted types.
- Add contract fixtures/golden tests.
- Add HTTP contract tests using generated payloads where useful.
- Add generated-code marker comments.
- Add build step for contract generation.
- Add first generated contracts for auth, users, orgs, teams, errors, and IDs.

Acceptance checks:

- Generated Elm compiles.
- Generated files are deterministic.
- Handwritten Elm can directly use generated read-model types.
- Generated contracts do not expose booleans for domain decisions.
- Generated contracts do not expose generic dicts for known business concepts.

### PR 6: Sharecrop Schema Parser And Validator

Goal:

- Implement the local Sharecrop schema language for task responses and sensitive-field metadata.

Tasks:

- Add `internal/schema`.
- Define schema domain types: object, array, string, integer, decimal string, enum, literal, union, freeform.
- Define field presence policy without generic optional fields.
- Define sensitivity categories, retention policies, and redaction behaviors.
- Parse schema JSON into boundary DTOs.
- Convert DTOs into strong schema domain types.
- Validate response payloads against schema.
- Produce typed validation errors with field paths.
- Build sensitive-field index from schema and submitted payload.
- Add redaction functions for sensitive paths.
- Add tests for parser, validator, redaction, and unsupported constructs.
- Add tests before parser features where practical.

Acceptance checks:

- No external JSON Schema dependency.
- Unsupported schema constructs fail explicitly.
- Sensitive fields can be indexed and redacted.
- Freeform response mode remains possible.

### PR 7: Task Series, Tasks, Visibility, And Capability Tokens

Goal:

- Implement task creation, task series, scoped visibility, and opaque task capability tokens.

Tasks:

- Add task series migrations.
- Add task migrations.
- Add task visibility scope migrations.
- Add capability token migrations.
- Add task owner domain types.
- Add visibility scope tagged unions.
- Add task state types: draft, open, closed, cancelled, expired.
- Add task series placement types.
- Add capability token lifecycle states.
- Add capability token hashing and verification.
- Add task create/open/cancel endpoints.
- Add public and scoped task listing.
- Add organization-default-hidden behavior.
- Add public publishing checks for org tasks.
- Add shareable task link generation.
- Add Elm task list and task detail views.
- Add unit, integration, HTTP E2E, and Playwright coverage for task visibility and task creation.

Acceptance checks:

- Individual, team, org, and org-team owners work.
- Organization tasks default to org-scoped visibility.
- Public publishing requires permission.
- Capability tokens do not encode task IDs.
- Revocation is lifecycle/event based, not nullable timestamp based.

### PR 8: Submissions And Sensitive Data

Goal:

- Implement authenticated submissions with schema validation, receipt tokens where needed, and sensitive-field handling.

Tasks:

- Add submissions migration.
- Add submission receipt token migration.
- Add sensitive-field index migration.
- Add submitter tagged unions.
- Add submission lifecycle states.
- Add authenticated submission endpoint.
- Add receipt status endpoint.
- Add schema validation integration.
- Add sensitive-field extraction and storage.
- Add redacted response output based on viewer permission.
- Add submission review UI.
- Add unit, integration, HTTP E2E, and Playwright coverage for submission and receipt status.

Acceptance checks:

- Public tasks can receive authenticated submissions.
- Receipt token can check status without exposing requester-private data where receipt flows are enabled.
- Schema-invalid submissions are recorded with typed validation errors.
- Sensitive fields are redacted from unauthorized responses.

### PR 9: Credits, Ledger, Escrow, And First Accepted Submission

Goal:

- Implement Sharecrop credits, signup grants, task funding, escrow, acceptance, payout, and refunds.

Tasks:

- Add credit account migration.
- Add ledger migration.
- Add signup grant insertion.
- Add credit amount domain type.
- Add ledger entry kinds.
- Add task escrow flow.
- Add task refund flow.
- Add accept submission flow.
- Add transactional first-accepted-submission enforcement.
- Add idempotency keys for fund/accept/refund.
- Add ledger UI.
- Add task funding UI.
- Add tests for ledger invariants and race-prone acceptance cases.
- Add HTTP E2E tests for funding, acceptance, payout, and refund.

Acceptance checks:

- New registered user receives 100 credits.
- Reward-bearing tasks require escrow before opening.
- No-reward tasks can open without escrow.
- Only one submission can be accepted.
- Accepted submission receives payout ledger entries.
- Cancelled escrowed task refunds correctly.

### PR 10: MCP And Agent Setup

Goal:

- Expose task discovery and submission to local agents.

Tasks:

- Add agent credential migration.
- Add agent credential domain types and scopes.
- Add local MCP protocol handling without a Go MCP library.
- Use official MCP specification behavior as the reference.
- Add tools: list tasks, get task, get schema, submit response, get submission status.
- Add requester tools where authorized.
- Add agent setup UI.
- Add MCP config generation.
- Add curl examples in task pages.
- Add tests for scoped agent access.
- Add HTTP E2E tests for equivalent REST/curl flows.

Acceptance checks:

- Local agent can list permitted tasks.
- Local agent can fetch schema and payload.
- Local agent can submit a response.
- Agent credentials are scoped.
- REST/curl remains canonical and documented.

### PR 11: UI Polish And shadcn-Inspired Elm/Tailwind Components

Goal:

- Make the product usable and visually coherent.

Tasks:

- Add Elm/Tailwind component modules for buttons, fields, badges, tabs, dialogs, sheets, tables, alerts, cards, and nav.
- Use shadcn visual conventions without importing React components.
- Add responsive layout for pending work, task detail, org pages, submissions, and ledger.
- Add empty/loading/error states.
- Add keyboard-friendly forms.
- Add icon strategy using Elm-rendered SVG functions.
- Add basic light theme tokens and leave room for dark mode later.
- Add Playwright coverage for primary UI workflows and responsive smoke checks.

Acceptance checks:

- Core workflows are usable without visual placeholders.
- UI components are Elm-owned.
- Tailwind build is deterministic.
- No React runtime is introduced.

### PR 12: Asset Economy Foundation

Goal:

- Add organization credit accounts and platform collectibles, and keep user/org tokens deferred.

Tasks:

- Add `internal/assets`.
- Add transfer policy variants.
- Add collectible kind and collectible state types.
- Add asset ownership model.
- Add generated contracts for asset read models.
- Add browser collectible minting and awarding flows.
- Add policy tests.
- Add unit tests for token and collectible policy variants.

Acceptance checks:

- Collectible model compiles and is type-safe.
- Platform collectible reward execution is enabled.
- User/org token reward execution remains deferred.
- Asset policies use variants, not booleans.
- The model can later plug into escrow.

## Current Roadmap

The numbered pull-request roadmap in this plan is fully implemented and merged into `main`, including every section below (reservation foundations, requester ergonomics, review outcomes, reward bundles, and MCP Streamable HTTP). The sections are kept as a record of the agreed scope and defaults. Later work is tracked in [DO_NEXT.md](./DO_NEXT.md), [STATUS.md](./STATUS.md), and [WHAT_WE_DID.md](./WHAT_WE_DID.md), not by extending this roadmap. The runtime and deployment model is documented in [docs/deployment.md](./docs/deployment.md).

### Reservation, Approval, And Discovery Availability Foundations

Goal:

- Add the domain and API foundation for exclusive task assignment by user or team.

Defaults:

- Reservation expiry defaults to 48 hours.
- Expired reservations automatically release the task.
- A task can have at most one active assignee.
- Reserved tasks are hidden from default discovery and shown only when the client explicitly includes reserved tasks.
- Requesting changes keeps the reservation exclusive for the same assignee.
- Task-local bans block the same user or team from the same task only.

Tasks:

- Add participation policy domain types: open submissions, reservation required, requester approval required.
- Add assignee scope domain types: one user, one public Sharecrop team, one organization team in the same organization.
- Add reservation lifecycle types and migrations.
- Add task-local implementor ban types and migrations.
- Add reservation expiry release logic.
- Add task availability and viewer-action read models.
- Add HTTP APIs for reserve, request approval, approve, decline, cancel, and list reservations.
- Update task list/detail responses and discovery filters for availability and include-reserved behavior.
- Add unit and HTTP end-to-end coverage for reservation and approval flows. Browser coverage follows when the browser workflow is added.

Acceptance checks:

- Open tasks still allow direct submission.
- Reservation-required tasks only allow the active assignee to submit.
- Approval-required tasks only allow the approved assignee to submit.
- Reserved tasks are hidden from default discovery for other workers.
- Reserved tasks appear with `include_reserved`.
- Reservation expiry releases the task automatically.

### Requester Ergonomics And Task Page Instructions

Goal:

- Improve browser workflows for requesters and workers.

Tasks:

- Replace raw task-ID entry in funding and collectible-award forms with task selection from the user's task list.
- Add participation-policy, assignee-scope, and reservation-expiry controls to task creation.
- Add reservation and approval panels to requester task detail.
- Add worker reserve/apply/submit actions to task detail.
- Add include-reserved discovery control.
- Show REST and MCP instructions on task pages for reserving/applying/submitting results.
- Show schema-derived example response JSON where practical.
- Add Playwright coverage and manual screenshot review for requester and worker task-detail flows.

Acceptance checks:

- Requesters can work from task selectors instead of copying task IDs.
- Workers can see exactly how to submit by API or MCP from the task page.
- Reserved tasks appear only when requested or when the current viewer is the active assignee/requester.

### PR 16: Review Outcomes, Partial Rewards, Tips, And Bans

Goal:

- Add requester review outcomes beyond accept/reject.

Defaults:

- Request changes requires notes.
- Rejected-work fair reward is allowed only after a submitted response exists.
- Tips come from requester balance and inventory at review time, not prefunded escrow.
- Collectible payout can include any subset of escrowed collectibles plus optional extra collectibles from requester inventory.

Tasks:

- Add review outcome domain types.
- Add request-changes state transition with required notes.
- Add reject-with-partial-reward and reject-without-reward commands.
- Add accepted payout command support for partial/full declared reward and optional tips.
- Add credit balance checks for extra credit tips.
- Add collectible inventory checks for extra collectible tips.
- Add task-local implementor ban controls.
- Update ledger and asset transfer records for partial rewards and tips.
- Add HTTP, integration, and Playwright coverage.

Acceptance checks:

- Request changes keeps the assignee exclusive.
- Rejecting can release the task and optionally ban the assignee.
- Partial credit and collectible rewards are paid only when requester balance/inventory permits.
- Tips do not require prefunding.

### PR 17: Reward Bundles

Goal:

- Replace single-kind reward handling with reward bundles.

Tasks:

- Model reward bundles containing credits, collectibles, both, or neither.
- Update task creation, funding, opening, acceptance, refund, and cancellation around bundled rewards.
- Support multiple collectible reward items.
- Update generated contracts, HTTP responses, MCP output, and browser reward displays.
- Add migration compatibility for existing credit and collectible reward data.

Acceptance checks:

- Credit-only, collectible-only, combined, and no-reward tasks work.
- Tasks with declared rewards require matching escrow before opening.
- Refunds return all remaining escrowed reward items.

### PR 18: MCP Workflow Tools And Full Streamable HTTP SSE

Goal:

- Finish MCP workflow coverage and Streamable HTTP behavior.

Tasks:

- Add MCP tools for reservation, approval, cancellation, request changes, reject, partial payout, and task-local bans.
- Update MCP task tools to include reward bundle, participation policy, availability, viewer action, and API instructions.
- Enforce Streamable HTTP session IDs after initialize.
- Add `GET /mcp` server-sent events streams.
- Add `DELETE /mcp` session termination.
- Add SSE event IDs and retained-event replay using `Last-Event-ID` where practical.
- Emit task, reservation, and submission notifications over SSE.
- Add protocol tests from the official MCP Streamable HTTP requirements.

Acceptance checks:

- JSON-only MCP clients still work over POST.
- Streamable HTTP clients can use GET SSE streams.
- Session lifecycle and protocol-version behavior match the official MCP specification.
- MCP workflow tools enforce the same permissions as HTTP APIs.
