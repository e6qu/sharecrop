# Status

## Implemented surface

- **Human-configured agent work budgets** (the delegation contract): every
  personal agent credential carries a work policy that starts
  `work_seeking_disabled` — a freshly minted credential cannot reserve or
  directly submit until its owner enables work-seeking with a required daily
  task budget. Optional allowances: concurrent-reservation cap, daily credit
  spend cap, permitted task types, minimum credit reward, and an advisory
  token budget the server stores but never enforces (the agent meters
  itself). Enforcement is in-transaction and fails closed: daily counters are
  consumed in the same transaction as the reservation or submission, and
  reservations are attributed to the credential that established them
  (`reserved_via_credential_id`), so concurrency and daily caps cannot
  overshoot. Refusals carry `budget_exceeded` (HTTP 429, distinct from
  `rate_limited`) with reset-at-UTC-midnight messages, or
  `permission_denied` plus operator guidance when work-seeking is off.
  Reservation-backed submissions are exempt (the engagement was budgeted at
  reserve time), and user sessions are never budget-limited. Humans
  configure and watch this at `PUT /api/agent-credentials/{id}/work-policy`
  and on the Agents page, which shows today's consumption against each
  allowance; agents read their own leash with MCP `sharecrop.get_my_budget`
  (derived remaining values and `resets_at`).
- **Signup grants are verification-gated**: registration creates an account
  with a zero balance; the 100-credit grant is written idempotently when the
  address is first verified, and organization grants require a verified
  creator. Peer transfers carry a 500-credit daily velocity ceiling (admin
  grants exempt).
- **Sixteen task types** covering the knowledge work agents actually do:
  general, code review, security review, product review, UI/UX review, QA
  testing, document review, documentation writing, diagram writing,
  planning, research, data extraction, troubleshooting, code analysis,
  architecture review, and threat analysis — with create-form templates and
  grouped selectors across creation, discovery, budgets, and webhook filters.
- **Operator counters**: `GET /api/admin/operations/counters` and an Admin
  page section report outbox backlog and `dispatch_failed`, webhook pending
  and dead counts with oldest-pending age, and today's grants, peer
  transfers, and budget refusals. The endpoint is served host-side and
  reports honestly when unavailable rather than fabricating zeros.

- **Commit-ordered, loss-free event pipeline**: event seq allocation is
  serialized through a single-row fence lock taken as the final lock of every
  mutation transaction, so cursor feeds (poll, long-poll, SSE resume, MCP
  `list_events`) can never skip a late-committing event. Reservation expiry
  records its events in whichever transaction performs the release (request
  path or sweep); task/reservation expiry sweeps record drafts in the expiry
  transaction (the `Recorder.Emit` post-commit path is gone). Dispatch flips
  an event to `dispatched` only after every fan-out leg succeeds; malformed
  rows are skipped and logged; after 20 attempts a row retires to terminal
  `dispatch_failed` (operator-readable). Delivery claim holds cover
  worst-case batch time (at-least-once documented). Cancel events resolve
  released holders inside the cancel transaction.
- **Direct reservations**: the approval gate is removed. Participation is
  `open` or `reservation_required`; reserving always yields an active,
  exclusive, TTL-bound reservation and the holder submits directly.
  `requested`/`declined` reservation states and the approval endpoints/tools
  are gone (historical rows stay parseable; the migration promoted the
  oldest pending request per task).
- **Peer-to-peer economy**: any user sends credits to any user or
  organization (`POST /api/credits/transfers`, MCP `sharecrop.send_credits`
  behind the new mintable `ledger_write` scope, Overview send panel);
  organizations send with billing permission. Collectibles transfer
  user↔user and user↔org (org side gated by the new `manage_collectibles`
  org permission). `peer_transfer` ledger entries are double-entry, atomic,
  idempotent per account, and notify receivers (`credits_received`).
- **Admin-controlled collectible catalog**: the default catalog lives in the
  database (seeded with the original 25). Platform admins add entries (art
  from the fixed sprite registry, kind-coherent edition caps), withdraw
  entries (no longer awardable) and release them back to circulation,
  delete withdrawn entries only when no instances reference them (live or
  withdrawn), withdraw individual catalog-minted collectibles from holders
  and release them back (releasing a unique re-checks its slot and
  conflicts if re-minted; holders are notified of both directions), and
  delete withdrawn instances — REST + MCP + the Collectibles page, all
  audited. Ownership is visible: collectible surfaces show the owner's
  display label, unique catalog entries show who owns the single instance,
  editions show live-owner counts. Uniqueness is engine-enforced: `unique` entries have at
  most one live instance, editions are numbered against a per-entry cap,
  custom mints are unique per issuer+name. New mints default to
  `transferable_between_users`; collectible provenance (catalog slug,
  edition number, issuer name) is serialized everywhere.
- **Org-credential reviewing**: org credentials with submission scopes list
  submissions and review (accept/request-changes/reject) on their own
  organization's tasks over REST and MCP, via a typed reviewer union;
  org reviewers cannot tip or ban.
- **Honest contracts and errors**: openapi request-body `required` lists are
  true (omitempty audit), 20 more operations declare their query
  parameters, MCP malformed ids return uniform domain-shaped messages, and
  the UI renders API failures as visible load-error states instead of
  empty lists. The golden-coins sprite marks credit amounts; one gnome
  brand mark is used across app, docs, and landing; demo and app bundles
  build with `--optimize`.

- **Durable event pipeline**: domain events and their recipients are written
  in the same transaction as the mutation they describe (`domain_events.
  dispatch_state` recorded→dispatched); notification fan-out and webhook
  expansion run as an idempotent post-commit dispatch, with a lifecycle-runner
  sweep re-dispatching stale recorded rows after a crash. Idempotent replays
  (fund, accept, request-changes, reject, grants, gifts) emit no duplicate
  events or deliveries. The webhook pump drains its whole backlog each cycle
  (claim batches of 10, up to 50 per cycle). Host-side expiry sweeps still
  emit post-commit (recorded exposure).
- **Agent event loop without webhooks**: `GET /api/events` admits personal
  agent and org credentials holding `notifications_read` and supports
  `wait` long-polling (≤25 s, guest-degradable); MCP `sharecrop.list_events`
  is the request/response mirror. Task discovery filters by funded state
  (`funded=reward_funded|reward_unfunded|no_credit_reward`), and task rows
  carry funded state, creator/holder display names, and pending-review
  counts on both REST and MCP. Pager-backed lists report `total`.
- **Work outcomes are complete**: accepting a submission supersedes every
  competing `submitted` submission in the same transaction (state
  `superseded`, event + notification to each loser); rejected workers can
  file a structured dispute (moderation reason `dispute`, submission
  subject) from REST, MCP, and the browser; org funding carries the acting
  user; refunds notify the released reservation holder; worker-cancelled
  reservations record `cancelled_by_worker`.
- **Contract honesty**: openapi.json declares per-operation success status
  codes and error statuses derived from handler source, plus a shared
  `ErrorResponse` schema carrying the 10-code enum. The browser forces
  sign-out on an `unauthenticated` error code. The guest-side rate-limit
  bridge fails closed.
- **Garden-gnome identity**: the pixel brand is a garden gnome (nav mark,
  auth hero with mushroom, favicons across app/site/demo/docs, landing
  hero); empty states are gnome scenes (watering, dozing, signpost); the
  scarecrow survives only as a collectible sprite. Gnome accents
  (hat crimson distinct from alert red, tunic blue) join the farm palette;
  chrome accents shifted from brown to sage with measured ≥4.5:1 contrast.
- **Operational tooling**: `tools/webhook_receiver_sample.ts` (signature
  verification + dedupe reference receiver) and
  `tools/rehearse_migrations.sh` (times new migrations against a populated
  scratch database).

- **Agent work loop, both directions**: agents discover work by polling
  (`GET /api/tasks` — `scope` optional defaulting to `public`, `created_after`
  incremental filter, personal agent credentials admitted for public listings)
  or by push (webhook subscriptions with a `marketplace` audience deliver every
  newly opened public `task_opened`, with optional task-type and
  minimum-credit-reward filters; `recipient` audience keeps the original
  own-work semantics). Webhook scopes are mintable (the credential scope CHECK
  constraints previously omitted them). Reviewer agents read submission
  content over MCP (`sharecrop.get_submission`), and `submit_response`
  returns validation errors, keeps the reservation on invalid submissions,
  and accepts attachments. `create_task` requires an explicit
  `visibility_kind`. MCP `initialize` returns orientation instructions,
  `tools/list` is scope-filtered, every MCP list tool pages with
  `next_offset`, and `sharecrop mcp` (stdio) no longer requires HTTP config.
- **Operable economy**: platform admins grant credits to users or
  organizations (`POST /api/admin/credits/grants`, `sharecrop.grant_credits`,
  Admin-page form) through `manual_adjustment` ledger entries with a required
  note, per-account idempotency, a `credit_granted` event/notification, and
  audit records. Ledger reads are newest-first and expose the note.
  Registration is throttled per IP (capacity 5, ~12 min refill;
  `SHARECROP_REGISTRATION_RATE_CAPACITY` overrides for test harnesses).
- **Humans have names**: users carry a required `display_name` (derived from
  the email local part when not provided; editable at
  `PATCH /api/account/display-name`). Read models and DTOs expose
  creator/holder/submitter/author/actor names, task subject titles,
  and owned-task `pending_review_count`; the UI shows names (UUIDs demoted
  to tooltips), the signed-in identity in the header, a first-run explainer,
  a Needs-review card, sentence-style inbox/feed with relative timestamps,
  a native UTC datetime expiry input, credential copy buttons and scope
  presets, webhook audience choice with signature-verification instructions,
  and the admin grant form. OpenAPI declares path and query parameters
  (enums and defaults included) from contract-level declarations.

- **Domain event stream** (`internal/event`, `domain_events` + `domain_event_recipients`):
  every externally meaningful mutation emits a typed event from the service
  layer, so REST and MCP actions produce identical downstream effects.
  Notifications, webhook deliveries, and the browser live feed all derive from
  this one stream. Emission is post-commit best-effort (no outbox yet).
- **Notifications**: sealed 18-kind enum (submission, reservation, task,
  series-comment, payout/tip, collectible kinds), unread filter
  (`GET /api/notifications?state=unread`) and unread count
  (`GET /api/notifications/unread-count`), fan-out from the event recorder
  with self-notification suppression.
- **Live browser updates**: cursor-based per-user feed
  (`GET /api/events?after=`) plus an SSE variant (`GET /api/events/stream`,
  streaming when the transport has a Flusher, bounded replay under the WASI
  bridge). The Elm client polls every 15 seconds and on tab-visibility,
  shows an unread badge, and renders a Recent-activity card on Overview.
- **Outbound webhooks**: user- or organization-owned subscriptions over event
  kinds (`/api/webhook-subscriptions`, MCP tools, Elm management UI on the
  Agents page), HMAC-SHA256-signed deliveries
  (`Sharecrop-Webhook-Signature: v1=...`), bounded retry schedule
  (30s/5m/30m/2h/8h ±20% then dead), `FOR UPDATE SKIP LOCKED` claims, and a
  dial-time SSRF guard (https-only, no redirects, non-public addresses
  rejected at the resolved socket). Secrets are shown once at creation and
  stored as written (HMAC requires the plaintext).
- **Lifecycle runner** (`internal/runner`, host-only, both native and WASI
  modes): reservation expiry (1m), task expiration (1m), privacy retention as
  the system actor (1h), Postgres rate-limit bucket eviction (5m), MCP session
  sweep (10m), webhook pump (5s). The seeded system actor
  (`core.SystemUserID()`, `system@sharecrop.invalid`) can never authenticate
  and registration rejects its address.
- **Task expiration is real**: tasks accept an optional `expires_at`
  (REST/MCP), the sweep transitions open tasks past the instant to the
  `expired` state, refunds credit and collectible escrow idempotently
  (`expire:<task_id>` keys), releases reservations, and emits `task_expired`.
- **API quality**: error bodies carry a machine-readable `code` (10-code
  enum); MCP tool failures return structured `{"code","message"}` JSON;
  org credentials are scope-checked on REST exactly as on MCP; ledger
  idempotency keys are unique per account (not globally); task creation
  escrows reward collectibles transactionally; list responses carry
  `next_offset`; previously unbounded lists are paginated; the moderation
  list filters in SQL before paginating; `ban_implementor` became the
  `ban_selection` enum; request-changes requires an idempotency key and
  replays; audit after a committed mutation is best-effort.
- **MCP parity**: `create_task` supports owner/visibility/assignee-scope/
  reservation-expiry/series/payload/attachments/reward-collectible-ids/
  expires_at exactly like REST; `accept_submission` takes
  `tip_collectible_id`; `list_tasks` takes the full REST filter set; credits
  tools (`get_credit_balance`, `list_ledger`) run under the now-live
  `ledger_read` scope; list tools take limit/offset.
- **HTTP/2 groundwork**: `SHARECROP_HTTP_PROTOCOL=h2c` opts the listener into
  cleartext HTTP/2 (`golang.org/x/net/http2/h2c`); default stays HTTP/1.1
  because the API Gateway edge speaks HTTP/1.1 with a 30-second cap.
- **Farm/pixel UI**: the shipped Elm app uses the farm design system end to
  end — `--color-farm-*` Tailwind tokens, self-hosted Press Start 2P/VT323,
  parchment cards with hard ink borders and offset shadows on the field
  green, pixel display headings, sprite-backed empty states and brand row,
  measured ≥4.5:1 contrast per tone, global `:focus-visible` outline. The
  demo's `arcade.css` is a stub; app and demo share one theme.
- **Service graph**: `internal/appgraph.Build` assembles the domain services
  for serve, mcp-stdio, the WASI guest (appmux), and the runner from one
  place.

- A Go HTTP API (`internal/http`) over domain services (`internal/task`,
  `internal/ledger`, `internal/assets`, `internal/submission`, `internal/org`,
  ...), an Elm browser client, an MCP interface at `/mcp` (Streamable HTTP with
  SSE), scoped agent and organization-wide credentials, and a generated OpenAPI
  document (`docs/openapi.json`).
- **One store implementation** (`internal/db`), engine-neutral behind a small
  `db.Querier`/`Beginner` handle abstraction, parameterised only by SQL engine:
  Postgres in production, SQLite (via ncruces) in the browser demo. There is no
  separate browser storage adapter — `internal/wasmdemo` is deleted.
- **One application, two runtimes from the same source:**
  - The **browser demo** (`cmd/sharecrop-wasm`, `js/wasm`, GitHub Pages) runs the
    real mux + domain services over in-browser SQLite.
  - The **backend** runs the same app server-side through the WASI guest pool.
    This is the production default: `cmd/sharecrop serve` embeds the `wasip1` app
    guest (`internal/wasiguest`, built by `make wasi-app-guest` as part of
    `make build`) and hosts it under a wazero runtime, dispatching its store
    calls to Postgres via `storehost`. `SHARECROP_WASI_MODE=native` runs the
    in-process mux instead; a binary built without the guest runs native.
- **Deployment:** a slim multi-architecture container (arm64 primary) on Amazon
  ECS Fargate in private subnets, reached by an Amazon API Gateway HTTP API
  through a VPC Link and AWS Cloud Map, with state in Sharecrop's distinct
  database inside the shared PostgreSQL service. No Application Load Balancer
  or Network Load Balancer is provisioned. The guest's machine code is baked
  into the image as a wazero AOT cache, so the server does no
  compile at startup. Every merge publishes an immutable 12-character commit-SHA
  manifest to the GitHub Container Registry, with direct arm64 and amd64 image
  tags and no mutable or semantic-version tags. The newest 20 complete releases
  are retained; untagged, incomplete, mixed-tag, and unrecognized versions are
  deleted. See
  [docs/deployment.md](./docs/deployment.md).
- **Shared environment deployment:** Terraform accepts an existing Amazon
  Elastic Container Service cluster ARN and an existing Amazon API Gateway VPC
  Link ID, so the service can run in the shared `dev` cluster and reuse its
  private network path. A plan-known ownership boolean selected dedicated or
  shared mode; shared mode required the paired link and security-group IDs,
  including when both came from unknown-until-apply wrapper resources. The
  standalone defaults still create both resources.
- **Ordered deployment:** an AWS Step Functions workflow runs the standalone
  migration task synchronously and updates the Amazon ECS service only after
  migration success. A one-time Amazon EventBridge Scheduler schedule starts
  each changed workflow. PostgreSQL advisory transaction locking and the
  migration ledger make duplicate cloud delivery safe and apply each SQL file
  once.
- **DNS integration:** Terraform configures the regional Amazon API Gateway
  custom domain and exposes its target domain and hosted-zone ID so an
  environment can create the exact Route 53 alias.
- **Monitoring integration:** Terraform exposes the actual CloudWatch Logs group
  used by serve tasks and the Amazon API Gateway access-log group. Detailed
  route metrics and bounded burst/rate throttles are enabled.
- **Provider compatibility:** The deployment module requires HashiCorp AWS
  provider 6.x, matching the shared `dev` environment and the other deployed
  service modules.
- **Health routing:** the distroless container's binary probes the real
  `/healthz` endpoint. Amazon ECS publishes task and container health to AWS
  Cloud Map, and Amazon API Gateway routes only to healthy discovered tasks.
  Terraform waits for steady state and unhealthy deployments roll back.

## State

The deployment used private Amazon ECS Fargate tasks without public IP
addresses. Amazon API Gateway reached them through a VPC Link and discovered
their address and port from AWS Cloud Map SRV registrations. The public
execute-api endpoint was disabled, the custom domain was TLS-only, and the
default route applied explicit throttles, access logs, and detailed metrics.
The `$default` route forwarded the unchanged request path, and its auto-deploy
stage depended on that route so a partial apply could not publish a route-less
custom domain. Security-group rules admitted the HTTP port only from the
selected VPC Link. A policy gate rejected any Application Load Balancer,
Network Load Balancer, public task IP, or incomplete private-ingress resource
from the Terraform module.

The latest work bound Sharecrop to Shauth's application-owned logout-completion
bridge and release-validation contract. The OpenID Connect session persisted
the provider username, verified email, and role alongside the immutable
issuer/subject identity. `/auth/validation` exposed that identity and the exact
12-character release revision, while `/auth/shauth/logout/complete` accepted no
caller destination and returned only to Shauth's correlated one-time completion
endpoint. Direct entry and Apps-catalog launch, automatic SSO, application and
provider logout, Front-Channel and Back-Channel Logout, hostile bridge input,
retained-credential rejection, and app-local signed-out recovery passed against
real Shauth, Ory Hydra, PostgreSQL, and the production WASI binary.

Shauth is an additional browser identity provider. A verified OpenID Connect
issuer/subject pair is persisted independently from mutable profile claims and
receives the same rotating Sharecrop session as a local login. Local passwords
and first-party tokens remain available. Existing password accounts are never
linked to a new external identity merely because their email addresses match.
The callback uses PKCE, nonce/state validation, an authenticated short-lived
transaction cookie, and exact HTTPS issuer/public URL coordinates. It retains
the provider-signed ID token and optional `sid` in PostgreSQL for
RP-Initiated Logout without exposing them in the browser cookie. In the
production WASI deployment, Shauth authorization, callback, logout,
Back-Channel Logout, and signed-out routes run on the native host boundary
because OpenID Connect discovery and token exchange require outbound HTTPS and
logout state is shared by every replica; the rest of the application remains
hosted by the WASI guest pool. When Shauth is configured, application entry routes require the
Sharecrop refresh-session cookie and redirect a new visitor to Shauth, so an
Apps-catalog launch and a direct application URL start the same identity flow.

Shauth Back-Channel Logout validated the exact issuer, audience, signature,
expiry, standard logout event, prohibited `nonce`, freshness, and either `sid`
or `sub`. PostgreSQL atomically claimed each logout-token `jti` and revoked the
matching active refresh-token families, so replay protection survived process
and replica changes. Browser logout revoked the local refresh family before
returning the issuer-origin end-session URL with the provider-signed ID token
hint and exact `/auth/shauth/logout/complete` redirect. The application bridge
returned to Shauth's fixed `/oauth/logout/complete` endpoint, where Shauth's
host-only one-time correlation selected `/api/auth/signed-out`; request query
parameters never selected a destination. The signed-out landing revoked
any residual local refresh family and did not restart authentication. It
rendered a branded, accessible light/dark Sharecrop page whose explicit
same-origin `Sign in with Shauth` control was stable across reloads. The logout
verifier cached provider discovery and its remote key set while retaining
normal signing-key rotation behavior. Shauth Front-Channel Logout also revoked
the exact issuer/session-ID relationship and returned a non-cacheable,
frame-safe completion document.

When Shauth was configured, the browser hid local registration, password reset,
and token entry, while programmatic first-party credentials remained supported.
The application shell and protected browser API rejected revoked refresh-token
families even when a previously minted access token was still present. External
identity provisioning used the immutable issuer/subject pair; Shauth's optional
email-verification claim was not treated as mandatory or used to link an
existing account.

The migration command loaded only its database URL and migration directory, so
the standalone Amazon ECS migration task did not depend on HTTP or access-token
runtime configuration. AWS Step Functions waited for that task before rolling
the service and then waited for the target task definition to be the sole
completed deployment. The server and MCP transports verified that every
migration baked into the image had been applied before serving requests,
preventing a partially migrated database from presenting a healthy application
whose authentication callback failed later.

Both the single-store-implementation program and the WASI-production-hosting
program are complete. Recent work hardened the production-default WASI path:
real randomness and clock in the guest, per-client rate limiting and MCP origin
checks (the request bridge now carries RemoteAddr and Host), fixed an MCP SSE
pool-exhaustion denial of service, forwarded request-shaping env into the guest,
and raised the bridge frame limit above the request-body limit while bounding the
host-side body read. The container/deploy work then baked the wazero AOT cache,
slimmed the image, and added the ghcr release workflow.

## Test status

The agent-loop-completion branch passed the full local battery: every static
gate, Go unit suites, Deno tests, PostgreSQL integration (marketplace webhook
expansion matrix, grants with idempotent replay, scope-mintability sweep,
invalid-submission-keeps-reservation, created_after), HTTP end-to-end
(including agent-credential public listing, admin grants, marketplace
subscription validation, display-name lifecycle, and the MCP walkthrough
scenarios: reviewer submission read, validation-error return with immediate
resubmit, scope-filtered tools/list), both Playwright suites (71 DB-backed +
16 demo/mobile), and WASM scenario parity. Screenshots of every changed
screen were reviewed at 1280px and 375px.

The prior review-upgrade branch passed the full local battery: every static gate
(format, contracts, openapi, policy, release contract, TypeScript, copy-paste,
dead code, WASI bridge, workflow timeouts, lint, vet), Go unit suites, Deno
tests, PostgreSQL integration (including lifecycle-sweep, webhook-pump
concurrency, and idempotency-scope tests), HTTP end-to-end (including the
MCP-mutation-produces-notification proof, the REST scope-gate proof, error-code
and next_offset contracts, and MCP/REST create-task parity), DB-backed
Playwright (52 specs including unread badge, activity feed, webhook management,
request-changes, expiration), the demo/mobile Playwright suites against the
rebuilt WASM demo, and WASM scenario parity. Screenshots of the rethemed
screens were reviewed manually.

PR CI runs format/contract/policy/type checks, Go unit and integration tests,
HTTP end-to-end tests, shared scenario parity against both SQL engines, and
Playwright browser tests. The Release workflow builds and publishes the image on
merge. The Shauth integration passed the frontend build, full Go suite,
WASI bridge generation checks, PostgreSQL integration and HTTP suites, and
native/WASI scenario parity. A real browser suite against Shauth commit
`74735a1710fa69d472e7eb27ae95ce317c7c1a3d`, Ory Hydra v26.2.0, PostgreSQL
17.5, and the production WASI binary passed direct entry, Apps-catalog entry,
automatic SSO, identity provisioning, account display, app-local logout and
reload, explicit local recovery, provider-initiated logout, rejection of
retained access and refresh credentials, and direct-entry fail-closed behavior.
It also checked the exact username, email, role, and release revision and
rendered distinct light and dark signed-out themes. All 62
general browser cases passed with retries disabled; the three previously
timing-sensitive paths also passed ten focused stress iterations without
retries. Authentication-operation rate limits were isolated per path and
client IP so registration or recovery traffic could not starve login traffic
for users behind the same NAT.
The release publisher verified that each architecture tag was a direct OCI
image manifest and that the generic tag contained exactly Linux amd64 and Linux
arm64 before retaining the newest 20 complete commit-SHA releases. The
retention gate deleted untagged, incomplete, mixed-tag, unrecognized, and old
versions and verified the package postcondition. The historical Sharecrop
package was cleaned to the current complete three-version release.
The Sharecrop command suite, generation checks, policy checks, release contract,
TypeScript checks, WASI bridge checks, lint, vet, Go/Deno tests, Terraform
formatting, and provider-backed Terraform validation passed after the private
ingress replacement.
The ordered deployment contract passed no-mock Deno checks, concurrent migration
execution passed against real PostgreSQL, and provider-backed plans against the
real development VPC covered the dedicated path, the existing-link path, and an
environment wrapper whose resource-derived link coordinates were unknown until
apply. Terraform working directories were repository-ignored and excluded from
the Deno formatter, so initializing the provider-backed wrapper could not stage
provider binaries or make the source-format gate inspect generated metadata.

## Blocking issues

None.
