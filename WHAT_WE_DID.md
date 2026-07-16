# What We Did

Terraform exposed the Application Load Balancer canonical hosted-zone ID beside
its DNS name. Environment Terraform can now create a Route 53 alias record for
Sharecrop without depending on an implicit or reconstructed zone ID.

Terraform deployment accepted an existing Amazon Elastic Container Service
cluster ARN. The Sharecrop service and migration command used that cluster when
configured, while the dedicated-cluster default remained available. This allowed
the `dev` deployment to share its one ECS cluster without creating another
cluster or network path.

WASI hosting became the **production default** and the backend was containerized
for ECS Fargate. The app guest is now embedded in the `sharecrop` binary
(`internal/wasiguest`, built by `make wasi-app-guest` as part of `make build`), so
`serve` hosts through the WASI guest pool by default; `SHARECROP_WASI_MODE=native`
opts out. (This supersedes the "opt-in via `SHARECROP_WASI_GUEST`" framing in the
cutover entry below — that variable still overrides the embedded guest, but the
default flipped.) The production-default WASI path was then hardened: real
`crypto/rand` and wall clock in the guest, per-client IP rate limiting and the MCP
origin check restored by carrying `RemoteAddr`/`Host` across the request bridge, a
fixed MCP SSE pool-exhaustion denial of service, forwarded request-shaping env,
and a bridge frame limit raised above the request-body limit with the host body
read bounded. The backend was then packaged as a slim multi-arch (arm64) container
on distroless with a baked wazero AOT cache so it does no compile at startup, a
`ghcr` release workflow (conventional-commit versions, no `:latest`), and ECS
Fargate task definitions. Running production on the same wasm app as the demo is
therefore done, not a "tracked goal." See
[docs/deployment.md](./docs/deployment.md).

The `task/wasi-cutover` branch is the **production cutover - the end of the WASI
hosting effort**. `cmd/sharecrop serve` can now serve production by running the
compiled app guest, so production and the browser demo run the same WASM
artifact. It is opt-in and reversible: when `SHARECROP_WASI_GUEST` points at a
wasip1 app-guest module (`make wasi-app-guest` builds it), `serve` builds an
`rpc.Pool` over it (sized by `SHARECROP_WASI_POOL_SIZE`, default GOMAXPROCS),
dispatches every store call the guest makes to Postgres via
`storehost.Dispatcher`, and routes the dynamic paths (`/api/`, `/mcp`,
`/healthz`) to the guest while serving static assets and the SPA shell host-side
(the guest carries no static files). With the variable unset, `serve` keeps the
native in-process mux, so nothing changes for existing deployments until an
operator opts in. Verified two ways: an integration test that assembles the same
handler and checks both halves (host serves the SPA shell and `/static/` assets;
the guest handles `GET /api/notifications` and returns real Postgres data), and a
live smoke test of the real `serve` binary in WASI mode - it logged "serving
dynamic routes through the WASI guest pool" (pool_size 12) and answered `/healthz`
200 (guest → Postgres) and `/` 200 (host static). A `make wasi-app-guest` target
builds the artifact, `.gitignore` covers it.

This completes the effort tracked since the store-bridging phase began: ten
domain stores and six RuntimeState infra services are bridged, the full mux runs
in a pool of reused guest instances, and `cmd/sharecrop serve` can now run
production on the same compiled WASM artifact as the demo - closing the
native-vs-WASM deviation the "one backend" section of STATUS named. One backend,
one artifact, two storage adapters. All gates green. Nothing about the native
server or browser demo changes for anyone who doesn't opt in.

---

The `task/wasi-bridge-mcpsession` branch bridges **MCP session persistence** -
the sixth and **last** RuntimeState infra service. Like the rate limiter it is
hand-written, because its methods return multi-value tuples (`(bool, error)`,
`(string, []byte, error)`, `([]string, [][]byte, error)`) that the code generator
(built for a single union result) doesn't model. `internal/wasibridge/mcpsessionbridge`
has a `GuestMCPSessionPersistence` that implements `httpserver.MCPSessionPersistence`
by RPCing each of the seven methods, and a `Dispatch` that routes back to the real
store; each method has a small args struct and a result struct carrying the return
values plus an error string (payloads cross as base64 via JSON's `[]byte` support).
MCP Streamable HTTP sessions and their replay events must be shared across every
request - a per-instance in-memory copy would strand a session on whichever pooled
instance created it - so this is exactly the kind of state that has to be bridged.
`appmux.Stores` gained an `MCPSessions` field, and `appmux.New` wraps it with
`httpserver.NewPersistedMCPHTTPSessionStore` (the same wrapper the native server
uses). Dual-run-verified across the whole lifecycle: create, count-for-subject,
append-event, list-events (with the replayed payload), touch, close, and
count-after-close - all against a unique session id and subject so nothing
contaminates the shared db-checks database. **This completes the RuntimeState
bridging: appmux now overrides every in-memory default, so a pooled guest shares
all state through Postgres, and the production cutover is unblocked.** All gates
green. Nothing about the native server or browser demo changed.

---

The `task/wasi-bridge-ratelimit` branch bridges the **rate limiter** - the fifth
of the six infra services, and the first that is **hand-written** rather than
generated. `RateLimiter.Allow(key) bool` takes no context and returns a bare
bool, which the code generator (built for ctx + single-union-result methods)
doesn't model, so `internal/wasibridge/ratelimitbridge` is written by hand: a
`GuestRateLimiter` that implements `httpserver.RateLimiter` by RPCing each call
(`Allow`/`ActiveBuckets`/`StorageKind`) to the host, and a `Dispatch` that routes
back to the real limiter. Because there are two limiters - one keyed by client IP,
one by MCP agent subject - the wire method carries a prefix (`ratelimit.ip.Allow`
vs `ratelimit.subject.Allow`) that selects which, and the guest holds one
`GuestRateLimiter` per prefix. `Allow` fails open on a transport error: a broken
bridge must never lock every client out. The db rate limiter genuinely queries
Postgres (a token-bucket row per key), so bridging keeps the buckets in one
shared store instead of a per-instance in-memory copy - a pooled guest now rate-
limits consistently across instances. `appmux.Stores` gained `IPRateLimiter` and
`SubjectRateLimiter` fields overriding the in-memory defaults. Dual-run-verified:
draining a unique key's 20-token bucket through the bridge enforces the shared
budget (20 allowed, then denied), and StorageKind/ActiveBuckets match a direct
call. The unique key keeps the bucket private so nothing contaminates the shared
db-checks database, and the full integration suite was run locally. Five of six
infra services bridged; only MCP session persistence remains (also hand-written -
multi-return tuples), then the cutover. All gates green. Nothing about the native
server or browser demo changed.

---

The `task/wasi-bridge-privacy` branch bridges the **privacy** RuntimeState
service - the fourth and last of the codegen-friendly infra services, and the
largest (6 methods, 3 result unions). Like moderation-triage it takes a
cross-package type - `RecordSensitiveFieldAccess(ctx, UserID, submission.Submission)`
- so the spec sets `extraImports` for internal/submission, and `Resolve(ctx,
string, string)` gets the two-string disambiguation. And like moderation it uses
a minimal codec for the heavy cross-package type: the store reads only the
submission's ID and each sensitive field's Path (it records one access event per
field), so the wire carries `{id, sensitive_field_paths}` and rebuilds a minimal
submission rather than duplicating submissionbridge's full codec - documented,
and the dual-run test would catch it if the store read more. `appmux.Stores`
gained a `Privacy` field overriding the in-memory default. Dual-run-verified
across all six methods (create / list-for-requester / resolve / record-access /
run-retention), with scoped-not-global list assertions so nothing contaminates
the shared db-checks database, and the full integration suite was run locally to
confirm. Four of six infra services are now bridged; the last two - the rate
limiter (`Allow(key) bool`, no ctx) and MCP session persistence (multi-return
tuples) - don't fit the codegen and are next. All gates green. Nothing about the
native server or browser demo changed.

---

The `task/wasi-bridge-moderationtriage` branch bridges the **moderation-triage**
RuntimeState service - the third of the six the cutover needs. It exercises two
generator features at once: `RecordOpen(ctx, audit.Event)` takes a type from a
third package (so the spec sets `extraImports` for internal/audit, as org did for
auth), and `Update(ctx, UserID, AuditEventID, string, string)` takes two strings
that the generator disambiguates (`State`/`State2`). One deliberate simplification:
both the in-memory and db moderation stores read only `event.ID` and
`event.CreatedAt` from the audit.Event (a moderation report is keyed by the audit
event that opened it), so the wire carries just those two fields and rebuilds a
minimal event rather than the whole thing - documented in the codec, and the
dual-run test would fail if `RecordOpen` ever read another field. Unlike
platform-admin, moderation `List` is by explicit audit-event-id (not a global
list), so it can't contaminate other tests in the shared db-checks database.
`appmux.Stores` gained a `ModerationTriage` field overriding the in-memory
default; dual-run-verified (record-open / list / update) with a test-only audit
action so the seed event never collides with scenario-parity. All gates green,
and the full integration suite passes (checked locally for shared-db
contamination). Three of six infra services bridged. Nothing about the native
server or browser demo changed.

---

The `task/wasi-bridge-platformadmin` branch bridges the **platform-admin**
RuntimeState service - the second of the six infra services the cutover needs,
and the second-simplest. Its `Grant(ctx, userID, actor)` takes two `core.UserID`
arguments, so it exercises the generator's repeated-arg disambiguation
(`UserID`/`UserID2`) - the same enhancement the assets bridge first needed. The
one wrinkle beyond the saved-queue-view pattern: the db platform-admin store is
constructed with a bootstrap-admins set (`db.NewPlatformAdminStore(pool,
bootstrap)`), so `storehost` now parses `SHARECROP_ADMIN_USER_IDS` (the same
host-side config the native server uses) to seed it; tests run with an empty set.
`appmux.Stores` gained a `PlatformAdmins` field, so `appmux.New` overrides the
in-memory default. Dual-run-verified (grant makes a user an admin; is-admin and
list match a direct db call) and the full-graph route tests pass with it wired
in. Four of the six infra services are now bridged or trivially so - saved-queue-
view and platform-admin are done; privacy and moderation-triage remain but each
references a cross-package value type (submission.Submission, audit.Event) that
will likely be extracted into a shared wire package (as attachmentwire was) to
avoid duplicating an existing bridge's codec. The last two (rate limiter, MCP
sessions) don't fit the codegen at all. All gates green. Nothing about the native
server or browser demo changed.

---

The `task/wasi-bridge-savedqueueview` branch begins bridging the **RuntimeState
infra services** so a faithful production cutover is possible. The prior work
bridged the ten domain stores and wired the full mux into a pooled guest, but the
mux also needs six RuntimeState services (rate limiters, MCP sessions, saved
queue views, privacy, platform admins, moderation triage) that `cmd/sharecrop
serve` backs with **db stores** while `appmux` used **in-memory** versions. Under
pooling that means per-instance state (inconsistent rate limits, unfindable MCP
sessions, etc.) - a regression. So these must be bridged too. This branch does
the first, `SavedQueueViewService`, and establishes the pattern: the bridge
codegen now targets an interface in `internal/http` (package `httpserver`) rather
than a domain package - `gen.Targets()` points at `internal/http`, the spec's
`interfaceName` is `SavedQueueViewService`, and the generated `GuestStore`
implements `httpserver.SavedQueueViewService` (which the db store already
implements, so the bridge just carries it across the boundary). `appmux.Stores`
gained a `SavedQueueViews` field, so `appmux.New` overrides the in-memory default
with the bridged store; the guest supplies the GuestStore, the tests supply
`db.NewSavedQueueViewStore`. It is dual-run-verified and route-verified
(`GET /api/saved-queue-views` byte-identical through the guest, reading a view
seeded directly in Postgres). The route-test setup that four tests were copying
was extracted into shared `serveRouteBothWays` + `assertBridgeMatchesNative`
helpers (jscpd would otherwise flag the duplication). Four of the six infra
services (saved-queue-view, privacy, platform-admin, moderation-triage) fit this
codegen recipe; the other two don't - the rate limiter (`Allow(key) bool`, no ctx)
and MCP session persistence (multi-return tuples like `(bool, error)`) - and will
need hand-written bridges or host-side placement. All gates green. Nothing about
the native server or browser demo changed.

---

The `task/wasi-instance-pool` branch adds **instance pooling** to the WASI app
host, resolving the open perf question from the spike (finding #8: is one fresh
instance per HTTP request viable, or does it need pooling?). The guest was a
wasip1 *command* - it ran `main()` once per unit of work and exited - so every
HTTP request paid the ~2-3ms guest-startup floor (Go runtime init inside wasm),
and command instances can't be reused. The Phase-1 findings had also proved that
a *shared* reactor instance (two goroutines touching one instance) corrupts
state. The design that threads that needle, now implemented: the guest's
`main()` loops over units of work read from stdin as "work" frames (via a new
`rpc.Serve` helper), staying alive between them so one instance serves many
units; the host keeps a pool of such instances (`rpc.Pool`) and checks one out
per unit of work. Safety is preserved because each instance is still driven by
exactly one goroutine and touched by no other: a per-instance `session` runs a
runner goroutine (the only one that touches the wazero instance) and a driver
goroutine (the only one that reads the guest's stdout and writes its stdin, so
the two-write frame encoding can never interleave); request goroutines reach the
driver only over Go channels. Pooling therefore adds zero shared-instance
concurrency - concurrency comes from having several independent sessions, one
checked out per in-flight unit. The unit-of-work now arrives over stdin ("work"
frames) instead of argv, so both `Host.Call` (fresh instance per call, still used
by every store dual-run and route test) and the new `Pool.Call` sit on the same
`session`; `httpbridge.Handler` takes an `rpc.Caller` so it drives either. The
production `cmd/sharecrop-wasi-app-host` now uses a pool sized by
`SHARECROP_WASI_POOL_SIZE` (default GOMAXPROCS) and builds each guest's mux +
service graph once per instance rather than per request. Two new integration
tests are the safety proof: 24 recipients x 6 rounds = 144 concurrent store
units through a pool of 4 reused instances, and 16 concurrent authenticated HTTP
requests through the pooled app host - every unit sees exactly its own data, no
cross-talk, no crash. The full integration suite (all ten store dual-runs, the
route tests, both pool tests) passes on the refactored transport. All gates
green. Nothing about the native server or browser demo changed.

---

The `task/wasi-appmux-full-graph` branch is the first step of the host-wiring
phase that follows store-bridging: it wires the **full production mux** into the
WASI app guest. Until now `internal/wasibridge/appmux` only wired the auth +
notification services (the Phase-4 slice); every other domain service was passed
as nil to `httpserver.NewWithRuntimeState`. Now that all ten stores are bridged,
appmux builds the complete domain-service graph - auth, notification, org, task,
submission, ledger, agent, orgcred, assets, audit - in the exact dependency order
`cmd/sharecrop serve` uses: the org and agent services feed the task service, and
the shared task store plus the org service feed the submission service. There are
no adapter types - the domain services satisfy each other's cross-service
interfaces (`task.OrganizationPermissions`, `task.TaskCredentialIssuer`,
`submission.TaskFinder`, `submission.OrganizationPermissions`) directly, and the
one store shared by two services (the task store, used by both task and
submission) is passed to both. The RuntimeState services that have no dedicated
domain store (rate limiters, MCP sessions, saved queue views, privacy, platform
admins, moderation triage) keep the in-memory defaults from
`httpserver.DefaultRuntimeState`; only audit and notification are overridden to
run over the bridged stores. `appmux.New`'s signature changed from three
positional args to an `appmux.Stores` struct of ten store interfaces, so the
guest passes bridge GuestStores and the tests pass real `internal/db` stores -
the assembled mux is byte-for-byte identical either way. The app guest
(`cmd/sharecrop-wasi-app-guest`) now constructs all ten GuestStores, and the two
existing route tests plus a new one (`GET /api/credits/balance`, backed by the
ledger service, returning the real 100-credit signup grant) verify routes run
byte-identically to the native mux through the full-graph guest. All gates green.
Nothing about the native server or browser demo changed.

---

The `task/wasi-bridge-task` branch bridged the `task` store - the tenth and
**final** store, and the widest by far (21 methods). This completes the
store-bridging phase: every one of the ten domain stores now round-trips through
the WASI guest. The `task` store is the deepest in structure: the `Task` model
carries roughly ten nested unions (owner, reward spec, visibility, series
placement, data payload, plus the list-scope and four list-filter unions), and
the store also covers first-class series, reservations, and both the task and
series comment threads. `internal/wasibridge/taskbridge` splits its codecs across
four files (value types + unions, models, commands, results) plus the generated
`bridge_gen.go`. It needed **no generator change** - the third enhancement
(`extraImports`) already covered its only cross-package concern, and here the
one cross-package type (`CreateCommand.Actor auth.UserSubject`) is nested inside
a command, handled in codec.go, so the generated signatures never reference
`auth`. `corewire` gained `TaskSeriesID`/`TaskReservationID`/`SeriesCommentID`/
`TaskCommentID` codecs. No new reconstruction constructors were needed: as with
ledger and org, every task value type round-trips through its existing validating
constructor. The attachment codec - which the submission bridge had introduced -
was extracted into a shared `internal/wasibridge/attachmentwire` package and
adopted by both bridges (jscpd would otherwise flag the duplicate). Shared task
test builders and deep diff helpers live in `internal/task/tasktest`; the codec
test round-trips a fully-populated Task (every union arm), a Series, a
Reservation, a comment, and the list scope/filters; the dual-run integration
test (`tests/integration/taskbridge_store_test.go`) drives create / find /
change-state / list / reserve / comment / create-series / attach-task / series-
comment through the guest against real Postgres, checking every read against a
direct call. Ten stores bridged - the whole set. All gates green. Nothing about
the native server or browser demo changed.

---

The `task/wasi-bridge-org` branch bridged the `org` store (organizations,
members, and teams) - the ninth store - and drove a third generator enhancement.
`org.Store.ProvisionMember(..., auth.EmailAddress, ...)` and `AddTeamMemberByEmail`
are the first methods whose argument type lives in a *third* package (neither the
domain package nor core), so the generated `GuestStore` signatures reference
`auth.EmailAddress` and the generated file must import `auth`. The generator's
import block was hardcoded to core/corewire/domain, so `storeSpec` gained an
`extraImports []string` field; `emit` writes those into the import group
(`format.Source` sorts them). Backward-compatible: every other spec leaves
`extraImports` nil, so their generated files are byte-unchanged (verified by
`check-wasi-bridge`), and a new gen unit test covers it. `internal/wasibridge/orgbridge`
serializes `Organization`, `OrganizationMember` (with its `MembershipStatus` and
`[]Role`), and `Team` - including the `TeamOwner` tagged union
(organization-owned vs standalone user-owned) - plus twelve result unions (four
accept/reject pairs share `acceptedRejectedWire`; provision and update-roles
share `memberResultWire`). No new reconstruction constructors were needed: org
value types (names, roles, membership status, email) round-trip through their
existing validating constructors because the store only emits valid values.
`corewire` gained `TeamID` and `OrganizationMembershipID` codecs. Shared test
builders live in `internal/org/orgtest` (to keep jscpd at 0 across the codec and
integration tests). The dual-run integration test
(`tests/integration/orgbridge_store_test.go`) drives create-org / provision /
update-roles / deactivate / create-team / add-member through the guest against
real Postgres and checks every read path against a direct call. The generic
store guest and `storehost` route `org.*`. Nine stores bridged (audit,
notification, auth, agent, orgcred, assets, submission, ledger, org); only
`task` remains. All gates green. Nothing about the native server or browser demo
changed.

---

The `task/wasi-bridge-ledger` branch bridged the `ledger` store - the eighth and
most union-dense store - and, unlike the previous few, needed **no generator
change**: it fit the existing framework. The ledger is the credit system, so its
types nest deeply. Commands carry selection unions (`CreditReviewSelection`
full/partial/none, `TipSelection` none/credit, `CollectibleTipSelection`
none/selected, `BanSelection` none/ban), and the accept/reject results carry a
`PayoutOutcome` (none/credit/collectible/bundle) and a `TipOutcome`
(none/credit/collectible/bundle). `internal/wasibridge/ledgerbridge` serializes
all of them, plus `LedgerEntry` (with its `EntryKind`, `SignedAmount`, and
`TaskReference` union), `TaskFund`, and `Balance`. Amounts cross the wire as
int64 base units. No new reconstruction constructors were needed: every ledger
value type round-trips through its existing validating constructor because the
store only ever produces valid values (credit amounts are always positive,
signed amounts non-zero, idempotency keys non-empty) - the fund/refund results
share `fundResultWire` and the accept/reject results share
`reviewedSubmissionWire` (with a shared success/decode helper) so jscpd stays at
0 clones. `corewire` gained `LedgerEntryID` and `CreditAccountID` codecs. The
codec test round-trips every payout/tip/selection variant; the dual-run
integration test (`tests/integration/ledgerbridge_store_test.go`) drives a full
fund -> accept -> refund through the guest against real Postgres and checks the
balance/allocated/entries reads match a direct call byte-for-byte (accept pays a
real `CreditPayout`). The generic store guest and `storehost` route `ledger.*`.
Eight stores bridged (audit, notification, auth, agent, orgcred, assets,
submission, ledger). All gates green. Nothing about the native server or browser
demo changed.

---

The `task/wasi-bridge-submission` branch bridged the `submission` store - the
seventh and widest store - and drove a second generator enhancement.
`submission.Store.CreateSubmission` takes `[]SensitiveField` (a slice of a
package-local type). The generator's `qualify` only handled bare local names, so
`[]SensitiveField` would have become the meaningless `submission.[]SensitiveField`;
it now recurses on the `[]` prefix and qualifies the *element*
(`[]submission.SensitiveField`). Backward-compatible: no existing store method
has a slice-of-local-type argument, so every already-generated `bridge_gen.go`
is unchanged (verified by `check-wasi-bridge`), and a new generator unit test
covers it. `internal/wasibridge/submissionbridge` has codecs for the whole
`submission.Submission` (state, response source, attachments as base64,
validation outcome union, sensitive fields, review note), the `SubmitCommand`,
the `SubmissionComment`, and six result unions (three of which share
`submissionResultWire` since they each carry one submission). Opaque value types
round-trip through their existing reconstruction paths (`ParseState`,
`NewResponseSource`, `NewStoredReviewNote`, `task.NewCommentBody`,
`attachment.NewStoredAttachment`); the receipt-token hash - which had no
from-string path - got a new `submission.ReceiptTokenHashFromString`. `corewire`
gained `SubmissionID`/`SubmissionReceiptTokenID`/`SubmissionCommentID` codecs;
the submission/comment comparison helpers live in
`internal/submission/submissiontest`. The dual-run integration test
(`tests/integration/submissionbridge_store_test.go`) covers create (with an
attachment and a sensitive field) / find / find-by-receipt / list-for-task /
list-for-submitter / the comment thread against real Postgres. The generic store
guest and `storehost` route `submission.*`. Seven stores bridged (audit,
notification, auth, agent, orgcred, assets, submission). All gates green.
Nothing about the native server or browser demo changed.

---

The `task/wasi-bridge-assets` branch bridged the `assets` store (collectibles) -
the sixth store - and drove a generator enhancement. `assets.Store`'s
`ListCollectiblesByOwner(context.Context, string, string, core.Page)` has two
arguments of the same type, which the type-based field naming would collide on;
the generator now suffixes repeated field names (`Query`/`Query2`), which is
backward-compatible (no existing method has repeated arg types, so their
generated files are unchanged - verified). `internal/wasibridge/assetsbridge`
has codecs for `assets.Collectible` (string-wrapper value types: name/kind/
state/policy), the four command structs (fund/refund/gift/award), and the result
unions (accept/reject, collectible, collectible-slice, and collectible-id-slice
payloads), plus a generated `bridge_gen.go`. `corewire` gained a `CollectibleID`
codec; the collectible comparison helper lives in `internal/assets/assetstest`.
The dual-run integration test (`tests/integration/assetsbridge_store_test.go`)
covers create/list/list-by-owner (the two-string method)/gift/task-held against
real Postgres. A new generator unit test covers the repeated-arg disambiguation.
The generic store guest and `storehost` route `assets.*`. Six stores are bridged
(audit, notification, auth, agent, orgcred, assets). All gates green. Nothing
about the native server or browser demo changed.

---

The `task/wasi-bridge-orgcred` branch bridged the `orgcred` store
(organization-wide credentials) - the fifth store - and extracted the shared
agent value-type codecs so the two credential bridges don't duplicate. orgcred's
`Credential` reuses agent's `Label`/`ScopeSet`/`State`, and its
`CreateStoreResult` is a type alias of agent's, so those codecs moved to a new
`internal/wasibridge/agentwire` package that both agentbridge and orgcredbridge
use (the agent bridge was refactored onto it and regenerated with no behavior
change). `internal/wasibridge/orgcredbridge` has its own `Credential`/result
codecs (using agentwire for the shared parts, corewire for ids and the
nullable `*time.Time`), a generated `bridge_gen.go`, and a dual-run integration
test (`tests/integration/orgcredbridge_store_test.go`). `orgcred` gained a
`SecretHashFromString` reconstruction constructor; `corewire` gained
`OrgCredentialID` and nullable-time codecs. The shared credential-field
comparison lives in `internal/agent/agenttest.SharedFieldsDiff`, called by both
the agent and orgcred test comparators (so those don't clone). The generic store
guest and `storehost` route `orgcred.*`. Five stores are bridged (audit,
notification, auth, agent, orgcred). All gates green. Nothing about the native
server or browser demo changed.

---

The `task/wasi-bridge-agent` branch bridged the `agent` store (MCP agent
credentials: create/verify/list/revoke) - the fourth store. It stretched the
codec vocabulary with **nullable pointer fields** (`*time.Time`, `*core.TaskID`,
carried as empty-when-nil strings) and a **scope set** (serialized as a string
list). Like auth, the opaque `SecretHash` round-trips via a new
`agent.SecretHashFromString` reconstruction constructor; `corewire` gained
`TaskID` and `AgentCredentialID` codecs. The generated `bridge_gen.go` and
hand-written codecs are dual-run-verified against real Postgres
(`tests/integration/agentbridge_store_test.go`), and the credential comparison
helper lives in `internal/agent/agenttest` (matching the audittest /
notificationtest pattern). The generic store guest and
`internal/wasibridge/storehost` now route `agent.*`. Four stores are bridged
(audit, notification, auth, agent). All gates green. Nothing about the native
server or browser demo changed.

---

The `task/wasi-auth-route` branch proved an auth-store-touching route runs end
to end through the guest. `internal/wasibridge/appmux` now wires a live auth
service (`auth.NewService` over the bridged auth `GuestStore`) alongside the
notification service, so `GET /api/users` - which reads the auth store's
directory via `authService.ListUsers` - is served entirely by the wasip1 guest,
with the read bridged back to real Postgres. The integration test
(`tests/integration/authroute_test.go`) mints an access token, seeds a user, and
asserts the guest's response is byte-identical (status, Content-Type, body) to
the same mux run in-process over the real store, and contains the seeded user.
This is the first route to exercise a bridged store's *service* (not just
stateless token verification). The host-side store routing (audit/auth/
notification by method prefix) was extracted into `internal/wasibridge/storehost`
and is shared by the app host and the tests. All gates green. Nothing about the
native server or browser demo changed.

---

The `task/wasi-bridge-auth` branch bridged the `auth` store - the largest and
most complex (13 methods, 10 result unions, a three-variant Subject union,
record and token structs). `internal/wasibridge/authbridge` has hand-written
codecs plus a generated `bridge_gen.go`, dual-run-verified against real Postgres
across all 13 methods (`tests/integration/authbridge_store_test.go`): credential
create/lookup/list, the account mutations, guest subject, and the refresh-token
and account-token store/consume/revoke flows. Auth exercised the codegen's
edges: (1) its opaque hash/token types (`RefreshTokenHash`, `AccountTokenHash`,
`AccountTokenKind`) have no public "from string" constructor, so `internal/auth`
gained reconstruction constructors (`RefreshTokenHashFromString`,
`AccountTokenHashFromString`, `AccountTokenKindFromString`, mirroring the
existing `ParsePasswordHash`) that storage adapters use to reload a stored hash;
(2) `corewire` grew codecs for `GuestID`/`RefreshTokenID`/`OrganizationID` plus
plain-`string` and `time.Time` method arguments; (3) the generator learned to
leave Go builtins unqualified and to import `time` when a method takes a
timestamp. The generic store guest now routes `auth.*` too. Three stores are
bridged (audit, notification, auth). All gates green. Nothing about the native
server or browser demo changed.

---

The `task/wasi-app-route` branch tied the Phase 3 store bridge to the Phase 4
HTTP hosting: a real authenticated, store-touching route
(`GET /api/notifications`) now runs end to end through the wasip1 guest. Access-
token verification is stateless (signature + clock, no store), so the route
needs only the verifier plus the already-bridged notification store. The app
guest (`cmd/sharecrop-wasi-app-guest`) builds the real `internal/http` mux via a
shared `internal/wasibridge/appmux` helper with a live notification service
backed by the generated `GuestStore`; the request runs through an `httptest`
recorder in-guest, and the notification read RPCs back to the host over the same
unit of work and hits real Postgres. `cmd/sharecrop-wasi-app-host` is the
production-shaped `net/http.Server` for it: a fresh guest per request, store
calls dispatched to `internal/db` by method prefix, and the token secret handed
to the guest via WASI env (new `rpc.Host.WithGuestEnv`). The request-bridging
handler moved into a shared `httpbridge.Handler` used by both the Phase 4 health
host and the app host. The integration test
(`tests/integration/approute_test.go`) asserts the guest's response is byte-
identical to the same mux run in-process against the same store, and that the
body actually contains the seeded notification (so the bridge read real data,
not an empty list). All gates green. Nothing about the native server or browser
demo changed.

---

The `task/wasi-bridge-multistore` branch started the post-spike implementation
effort: generalize the bridge codegen beyond one hard-coded store and prove it
scales. `internal/wasibridge/gen` is now store-agnostic — each store is a
`storeSpec` naming its codecs, and `go run ./cmd/sharecrop generate wasi-bridge`
regenerates every store in `gen.Targets()`. Shared core-type codecs (typed ids,
page, time) moved to a new `internal/wasibridge/corewire` package so the
per-store bridges don't duplicate them (the audit bridge was refactored onto it
and regenerated with no behavior change). `internal/notification.Store` is
bridged as the second store (`internal/wasibridge/notificationbridge`):
hand-written round-trip-tested codecs for `Notification` and its three result
unions, plus a generated `bridge_gen.go`, dual-run-verified against real
Postgres (`tests/integration/notificationbridge_store_test.go` — create/list/
mark-read through both the direct-db path and the compiled guest, incl. the
not-found rejection and the write path). One generic guest
(`cmd/sharecrop-wasi-store-guest`) now routes every bridged store by method
prefix, replacing the audit-specific guest. Shared test-comparison helpers moved
to `internal/audit/audittest` and `internal/notification/notificationtest` to
avoid duplication. `check-wasi-bridge` regenerates and diffs both bridges. All
gates green (policy, dead-code, copy-paste, vet). Remaining stores follow the
same recipe. Nothing about the native server or browser demo changed.

---

The `task/wasi-spike-phase4` branch executed Phase 4 — the final phase of the
WASI production hosting spike: one real HTTP request end to end through the
guest. `cmd/sharecrop-wasi-http-host` runs a real `net/http.Server` whose
handler, per request, serializes the request
(`internal/wasibridge/httpbridge`), runs a fresh wasip1 guest
(`cmd/sharecrop-wasi-http-guest`) over the Phase 3 `rpc` transport, and writes
the serialized response back. The guest builds the **real production mux** —
`httpserver.New(...)`, the same routing table `cmd/sharecrop serve` builds
(services are nil for this slice since `GET /healthz` touches none) — and runs
the request through an `httptest` recorder, exactly as the browser demo does. A
probe confirmed the real mux constructs and serves `/healthz` with nil services.
The integration test (`tests/integration/httpbridge_test.go`) asserts the
response through the compiled guest is byte-identical (status, `Content-Type`,
body) to the same mux run in-process, for both `GET /healthz` (200) and an
unknown `/api` route (404); a real `curl` against the host returns
`200 {"status":"ok"}` through the wasm guest. This is the last spike checkpoint:
the real `internal/http` handler runs inside a wasip1 guest behind a native HTTP
server, output-identical to the native path. With all four phases verified, the
spike is complete and the direction is confirmed (see the "After Phase 4" note
in `docs/wasi_production_hosting_spike_plan.md`); the follow-up is the full
implementation effort, not part of the spike. Nothing about the native server or
browser demo changed.

---

The `task/wasi-spike-phase3` branch executed Phase 3 of the WASI production
hosting spike: codegen the bridge for one full store, with a CI drift gate and
dual-run tests. It added a generic method-keyed transport
(`internal/wasibridge/rpc`, generalizing Phase 2's hand-wired shape) and pulled
the framing and the `DomainError` codec into shared packages
(`internal/wasibridge/wire`, `.../domainwire`) that both the auth spike and the
audit bridge now use. `internal/wasibridge/auditbridge` bridges the whole
`internal/audit.Store` (Record/Get/List): hand-written, round-trip-tested
per-type codecs (`codec.go`, covering typed ids, a string-wrapper enum, nested
structs, three sealed-union filters, a slice, `time.Time`, and `DomainError`)
plus a generated `bridge_gen.go` (the `Dispatch` host router and the
`GuestStore` client) emitted by `internal/wasibridge/gen` via
`go run ./cmd/sharecrop generate wasi-bridge`. A `check-wasi-bridge` Make target
+ CI step regenerates and diffs (mirroring `check-openapi`), and the generator
errors loudly on a Store method whose type has no registered codec
(`gen_test.go`). The dual-run integration test
(`tests/integration/auditbridge_store_test.go`) runs every method against real
Postgres through both the direct-`internal/db` path and the compiled
`GOOS=wasip1` guest (`cmd/sharecrop-wasi-audit-guest`) + host bridge and asserts
they match, including the not-found rejection and the write path;
`cmd/sharecrop-wasi-audit-host` is the manual counterpart. The generated
`GuestStore` carries a `var _ audit.Store = GuestStore{}` assertion, so a new
interface method without regeneration fails to compile. Anti-drift safeguards
#1 (no business logic in the bridge), #2 (codegen + gate), and #3 (dual-run) are
all in place. Nothing about the native server or browser demo changed. The
Phase 2 auth spike was refactored onto the shared `wire`/`domainwire`/`rpc`
packages with no behavior change (its tests still pass). Findings and the
Phase 3 checkpoint are recorded in
`docs/wasi_production_hosting_spike_plan.md`.

---

The `task/wasi-spike-phase2` branch executed Phase 2 of the WASI production
hosting spike: proving one real store method round-trips from a `GOOS=wasip1`
WASM guest to real Postgres and back. New package `internal/wasibridge` holds
a length-prefixed-JSON stdin/stdout RPC (`protocol.go`), a guest-side client
that calls the store over that pipe instead of driving pgx (`guest.go`), and a
native wazero host that instantiates a *fresh* guest per unit of work — driven
by exactly one goroutine, with a pump goroutine servicing the guest's storage
calls against the real `db.NewAuthStore(pool)` (`host.go`, `!wasip1`). The
guest command (`cmd/sharecrop-wasi-spike-guest`, compiled to wasip1) and a
native host CLI (`cmd/sharecrop-wasi-spike`) exercise it. The bridged method is
`AuthStore.FindCredentialByEmail` — the smallest real read, chosen so the spike
covers the found, missing, and rejected (`DomainError`) paths. Fast unit tests
cover the wire format and the guest transport (`protocol_test.go`); an
integration test drives the full guest→host→Postgres path
(`tests/integration/wasibridge_store_test.go`), asserting found matches a
direct store call field-for-field, missing returns `CredentialMissing`, and
rejected preserves the `core.ErrorCode` across two serialization crossings. It
also measures fresh-instance-per-call cost (~2.7ms idle, instantiation-
dominated). The bridge holds no business logic (anti-drift safeguard #1);
codegen and dual-run (safeguards #2/#3) come in Phase 3. Nothing about the
existing native server or browser demo changed. Findings recorded as #7/#8 in
`docs/wasi_production_hosting_spike_plan.md`, which also marks Phase 2 done and
resolves the "After Phase 2" decision point (error-shape round-tripping is not
lossy).

---

The `task/pages-verify-and-boyscout` branch verified the deployment and
corrected framing. It confirmed the deployed GitHub Pages demo works end to
end after the #142 merge (spendable/allocated wallet, "Task funding" ledger
kind, the per-task funding line and refund gating, populated inbox and
collectibles, no console errors). It corrected the imprecise "two backends"
language in the continuity docs: there is one backend (the `internal/http`
mux plus the domain services), with two storage adapters bound to the domain
`Store` interfaces — `internal/db` (Postgres, for the server) and
`internal/wasmdemo` (browser key/value, for the demo). A browser cannot open
a Postgres connection, so a browser-local storage adapter is unavoidable for
the demo; the two adapters are kept behaviorally identical by the shared
scenario-parity suite. Running production on the same compiled WASM artifact
via a WASI host (bridging storage to `internal/db`) remains the tracked goal
in `DO_NEXT.md`. Also fixed stale escrow wording in the marketing shell, made
the organization operations dashboard show Spendable + Allocated (matching the
wallet model), corrected three success confirmations that were rendered in
red failure styling (org-team create, saved-view save), and added a
missing empty-input guard on the add-team-member form.

---

The `task/wallet-followup-and-boyscout` branch is a post-#141 follow-up.

Per-task funding on the task detail. The task response now reports
`allocated_credits` and the individual `allocated_collectible_ids` a task
currently holds (collectibles are non-fungible and tracked individually, not
counted), read from `task_funds` / `task_fund_collectibles` on both backends
via new `ledger.TaskAllocatedCredits` and `assets.TaskHeldCollectibles`
methods. The detail, create, and state-change (open/cancel/unpublish)
responses all carry the live figures. The browser gates the Refund button on
actual funding (no more Refund on an unfunded declared reward), shows a
per-task "Allocated to this task" line, and shows the prominent fund callout
for any unfunded draft (not just no-reward drafts).

Boy-scout fixes from a fresh two-agent review. Client validation for a
collectible/bundle reward with nothing selected (was an avoidable 400);
Cancel now refetches the wallet and ledger like Fund/Refund/Accept; the
stale "escrowed reward" cancel copy now describes the wallet model; the
raw-JSON submit editor seeds from the structured fields the worker already
filled; the login and password-reset controls are separate forms so Enter in
a reset field no longer attempts a login; the review Accept/Reject/Request-
changes buttons got `type="button"`. Security: `parseHashString` rejects a
zero-length salt or key so a malformed stored Argon2id hash cannot act as a
universal password. Data hygiene: refunding or cancelling a task now releases
the worker's reservation (to `cancelled_by_requester`) on both backends
instead of leaving it dangling in an active/submitted state on a cancelled
task.

---

The `task/journey-review-fixes` branch, after the review pass below, also
replaced the credit-escrow system with a two-section wallet and switched
password hashing to Argon2id.

Two-section wallet. Every user and organization credit account now has a
**spendable** section (still `sum(ledger_entries.amount)`) and an
**allocated** section (`sum(task_funds.credit_amount)` for tasks the account
funded). The escrow state machine (`task_escrows` with held/released/refunded
and the parallel `task_collectible_rewards`) is gone, replaced by stateless
temp-store tables: `task_funds(task_id, funder_account_id, credit_amount)`
and `task_fund_collectibles(task_id, collectible_id)` — a row exists while
the task holds the reward, deleted on award or refund. Funding moves credits
from spendable to allocated (a `task_fund` ledger entry, formerly
`task_escrow`), so allocated credits cannot be double-spent; funding
validates the spendable section only. Accepting a submission moves the
funder's allocated credits into the worker's spendable balance (a
`task_payout`, remainder refunded on a partial payout). Refunds are
auto-granted for the task owner or the user holding the active reservation
while the task is not yet awarded, returning the allocated credits to the
funder's spendable balance and cancelling the task; cancelling a funded
un-awarded task settles the same way. (This intentionally relaxes the
earlier "block refund while a submission is pending review" guard — the
requester can refund by default.) Held reward collectibles stay in the
collectible's `escrowed` lifecycle state as the trade lock. Balance
endpoints now return `{spendable_credits, allocated_credits}`; the Overview
and organization pages show spendable plus a locked-to-tasks line. Both
backends (Postgres and the WASM demo) implement this identically, verified
by the shared scenario-parity suite.

Password hashing. Replaced PBKDF2-HMAC-SHA256 with Argon2id (OWASP's
first-choice algorithm, m=19 MiB, t=2, p=1) via
`golang.org/x/crypto/argon2` — the dependency PLAN.md/AGENTS.md already
specified for `internal/auth` but which the code had never actually used.

---

The `task/journey-review-fixes` branch was a review-driven fix pass: four
parallel read-only reviews (security, backend correctness/parity, Elm
client UX, docs accuracy) plus hands-on browser walking of the demo, then
fixes for the confirmed findings.

Backend correctness and real-vs-WASM parity (16 findings). Made
`internal/db` `ChangeTaskState` transactional with an expected-prior-state
predicate so concurrent cancel/fund/refund can no longer orphan escrow or
reopen a cancelled task. Added guards, on both backends, that reject
refund/cancel while a submission is pending review and reject
self-deactivation while the user owns tasks holding escrow. Closed store
divergences where `internal/wasmdemo` behaved differently from
`internal/db`: collectible-reward count derivation (a mismatch had poisoned
task parsing), opening a funded collectible/bundle task, cancel blocking on
held collectibles, reservation expiry sweeps, one-active-reservation-per-
task, implementor bans, three idempotency semantics (refund replay, fund-
key reuse, accept replay payout reconstruction), ledger pagination past
idempotency markers, series membership bookkeeping, team scope including
team-reserved tasks, unknown-task 404 (was 409), member-provision and
member-role edge cases, and partially-released collectible refunds. Made
pagination strict across all list endpoints, propagated a swallowed
session-revocation error in `UpdatePassword`, and added audit events for
funding. Tests accompany each fix (store unit tests, integration guards,
and shared scenario-parity assertions that run against both backends).

Security. Account-token delivery now defaults to `log` (fail closed):
production `serve` no longer returns password-reset or email-verification
tokens in the HTTP response, closing an account-takeover path where anyone
knowing a victim's email could reset their password; the browser demo opts
into `api` explicitly (browser-local, no real accounts). Password-reset
returns the same neutral response for unknown and known emails, removing an
account-enumeration oracle. The token-confirm endpoints and the
email-verification request are now IP rate limited. The user-directory
search escapes LIKE metacharacters and caps the query length. The demo
shell rebuilds its `<base href>` from validated path segments so a crafted
URL cannot point script loads at another origin.

Elm client. Logging in on a deep link now loads that page's data (it stayed
on "Loading…" before). Added mid-session token refresh with an explicit
"session expired" logout, so a tab left open past the 15-minute access-token
lifetime recovers cleanly instead of silently failing every request into
fake empty states. Added a schema-driven worker response form: a task with a
structured response schema renders one typed input per field (with a raw-JSON
escape hatch) instead of a bare `{}` textarea. Fixed a silent reward
downgrade where a credit/bundle reward with a blank amount was created as
no-reward (now blocked with inline validation). Stopped `Ui.disclosure`
panels from snapping shut mid-edit by deriving their open state from live
form fields. Made successes and failures visually distinct (a typed
`Note`), replaced false "instructions sent" copy under the no-email scope,
hid the guest button outside the demo, gated the collectible trade/tip and
funding pickers to states the server accepts, disabled the Next pagination
button on the last page, added self-service privacy-request listing, added
standalone-team creation and browsable team pages, added a deactivate-
account confirmation step, and surfaced previously-swallowed revoke/award
errors.

Demo seed. Added review-side data targeting the single demo actor (mara): a
pending submission on a dedicated review task, an approval-required
reservation request, a matching inbox notification, and a held collectible —
so the review, approval, and collectible journeys are demonstrable. The
funded fraud task is kept free of pending work so the demo refund flow still
applies to it.

Docs. Refreshed the continuity files and fixed stale/wrong claims across
`docs/` and `site/` (the MCP scheduling recipe now does the initialize
handshake; operations/readiness docs corrected about Postgres-backed
production storage; user-stories/onboarding corrected about the demo and
self-registration; api_reference route list and counts updated; the deployed
docs' broken demo link and stale scope list fixed).

Deferred: exposing held-escrow/funding state on the task detail (to hide the
Refund button on an unfunded task and show funding status) — a separate
multi-layer change tracked in `DO_NEXT.md`.

PR 140 was an audit pass after the WASM cutover: it fixed authorization
checks, WASM demo parity gaps, and Elm client issues found by review.

PR 139 cut the browser WASM demo over to the real backend.
`cmd/sharecrop-wasm/main_js_wasm.go` now builds the real `internal/http`
mux over the real domain services (the same services `cmd/sharecrop` wires
against Postgres) and serves every demo request through `mux.ServeHTTP`
via `httptest` request/recorder pairs, with identity resolved from real
bearer tokens and the refresh-token cookie replayed in Go.
`internal/wasmdemo`'s separate request classification/handler layer
(including `request_handler.go`) was removed; demo seeding goes through
the real services (`internal/wasmdemo/seed.go`). The demo is no longer a
parallel reimplementation of routing, validation, or business rules.

PR 138 added browser-storage-backed `Store` implementations for all 9
domain packages (`internal/wasmdemo/browserstore_*.go`): auth,
notification, agent credentials, organization credentials,
organizations/teams, ledger, tasks/reservations/comments/series, assets,
and submissions — all built on the shared `BrowserStorage` key/value
primitives so the real, unmodified domain services can run in the browser.

`task/unpublish-escape-hatch-and-scenario-parity-ci` (PR 137) traced the user's
report ("the task view still! still! does not have a visible drawer to
change funding of a task") to a real architectural root cause, verified
live against the actual deployed demo rather than assumed.

Read `taskRow`'s canFund logic (`web/elm/src/Sharecrop/View.elm`): it only
offers funding for a task in `draft` state, on the stated assumption that
"once a task is open, a credit/bundle reward is always already funded (the
backend requires funding before opening)". Checked that assumption by
directly reading `internal/task/service.go`'s `Open`/`OpenState` - neither
checks reward or escrow at all. Verified empirically (curl against a real
local server, not just code reading) that the *actual* enforcement lives
in `internal/db/task_store.go`'s `requireOpenableReward`, called from
`ChangeTaskState` - a Postgres store-layer check, not something in the
shared `internal/task` domain package. `internal/wasmdemo` is a wholly
separate reimplementation of task state transitions (its own storage, its
own checks) and its "open" handler
(`internal/wasmdemo/request_handler.go`) never got this invariant added -
confirmed by creating a task with a declared-but-unfunded credit reward
against a local demo build and successfully opening it, which the real
backend rejects with 409.

Took a real screenshot of the live deployed demo
(`https://e6qu.github.io/sharecrop/demo/`) to check whether the specific
seeded task the user pointed at ("Verify 10 ledger transfers for fraud
signals") was actually in this broken state. It wasn't: reading
`site/demo/wasm-host.js`'s seed script showed every seeded task's
`reward_credit_amount`/`escrow_amount` pair matches (explicit values, or
both falling back to the same `|| 25` default) - so the seed data itself
is internally consistent. The user's actual need was simpler and more
general: there was no way to add or change funding on *any* open task
through the UI at all, regardless of whether it happened to be correctly
funded.

Fixed both things:

1. Added the missing invariant to `internal/wasmdemo/request_handler.go`'s
   "open" case, mirroring the real backend's check exactly - so a *new*
   task created through normal demo interaction (not seed data) can no
   longer reach "open" without its reward actually being funded, closing
   the gap for the future.
2. Found and wired up a real escape hatch that already existed
   server-side on both backends but had no UI: `Task.Unpublish`
   (`open` → `draft`, `POST /api/tasks/{id}/unpublish`) was fully
   implemented and even had a UI button for task *series* - just never for
   individual tasks. Added the button plus the
   `UnpublishTaskClicked`/`UnpublishTaskReceived`/`Api.postUnpublishTask`
   chain. An owner can now move an open task back to draft, use the
   already-existing draft-only funding panel, and reopen - solving the
   user's stated need directly, independent of the underlying invariant
   question.

Verified the reward/escrow invariant fix is a real check, not vacuous: ran
the shared `tests/scenario_parity/scenario.ts` (a new assertion added
there: attempt to open before funding, expect 409) against a build of the
WASM demo *without* the fix first, confirmed it actually fails
(200 instead of 409), then restored the fix and confirmed it passes. This
directly answered the user's "I think we might be doing fake checks"
concern for this specific check.

That investigation surfaced a bigger, systemic gap the user pushed on
directly: `check-wasm-scenario-parity` (the CI job that runs the shared
scenario script) only ever ran it against the WASM demo -
`check:scenario-parity`, the same script's real-backend variant, had *no
CI job at all*. A new assertion could be added to the shared script,
verified once locally against whichever backend was handy, and never
checked against the other one in automation again - exactly how a
divergence like the missing invariant could persist unnoticed. Wired
`tools/run_local_real_scenario_parity.ts` into `tools/run_db_checks.sh`
(spins up a real DB-backed server in the background, waits for health,
runs the shared scenario against it, tears down) and added the missing
Deno setup step to the `db-checks` CI job. Hit a real bug while wiring
this up: an unredirected backgrounded `go run ... serve &` inside the
script left its compiled-binary child process holding the script's output
pipe open indefinitely after a plain `kill` on the `go run` wrapper PID -
redirected its output to a temp file and added `pkill -P` cleanup for the
child process too.

The user separately raised a much bigger architectural question directly
tied to this bug class: unifying onto a single WASM-compiled backend for
both the browser demo and a multi-replica production deployment, instead
of two independently-reimplemented backends that can silently diverge on
invariants like this one. Asked a clarifying scope question (full
unification now vs. a scoped-down stepping stone vs. tactical-only) and
got no response within the session; proceeded with the concrete, verified
fixes above rather than guessing at scope for what would be a
multi-session rewrite. Flagged as still open in `DO_NEXT.md`.

---

`task/reservation-fixes-and-reward-badges` fixed two reservation bugs the
user hit live ("I can't cancel reservation", "the reservation drawer is not
visible when toggled off") plus a follow-up request for reward badges with
icons. Debugged entirely by real browser reproduction with screenshots
before touching any code, per explicit instruction not to guess.

Reproducing "can't cancel reservation" surfaced two independent, stacked
bugs, both in `internal/task/service.go`:

1. `ListReservations` called `requireOwnerPermission` unconditionally,
   rejecting anyone who wasn't the task's owner. A worker who reserved a
   task got an empty list back from `GET /api/tasks/{id}/reservations` -
   their own reservation (and its Cancel button) never rendered at all,
   even though `reservationButtons` already had `isHolder`-based logic
   ready to show it. Fixed by widening the permission: the owner still sees
   every reservation; anyone else sees only the reservation(s) they
   themselves requested (not other workers' attempts on the same task -
   privacy, not just unblocking).
2. Once the row *was* visible (after fix 1), clicking Cancel still 403'd:
   `CancelReservation` reused `changeReservationByRequester`, the same
   owner-only helper as `ApproveReservation`/`DeclineReservation`. But
   cancelling isn't meant to be owner-only - the worker who holds the
   reservation needs to release it themselves. Gave `CancelReservation` its
   own permission path (owner, or the reservation's own requester) instead
   of sharing the approve/decline helper.

A live screenshot taken while verifying fix 1 turned up a third, related
bug: after reserving, the "Reserve" button never disappeared, and the top
badge still read "reserve" instead of reflecting the reservation just
made. Traced to `taskViewerAction` (`internal/http/tasks.go`), which
computes `viewer_action` purely from the task's own state/participation
policy - it has no notion of *which viewer* is asking, so every viewer
sees the same "reserve" action regardless of whether they already used
it. `taskToResponseForActor` (already used by `getTask`, the exact
endpoint the browser's task-detail page calls) now overrides
`viewer_action` to `wait` when the actor already holds a requested/active
reservation on that task - reusing the same widened `ListReservations` to
check. Verified with a real Go http_e2e test since this needed checking
from two different viewers' perspectives at once (the reserving worker
sees `wait`; an uninvolved viewer still sees `reserve`), not just a
Playwright screenshot.

Separately, per boy-scout rule, added a reward badge to task list rows: the
reward previously rendered as muted trailing text ("· 20 credits"); it's
now its own small badge (new `reward` tone, purple, verified at 7.39:1
contrast) with a `◆` icon, matching the treatment task-state badges
already got in the prior PR. Also gave all 5 state badges a small
decorative icon (`●`/`○`/`✓`/`✕`/`⏳`, `aria-hidden` since the badge's own
text already names the state - WCAG 1.4.1 still holds, the icon adds
nothing load-bearing) - this was the "mini icons for the badges" half of
the same follow-up request.

---

`task/task-visual-language-and-multiselect-filter` addressed a batch of
task-UI requests, prefaced by a design review the user approved before any
code was written: a palette proposal (5 badge tones, 4 already existing
plus one new "info"/blue), WCAG contrast numbers for each, and mockups of
the funding-callout and field-validation treatments, published as an
artifact.

Confirmed the "funding is invisible" complaint by live reproduction rather
than assuming: the "Fund this task" control did exist (a collapsed
`<details>` disclosure), but sat unstyled among unrelated collapsed
sections ("API & MCP", "Report task") with nothing marking it as the next
step - a discoverability problem, not a missing feature. Fixed by showing
an open-by-default blue callout ("Before you open this task...") only when
a task is a draft with no reward yet (`needsFundingGuidance` in
`ownerControlsCard`); once funded, or for a task that already declared a
reward, the original disclosure renders as before.

Task list rows (`taskRow` in `View.elm`, both the My-tasks and
Discover-public-tasks lists) now show state as a colored badge
(`taskStateBadge`) instead of plain text, and get a blue left-accent border
plus a small "MINE" tag when the viewer created the task or is its active
assignee (`isMyTask`). Added a 5th badge tone, `info` (blue,
`bg-blue-100`/`text-blue-800`, 7.15:1 contrast), to `Ui.badgeToneClass` -
needed because `Closed` previously shared "neutral" (slate) with `Draft`
and wasn't visually distinct from it.

Added per-field required-field validation to the create-task form
(`createTitleInvalid`/`createDescriptionInvalid` in `LoggedInModel`,
new `Ui.textInputToned`/`Ui.textareaToned`/`Ui.fieldError`): a field
gets a muted-red border + inline message only after a submit attempt
fails with it empty, and clears as soon as the user fills it in. Found a
real WCAG tension while designing this: a genuinely *muted* border can't
independently clear the 3:1 non-text contrast target (`red-400` on white
measures 2.77:1) - and neither do this app's own existing input borders
(`slate-200` on white measures 1.23:1). Resolved by never relying on color
alone: the border is always paired with an icon + explicit error text
(WCAG 1.4.1), which is also why a plain color change was never a
sufficient fix on its own.

Replaced the single-select task-state filter (5 buttons, one of which was
a redundant "All", covering only 3 of 5 real states) with multi-select
chips covering all 5, letting several states combine (e.g. "Open" +
"Closed"). This needed a genuine backend addition, not just a UI change:
the server only supported one `state=` value at a time
(`task.StateEquals` in `internal/task/service.go`). Added `task.StateIn`
plus a `tasks.state = some(@filter_states)` SQL clause
(`internal/db/task_store.go`) - written as `some(...)` rather than the
otherwise-idiomatic `any(...)` (an exact Postgres synonym) purely to dodge
this project's `check:policy` weak-wildcard-type lint, which flags the
literal word "any" in Go source including inside SQL string literals.
`internal/http/tasks.go`'s `parseTaskListFilters` now reads repeated
`state=` query params instead of just the first. Mirrored in
`internal/wasmdemo`'s `ListTasks`, which now takes `[]string` and matches
via a set instead of a single string comparison.

New tests: a Go http_e2e case for the multi-state backend filter: an
integration test isn't needed since http_e2e already exercises the store
through the real route; several new Playwright tests (field validation
highlighting and clearing, the funding callout appearing/disappearing
correctly across both the real and demo backends, multi-select filter
combining and un-combining states). Verified visually with real browser
screenshots of the task list (colored badges + mine tags) and the funding
callout before considering this done.

---

`task/fund-any-reward-kind-and-open-on-create` covered two of a batch of
related feature requests. (1) A draft task is now always fundable by its
creator regardless of the reward kind it was created with: funding a
none-reward task transitions it to `credit`, funding a collectible-only task
transitions it to `bundle`, both with the funded amount becoming the
declared credit amount. Previously `requireCreditRewardFunding` rejected
funding outright unless the task already declared `credit`/`bundle`.
Implemented as a new `requireFundableTask` helper in
`internal/db/ledger_store_helpers.go` (combining the draft-state check,
reward validation, and the reward_kind transition in one place shared by
personal and organization funding — extracted after `jscpd` correctly
flagged the two call sites duplicating that logic when first written
separately), mirrored in `internal/wasmdemo/request_handler.go`'s funding
case for the demo backend, and in the Elm client by simplifying the fund
panel's visibility gate (`canFund`) from "draft task with credit/bundle
reward" down to just "draft task." New regression coverage: a Go
integration test (`TestFundTaskWithoutADeclaredRewardTransitionsRewardKind`)
and Playwright tests against both the real and demo backends.
(2) Creating a task now opens it in the UI for further editing, with the
browser URL updating to `#/tasks/{id}` (`Main.elm`'s `CreateTaskReceived`
now calls `enterPage` + `Nav.pushUrl` instead of resetting the create form
and staying on it) — this changed the assumption behind roughly 13 existing
Playwright tests that expected to remain on the create page after
submission, all updated to check the new detail page instead.

The same task continued with the remaining three items from that batch.
(3) A creator can add a collectible to an existing task's reward from the
task detail page: the backend route
(`POST /api/tasks/{task_id}/collectible-reward`) and its Elm API wrapper
(`postCollectibleReward`/`AwardReceived`) already existed but were never
wired to any UI trigger. Added a draft-only "Add a collectible to this
task's reward" panel that reuses the existing plumbing, syncing
`awardTaskId` to the viewed task the same way `fundTaskId` already was.
This surfaced two real bugs, fixed the same way as the credit-funding
transition: `internal/db/collectible_store.go`'s `FundCollectibleReward`
never transitioned the task's `reward_kind` (none -> collectible,
credit -> bundle), so a task that received its first collectible via this
path would still show "No reward" in the UI despite holding an escrowed
collectible; and neither the fund nor the award success handlers
re-fetched the viewed task's own detail record, so the owner-controls
buttons (e.g. a refund button appearing) stayed stale until a manual page
reload — added `Api.refreshLedgerAndTaskDetail` and used it from both
`FundReceived`/`AwardReceived`. Caught the staleness bug by manually
driving a real browser through the flow and noticing the confirmation
message disappeared the instant the org's collectible count hit zero (see
item 5 below) — not by an automated test, though one now covers it.

(4) Checked whether an admin can award collectibles to any user: already
fully built, both backend (`POST /api/collectibles/award`, platform-admin
gated, catalog-only, supports user/team/organization recipients) and Elm
UI ("Admin: award a default collectible" on the Collectibles page). No
work needed.

(5) An org admin can award an org-owned collectible to an active member —
genuinely new, unlike items 3/4. The existing collectible-transfer
mechanism (`GiftCollectible`) hard-requires personal (`owner_kind: user`)
ownership, so it couldn't be reused for an org-owned collectible. Added
`assets.Service.AwardOrganizationCollectible` /
`CollectibleStore.AwardOrganizationCollectible`
(`internal/db/collectible_store.go`), gated by `org.PermissionManageMembers`
at the HTTP boundary (new route
`POST /api/organizations/{organization_id}/collectibles/{id}/award`,
`internal/http/collectibles.go`), verifying in the same transaction that
the collectible actually belongs to that org and the recipient is an
active member (`organization_memberships` query). New Elm UI on the
organization detail page: a member picker plus an "Award to member" button
per org-owned collectible, added to the existing "Collectibles (N)"
disclosure. Covered by a new HTTP e2e test
(`TestOrganizationAdminAwardsOrganizationCollectibleToAMember`: permission
denied for a non-admin member, rejected for a non-member recipient, happy
path, and rejected on a second award attempt since the collectible no
longer belongs to the org). Not covered by an automated Playwright test —
the shared Playwright server has no platform admin bootstrapped (needed to
get a collectible into org ownership in the first place, since minting
directly to an org isn't exposed to non-admins), so this was verified
manually instead: started a separate server with `SHARECROP_ADMIN_USER_IDS`
set, drove a real Chromium browser through mint -> award-to-org -> award-to-
member, and confirmed the success message ("Awarded Golden Sickle
(minted).") renders correctly even after the org's collectible count drops
to zero.

`internal/wasmdemo` does not implement item 5's new endpoint — the demo
supports item 3 (adding a collectible to a task's reward) but not the
org-admin-awards-to-member flow. Not addressed in this task.

Also trimmed `STATUS.md` from 576 to about 70 lines: nearly all of the
removed content was stale historical narrative from long-merged PRs
(the RBAC effort, PR 122's clean-up pass, an old task-series bug fix),
already captured in this file, that had accumulated without ever being
trimmed — contrary to `AGENTS.md`'s own instruction to keep `STATUS.md`
short and current-state-only.

---

`task/fix-fund-panel-and-demo-status-codes` fixed a user-reported bug found
by live reproduction, not by guessing: funding a task from its detail page
returned "status 500" against the demo (WASM) backend. Traced to
`cmd/sharecrop-wasm/main_js_wasm.go` unconditionally mapping every
`wasmdemo.RequestHandleRejected` to HTTP 500 regardless of cause — a plain
validation rejection like "task is already funded" looked identical to a
real crash. Fixed by changing `RequestHandleRejected.Reason` from a plain
`string` to `core.DomainError` (reusing the real backend's existing
`core.ErrorCode` taxonomy rather than inventing a parallel one), classifying
all 316 call sites across `internal/wasmdemo/{interaction_handler,
request_handler,runtime_handlers}.go` and `main_js_wasm.go` by keyword into
the correct code (`invalid_argument`/400, `not_found`/404, `conflict` or
`invalid_state`/409, `permission_denied`/403), and adding a `statusForError`
mapping in `main_js_wasm.go` mirroring `internal/http/server.go`'s.
Cross-checked one classification against the real backend's own code for the
same business rule ("task requester cannot reserve their own task" →
`ErrorCodeConflict` in both) to validate the classification approach.
Verified live before and after: the exact repro (fund an already-open,
already-funded demo task) went from "status 500" to "status 409."

Also fixed, per the user's explicit request: removed the task-list row's
"Fund" shortcut link entirely (it only ever navigated to the task's detail
page — funding now happens only from there). Fixed the detail page's
"Fund this task" panel to show only for a **draft** task with a
**credit/bundle** reward, matching the pattern the adjacent refund button
already used — it previously showed for any draft-or-open task regardless
of reward kind, including already-funded open tasks (a credit/bundle
reward must be funded before a task can be opened at all, so an open one is
always already funded) and none/collectible-only tasks that can never
accept credit funding, both of which would always end in a rejection.

Regression tests: `internal/wasmdemo`'s existing suite covers the
`RequestHandleRejected`/`core.DomainError` change; two new Playwright tests
(`tests/playwright/ledger.spec.ts`, `tests/playwright/demo.spec.ts`) assert
the fund panel is absent on an already-funded, open task against both the
real backend and the demo backend; a new demo test funds a freshly-created
draft credit task end to end; `tests/playwright/mobile.spec.ts`'s existing
overflow check now creates its own draft task rather than relying on the
removed task-list shortcut.

---

`task/deprecate-demo-backend-js` removed `site/demo/backend.js` (the
hand-maintained JS mock backend used before the demo defaulted to WASM) and
its two Deno tests (`tests/deno/demo_backend_test.ts`,
`tests/deno/scenario_parity_test.ts`), sequenced deliberately: PR 129 first
wired `deno task check:scenario-parity:wasm` into CI as a new
`wasm-scenario-parity` job and proved it green, *then* this task deleted
what it replaces — the user explicitly asked to build the alternative
before removing the old thing, not the reverse. Updated four docs that
described `backend.js` as "legacy, still present" to describe it as
removed instead: `docs/demo_semantic_parity.md`,
`docs/wasm_demo_backend_spike.md`, `docs/application_readiness_review.md`,
and `BUGS.md`. One coverage gap is permanent and was explicitly accepted
rather than solved: `backend.js`'s route-drift-detection test (checking
real REST routes against a mock's route table) has no equivalent for the
WASM dispatch path, so that specific check is simply gone now.

---

`task/comprehensive-cleanup-boyscout` is a **second, explicitly more
exhaustive** clean-up pass, requested right after the first one (PR 122,
described below) merged: "act on all issues... any bugs or UI issues or API
issue or performance, deadlock etc... don't ignore any error or warning...
make sure the tests are correctly wired." The user also asked to assume
boy-scout rule on every future task by default from now on, not just when
explicitly requested — recorded as a standing preference.

- **Resolved PR 122's flag #1 — `internal/org/service.go`'s authz
  duplication.** PR 122 found that `requirePermissionForActor`,
  `canViewTeam`, and `canManageTeam` each hand-roll the same "an org-wide
  credential gets unconditional access to its own organization" check that
  `internal/authz.RequireOrganizationAccess` already centralizes for
  task/series/submission, but couldn't fix it because `internal/authz`
  already imports `internal/org` (for `org.Permission`), so `org` importing
  `authz` back would cycle. The fix: a new leaf package,
  `internal/orgactor`, containing exactly the actor-kind dispatch both
  sides need (`Check(actor, organizationID) Result` — `Match`/`Mismatch`/
  `NotApplicable`), with no dependency on either `authz` or `org`. Both
  packages now call the same function instead of each keeping their own
  copy. All pre-existing `org`/`authz` tests pass unchanged, confirming the
  refactor preserved behavior exactly; added a small unit test suite for
  the new package itself.
- **A real, narrow TOCTOU race in the Postgres rate limiter**
  (`internal/db/rate_limit_store.go`'s `Allow`): `select ... for update` on
  a bucket row that doesn't exist yet has nothing to lock, so concurrent
  *first* touches of a brand-new rate-limit key could each independently
  read "no bucket yet," each compute a fresh `tokens = capacity`, and each
  grant — over-admitting past capacity on that first burst. Fixed by
  inserting the row (`on conflict (key) do nothing`) *before* the
  lock-and-read, so Postgres's own documented behavior for concurrent
  conflicting inserts (a second inserter blocks until the first commits)
  closes the gap for every caller, including the first. Attempted to add a
  concurrent regression test (fire 20, then 200, concurrent `Allow()` calls
  against a fresh key and assert exactly one is admitted) — it passed
  reliably with *and without* the fix, meaning the actual race window in
  this local, low-latency Postgres setup is too narrow to trigger through
  plain goroutine concurrency (likely because pgxpool's connection pool and
  fast local round-trips mean genuinely-overlapping query execution at the
  Postgres engine level is rare without artificially widening the window).
  Removed that test rather than keep one that gives false confidence — the
  fix is still correct and cheap (verified by reasoning about Postgres's
  documented `INSERT ... ON CONFLICT` semantics), just not provably
  regression-tested. Recorded this honestly rather than overstate
  verification that wasn't actually achieved.
- **MCP session-store methods panicking on any transient database error**
  (`internal/http/mcp_sessions.go`) — crashing the entire process, not just
  failing the one request in flight. Every method (`create`,
  `existsForSubject`, `terminate`, `appendEvent`, `replayAndSubscribe`,
  `activeSessionCount`) `panic()`'d on a Postgres error from its
  persistence layer. Especially serious: `pollPersistedEvents` runs one
  goroutine per live SSE subscriber, polling Postgres every 500ms — with
  the user's stated target of ~100 concurrent MCP streaming sessions (while
  the server keeps serving regular API traffic), that's up to 200
  Postgres polling queries/second from background goroutines alone, and a
  single transient hiccup among them would have taken down every session
  and all other traffic at once. Fixed every call site to `slog.Error` and
  fail closed — returning the same "not found"/"refused"/zero value every
  caller already treats as a normal negative outcome — instead of
  panicking. Added `TestMCPSessionStoreDegradesGracefullyOnPersistenceFailure`
  with a fake `MCPSessionPersistence` that always errors, proving every
  method now degrades instead of panicking (this is a real regression test,
  unlike the rate-limiter one above — it reliably fails without the fix and
  passes with it, since the error injection itself is deterministic rather
  than depending on a timing race).
- **Two genuinely dead code paths**, found by a dedicated sweep since
  neither `go tool deadcode` nor the Elm compiler catches this class of
  issue (unused functions with a same-shape live sibling, or simple unused
  wrappers): `Api.elm`'s `fetchUserDirectoryQuery` (a wrapper with zero call
  sites — the real caller, `SearchUserDirectoryClicked` in `Main.elm`,
  calls `fetchUserDirectoryPage` directly instead), and
  `internal/wasmdemo/request_handler.go`'s `ModerationTriageHandler`/
  `NewModerationTriageHandler`/its `.Handle` method/`moderationTriagePathReportID`
  — reachable only from their own three dedicated tests, with the live WASM
  dispatch path (`cmd/sharecrop-wasm/main_js_wasm.go` →
  `AdminHandler.handleAdminModerationReports`) routing through a completely
  separate, independently-written implementation instead. Removed the dead
  code and its three now-pointless tests; confirmed `SaveModerationTriage`/
  `LoadModerationTriage` (the storage layer both implementations share)
  still has direct test coverage in `browser_storage_test.go`, so no
  coverage was actually lost.
- **Confirmed clean, no changes needed**: `go test -race` across the full
  unit/integration/http_e2e suite reports zero data races (including a
  focused look at the most concurrency-sensitive code, the MCP SSE session
  store's subscriber fan-out and teardown paths). The test suite's wiring
  is genuinely correct: no `t.Skip`, no orphaned build-tagged files (both
  `-tags integration` and `-tags http_e2e` compile and run in CI exactly as
  `make db-checks` expects), both Deno (`deno test`) and Playwright tiers
  are live CI jobs (not just available scripts nobody runs), no
  over-mocking that would blunt failure-path coverage, and Playwright
  assertions are state-based (balance deltas, ledger content, schema
  validation results) rather than superficial visibility checks. The one
  acknowledged gap: Go code has no static analysis beyond `go vet` (no
  golangci-lint/staticcheck configured) — noted as a possible future
  addition, not fixed here, since introducing a new linter's full ruleset
  onto a mature codebase is its own scoped decision with unpredictable
  blast radius, not a drive-by fix.
- **Still open, explicitly not resolved**: `site/demo/backend.js` (the
  hand-maintained JS mock backend) remains unreachable from the live
  browser demo (`site/demo/index.html` only supports `backend=wasm`) but is
  still exercised directly by two Deno tests. Found and verified a working
  replacement for its scenario-parity coverage
  (`deno task check:scenario-parity:wasm`, which runs the exact same shared
  scenario suite against the real, deployed WASM backend — not currently
  wired into any CI workflow) — but found no equivalent replacement for its
  route-drift-detection test specifically. Asked the user to confirm before
  deleting; the safety classifier correctly blocked the deletion attempt as
  an irreversible action needing real consent, and a follow-up clarifying
  question went unanswered within the turn. Left untouched rather than
  guess. This will likely resolve naturally once the separately-planned
  WASM/production backend unification actually supersedes both backends.
- **Also captured as separate, deferred future efforts** during this
  session (not part of this PR or any current work — see memory for full
  detail): (1) replacing `cmd/sharecrop` entirely with a single WASI-hosted
  WASM binary compiled from the real domain services, so the exact same
  artifact runs both the browser demo and production, once feasibility
  (Postgres-from-WASI, crypto/rand, etc.) is confirmed; (2) moving MCP/SSE
  streaming to HTTP/2-by-default (HTTP/3-ready) to genuinely support ~100
  concurrent sessions while continuing to serve regular API traffic — with
  HTTP/1.1 kept as an explicit, supported option for non-streaming UI/API
  traffic, not a fallback. A known mechanism relevant to that second effort
  was found and recorded rather than fixed now (per explicit scope
  agreement): `mcpHTTPSessionStore` guards its entire session map with one
  global mutex held across synchronous Postgres round-trips, serializing
  all session-store DB calls across all sessions — a real throughput
  bottleneck at real concurrency, separate from the panic-on-error bug
  above (which was fixed now since it's a plain correctness bug, not new
  infrastructure).
- **Verification**: `go build`/`vet`/`test` (including `-race`) across the
  whole module, `-tags integration`/`-tags http_e2e`, `elm make`, the full
  `make check-*`/`lint`/`vet`/`test-deno` suite, and the full Playwright
  suite (51/51, after confirming one apparent failure was a resource-
  contention flake under full parallel load by rerunning twice more
  cleanly) — all pass.

---

`task/rbac-cleanup-boyscout-fixes` is a requested clean-up pass over the
just-completed 5-phase RBAC + API-token effort (PRs 115-121, all merged
first), plus anything else opportunistic found along the way. Used 4
parallel review agents (read-only investigation, no edits) covering: the
credential/authz backend (`internal/agent`, `internal/orgcred`,
`internal/authz`), the MCP layer (dispatch, admin gating, adapters), the
Elm frontend (the Phase 5 additions specifically), and a broad repo-wide
sweep (demo backend parity, dead code, stale docs). Then triaged every
finding and fixed what was real.

- **Stale one-time task-token leak across tasks** (`web/elm/src/Main.elm`):
  `TaskDetailPage taskId`'s navigation reset cleared `reservations`,
  `reservationMessage`, `taskAgentToken`, etc. but missed the newer
  `reservationSecret` field. Since `issuedCredentialSecret` deliberately
  *keeps* the previous secret when a reservation event carries no fresh one
  (so cancel/decline events after a reveal don't blank it), navigating from
  a task whose reservation you'd just revealed a token for to a *different*
  task still showed that stale secret under the new task's "Reservation"
  card, labeled "Agent token for this task" — misleadingly implying it
  belonged to the task currently being viewed. Fixed by adding
  `reservationSecret = Nothing` to the same reset. Added a Playwright
  assertion (extending the existing reservation test) that visiting an
  unrelated task after revealing one shows no `reservation-agent-secret`
  element at all.
- **Credential-mint forms never reset after a successful mint**
  (`web/elm/src/Main.elm`'s `AgentCreated (Ok created)` /
  `OrgCredentialCreated (Ok created)`): only `newCredential`/
  `newOrgCredential` and the message field were set; `agentLabel`/
  `agentScopes`/`agentExpiresHours` (and the org-credential equivalents)
  were left as-is, unlike the rest of the app's established convention
  (`createOrgTeamName`/`provisionMemberEmail` both clear on success). A
  second mint would silently resubmit the same label, the same checked
  scopes, and the same now-stale "expires in hours" value. Fixed by
  clearing all three draft fields alongside the existing message-clear.
  Added a Playwright test asserting the form fields are empty right after a
  successful mint.
- **Invalid "expires in hours" silently minted a *never-expiring*
  credential** (`web/elm/src/Sharecrop/Api.elm`): `expiresAtFromHours`
  correctly returns `""` (never expires) for a blank field, but also for
  `"0"`, negative numbers, and non-numeric garbage — indistinguishable from
  the intentional blank case, with zero user feedback. For a
  least-privilege credential system, silently falling back to *no
  expiration* on a typo is the worst-case outcome. Added a separate
  `expiresHoursIsValid` check (blank, or a positive whole number) used by
  `createAgentCommand`/`createOrgCredentialCommand` before submission,
  surfacing "Expires in (hours) must be a positive whole number, or blank
  for never." otherwise. (An initial fix also added `Html.Attributes.min
  "1"` to the number inputs for a browser-level hint — reverted after
  confirming no other numeric field in this app does that, and because it
  actually broke testability: a browser's native constraint validation
  blocks the `submit` event entirely before Elm's own `onSubmit` handler
  — and therefore Elm's own validation message — ever runs.) Added a
  Playwright test confirming a negative value is rejected with the message
  and mints nothing.
- **MCP's task-scoped credential check used a raw string compare instead
  of parsing both sides** (`internal/mcp/server.go`): `toolArgumentTaskID`
  extracted the caller-supplied JSON string and compared it directly
  against `credential.TaskID.String()` (which is always canonical
  lowercase). A task ID sent in different casing — still the same,
  correct task — would be spuriously rejected with "agent credential is
  not valid for this task". Fails closed (not a security bypass) but a
  real correctness/availability bug for any caller that doesn't happen to
  echo the exact canonical form. Fixed by parsing the argument through
  `core.ParseTaskID` first and comparing the parsed value's `.String()`,
  normalizing case on both sides; an unparseable argument still fails
  closed. Added `TestToolsCallAcceptsTaskScopedCredentialWithDifferentlyCasedTaskID`.
- **`internal/mcp/tool_calls_admin.go`'s `callResolveAdminPrivacyRequest`**
  passed `args.PrivacyRequestID` straight through untrimmed, while the
  sibling moderation-report handler in the same file trims its equivalent
  ID-like field (`SubjectID`). A whitespace-padded ID would silently
  "not found" instead of matching. Added the missing `strings.TrimSpace`.
- **`docs/api_reference.md` was missing the entire agent-credential and
  org-credential REST surface** (never updated across Phases 1-2) and
  still documented `POST /api/tasks/{task_id}/capability-tokens`, deleted
  in Phase 1. Removed the stale line and added a new "Agent Credentials"
  section documenting all 6 routes plus the auto-issuance-on-reservation
  behavior.
- **`docs/mcp_reference.md` had two stale/misleading scope descriptions**:
  `submissions_review`'s bullet claimed it covered "list ... reservations",
  but `list_task_reservations` is actually gated by `submissions_read` (like
  `list_task_submissions`); and `privacy_read`/`privacy_manage`'s bullet
  said "(admin-gated)" without noting `privacy_read` is *also* the scope
  for the two non-admin self-service tools (`create_privacy_request`,
  `list_privacy_requests`) — only the runtime admin re-check, not the
  scope, distinguishes the admin-only listing tool. Corrected both.
- **`site/demo/backend.js`'s reservation objects (both the live handler and
  two hardcoded seed reservations) were missing `issued_worker_credential`**,
  a field `Generated/Task.elm`'s decoder requires (not `Decode.maybe`).
  Currently unreachable from the live browser demo (which defaults to WASM;
  the JS-backend `index.html` branch was removed in a prior PR), but the
  Deno tests (`demo_backend_test.ts`, `scenario_parity_test.ts`) still load
  `backend.js` directly, and the fix future-proofs the mock in case the JS
  backend path is ever restored. Added `issued_worker_credential: ""`
  (never issues one — kept intentionally minimal, not a full credential
  simulation) to all three reservation object literals.
- **Flagged, not fixed** (judgment calls, not opportunistic fixes):
  `internal/org/service.go` (`requirePermissionForActor`, `canViewTeam`,
  `canManageTeam`) hand-rolls the same "an `OrgSubject` gets unconditional
  access to its own org" rule `internal/authz.RequireOrganizationAccess`
  centralizes for task/series/submission — but `internal/authz` imports
  `internal/org` (for `org.Permission`), so `org` importing `authz` back
  would be a cycle. `org`'s own membership/team endpoints are structurally
  stuck outside the centralized layer; worth a real design discussion
  (e.g. extracting a lower-level shared package), not a cleanup-pass fix.
  Separately, `site/demo/backend.js` is now orphaned from the live demo
  (confirmed via `site/demo/index.html`'s current `main`: the `?backend=js`
  branch was removed) but still CI-tested directly — worth an explicit
  decision on whether to restore browser access or deprecate the JS mock
  and its Deno tests now that WASM is authoritative.
- **Verification**: `elm make` clean; full Go test/check suite green
  (`go build`/`vet`/`test`, `-tags integration`, `-tags http_e2e`, and the
  full `make check-*`/`lint`/`vet`/`test-deno` suite); full Playwright
  suite green (53/53, including 3 new specs and 1 extended existing spec);
  a new Go unit test for the task-ID case-insensitivity fix.

---

`task/mcp-rbac-elm-ui-scopes-expiration-org-tokens` is **Phase 5, the final
phase**, of the same 5-phase RBAC + API-token effort described below
(Phases 1-3 shipped as PRs 115-117, Phases 4a-4c as PRs 118-120, all merged
first). This phase builds the Elm UI for everything the backend already
supported but the frontend couldn't reach yet.

- **Contracts regeneration was step one**: `internal/contracts/
  definitions.go`'s `agentModule()` still only declared the original 5
  scopes, even though `internal/agent/values.go` had 19 as of Phase 2. Widened
  the `AgentScope` enum to all 19 and regenerated `Generated/Agent.elm`
  (`go run ./cmd/sharecrop generate elm-contracts`). Also updated
  `Sharecrop.Labels`' `allScopes`/`scopeTag`/`scopeLabel` to match — Elm's
  exhaustive `case` checking meant the compiler itself caught every place
  that needed updating.
- **Org-credential wire types went into the existing `Agent` module, not a
  new one — a real structural finding, not a style choice**: generated Elm
  modules only ever `import Json.Decode`/`Json.Encode`, never each other
  (confirmed by reading `internal/contracts/elm.go`'s file-header
  generation). Since `internal/orgcred.Credential` reuses `agent.Scope`/
  `agent.State` directly on the Go side (`internal/orgcred/models.go`),
  putting `OrgCredentialResponse`/`OrgCredentialsResponse`/
  `OrgCredentialCreatedResponse` inside `agentModule()` let them reuse the
  already-generated `AgentScope`/`AgentCredentialState` types instead of
  generating a duplicate enum that could drift from the real one.
- **Expiration input required more plumbing than expected**: the plan's
  original assumption was a plain "expires in N hours" number input mirroring
  the existing "reservation expiry hours" field, sent to the server as-is.
  Reading `internal/http/agent_mcp.go`'s `parseOptionalExpiresAt` showed the
  REST API actually expects an *absolute* RFC3339 timestamp, not a relative
  duration — unlike the reservation-hours field, where the server does that
  math. Elm has no access to "now" without an effect, so `Api.elm` gained a
  `Time.now`-driven flow: submitting either credential-mint form now defers
  the actual POST to a `AgentExpiresAtResolved`/`OrgCredentialExpiresAtResolved`
  message carrying `Time.Posix`, which combines with the hours draft field
  to compute the absolute timestamp. No existing Elm code touched `Time` at
  all, so this needed a small hand-rolled RFC3339 formatter (`Time` gives
  year/month/day/hour/minute/second accessors but no string formatting) and
  promoting `elm/time` from an indirect to a direct `elm.json` dependency
  (Elm refused to compile an `import Time` otherwise, even though the
  package was already present transitively).
- **The task-token reveal was mostly-wired already, just never read**:
  `Task.TaskReservationResponse.issuedWorkerCredential` has existed in
  generated code since Phase 1 (the backend already returns the one-time
  plaintext secret when a reservation becomes active), but `Main.elm`'s
  `ReservationReceived`/`ReservationChangeReceived` handlers only ever read
  it indirectly for a success-message label — the actual secret value was
  discarded. Added a `reservationSecret : Maybe String` model field, an
  `issuedCredentialSecret` helper (empty string → keep whatever was already
  revealed, non-empty → reveal it) so cancel/decline events after a reveal
  don't blank it out, and a `reservationSecretView` in `View.elm` reusing
  the same always-visible (not `Ui.disclosure`-collapsed) one-time-reveal
  shape the existing personal/task-token mint flows already use.
- **Org detail page**: added a 6th `Ui.disclosure "org-credentials-section"`
  sibling alongside the existing teams/members/collectibles sections
  (`View.elm`'s `activeOrganizationView`), with its own mint form (label,
  scope checkboxes reusing `allScopes`, expiration input), list, and revoke
  buttons — mirroring the personal-credential UI almost exactly, since org
  credentials are functionally the same shape with one more identifying
  field (`organizationID`).
- **Verification**: `elm make` compiled cleanly throughout (Elm's
  exhaustiveness checking caught every missing case as new Msg
  constructors/scope variants were added — no runtime surprises there), the
  full Go test/check suite passed unchanged, and the full Playwright suite
  (49/49) passed against a real Postgres-backed server, including 3
  new/extended specs: minting a credential with an expiration + new scope
  and seeing "expires" on its row, an org admin minting and revoking an
  org-wide credential end to end (asserting the `scrop_org_` secret
  prefix), and the existing reservation-becomes-active test now also
  asserting the one-time task-token reveal appears with a `scrop_agent_`
  secret.
- **This completes the entire 5-phase plan.** No further phases are planned
  for this effort.

---

`task/mcp-admin-moderation-privacy-audit` is Phase 4c of the same 5-phase
RBAC + API-token effort described below (Phases 1-3 shipped as PRs 115-117,
Phase 4a as PR 118, Phase 4b as PR 119, all merged first). This is the last
of the three MCP-parity sub-phases: it adds the 14 admin-gated tools and the
double-check mechanism MCP dispatch was missing.

- **The double-check mechanism, mirrored exactly from REST**: REST's
  `requireAdminSubject` (`internal/http/admin_config.go`) does two checks —
  a valid user session, *then* `server.platformAdmins.IsAdmin(ctx, actor.ID)`
  — because a user's admin status can change after a credential was minted;
  checking only the credential's scope would let a demoted admin's old
  credential keep working. Before this phase, MCP's `handleToolsCall` only
  ever checked the credential's scope (`credential.Scopes.Allows(...)`) —
  there was no second check at all. Added `requireAdminSubjectForTool`
  (`internal/mcp/server.go`), which calls the existing
  `requireUserSubjectForTool` then a new `Services.IsPlatformAdmin(ctx,
  userID) bool` method, gating all 10 admin tools. Verified this is a real
  gap that would have mattered: minted an agent credential with the
  `platform_admin` scope for an ordinary (non-admin) user and confirmed
  `list_platform_admins` used to have no code path that would reject it —
  now it does, both via a new `http_e2e` regression test
  (`TestMCPAdminScopeAloneIsNotEnough`) and by hand against the real server.
- **A structural discovery that shaped the design**: platform-admin,
  moderation, and privacy have no standalone domain package like
  `org`/`assets`/`notification` do — they're in-memory services defined
  entirely inside `internal/http` (`admin_config.go`, `moderation.go`,
  `privacy.go`). This means `internal/mcp` cannot reference their types
  directly: `internal/http` already imports `internal/mcp` (for
  `mcp.Server`), so the reverse import would be a cycle. Solved the same way
  Phase 4a solved it for `agent.Credential`/`orgcred.Credential`
  (`mcp.CallerCredential`): added small MCP-local mirror types
  (`mcp.PlatformAdminRecord`, `mcp.ModerationReport`,
  `mcp.PrivacyRequestRecord`, plus their Result-type families following the
  same closed-interface pattern as everywhere else in this codebase) that
  `mcpServices` converts to/from. Audit, by contrast, *is* a proper domain
  package (`internal/audit`) with no such constraint, so its MCP tools
  (`list_organization_audit_events`, `list_admin_audit_events`) reuse
  `audit.ListFilters`/`audit.ListResult`/`audit.Event` directly — only one
  new `Services.ListAuditEvents` method needed, with each tool constructing
  its own filters at the call site.
- **No logic duplication despite the new mirror types**: `mcpServices`'
  conversion functions (in a new `internal/http/agent_mcp_admin.go`) reuse
  the *exact* unexported helpers the REST handlers already call —
  `validModerationSubjectKind`, `validModerationReason`,
  `encodeModerationMetadata`, `validPrivacyRequestKind`, the
  `PlatformAdminService`/`ModerationTriageService`/`PrivacyService`/
  `AuditService` interfaces themselves — rather than re-implementing
  moderation/privacy business rules a second time in a new place. One new
  helper, `moderationReportFromEventAndTriage`, was added because the
  existing `moderationReportFromAuditEvent`/`applyModerationTriage` produce
  an already-stringified HTTP response shape; the new one keeps
  `time.Time` fields as `time.Time` so `internal/mcp`'s own summary
  functions do the JSON formatting, consistent with every other domain
  type's pattern in this codebase (e.g. `notification.Notification`).
- **14 new tools**: `list_platform_admins`/`grant_platform_admin`/
  `revoke_platform_admin` (admin-gated), `create_moderation_report`
  (any user) plus `list_admin_moderation_reports`/`triage_moderation_report`
  (admin-gated), `create_privacy_request`/`list_privacy_requests` (any
  user, own requests only) plus `list_admin_privacy_requests`/
  `resolve_admin_privacy_request`/`run_privacy_retention` (admin-gated),
  `list_organization_audit_events` (any user with `PermissionManageMembers`
  on the org) plus `list_admin_audit_events` (admin-gated), and
  `award_collectible` (admin-gated, deferred from Phase 4b specifically
  because it needed this phase's double-check mechanism to exist first).
- **`mcpServices` gained 4 more service dependencies**:
  `platformAdmins`, `moderationTriage`, `privacyService`, `auditService` —
  again all already existed on `httpserver.Server`, just not threaded into
  `mcp.NewServer(mcpServices{...})`. A **real bug caught before it shipped**
  while wiring this up: the struct-literal construction in `newServer`
  compiles fine even when new named fields are omitted (they just default
  to their zero value — `nil` for interfaces), so the build stayed green
  after adding the new `Services` methods but *before* actually passing the
  real runtime services in — every new admin/moderation/privacy MCP tool
  would have nil-panicked in production. Caught by re-reading the
  construction site deliberately rather than trusting a passing build,
  and fixed by threading `runtime.PlatformAdmins`/`runtime.ModerationTriage`/
  `runtime.PrivacyService`/`runtime.AuditService` through explicitly. The
  exported `NewMCPServer` (used by the stdio bootstrap) and
  `cmd/sharecrop/main.go`'s `runMCPStdio` needed the same 4 new
  constructor arguments, using the same DB-backed constructors
  (`db.NewPlatformAdminStore`, `db.NewModerationTriageStore`,
  `db.NewPrivacyStore`, `audit.NewService(db.NewAuditStore(pool))`)
  `runServe` already used for the HTTP path.
- **Verification**: 6 new `http_e2e` tests covering the double-check
  regression, a real admin driving all 3 platform-admin tools, the full
  moderation lifecycle (report → list → triage), the full privacy lifecycle
  (request → list → admin-list → resolve → retention), both audit tools,
  and `award_collectible` (including confirming a non-admin is denied at
  the scope-check layer, one level earlier than the double-check). All
  pass alongside the full pre-existing suite unchanged. Hand-verified
  against the real server: minted an agent credential with the
  `platform_admin` scope for a genuinely non-admin user and confirmed
  `list_platform_admins` is rejected with "platform admin access is
  required" rather than succeeding.
- **Phase 4 is now complete** across all three sub-phases (4a/4b/4c).
  Remaining in the original 5-phase plan: Phase 5, the Elm UI.

---

`task/mcp-orgs-teams-collectibles-notifications-users` is Phase 4b of the
same 5-phase RBAC + API-token effort described below (Phases 1-3 shipped as
PRs 115-117, Phase 4a as PR 118, all merged first). This phase adds the new
MCP tool categories Phase 4's research identified as missing.

- **Scoped by re-checking REST, not by trusting the domain layer**: dispatched
  a fresh research pass (rather than reusing Phase 4's older notes verbatim)
  to get exact signatures/result-types for `org.Service`, `orgcred.Service`,
  `assets.Service`, `notification.Service`, and the 4 users/profile REST
  handlers — and, critically, which specific REST handler backs each
  operation and whether *that* handler (not just the domain method) accepts
  an organization-wide credential. This surfaced a real subtlety: `org.Service`'s
  `ProvisionMember`/`DeactivateMember`/`UpdateMemberRoles` already accept
  `auth.Subject` at the domain layer (from Phase 2's plumbing), but REST's
  `provisionOrganizationMember`/`deactivateOrganizationMember`/
  `updateOrganizationMemberRoles` handlers still call `requireUserSubject`,
  not `requireUserOrOrgSubject` — so the corresponding MCP tools stayed
  `auth.UserSubject`-only too, even though widening them would have
  compiled fine and "worked." Only `get_team` and `add_team_member`
  actually needed `auth.Subject` — REST's only two org-token-capable
  team handlers.
- **~30 new tools across 5 categories**: organizations/teams (13 —
  create/list organizations, provision/deactivate/update-roles members,
  create/list org and standalone teams, get-team/add-team-member/
  get-team-work), org-wide credential self-management (3 — mirrors the
  REST org-credential endpoints exactly, including replicating the inline
  `PermissionManageMembers` check REST does before minting, since
  `orgcred.Service.Create` itself has no permission gate of its own), 
  collectibles (8 — mint/catalog/transfer/list/fund-reward/refund-reward/
  list-by-organization/list-by-team), notifications (2), users/profile (4).
  Deliberately excluded `award_collectible` (REST's only
  `requireAdminSubject`-gated collectible endpoint) — that belongs with
  Phase 4c's admin-gated tools and the double-check mechanism they need,
  not bolted on here without that mechanism existing yet.
- **File organization**: rather than keep growing the two existing
  monolithic `tools.go`/`tool_calls.go` files (already large before this
  phase), added 5 new per-domain file pairs
  (`tools_orgs.go`/`tool_calls_orgs.go`, `tools_collectibles.go`/
  `tool_calls_collectibles.go`, `tools_notifications.go`/
  `tool_calls_notifications.go`, `tools_users.go`/`tool_calls_users.go`,
  plus org-credential tools folded into the orgs pair since they're
  organization-scoped too). `toolDefinitions()` now composes a base list
  with `append` from each domain's own `xxxToolDefinitions()` function
  rather than one giant literal.
- **Reused existing helpers rather than duplicating them**: the new
  `callGetUserWork`/`callGetUserProfile` reuse the existing `taskSummary`/
  `tasksPayload` types and `taskToSummary` helper already in
  `tool_calls.go`; `callGetUserSubmissions` reuses `submissionSummary`/
  `submissionsPayload`/`submissionToSummary`. This is why `make
  check-copy-paste` still reports 0 duplicate clones despite adding ~30
  tools — the boilerplate that *would* have been copy-pasted (payload
  marshaling, ID parsing) was extracted into shared helpers instead.
- **`mcpServices` (the HTTP-to-MCP adapter) gained 5 new service
  dependencies**: `organizationService`, `orgCredentialService`,
  `assetService`, `notificationService`, `authService` — all already
  existed on `httpserver.Server`, just not threaded into
  `mcp.NewServer(mcpServices{...})` before. Both call sites (`newServer`
  and the exported `NewMCPServer` used by the stdio bootstrap) needed
  updating, and the stdio bootstrap (`cmd/sharecrop/main.go`) needed a new
  `authService` construction it didn't have before (list_users needs it).
- **A deliberate, documented gap, not an oversight**: `getUserSubmissions`'s
  REST handler calls `recordSensitiveFieldAccess` for audit logging on each
  returned submission — an `*http.Request`/`http.ResponseWriter`-coupled
  helper with no non-HTTP variant. Consistent with Phase 4a's precedent
  (no MCP tool calls `recordAudit` either), the MCP `get_user_submissions`
  tool skips this audit step. Revisit if/when audit logging becomes a
  cross-transport concern rather than an HTTP-handler-local one.
- **Verification**: 9 new `http_e2e` tests (one per category, plus
  `TestMCPGetTeamAndAddTeamMemberAcceptOrgCredential` specifically
  regression-testing the org-token-parity boundary for the two tools that
  have it) all pass, alongside the full pre-existing suite unchanged. Also
  hand-verified against a real server: minted an agent credential scoped to
  the new read/manage scopes, drove `create_organization`, `list_organizations`,
  `mint_collectible`, and `list_users` over real MCP JSON-RPC calls, and
  confirmed the scope gate still correctly rejects `create_task` for a
  credential missing `tasks_write`.

---

`task/mcp-tool-parity-orgsubject` is Phase 4a of the same 5-phase RBAC +
API-token effort described below (Phases 1-3 shipped as PRs 115-117, merged
first). Phase 4 ("MCP tool parity") turned out much bigger than the
original plan estimated once actually researched, so it's split into three
sub-phases; this is the first.

- **Scoping first**: before writing any code, dispatched a research pass
  comparing the full REST route table (`internal/http/server.go`) against
  the MCP tool surface (`internal/mcp/tools.go`). Found ~60 REST operations
  across 14 categories with no MCP equivalent (organizations, teams,
  collectibles, notifications, moderation, privacy, platform-admin, audit,
  saved-queue-views, account self-service, plus 4 gaps in already-covered
  categories), and — more load-bearing for this phase — that MCP's entire
  actor-handling stack was hardcoded to `auth.UserSubject`: the `Services`
  interface, `Handle`/`HandleRaw`/`ServeStdio`, every `dispatchTool` case,
  all ~29 `call*` implementations, and the transport-auth resolver
  (`verifyAgent`) that only ever looks in the personal-agent-credential
  store, never an org-credential one. Given the scale, split Phase 4 into
  4a (this PR, MCP transport + `OrgSubject` wiring + cheap gap-fills), 4b
  (new tool categories), 4c (admin-gated tools + the missing double-check
  mechanism) rather than one enormous PR.
- **`mcp.CallerCredential`**: a new type (`Scopes agent.ScopeSet, TaskID
  *core.TaskID`) that decouples MCP's scope-gate and task-scoping checks
  from the concrete credential type. Before this phase, `handleToolsCall`
  took a hard `agent.Credential` — impossible to also carry an
  `orgcred.Credential` (a different struct, and org credentials are never
  task-scoped, so no `TaskID` field to speak of). Both credential kinds now
  convert to this one shape before entering MCP's dispatch layer.
- **`auth.Subject` through the whole stack**: `mcp.Services`, `Handle`,
  `HandleRaw`, `handleBatch`, `ServeStdio`, and `dispatchTool` all widened
  from `auth.UserSubject` to `auth.Subject`. A new `requireUserSubjectForTool`
  helper centralizes the "this tool needs a person, not an org token" check
  at each of the ~23 still-user-only dispatch cases, so those ~23 `call*`
  functions themselves stay untouched (still typed `auth.UserSubject`,
  zero risk to their existing logic) — only the dispatch wrapper changed.
- **Exactly 6 tools widened, matching REST — not exceeding it**: rather
  than widen every tool that theoretically *could* accept an org token (the
  underlying domain method being `auth.Subject`-typed doesn't automatically
  mean it should be exposed that way), checked each candidate tool's REST
  counterpart specifically: does that REST handler already use
  `requireUserOrOrgSubject`? Only 6 do (`list_tasks`, `open_task`,
  `unpublish_task`, `list_task_reservations`, `approve/decline/
  cancel_task_reservation`), so only those 6 MCP tools got org-token
  support this phase — e.g. `get_task` stays worker-credential/user-only
  because its REST handler (`requireWorkerSubject`) has no org-credential
  fallback either. This is a deliberate design principle for the whole
  effort: MCP capability tracks REST capability, it doesn't get ahead of it.
- **New MCP transport auth resolver**: `internal/http/agent_mcp.go` gained
  `verifyMCPCaller`, mirroring `requireUserOrOrgSubject`'s prefix-dispatch
  pattern (`orgcred.HasSecretPrefix`) but for MCP's credential-secret-only
  auth model (MCP has no user-session concept at all, unlike REST, so it
  never tries a user access token — just personal-agent-or-org secret).
  Wired into all three MCP HTTP handlers (`mcpEndpoint`, `mcpStream`,
  `mcpDeleteSession`). Rate-limit and session-ownership keys switched from
  `verified.Subject.ID.String()` (assumed a `core.UserID`) to a new
  `mcpSubjectIdentity` helper that prefixes `"user:"`/`"org:"` so a
  `core.UserID` and `core.OrganizationID` that happened to stringify
  identically could never collide.
- **Stdio bootstrap gets the same fallback**: `cmd/sharecrop/main.go`'s
  `runMCPStdio` (reads `SHARECROP_AGENT_TOKEN`) now tries org-secret parsing
  first, falling back to the personal-agent path — the CLI/stdio transport
  and the HTTP transport now authenticate identically.
- **4 cheap gap-fill tools**, reusing existing scopes and plumbing per the
  research's own recommendation: `cancel_task` and `refund_task` (REST
  already had `cancel`/`refund` handlers with no MCP equivalent —
  `cancel_task` got org-token parity since `changeTaskState` already
  supports it; `refund_task` stayed user-only, matching `refundTask`'s
  still-user-only REST handler), `update_series` and `reorder_series`
  (REST already had these too, both stay user-only since their REST
  handler shares the user-only `seriesActor` helper).
- **A narrow risk caught while widening**: the MCP `list_tasks` tool's
  `scope=user` case dereferences `subject.ID` directly, which only compiles
  for a concrete `auth.UserSubject`. Widening the tool's parameter to
  `auth.Subject` meant this needed an explicit guard rather than a bare
  type assertion — added a check that cleanly rejects `scope=user` for a
  non-`UserSubject` actor (an org-wide credential using `scope=public`
  still works, matching what an org-admin member could browse; `scope=
  organization` isn't exposed via MCP at all yet, deferred with the rest
  of the new-capability work).
- **Verified end-to-end against the real transport**, matching the
  established per-phase discipline: a new `http_e2e` test
  (`TestMCPOrgCredentialActsWithFullParityOnItsOwnOrgOnly`) mints an org
  credential, uses it over real MCP JSON-RPC calls to open and cancel its
  own organization's tasks (succeeds), confirms a user-only tool
  (`create_task`) fails cleanly with a tool-level error rather than a
  protocol-level crash, then confirms the same org credential is rejected
  outright — not silently scoped down — against a second organization's
  task. Plus two more new tests exercising the gap-fill tools
  (`TestMCPUpdateAndReorderSeries`, `TestMCPRefundTask`). All pre-existing
  MCP `http_e2e` tests pass unchanged, confirming this phase didn't disturb
  the existing worker/reviewer/requester/series tool loops.
- **Documentation**: updated `docs/mcp_reference.md` with the 4 new tools
  and a new section explaining which tools accept an organization-wide
  credential and why (mirrors REST parity, not a special MCP-only rule).
- **Deferred to Phase 4b/4c, explicitly**: the ~14 new tool categories
  (organizations, teams, org-credentials, collectibles, notifications,
  users, moderation, privacy, platform-admin, audit) and the admin
  double-check mechanism MCP dispatch doesn't have yet (REST's
  `requireAdminSubject` checks `PlatformAdminService.IsAdmin` on top of the
  scope check; MCP's `handleToolsCall` only ever checks scope today — a
  credential minted with a `platform_admin` scope by an admin who is later
  demoted would still pass MCP's gate, a gap that needs its own careful
  design rather than being bolted on here).

---

`task/authz-centralize-ownership-visibility` is Phase 3 of the same 5-phase
RBAC + API-token effort described below (Phase 1 shipped as PR 115, Phase 2
as PR 116, both merged first). This phase centralizes the authorization
pattern Phase 2 introduced, per the plan's original Phase 3 scope.

- **New `internal/authz` package**, one function:
  `RequireOrganizationAccess(ctx, actor auth.Subject, organizationID, checker, permission org.Permission, deniedCode core.ErrorCode, deniedReason string) Decision`.
  It grants access when actor is that exact organization's own credential
  (`auth.OrgSubject`, full parity with an org-admin member — unconditional
  id match, no per-member permission lookup, since the token *is* the org),
  or when actor is a user who holds `permission` via `checker`. It does
  **not** replace `org.CheckPermission` — that stays the single source of
  truth for per-member role/permission logic; `authz` only decides which
  actor kinds may reach it and how an org-wide credential bypasses it.
- **Why not a richer abstraction**: `internal/authz` deliberately does not
  import `internal/task` at all — `task`/`submission` both need to call
  into `authz`, so `authz` importing `task.Owner`/`task.Visibility` back
  would create an import cycle. This is why `authz` only knows about
  organization ids and `org.Permission`, not the owner/visibility sum types
  themselves — those stay resource-specific in each package. Re-reading the
  actual code (rather than the plan's pre-implementation guess) also showed
  the "visibility switch" shape genuinely differs per resource: Task has a
  separate `Visibility` field distinct from `Owner`; `Series` only has
  `Owner`, plus a draft-privacy guard task doesn't have; submission's
  organization-view-permission checks an OR of two different permissions,
  not the single-permission shape everything else uses. Forcing all of
  these into one shared function would have obscured real per-resource
  authorization rules behind indirection for a modest line-count win — so
  only the piece that's genuinely identical everywhere (the
  org-token-or-per-member-permission decision) got centralized.
- **Four near-duplicate functions collapsed into thin callers, three
  deleted outright**: `task.requireViewPermission` +
  `task.requireOrgActorViewPermission` (two functions, one per actor kind)
  merged into one `requireViewPermission` that branches once at the top
  and shares the visibility switch; `task.requireOrganizationViewPermission`
  and `task.requireOrganizationPermission` are gone, replaced by direct
  `authz.RequireOrganizationAccess` calls at their former call sites;
  `series.requireSeriesViewPermission`'s org/user branches unified the same
  way; `submission.requireReviewPermission` (userID-based) is gone, folded
  into `requireReviewPermissionForActor` (actor-based) as its sole caller.
- **A real behavioral risk caught before it shipped, not after**: task's
  and series' "denied" paths look identical at a glance but use *different*
  `core.DomainError` codes — task uses `ErrorCodePermissionDenied` (maps to
  HTTP 403), series uses `ErrorCodeInvalidState` (maps to HTTP 409). A
  naive shared helper hardcoding one code would have silently changed the
  other resource's HTTP status on denial. Fixed by having
  `RequireOrganizationAccess` take the error code as an explicit parameter
  rather than hardcoding it, with a dedicated unit test
  (`TestRequireOrganizationAccessUsesCallerSuppliedErrorCode`) asserting the
  code passes through unchanged.
- **Also caught and fixed**: the original `task.requireViewPermission`
  rejected any actor that was neither `auth.UserSubject` nor
  `auth.OrgSubject` (e.g. a hypothetical `auth.GuestSubject`) *before* ever
  checking task visibility — even for a `PublicVisibility` task. An early
  version of this refactor's unified function accidentally dropped that
  early rejection, which would have let such an actor view public tasks it
  couldn't before. Caught by tracing through every actor-kind case against
  the original code side by side (not by a failing test — no existing test
  exercised this actor kind) and restored the exact original short-circuit.
- **Verification**: full `go test ./...` (including the new
  `internal/authz` unit tests), `go test -tags integration
  ./tests/integration`, and `go test -tags http_e2e ./tests/http_e2e` —
  which includes Phase 2's `TestOrgCredentialActsWithFullParityOnItsOwnOrgOnly`
  regression test — all pass unchanged, confirming this refactor is
  behavior-preserving. `make check-copy-paste` (0% duplication threshold)
  reported 0 clones even before this phase (the duplicated blocks were each
  under jscpd's 12-line/150-token match window, so the tool never flagged
  them) — this phase's motivation was centralizing security-critical logic
  for auditability, not clearing a failing check.
- **Deferred, unchanged from the plan**: MCP tool parity and MCP-side
  `OrgSubject` support (Phase 4); Elm UI (Phase 5).

---

`task/org-credentials-orgsubject-authz` is Phase 2 of the same 5-phase RBAC +
API-token effort described below (Phase 1 shipped as PR 115 and merged
first). This phase adds organization-wide credentials and widens
authorization to recognize them with full org-admin parity.

- **`auth.OrgSubject{ID core.OrganizationID}`**: a third variant of the
  closed `auth.Subject` interface (alongside `UserSubject`/`GuestSubject`),
  representing an organization acting as itself via an org-wide credential
  rather than through any one member.
- **New `internal/orgcred` package**, mirroring `internal/agent`'s
  create/verify/list/revoke shape exactly (same result-type idiom, same
  `Store` interface pattern) rather than inventing a new one — reuses
  `agent.Label`/`agent.ScopeSet`/`agent.State` directly since those carry no
  user-specific semantics, and even reuses `agent.CreateStoreResult` itself
  (a type alias) since that particular result carries no credential-kind
  payload either. Secrets use a distinct `scrop_org_...` prefix
  (`orgcred.HasSecretPrefix`) so verification can dispatch on the prefix
  alone, matching `internal/agent`'s existing `scrop_agent_...` convention.
  New migration `migrations/000033_org_credentials.sql` adds
  `org_credentials`/`org_credential_scopes` tables mirroring
  `agent_credentials`'s shape (organization id instead of user id, same
  scope-check constraint).
- **New REST endpoints**: `POST/GET /api/organizations/{id}/credentials`,
  `POST /api/organizations/{id}/credentials/{credential_id}/revoke`. Minting
  itself is a *user* action, gated by the existing
  `org.PermissionManageMembers` check (minting an org-wide admin-equivalent
  credential is at least as sensitive as membership management) — the
  resulting token then acts on its own, with `auth.OrgSubject` parity, once
  issued.
- **Widened authorization, deliberately scoped**: went through every
  `task`/`org`/`submission` service method one at a time and asked "does
  this need an individual human identity, or is it a view/list/manage
  action an org token can stand in for?" View/list/manage/review methods
  (`task.Service.Get/Open/Cancel/Unpublish/List/GetSeries/*Reservation*`,
  `org.Service.ProvisionMember/DeactivateMember/UpdateMemberRoles/GetTeam/
  AddTeamMember`, `submission.Service.Get/ListForTask/
  ListSubmissionComments`) widened from `actor auth.UserSubject` to
  `actor auth.Subject`, with a new `case auth.OrgSubject:` branch per helper
  granting access unconditionally when the resource's owning organization
  matches the token's own id — full parity, no per-member permission-table
  lookup, since the token *is* the org. Creation and individual-authored-work
  methods (task/series/team creation, comment authoring, submission
  submitting) deliberately stayed `auth.UserSubject`-only: an organization
  has no individual identity to attribute `CreatedBy`/`AuthorID`/
  `SubmitterID` to. Caught and self-corrected one over-permissive draft
  before it shipped: series-view permission initially let an org token see
  *any* of its org's draft series, but re-reading the human-actor code path
  showed a draft series is creator-private even to other org-admin members
  — added the same guard to the `OrgSubject` branch so org-token access
  matches a human org-admin member exactly, not more.
- **REST wiring is intentionally partial, not exhaustive**: the handlers for
  task get/list/open/cancel/unpublish/reservations and team get/add-member
  now accept either a user session or an org credential
  (`requireUserOrOrgSubject`, a new resolver alongside the existing
  `requireUserSubject`/`requireWorkerSubject`). Series-detail and
  organization-member-management handlers were deliberately left
  user-only for this phase: series routes share a `seriesActor` helper used
  by every series handler including creation, and member-management
  handlers thread the actor's user id into audit-log attribution
  (`recordAudit` is hard-typed to `core.UserID`) — extending either cleanly
  would mean touching shared helpers or the audit model itself, judged out
  of scope here. The service-layer widening for these already-completed
  paths stays forward-compatible and dormant until a later phase wires it up.
- **A real gap found and fixed, not introduced this phase**: Phase 1's
  migration widened the DB scope-check constraint to allow 19 scope
  strings, but only the original 5 had corresponding `agent.Scope` Go
  values — `agent.ParseScope("org_manage")` would have rejected a
  perfectly DB-legal scope. Added the other 14 (`org_read`/`org_manage`/
  `collectibles_read`/`collectibles_manage`/`notifications_read`/
  `notifications_manage`/`users_read`/`ledger_read`/`moderation_read`/
  `moderation_manage`/`privacy_read`/`privacy_manage`/`platform_admin`/
  `credentials_manage`), completing the taxonomy Phase 1 only half-finished
  at the DB layer.
- **Verified end-to-end by hand against the real Postgres-backed server**,
  matching Phase 1's precedent of not trusting green tests alone for
  security-sensitive changes: registered two organizations, minted an org
  credential for the first, used it to open and list its own organization's
  tasks (200), then used the *same* credential against the second
  organization's tasks and reservation-open action — rejected outright
  (403) both times, not silently scoped down to nothing. Also confirmed
  a `scope=organization` task-list returning zero results was a
  *pre-existing* semantic (that scope filters by explicit
  organization-visibility sharing, not by owning organization) by
  reproducing the identical zero-count result with a plain user token,
  not something this phase's changes caused — deliberately left alone as
  out of scope. This flow is now also an automated `http_e2e` regression
  test (`TestOrgCredentialActsWithFullParityOnItsOwnOrgOnly`).
- **New tests**: `internal/orgcred` unit tests (create/verify/expiry/revoke,
  distinct secret prefix from `agent`'s), a store integration test
  (`tests/integration/org_credential_store_test.go`), and the `http_e2e`
  regression test above.
- **Boy-scout / hygiene**: extracted a shared `parseCredentialFields`
  helper (label/scopes/expiration parsing) out of the near-identical
  `createAgentCredential`/`createOrgCredential` handler bodies, and aliased
  `orgcred.CreateStoreResult` to `agent.CreateStoreResult` — both found by
  `make check-copy-paste` (0% duplication threshold) flagging genuine
  copy-paste, not worked around by weakening the check.
- **Deferred to later phases, explicitly**: MCP-side `OrgSubject` support
  (Phase 4, per the original plan); centralizing the
  `switch actor.(type) { UserSubject; OrgSubject }` shape now duplicated
  across several authorization helpers into `internal/authz` (Phase 3, per
  the original plan — extracting it now would be designing the abstraction
  before enough duplication existed to shape it correctly); Elm UI for
  minting/listing/revoking org tokens (Phase 5).

---

`task/agent-credential-scopes-expiry-task-tokens` is Phase 1 of a larger,
explicitly-planned RBAC + API-token effort (users can create scoped/expiring
API tokens; organizations get org-wide tokens; MCP gets full parity with the
API; the system gets a real RBAC layer) — see the design plan discussed with
the user for the full 5-phase breakdown. This phase lays the foundation:
credential scopes/expiration, a real per-task credential, and auto-issuance
on reservation.

- **`agent.Credential` gained `ExpiresAt`/`TaskID`**, both nil-able. `Verify`
  now rejects an expired credential the same way it already rejected a
  revoked one. The scope taxonomy widened from 5 to 19 values (adding
  organizations/teams/collectibles/notifications/users/ledger/moderation/
  privacy/platform-admin/credential-self-management scopes) so later phases
  have room to grow without another migration touching the same constraint.
- **Auto-issued task-scoped worker credential**: when a worker's reservation
  on a task becomes active — confirmed via direct source reading that this
  happens at *two* distinct call sites, `Service.reserve` (immediately
  active under `reservation_required` policy) and `Service.ApproveReservation`
  (`approval_required` policy's explicit approval step), not just the one
  the initial research suggested — the task now auto-mints a credential
  scoped to `{tasks_read, submissions_write, submissions_read}`, restricted
  to that one task, 30-day expiry, surfaced as a one-time plaintext secret
  in the reservation/approval response (`issued_worker_credential`) for both
  REST and MCP. This is the concrete mechanism behind "hand a task-specific
  token to an agent, compartmentalized to just that task."
- **A real security gap found by manual end-to-end testing, not by code
  review or unit tests**: the task-scoping was modeled in
  `Credential.MatchesTask` and wired into `Verify`'s expiration check, but I
  never actually wired the *match* check into the REST (`requireWorkerSubject`)
  or MCP (`handleToolsCall`) request paths — so a freshly-minted task-scoped
  credential worked against *any* task, not just its own. Caught this by
  hand-testing the full flow with curl against the real Postgres-backed
  server (create task → reserve → approve → use the issued secret against
  its own task, then a different one) exactly as the plan's verification
  section called for, rather than trusting that "the tests pass" meant the
  feature was actually secure. Fixed on both REST and MCP: `core.TaskID` now
  threads through `requireWorkerSubject`'s three call sites (task
  reordered-before-auth so the ID is known at check time), and MCP's
  `handleToolsCall` now checks a tool call's `task_id` argument against the
  credential — the whole `scopes`-only plumbing (`ServeStdio` →
  `HandleRaw` → `Handle` → `handleToolsCall`) widened to carry the full
  `agent.Credential` instead of just its `ScopeSet`, since the check needs
  the credential's `TaskID` too. Documented one narrower, accepted residual
  gap in the code: submission-comment tools take a `submission_id`, not a
  `task_id`, so a task-scoped credential could technically reach a comment
  thread on a *different* task's submission — but only one the same
  underlying user is already legitimately the submitter or task owner for,
  since the service-layer authorization still applies regardless. Added a
  dedicated regression test for the fixed case
  (`TestToolsCallRejectsTaskScopedCredentialForADifferentTask`).
- **Boy-scout: deleted `task.CapabilityToken` entirely.** Confirmed via a
  full-repo grep that this task-bound token type — minted via
  `POST /api/tasks/{id}/capability-tokens` — had no verification/lookup
  path anywhere in the codebase; it was mint-only dead code, almost
  certainly an abandoned earlier stub for exactly the auto-issuance feature
  built in this phase. Removed the Go types, the HTTP route and handler, the
  DB table (migration), the OpenAPI/Elm-contract entries, and the WASM
  demo's separate stale copy of the same route (a real, distinct parity bug
  caught by `tests/deno/demo_backend_test.ts`'s route-surface check, which
  compares the WASM/JS demo backend's routes against the real Go router).
- **Boy-scout: the WASM demo allowed reserving your own task.** The real
  backend already rejects this (`"task requester cannot reserve their own
  task"`); the WASM demo's reservation handler had no equivalent check.
  Fixed for parity — a pre-existing gap unrelated to this phase's core work,
  found while reading the reservation code path closely enough to place the
  new auto-issuance hooks correctly.
- Verified: all 47 real-backend Playwright specs, all 13 WASM-demo specs,
  Go unit/integration/http_e2e suites, the full non-browser check suite,
  and (given the security-sensitive nature of this phase) a hand-run
  end-to-end curl verification against the real Postgres-backed server
  confirming the issued credential works against its own task and is
  correctly rejected (401) against a different one.

`task/task-detail-reorder-profile-links-uiux` refined the task detail page
and profile pages for usability, at the user's explicit direction (make the
report panel collapsible, put reservation status at the top, link people to
their profiles), plus a boy-scout pass that found a real, previously-
unreachable capability gap.

- **"Report task" is now a collapsed-by-default `Ui.disclosure`** (was
  always-expanded at the bottom of every task detail page, competing for
  attention with content people actually need most of the time).
- **Reservation status moved to the top of the task detail page**, right
  after the task's own details and before any role-specific controls — "can
  I reserve this / who has it" is usually the first thing a visitor wants to
  know. This surfaced a real, significant pre-existing gap: `reservationCard`
  used to render only in the non-owner/non-reviewer branch, so **a task's
  owner had no way to see or act on a pending reservation request through
  the browser at all** — the Approve/Decline buttons existed in the code but
  were never reachable by the one role that's actually allowed to click them.
  Rendering the card unconditionally fixes this outright: the server's
  per-viewer `viewerAction` already resolves correctly for owners (never
  `Reserve`/`RequestApproval`), so no new gating was needed for the
  read-only display — except one thing the fix itself exposed: the server
  doesn't prevent an owner from reserving their own task, which isn't a real
  workflow, so the "go claim this yourself" action specifically (not the
  display) is now owner-gated client-side.
  Added a new test (`screens.spec.ts`, "an owner approves a worker's
  reservation request from the task detail page") since this exact flow had
  zero prior coverage — it simply couldn't be exercised through the UI
  before.
- **Reservation and submission actions are now scoped to who's actually
  entitled to take them**: previously *any* worker viewing a task saw
  Approve/Decline/Cancel buttons on *every* reservation, including other
  people's — buttons that would just fail server-side if clicked by someone
  without authority. Now Approve/Decline only render for the task's owner,
  and Cancel only for the owner or the reservation's actual holder.
- **People now link to their profiles wherever their user ID appears** on
  the task detail page and elsewhere: the reservation holder, a submission's
  submitter, a task's creator ("Posted by …", previously not shown
  anywhere), a notification's actor, and a platform admin's user ID. This
  extends a linking pattern that already existed for comments and org
  members to the handful of places that were missing it.
- **Profile page task-history discoverability**: research into
  `/api/users/{id}/work` and `/api/users/{id}/submissions` semantics showed
  the submissions endpoint rejects any viewer who isn't the submitter
  themselves — so the "Submissions" link on someone *else's* profile was
  reachable but would always 403. Now shown only on your own profile.
  Relabeled "Public work" to "Currently working on" (it's the tasks a user
  is *currently* actively reserved on, not a full history — there's no
  backend support for viewing another user's past/completed work, so the
  label now says what it actually shows) and enriched its rows with
  state/reward/reserved-by info instead of just a title.
- Verified: all 47 real-backend Playwright specs (46 existing + 1 new,
  `make e2e-ui`), all 13 WASM-demo Playwright specs, Go
  integration/http_e2e suites, `go test ./...`, and the full non-browser
  check suite all pass. Screenshot-verified the reordered task detail page
  (reservation section showing a clickable holder, a working Cancel button
  for the owner, and the collapsed Report task panel at the bottom) and both
  the owner's own profile and another user's profile (confirming the
  Submissions link is correctly absent on the latter) in the arcade demo
  skin.

`task/merge-tasks-nav-uiux-polish` consolidated the navbar further, at the
user's explicit direction: several destinations that were separate top-level
nav items/menus were "useless" as their own destinations and should live on
the Tasks page instead. The nav went from 8 items (2 flat + Work/Manage/
Account menus) down to 4 (Overview, Tasks flat, Manage/Account menus).

- **Tasks became the one hub for everything task-related.** Merged what
  used to be four separate nav destinations into one page:
  - **New task**: no longer a top-level nav link; a "+ New task" button now
    sits at the top of the Tasks hub and links to the same `/tasks/new`
    route (`CreateTaskPage` itself is unchanged — only its entry point moved).
  - **Discovery**: fully merged — `DiscoveryPage` no longer exists as a
    route. Its "Discover public tasks" section is now permanently embedded
    on the Tasks hub, always expanded (equally primary to "My tasks").
    `/discovery` and `/series` (bare) redirect to `/tasks` for anyone with
    an old bookmark rather than 404ing.
  - **Work menu (Submissions, Series)**: also fully merged — `SeriesListPage`
    no longer exists as a route (its content is a collapsed-by-default
    `Ui.disclosure` on the hub, since `nav-series-list` was clicked by only
    one spec). `UserSubmissionsPage` **stays as its own route**, unlike
    Discovery/Series — the Profile page's "Submissions" link
    (`user-submissions-link`) explicitly targets `/users/<id>/submissions`
    as its own linkable URL, and a real test (`screens.spec.ts`) asserts
    that URL loads; removing it would have broken a genuine, still-used
    entry point, not just an unused nav item. Its content is also embedded
    (collapsed) on the hub via a shared content-only function, reused by both
    the standalone page and the hub — verified this distinction by grepping
    actual test usage before deciding, the same discipline as prior branches.
  - **Inbox** moved from a flat nav item into the Account menu.
- **Two real, pre-existing bugs found while doing this refactor** (not
  visible until Discovery/Series/Submissions could be reached from the same
  page as Tasks): `PreviousUserSubmissionsPageClicked`/
  `NextUserSubmissionsPageClicked` and the series-list refresh after
  creating a series (`seriesListRefresh`) both gated on `state.page` being
  the exact standalone route (`UserSubmissionsPage userId` / `SeriesListPage`)
  — so paginating submissions or creating a series from the *hub* would have
  silently no-op'd (wrong page match) instead of refreshing. Fixed the
  submissions handlers to key off `state.subjectId` directly instead of
  extracting a userId from the current page, and extended the series
  refresh condition to include `TasksPage`.
- **Inline task funding**: a task's "Fund this task" panel is now a
  default-collapsed `Ui.disclosure` inside `ownerControlsCard` (owner-only,
  gated the same way the existing Open/Cancel/Refund buttons are — only for
  draft/open tasks), reusing the exact same `FundTaskIdChanged`/
  `FundAmountChanged`/`FundClicked`/`fund-*` plumbing as the standalone
  Funding page — no new Msg needed. Correctness subtlety: `state.fundTaskId`
  is now synced to the currently-viewed task on every entry to a task detail
  page (`enterPage`), so submitting always targets *this* task regardless of
  whatever was last selected on the standalone page — considered and
  rejected an "auto-open the panel when arriving via a shortcut" design
  (would have required tracking open-intent separately from the task-id sync
  to avoid the panel silently targeting the wrong task) in favor of this
  simpler, unambiguously-correct version. Also added a "Fund" button
  alongside "View" on each My-tasks row (gated the same way, draft/open only)
  that jumps straight to the task's detail page.
- **A real mobile overflow bug found by the existing (extended) horizontal-
  overflow check**: with only Overview/Tasks/Manage on the first row now
  (Work having been removed), Manage sat in a different horizontal position
  than before and its left-aligned dropdown panel overflowed a 375px
  viewport. Fixed by aligning its panel to the right, matching the same fix
  already applied to Work in the prior branch for the same underlying reason.
- **A real strict-mode test failure found by running the suite, not by
  review**: mara's seeded task (`task-ledger-review`) is both her own and
  public, so once My tasks and Discover public tasks render on the same
  page, its title appears twice — an unscoped text assertion in
  `demo.spec.ts` broke. Scoped it to the `tasks` (My tasks) container
  specifically.
- Verified: all 46 real-backend Playwright specs (`make e2e-ui`), all 13
  WASM-demo Playwright specs, Go integration/http_e2e suites, `go test
  ./...`, and the full non-browser check suite all pass. Screenshot-verified
  the consolidated hub (My tasks/Discover public tasks always expanded, My
  submissions/Series collapsed), the Fund panel's collapsed-by-default state
  and its expansion via both entry points, and the Account menu with Inbox
  added, in both the base theme and the arcade demo skin.

`task/navbar-dropdown-menu-more-seed-tasks` turned the prior branch's
3-row/15-button nav grouping into a genuinely structured navbar with real
dropdown menus, and substantially grew the WASM demo's seeded task volume.
Screenshots (both the arcade demo skin and a standalone static mockup) were
used to align on the design with the user before implementing.

- **Navbar collapses to one row.** Overview/Tasks/New task/Discovery/Inbox
  stay flat (the busiest links, per the same test-usage grep methodology as
  the prior branch: Tasks is clicked 13×, Discovery 8× across the spec
  suite — moving either would have forced ~20 test edits for no real
  decluttering win). Submissions/Series fold into a new "Work" menu;
  Funding/Collectibles/Agents/Organizations fold into "Manage";
  Profile/Admin/Log out/Reset demo fold into "Account". 15 buttons across 3
  rows became 8 items in one row (two on the narrowest viewports, since the
  app's `max-w-3xl` content column — not the browser width — is what
  actually constrains wrapping).
- **First attempt used native `<details>`/`<summary>` for the dropdowns**
  (matching `Ui.disclosure`'s no-Elm-state philosophy) and looked right in a
  screenshot, but had two real bugs caught by the existing Playwright suite,
  not by eyeballing screenshots:
  - Elm's `elm/virtual-dom` silently drops `onclick` (and other `on*`)
    attributes set via `Html.Attributes.attribute` — a deliberate security
    measure, not a bug — so an attempted inline-JS fix to close the dropdown
    on navigation was a complete no-op (confirmed by checking
    `getAttribute("onclick")` in a real browser: `null`).
  - Without that, a native `<details>` dropdown has no way to close itself
    when a link inside it navigates: the DOM node's `open` state is
    untouched by navigating, so the floating panel kept rendering on top of
    whatever page loaded next and intercepted clicks on it — caught by
    `locator.click: ... subtree intercepts pointer events` failures in
    `screens.spec.ts`'s organization-creation tests, not by visual review.
  - Fixed by making the menu Elm-controlled instead: one
    `openNavMenu : Maybe String` field, one `ToggleNavMenu String` message,
    and — the key part — `enterPage` (already the single place every route
    change flows through) now resets `openNavMenu` to `Nothing` on every
    navigation, so a menu never survives past the page it linked to.
    `Ui.navMenu` became a plain `button` + conditionally-rendered `div`
    (with `aria-expanded`/`aria-haspopup`) instead of `details`/`summary`.
  - This in turn required re-examining every "does the menu stay open
    across these two clicks" assumption made while updating tests for the
    prior native-details version: it doesn't, ever, now — each nav-menu
    item click needs the menu re-opened immediately before it, even two
    items in the same menu back to back, if a navigation happened in
    between.
- **A real, previously-invisible mobile overflow bug found by the existing
  `mobile.spec.ts` check** (extended to also open each of the 3 new menus
  and check for overflow, not just the pages they link to): the "Work" menu,
  positioned as the 3rd item on its wrapped mobile row, floated a
  fixed-width panel rightward off the edge of a 375px viewport. Fixed by
  aligning its panel to hang from its trigger's right edge instead of its
  left (`Manage` needed to stay left-aligned, since on mobile it wraps to
  being the *first* item on its own row — aligning it right would have
  overflowed the *left* edge instead).
- **Demo seed data grew from 6 tasks to 14** (`site/demo/wasm-host.js`):
  added `security_review`/`product_review`/`ui_ux_review`-typed tasks, a
  second task for the demo's own logged-in user (`user-mara`, since Tasks —
  "My tasks" — only ever showed her `task-ledger-review` before), a `closed`
  task for realism (added a `state` override, previously hardcoded to
  `"open"` for every seeded task), and spread the rest across
  `user-sol` (previously owned nothing) and the existing owners. Re-verified
  the frozen invariants from the prior branch (mara's 1250 credits, the
  org's 7200 credits, the 25-entry catalog count) still hold — none of the
  new tasks touch a ledger entry.
- Fixed a `check:policy` failure caught by the pre-commit hook, not by
  review: the project forbids the literal `null` in TypeScript source, and
  the mobile-overflow loop's "no menu needed for this link" placeholder used
  `null` — switched to an empty string sentinel instead.
- Verified: all 46 real-backend Playwright specs (`make e2e-ui`), all 13
  WASM-demo Playwright specs, Go integration/http_e2e suites, `go test
  ./...`, and the full non-browser check suite (`check-format`,
  `check-contracts`, `check-openapi`, `check-policy`, `check-ts`,
  `check-copy-paste`, `check-dead-code`, `lint`, `vet`) all pass.
  Screenshot-verified the closed-by-default nav on both desktop and mobile
  viewports, each menu's open state in both the base theme and the arcade
  demo skin, the fixed post-navigation auto-close, and the richer Discovery
  task list.

`task/ui-navbar-declutter-a11y-seed` is a deliberately large, bundled UI/UX
pass across the Elm frontend and WASM demo seed data, explicitly requested as
one PR rather than the project's usual one-task-per-branch split. Research
first: three parallel exploration passes mapped every page/nav/disclosure in
`View.elm`, audited colors/headings/ARIA/focus-state across both the base
Tailwind theme and the demo's pixel-art `arcade.css` skin, and inventoried the
current seed data plus the 24 existing multi-actor Playwright journeys. A
design pass then produced a phased plan, which was corrected in two places by
directly grepping the actual spec files for exact test coupling before
touching any code (see below) rather than trusting an estimate.

- **Navbar redesign.** `navBar` (`View.elm`) is now a semantic
  `<nav aria-label="Primary">` with the same 14 links (plus a new
  `nav-submissions` entry — see below) grouped into three visual rows: daily
  work (Overview/Tasks/New task/Discovery/Inbox/Submissions), build-and-manage
  (Funding/Series/Collectibles/Agents/Organizations), and account/system
  (Profile/Admin/Log out/Reset demo). Every existing `nav-*` `data-testid` is
  unchanged, so no Playwright spec needed to change for the regrouping itself.
- **A worker-journey gap closed.** Per `docs/onboarding.md`, a worker's own
  submissions/revision-inbox is a primary destination, but it was only
  reachable by clicking through Profile first. Added a top-level "Submissions"
  nav link (`nav-submissions`, purely additive testid).
- **Fixed the Profile nav link's missing active-state.** Profile was a raw
  `<a>` that never got the primary-button highlight even when viewing your own
  profile; it's now routed through the same active-state comparison every
  other nav link uses.
- **A much bigger pre-existing bug this surfaced**: fixing Profile's
  active-state in the Elm model didn't change what rendered on screen in the
  demo. Traced it to `site/demo/arcade.css`'s `[data-testid^="nav-"]` rule,
  which set `background: var(--pa-surface) !important` unconditionally on
  every nav-prefixed element — overriding the active page's green
  `bg-slate-900` background regardless of which link was actually active.
  **This meant the demo has never visually shown which page you're on,
  anywhere in the app, since the pixel-art skin was added** — not a
  regression from this branch, a latent bug the Profile fix happened to make
  visible. Fixed by adding a more specific `[data-testid^="nav-"].bg-slate-900`
  override rule matching the same accent color `button.bg-slate-900` already
  uses.
- **Declutter pass, all via the existing `Ui.disclosure` pattern** (native
  `<details>`/`<summary>`, no new Elm state):
  - **Create Task**: grepped every spec file first and found `create-visibility-*`
    is clicked in 13 places (a near-universal step in task-creation tests) while
    `create-participation-*`/`create-assignee-*` are each clicked once and
    `create-owner-*` is never clicked — so Reward and Visibility stay always
    visible (hiding Visibility would have forced 13 test edits for no real
    win), while Owner/Participation/Assignee now collapse into one
    `create-task-ownership` disclosure (2 test edits).
  - **Task detail's "API & MCP" panel** used a bespoke
    `state.taskIntegrationOpen`/`Msg.ToggleTaskIntegration` toggle predating
    `Ui.disclosure`. Replaced with `Ui.disclosure "toggle-integration" False ...`
    — reusing the exact same testid the 3 existing specs already click meant
    zero test edits — then removed the now-dead model field and message case.
  - **`reviewControls`** hand-rolled its label/input markup instead of
    `Ui.fieldLabel`/`Ui.textInput`/`Ui.textarea_`; switched to the shared
    helpers (and picked up the new focus-ring fix below for free).
  - **Collectibles**: wrapped the mint form and the award-to-task form in two
    new disclosures. Discovered mid-implementation that giving a disclosure a
    content-derived `openByDefault` (e.g. "open if the name field is
    non-blank") breaks for forms that reset their own trigger field after a
    successful submit but stay on the same page — the disclosure would snap
    shut right after minting, failing a test that minted twice in a loop.
    Fixed by using a static `False` default instead (a form field's value
    should never be read back into whether its own disclosure stays open).
  - **Account settings**: kept "Save profile" open by default; wrapped Email
    verification, Change password, Privacy requests, and Deactivate account
    into 4 separate disclosures (3 test edits across `account.spec.ts`/
    `demo.spec.ts`/`screens.spec.ts`, all "open the disclosure before the
    existing interaction").
  - **User submissions**: grepped this too before touching it — a real
    backend test checks the Revision timeline's heading/content immediately
    after a fresh page load with no interaction, so it stays always-visible
    (the plan's first draft would have collapsed it and broken that test);
    only the generic "All submissions" list collapsed into a disclosure (1
    test edit).
  - **Task series**: wrapped creator-controls and the comments section into
    two disclosures (tasks-in-series stays open); 2 test edits in the one
    real-backend test that drives both sections.
- **Accessibility pass**:
  - **Per-page `<h1>`**: previously the entire app had exactly one `<h1>`,
    the static "Sharecrop" wordmark, which never changed per route — every
    page's actual title was an `<h2>` with no top-level heading above it.
    `pageView`'s 19-case dispatch now returns a `(title, content)` pair and
    wraps every page in its own `Ui.pageTitle`. The logged-out screen (not
    part of the routed `Page` union) keeps "Sharecrop" as its own `<h1>`,
    matching `app.spec.ts`'s existing `getByRole("heading", {name:
    "Sharecrop"})` check; the wordmark is dropped entirely once logged in
    rather than duplicated alongside each page's new title. Also removed 4
    now-literally-duplicate inner `<h2>`s (Collectibles, Organizations, User
    submissions, Inbox all had an `Ui.sectionTitle` repeating the exact page
    name the new `<h1>` already says).
  - **Decorative sprites**: `Sprites.elm`'s pixel-art collectible renderer had
    zero text alternative. Every call site already renders the collectible's
    name as adjacent visible text, so added `aria-hidden="true"` once at the
    sprite's root element rather than threading a label through 4 call sites.
  - **Color-differentiated status badges**: added `Ui.badgeVariant` (neutral/
    success/warning/danger tones, contrast ratios documented in the function's
    doc comment) and applied it to task/submission/collectible/series state,
    privacy-request status, and moderation-report state — previously all
    flat gray regardless of state.
  - **Focus states**: `fieldClass`/`textareaClass` suppressed the native
    outline with only a border-color change as a (weak) replacement; added a
    `focus-visible:ring-2` that's additive with the demo skin's existing
    compensating `outline: 3px solid` rule (a box-shadow-based ring doesn't
    collide with an outline). `dangerButtonClass` was the only button class
    missing `min-h-[44px]`; added it.
- **Demo seed data expansion** (`site/demo/wasm-host.js`). Before touching
  anything, grepped `demo.spec.ts`/`mobile.spec.ts` for every hardcoded
  count: mara's "1250 credits" balance is asserted 13 times, the org's "7200
  credits" once, and the catalog's 25-entry count once — all three are
  frozen invariants. Added, without touching any of them: credit-ledger
  entries for jules/ren/tala/sol (previously all zero); an organization team
  ("Field Crew") and two more organization members; `funded_organization_id`
  set on one task so the "funded from org balance" state is visible; two
  more discoverable tasks with different owners/participation policies; a
  reservation, a submitted-and-pending submission, and an unread inbox
  notification; and two awarded collectible instances. **A single-actor-demo
  correction made mid-implementation**: the demo always logs in as `user-mara`
  (there's no real multi-user session), and `/api/collectibles` /
  notifications are scoped to the authenticated actor server-side — so the
  first draft of this seed data (a submission/reservation/notification
  against a jules-owned task, and both awarded collectibles going to
  jules/ren) would have been entirely invisible to whoever actually opens the
  demo. Redirected the submission/reservation/notification to
  `task-ledger-review` (already owned by mara) and one collectible to mara
  directly, verified by screenshot that her Inbox and Collectibles page now
  show real content on a fresh load.
- Verified: all 46 real-backend Playwright specs (`make e2e-ui`), all 13
  WASM-demo Playwright specs (`demo.spec.ts`/`mobile.spec.ts`), the Go
  integration/http_e2e suites, `go test ./...`, `go vet ./...`,
  `go tool deadcode -test ./...`, `make check-format`/`check-contracts`/
  `check-openapi`/`check-policy`/`check-ts`/`check-copy-paste`, and
  `deno task test`/`deno task lint` all pass. Screenshot-verified the navbar
  active-state fix, the per-page `<h1>` structure, and the enriched seed data
  (Discovery, Organization detail, Team detail, Inbox, Collectibles) in a
  real browser.

`task/task-series-wasm-support` continued the hand-testing pass onto the Task
Series feature and found the biggest gap yet: the whole feature had zero WASM
demo support, not just one missing route.

- **A fifth `internal/wasmdemo` bug, found by creating a task series in a
  browser.** `/api/task-series` (list, create) and `/api/task-series/{id}`
  (detail) were entirely unclassified — creating a series through the browser's
  "Task series" page failed outright with a 404. Implemented `StoredTaskSeries`
  storage (`SaveTaskSeries`/`ListTaskSeries`/`LoadTaskSeries` in
  `browser_storage.go`, mirroring the existing organization/team storage
  pattern) and a new `TaskSeriesHandler` (`request_handler.go`) covering
  create/list/detail, wired through a new `TaskSeriesIDSource` interface and
  `jsHostIDs.NextTaskSeriesID()` in `cmd/sharecrop-wasm/main_js_wasm.go`.
- **Found a second bug the same way, one layer deeper.** After adding the route,
  creating a series failed again with a different error: "Expecting an OBJECT
  with a field named `series`". Reading `internal/http/series.go`'s
  `writeSeriesMutation` showed `createTaskSeries` actually returns the full
  `{series, tasks, comments}` detail wrapper, not a bare series object as every
  other create endpoint in this codebase does — an easy assumption to get wrong
  without checking, which is exactly what happened. Fixed by wrapping the create
  response the same way as the detail response.
- **Deliberately scoped out series lifecycle, membership, and comments.**
  Publishing, closing, reopening, adding/removing/reordering tasks in a series,
  and series comments are NOT implemented — verified live that clicking
  "Publish" on a freshly created series shows a graceful inline error ("The
  request failed with status 404.") rather than crashing the page or corrupting
  state. Documented as a known, larger remaining gap in `BUGS.md` and
  `DO_NEXT.md`, alongside the similarly-scoped team-membership gap from the
  prior branch — implementing full series CRUD (a state machine plus ordered
  task membership plus a comment thread) is a meaningfully bigger undertaking
  than a single missing route, and out of scope for this fix.
- Added a new `demo.spec.ts` test (`demo creates and opens a task series`)
  alongside the `internal/wasmdemo` regression test. Verified: all 12 WASM-demo
  Playwright specs, all 46 real-backend Playwright specs (including the existing
  real-backend series lifecycle test, unaffected), the Go integration/http_e2e
  suites, and the full non-browser check suite pass.

Also: PR 108's GitHub Pages deployment failed three times after merge with three
different symptoms — a 10-minute timeout, then "multiple artifacts...
unexpectedly found" after re-running only the failed job (which re-uploads the
artifact under the same run, creating a duplicate), then "deployment cancelled",
then another 10-minute timeout stuck at `deployment_queued`. Build and
artifact-upload steps succeeded every time; only the final `deploy-pages` step
failed or hung. Stopped auto-retrying after three failures and flagged it for a
human. PR 109's deployment then succeeded on the first try with no code or
workflow changes, confirming it really was a transient GitHub-side issue that
has since cleared.

`task/team-detail-404-fix-and-declutter` (merged into `main`) continued the
hand-testing pass onto the team detail page, found a fourth real bug the same
way, and applied `Ui.disclosure` there too.

- **A fourth `internal/wasmdemo` bug, found by creating an organization team in
  a browser and clicking into it.** `GET /api/teams/{team_id}` was completely
  unclassified in the WASM demo's route adapter — not a wrong shape, an outright
  missing route, so the team detail page could never load for any team,
  standalone or organization-owned. Fixed by adding `teamDetailPathID` (a
  3-segment `/api/teams/{id}` match, distinct from the existing 4-segment
  `/work`/`/collectibles` suffixed routes) and a `handleTeamDetail` method on
  the existing `OrganizationHandler`, returning `{team, members: []}` via a new
  `teamDetailBody` type. Members is always empty because the WASM demo has no
  team-membership storage at all yet (only organization membership) — matching
  what a freshly created team with nobody added would return from the real
  backend's `getTeam`/`teamDetailResponse`, not a guess.
- **Applied `Ui.disclosure`** to the team detail page's team-work
  search/type/sort/state-filter/saved-views panel (`teamWorkDashboard`), cutting
  a representative team page from ~1200px to ~730px, consistent with the same
  pattern applied to Tasks/Discovery/organization-tasks filters in prior
  branches.
- **Fixed the resulting Playwright regression** the same way as prior branches:
  expanded the new `team-work-filters` disclosure before interacting with it in
  `screens.spec.ts`. Verified: all 45 real-backend Playwright specs
  (`make e2e-ui`), the Go integration/http_e2e suites, and the full non-browser
  check suite pass.

`task/org-detail-declutter-and-audit-fix` (merged into `main`) continued the
prior branch's hand-testing pass onto the organization detail page, found a
third real bug the same way, and applied the `Ui.disclosure` component there
too.

- **A third `internal/wasmdemo` bug, found by loading an organization's detail
  page in a browser and reading the "Organization audit" section's error.**
  `GET /api/organizations/{organization_id}/audit-events` was unclassified in
  the WASM demo's route adapter (`request_adapter.go`), falling through to
  `RequestUnsupported` (404). Traced it by patching
  `window.XMLHttpRequest`/`sharecropHandleRequest` to log every request the demo
  makes, since the WASM demo intercepts XHR entirely in JS and never touches the
  real network stack — `page.on("response")` sees nothing. Fixed by adding
  `organizationAuditEventsPathID` and routing it to the existing
  `RouteAuditEvents`/`AdminHandler.handleAuditEvents`, scoped to the
  organization's subject id (the same `ListAuditEvents` the platform-admin audit
  list already uses), matching `internal/http`'s `listOrganizationAuditEvents`.
- **A related Elm-side bug this uncovered**: `OrgAuditEventsReceived`'s error
  case stored the failure in `orgTaskMessage` (evidently copy-pasted from
  `OrgTasksReceived`), so the audit fetch's error rendered under "Organization
  tasks" instead of "Organization audit". Added a dedicated `orgAuditMessage`
  field and wired it to the right panel.
- **Applied `Ui.disclosure`** (introduced in the prior branch) to the
  organization detail page's task filters, Teams, Members (+ provision-member
  form), and Collectibles sections — cutting a representative organization page
  from ~2050px to ~1180px — and to the Collectibles page's admin-only "award a
  default collectible" recipient picker, which was previously mixed into a page
  every user sees regardless of role.
- **Fixed the resulting Playwright regressions** the same way as the prior
  branch: expanded the relevant disclosure before interacting, in both the
  WASM-demo-only and real-backend suites. Verified: all 12 WASM-demo Playwright
  specs, all 45 real-backend Playwright specs (`make e2e-ui`), the Go
  integration/http_e2e suites, and the full non-browser check suite pass.

`task/ui-ux-declutter-and-profile-fix` (merged into `main`) fixed two real
broken browser flows found by hand-testing the Go/WASM demo, and decluttered the
Elm client's busiest pages.

- **Two `internal/wasmdemo` bugs, found by actually loading the demo in a
  browser, not by reading code.** `GET /api/users/{user_id}` matched the demo's
  generic users-collection route and returned the raw stored user record
  (`{id, email, status}`); the real backend's `getUserProfile` (and the
  browser's decoder) expects `{id, tasks}` — tasks the user created — so every
  profile page view failed with a JSON decode error. Fixed in
  `internal/wasmdemo/runtime_handlers.go`'s `UsersHandler.Handle` by listing
  tasks via the existing `"user"` scope on `ListTasks`. Separately,
  `GET /api/users/{user_id}/work` (tasks the user holds an active reservation on
  — the profile's "Public work" tab) did not exist in the demo at all and fell
  through to the same generic route; added
  `userWorkPathID`/`TaskHandler.handleUserWork` (`request_adapter.go`,
  `request_handler.go`), scanning each stored task's reservations for an active
  user-assignee match since reservations are indexed by task, not by assignee.
  Both covered by new regression tests (`TestUsersHandlerListsAndLoadsUsers`,
  `TestTaskHandlerHandleUserWork`) and verified live: `docs/openapi.json` decode
  error gone, "Public work" tab renders "Nothing to show." instead of crashing.
- **A new `Ui.disclosure` collapsible-section component** (`Sharecrop/Ui.elm`,
  native `<details>`/`<summary>`, no Elm model/message wiring needed) applied to
  the Admin page's five always-expanded sections (only Operations stays open by
  default), the Tasks/Discovery filter panels, and Create Task's
  advanced/optional fields (reference URL, raw JSON schema, task input JSON,
  attachments) — cutting the Admin page from ~1900px to ~680px collapsed and
  Create Task from ~1500px to ~975px, while keeping every field reachable. Each
  disclosure that guards state a user might already have set (task/discovery
  filters, create-task advanced fields, a non-freeform template's schema) opens
  by default when that state is non-default, via a plain reactive `Bool`
  computed from the model — Elm re-applies the `open` attribute on every render,
  so e.g. picking a template auto-expands Advanced options with no extra wiring.
- **Found and fixed a virtual-DOM node-reuse bug this surfaced.** Without a key,
  Elm's non-keyed diffing can match two structurally similar pages' `<details>`
  elements at the same tree position as "the same node" and carry over the real
  DOM's native (Elm-invisible) `open` state across a route change — e.g.
  expanding Admin's second section, then navigating to Tasks, showed its first
  section pre-expanded. Fixed by keying the page's root view node by route
  (`Html.Keyed.node` with `pageToPath state.page` as the key) in `loggedInView`,
  forcing a fresh subtree on every navigation.
- **Fixed the resulting Playwright regressions.** Six existing `demo.spec.ts`
  tests, plus several in `screens.spec.ts` (real-backend suite), clicked
  directly into sections now collapsed by default; added an explicit
  disclosure-summary click before each. Verified: all 12 WASM-demo Playwright
  specs, all 45 real-backend Playwright specs (`make e2e-ui`), the Go
  integration/http_e2e suites, and the full non-browser check suite pass.

`task/openapi-schema-field-access` (merged into `main`) closed the one confirmed
gap the prior branch left open: `createModerationReport`'s response schema
stayed generic because it's written as `writeJSON(w, status, response.value)` —
a struct field access on a result-union wrapper, not a bare local variable or
composite literal.

- **Two new resolution patterns in `internal/openapi/dto_resolve.go`.** A
  two-value type assertion
  (`response, ok := result.(moderationReportConverted)`) now records
  `response`'s type from the assertion itself. A field-access expression
  (`response.value`) now resolves the base variable's type, then looks up that
  struct's field type — including fields with no JSON tag at all, since the
  struct being accessed (`moderationReportConverted`) is an internal
  result-union wrapper, not a DTO. Added
  `collectStructFieldTypes`/`collectStructTypeDecls` (the latter factored out of
  `collectStructShapes` so both share one AST traversal) for this, distinct from
  the JSON-tag-filtered `collectStructShapes` used for schema generation.
- **Verified against the real handler**, not just the synthetic test: after the
  fix, `docs/openapi.json`'s `POST /api/moderation/reports` response schema has
  real properties, response resolution went from 98/106 to 99/106 with no other
  route regressing, and the generator is still deterministic across repeated
  runs.

`task/openapi-typed-schemas` (merged into `main`) added typed per-route
request/response JSON schemas to `docs/openapi.json`, derived from the actual
`internal/http` Go DTO structs via `go/ast` rather than a hand-authored mapping
to `internal/contracts` (research beforehand found `internal/contracts` has no
request-body types at all and its response shapes have already drifted from
`internal/http`'s in field order, confirming a direct DTO-source approach was
both more complete and safer than trusting the two shapes stay in sync).

- **Struct-shape extraction (`internal/openapi/structs.go`).** Parses every
  `type X struct {...}` declaration in `internal/http` into a JSON field shape:
  wire name and required-ness from the `json` tag (`,omitempty` absent means
  required), and a `FieldKind` (string/integer/number/boolean/ array/struct)
  from the Go field type, with pointers unwrapped and anything not confidently
  classified left as unconstrained rather than guessed.
- **Request/response type resolution (`internal/openapi/dto_resolve.go`).**
  Finds, per handler, the request DTO from its `var request <Type>` +
  `.Decode(&request)` pair and the response DTO from a dedicated
  `write<Foo>Response` wrapper or the argument to a direct `writeJSON` call
  (composite literal, local variable, or single-return-value converter call),
  walking the local call graph transitively so handlers that delegate to a
  shared helper (`openTask`/`cancelTask` → `changeTaskState`) still resolve.
- **Two real bugs found and fixed via testing, not inspection.**
  `writeError(w, status, message string)` has the exact same
  `(http.ResponseWriter, int, <named type>)` shape as a real
  `write<Foo>Response` wrapper, so an early auth-check's error write was winning
  over the handler's actual success-path response — excluded builtin-primitive
  third parameters. Separately, `writeJSON`'s own signature
  (`value writableResponse`) matched the same wrapper pattern against itself, so
  every `writeJSON` call was resolving to the interface name
  `"writableResponse"` instead of its actual argument — excluded `writeJSON` by
  name. Both are now regression tests.
- **Schema generation (`internal/openapi/document.go`).** Recursively builds an
  object schema from a resolved struct, with a cycle guard scoped to the current
  call chain (not a global visited set) so the same struct can legitimately
  appear as sibling fields without being mistaken for a cycle.
- **Result: 98/106 responses and 39/61 request bodies now resolve to a typed
  schema.** The remainder are genuinely out of scope for this generator (three
  `/mcp` JSON-RPC passthrough routes, `healthz`/`index`/`/static/`, bodyless
  action endpoints) or one known gap (`createModerationReport`'s response is a
  struct-field access, `response.value`, not a bare variable or literal)
  recorded in `DO_NEXT.md`.
- **`site/docs/openapi.html`** gained a Schema column (typed vs. plain per
  route) and a typed-response-schema count in its summary.
- Verified the generator is deterministic (ran it twice, diffed byte-for-byte
  identical) since the schema/property maps could otherwise leak Go's randomized
  map iteration order into the committed JSON.

`task/landing-docs-visual-redesign` (merged into `main`) fixed the unstyled
`site/index.html` and `site/docs/index.html` pages found during the prior task,
with a full visual redesign rather than a minimal patch (explicit user choice).

- **`site/marketing.css`.** A new hand-authored stylesheet implementing every
  class those two pages already referenced but that had no matching CSS anywhere
  (`.landing-shell`, `.landing-hero`, `.hero-copy`, `.hero-actions`,
  `.landing-board`, `.board-column`, `.status-pill`, `.docs-shell`, `.panel`,
  `.topbar`, `.brand`, `.button`/`.primary`/`.secondary`, `.eyebrow`,
  `.objective-list`). Hand-authored rather than routed through the Tailwind/Elm
  build pipeline: these are static content pages with no build step of their
  own, and the chosen aesthetic (paper texture, dashed tears, keyframes) is
  mostly literal CSS that would not benefit from Tailwind utility composition.
  Also fixed the broken `demo/styles.css` link both pages had (a file no build
  step produces).
- **A distinctive aesthetic, not a generic fix.** A "dispatch desk"
  paper/typewriter direction: kraft-paper texture, Special Elite display type
  paired with IBM Plex Sans/Mono (fonts loaded from Google Fonts, matching the
  existing pattern in `site/demo/index.html`), rust/navy ink accents,
  rubber-stamp buttons with a pressed hover/active state, work-order cards on
  the landing page pinned at slightly different rotations, carbon-copy-styled
  `<pre>` blocks, and a staggered fade/rise page-load animation. Deliberately
  distinct from the demo's existing pixel-arcade skin (`site/demo/arcade.css`)
  rather than reusing it.
- **Restyled `site/docs/openapi.html` to match.** The subpage added in the prior
  task had its own inline blue/slate style block; recolored its summary cards,
  table, and method badges to the same palette and swapped its topbar/panel
  markup to reuse the shared `marketing.css` classes, so all three static pages
  now read as one system. Fixed a `margin-left: auto` selector bug this
  surfaced: `nav.topbar a.button` applied to every button in the bar, and with
  two buttons (this page has "Docs" and "Demo" instead of one) flexbox gave each
  its own auto margin instead of only pushing the first one right. Narrowed to
  `nav.topbar .brand + a.button`.
- **Verified by rendering, not just by reading the CSS.** Loaded all three pages
  in a real Chromium browser via Playwright at desktop and mobile viewports,
  checked for console/page errors (none), and reviewed screenshots; confirmed
  `site/docs/openapi.html`'s route table still rendered 106 rows with correct
  summary counts after the restyle, and that `tools/check_pages_routing.ts`
  still passed against a local static server.

`task/openapi-pages-subpage` (merged into `main`) published the generated
OpenAPI document on the deployed GitHub Pages site and closed a CI gap left by
the prior branch.

- **OpenAPI Pages subpage.** `site/docs/openapi.html` fetches
  `site/docs/openapi.json` and renders a method/path/operationId/auth-
  requirement table plus a public/protected summary count, entirely with vanilla
  JS and a self-contained inline stylesheet — no Swagger UI/Redoc or other
  third-party viewer, matching the deployed site's existing pattern of
  self-hosted assets only. `site/docs/openapi.json` is a committed copy of
  `docs/openapi.json` kept in sync by `deno task site:openapi:copy`, folded into
  `make openapi`/`make check-openapi`. `site/docs/index.html`'s References list
  links to the new page.
- **Verified rendering, not just presence.** Served `site/` with a local static
  file server and loaded `site/docs/openapi.html` in a real Chromium browser
  through Playwright: the table rendered 106 rows with zero console/page errors,
  and the summary (106 routes, 12 public, 94 requiring a bearer token) matched
  `docs/openapi.json`.
- **Pages routing check extended.** `tools/check_pages_routing.ts` now checks
  `/docs/openapi.html` and `/docs/openapi.json` after deployment, verified
  locally against the same static server before relying on the post-deploy Pages
  workflow step.
- **Found and fixed a CI gap from the prior branch.**
  `task/generated-openapi-
  reference` added `make check-openapi` to the
  Makefile but never wired it into `.github/workflows/ci.yml`, so PR CI was not
  actually enforcing that `docs/openapi.json` stays in sync with the route
  table. Added a "Check OpenAPI document" step to the `static` job, next to
  "Check contracts".
- **Found a pre-existing styling defect, left unfixed as out of scope.**
  `site/index.html` and `site/docs/index.html` link a `demo/styles.css` that no
  build step produces (confirmed 404 on the live deployed site) and use custom
  classes (`.docs-shell`, `.panel`, `.topbar`, ...) that `web/styles/input.css`
  never `@source`-scans, so `app.css` has no rules for them either — both pages
  render fully unstyled in production. Recorded in `BUGS.md`/`DO_NEXT.md`; the
  new OpenAPI page avoids the issue with its own inline styles rather than
  depending on the broken shared stylesheet.

`task/generated-openapi-reference` (merged into `main`) closed the "no generated
OpenAPI reference" documentation gap recorded in
`docs/application_readiness_review.md`.

- **Generated OpenAPI document.** `go run ./cmd/sharecrop generate openapi`
  (`make openapi`) writes `docs/openapi.json`, an OpenAPI 3.0 document with an
  accurate `paths`/method/`operationId` inventory and a global bearer security
  requirement. `make check-openapi` regenerates and asserts no diff, mirroring
  `check-contracts`.
- **`internal/openapi` package.** `Extract` parses the `internal/http` package's
  non-test source files with `go/ast`, finds every `mux.HandleFunc`/`mux.Handle`
  route registration in `server.go`, and rejects duplicate registrations or
  unparsable source. `Generate` builds the OpenAPI document; `Write` marshals
  and writes it.
- **Auth requirement from the local call graph, not a substring search.** A
  first pass checked each handler body for a direct call to one of five known
  identity-resolving functions (`requireUserSubject`, `requireWorkerSubject`,
  `requireAdminSubject`, `requireOrganizationBilling`, `verifyAgent`) and
  wrongly marked delegating handlers like `openTask`/`cancelTask` (which call a
  shared `changeTaskState` helper, not a gateway directly) as public. The final
  version builds a local call graph and checks transitive reachability, matching
  the real 12 public vs. 94 protected operations.
- **Caught by testing, not by inspection.** An `omitempty` tag on a non-pointer
  empty-slice `Security` field made public and protected routes serialize
  identically (both omitted the field), silently losing the one fact the
  generator exists to prove. Changed `Security` to a pointer so `nil`
  (protected, omitted) and a pointer to an empty slice (public, encoded as `[]`)
  round-trip distinctly, with a test that marshals and unmarshals the real
  `Document` type to catch a regression.
- **Docs.** `docs/api_reference.md`, `README.md`, `docs/onboarding.md`, and
  `docs/application_readiness_review.md` link the generated document and record
  that its request/response bodies are generic JSON object placeholders, not
  typed per-route schemas.

`task/wasm-nonbrowser-host-measurement` (merged into `main`) closed the two
remaining "production WASM gates" recorded in `docs/wasm_demo_backend_spike.md`:
runtime measurement and non-browser host documentation.

- **Runtime measurement tool.**
  `deno task measure:wasm -- --wasm
  <compiled.wasm> [--requests-per-route <n>]`
  (`tools/measure_wasm_runtime.ts`) loads a compiled `cmd/sharecrop-wasm`
  artifact through the non-browser reference host and reports artifact size,
  startup time (instantiate through first status call, and host configuration),
  Deno host-process memory before load/after configure/after N requests, and
  per-route request latency (min/mean/p50/p95/max).
  `docs/wasm_demo_backend_spike.md` records a baseline run against
  `site/demo/sharecrop-wasm-backend.wasm`.
- **Non-browser host adapter reference documented.** `createHost` in
  `tools/wasm_runtime_loader.ts` (already exercised by
  `check:scenario-parity:wasm`) is now documented as the reference non-browser
  `HostFunctions` implementation, alongside what a production non-browser host
  still needs that this reference host does not provide: persistent storage, a
  real clock, verified-session actor resolution, and cryptographically random
  IDs/secrets (the reference host's sequential `NextAgentCredentialSecret` is
  explicitly flagged as unsafe to reuse in production, unlike the real backend's
  `crypto/rand`-based secret in `internal/agent/values.go`). Docs that
  overstated existing randomness/ networking host adapters were corrected: no
  route currently needs one, so neither exists in the `HostRuntime` interface
  yet.
- **Removed tool duplication.** `tools/run_wasm_scenario_parity.ts`'s Go/WASM
  loader, host, and request-assertion helpers were extracted into
  `tools/wasm_runtime_loader.ts` so the new measurement script could reuse them
  without duplicating code (`make check-copy-paste` stayed at zero clones).
- **Docs.** `docs/wasm_demo_backend_spike.md`, `docs/demo_semantic_parity.md`,
  and `docs/application_readiness_review.md` were updated to reflect the
  measurement tool, the non-browser host reference, and the corrected
  randomness/networking adapter status.

`task/wasm-default-demo-shared-parity` (PR 100, merged into `main`) made the
compiled Go/WASM backend the default static-demo backend:

- **Default Go/WASM demo backend.** `site/demo/index.html` now defaults to
  `wasm`, loads `wasm-host.js`, requires `wasm_exec.js` and
  `sharecrop-wasm-backend.wasm`, configures explicit browser host functions,
  seeds deterministic demo data, and routes `/api/*` XHR requests through
  `sharecropHandleRequest`. The legacy `backend.js` path is not loaded as a
  fallback.
- **Expanded WASM behavior slices.** `internal/wasmdemo` now has explicit
  storage and handler support for users, account tokens, platform admins, audit
  events, collectibles, agent credentials, auth/account routes, admin
  operations, privacy resolution/retention, moderation projection writes/triage,
  task lists and actions, team work, submission validation, sensitive-field
  indexing, notifications, ledger, organizations, teams, comments, and
  reservations.
- **Shared parity through WASM.** The WASM scenario runner seeds explicit host
  storage, configures the compiled Go artifact, verifies unconfigured requests
  fail, and runs the shared scenario parity suite through
  `sharecropHandleRequest` without calling `site/demo/backend.js`.
- **Docs and continuity.** Demo semantic parity, WASM target docs, readiness
  review, Playwright comments, status, next-task queue, and risk notes were
  refreshed to record the Go/WASM demo default and remaining production WASM
  hardening work.
- **CI wiring.** `make e2e-ui` now builds the demo WASM artifacts before
  Playwright starts the static demo server, so clean CI runners have
  `wasm_exec.js` and `sharecrop-wasm-backend.wasm` available.

`task/wasm-browser-host-full-parity-gates` expanded the browser-facing Go/WASM
backend path:

- **Browser WASM host path.** The demo now supports an explicit `?backend=wasm`
  mode. It loads `wasm-host.js`, requires `wasm_exec.js` and
  `sharecrop-wasm-backend.wasm`, configures explicit browser host functions, and
  routes `/api/*` XHR requests through `sharecropHandleRequest`. Unknown backend
  modes, missing artifacts, missing host functions, invalid host values, and
  missing storage keys fail loudly.
- **Demo WASM artifact build.** `deno task wasm:demo:build` builds the Go
  `cmd/sharecrop-wasm` artifact into `site/demo/sharecrop-wasm-backend.wasm` and
  copies Go's `wasm_exec.js`. The Pages workflow runs this task before uploading
  the static site, and the generated artifacts are ignored rather than
  committed.
- **Expanded WASM dispatch and scenario.** The WASM command now dispatches
  existing explicit privacy request, saved queue view, notification,
  organization/member/team, task, comment, reservation, submission, and ledger
  route families where handlers exist. The WASM scenario runner now covers
  privacy request, saved queue view, organization/member/team, task/comment/
  reservation/submission acceptance, ledger, and unsupported-route failure
  checks through the compiled Go WASM binary.
- **Docs and continuity.** WASM target docs, status, next-task queue, and risk
  notes were refreshed to record the opt-in browser path and remaining slices.
- **Verification.** Passed: `go test ./...`; `deno task check:ts`;
  `deno task lint`; `deno task check:policy`; `deno task test`;
  `deno fmt --check deno.json tools tests site/demo/index.html
  site/demo/wasm-host.js site/demo/backend.js`;
  `make check-contracts`; `go tool deadcode -test ./...`;
  `deno task wasm:demo:build`;
  `deno task check:scenario-parity:wasm -- --wasm
  site/demo/sharecrop-wasm-backend.wasm`;
  and `git diff --check`.

`task/wasm-host-adapters-scenario-parity` wired the first configured Go/WASM
request execution path:

- **Configured WASM host.** `cmd/sharecrop-wasm` now exports
  `sharecropConfigureHost` alongside status and request handling. Requests fail
  before host configuration. A configured host must provide explicit storage,
  clock, actor, and ID adapters.
- **Request execution.** The WASM bridge dispatches task routes through
  `TaskHandler` and task-comment, submission-comment, reservation, submission,
  and ledger routes through `InteractionHandler`. Missing host capabilities,
  unsupported routes, and handler errors return explicit error responses.
- **WASM scenario runner.** `tools/run_wasm_scenario_parity.ts` now loads the
  compiled Go WASM artifact, verifies the unconfigured failure, configures an
  explicit host storage adapter, and runs task creation, task comments,
  reservation approval, submission creation, submission comments, acceptance,
  worker balance, and worker ledger checks through `sharecropHandleRequest`
  without calling `site/demo/backend.js`.
- **Docs and deployment check.** WASM target docs, status, next-task queue, and
  risk notes were refreshed. Deployed GitHub Pages routing passed for
  `https://e6qu.github.io/sharecrop`.
- **Verification.** Passed: `go test ./...`; `deno task check:ts`;
  `deno task lint`; `deno task check:policy`; `deno task test`;
  `deno fmt --check deno.json tools tests site/demo/backend.js`;
  `make
  check-contracts`; `go tool deadcode -test ./...`;
  `GOOS=js GOARCH=wasm go
  build -o /private/tmp/sharecrop-wasm-backend.wasm ./cmd/sharecrop-wasm`;
  `deno task check:scenario-parity:wasm -- --wasm
  /private/tmp/sharecrop-wasm-backend.wasm`;
  `deno task check:pages-routing --
  --origin https://e6qu.github.io/sharecrop`;
  and `git diff --check`.

`task/wasm-submission-parity-host-adapters` expanded the Go/WASM backend target
and interaction parity groundwork:

- **WASM interaction slices.** `internal/wasmdemo` now classifies task-comment,
  submission-comment, reservation, submission, user-submission, user-ledger, and
  organization-ledger routes. It has explicit browser-storage boundaries and
  request handlers for task/submission comments, reservation create/list/
  approve/decline/cancel, submission create/list/accept, user submission lists,
  ledger entries, and balances. Missing storage, clocks, actors, ID sources,
  invalid pages, invalid attachments, invalid records, invalid reservation
  transitions, invalid submission-task links, unsupported methods, and
  unsupported routes reject explicitly without fallback stores.
- **Host adapter shape.** `internal/wasmdemo` now defines explicit host request
  and host runtime validation shapes. A host must provide storage, clock,
  actor/session, and interaction-ID adapters before request execution can run.
- **Compiled Go WASM target.** Added `cmd/sharecrop-wasm`, which builds with
  `GOOS=js GOARCH=wasm` and exports `sharecropWasmBackendStatus` and
  `sharecropHandleRequest`. The current exported handler classifies routes and
  fails with a host-adapter-required error until host runtime adapters are
  wired.
- **WASM runner.** Added `tools/run_wasm_scenario_parity.ts` and
  `deno task check:scenario-parity:wasm`, which load a compiled Go WASM artifact
  through Go's `wasm_exec.js` and verify required Sharecrop exports without
  calling `site/demo/backend.js`.
- **Docs.** The WASM target, status, next-task queue, and risks were refreshed
  to record the new interaction slices, host adapter requirement, and remaining
  work before the JavaScript demo backend can be replaced.
- **Verification.** Passed: `go test ./...`; `deno task check:ts`;
  `deno task lint`; `deno task check:policy`; `deno task test`;
  `deno fmt --check deno.json tools tests site/demo/backend.js`;
  `make
  check-contracts`; `go tool deadcode -test ./...`;
  `GOOS=js GOARCH=wasm go
  build -o /private/tmp/sharecrop-wasm-backend.wasm ./cmd/sharecrop-wasm`;
  `deno task check:scenario-parity:wasm -- --wasm
  /private/tmp/sharecrop-wasm-backend.wasm`;
  and `git diff --check`.

`task/wasm-org-team-parity-contracts-rawid` expanded WASM demo groundwork,
organization/member/team parity, contracts, and lifecycle coverage:

- **WASM demo organization/team slice.** `internal/wasmdemo` now classifies
  organization, organization-member, organization-team, and standalone-team
  routes. It has explicit browser storage for organizations, organization
  members, organization-owned teams, and standalone teams, plus request handlers
  for create/list/provision/role-update/deactivate flows. Missing storage,
  actors, ID sources, user resolvers, invalid pages, invalid lifecycle states,
  invalid roles, ownership mismatches, unsupported routes, and unsupported
  methods reject explicitly without fallback stores.
- **Scenario and demo parity.** Shared scenario parity now covers organization
  member provisioning, listing, role update, and deactivation shape. The
  backendless demo returns the real API's deactivation response body for
  organization member deactivation.
- **Contracts and browser coverage.** HTTP fixtures now pin organization-member
  deactivation and organization-owned team response shapes. DB-backed Playwright
  coverage was extended around organization member role and deactivation
  controls.
- **Lifecycle-list fix.** The real Postgres organization member list now returns
  non-removed memberships, so managers can see deactivated members while
  permission checks still require active membership. HTTP E2E coverage verifies
  the deactivated member appears to the owner and no longer has roster access.
- **Docs and raw-ID audit.** WASM spike, demo semantic parity, raw-ID audit,
  status, bugs, next-task docs, local command docs, Docker Compose, CI, and
  Playwright local port configuration were refreshed. Local testing now uses
  non-default ports by convention: Postgres `25432`, app `29180`, and
  backendless demo `29181`. The latest raw-ID scan still found no confirmed
  high-traffic raw-ID browser workflow. The docs now record Go/WASM as a
  first-class backend execution target, not only a demo mechanism; the target is
  a `.wasm` binary compiled from Go with explicit host adapters and no
  JavaScript reimplementation or fallback stores.
- **Verification.** Passed after the lifecycle-list fix: `go test ./...`;
  `deno task check:ts`; `deno task lint`; `deno task check:policy`;
  `deno task test`;
  `deno fmt --check deno.json tools tests
  site/demo/backend.js`;
  `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build
  make check-contracts`;
  `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build
  go tool deadcode -test ./...`;
  `ELM_BIN=/opt/homebrew/bin/elm deno task
  frontend:build`;
  `GOOS=js GOARCH=wasm
  GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build go test -c -o
  /private/tmp/sharecrop-wasmdemo.test.wasm ./internal/wasmdemo`;
  and `tools/run_db_checks.sh` against local PostgreSQL 15 on port `25432`;
  local real scenario parity against the app on port `29180`; DB-backed
  Playwright screens against app port `29180`, demo port `29181`, and Postgres
  port `25432`; and `git diff --check`.

`task/real-parity-wasm-submission-contracts-rawid-attachments` expanded real
parity execution, WASM demo groundwork, contracts, pagination coverage, and
attachment browser coverage:

- **Local real API parity.** Added `tools/run_local_real_scenario_parity.ts` and
  a Deno task that probes `/healthz`, registers a scenario admin, grants
  platform-admin state through `DATABASE_URL` and `psql`, and runs the shared
  scenario suite against a real local API. The explicit real runner now carries
  refresh-token cookies, accepts refresh-token file input, and reports response
  error context in status failures.
- **Shared scenario parity.** The shared suite now satisfies the real
  task-create and submission lifecycle contract by sending explicit task
  placement and opening the submission task before submission. It also verifies
  adjacent one-row pages for personal ledger, organization ledger, and
  notification inbox routes.
- **WASM demo spike.** `internal/wasmdemo` gained explicit notification browser
  storage, notification route classification, and list/mark-read request
  handlers with actor-scoped validation and no fallback store.
- **Contracts and raw-ID audit.** HTTP fixtures now pin task owner, visibility,
  placement, and payload request subshapes. The raw-ID browser-flow audit and
  demo/WASM parity docs were refreshed.
- **Attachment browser edge cases.** DB-backed Playwright coverage now verifies
  rejected task attachment type, oversized file, and five-file limit guardrails
  through the real UI.
- **Verification.** Passed: `go test ./...`; focused Go tests for
  `internal/wasmdemo` and `internal/http`; `deno task check:ts`;
  `deno task lint`; `deno task check:policy`; `deno task test`;
  `deno fmt --check deno.json tools tests site/demo/backend.js`;
  `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build make
  check-contracts`;
  `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build
  go tool deadcode -test ./...`;
  `ELM_BIN=/opt/homebrew/bin/elm deno task
  frontend:build`;
  `GOOS=js GOARCH=wasm
  GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build go test -c -o
  /private/tmp/sharecrop-wasmdemo.test.wasm ./internal/wasmdemo`;
  `tools/run_db_checks.sh` against local PostgreSQL 15;
  `deno task check:scenario-parity:local-real -- --origin
  http://127.0.0.1:18080`
  against the local real API; and DB-backed Playwright
  `tests/playwright/screens.spec.ts` against local PostgreSQL 15.

`task/real-parity-wasm-contracts-pagination-hardening` hardened parity,
pagination, attachments, and the WASM demo spike:

- **Real scenario runner.** `tools/run_scenario_parity.ts` now probes
  `/healthz`, accepts `--token-file`, and reports invalid JSON with request
  context before running the shared scenario suite against a real API.
- **WASM demo spike.** `internal/wasmdemo` now classifies `POST /api/tasks` and
  `GET /api/tasks/{task_id}`, persists task records through an explicit browser
  storage boundary, and handles task create/detail requests with explicit actor,
  ID-source, task, and attachment validation. Invalid routes, methods, missing
  dependencies, invalid bodies, and invalid records reject loudly.
- **Attachment hardening.** Task and submission creation now allow up to five
  attachments per request. The Go API, backendless demo, browser upload guards,
  and WASM attachment storage enforce the limit; each file still remains under
  500 KiB.
- **Contracts and demo parity.** HTTP fixtures now include standalone attachment
  request/response shapes. The backendless demo now paginates personal ledger,
  organization ledger, and inbox notification routes, and its task payload kind
  now matches the real API's `json` value instead of the stale `inline` value.
- **Browser pagination and upload coverage.** The browser now has explicit
  previous/next controls for personal ledger, organization ledger, and inbox
  notifications. DB-backed Playwright coverage verifies creating a task with a
  small attachment through the real backend.
- **Docs and verification.** API, readiness, demo-parity, WASM-spike, plan,
  status, bugs, and next-task docs were refreshed. Passed: `go test ./...`;
  `deno task check:ts`; `deno task lint`; `deno task check:policy`;
  `deno task test`;
  `deno fmt --check deno.json tools tests
  site/demo/backend.js`;
  `make check-contracts` with repo-local `GOCACHE`;
  `go tool deadcode -test ./...` with repo-local `GOCACHE`;
  `ELM_BIN=/opt/homebrew/bin/elm deno task frontend:build`; Playwright
  demo/mobile specs; DB-backed migrations, integration tests, HTTP E2E tests,
  and DB-backed Playwright screens against local PostgreSQL 15;
  `GOOS=js
  GOARCH=wasm go test -c ./internal/wasmdemo`; deployed Pages routing
  check; and `git diff --check`.

`task/parity-contract-wasm-pagination-uploads` added small attachments,
pagination polish, parity coverage, and the next WASM storage slice:

- **Small attachments.** Task creation and submission creation now accept small
  PNG, JPEG, GIF, WebP, plain-text, JSON, or PDF attachments under 500 KiB. The
  backend validates names, content types, data URLs, and decoded size, stores
  bytes inline in Postgres, and returns attachment metadata plus data URLs.
- **Browser and demo.** The Elm app can pick/remove attachments on task and
  submission forms, shows task/submission attachment links, and includes
  Playwright demo coverage for both upload paths. The backendless demo validates
  the same small attachment shape and now normalizes default visibility before
  returning task responses.
- **Contracts and parity.** Generated Elm contracts, HTTP wire-shape fixtures,
  and shared scenario parity now cover attachment request/response surfaces.
- **Pagination.** User submission history now supports `limit`/`offset` through
  the API, backendless demo, and browser previous/next controls.
- **WASM spike.** `internal/wasmdemo` gained explicit attachment browser storage
  for task/submission parents with fail-loud validation and no fallback store.
- **Docs and audits.** API, readiness, raw-ID audit, demo parity, WASM spike,
  status, bugs, and next-task docs were refreshed. Object storage remains out of
  scope; small inline attachments are the current file path.
- **Verification.** Passed: `go test ./...`; focused Go tests for
  attachment/db/http/submission/task/wasmdemo packages; `deno task check:ts`;
  `deno task lint`; `deno task check:policy`; `deno task test`;
  `deno fmt --check deno.json tools tests site/demo/backend.js`;
  `go tool deadcode -test ./...`;
  `ELM_BIN=/opt/homebrew/bin/elm deno task
  frontend:build`; Playwright
  demo/mobile specs; DB-backed migrations, integration tests, HTTP E2E tests,
  and DB-backed Playwright screens against isolated local PostgreSQL 15;
  deployed Pages routing check; and `git diff
  --check`.

`task/postmerge-db-parity-wasm-pagination-coverage` cleaned post-PR-91 state and
expanded parity, pagination, WASM-demo groundwork, and testability:

- **Admin pagination.** The browser Admin page now has explicit pagination for
  audit events, platform-admins, privacy requests, and moderation reports.
  Platform-admin grants refetch page zero so the list stays page-consistent
  while preserving the success message.
- **Backendless demo parity.** `site/demo/backend.js` now honors `limit` and
  `offset` on admin audit, platform-admin, moderation, privacy, and organization
  audit list routes. Shared scenario parity now checks adjacent one-row admin
  audit pages.
- **HTTP contracts.** Wire-shape fixtures now include resolved data-export
  privacy responses with embedded JSON.
- **WASM spike.** `internal/wasmdemo` gained explicit saved-queue-view browser
  storage, route classification, and create/list request handlers. Invalid
  scopes, corrupt records, mismatched storage keys, missing actors, missing ID
  sources for upserts, unsupported methods, and unsupported routes reject
  explicitly without fallback stores.
- **Browser testability.** `tests/playwright/demo.config.ts` runs backendless
  demo/mobile specs without requiring the DB-backed API server. DB-backed
  registration helper failures now include response status and body text.
- **Docs.** Raw-ID audit, readiness review, demo semantic parity, WASM spike,
  status, bugs, and next-task docs were refreshed.
- **Verification.** Passed: `go test ./...`; focused Go tests for
  `internal/http` and `internal/wasmdemo`; `deno task check:ts`;
  `deno task lint`; `deno task test`; `deno task check:policy`;
  `deno fmt --check deno.json tools tests site/demo/backend.js`;
  `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build make
  check-contracts`;
  `go tool deadcode -test ./...`;
  `ELM_BIN=/opt/homebrew/bin/elm deno task frontend:build`;
  `git diff
  --check`; and Playwright demo/mobile specs through
  `tests/playwright/demo.config.ts`. DB-backed migrations, integration tests,
  HTTP E2E tests, and DB-backed Playwright screens passed against isolated local
  PostgreSQL 15 after the Podman machine failed to stay reachable through
  Docker's socket.

`task/parity-wasm-dashboard-revision-polish` expanded parity coverage, WASM demo
groundwork, queue/revision polish, browser coverage, and docs:

- **Scenario parity and demo parity.** Shared scenario parity now covers
  persisted saved queue views. That scenario caught a backendless-demo saved
  queue view status-code mismatch, and `site/demo/backend.js` now returns the
  real API's `200` status for saved-view upserts.
- **HTTP contracts.** Wire-shape fixtures now cover privacy resolution requests
  and saved queue view commands.
- **WASM spike.** `internal/wasmdemo` gained explicit privacy-request browser
  storage plus create/list request handlers. Missing storage, clocks, actors, ID
  sources, invalid kinds, invalid states, unsupported methods, and unsupported
  routes reject explicitly without fallback stores.
- **Browser polish.** Organization task queues, team work sections, revision
  inboxes, and revision timelines now show loaded item counts. Saved-view labels
  continue to expose state/type/sort context, and browser coverage verifies
  those labels.
- **Revision flow coverage.** Browser coverage now verifies that a requested
  revision can be opened from the inbox, resubmitted, and shown as another
  timeline row.
- **Raw-ID audit and docs.** The raw-ID audit, readiness review, demo parity,
  WASM spike, status, bugs, and next-task queue were refreshed.
- **Verification.** Passed: `go test ./...`; focused Go tests for
  `internal/http` and `internal/wasmdemo`; `deno task check:ts`;
  `deno task lint`; `deno task test`; `deno task check:policy`;
  `deno fmt --check deno.json tools tests`; `make check-contracts`;
  `go tool deadcode -test ./...`;
  `ELM_BIN=/opt/homebrew/bin/elm deno task
  frontend:build`; and Playwright
  demo/mobile specs. DB-backed Playwright screens and local DB checks could not
  complete because Docker/Postgres was not reachable in the local environment.

`task/db-admin-wasm-parity-hardening` hardened the PR 89 admin/privacy/
moderation work with database-backed checks, integration tests, browser
coverage, parity coverage, and the next WASM demo step:

- **Database-backed validation.** Local Postgres migrations, tagged integration
  tests, and HTTP E2E checks passed through `tools/run_db_checks.sh`.
- **Platform admin lifecycle.** Revoked persisted platform admins are no longer
  authorized by `IsAdmin`. Integration coverage now verifies bootstrap admin
  protection, grant/list/revoke behavior, revoked state persistence, and
  post-revoke authorization denial.
- **Moderation and privacy persistence.** Integration coverage now verifies
  moderation triage state transitions, invalid triage rejection, privacy
  retention runs, sensitive-field redaction persistence, retention-run rows, and
  sensitive-field access audit events.
- **Admin browser coverage.** Focused Playwright demo coverage now exercises
  platform-admin grants/revokes, privacy retention execution, moderation report
  subject links, triage resolution, and moderation state filtering.
- **WASM spike.** `internal/wasmdemo` gained a no-fallback moderation-triage
  request handler over explicit browser storage and explicit clock boundaries.
  Missing storage, missing clocks, unsupported routes, invalid methods, invalid
  bodies, and invalid states are rejected explicitly.
- **Parity and contracts.** Shared scenario parity now checks admin audit event
  shapes for privacy retention, platform-admin grant/revoke, and moderation
  triage. API, readiness, raw-ID audit, demo parity, and WASM spike docs were
  refreshed.
- **Verification.** Passed: `go test ./...`; `deno task check:ts`;
  `deno task lint`; `deno task test`; `deno task check:policy`;
  `deno fmt --check deno.json tools tests`; `make check-contracts`;
  `go tool deadcode -test ./...`; local `tools/run_db_checks.sh` against
  Postgres; and focused Playwright `tests/playwright/demo.spec.ts`.

`task/admin-moderation-retention-wasm` added admin moderation, retention, and
WASM storage work:

- **Platform admin configuration.** Bootstrap admins still come from
  `SHARECROP_ADMIN_USER_IDS`. Admin-granted platform admins are persisted,
  listed, and revoked by lifecycle state instead of row deletion.
- **Moderation triage.** Moderation reports now carry triage state, resolution
  notes, updater metadata, state filtering, and direct subject links where a
  browser route exists. Platform admins can reopen, resolve, or dismiss reports.
- **Privacy retention and access events.** Platform admins can run
  delete-on-request sensitive-field retention. The Postgres store records
  retention runs, per-field redaction events, and sensitive-field access events
  for authorized submission-list/profile reads.
- **Admin UI and raw-ID fixes.** The Admin page gained selector-backed platform
  admin grants, revoke controls, retention execution, moderation filters, direct
  subject links, and triage controls. Blank select options now submit explicit
  empty values instead of placeholder text.
- **Contracts, parity, and demo.** Generated Elm contracts, HTTP wire-shape
  fixtures, shared scenario parity, and `site/demo/backend.js` now cover
  platform-admin grant/revoke, privacy retention, and moderation triage shapes.
- **WASM spike.** `internal/wasmdemo` gained an explicit moderation-triage
  browser storage boundary. Missing records, invalid keys, invalid states, and
  storage failures are rejected explicitly; no fallback store is selected.
- **Verification.** Passed: `go test ./...`; `deno task check:policy`;
  `deno task check:ts`; `deno task lint`; `deno task test`;
  `ELM_BIN=/opt/homebrew/bin/elm deno task frontend:build`;
  `go tool deadcode
  -test ./...`; focused Playwright
  `tests/playwright/demo.spec.ts`; and local admin desktop/mobile screenshot
  overflow checks. Tagged integration tests were attempted but require
  `DATABASE_URL`.

`task/moderation-parity-contract-wasm` added moderation foundations, parity
coverage, contract coverage, and a bounded WASM adapter spike:

- **Moderation workflow foundation.** Authenticated users can report tasks from
  task detail. Reports are persisted as `moderation_report_created` audit events
  and listed for platform admins through the Admin moderation panel and
  `/api/admin/moderation/reports`.
- **Audit event echoing.** Audit record results now carry the exact recorded
  event, so audit-backed workflows can return the created record without
  reloading a latest matching event.
- **Contracts and parity.** Generated Moderation Elm contracts, HTTP wire-shape
  fixtures, backendless demo behavior, shared scenario parity, and focused
  Playwright demo coverage were added for moderation report/admin-list/audit
  shape.
- **Raw-ID audit.** The current browser raw-ID audit is documented in
  [docs/raw_id_browser_flow_audit.md](./docs/raw_id_browser_flow_audit.md). No
  confirmed high-traffic user-entered raw-ID flow remains listed; protocol,
  route, audit, metadata, and API/MCP example IDs remain visible.
- **WASM spike.** `internal/wasmdemo` now classifies the privacy/moderation
  route pairs through explicit request-adapter results. Unsupported routes fail
  explicitly. No fallback stores or replacement demo backend were added.
- **Demo/readiness docs.** `docs/wasm_demo_backend_spike.md` was refreshed for
  the adapter spike and current adoption gates.
- **Verification.** Passed: `go test ./...`; focused Go tests for
  audit/db/http/wasmdemo; `make check-contracts`; `make check-format`;
  `deno
  check tools/*.ts tests/**/*.ts`; `deno lint tools tests`;
  `deno test
  --allow-read tests/deno`;
  `deno run --allow-read tools/check_policy.ts`;
  `ELM_BIN=/opt/homebrew/bin/elm deno task frontend:build`;
  `go tool deadcode
  -test ./...`;
  `deno run -A npm:jscpd@5.0.11 site/demo internal cmd tools
  web/elm/src tests`;
  tagged integration tests with local `DATABASE_URL`; tagged HTTP E2E tests with
  local `DATABASE_URL` and `SHARECROP_ACCESS_TOKEN_SECRET`;
  `GOOS=js GOARCH=wasm go test -c` for `./internal/wasmdemo`; and focused
  Playwright `tests/playwright/demo.spec.ts`. Focused task moderation/admin
  moderation screenshots were inspected.

`task/privacy-ops-demo-wasm-parity` deepened privacy/operator handling and demo
parity:

- **Privacy lifecycle.** Privacy request responses now include timestamps and
  redacted-field counts. Data-export resolution stores a JSON document with
  account, submission, sensitive-field, notification, ledger, and privacy
  request data. Sensitive-field deletion resolution marks delete-on-request
  sensitive-field metadata as redacted, stores affected counts, and records
  per-field redaction events.
- **Admin UI.** The browser Admin page lists privacy requests, accepts a
  resolution note, resolves queued requests, and shows export JSON and redaction
  counts.
- **Sensitive-field visibility.** Submission sensitive-field responses include
  lifecycle state and redaction time, and the browser shows those values in
  submission privacy summaries.
- **Contracts and parity.** Generated Elm contracts, HTTP wire-shape fixtures,
  shared scenario parity, backendless demo behavior, and Deno demo tests were
  updated for privacy resolution and redaction effects.
- **Demo fixes.** The frontend build now copies generated CSS into the static
  demo, preventing stale demo styling. Shared code blocks wrap long JSON/token
  lines, and a focused screenshot confirmed the admin privacy export no longer
  causes horizontal page overflow.
- **Queue polish.** Saved queue-view chips now include their filter/type/sort
  context.
- **WASM investigation.** Go `js/wasm` compile-only checks passed for
  representative packages and the main command. The documented blocker remains
  explicit browser storage/request adapters, deterministic reset, startup
  measurement, and a JS/WASM scenario runner. No fallback stores were added.
- **Docs.** API, readiness, demo semantic parity, WASM spike, and continuity
  docs were refreshed. Hard deletes remained prohibited.
- **Verification.** Passed: `go test ./...`;
  `deno check tools/*.ts
  tests/**/*.ts`; `deno lint tools tests`;
  `deno run --allow-read
  tools/check_policy.ts`;
  `deno test --allow-read tests/deno`; `make
  check-format`;
  `ELM_BIN=/opt/homebrew/bin/elm deno task frontend:build`; `go vet ./...`;
  `go tool deadcode -test ./...`;
  `deno run -A
  npm:jscpd@5.0.11 site/demo internal cmd tools web/elm/src tests`;
  CI-style local `make db-checks`; and focused Playwright
  `tests/playwright/demo.spec.ts`.

`task/persisted-ops-privacy-lifecycle` added persisted operations and privacy
lifecycle work:

- **Saved queue views.** Team work and organization task saved views are now
  persisted in Postgres and mirrored by the backendless demo.
- **Organization operations.** Organization detail pages now include
  organization ledger rows and org-scoped audit rows in the operations
  dashboard.
- **Privacy lifecycle.** Privacy requests are persisted, listable by requester
  and platform admin, and resolvable by platform admins. Resolution stores basic
  data-export JSON or marks delete-on-request sensitive-field metadata as
  redacted without removing core rows.
- **Audit.** Organization create/member actions now also write
  organization-subject audit rows for org-scoped panels.
- **Team assignees.** Standalone teams are valid task assignees; reservations
  require team membership and submission eligibility recognizes active team
  reservations.
- **MCP.** Persisted MCP SSE subscribers poll the replay table for cross-process
  fan-out groundwork.
- **Contracts and parity.** Generated Elm contracts, backendless demo routes,
  HTTP fixture coverage, scenario/demo tests, and continuity docs were updated.
  Hard deletes remained prohibited.
- **Verification.** Passed: `go test ./...`;
  `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build go vet ./...`;
  `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build go tool deadcode
  -test ./...`;
  `make check-format`; `deno run --allow-read tools/check_policy.ts`;
  `deno check tools/*.ts tests/**/*.ts`; `deno lint tools tests`;
  `deno test --allow-read tests/deno`; and
  `ELM_BIN=/opt/homebrew/bin/elm deno task frontend:build`.
- **Skipped local browser run.** Focused Playwright demo/mobile/screens coverage
  could not run locally because sandboxed port binding failed and escalation was
  blocked by the approval system usage limit.

`task/org-ops-queues-privacy` combined saved queue views, organization
operations, revision timeline polish, audited privacy requests, contracts,
parity, demo behavior, browser coverage, and docs:

- **Saved queue views.** Team work and organization task queues now let users
  save and reapply in-session query/filter/type/sort combinations.
- **Organization operations.** Organization detail pages show loaded balance,
  team, active/inactive member, collectible, and task-state counts.
- **Revision timeline.** Worker submission pages now include a revision timeline
  alongside the revision inbox and submission history.
- **Privacy requests.** Authenticated users can create audited privacy requests
  for data export or sensitive-field deletion. Requests are queued audit
  records; export generation, deletion, redaction, and retention jobs remain
  explicit future work.
- **Contracts and parity.** Added generated Privacy Elm contracts, HTTP
  request/response fixture tests, handler tests, backendless demo parity, shared
  scenario privacy request/audit assertions, and Playwright coverage for the new
  browser controls.
- **Docs.** Updated API reference, deletion semantics, operations runbook,
  readiness review, status, bugs, and next-task queue.
- **Verification.** Passed: `go test ./...`;
  `deno check tools/*.ts tests/**/*.ts`; `deno lint tools tests`;
  `deno fmt --check deno.json tools tests`;
  `deno run --allow-read tools/check_policy.ts`;
  `deno test --allow-read tests/deno`; `make check-format`;
  `make check-contracts`;
  `ELM_BIN=/opt/homebrew/bin/elm deno task
  frontend:build`;
  `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build
  go vet ./...`;
  `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build go
  tool deadcode -test ./...`;
  `deno run -A npm:jscpd@5.0.11 site/demo internal cmd tools web/elm/src
  tests`;
  and focused Playwright demo/mobile/screens tests with local-server escalation.

`task/queue-revisions-ops-privacy` combined queue tooling, revision-history
polish, admin inspectability, parity, contracts, docs, and privacy groundwork:

- **Queue tooling.** Task list APIs now accept `task_type` and `sort` filters in
  addition to existing scope/state/search/pagination filters. Team work,
  organization tasks, and requester task lists expose browser controls for task
  type and sort.
- **Revision and privacy views.** Worker submission history rows show response
  bodies, validation errors, review notes, sensitive-field metadata, and the
  existing revision shortcut for requested changes.
- **Admin inspectability.** Admin audit events can be filtered by action,
  subject kind, and subject ID through HTTP and browser controls.
- **Contract and parity coverage.** Submission response contracts now include
  `sensitive_fields`; fixture tests, HTTP E2E helpers, shared scenario parity,
  and the backendless demo were updated for queue type/sort and sensitive-field
  response metadata.
- **Docs.** API, operations, readiness, demo-parity, and continuity docs were
  updated. Export/delete request flows and sensitive-field deletion jobs remain
  deferred until their lifecycle design is written.

`task/readiness-dashboard-docs-parity` combined post-PR80 cleanup,
dashboard/navigation polish, discussion notifications, docs, and demo parity:

- **Continuity and readiness cleanup.** Updated
  status/bugs/next/readiness/user-story docs after PR 80 and confirmed PR 80 CI
  passed with `db-checks` and Playwright.
- **Team work dashboard.** Team detail pages now split loaded team work into
  review queue, ready-for-team work, and assigned-to-team work sections.
- **Submission discussion notifications.** Submission comments now notify the
  other side of the private thread. Inbox rows link to the task when
  notification metadata includes `task_id`.
- **Demo parity.** The backendless demo enforces submitter/reviewer access for
  submission comment threads and emits `submission_commented` notifications.
  Shared scenario parity now checks that notification path.
- **List/navigation polish.** Browser task and discovery lists have explicit
  previous/next pagination controls backed by the existing `limit`/`offset` API.
- **Docs.** Added repository HTTP API and MCP references plus an agent-side
  scheduling recipe, and linked them from README and hosted docs.
- **Verification.** Passed: `go test ./...`;
  `deno test --allow-read tests/deno`; `deno check tools/*.ts tests/**/*.ts`;
  `deno lint tools tests`; `make check-contracts`; `make check-format`;
  `deno run --allow-read tools/check_policy.ts`; `go vet ./...`;
  `go tool deadcode -test ./...`;
  `deno run -A npm:jscpd@5.0.11 site/demo internal cmd tools web/elm/src tests`;
  `ELM_BIN=/opt/homebrew/bin/elm deno task frontend:build`. Focused local
  Playwright reached the app server but could not complete because PostgreSQL on
  `localhost:15432` was not reachable.

`task/parity-contract-discussion-polish` combined post-merge continuity cleanup,
parity growth, contract coverage, demo semantics, and submission discussion
polish:

- **Continuity and verification.** Updated status/readiness/user-story docs
  after PR 79, removed stale raw-ID and collectible notes, and confirmed PR 79
  CI passed with `db-checks`.
- **Shared scenario parity.** Added a multi-actor organization reviewer path:
  the scenario provisions a reviewer by email, creates and organization-funds an
  organization-owned task, has a separate worker submit, and accepts the
  submission as the organization reviewer.
- **Backendless demo parity.** The demo now checks task-owner or organization
  reviewer permission before accept/reject/request-changes review actions, and
  newly created demo organizations receive an organization balance for funding
  parity.
- **Contract coverage.** Submission comment creation now has its own request
  type and HTTP fixture instead of reusing the series-comment request shape.
- **Discussion UX.** After a worker submits or a reviewer action succeeds, the
  browser opens that submission's discussion thread and labels the active
  discussion as open. Playwright assertions cover both flows.
- **Verification.** Passed: `go test ./...`;
  `deno test --allow-read tests/deno`; `deno check tools/*.ts tests/**/*.ts`;
  `deno lint tools tests`; `deno run --allow-read tools/check_policy.ts`;
  `make check-contracts`; `make check-format`; `go vet ./...`;
  `go tool deadcode -test ./...`;
  `deno run -A npm:jscpd@5.0.11 site/demo internal cmd tools web/elm/src tests`;
  `ELM_BIN=/opt/homebrew/bin/elm deno task frontend:build`. A focused Playwright
  run was not completed locally because the sandbox blocked local server binding
  and escalation was unavailable after the environment usage limit was reached.

`task/runtime-parity-reward-hardening` combined runtime verification ergonomics,
scenario parity, reward transaction hardening, organization-scoped collectibles,
deletion semantics, contract fixtures, and deployed routing checks:

- **Runtime verification.** Added `tools/run_db_checks.sh` and `make db-checks`,
  and replaced duplicate PR database jobs with one `db-checks` job that runs
  migrations, integration tests, and HTTP E2E tests against Postgres.
- **Deployed Pages routing.** The Pages workflow now installs Deno and runs
  `deno task check:pages-routing` against the deployment URL after GitHub Pages
  deploys.
- **Collectible tip transaction hardening.** Accepting a submission now settles
  credit payouts, credit tips, collectible payouts, and collectible tips inside
  `LedgerStore.AcceptSubmission`. The HTTP accept handler validates the
  collectible id up front and no longer performs a post-accept `GiftCollectible`
  call.
- **Organization-scoped collectibles.** Collectibles now carry
  `organization_id`; organization-owned mints default their scope to the owning
  organization. `transferable_within_organization` tips are allowed only when
  the collectible has a scope and both sender and worker are active members of
  that organization.
- **Contracts and demo parity.** Collectible responses and award requests
  include `organization_id`; generated Elm contracts and compiled bundles were
  refreshed. The backendless demo mirrors the new collectible shape and
  within-org tip checks. The shared scenario now covers collectible response
  shape and create-time collectible reward refund.
- **Deletion semantics.** Added
  [docs/deletion_semantics.md](./docs/deletion_semantics.md), defining current
  deactivation/state/redaction behavior and the rules for lifecycle and
  redaction workflows.
- **Verification.** Passed: `go test ./...`;
  `go test ./internal/http ./internal/db ./internal/assets ./internal/ledger`;
  `deno check tools/*.ts tests/**/*.ts`; `deno lint tools tests`;
  `deno run --allow-read tools/check_policy.ts`;
  `deno test --allow-read tests/deno`; `make check-format`; `go vet ./...`;
  `go tool deadcode -test ./...`;
  `deno run -A npm:jscpd@5.0.11 site/demo internal cmd tools web/elm/src tests`;
  `ELM_BIN=/opt/homebrew/bin/elm deno task frontend:build`. Local
  `make db-checks` stopped at `DATABASE_URL is required`.

`task/multi-actor-parity-demo-contracts` combined multi-actor parity, demo actor
semantics, fixture expansion, the WASM decision, and the remaining series
selector cleanup:

- **Multi-actor shared scenario parity.** The shared scenario runner now
  supports actor-specific clients. It registers distinct owner and worker
  actors, creates an approval-required funded task, requests and approves a
  reservation, submits as the worker, accepts as the owner with credit payout
  and tip, checks the worker balance, and verifies owner/worker notifications.
- **Backendless demo actor semantics.** `site/demo/backend.js` now maps local
  demo bearer tokens to users, passes Authorization headers through the XHR shim
  and Deno test adapter, and rejects protected routes with missing or unknown
  tokens. Demo task creation, reservations, submissions, review, balances,
  collectibles, notifications, org/team creation, comments, and series mutations
  use the resolved actor where relevant.
- **Contract fixtures.** Added HTTP wire-shape fixtures for organization and
  member wrappers, team-member request, task-series list response, collectible
  response, and collectibles response.
- **Selector replacement.** The series add-task control now uses the existing
  task selector populated from loaded user tasks instead of a raw task-ID text
  field. The focused Playwright series flow was updated to use the selector.
- **WASM decision.** The WASM demo-backend spike now records the current
  decision to keep `site/demo/backend.js` until explicit browser storage
  adapters can satisfy the adoption gates without fallbacks.
- **Verification.** Passed: `go test ./...`; `go test ./internal/http`;
  `deno check tools/*.ts tests/**/*.ts`; `deno lint tools tests`;
  `deno run --allow-read tools/check_policy.ts`;
  `deno test --allow-read tests/deno`;
  `ELM_BIN=/opt/homebrew/bin/elm deno task frontend:build`; focused Playwright
  test for the first-class task-series management flow. Manual screenshot review
  passed for the series add-task selector on the backendless demo series detail
  page.

`task/expand-parity-routing-fixtures-selectors` expanded the parity and selector
work after PR #76:

- **Shared scenario parity.** The shared scenario now covers admin operations,
  account-token issue shape, collectible catalog/mint/transfer, selector
  pagination/query, organization/team/task/task-comment creation, submission
  creation/listing/comments, and notification read shape. The real-API runner
  documentation now states that an explicit platform-admin token is required for
  the full scenario.
- **Selector replacements.** Admin default-collectible award recipients now use
  user/team/organization selectors instead of raw recipient IDs where directory
  data exists. Collectible transfer uses the user selector instead of a raw
  user-ID field. Demo Playwright coverage was updated to exercise the selector
  path.
- **Contract fixtures.** Added wire-shape fixtures for health/error/empty
  responses, ledger lists, teams/tasks/reservations/submissions wrappers, full
  task response, agent credential request/created/list responses, and user
  profile response.
- **Deployed Pages verification.** Ran
  `deno task check:pages-routing -- --origin https://e6qu.github.io/sharecrop`;
  the deployed root/docs/demo entry paths and demo assets passed.
- **Verification.** Passed: `go test ./...`; `go test ./internal/http`;
  `deno check tools/*.ts tests/**/*.ts`; `deno lint tools tests`;
  `deno run --allow-read tools/check_policy.ts`;
  `deno test --allow-read tests/deno`;
  `ELM_BIN=/opt/homebrew/bin/elm deno task frontend:build`. Manual screenshot
  review passed for the default-collectible award selectors and collectible
  transfer selectors on desktop/mobile demo viewports.

`task/scenario-parity-selectors-contracts` combined scenario parity, selector
pagination/typeahead, fixture expansion, deployed routing checks, and a WASM
demo-backend spike:

- **Shared scenario parity.** Added `tests/scenario_parity/scenario.ts`, a Deno
  test that runs the shared scenario against `site/demo/backend.js`, and
  `tools/run_scenario_parity.ts` for running the same scenario against a real
  API with explicit `--origin` and `--token`. The first scenario covers auth
  refresh, user/org/team selector pagination/query behavior, organization/team
  creation, task creation, task detail, and task comments.
- **Selector pagination/typeahead.** Users, organizations, standalone teams, and
  organization teams now use selector requests with `query`, `limit`, and
  `offset`. The Elm create/reservation/funding/award selectors gained query
  boxes, Search, Previous, Next, and offset display. Query input performs
  typeahead fetches and dependent organization-team state resets when the
  selected organization changes.
- **Backend and demo selector parity.** The Go org service/store and HTTP
  handlers accept selector queries for organizations and teams.
  `site/demo/backend.js` mirrors selector query/pagination for users,
  organizations, standalone teams, and organization teams, and malformed
  selector params fail visibly instead of silently defaulting.
- **Contract fixtures.** Expanded HTTP wire-shape tests for request/command
  contracts and newer response surfaces: auth/account token/password/profile,
  organization/member/team/task/funding/reservation/submission/review requests,
  series requests/detail, task comments, team detail, collectible
  requests/catalog, and account-token responses.
- **Deployment checks and WASM spike.** Added `tools/check_pages_routing.ts` and
  `deno task check:pages-routing` for post-deploy GitHub Pages
  root/docs/demo/asset checks. Updated
  [docs/demo_semantic_parity.md](./docs/demo_semantic_parity.md) and added
  [docs/wasm_demo_backend_spike.md](./docs/wasm_demo_backend_spike.md), which
  requires explicit browser storage adapters and no fallback behavior before a
  WASM demo backend can replace the JS fake.
- **Verification.** Passed: `go test ./...`;
  `go test ./internal/org ./internal/db ./internal/http`;
  `deno check tools/*.ts tests/**/*.ts`; `deno lint tools tests`;
  `deno run --allow-read tools/check_policy.ts`;
  `deno test --allow-read tests/deno`;
  `ELM_BIN=/opt/homebrew/bin/elm deno task frontend:build`;
  `deno task check:pages-routing -- --origin http://127.0.0.1:18082` against a
  local `site/` server. Manual screenshot review of the create-task selector
  controls passed on desktop and mobile demo viewports.

`task/runtime-notifications-demo-parity` combined runtime coverage,
notifications, MCP replay persistence, contract fixtures, and demo parity
planning:

- **Notification inbox.** Added `NotificationID`, a notification domain service,
  memory store, Postgres store, migrations, runtime wiring, and HTTP routes for
  `GET /api/notifications` and `POST /api/notifications/{notification_id}/read`.
  Submission creation notifies the task creator, and
  accept/request-changes/reject notifies the submitter. Self-notifications are
  skipped as an explicit domain outcome.
- **Inbox UI and contracts.** Added generated `Sharecrop.Generated.Notification`
  Elm contracts, notification HTTP wire-shape fixtures, browser
  state/messages/API calls, an Inbox nav entry, the Inbox page, unread/read
  badges, metadata rendering, and mark-read behavior.
- **Backendless demo parity.** `site/demo/backend.js` now has seeded
  notifications, notification routes, mark-read behavior, and notification
  creation on submission/review events. Deno demo tests validate route parity
  and the notification response shape. The demo bundle was rebuilt from the
  current Elm app.
- **MCP replay persistence.** Added `mcp_http_events` and Postgres persistence
  for MCP HTTP replay events. Session identity, active counts, close state, and
  replay rows are persisted; live SSE subscriber channels remain process-local.
- **Runtime-store coverage.** Added integration tests for notification
  lifecycle, audit event listing, persisted MCP HTTP session counts/replay
  events, and Postgres rate-limit buckets. These tests require `DATABASE_URL`
  and `SHARECROP_MIGRATIONS_DIR`.
- **Demo semantic-parity planning.** Added
  [docs/demo_semantic_parity.md](./docs/demo_semantic_parity.md), recommending
  shared scenario parity tests before a Go/WASM demo-backend spike.
- **Verification.** Passed: `go test ./...`; `make check-format check-ts lint`;
  `make check-contracts`;
  `ELM_BIN=/opt/homebrew/bin/elm deno task frontend:build`; `deno task test`;
  `make check-policy vet check-dead-code`; `make check-copy-paste`; targeted
  Playwright demo/mobile specs. Manual screenshot review of `#/inbox`
  desktop/mobile found no layout or overflow issue. Integration tests were
  present but local execution stopped at `DATABASE_URL is required`.

`task/account-directories-rewards-playwright` combined four follow-up tracks
into one branch by explicit request:

- **Account lifecycle.** Added account status and account-token storage
  migrations; login rejects deactivated accounts. The auth service, DB store,
  and HTTP API now support browser guest entry, email-verification token
  issue/confirm, password-reset token issue/confirm, password change, profile
  email update, and account deactivation with session/token revocation. The
  browser exposes reset controls on the signed-out page and settings controls on
  the user's profile page.
- **Directories and selectors.** Added an authenticated user directory endpoint
  (`GET /api/users`) backed by the auth store. The Elm client loads users and
  standalone teams after auth / task-create entry and uses selectors for task
  visibility user/team scopes instead of raw inputs where data is loaded.
- **Reward setup.** Task creation accepts selected collectible IDs for
  collectible and bundle rewards. The create handler escrows those collectibles
  immediately after creating the task, refetches the task so the response
  reflects the held collectible count, and preserves the existing post-create
  funding flow for bundle tasks with no selected IDs. The task store continues
  to derive collectible counts from held/released reward rows.
- **Browser coverage.** Added real-app Playwright coverage for guest/account
  lifecycle controls and selector-backed task creation with create-time
  collectible escrow. Updated the existing standalone-team visibility test to
  select from the new team picker.
- **HTTP coverage.** Added account lifecycle HTTP e2e coverage for directory
  lookup, email verification, password reset, password change, profile email
  update, deactivation, and post-deactivation login rejection. Added HTTP e2e
  coverage for create-time collectible escrow and multi-collectible counts.
- **Verification.** Passed: `go test ./...`;
  `go test -tags http_e2e ./tests/http_e2e` with local Postgres;
  `make check-contracts`; `make check-format`; `make check-policy`;
  `make check-ts`; `make lint`; `make test-deno`; `make check-dead-code`;
  `make vet`; `make build`; and `deno task e2e:ui` (36 Playwright tests).
- **Deferred.** Hard account deletion, real email delivery for account tokens,
  and paginated/typeahead browser directory search remain queued.

`task/org-worker-selectors-rewards-docs` combined five follow-up tracks into one
branch by explicit request:

- **Organization role management and reviewer parity.** Organization member
  provisioning gained a role picker. Members can have roles updated or be
  deactivated through new API/store/service paths and browser controls.
  Organization reviewers can review organization-owned task submissions even
  when they did not create the task; task responses now carry `reviewer_action`.
- **Worker submission UX.** Task detail loads the viewer's own submissions and
  renders a task-local "My submissions" panel with state, review notes,
  validation errors, response body, and submission comments. Reviewer-only
  submission-list failures no longer appear as worker submit errors.
- **Selectors.** Organization-team reservation uses organization/team selectors
  and loads teams after selecting an organization. Organization funding,
  organization visibility, and organization award-recipient flows use
  organization selectors. Raw user/team recipient fields remain where no
  searchable directory endpoint exists.
- **Reward creation.** The create-task form has an explicit reward-kind chooser
  for no reward, credits, collectible, and bundle rewards. Credit and bundle
  rewards collect a credit amount; collectible and bundle creation use the
  current HTTP parser's fixed one-collectible count.
- **Docs and demo parity.** The static demo backend includes `reviewer_action`,
  organization role/deactivate routes, and regenerated demo bundles. The landing
  page links to real docs, and readiness/user-story docs no longer call `/docs/`
  a placeholder.
- **Tests and checks.** Passed: `go test ./...`,
  `deno test --allow-read tests/deno`, `deno check tools/*.ts tests/**/*.ts`,
  `deno lint tools tests`, `deno run --allow-read tools/check_policy.ts`, format
  checks, and
  `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build ELM_BIN=/opt/homebrew/bin/elm make frontend`.
  A focused Playwright demo smoke captured screenshots for reward selectors,
  funding selector, organization member controls, and worker "My submissions".
- **Skipped locally.** Database-backed HTTP E2E and full real-app Playwright
  were not run because `DATABASE_URL` is not set and local server binding is
  restricted in the sandbox. The new HTTP E2E test coverage is present for
  organization role update/deactivation.

`task/org-team-assignment` made organization-team assignment workable across the
main interfaces:

- **Domain and permissions.** Task reservation now has an organization-team path
  in addition to the existing user path. The organization service exposes a
  team-membership check that verifies the team is organization-owned, belongs to
  the requested organization, and includes the acting user.
- **Submission eligibility.** Non-open tasks now treat an active
  organization-team reservation as satisfying eligibility for users who belong
  to the reserved team.
- **HTTP and MCP.** `POST /api/tasks/{task_id}/reservations` still accepts an
  empty body for user reservations, and now accepts
  `{"assignee_kind":"organization_team","organization_id":"...","team_id":"..."}`
  for organization-team reservations. `sharecrop.reserve_task` accepts the same
  optional assignee fields.
- **Browser and demo.** The task detail reservation card shows organization/team
  ID inputs for `organization_team` tasks and posts those IDs when reserving or
  requesting approval. The demo backend accepts team reservations and checks
  seeded team membership; generated browser bundles were refreshed.
- **Tests.** Unit tests cover organization-team reservation success and
  non-member rejection. A new HTTP e2e test covers organization team creation,
  member reservation, outsider rejection, and member submission. Local
  verification passed for `go test ./...` and `make frontend`; the new
  database-backed HTTP e2e test could not run in this environment because
  `DATABASE_URL` is not set.

`task/application-completeness-review` documented a project readiness review:

- Added
  [docs/application_readiness_review.md](./docs/application_readiness_review.md),
  comparing the implemented backend, Elm UI, HTTP API, MCP surface, tests, and
  user stories against the product thesis.
- The review found that the registered requester/worker task loop is implemented
  and tested, but the application is not yet ready for ordinary production use.
  The highest-priority gaps are organization-team assignment, organization
  reviewer browser parity, worker submission status/discussion, coherent reward
  creation, account lifecycle, operations, and product/API/MCP docs.
- Updated [DO_NEXT.md](./DO_NEXT.md) with the review-driven priority queue.
- Corrected stale user-story and bug/test-gap notes: collectible review tips and
  browser organization-member listing exist; organization-team assignee
  selection exists but is not workable end to end; browser organization roles
  and worker submission UX remain partial.
- Verification: `make check-policy` and `make test-deno` passed. A docs-only
  `deno fmt --check` probe was not treated as required because Markdown is
  outside the repository's `make check-format` target and would reflow old
  continuity text unrelated to this branch.

`task/bundle-refund-ui-parity` corrected the bundle-refund UX and a stale BUGS
claim:

- Investigation found the "no one-shot bundle refund" risk recorded in BUGS was
  wrong: the credit `/refund` endpoint already calls
  `refundHeldCollectibleReward` inside its transaction, so it returns held
  credits AND collectibles together (covered by
  `TestBundleRefundReturnsCreditsAndCollectible`). The actual gap was UI-only —
  a bundle task rendered both "Refund credits" and "Refund collectible", and the
  collectible one 409'd on bundle.
- **Owner refund controls** now offer exactly one refund action per reward kind:
  credit → `/refund` ("Refund credits"), collectible → `/collectible-refund`
  ("Refund collectible"), bundle → `/refund` ("Refund reward", returns credits +
  collectible together). The dead bundle "Refund collectible" button is gone.
- **Demo parity.** `site/demo/backend.js` `/refund` now releases escrowed
  collectibles too (mirroring `refundHeldCollectibleReward`), so a demo bundle
  refund returns everything in one call; the cancel guard and collectible-refund
  routes already matched.
- **Tests.** A real-backend Playwright flow escrows credits + collectible on a
  bundle task, opens it, refunds via the "Refund reward" UI button, and asserts
  both balance restoration and the collectible returning to holdings. Stabilized
  `openTaskFromDiscovery` (network-idle + 15s balance wait) against intermittent
  shared-server load flakiness that had been timing out post-login navigation.
- Removed the stale BUGS entry.

`task/fix-cancel-escrow-guard` closed the active orphan-escrow-on-cancel bug
surfaced in the previous PR:

- **Cancel now rejects while escrow is held.** The task store's
  `ChangeTaskState` gained a `requireNoHeldEscrow` guard on the cancellation
  path: it counts held credits (`task_escrows.state = 'held'`) and held
  collectibles (`task_collectible_rewards.state = 'held'`) and returns a 409
  "refund the task's held escrow before cancelling" when either exists.
  Previously the `Cancel` state transition left held escrow stranded against a
  cancelled task with no return path. This is the documented "reject Cancel
  while escrow is held" option. http_e2e covers a funded task returning 409 on
  cancel and a subsequent successful refund. The browser already routes funded
  tasks to Refund, so the only behavioural change users see is the rare
  funded-draft Cancel now surfacing that 409 (with the Refund action alongside)
  instead of silently orphaning.
- **Demo parity.** The `site/demo/backend.js` cancel route now rejects when the
  task holds escrow (`escrow > 0` or any escrowed collectible), matching the
  real backend's precondition.
- **Playwright helper race fix.** `openTaskFromDiscovery` in `screens.spec.ts`
  intermittently timed out clicking `nav-discovery` because it ran before the
  post-login data load finished. It now waits for the balance to render before
  navigating; the helper and `loginViaUi` were retyped from ad-hoc structural
  types to the real Playwright `Page`.
- **Stale-doc cleanup.** Removed a BUGS entry claiming the demo deep-link 404s
  on GitHub Pages — that was already fixed by fragment (hash) routing in PR #61.
  Documented the residual bundle-refund gap (no one-shot refund for bundle
  rewards) as a known risk.

`task/ui-cancel-collectible-tip` exposed the task-lifecycle and review actions
the HTTP API already supported but the browser lacked:

- **Cancel a task.** The owner controls gained a Cancel button. It is offered
  for draft tasks (any reward) and for open no-reward tasks. Reward-bearing OPEN
  tasks are ended via Refund instead: the backend's `Cancel` (task state
  transition) does not return held escrow, while `RefundTask` does — so gating
  Cancel away from funded tasks avoids orphaning escrow. (That
  Cancel-of-a-funded-task gap is recorded in BUGS as a backend-level risk.)
- **Collectible tip on accept.** The review form gained a "Tip a collectible"
  select populated from the requester's holdings; the selected collectible is
  sent as `tip_collectible_id`, which the backend's accept handler gifts to the
  worker via `GiftCollectible`. The selection is reset alongside the rest of the
  review form after each review action (extending the existing review-form
  reset).
- **Refund a collectible reward.** Owner controls offer "Refund collectible" for
  draft/open collectible- or bundle-reward tasks, wired to
  `POST /api/tasks/{id}/collectible-refund`.
- **Owner-controls gating rewrite.** The buttons are now built from a single
  `List.filterMap identity` over state/reward conditions, replacing the prior
  exclusive if/else (which showed only ever one of Open/Refund). Open is
  draft-only; Cancel/Refund-credits/Refund-collectible appear contextually.
- **Demo parity.** `site/demo/backend.js` implements
  `POST /api/tasks/:id/collectible-refund` (returns escrowed collectibles to the
  requester, resets the task reward kind) and honors `tip_collectible_id` on
  accept (gifts the collectible, sets
  `payout_kind`/`collectible_ids`/`worker_user_id`).
- **Tests.** Two real-backend Playwright flows: an owner cancels a no-reward
  task (asserts Cancel visible, Refund hidden, "cancelled" message); an owner
  tips a transferable collectible on accept (asserts the tipped collectible
  leaves their holdings). Backend collectible-tip and collectible-refund
  semantics were already covered by http_e2e
  (`TestCollectibleTipTransfersOnAccept`,
  `TestCollectibleRewardRefundReturnsToOwner`,
  `TestBundleRefundReturnsCreditsAndCollectible`).

`task/polish-bugfix-uiux-review` was a combined bug-sweep + UI/UX review pass
driven by three parallel review agents (Go backend, demo-vs-real drift, Elm
client), with the findings fixed in one branch:

- **Elm review-form state leak (HIGH).** The review form (note / partial credit
  / tip / ban) carried across task→task navigation, across discovery→detail, and
  from one submission to the next within a task — so rejecting submission A with
  ban=true silently re-banned submission B. Fixed by resetting the four fields
  in `enterPage` (TaskDetailPage), `DiscoveryViewClicked`, and the
  `ReviewActionReceived` Ok branch. Also added missing `enterPage` resets for
  `CollectiblesPage`, `CreateTaskPage`, and `FundingPage` (the same leak family:
  stale award/mint/create/fund messages and prefilled drafts reappeared on
  return).
- **Stale task detail after refund (HIGH).** `RefundTaskReceived` only refreshed
  the task list + ledger, leaving the detail card badge on open/funded next to a
  "Task refunded" note. It now refetches the detail (and
  reservations/submissions) via `refreshAfterAccept`.
- **Perpetual "Loading…" on bad deep-links (MED).** A forbidden or failed task
  detail wrote its error to `submitMessage`, which is hidden when
  `detail == Nothing`, so the page hung on "Loading task…" forever. Added a
  `detailError : Maybe String` field; the detail card now renders the error.
- **Token-mint errors surfaced (MED).** `TaskTokenMinted`/`UserTokenMinted`
  error branches returned `(model, Cmd.none)` — a no-op button. They now surface
  a friendly error.
- **Dead/no-op controls gated by state (MED).** Owner Open/Refund rendered for
  every task state (clicking them on an open/closed task was server-rejected);
  they now render only for states they can act on (Open: draft; Refund:
  draft/open). The worker submit form rendered on closed/cancelled/refunded
  tasks; it now renders only when the task is open. (Gating is on `state` only —
  `viewerAction` is viewer-independent in both the real backend and the demo, so
  it cannot express "this viewer has reserved".)
- **Go backend cleanup.** Deleted the dead `requireAdmin` helper (the single
  admin gate is inline in `collectibles.go`). `writeSeriesDetailStatus` now
  propagates a `ListSeriesComments` rejection through `writeDomainError` instead
  of swallowing it and returning an empty comment list. The receipt-status
  handler routes through `writeDomainError` (was a hardcoded 404). The `/mcp`
  body read uses `http.MaxBytesReader` so an oversized body returns 413 instead
  of being silently truncated into a misleading "not valid JSON-RPC" 400.
- **Demo fidelity (`site/demo/backend.js`).** Reject no longer closes the task
  (matches prod's `closeTask: false`) and now releases the rejected worker's
  reservation to `cancelled_by_requester`; reject/request-changes now require an
  open task; `payout_kind` reflects the actual payout (not the reward kind),
  `worker_user_id` is populated only when a payout matched, and request-changes
  reports `payout_kind: "none"`; refund returns the released `amount` (was 0);
  the ledger seed reconciles with the balance (1250, was off by 10); PATCH
  task-series no longer wipes the description when the field is omitted; awarded
  collectibles use `state: "awarded"` (was "minted").
- **Reviews done.** Go backend lens (authz/IDOR, ledger, input bounds, dead code
  — clean apart from the items fixed), demo drift lens (state machine, economy,
  enum/shape, seeds), Elm client lens (state leak, stale data, dead controls,
  decoder/error, a11y). A visual screenshot set was generated to
  `/tmp/sharecrop-review-screens` but the agent could not inspect images; the
  user should review those captures.
- **Deferred (recorded in BUGS/DO_NEXT):** Team/Series/User detail load-vs-error
  distinction (only TaskDetail was upgraded); demo `reservationChange`/reserve
  still skip the ownership + assignee-scope guards; `type_ "button"` on assorted
  secondary buttons and free-text-id → picker follow-ups.

Follow-up commit on the same branch tackled the deferred load-vs-error and
demo-guard items:

- **Detail load-vs-error extended.** TeamDetail, SeriesDetail, and UserProfile
  now carry their own `*Error : Maybe String` field and render the error message
  on a failed/forbidden fetch instead of hanging on "Loading…". The
  `SeriesDetailReceived` handler was rewritten to a case (the
  `seriesRenameTitleFor`/`seriesRenameDescriptionFor` helpers became dead and
  were removed).
- **Demo reservation guards.** `site/demo/backend.js` `reserve` now rejects
  non-user-scoped tasks ("this task does not accept user reservations"), and
  `reservationChange` requires the task requester (`created_by === ME`,
  else 403) and only transitions reservations in `requested`/`active` states
  (else 409) — matching the real backend's `changeReservationByRequester` +
  store guard.
- **Client-side validation.** The submit form rejects empty or non-JSON input
  before posting; the fund form rejects non-positive amounts.

`task/backlog-cleanup` cleared bounded backlog deferrals and applied a UI/UX +
QA boyscout review (one background review agent):

- **Admin-panel gating.** The auth response now carries a `role`
  ("admin"/"member", stamped from `SHARECROP_ADMIN_USER_IDS` in
  `writeAuthResponse`; contract field added as a string since the codebase bans
  `Bool` in contracts). The client stores `isAdmin` and **hides the "Admin:
  award" panel and the catalog Award buttons for non-admins** (the catalog stays
  browsable). The demo's auth role is `admin`, so the showcase keeps them.
- **Back-button regression (critical).** The task-detail Back button used a
  non-fragment href (`/tasks`), which after the hash-routing switch dumped users
  on Overview. Now `#/tasks` / `#/discovery`.
- **Go status codes.** `getTask` returned 403 for a _missing_ task — now
  `writeDomainError` (real 404). A sweep replaced hardcoded
  `writeError(w, http.Status…, reason.Description())` with
  `writeDomainError(w, reason)` across the
  organization/series/collectible/org-credit handlers, fixing contradictory
  siblings (one list endpoint 403, its twin 500) and wrong 400s so each
  rejection maps to its correct status. Validated by the http_e2e status
  assertions.
- **Dead/no-op controls.** The "Submit a response" form no longer renders when
  the task failed to load (was a live form posting to an unreadable task);
  review controls (note/payout/tip/ban) render only when there are submissions
  (was a full review form above "No submissions").
- **Demo user-submissions** endpoint returns the user's real submissions (was
  always `[]`).
- **Deferred (noted):** id-picker dropdowns for free-text scope ids;
  org-reviewer review controls in the browser (needs a `viewer_action` "manage"
  value); per-page loading-vs-error states (a forbidden deep-link still shows a
  perpetual "Loading…"); plus the large standalone initiatives for
  out-of-process Postgres session/rate-limiter storage and anonymous-worker
  identity. Crypto reward metadata is out of scope. The QA review found **no
  WCAG contrast failures** and confirmed full demo route/decoder parity.

`task/demo-fidelity` minimized the in-browser demo's "fakes" so
`site/demo/backend.js` behaves like the real Go backend. A specialized agent
compared all ~69 demo routes to the Go handlers/domain; the high-impact
divergences are fixed:

- **Review state machine.** accept/reject/request-changes now act only on
  `submitted` work (409 otherwise); accept additionally requires an open task
  and rejects when a submission was already accepted — so double-clicking Accept
  or reviewing an `invalid`/seeded submission no longer replays payouts and
  drifts the balance.
- **Lifecycle + economy guards** matching prod's 409s: open (draft only), cancel
  (draft/open), unpublish (open only); refund (draft/open + must have escrow,
  and it returns to the funding wallet); funding (rejects re-funding an
  already-funded task — escrow is a single hold); reservation (open state,
  non-open policy, and the requester can't reserve their own task).
- **Create fidelity.** task-create honors the `owner` (so org-owned tasks store
  `owner_kind`/`owner_id`), the `visibility` scope id, the `assignee_scope`, and
  `reservation_expiry_hours` — previously all dropped, which made
  org-owned/org-scoped tasks look user-owned and miss their org page.
- **Series consistency.** add/reorder set `series_kind = "existing_series"` and
  remove resets to `"standalone"`; the seed values were aligned (was
  `"existing"`).
- **Seed nit.** the seeded "Golden Sickle" collectible is now
  `transferable_between_users`, matching its catalog template (was
  inconsistently non-transferable).
- **Intentional demo affordances kept** and documented in the file header:
  auto-login on `/api/auth/refresh`, the Reset button, and unvalidated tokens
  with a single seeded "you" (Mara).
- The 31-test Playwright suite still passes (no flow regressions). Deferred (low
  impact, noted): the user-submissions page stub and member/team provisioning
  showing a synthetic id rather than the typed email.

`task/demo-pages-routing` fixed the GitHub Pages demo URL/refresh problem
cleanly (no fallback), and hardened logout:

- **Fragment (hash) routing.** The demo runs at `/sharecrop/demo/` but the
  client built root-absolute paths, so click-navigation left the base and
  hard-refresh/deep-links 404'd (Pages bounced to the site root). Rather than
  add a 404.html SPA fallback + base-path threading (a "needless fallback" that
  hides bugs), the router now keeps the whole route in the URL **fragment**
  (`#/...`). The path stays a real file, so hard-refresh and deep-links work
  with **no 404.html, no base-path config, and no Go SPA catch-all**.
  `pageFromUrl` reads `url.fragment`; every internal href is `#/...`; the two
  path-building `Nav.pushUrl` calls became `#/...`.
- **Explicit NotFoundPage.** The router's `_ -> OverviewPage` catch-all silently
  turned bad links into Overview. It's now an explicit `NotFoundPage` (root `""`
  → Overview; unknown → NotFound), so dead links are visible instead of
  masquerading.
- **Demo-only Reset button.** A new **required** `demo : Bool` flag (no implicit
  default) — `web/static/index.html` passes `false`, `site/demo/index.html`
  passes `true`. When true, the nav shows a "Reset demo" button; a `reloadDemo`
  port reloads the page, which re-seeds the in-browser backend and auto-logs-in.
  Shown only in the demo.
- **Real logout revokes the session.** Logout previously only cleared the
  cookie; the refresh token stayed valid server-side. Now `auth.Service.Logout`
  → store `RevokeRefreshFamily` revokes the whole token family (mirroring the
  reuse-detection revoke), and the handler reads the cookie + revokes before
  clearing it. Re-login is not blocked. http_e2e confirms a post-logout refresh
  is rejected and a subsequent login succeeds.
- **Tests.** Deep-link Playwright navigations switched to `/#/...`; added hash
  hard-refresh + NotFound + reset presence/absence tests and the logout-revoke
  http_e2e test.
- **Deferred to the next PR (your broader ask):** a demo-fidelity QA pass to
  minimize the `site/demo/backend.js` fakes so the in-browser demo behaves as
  close to a real deployment as possible.

`task/uiux-journey-review` was a thorough UI/UX + user-journey review round (two
specialized subagents) with boyscout fixes, and it recorded a product decision:

- **Scheduling/recurrence descoped server-side** (decision 2026-06-25):
  recurring/scheduled task posting is a local-agent responsibility (a client
  cron/work-loop calling the existing MCP/API
  `create_task`/`open_task`/`fund_task`). No server
  scheduler/`task_schedules`/recurrence model. Recorded in `DO_NEXT.md` and the
  parity-roadmap memory so it isn't re-proposed.
- **No contrast failures** were found across the newest surfaces (collectibles
  gallery/award/trade, org/team holdings, submission comments, template/schema
  designer) — verified with computed WCAG ratios.
- **Bug fixes** (all from the review):
  - Re-funding a task was a silent no-op — `fundingRequestBody` hardcoded
    `idempotency_key = "fund:" ++ taskId`, so a second funding replayed. Now
    keyed per attempt (`fundNonce`), so adding escrow works while network
    retries stay idempotent.
  - Award feedback cross-contaminated: award-to-task and admin award-default
    shared `awardMessage`. Split into `awardMessage` / `awardDefaultMessage`.
  - `transferMessage`/`transferRecipientId` leaked across collectible detail
    pages — added a `CollectibleDetailPage` reset to `enterPage`.
  - The task-detail card stayed stale after accept/reject/request-changes —
    `refreshAfterAccept` now refetches the task detail + reservations.
  - Task-comment add now guards empty bodies and surfaces errors (was posting
    blank / swallowing failures), matching the submission/series threads.
  - The catalog Award and "Award to selected task" buttons are disabled until a
    recipient/task is chosen.
  - Exclusive chooser buttons
    (participation/visibility/assignee/owner/collectible-kind/policy/award-kind/state-filter)
    gained `aria-pressed`.
  - The admin-award 403 shows a friendly message plus an admin-only note on the
    panel.
  - Demo: removing a task from a series sets `series_id = ""` (not `null`),
    which previously blanked the strict-decoded task detail.
- **Deferred (flagged):** full `is_admin`-on-session gating to hide the admin
  panel for non-admins; org-reviewer review controls in the browser (today only
  the literal creator sees them); free-text id inputs → picker dropdowns;
  org/team holdings auto-refresh right after an award.

`task/submission-comments` added a private comment thread on each submission (PR
2 of the backlog sequence), mirroring the existing task-comment vertical end to
end:

- **Domain/store/HTTP:** migration `000022_submission_comments`;
  `core.SubmissionCommentID`;
  `submission.AddSubmissionComment`/`ListSubmissionComments` with a visibility
  rule that permits only the submission's author (worker) or the owner of its
  task (requester); `GET`/`POST /api/submissions/{id}/comments`.
- **MCP:** `sharecrop.add_submission_comment` (submissions:write) and
  `sharecrop.list_submission_comments` (submissions:read).
- **Contracts:** `SubmissionCommentResponse` (id, submission_id, author_user_id,
  body, created_at) + `SubmissionCommentsResponse`.
- **Client/demo:** each submission row on the task detail has a "Comments"
  toggle opening its thread (list + add box); the demo backend serves the routes
  and seeds a comment.
- **Tests:** submission domain unit, http_e2e (owner + submitter post/list;
  unrelated user → 403), Playwright (owner comments on a worker's submission).
- **Deferred:** a dedicated worker-side submission view in the client — the
  backend + MCP already let the worker participate; only the owner-side UI
  shipped here.

`task/admin-collectible-ownership` finished the two collectible follow-ups — a
real admin gate on awarding, and real org/team ownership (PR 1 of a sequence
burning down the follow-up + roadmap backlog):

- **Platform-admin role:** the server reads `SHARECROP_ADMIN_USER_IDS`
  (comma-separated user ids) into an admin set (`parseAdminUserIDs`) and a
  `requireAdmin` helper. `POST /api/collectibles/award` now returns 401
  (unauthenticated) / 403 (authenticated non-admin); only an admin can mint
  catalog copies. The demo award panel is unchanged (the demo has no real auth,
  so it stays the showcase).
- **Org/team ownership:** migration `000021` adds `owner_kind` to collectibles
  and drops the users foreign key on `owner_user_id` so it can hold any owner
  entity's uuid. `assets.Collectible` now has `OwnerKind string` +
  `OwnerID string`; the fund/tip/transfer flows require `OwnerKind == "user"`.
  Awarding accepts `recipient_kind` ∈ {user, team, organization}. New
  `GET /api/organizations/{id}/collectibles` and
  `GET /api/teams/{id}/collectibles` (service `ListByOwner`, store
  `ListCollectiblesByOwner`), surfaced as a "Collectibles" holdings section on
  the org and team detail pages. `CollectibleResponse` gained `owner_kind`
  (contracts regenerated); the demo seeds/award/mint set `owner_kind`.
- **Tests:** http_e2e covers the admin gate (non-admin 403), award to a user and
  to a team, the team holdings endpoint, and trade-back; assets unit +
  integration updated for the new owner fields; the store's list loop was
  extracted to a shared helper to satisfy copy-paste detection.
- Bootstrap note: the http_e2e admin test registers users on one server, then
  rebuilds with `SHARECROP_ADMIN_USER_IDS` set to the admin id (the shared test
  DB persists registrations).

`task/default-collectibles` added 25 hand-crafted pixel-art default collectibles
with an admin award flow and user-to-user trading, in a single PR:

- **25-item catalog** (`internal/assets/catalog.go`): a farm/harvest-themed set
  where `kind` doubles as rarity — 15 badges (common), 5 editions (rare), 5
  unique (legendary), all tradeable. Each carries an `art` slug.
- **Pixel sprites** (`web/elm/src/Sharecrop/Sprites.elm`): each of the 25 is a
  hand-authored CSS pixel-art icon (`pixel slug cell` → a grid of colored
  cells), themed to the arcade palette. Shown in the catalog gallery, on
  holdings rows, and large on the detail page.
- **Model/contracts:** collectibles gained an `art` field — migration `000020`,
  `assets.Collectible.Art`, `Mint(..., art)`, `CollectibleResponse.art`, plus
  generated `CollectibleCatalogEntry`/`CollectibleCatalogResponse` types.
- **Real backend endpoints:** `GET /api/collectibles/catalog`;
  `POST /api/collectibles/award` (mints a catalog copy owned by a user; team/org
  recipients are rejected with a demo-only note);
  `POST /api/collectibles/{id}/transfer` (user→user, reusing the policy-enforced
  `GiftCollectible`).
- **Demo (full showcase):** `backend.js` seeds the 25-entry catalog, implements
  award to user/team/org (owner_kind) and transfer; the Collectibles page has an
  admin award-recipient control and the gallery; awarding to yourself surfaces
  the copy in your holdings; trading moves it on. The transfer confirmation is
  rendered at the detail-card level so it persists after the traded item leaves
  your holdings.
- **Tests:** `assets` unit, `http_e2e` (catalog count, award, holdings,
  trade-back, team-award rejection), Playwright (`demo.spec` gallery + award +
  trade); the mobile overflow test now also covers the gallery.
- **Deferred (flagged in the PR):** real-DB org/team _ownership_ of collectibles
  (the real backend keeps user ownership; org/team awarding is demonstrated in
  the demo) and a real system-admin role (award is an ungated faucet for now).

`task/create-template-menu` reframed task creation around templates and applied
a batch of usability fixes from a specialized UI/UX review:

- Create-task: the "Task type" select became a **Template** menu — "Freeform (no
  template)" or a named template (Code review, Security review, Product review,
  UI/UX review, QA testing). Freeform shows the structured schema designer;
  selecting a template hides the designer, prefills the description + response
  schema, and shows a note explaining that. `CreateTaskTypeChanged` now clears
  `createSchemaFields` when applying a template (and resets to a freeform schema
  when switching back), which fixes the designer-vs-raw-JSON silent-clobber the
  review flagged.
- `createMessage` leak (review P0): the task-detail owner-controls card now has
  its own `taskActionMessage`, so "Created task X" no longer appears under owner
  controls and "Task opened/refunded" no longer appears under the create form.
- Reservation-expiry field (review P1): shown and validated only when
  participation is `reservation_required`/`approval_required` (shared
  `Labels.participationUsesReservation`), instead of always-on and
  always-blocking.
- Labels (review P2): `kindLabel` returns human ledger labels and the ledger
  renders signed, colored amounts; `collectibleKindLabel` returns
  "Unique"/"Edition"/"Badge"; a new `scopeLabel` names agent scopes, and the
  scope checkbox uses the shared styled checkbox via `onCheck`.
- Stale task detail (review P3): `enterPage` clears the previous task's
  detail/submissions/reservations/comments/token state for `TaskDetailPage`, so
  a task→task link no longer flashes the prior task's content.
- Generic reference-URL helper. Playwright covers the template menu (prefill +
  designer toggle); label/message assertions updated.
- Deferred from the review (noted, not done this PR): chooser buttons →
  radio/aria semantics, org-ID free-text → picker dropdowns, award-collectible
  disabled-state, a unified secret-display convention.

`task/round-fuzz-mobile-design` was a fuzz + mobile/UI-UX/contrast +
demo-functionality review round (two specialized subagents) with a
design-surface increase and boyscout fixes:

- Fuzz: added `FuzzAgentValueParsers` (internal/agent) over the credential value
  parsers — the scope enum, the label, and the secret, which base64-decodes
  untrusted input. Crash-free; the existing
  schema/value/token/parsePage/task-value targets were re-run.
- Design/edit surface: the structured response-schema designer gained `enum`
  (comma-separated allowed values) and `array` (item type) field kinds,
  alongside the scalar kinds. `SchemaFieldDraft` carries
  `itemKind`/`enumValues`; `encodeFieldSchema` emits the right schema per kind.
  The designer can now express the developer-template schemas (which use
  enum/array), not just flat scalar objects.
- Mobile/UI-UX (from the review): `Ui.fieldClass` gained `min-h-[44px]` so all
  inputs/selects meet the 44px tap target; the schema-row Required checkbox now
  uses the shared styled `checkboxClass` instead of a tiny raw input;
  `copyButton` is full-width on a phone; the designer helper text moved from
  slate-500 to slate-600 for contrast headroom. The mobile Playwright test now
  opens the task API/MCP panel and mints a token, and visits the profile
  agent-access card, asserting no horizontal overflow on those long-code-block
  surfaces. The contrast review found no WCAG failures.
- Usability: added a "Profile" nav link to the user's own page — last round's
  user-page token had no in-app navigation to reach it.
- Demo boyscout (from the demo audit): opening the demo from `file://` no longer
  renders `null/mcp` commands (both index.html files fall back to a placeholder
  origin); org-owned task funding debits the organization wallet (not the
  personal balance), and refunds/tips return to whichever wallet funded the
  task. The audit otherwise confirmed full route/decoder parity and that prior
  fixes (economy, seeds, validator) hold.

`task/user-page-token` put a personal agent token and MCP install commands on
the user's own profile page:

- On `/users/{id}` when the viewer is that user (`userId == subjectId`), a "Your
  agent access" card mints a full-capability agent credential inline (scopes
  tasks_read/tasks_write/submissions_read/submissions_write/submissions_review)
  and shows the real token in a copyable code block with a Rotate button. The
  section is omitted entirely when viewing someone else's page, so the token is
  owner-only.
- Below the token, copyable MCP install/update commands with real values (no
  placeholders):
  `claude mcp add --transport http sharecrop <origin>/mcp --header "Authorization: Bearer <token>"`
  (Claude Code), an update form
  (`claude mcp remove sharecrop && claude mcp add ...`), and the `.mcp.json`
  config block (reusing `mcpConfig`, for Codex / Claude Desktop / generic
  clients). Copy buttons reuse the clipboard port from the previous PR. Reuses
  `mintTaskToken`'s pattern with a new
  `mintUserToken`/`UserTokenMinted`/`userAgentToken`.
- Playwright covers minting on your own page (token + install commands present
  and placeholder-free) and that the section is absent on another user's page.

`task/task-integration-panel` made the task's API/MCP instructions uniform,
collapsible, and placeholder-free, with one token for both surfaces:

- A single agent token now drives both REST and MCP. New
  `requireWorkerSubject(r, scope)` (in `internal/http/server.go`) accepts either
  a user access token or an agent credential that holds the required scope,
  resolving to the credential's owning user (the same way MCP already treats an
  agent credential). It is applied to the worker REST endpoints the task
  instructions demonstrate: `GET /api/tasks/{id}` (tasks_read), reserve and
  submit (submissions_write). An http_e2e test proves an agent token works on
  those endpoints and that a token missing the scope is rejected.
- The task-detail "API & MCP" section is now a collapsible panel (collapsed by
  default). Inside, a "Create agent token" button mints an agent credential
  inline and shows the real, copyable token; below it, uniform REST and MCP
  entries each have a one-line description of what the command does, a Copy
  button, and real values (origin, task id, token) — no `<...>` placeholders.
  MCP is presented as the `.mcp.json` install snippet plus the JSON-RPC
  tool-call bodies (the client manages the session, so there is no manual
  `Mcp-Session-Id`); REST is the equivalent curl using the same token. Copy
  buttons use a new `copyToClipboard` Elm port (Main became a `port module`),
  wired in both `web/static/index.html` and `site/demo/index.html`. The old
  placeholder curl examples were removed.
- Playwright covers the collapsed-by-default panel, minting a real token, and
  that the rendered commands embed the token with no placeholders.

`task/mobile-demo-design` was a mobile-usability + demo-completeness +
design-surface round (two specialized review subagents) plus a re-run fuzz pass:

- Demo full functionality: the in-browser fake backend now moves credits on
  review. Accept releases the held escrow, records a `task_payout`, refunds any
  unpaid remainder to the balance, and charges a tip — previously it closed the
  task without moving any credits, so the seeded reward economy had no payoff.
  Reject/request-changes honor partial credit + tip and refund the rest
  (request-changes leaves escrow held). Added the missing
  `POST /api/tasks/:id/unpublish` demo route. Seeded task-7 with a `task_type`,
  `reference_url`, and a comment so those recently-added surfaces are visible
  without creating a task.
- Mobile: added a Playwright mobile project (`mobile.spec.ts` at 375x667)
  asserting no horizontal overflow across every page; it caught genuine
  overflows (the collectibles policy/award buttons, long-ID rows). Fixes:
  responsive outer page padding (`p-4 sm:p-8`) and card padding; 44px-min tap
  targets on buttons via `Ui` button classes; action rows wrap with
  shrink-0/min-w-0; review buttons stack on mobile; long-ID rows get
  `min-w-0`/`break-words`; inline forms let the field grow/shrink (`fieldLabel`
  gains `grow min-w-0`); arcade buttons may wrap. The collectible
  transfer-policy labels became human-readable ("Transferable within
  organization" instead of `transferable_within_organization`), which also
  removed an unbreakable-token overflow.
- Design/edit surface: the create-task form gained a structured response-schema
  designer — add fields (name, type, required) and the schema JSON is generated
  for you, with the raw JSON kept as an advanced fallback; the field rows stack
  on mobile.
- Fuzzing: re-ran the suite to confirm the new MCP tools (covered by
  `FuzzHandleRaw`) and all value parsers stay crash-free.

`task/fuzz-journeys-uiux` was a fuzz + user-journey + UI/UX review round (two
specialized subagents) with opportunistic boyscout fixes:

- Fuzzing: `internal/task/fuzz_test.go` `FuzzTaskValueParsers` exercises the
  value parsers added for templates/series/comments (task type, reference URL,
  comment body, series state/description) — no panics, and an accepted reference
  URL is always an absolute http(s) URL (confirming the real backend rejects
  `javascript:` and relative URLs).
- Journey/flow fixes: the demo `validateValue` understood only the designer/seed
  schema dialect (fields as a map, `items`, `required`), so tasks created from
  the new templates (canonical `fields` array, `item`, `presence`,
  `kind:"enum"`) had their submissions silently accepted; it now handles both
  dialects (and a real `enum` kind). The flagship seeded series `series-orchard`
  was published with a "tasks added here" comment but had zero member tasks;
  linked two seeded tasks (positions 1-2). A worker who submits an invalid
  response now sees the field-level `path: message` errors, not just
  "(invalid)".
- UI/UX fixes (contrast verified clean by the reviewer): comment authors render
  as links to their user page instead of a raw UUID; long reference URLs and
  comment bodies wrap (`break-all`/`break-words`) instead of overflowing the
  card; the task reference link opens in a new tab with
  `rel="noopener noreferrer"`; and arcade-theme anchor buttons (Open/View/Back)
  now get the chunky pixel-button treatment for consistency.

`task/dev-templates-comments` added pre-baked developer task types, a typed
reference URL, and per-task comments:

- A task now carries a `task_type` (general, code_review, security_review,
  product_review, ui_ux_review, qa_testing) and an optional `reference_url`
  (validated as an absolute http(s) URL — the specific pull request or resource
  to work on). `migration 000019` adds both columns (with a CHECK on the type
  and an index), plus a `task_comments` table. New domain: `TaskType`,
  `ReferenceURL`, `TaskComment`, `core.TaskCommentID`, and viewer-gated
  `AddTaskComment`/`ListTaskComments`.
- The create-task form gained a task-type picker that prefills the description
  and response schema from a client-side template catalog (each developer type
  ships a description skeleton and a ready-made object schema), plus a
  reference-URL field; the task detail shows the type badge, a clickable
  reference link, and a comment thread. `create_task` (HTTP + MCP) accepts
  `task_type`/`reference_url`, task responses and `get_task` expose them, and
  new `add_task_comment`/`list_task_comments` MCP tools plus
  `GET/POST /api/tasks/{id}/comments` back the thread. Generated `TaskResponse`
  gained the two fields; new `TaskCommentResponse`.
- The demo fake backend stores the type/reference and serves the comment
  endpoints. Tests: an http_e2e round-trip (type + reference + bad-URL
  rejection + comment thread) and a Playwright test driving the code-review
  template, the PR link, and a task comment.

`task/series-first-class` promoted task series from a grouping label to a
first-class managed domain:

- A series now carries a description and a `draft`/`published`/`closed`
  lifecycle (transitions publish/unpublish/close/reopen), supports a comment
  thread, owns a stable `/series/{id}` URL, and lets its creator add, remove,
  and reorder member tasks. Only the creator can edit a series; a draft series
  is private to its creator, and a task whose series is not published cannot be
  reserved or submitted to (enforced in the reserve and submission-eligibility
  store queries). Tasks gained an `Unpublish` transition (open -> draft) so a
  task can be pulled back to draft.
- Migration `000018` adds `description`/`state`/`updated_at` to `task_series`, a
  `series_comments` table, and an index on `tasks(series_id)`. New domain code:
  `SeriesState` + transitions, `SeriesDescription`, `CommentBody`,
  `SeriesComment`, `core.SeriesCommentID`, the creator-only service methods, and
  the store mutations (append at max-position+1, reorder by rewriting positions
  in a transaction, comment insert/list).
- New HTTP surface (`internal/http/series.go`): create/get/update, the four
  state transitions, add/remove/reorder member tasks, and the comment thread,
  plus `POST /api/tasks/{id}/unpublish`. The series detail response is
  `{series, tasks, comments}`. New MCP tools mirror the agent-relevant subset
  (`create_series`, `add_task_to_series`, `remove_task_from_series`,
  publish/unpublish/close/reopen, `add_series_comment`, `list_series_comments`,
  `unpublish_task`).
- Elm UI: a `/series` list page in the nav with a create form, and the
  `/series/{id}` detail page (previously orphaned and decoding the wrong shape)
  now shows the series, its ordered tasks linked to their detail pages, the
  comment thread, and creator-only controls (rename, the lifecycle buttons, add
  task by id, remove, reorder up/down) plus an add-comment box; each task detail
  links back to its series. The generated `TaskSeriesResponse` gained
  `description`/`state` and a new `SeriesCommentResponse` contract.
- The in-browser demo backend implements the whole series surface (seeded with a
  published "Orchard intake" series and a comment). Tests: an http_e2e lifecycle
  test (create -> add task -> publish -> comment, plus a creator-only 403), an
  MCP e2e test (create_series -> add_task_to_series -> publish -> comment), and
  a Playwright test that drives the series UI end to end.

`task/lifecycle-parity` (PR1 of a 4-PR roadmap from a full user-journey/gap
review) completed the post-and-work-a-task lifecycle for both agents and humans:

- A four-surface gap review (MCP, HTTP API, Elm UI, domain model) found that an
  agent could not actually post a workable task: `create_task` never set a
  participation policy (so the task was un-reservable) and there was no MCP tool
  to open or fund it. Fixed: `create_task` sets `Participation` (new optional
  `participation_policy` arg, default `open`), `AssigneeScope` (user), and
  `ReservationTTL` (default); added `open_task` and `fund_task` MCP tools
  (`internal/mcp/tools.go`, `tool_calls.go`, `server.go`, with adapter methods
  in `internal/http/agent_mcp.go`). `list_tasks` gained an optional `state`
  filter, and `get_submission_status` now returns `review_note` so a worker sees
  reviewer feedback. A new http_e2e test
  (`TestMCPAgentCreatesFundsOpensWorkableTask`) drives create -> fund -> open
  and has a different worker submit, proving the task is genuinely workable.
- The human web UI previously hardcoded `response_schema_json` to
  `{"kind":"freeform"}` and the payload to `none` (`Api.elm`), so a person could
  not constrain submissions or embed task input. The create-task form now has a
  response-schema textarea (defaulting to freeform) and an optional task-input
  JSON textarea, wired through new model fields/messages. The task-detail "Task
  input" block now also renders the real backend's `json` payload kind (it
  previously only matched the demo's `inline`). A Playwright test authors a
  structured schema + payload and asserts the detail surfaces both.
- The MCP install docs named scopes that do not exist (`reservations_write`,
  `reviews_write`), so a copy-paste produced a 400; corrected to the real scope
  set and documented `fund_task`/`open_task` in the propose-work loop.

`task/fuzz-flows-contrast` added a fuzz target, fixed WCAG contrast/focus
failures, and fixed a demo flow dead-end and example wording:

- Fuzzing: added `internal/http/fuzz_test.go` `FuzzParsePage` — arbitrary
  `?limit=&offset=` query strings must produce a `core.Page` whose limit stays
  in [1,200] and whose offset is never negative, so a malformed query can never
  reach SQL as an out-of-range LIMIT or a negative OFFSET, and it must never
  panic. Holds (the existing `FuzzHandleRaw` already drives the MCP tool-call
  argument decoders).
- Contrast review (computed real WCAG 2.1 ratios with the relative-luminance
  formula; confirmed that the only text on the bare green page background is the
  page-title h1 — everything else renders inside parchment cards): lightened the
  arcade theme's page green from `#6b8f3a` to `#b3cf86` so the dark heading ink
  clears AA (2.21:1 -> 4.79:1) and body ink reaches 9.17:1; added a visible
  keyboard focus outline (`:focus`/`:focus-visible`), which the theme entirely
  lacked (WCAG 2.4.7 failure for keyboard users); pinned arcade `::placeholder`
  to the muted ink (`opacity:1`), up from app.css's ~3:1 50%-of-text default. In
  the shipped app, moved the low-contrast "revoked" credential label from
  `text-slate-400` (2.56:1) to `text-slate-600` (7.58:1), and added a base
  `::placeholder` rule in `web/styles/input.css` pinning placeholders to
  slate-500 (4.76:1) instead of Tailwind v4's ~2.7:1 preflight default.
- Demo flow + example audit (two subagents — route-by-route flow functionality
  and example self-containment): the only functional gap was the task-series
  detail seed (`site/demo/backend.js`) missing `owner_kind` and `created_by`,
  which made `GET /api/task-series/:id` and the series list fail the strict
  decoder so the series page hung on "Loading series…"; seeded the contract
  fields. Reworded the review-extraction task whose "before the first colon"
  rule collided with each line's "Rating:" prefix (now "between the em dash and
  the next colon"). Every other Elm client route was verified to have a
  correctly-shaped fake-backend handler, and all seven seeded tasks remain
  solvable from their own embedded data.

`task/fuzz-and-polish` added fuzz tests over the untrusted-input parsers, fixed
a JSON-encoding bug they found, and applied a demo usability review:

- Added Go native fuzz harnesses: `internal/schema/fuzz_test.go`
  (`FuzzParseSchemaJSON`, `FuzzParseValueJSON` with an encode/re-parse
  round-trip, and `FuzzValidate` driving parse + validate + sensitivity index +
  redact), `internal/auth/fuzz_test.go` (`FuzzVerifyAccessToken` — asserts the
  token verifier never panics and never accepts a token it did not itself sign,
  i.e. no HMAC forgery), and `internal/mcp/fuzz_test.go` (`FuzzHandleRaw` — the
  JSON-RPC transport against the in-memory fake services must never panic and
  must emit valid JSON or no response). The seed corpora run as ordinary
  regression tests under `go test`.
- The round-trip fuzz found a real bug: `schema.EncodeValueJSON` quoted strings
  with `strconv.Quote`, which produces Go literal syntax — byte `0x7f` becomes
  `\x7f`, which is invalid JSON. This function encodes the redacted submission
  source that is stored and returned to task owners, so a response containing
  such a byte round-tripped to invalid JSON. Switched `writeJSONString` to
  `encoding/json`. The crasher is checked in as a regression seed under
  `internal/schema/testdata`.
- Verified (did not assume) that the nested-union validator is not a DoS vector:
  exponential validation cost requires an exponential-size schema, which the
  HTTP body-size cap already prevents; a body-sized nested/wide union validates
  in microseconds-to-milliseconds. No artificial complexity budget was added.
- Demo usability review (two subagents — example correctness and flow
  dead-ends), all fixed in `site/demo/backend.js` (+ one Elm label change
  rebuilt into the demo bundle): added `POST /api/tasks/:id/refund` (the Refund
  owner control previously hit the catch-all `ok({})` and failed to decode the
  escrow shape) and `GET /api/organizations/:id/credits/balance` (the org page
  balance was stuck on "Loading…"); fixed the date-normalization task whose
  stated "day-first" rule contradicted its own month-first worked example;
  reworded a support-ticket example that was equally defensible as `bug` or
  `billing`; and relabeled the collectibles award flow ("Award a collectible to
  a task" + helper text + "Award to selected task" button) so the separate task
  picker and per-collectible button read as one two-step action. `demo.spec.ts`
  gained tests for the refund and org-balance flows.

`task/demo-on-real-elm` rebuilt the demo to run the real Elm client against an
in-browser fake backend:

- Replaced the hand-built static demo with the actual compiled Elm client
  (`site/demo/elm.js` + `app.css`) served alongside `backend.js`, an
  `XMLHttpRequest` shim (elm/http uses XHR, not fetch) with a stateful in-memory
  store that answers every `/api/*` call. Seeded with realistic, specific
  agentic-work tasks — invoice line-item extraction, support-ticket
  classification, ledger fraud verification, a weather agent, field-note
  transcription, photo alt-text — each with real input payloads and strict
  response schemas. The demo is now the same code as the shipped client and
  cannot drift.
- Ported the pixel-art arcade theme onto the real client via `arcade.css`
  (loaded only by the demo): grassy backdrop, parchment dialog-box panels with
  hard offset shadows, blocky pressable buttons and nav, terminal-green schema
  blocks, and Press Start 2P / VT323 fonts, overriding the client's Tailwind
  utilities. The shipped app never loads it.
- Seeded two public tasks owned by other users (and one agent-submitted task) so
  the discover -> reserve -> submit worker loop and the reverse-MCP story are
  exercisable, not just the requester side.
- Ran a UI/UX + flow-correctness review of the rebuilt demo. It caught that the
  shim emitted enum values the real Elm decoders reject (`availability_kind`
  like `pending_approval`/`submitted`, a `funded` task state, a `task_funding`
  ledger kind, a `reservations_write` scope, a `cancelled` reservation state) —
  and because Elm's `Decode.list` is all-or-nothing, those blanked the My-tasks
  list, ledger, agents screen, and task detail. Fixed every enum to the
  decoder-valid set and corrected the post/fund/open and review flows. Replaced
  `demo_static.spec.ts` with `demo.spec.ts` (serves `site/demo` in-process;
  asserts boot, ledger + My-tasks populate, and detail schema render).
- Known limitation: hard-refresh / deep-link on a demo sub-route 404s on GitHub
  Pages because the path-routed Elm `Browser.application` builds root-absolute
  URLs under the `/sharecrop/demo/` base; in-app click navigation works.
  Recorded in BUGS.md.

`task/collectible-tips-arcade-mcp` added collectible tips, a pixel-art demo
theme, MCP docs, and fixtures, with reviews:

- Collectible/inventory tips (real app + demo): added `assets.AllowsTip`,
  `assets.GiftCollectible` (service), and a `GiftCollectible` store transfer
  (lock the collectible, enforce ownership + minted state + transfer policy,
  update `owner_user_id`). The accept handler parsed `tip_collectible_id`,
  settled credits, derived the worker from the payout, and gifted the
  collectible. A later branch moved collectible tips into the ledger accept
  transaction. An e2e test covers a successful tip and the policy refusal. The
  demo review console offers a "Tip a collectible" select that transfers from
  the reviewer's inventory to the worker on accept.
- Pixel-art "arcade" theme (now the demo default): a farm-RPG palette, chunky
  hard-outlined dialog-box panels with hard offset shadows, blocky pressable
  buttons, square pills, terminal-green schema blocks, and pixel fonts (Press
  Start 2P headings, VT323 body), inspired by Habitica / idle-clicker UIs.
  Scoped to `body[data-theme="arcade"]`; the other themes stay selectable.
- MCP docs: precise install steps (scoped agent token, `/mcp` client config, an
  initialize handshake) and the agent work loop as concrete tool calls — poll
  (`list_tasks`/`get_task`/`get_task_schema`), claim (`reserve_task`), submit
  (`submit_response`/`get_submission_status`), review
  (`accept_submission`/`reject_submission`/`request_submission_changes`,
  approve/decline reservation), and propose (`create_task`).
- Contract fixtures: pinned the wire JSON shape of six uncovered response DTOs
  (reservation, team, organization, organization member, task capability token,
  submission-created).
- Reviews + fixes: a security review of the new collectible-tip/rate-limit code
  surfaced only medium/low items, all addressed (within-org tip denied until org
  is modeled, idempotent gift, uniform tip error, accept/reject now rate-limited
  per subject). A UI review of the arcade theme drove fixes: dark-mode
  primary-button/active-nav contrast, button labels kept whole (whole buttons
  wrap), schema-block padding, a clear disabled-button state, more legible VT323
  eyebrows, scrolling mobile tabs, and wrapping reward rows.
- Deferred: the out-of-process Postgres session/SSE/rate-limiter store
  (cross-process SSE replay needs `LISTEN/NOTIFY`); queued as DO_NEXT #1.

`task/ratelimit-tipkey-reviews` landed the security follow-ups and ran
third-pass reviews:

- Added an in-memory token-bucket rate limiter (`internal/http/rate_limit.go`):
  per-client-IP on the unauthenticated login/refresh/receipt endpoints and
  per-agent-subject on MCP requests, returning HTTP 429 when exceeded. Idle
  buckets are evicted so keys cannot accumulate; client IP uses the direct peer
  (X-Forwarded-For is not trusted). Unit-tested.
- Gave the two `task_tip` ledger inserts derived idempotency keys (`:tip-debit`
  / `:tip-credit`), matching payout/refund entries, so the unique constraint
  would catch a double-tip if the task-lock ordering ever changed. This closes
  both lower-risk follow-ups that prior security reviews had recorded in
  BUGS.md.
- A third-pass security review (told what was already fixed and what was in
  flight) returned no new findings: the authz/RBAC and ledger lenses were clean
  and the single raw input finding did not survive synthesis's re-verification —
  the expected outcome at this maturity.
- A round-5 UI/UX review found a real logic bug and a product gap, both fixed:
  reject left a task open and re-submittable after releasing escrow (so it could
  never be paid) — reject now closes the task; and agent submissions were
  indistinguishable from human ones at the review surface — agent-originated
  reservations/submissions are now marked and rendered as "Sol Rivera · agent"
  with a "via MCP · scoped token" chip in place of the human track record. Also:
  a structured schema with no fields now warns and shows an empty state;
  dashboard cards have parity sub-notes; the Release button is hidden once a
  result is submitted; a credits/bundle reward warns on a non-positive amount;
  review panels size to their own content; and the docs MCP tool names match the
  Agent/API console.

`task/http-dtos-and-reviews` continued the HTTP split, landed the deferred UI
minors, and ran second-pass reviews:

- Moved the HTTP request/response DTO struct declarations and the
  `writableResponse` interface out of `server.go` into `dtos.go` (package
  `httpserver`); `server.go` is about 906 lines. No behavior change.
- A second-pass security review (given the prior findings and accepted risks)
  found one new, real, high-severity IDOR: `changeReservationByRequester`
  checked ownership of the URL-path task, but `ChangeReservationState` matched
  the reservation by id only via an auto-commit `Exec`, checking the
  reservation-to-task binding only after the write. An actor owning any task
  could approve/decline/cancel a reservation belonging to another task
  (force-granting submission eligibility or denying a legitimate worker). Fixed
  by binding the `UPDATE` to `task_id` in the same statement, with an e2e test;
  the service-layer post-check is now defense-in-depth. No other new issues; the
  prior accepted risks remain in BUGS.md.
- Landed the deferred demo UI minors: unified the two neutral metadata-chip
  styles into one (toned reward/status chips keep their color); replaced the
  docs placeholder with a real quickstart (task lifecycle, MCP connect config,
  scoped/revocable tokens, REST/MCP tool reference); and added a per-persona
  lifetime track record (settled tasks + acceptance rate) shown as a trust
  signal on profiles, the reservation queue, and submission rows.
- A round-4 UI/UX review drove further fixes: "Run as Sol agent" now requires
  the Agent operator persona and an approved reservation for approval-policy
  tasks (it could previously inject a submission under Sol's identity from any
  persona and bypass the approval gate); task-list status renders as a colored
  pill everywhere (a shared `status-pill` base on `toneSpan`); funding failure
  shows an inline reason at the Fund control; the dashboard open-task count is
  scoped to the persona's visible tasks; the invoice timeline actor was
  corrected and reservation-state labels humanized.

`task/http-split-and-security` split `server.go` and applied security + UI/UX
reviews:

- Split `internal/http/server.go` (about 2476 lines) into cohesive files in
  package `httpserver`: `tasks.go` (task + reservation handlers, task request
  decoders, task response converters), `submissions.go`, `reviews.go`, and
  `credits.go`. `server.go` retains the router, shared request/response types,
  and shared helpers (about 1186 lines). No behavior change; Go unit + http_e2e
  suites, vet, gofmt, copy-paste, and dead-code all pass.
- Ran a multi-agent security review of the Go backend (lenses: authz/RBAC/IDOR,
  input/injection, secrets/tokens/crypto, ledger/escrow integrity & concurrency,
  MCP/session/DoS). The authz/RBAC lens found no issues, and the synthesis
  verified-and-dropped false positives (receipt-token enumeration, tip
  replay/overdraft — both ruled out by a 256-bit random capability and a
  `FOR UPDATE` task lock). Applied the real findings: the refresh-token cookie
  is now `Secure` by default (env opt-out for local HTTP dev); MCP HTTP sessions
  are capped per agent subject and globally with a 429 on overflow (covered by a
  new test); and the submission response-value parser caps array items and
  object fields. Rate limiting and tip-entry idempotency keys were recorded in
  BUGS.md as accepted lower-risk follow-ups.
- Ran a multi-agent UI/UX/product review of the demo and fixed the majors: every
  open task is now escrow-backed (the collectible-only audit task carries
  credits, so "open = funded escrow" holds); Reject defaults to paying 0 while
  Accept defaults to the full reward, each shown on its button; "Run as Sol
  agent" is gated by the same claimability rules as the task page (it can no
  longer reopen a settled task or steal another worker's); Post Task shows only
  the reward inputs for the chosen reward kind; and the tip hint / worker
  response nudge copy were clarified. Three moderate minors (neutral-chip style
  unification, a real Docs page, a worker trust signal) were deferred to
  DO_NEXT.

`task/elm-split-schema-polish` finished the Main.elm decomposition, deepened the
schema designer, and fixed the demo economy:

- Finished decomposing `Main.elm`: extracted `Sharecrop.View` (the view layer
  plus the `*SuccessLabel` strings) and `Sharecrop.Api` (the HTTP commands,
  request-body encoders, decoders, result extractors, and the
  `withSession`/`updateLoggedIn`/`*Command` update glue), and lifted the shared
  `pageToPath` and visibility helpers into `Sharecrop.Types`. `Main.elm` went
  from about 2944 lines to about 681 (init, update dispatch, routing, and
  wiring). No behavior change; 21 Playwright tests and the copy-paste/dead-code
  checks pass.
- Schema designer: list fields gained min/max item constraints, validated
  against worker submissions; field names are normalized to lowercase
  identifier-safe keys shown inline; the designer warns when min items exceed
  max items; and the decimal field type is validated as a real number.
- Demo economy: seed balances are now net of each requester's escrow (the
  persona balance is the total; committed escrow is carved out), so available +
  escrow reconciles and closing seeded tasks no longer mints credits on refund
  or settle. Review Settle/Tip became numeric inputs pre-filled with real
  defaults in a two-column grid; a settle is bounded by escrow and the tip by
  the requester's spendable balance, with over-limit settles refused via a
  warning rather than clamped; Accept/Reject surface the amount paid; escrow is
  shown on the worker-facing task page; declining an approval returns the task
  to a claimable state; and the Agent/API console and Reviews page select only
  tasks visible/actionable to the persona.
- A multi-agent UI/UX/user-journey/product review of the demo drove the
  economy/designer fixes. It found a real escrow-accounting blocker (seeded
  escrow was never debited from balances, so the dashboard double-counted and
  settles minted credits) and an org-task leak in the Agent/API console (the
  external operator could read org-only tasks). Both are fixed.

`task/demo-orgs-elm-split-polish` modeled demo organizations, decomposed
Main.elm, and applied a specialized review:

- Demo organizations as entities: demo users now belong to an organization (the
  agent operator is external, with no org), and organization-visibility tasks
  carry an org id. `canSeeTask` and `canReviewTask` scope organization tasks to
  members of the owning org instead of a role-string check, so external users no
  longer see or review org-internal work. The org name surfaces in the topbar,
  the persona switcher, and the task badge.
- Main.elm decomposition: lifted the
  `Flags`/`Session`/`Page`/`LoggedInModel`/`TaskDetail`/`Model`/`Msg` type block
  out of `Main.elm` into a new `Sharecrop.Types` module (no behavior change),
  shrinking the monolith by ~230 lines and unblocking later view/command splits.
- Ran a specialized multi-agent UI/UX/user-journey/product review of the evolved
  demo and fixed what it found: a real escrow bug (settle netted escrow/payout
  against the reviewer clicking Accept rather than the task's requester — now
  resolves the requester explicitly); the Agent/API console leaked org-internal
  tasks to the external agent (now lists only the persona's visible tasks);
  submit validation accepted the empty prefilled template (now rejects empty
  required fields/arrays); the simulated agent run submitted an empty skeleton
  (now a realistic schema-filled payload); the reservation queue offered
  self-approval (now hidden, and the seeded org reservation belongs to a
  worker); org identity was invisible (now in the topbar + persona switcher);
  and the task badge row was all-green (metadata pills are now neutral so only
  status pills carry color). A few minor items (seed-escrow presentation,
  settle-input default legibility, over-payout clamping) are recorded in
  DO_NEXT.

`task/economy-orgs-and-polish` bundled a reward economy, schema validation, and
team membership:

- Demo reward economy: Fund now checks the requester's available balance and
  moves the reward credits into a per-task escrow bucket; accept/reject settles
  from escrow (refunding any unused credits and netting the tip) and cancel
  refunds it. The dashboard shows "Credits available" and "Held in escrow", and
  seed escrow is derived from already-funded open tasks so balances stay
  consistent.
- Demo schema designer + validation: each text field can be marked required and
  given allowed values (enum); the designer warns on duplicate/empty field names
  and shows the constraints in the friendly summary. Worker submissions are
  validated against the schema on submit — missing required keys, wrong types,
  and out-of-enum values are reported inline and block the submission; a valid
  one is confirmed and stored.
- Demo polish: the comms/activity feed linkifies persona names to their profiles
  (matching the cross-linked app), and the unused difficulty field was removed
  from the seed data.
- Real app standalone-team membership: `POST /api/teams/{id}/members`
  (org.Service.AddTeamMember + store AddTeamMemberByEmail) lets a user-owned
  team's owner — or a manager of an owning organization — add members by email;
  the team page shows an add-member form to the owner and refreshes the roster.
  RBAC denies others (403). Covered by an e2e test and a Playwright flow.
- Deferred to focused follow-up PRs: modeling organizations as real entities
  (org id on users + tasks, a DB migration plus RBAC rewrite) and decomposing
  the `Main.elm` monolith (which needs the interdependent Model/Msg types lifted
  into shared modules first). Both are large refactors and were left out of this
  bundle to keep it clean and green; they are recorded in DO_NEXT.

`task/demo-cross-linking` made demo entities real links and acted on a
specialized review:

- Converted task and user references into real anchors. Task rows use a
  stretched `#/tasks/{id}` link over the whole row and user names are
  `#/users/{id}` anchors, so they support left-click, open-in-new-tab, and
  right-click; `handleClick` returns early for anchors so the browser handles
  them. The dead kanban board/card code and CSS were removed (the flat list is
  the only task list).
- Added a per-row reserve control: Reserve / Request approval when claimable,
  the requester's context action (Fund / Open / Review queue) now carrying the
  row's task id, an "Open to submit/run agent" action for available tasks, or a
  muted non-interactive "Reserved" pill when already claimed.
- Ran a UI/UX + user-journey + product review (multi-agent) and applied its
  fixes: the per-row Review-queue button now opens that row's task; available
  rows always show a next action; `nextAction` blocks stealing another worker's
  active reservation on the detail page; rows top-align so status no longer
  floats; rows get cursor/hover/underline affordance; the dashboard hero states
  the reverse-MCP value proposition; Post Task explains what a credit is; stray
  "mission" copy and the RPG-style S/A/B/C difficulty badge were removed. Full
  reward escrow accounting and linkifying the activity feed were left for
  DO_NEXT.
- Replaced the bespoke copy-paste script with jscpd (the standard cross-language
  detector), pinned to 5.0.11 and tuned to 12 lines / 150 tokens (now also
  scanning `site/demo`). Added a `.pre-commit-config.yaml` that runs jscpd and
  `go tool deadcode`, and a CI `pre-commit` job that runs the hooks via the
  framework.
- Verified every GitHub Action and tool version against its registry/release
  feed and pinned each to the latest published more than a day old (checkout
  v7.0.0, setup-go v6.4.0, setup-python v6.2.0, setup-deno v2.0.4, deno v2.8.3,
  pages actions v6/v5/v5; playwright kept at 1.61.0 since 1.61.1 was under a day
  old).

`task/demo-stakeholder-review-polish` acted on a multi-stakeholder review
(requester, worker, agent operator, org reviewer, first-time visitor, visual/UX,
accessibility) of the demo and fixed what it surfaced:

- Payout: Accept now settles the full funded reward by default instead of a
  canned 18-credit partial that silently underpaid, and the requester is debited
  the payout plus tip (credits actually move). The canned review note and
  amounts no longer bleed across tasks.
- Submissions: the response box is stored per task and prefilled from that
  task's schema; the orchard seed submission was corrected to match its own
  schema; the agent-run submission is schema-shaped.
- Clarity: the dashboard hero states the product (request agentic tasks from
  people and their agents); mission/payload/reward-crate/uplink jargon is
  replaced with task/response/reward across copy, buttons, and timelines;
  schemas are rendered in plain language ("labels: list of text") next to the
  raw JSON in the designer, the briefing, and the review console; a worker sees
  the review note and their prior response when changes are requested.
- Agent/API console: one host and one token placeholder, REST and agent payloads
  generated from the task schema, worker-only scopes, and a policy-aware MCP
  workflow. An organization reviewer can no longer review public tasks they did
  not request.
- Visual: the dark-mode hard offset shadow was replaced with a soft shadow, a
  visible focus ring was added, status badges are color-coded by
  lifecycle/availability, the persona control reads as a role switcher, and the
  reservation-expiry field is hidden for open-submission tasks.

Earlier, \`task/demo-selfcontained-tasks-and-redesign\` made the demo tasks real
and redesigned the demo:

- Reworked every demo seed task so it carries its own input material. Each task
  gained an `inputs` array of blocks (records table, list, text, or code)
  rendered as an "Input / materials" section in the briefing, and the objectives
  now reference that on-screen material. This fixes tasks that described inputs
  ("20 photos by URL", "the linked ledger") that did not exist anywhere — they
  are now completable from what is shown. The tasks are framed as reverse-MCP
  agentic requests: humans asking other people and their agents for a structured
  result.
- Added a receive-schema designer to the Post Task page: a requester writes
  free-form instructions and either keeps a free-form response or builds a
  structured one by adding named fields with types (text, whole number, decimal,
  list of text); the generated response schema is shown live.
- Redesigned the demo visuals: switched the default to the clean "showcase"
  theme, replaced the hard offset shadow with a soft shadow, increased the
  corner radius, lightened the heavy panel borders, and styled the new input
  tables/lists/code and schema designer. Replaced the full task briefing that
  was wedged onto the dashboard with a compact "continue where you left off"
  spotlight.

Earlier, `task/team-pages-and-module-split` finished the entity-page work and
paid down the HTTP and browser monoliths:

- Added `GET /api/teams/{id}`, returning a team and its member roster, with a
  new `org.Service.GetTeam` that allows a viewer only when they own the team,
  belong to it, or (for an organization team) are a member of the owning
  organization, backed by store `FindTeam` and `ListTeamMembers`. A routed
  `/teams/{id}` page renders the team name, owner kind, and roster (each member
  linking to their profile); organization team rows link to it. An e2e test
  proves the roster is denied to unrelated users.
- Added an assignee-scope selector (user or organization team) to the
  create-task form, wiring the existing `assignee_scope` field instead of always
  assigning to a user. A browser test confirms a worker sees the
  organization-team assignee scope.
- Split the HTTP handler monolith: organization and team handlers moved to
  `internal/http/organizations.go`, funding and refund handlers to
  `internal/http/funding.go`, and the team-detail handler is in
  `internal/http/teams.go` (joining the earlier `users.go`, `series.go`, and
  `org_credits.go`). No behavior change; shared request and response types and
  writers stay in `server.go`.
- Split the Elm monolith: the pure enum, label, and format helpers moved from
  `Main.elm` into a new `Sharecrop.Labels` module, shrinking `Main.elm` by
  roughly 300 lines with no behavior change.

Earlier, `task/org-followups` added linkable, RBAC-aware pages for every entity
a user can reach and finished the organization follow-ups:

- Rewrote the static demo seed tasks to be self-contained (concrete input,
  deliverable, and acceptance criteria) and de-jargoned the personas and areas,
  then added hash-routed demo pages including per-user profiles and an
  always-visible reset control.
- Gave the browser app a URL per entity: routed `/organizations/{id}`; a
  role-aware `/tasks/{id}` that shows owner controls (open, refund, review) to
  the task creator and worker controls (reserve, submit) to others, replacing
  the inline owner detail; `/users/{id}` profiles; `/users/{id}/work`;
  `/users/{id}/submissions`; `/collectibles/{id}`; and `/series/{id}`.
- Added `GET /api/organizations/{id}/members` (real membership-and-roles query,
  restricted to active members) with a member list in the organization page;
  `GET /api/users/{id}` (a user's public tasks via a public-only
  `CreatorListScope`); `GET /api/users/{id}/work` (public assignments via
  `AssigneeListScope`); and `GET /api/users/{id}/submissions` (the caller's own
  submissions only).
- Enforced and tested role-based access control on every new surface: a private
  task is denied through task detail and is absent from public discovery and
  from the owner's public profile; submissions are visible only to their
  submitter (others get 403); the member roster is visible only to members.
- Extended the create-task form with team and organization visibility scopes (a
  standalone team id is a valid scope) and let the funding form fund a task from
  organization credits via the existing org-funding endpoint.
- Moved the user-profile, work, and submissions handlers into
  `internal/http/users.go`.
- Verified all responses are real persisted data with no mocks, placeholders, or
  stubs in production code.

Earlier, `task/multi-page-routing` gave the browser app real per-section URLs
and decomposed the single dashboard panel:

- The HTTP server now serves the single-page-application shell for every non-API
  route (`index` no longer 404s non-root paths), so deep links and refreshes
  load the app. Unmatched API paths still return 404.
- The Elm app routes each section to its own URL and page: `/` overview,
  `/tasks`, `/tasks/new`, `/tasks/{id}`, `/discovery`, `/funding`, `/agents`,
  `/collectibles`, `/organizations`. The navigation bar uses real `<a href>`
  links, and the one stacked dashboard was split into focused pages, with
  per-page data loading.
- The static demo gained an always-visible reset control in the top bar, in
  addition to the settings-page control.
- Added Playwright coverage that link navigation updates the URL and that
  deep-linking a page loads it, plus HTTP end-to-end coverage that deep routes
  serve the shell while unknown API paths 404.

Following the review branch, `task/teams-org-context-collectible-ui` picked up
three deferred follow-ups:

- Added standalone (user-owned) teams. Team ownership is a tagged union over
  organization-owned and user-owned teams, with migration 000017, store create
  and list methods, `POST` and `GET /api/teams`, the owner exposed on the team
  contract, and e2e coverage. This is the clean redo of the standalone-teams
  attempt that was reverted on the review branch.
- Added organization context to the browser: an organization switcher that loads
  the organization credit balance, organization-scoped task list, and teams;
  team creation and member provisioning for the active organization; and
  organization-owned task creation through an owner chooser. Member listing and
  organization-funded tasks from the browser remain follow-ups because no member
  listing endpoint exists yet.
- Surfaced multiple collectible rewards in the browser: the reward label is
  pluralized, the escrowed collectible count shows on tasks even when
  collectibles are awarded ad hoc, and the task list refreshes after awarding.

A multi-area review of the product, browser UI, HTTP and MCP API, backend
domain, data model, tests, and security produced a set of improvements landed on
`task/full-review-improvements`:

- Pinned submission redaction behavior: unauthorized receipt-token holders
  receive redacted data, authorized requesters and organization reviewers
  receive unredacted data, and non-reviewers receive `403`. The earlier reported
  "list leak" was not a leak; the redaction model is for unauthorized viewers.
- Authorized submission accept, reject, and request-changes for organization
  members with the review-submissions permission, resolved inside the review
  transaction so authorization cannot drift from the write.
- Bounded the Sharecrop schema and value parsers by nesting depth.
- Added a request body size limit to the JSON HTTP endpoints.
- Added refresh-token-family reuse revocation backed by a `family_id` column.
- Added an idle-timeout eviction to the in-memory MCP HTTP session store.
- Kept the transactional acceptance checks in the database transaction rather
  than moving them to the service layer, and added a concurrency test proving at
  most one accepted submission per task. Moving the checks out would have
  reintroduced a time-of-check/time-of-use gap.
- Added multiple collectible rewards per task, transferring all held
  collectibles on acceptance and returning them on refund.
- Added `limit`/`offset` pagination to list endpoints through a `core.Page`
  value type.
- Added `state` and `participation_policy` filters to the task list and exposed
  the active reservation assignee on task list items, fixing an
  operator-precedence bug in the user-scope list query.
- Added browser task visibility controls (public, private, specific user),
  task-state guidance, a task-state filter, active-assignee display, an
  organizations panel, and improved checkbox and label accessibility and
  contrast through shared `Sharecrop.Ui` helpers.
- Extracted the auth HTTP handlers into `internal/http/auth_handlers.go`.
- Ran a minimalism review of the branch and applied the safe,
  behavior-preserving simplifications it confirmed.

Standalone (user-owned) teams, deeper organization context in the browser, and a
fuller decomposition of `internal/http/server.go` and `web/elm/src/Main.elm`
were left as follow-ups in [DO_NEXT.md](./DO_NEXT.md).

The project plan was written in [PLAN.md](./PLAN.md).

The agent workflow was documented in [AGENTS.md](./AGENTS.md).

The Claude pointer file was added in [CLAUDE.md](./CLAUDE.md).

The continuity-file policy was clarified:

- Continuity files were set to update before and after each task.
- [STATUS.md](./STATUS.md) was set to summarize implementation status precisely
  and factually.
- [WHAT_WE_DID.md](./WHAT_WE_DID.md) was set to remain append-oriented while
  allowing old or irrelevant parts to be compressed.
- [DO_NEXT.md](./DO_NEXT.md) was set to hold a prioritized queue.
- [BUGS.md](./BUGS.md) was set to include confirmed defects, test gaps, and open
  risks.
- pull request descriptions were set to be precise and timeless without
  reproducing code.

The remaining agent-practice questions were resolved:

- [STATUS.md](./STATUS.md) was set to stay short and cover current implemented
  surface, test status, active task, and blocking issues.
- Continuity updates were set to happen in the same task branch, with the final
  after-task update near the end of the branch.

The task workflow was updated to use one commit per task, with code, tests, and
continuity-file updates included in that task commit.

Testing was set to happen throughout each task and again before finishing the
task.

The task workflow was updated again so agents create one git commit at the end
of each task by default.

The task workflow was updated so each task uses its own task branch and pull
request.

The pull request workflow was constrained to one open pull request at a time.

New task branches were set to start from synced `origin/main` after the previous
task pull request is merged.

user interface changes were set to require manual screenshot review when
practical.

Playwright user interface tests were set to grow as the user interface matures
and workflows stabilize.

The project repository and pull request 1 implementation defaults were recorded:

- GitHub project URL: `https://github.com/e6qu/sharecrop`.
- Canonical SSH remote: `git@github.com:e6qu/sharecrop.git`.
- Go module path: `github.com/e6qu/sharecrop`.
- Local development was set to use Docker Compose for Postgres.
- App config was set to use `DATABASE_URL`.
- The task runner was set to `make`.
- The frontend tool runner was changed to Deno.
- npm was excluded from the frontend toolchain.
- Elm and Tailwind were set to run through Deno-managed tooling or pinned local
  tooling without npm.
- The first test database strategy was set to one resettable test database per
  test run.
- The first migration command was set to `sharecrop migrate up`.
- The default app port was set to `18080`.
- The default local Postgres port was set to `15432`.
- Common development ports such as `3000`, `5432`, `8000`, and `8080` were
  avoided.

The MCP implementation direction was changed:

- No Go MCP library was selected.
- MCP protocol handling was set to be implemented locally from the official MCP
  specification.

Vitest was considered and not selected.

Deno's built-in test runner was selected for Deno tooling unless a
TypeScript/Vite layer is introduced later.

pull request 1 added the project skeleton and build system:

- The Go module `github.com/e6qu/sharecrop` was created.
- The `cmd/sharecrop` binary entry point was added.
- A `net/http` server was added with `/healthz` and an embedded static app
  shell.
- Config loading was added for HTTP address, `DATABASE_URL`, and migrations
  directory.
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
- `make` commands were added for build, test, serve, migration, frontend, and
  user interface end-to-end.
- Generated local artifacts were excluded through `.gitignore`.

pull request 1 verification was performed:

- `GOCACHE=$PWD/.cache/go-build go test ./...` passed.
- `deno task test` passed.
- `deno task frontend:build` passed.
- `make build` passed.
- `deno task e2e:ui` passed earlier in the task.
- Manual screenshot review showed the app shell rendering the Sharecrop heading
  and skeleton text.

pull request 1 verification gaps were recorded:

- Docker Compose Postgres startup was not verified because the environment
  rejected Docker approval.
- `sharecrop migrate up` against live Postgres was not verified for the same
  reason.
- Final rerun of `deno task e2e:ui` was not performed because
  local-network/browser permissions had already been exhausted in this
  environment after an earlier successful run.
- `make build` with both `GOCACHE` and `GOMODCACHE` isolated inside the
  workspace could not fetch `pgx` because network access was restricted. The
  build had passed earlier with the existing module cache.

pull request 2 added core domain foundations and continuous integration quality
gates:

- Core domain errors were added.
- Strong ID wrappers were added for users, tasks, and organizations.
- UUIDv7 generation and parsing were isolated behind `internal/core/id`.
- Lifecycle state parsing was added.
- Visibility scope variants and parsing were added.
- Per-type result variants were used instead of generic result types.
- continuous integration was added for formatting, TypeScript checks, policy
  checks, copy-paste detection, dead-code detection, Deno linting, Go vet, unit
  tests, frontend build, binary build, migrations, HTTP end-to-end, and user
  interface end-to-end.
- continuous integration was limited to pull requests targeting `main`, without
  direct `main` push runs or bare branch push runs.
- The Elm build tool was changed to require explicit `ELM_BIN`.
- Config loading was changed to require explicit environment variables instead
  of fallback values.
- Docker Compose was fixed for PostgreSQL 18 by mounting the volume at
  `/var/lib/postgresql`.

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
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod make build`
  passed.
- `make test-http` passed with local listener permission.
- `deno task e2e:ui` passed with local browser permission.
- Manual screenshot review showed the app shell rendering without visible layout
  issues after pull request 2 changes.
- `docker compose up -d postgres` passed.
- `make migrate-up` passed against local Postgres.
- `docker compose down` passed.

pull request 2 verification gaps were recorded:

- Aggregate `make ci` was not run locally because the environment approval
  request timed out twice.

Pull request 3 added authentication, sessions, and guest identity:

- Guest and refresh-token identifiers were added to the core identifier set.
- Authentication value types were added for email addresses, password secrets,
  access-token secrets, refresh tokens, subjects, and session results.
- Password hashing was implemented with standard-library PBKDF2 and SHA-256
  behind `internal/auth`.
- JSON Web Token access-token signing was implemented with standard-library HMAC
  SHA-256 behind `internal/auth`.
- Opaque refresh-token generation and hashing were added.
- The authentication service added registered user creation, login, guest
  subject creation, refresh-token rotation, and refresh-token reuse rejection.
- PostgreSQL tables were added for users, guest subjects, password credentials,
  and refresh tokens.
- The PostgreSQL authentication repository was added under `internal/db`.
- HTTP endpoints were added for registration, login, refresh, and guest session
  creation.
- Refresh tokens were returned as HttpOnly cookies.
- Config parsing was split into pure `ParseConfig` and the environment-reading
  `LoadConfig` shell.
- `SHARECROP_ACCESS_TOKEN_SECRET` was added as an explicit required environment
  variable.
- Dead-code detection was changed from `go run ...@latest` to a declared Go tool
  dependency invoked through `go tool deadcode`.

Pull request 3 verification was performed:

- `make check-format` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `make check-ts` passed.
- `make lint` passed.
- `GOCACHE=$PWD/.cache/go-build make vet` passed.
- `GOCACHE=$PWD/.cache/go-build make test` passed.
- `GOCACHE=$PWD/.cache/go-build make check-dead-code` passed.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build go test -run '^$' -tags http_e2e ./tests/http_e2e`
  passed.
- `ELM_BIN=/opt/homebrew/bin/elm make frontend` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed
  with a non-fatal global module stat-cache warning from the sandbox.

Pull request 3 verification gaps were recorded:

- Runtime HTTP end-to-end tests were not run locally because the environment
  rejected the required local listener and PostgreSQL approval after the usage
  limit was reached.
- Playwright browser tests were not rerun locally because the user interface was
  not changed and the environment could not grant further browser/listener
  approval.

Pull request 4 added organizations, teams, and provisioning:

- Team and organization membership identifiers were added to the core identifier
  set.
- Organization names, team names, organization membership statuses, organization
  roles, and organization permissions were added under `internal/org`.
- Organization public-publisher permission was modeled separately from reviewer
  and billing roles.
- Organization service methods were added for organization creation,
  organization listing, member provisioning, member deactivation, team creation,
  and team listing.
- Access-token verification was added to the authentication boundary.
- PostgreSQL tables were added for organizations, organization memberships,
  organization membership roles, teams, and team members.
- PostgreSQL organization repository code was added under `internal/db`.
- HTTP endpoints were added for organization creation, organization listing,
  organization member provisioning, organization member deactivation,
  organization team creation, and organization team listing.
- Organization HTTP endpoints required verified bearer access tokens and
  service-level permission checks.

Pull request 4 test strategy was evaluated:

- Domain constructors, enums, permissions, and service permission checks were
  covered by unit tests.
- HTTP handler mapping was covered with unit tests using typed test doubles.
- API and PostgreSQL behavior were covered by HTTP end-to-end tests using the
  real migration runner, repository, service, access tokens, and PostgreSQL.
- Browser user interface tests were not expanded because this task did not
  change browser user interface source.

Pull request 4 verification was performed:

- `make check-format` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `make check-ts` passed.
- `make lint` passed.
- `GOCACHE=$PWD/.cache/go-build make vet` passed.
- `GOCACHE=$PWD/.cache/go-build make test` passed.
- `GOCACHE=$PWD/.cache/go-build make check-dead-code` passed.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build go test -run '^$' -tags http_e2e ./tests/http_e2e`
  passed.
- `docker compose up -d postgres` passed.
- `SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make migrate-up`
  passed.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make test-http`
  passed.
- `docker compose down` passed.
- `ELM_BIN=/opt/homebrew/bin/elm make frontend` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed
  with a non-fatal global module stat-cache warning from the sandbox.

Pull request 5 added the Go-to-Elm contract generator:

- Go-owned contract definitions were added under `internal/contracts`.
- The contract model covered aliases, product types, enums, named type
  references, string type references, and list type references.
- Elm generation was added for modules, type aliases, enums, decoders, and
  encoders.
- Generated Elm modules were added under `web/elm/src/Sharecrop/Generated/`.
- First generated contracts covered auth responses, error responses,
  identifiers, organization responses, organization member responses, team
  responses, membership statuses, subject kinds, and organization roles.
- The `sharecrop generate elm-contracts` command was added.
- The Makefile gained `contracts` and `check-contracts` targets.
- Frontend builds were changed to generate contracts before compiling Elm.
- The handwritten Elm app consumed the generated
  `Sharecrop.Generated.Auth.SubjectKind` type directly.

Pull request 5 test strategy was evaluated:

- Generator unit tests checked generated auth output, deterministic output, and
  absence of weak generated Elm shapes such as `Bool` and `Dict`.
- `check-contracts` verified generated files were current and deterministic.
- Elm compilation verified generated modules worked with Elm 0.19.1.
- The handwritten Elm app imported a generated module to ensure generated
  contracts were usable from normal Elm code.
- Existing HTTP end-to-end tests remained the API behavior checks for this
  slice.
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
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make frontend`
  passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed
  with a non-fatal global module stat-cache warning from the sandbox.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build go test -run '^$' -tags http_e2e ./tests/http_e2e`
  passed.
- `docker compose up -d postgres` passed.
- `SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make migrate-up`
  passed.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make test-http`
  passed.
- `ELM_BIN=/opt/homebrew/bin/elm SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build deno task e2e:ui`
  passed.
- Manual screenshot review passed for `/tmp/sharecrop-pr5-shell.png`.
- `docker compose down` passed.

Pull request 6 added the Sharecrop schema parser and validator:

- Local schema domain types were added under `internal/schema`.
- Schema kinds were added for object, array, string, integer, decimal string,
  enum, literal, union, and freeform schemas.
- Field presence was modeled as explicit `required` and `may_omit` values.
- Sensitivity categories, retention policies, and redaction policies were
  modeled as typed values.
- Schema JSON parsing converted boundary data into Sharecrop-owned schema types.
- Response payload JSON parsing converted payloads into Sharecrop-owned value
  types without using generic maps.
- Schema validation produced typed validation errors with field paths.

- Sensitive-field indexing and redaction were added for typed response values.

Pull request 7 added task series, tasks, visibility, and capability tokens:

- Task-series, task, task-visibility-scope, and task-capability-token migrations
  were added.
- Task-series and task-capability-token identifiers were added to the core
  identifier set.
- Task owner, task state, task series placement, task visibility, task payload,
  and task capability-token lifecycle types were added under `internal/task`.
- Opaque task capability-token generation and hashing were added without
  encoding task identifiers into token strings.
- The task service added task creation, opening, cancellation, listing, and
  capability-token creation.
- Organization-owned tasks required organization task-creation permission.
- Public organization tasks required organization public-publisher permission.
- Default task visibility mapped user-owned tasks to user visibility and
  organization-owned tasks to organization visibility.
- PostgreSQL task repository code was added under `internal/db`.
- HTTP task endpoints were added for creation, listing, opening, cancellation,
  and capability-token creation.
- Task response schemas were parsed with the local Sharecrop schema parser
  during task creation.
- Generated Elm contracts were extended with task identifiers, task enums, task
  list items, task lists, and task capability-token responses.

Pull request 7 test strategy was evaluated:

- Unit tests covered task state transitions, capability-token opacity,
  capability-token parsers, identifier round trips, and organization
  public-publishing permission behavior.
- HTTP unit tests covered task request parsing and default user visibility.
- HTTP end-to-end tests covered task creation, user-scoped listing, task
  opening, task cancellation, capability-token creation, and organization
  public-publishing permissions against PostgreSQL.
- Playwright and manual screenshot checks were run because generated Elm source
  changed.

Pull request 7 verification was performed:

- `GOCACHE=$PWD/.cache/go-build go test ./...` passed.
- `make check-format` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `deno task check:ts` passed.
- `deno task lint` passed.
- `GOCACHE=$PWD/.cache/go-build make vet` passed.
- `GOCACHE=$PWD/.cache/go-build make check-dead-code` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make frontend`
  passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed
  with a non-fatal global module stat-cache warning from the sandbox.
- `docker compose up -d postgres` passed.
- `SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build go run ./cmd/sharecrop migrate up`
  passed.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build go test -tags http_e2e ./tests/http_e2e`
  passed.
- `ELM_BIN=/opt/homebrew/bin/elm SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build deno task e2e:ui`
  passed.
- Manual screenshot review passed for `/tmp/sharecrop-pr7-shell.png`.

Pull request 8 added submissions, anonymous access, and sensitive-field
handling:

- Submission and submission receipt-token identifiers were added to the core
  identifier set.
- Submission tables were added for submissions, receipt tokens, validation
  errors, and sensitive-field index rows.
- Submission domain types were added for authenticated submitters, anonymous
  wallet submitters, submission states, validation outcomes, response JSON,
  wallet addresses, and receipt tokens.
- Opaque receipt-token generation and hashing were added.
- The submission service added authenticated submission, anonymous public
  submission, receipt lookup, and requester submission listing.
- Anonymous submissions were limited to public tasks.
- Anonymous submitters stored payout wallet addresses without linking them to
  user identifiers.
- Submitted response JSON was parsed and validated against the task response
  schema.
- Schema-invalid submissions were recorded with `invalid` state and
  validation-error rows.
- Sensitive submitted fields were indexed from the task response schema.
- Receipt lookup returned redacted response JSON for sensitive fields.
- PostgreSQL submission repository code was added under `internal/db`.
- HTTP endpoints were added for authenticated task submissions, anonymous public
  task submissions, receipt status, and requester submission listing.
- Generated Elm contracts were extended with submission identifiers, submission
  states, submitter kinds, validation-error responses, submission responses,
  submission lists, and submission-created responses.

Pull request 8 test strategy was evaluated:

- Unit tests covered anonymous/public submission permission, receipt-token
  creation, invalid submission recording, sensitive redaction for receipt
  lookup, and identifier round trips.
- HTTP unit tests covered authenticated submission request handling and
  receipt-token response shape.
- HTTP end-to-end tests were added for anonymous public submission, receipt
  redaction, invalid response recording, and requester submission listing.
- Browser user interface tests were not expanded because pull request 8 did not
  add visible submission screens.

Pull request 8 verification was performed:

- `GOCACHE=$PWD/.cache/go-build go test ./...` passed.
- `make check-format` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `deno task check:ts` passed.
- `deno task lint` passed.
- `GOCACHE=$PWD/.cache/go-build go vet ./...` passed.
- `GOCACHE=$PWD/.cache/go-build make check-dead-code` passed.
- `go run ./cmd/sharecrop generate elm-contracts` regenerated identical
  generated Elm contracts.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make frontend`
  passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build`
  passed.
- `docker compose up -d postgres` passed.
- `DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build go run ./cmd/sharecrop migrate up`
  applied the submission migration.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make test-http`
  passed, including anonymous submission, receipt redaction, invalid-response
  recording, and requester listing tests.
- `ELM_BIN=/opt/homebrew/bin/elm ... make e2e-ui` passed the app-shell
  Playwright smoke test.
- `deno task test` passed.
- `docker compose down` passed.
- Sensitive-field indexing located sensitive values in submitted payloads.
- Redaction replaced or removed sensitive fields according to schema policy.

Pull request 6 test strategy was evaluated:

- Parser tests covered typed parsing, unsupported schema kinds, freeform mode,
  union schemas, and enum rejection.
- Validator tests covered required field failures and valid object payloads.
- Sensitive-data tests covered sensitive path indexing, replacement redaction,
  and remove redaction.
- Existing HTTP end-to-end tests remained the API behavior checks for this slice
  because task and submission endpoints are not implemented yet.
- Browser user interface tests were not expanded because this task did not
  change browser user interface source.

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
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make frontend`
  passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed
  with a non-fatal global module stat-cache warning from the sandbox.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build go test -run '^$' -tags http_e2e ./tests/http_e2e`
  passed.
- `docker compose up -d postgres` passed.
- `SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make migrate-up`
  passed.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make test-http`
  passed.
- `docker compose down` passed.

Pull request 9 added credits, ledger, escrow, first accepted submission, the
credit ledger user interface, and an expanded test pyramid:

- Credit account and ledger entry identifiers were added to the core identifier
  set.
- Credit account, append-only ledger entry, and task escrow tables were added,
  along with a single-accepted partial unique index on submissions.
- Credit domain types were added under `internal/ledger` for positive credit
  amounts, signed ledger amounts, derived balances, ledger entry kinds, escrow
  states, idempotency keys, ledger entries, and task escrows.
- Balance derivation summed the signed amounts of an account's ledger entries.
- The ledger service added task funding, submission acceptance with payout, task
  refund, balance lookup, and ledger listing.
- The PostgreSQL ledger repository performed funding, acceptance, and refund
  inside row-locked transactions.
- Each new registered user received a credit account and a `signup_grant` of 100
  credits inside the user-creation transaction.
- Task funding escrowed credits from the funder's account and required
  sufficient balance and a draft, owner-held task.
- Submission acceptance was transactional, closed the task, enforced a single
  accepted submission per task, and paid the accepted authenticated worker from
  the escrow.
- Task refund cancelled a funded task and returned escrowed credits to the
  funder.
- Fund, accept, and refund used idempotency keys so retries did not
  double-charge or double-pay.
- HTTP endpoints were added for credit balance, ledger listing, task funding,
  submission acceptance, and task refund.
- The contract generator gained an integer reference type, and a single-field
  record decoder was changed from `Decode.mapN` to `Decode.map`.
- Generated Elm contracts were extended with credit account and ledger entry
  identifiers, ledger entry kinds, escrow states, balance responses, ledger
  entry responses, ledger responses, and task escrow responses.
- The Elm app was changed from an app shell into an interactive client with
  register and login, a credit balance and ledger view, and a task funding form
  backed by the API.
- A Postgres-backed integration test tier was added under the `integration`
  build tag with a `make test-integration` target.
- continuous integration was split into parallel static, unit, build,
  integration, HTTP end-to-end, and Playwright jobs.

Pull request 9 test strategy was evaluated:

- Unit tests covered credit amount validation, signed amount parsing, ledger
  entry kind and escrow state parsing, idempotency key validation, balance
  derivation, and ledger service delegation.
- HTTP unit tests covered the credit balance endpoint and task funding request
  handling with typed test doubles.
- Integration tests covered the signup grant, funding, single-escrow
  enforcement, acceptance payout, idempotent acceptance, and refund against
  PostgreSQL.
- HTTP end-to-end tests covered the signup grant, the
  fund-open-submit-accept-payout flow, idempotent acceptance, single-accepted
  enforcement, refund, insufficient-credit funding, and no-reward acceptance.
- Playwright tests covered registering through the browser to see the signup
  grant balance and ledger entry, and funding a task through the browser.
- Manual screenshot review covered the logged-out shell and the logged-in credit
  dashboard.

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
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build`
  passed.
- `docker compose up -d postgres` passed.
- `DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make migrate-up`
  applied the credits and ledger migration.
- `DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make test-integration`
  passed and was rerun to confirm idempotency safety against a persistent
  database.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make test-http`
  passed.
- `ELM_BIN=/opt/homebrew/bin/elm ... make e2e-ui` passed the app-shell,
  signup-grant, and browser-funding Playwright tests.
- Manual screenshot review passed for `/tmp/sharecrop-pr9-shell.png` and
  `/tmp/sharecrop-pr9-dashboard.png`.
- `docker compose down` was run after verification.

Pull request 10 added MCP, agent credentials, agent setup, and task discovery
surfaces:

- Agent credential identifiers were added to the core identifier set.
- Agent credential and agent credential scope tables were added.
- Agent credential domain types were added under `internal/agent` for scopes,
  lifecycle state, labels, opaque secrets and hashes, scope sets, and scope
  checks.
- The agent credential service added scoped creation, verification, listing, and
  revocation, with PostgreSQL repository code that stored scopes in a child
  table.
- A local MCP JSON-RPC server was added under `internal/mcp`, implemented from
  the MCP specification without a Go MCP library, handling `initialize`, `ping`,
  `tools/list`, and `tools/call`.
- MCP tools were added for `sharecrop.list_tasks`, `sharecrop.get_task`,
  `sharecrop.get_task_schema`, `sharecrop.create_task`,
  `sharecrop.submit_response`, `sharecrop.get_submission_status`,
  `sharecrop.list_task_submissions`, and `sharecrop.accept_submission`, each
  gated by an agent scope and adapted over the existing task, submission, and
  ledger services.
- A task service `Get` method and a `GET /api/tasks/{task_id}` endpoint were
  added with a task view-permission check covering creators, public tasks, user
  visibility, and organization visibility.
- HTTP endpoints were added for agent credential creation, listing, and
  revocation, and a `POST /mcp` endpoint authenticated by agent credentials with
  per-tool scope enforcement.
- Generated Elm contracts were extended with the agent credential identifier,
  agent scopes, agent credential state, and agent credential responses.
- The browser app gained a task list panel with REST and MCP curl examples per
  task, and an agent setup panel for creating, viewing, and revoking scoped
  credentials with generated MCP client configuration and a one-time token.
- The Elm app was changed to accept an `origin` flag so the generated MCP
  configuration and curl examples use the live server origin.

Pull request 10 test strategy was evaluated:

- Unit tests covered agent scope parsing, scope-set de-duplication and checks,
  opaque secret round trips, label validation, and agent service
  create/verify/revoke.
- MCP unit tests covered initialize, tools/list, unknown methods, scope
  enforcement, tool dispatch, unknown tools, and domain rejections surfaced as
  tool errors.
- HTTP unit tests covered agent credential creation, unknown-scope rejection,
  and the MCP endpoint requiring an agent credential.
- Integration tests covered agent credential create, verify, list, and revoke
  against PostgreSQL.
- HTTP end-to-end tests covered the agent discover-submit-status-list-accept
  flow over MCP with a credit payout, MCP scope enforcement, revoked-credential
  rejection, and the single-task REST endpoint.
- Playwright tests covered creating an agent credential through the browser to
  see the token and MCP configuration, and listing the user's tasks with agent
  curl examples.
- Manual screenshot review covered the agent setup panel.

Pull request 10 verification was performed:

- `GOCACHE=$PWD/.cache/go-build go test ./...` passed.
- `make check-format`, `make check-contracts`, `make check-policy`,
  `make check-ts`, `make check-copy-paste`, `make check-dead-code`, `make lint`,
  and `make vet` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build`
  passed.
- `docker compose up -d postgres` passed and the agent credentials migration
  applied.
- `make test-integration` passed and was idempotency-safe across reruns.
- `make test-http` passed, including the MCP and agent flows.
- `ELM_BIN=/opt/homebrew/bin/elm ... make e2e-ui` passed the app-shell, ledger,
  and agent Playwright tests.
- Manual screenshot review passed for `/tmp/sharecrop-pr10-agents.png`.

Pull request 11 added deferred backend gaps, MCP transports, and user interface
polish with new screens:

- UUIDv7 generation was verified in code for version 7 and time ordering, and a
  parser-rejection test was added.
- HTTP contract fixture tests were added to pin the wire JSON shape of
  representative API responses.
- A task-series read API was added: `task` service `ListSeries` and `GetSeries`
  with a series view-permission check, PostgreSQL `ListSeries` and `FindSeries`
  repository code, `GET /api/task-series` and `GET /api/task-series/{id}`
  endpoints, and generated Elm task-series contracts.
- The MCP server gained `sharecrop.list_task_series` and
  `sharecrop.get_task_series` tools.
- The MCP server gained JSON-RPC batch handling through a shared `HandleRaw`
  entry point used by both transports.
- The MCP HTTP endpoint was hardened toward Streamable HTTP: a `Mcp-Session-Id`
  header on initialize, `Origin` validation for DNS-rebinding protection, a
  `405` response on `GET`, and a request body size limit.
- A stdio MCP transport was added through a `sharecrop mcp` command that
  authenticates with `SHARECROP_AGENT_TOKEN`, verifies the agent credential, and
  drives the same MCP server over newline-delimited JSON-RPC on stdin and
  stdout. This is the transport local agent clients launch.
- The transport surface was chosen from what Claude Code and Codex both
  implement as MCP clients: stdio and Streamable HTTP with a static bearer
  token. HTTP/1.1 and HTTP/2 are negotiated by the web server, and HTTP/3 and
  raw sockets were intentionally not added.
- A reusable shadcn-inspired Elm component module was added under `Sharecrop.Ui`
  with cards, buttons, inputs, badges, code blocks, and labels, and the app was
  refactored to use it.
- Browser page navigation was added with a public task discovery screen and a
  task detail screen that submits responses and lets task owners review and
  accept submissions.

Pull request 11 test strategy was evaluated:

- Unit tests covered UUIDv7 version and ordering, contract wire shapes, the
  series view-permission check, the MCP series tools, JSON-RPC batch and
  notification handling, and the stdio loop.
- Integration tests covered the task-series store list and find against
  PostgreSQL.
- HTTP end-to-end tests covered the series REST endpoints, the MCP series tools,
  MCP batch requests, the `GET` `405`, and the `Mcp-Session-Id` header.
- The stdio command was smoke-tested end to end against PostgreSQL by piping
  `initialize` and `tools/list` to `sharecrop mcp`.
- Playwright tests covered discovering a public task, submitting through the
  browser, and an owner reviewing and accepting the submission, while preserving
  the existing dashboard and agent-setup tests.
- Manual screenshot review covered the task detail screen.

Pull request 11 verification was performed:

- `make check-format`, `make check-contracts`, `make check-policy`,
  `make check-ts`, `make check-copy-paste`, `make check-dead-code`, `make lint`,
  and `make vet` passed.
- `GOCACHE=$PWD/.cache/go-build make test` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build`
  passed.
- `docker compose up -d postgres` passed and the existing migrations applied.
- `make test-integration` passed and remained idempotency-safe across reruns.
- `make test-http` passed, including the series and MCP transport tests.
- `ELM_BIN=/opt/homebrew/bin/elm ... make e2e-ui` passed the app-shell, ledger,
  agent, and screens Playwright tests.
- `SHARECROP_AGENT_TOKEN=... go run ./cmd/sharecrop mcp` returned the initialize
  result and tool list over stdio.
- Manual screenshot review passed for `/tmp/sharecrop-pr11-detail.png`.

Pull request 12 narrowed the asset economy to platform-only rewards, removed
anonymous workers, and added organization credit accounts and platform
collectibles:

- Anonymous wallet-based submission was removed: the anonymous submitter domain
  type, wallet address value, public submission route and handler, anonymous
  columns, and anonymous tests were deleted, and a migration dropped the
  `submitter_kind` and `wallet_address` columns and made submissions
  registered-users-only.
- Organization credit accounts were added: a migration extended
  `credit_accounts` to support organization owners, organizations received a
  credit account and grant inside the organization-creation transaction,
  organization-owned tasks can be funded from the organization account behind
  the manage-billing permission, and an organization credit balance endpoint was
  added.
- The ledger funding logic for user and organization funding was unified behind
  a shared escrow-completion helper.
- A platform collectible model was added under `internal/assets` with
  collectible kinds, lifecycle states, names, and transfer-policy variants, plus
  a reward-payout policy check.
- The collectible service and PostgreSQL repository added minting, listing,
  collectible task reward escrow, and refund.
- The submission-acceptance flow was generalized so accepting a submission for a
  collectible-reward task transfers the collectible to the worker, reported as a
  collectible payout.
- HTTP endpoints were added for minting and listing collectibles, funding and
  refunding collectible rewards, and the organization credit balance.
- Generated Elm contracts were extended with the collectible identifier,
  collectible kinds, states, transfer policies, and collectible responses.
- The browser app gained a collectibles panel for minting, viewing holdings, and
  awarding a collectible to a task, and the submission request dropped the
  wallet address field.

Pull request 12 scope decisions were recorded:

- Rewards were kept entirely on-platform: Sharecrop credits are the platform
  token and platform collectibles are the non-fungible reward. User-issued
  tokens, organization-issued tokens, crypto rewards, and external wallets were
  intentionally excluded.
- Anonymous workers were deferred until the anonymous identity and payout model
  is decided.

Pull request 12 test strategy was evaluated:

- Unit tests covered collectible kind, state, and transfer-policy parsing, the
  reward-payout policy check, and collectible minting.
- HTTP unit tests covered the collectible response wire shapes through the
  existing handler doubles.
- Integration tests continued to cover the ledger and series stores against
  PostgreSQL.
- HTTP end-to-end tests covered organization credit account funding and balance,
  the collectible award-on-accept flow, collectible reward refund, the
  issuer-controlled policy denial, and the rewritten registered-user submission
  tests.
- Playwright tests covered minting a collectible and awarding it to a task
  through the browser, while preserving the existing dashboard, agent,
  discovery, and acceptance tests.
- Manual screenshot review covered the collectibles panel.

Pull request 12 verification was performed:

- `make check-format`, `make check-contracts`, `make check-policy`,
  `make check-ts`, `make check-copy-paste`, `make check-dead-code`, `make lint`,
  and `make vet` passed.
- `GOCACHE=$PWD/.cache/go-build make test` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build`
  passed.
- `docker compose up -d postgres` passed and the credit account and collectible
  migrations applied on a fresh database.
- `make test-integration` and `make test-http` passed.
- `ELM_BIN=/opt/homebrew/bin/elm ... make e2e-ui` passed the app-shell, ledger,
  agent, screens, and collectible Playwright tests.
- Manual screenshot review passed for `/tmp/sharecrop-pr12-collectibles.png`.

Pull request 13 fixed reward, lifecycle, requester, contract, HTTP, MCP, and
session issues found during review:

- Tasks gained an explicit reward specification for no-reward and credit-reward
  tasks, with response fields for reward kind and credit amount.
- Credit escrow funding now requires the task to declare a matching credit
  reward, and credit-reward tasks cannot be opened until matching escrow is
  held.
- Submission acceptance stores the accept idempotency key, same-key retries
  return the accepted outcome without paying twice, and different-key re-accepts
  are rejected.
- Submission creation now requires an open visible task, and requester
  submission listing allows the task creator or organization reviewers.
- Domain errors now distinguish missing resources, permission denials,
  conflicts, and invalid states so HTTP handlers can return `404`, `403`, and
  `409` where applicable.
- Organization and collectible funding endpoints use the shared domain HTTP
  status mapping.
- Generated Elm product decoders now support records larger than eight fields,
  and generated task and ledger contracts include task detail and
  accept-submission response shapes.
- MCP task creation requires reward arguments, tool output includes reward
  details, raw JSON-RPC handling responds to `id:null`, client response objects
  are ignored as server input, and `/mcp` validates `Accept` and
  `MCP-Protocol-Version`.
- Browser routing moved to `Browser.application` with dashboard, discovery, and
  task detail routes.
- Browser auth restores sessions through refresh cookies and clears the refresh
  cookie through `POST /api/auth/logout`.
- The dashboard gained task creation with optional credit rewards, funding
  prefill for newly created credit-reward tasks, open and refund controls, task
  detail viewing, submission detail review, and accept controls.
- Browser task rows and detail screens show reward labels.

Pull request 13 test strategy was evaluated:

- Unit tests covered reward parsing and service-level submission
  visibility/open-state behavior, organization reviewer submission listing, MCP
  raw handling, and logout cookie clearing.
- Integration tests covered credit reward funding, acceptance, idempotent
  re-accept, and refund persistence against PostgreSQL.
- HTTP end-to-end tests covered credit-reward funding and payout, no-reward
  acceptance, organization credit funding, collectible funding status mapping,
  task lifecycle status mapping, MCP reward-aware flows, and submission
  visibility/open-state behavior.
- Playwright tests covered browser task funding with declared rewards, task
  discovery, worker submission, owner review and acceptance, session switching
  through logout, and the existing dashboard, agent, ledger, and collectible
  workflows.
- Manual screenshot review covered the updated dashboard.

Pull request 13 verification was performed:

- `docker compose up -d postgres` passed.
- `GOCACHE=$PWD/.cache/go-build go test ./internal/org ./internal/task ./internal/submission ./internal/http ./internal/db ./internal/mcp`
  passed.
- `DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations GOCACHE=$PWD/.cache/go-build make test-integration`
  passed.
- `DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 GOCACHE=$PWD/.cache/go-build make test-http`
  passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make frontend`
  passed.
- `ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 GOCACHE=$PWD/.cache/go-build make e2e-ui`
  passed.
- Manual screenshot review passed for `/tmp/sharecrop-dashboard.png`.
- `ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 GOCACHE=$PWD/.cache/go-build make ci`
  passed before the logout endpoint was added.
- After the logout endpoint was added, the equivalent final local check set
  passed in target groups:
  `make check-format check-contracts check-policy check-ts check-copy-paste check-dead-code lint vet test frontend build`,
  `make test-integration`, `make test-http`, and `make e2e-ui`.

The post-PR13 workflow plan was updated:

- Task rewards were planned as bundles that can contain credits, collectibles,
  both, or neither.
- Reservation-required and requester-approval-required task flows were planned.
- The default reservation expiry was set to 48 hours, with automatic release on
  expiry.
- Tasks were planned to allow exactly one active assignee: one user or one team.
- First implementation of team assignment was scoped to users and
  same-organization teams; public teams remain deferred until public-team
  modeling exists.
- Reserved tasks were planned to disappear from default discovery and reappear
  only when the viewer selects include-reserved, except for the active assignee
  and requester.
- Request changes was planned to require requester notes and keep the same
  assignee exclusive.
- Review outcomes were planned for accept, request changes, reject with optional
  partial reward, reject without reward, optional task-local implementor ban,
  and optional tips from requester balance or inventory.
- MCP work was planned to add workflow tools and full Streamable HTTP SSE with
  `GET /mcp`, `DELETE /mcp`, session enforcement, event IDs, and replay where
  practical.
- The next implementation sequence was recorded as PR 14 reservation/approval
  foundations, PR 15 requester ergonomics and task-page instructions, PR 16
  review outcomes, PR 17 reward bundles, and PR 18 MCP workflow tools plus full
  SSE.

The reservation, approval, and discovery availability foundation branch added
backend task assignment support:

- A task reservation identifier was added to the core identifier set.
- Task domain models gained participation policies, assignee scopes, reservation
  expiry, assignee variants, reservation lifecycle states, availability kinds,
  and viewer actions.
- Task creation commands and HTTP task creation requests gained participation
  policy, assignee scope, and reservation expiry values with defaults of open
  participation, user assignees, and 48 hours.
- PostgreSQL migrations added task participation fields, task reservations, and
  task-local implementor-ban storage.
- PostgreSQL task storage creates and reads participation fields, releases
  expired reservations, enforces one active reservation per task, rejects
  duplicate pending or active reservations by the same assignee, and gates
  submission eligibility to the active user reservation for reservation-required
  and approval-required tasks.
- Public task discovery hides actively reserved tasks from unrelated workers by
  default and shows them when `include_reserved=true`, while keeping them
  visible to the requester and active assignee.
- HTTP APIs were added for reserving a task, listing task reservations,
  approving a reservation, declining a reservation, and requester cancellation.
- Submission creation checks task reservation eligibility before validating and
  storing a response.
- Submission storage marks an active user reservation as submitted when that
  assignee submits.
- Generated Elm task contracts gained participation, assignee, availability,
  viewer-action, and reservation response types.
- Unit tests covered reservation service rules and submission eligibility
  rejection.
- HTTP end-to-end coverage was added for a reservation-required public task:
  reserve, unrelated submit rejection, default discovery hiding,
  include-reserved discovery, requester and active-assignee discovery
  visibility, and active-assignee submission.

The reservation foundation branch verification was performed:

- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-format check-policy check-ts check-copy-paste check-dead-code lint vet test frontend`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 go test -tags http_e2e ./tests/http_e2e -run TestReservationRequiredTaskDiscoveryAndSubmission`
  passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make test-integration`
  passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make test-http`
  passed with local Postgres access.

The requester ergonomics and task page instructions branch improved browser task
workflows:

- The requester task list now includes tasks created by the requester even when
  those tasks are publicly visible.
- Browser task creation gained participation-policy controls and a
  reservation-expiry field.
- The funding form and collectible-award form now use task selectors sourced
  from the requester's task list instead of manual task identifier fields.
- Public discovery gained an include-reserved checkbox.
- Task detail pages gained reserve/request-approval actions, requester
  reservation review controls for approve, decline, and cancel, and
  task-specific REST and MCP examples.
- Generated static browser assets were rebuilt.

The requester ergonomics branch test coverage was updated:

- HTTP end-to-end coverage checks that a requester-created public task appears
  in that requester's task list.
- Playwright funding and collectible tests use the new task selectors.
- Playwright coverage was added for creating a reservation-required public task
  through the browser, opening it, reserving it as a worker, hiding it from
  another worker by default, and showing it with include-reserved.

The requester ergonomics branch verification was performed:

- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-format check-contracts check-policy check-ts check-copy-paste check-dead-code lint vet test frontend`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make test-integration`
  passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make test-http`
  passed with local Postgres access.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui`
  passed with local Postgres access.
- Manual screenshot review passed for `/tmp/sharecrop-requester-ergonomics.png`.

The review outcomes branch added requester review flows:

- A migration added the `changes_requested` submission state, stored review
  notes, reviewer metadata, review idempotency keys, and the `task_tip` ledger
  kind.
- Submission responses expose `review_note`.
- Acceptance supports full or partial credit payout and optional credit tips
  from the requester balance. Partial acceptance refunds withheld escrow to the
  funder.
- Request changes requires requester notes, stores the note, moves the
  submission to `changes_requested`, and reactivates a submitted user
  reservation for the same implementor.
- Rejection requires requester notes and supports optional partial credit payout
  from held escrow, optional credit tip from requester balance, and optional
  task-local implementor ban.
- Task-local implementor bans block direct open-task submissions as well as
  future reservations.
- HTTP endpoints were added for request changes and rejection, and the existing
  accept endpoint gained optional `payout_amount` and `tip_amount`.
- MCP tools were added for request changes and rejection, and the accept tool
  gained optional `payout_amount` and `tip_amount`.
- Browser task detail review controls now include review note, partial payout,
  tip, ban implementor, accept, request changes, and reject controls.
- Generated Elm contracts and static browser assets were rebuilt.

The review outcomes branch test coverage was updated:

- Ledger service tests cover rejection delegation and ban selection.
- Integration tests cover partial accept with tip, request-changes note storage
  and reservation reactivation, and reject with partial payout, tip, and
  implementor ban.
- HTTP contract fixture tests cover submission review notes, accept tips, and
  review responses.

The review outcomes branch verification was performed:

- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make test-integration`
  passed after the review outcome integration tests were added.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make test-http`
  passed after HTTP and MCP review endpoints were added.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod make check-format check-policy check-ts check-copy-paste lint vet frontend build`
  passed.
- `make check-dead-code` could not be rerun after final changes because the
  required network escalation for downloading `golang.org/x/tools` was rejected
  by the approval system.
- A final rerun of `make test-integration`, `make test-http`, and UI screenshot
  review could not be performed after the final frontend and handler refactor
  because escalation was rejected by the approval system.

The reward bundles branch added combined reward modeling:

- Task reward specs gained collectible-only and bundled credit-plus-collectible
  variants.
- A migration allowed `collectible` and `bundle` task reward kinds while keeping
  credit amounts required only for credit-bearing rewards.
- Task creation through HTTP and MCP accepts reward kinds `none`, `credit`,
  `collectible`, and `bundle`.
- Task list, detail, generated Elm contracts, MCP summaries, and MCP detail
  outputs expose `reward_collectible_count` alongside reward kind and credit
  amount.
- Opening a task now requires held credit escrow for credit-bearing rewards and
  a held collectible reward for collectible-bearing rewards.
- Credit funding can coexist with a collectible reward on bundled tasks.
- Accepting a bundled task pays the credit escrow and transfers the collectible
  in one accepted payout outcome.
- Same-key accept retries reconstruct bundled payout responses without paying
  twice.
- Refunding a bundled task through the credit refund endpoint returns both the
  held credits and the held collectible; the collectible-only refund endpoint
  rejects declared bundles so it cannot strand credit escrow.
- Browser reward labels show credits, collectibles, or both.

The reward bundles branch test coverage was updated:

- HTTP end-to-end coverage verifies that bundled tasks cannot open until both
  reward components are funded, acceptance pays both components, same-key accept
  retries remain idempotent, and bundled refunds return both credits and the
  collectible.
- HTTP end-to-end helper response shapes include reward kind, credit amount, and
  collectible count.

The reward bundles branch verification was performed:

- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make frontend`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 go test -tags http_e2e ./tests/http_e2e`
  passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 go test -tags integration ./tests/integration`
  passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-format check-policy check-ts check-copy-paste lint vet test-deno check-dead-code frontend`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui`
  passed with local Postgres access.
- `make check-contracts` regenerated the intended Elm contract changes and
  failed before commit because the generated files differed from `HEAD`; it
  should be rerun after the reward bundles commit.
- Manual screenshot review was skipped; Playwright UI coverage passed.

The MCP workflow and Streamable HTTP SSE branch added the remaining MCP workflow
surface:

- MCP services and tools gained task reservation support: reserve/request
  approval, list task reservations, approve reservation, decline reservation,
  and cancel reservation.
- Reservation tool results return reservation identifiers, task identifiers,
  assignee kind and identifier, state, and requester identifier.
- Streamable HTTP MCP now stores initialized HTTP sessions and requires
  `Mcp-Session-Id` on later non-initialize POST requests.
- `GET /mcp` now serves `text/event-stream`, replays recent session response
  events after `Last-Event-ID`, stays open, and streams later POST responses to
  connected clients with event IDs.
- `DELETE /mcp` terminates the current session and later requests with that
  session ID fail.
- MCP sessions and recent response events are kept in the app process memory.
- Browser task detail MCP curl examples now show initialize first, then
  session-aware `submit_response` and `get_task_schema` tool calls.

The MCP workflow and Streamable HTTP SSE branch test coverage was updated:

- MCP unit tests cover the new reservation tool dispatch path.
- HTTP end-to-end MCP tests now initialize sessions, include `Mcp-Session-Id` on
  tool calls, cover reserve/list/approve reservation tools, cover SSE replay,
  cover live SSE delivery after a later POST, and cover session deletion.
- Existing MCP series tool HTTP tests now use initialized sessions.

The MCP workflow and Streamable HTTP SSE branch verification was performed:

- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 go test -tags http_e2e ./tests/http_e2e`
  passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 go test -tags integration ./tests/integration`
  passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui`
  passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-format check-contracts check-policy check-ts check-copy-paste check-dead-code lint vet test-deno`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build`
  passed.
- Manual screenshot review was skipped; Playwright UI coverage passed.

The UI themes and GitHub Pages demo branch added static demo and documentation
surfaces:

- [docs/user_stories.md](./docs/user_stories.md) was added to map demo visitor,
  requester, implementor, organization operator, agent operator, platform
  reviewer, and deferred stories.
- A GitHub Pages static site was added under `site/`.
- The Pages root serves the project landing page.
- `/demo/` serves an interactive static demo with localStorage-backed state.
- `/docs/` serves a documentation placeholder.
- The demo supports light and dark mode selection.
- The demo supports corporate, rustic, blocky, and showcase themes.
- The demo supports demo user selection for requester, implementor, organization
  reviewer, and agent operator perspectives.
- The demo includes mock Google, Apple, Microsoft, Facebook, and X.com sign-in
  buttons without implementing provider authentication.
- The demo includes a visible clear-state control.
- The demo maps requester creation, discovery, reservation, approval,
  submission, review, partial payout, tip, ban, REST instruction, and MCP
  instruction stories into one static workflow surface.
- A GitHub Actions Pages workflow was added to publish `site/` after pushes to
  `main` or manual dispatch.
- Playwright coverage was added for static demo theme switching, user switching,
  local state persistence, and state reset.
- A screenshot helper was added under `tools/` for repeatable demo screenshots.

The UI themes and GitHub Pages demo branch verification was performed:

- `make check-format` passed.
- `make check-ts` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `make test-deno` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make frontend`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-contracts check-dead-code lint vet`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui`
  passed with local Postgres access.
- Screenshot review was performed for
  `/tmp/sharecrop-screens/desktop-corporate-light.png`,
  `/tmp/sharecrop-screens/desktop-blocky-dark.png`,
  `/tmp/sharecrop-screens/mobile-rustic-light.png`, and
  `/tmp/sharecrop-screens/mobile-showcase-dark.png`.

The demo UI/UX repair branch refined the GitHub Pages demo:

- The demo was split into separate pages for overview, discovery, requester
  workflow, review queue, API/MCP instructions, and demo settings.
- The previous all-in-one scrolling demo was replaced with a focused app shell,
  page tabs, task tables, detail panels, and review panels.
- Demo login was moved into a discrete top-right account control that starts as
  Guest and opens a login panel.
- The login panel supports selecting a demo user and choosing mock Google,
  Apple, Microsoft, Facebook, and X.com provider buttons.
- The login panel closes after user selection, provider selection, or page
  navigation so it does not block other controls.
- Dark theme overrides were fixed for all visual themes.
- A demo audit helper was added to check deployed and local demo pages for
  console warnings, console errors, page errors, failed requests, and horizontal
  overflow.
- Playwright static demo coverage was updated for the separated navigation and
  login flow.

The demo UI/UX repair branch verification was performed:

- `make check-format` passed.
- `make check-ts` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `make test-deno` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-contracts check-dead-code lint vet`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui`
  passed with local Postgres access.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys tools/audit_demo_ui.ts`
  passed for deployed and local demo pages.
- Screenshot review was performed for
  `/tmp/sharecrop-screens/desktop-corporate-light.png` and
  `/tmp/sharecrop-screens/mobile-showcase-dark.png`.

The `task/demo-performance-flow-review` branch repaired the static demo after
performance and flow review:

- The demo runtime was changed from per-render event binding to document-level
  delegated event handling.
- Text input now updates in-memory state and debounces localStorage writes
  instead of re-rendering the page on every keystroke.
- Locally created demo tasks, reservations, and submissions are bounded so
  localStorage state cannot grow without limit.
- localStorage quota failures clear the demo state and return the user to
  settings.
- The expensive sticky translucent top bar with backdrop blur was replaced with
  an opaque non-sticky top bar.
- The top-level demo pages were kept separate for overview, discovery, requester
  create, review, API/MCP instructions, and settings.
- Demo clear-state controls were moved out of the header and into settings.
- Requester task creation now shows title, description, reward, visibility,
  participation policy, and reservation expiry fields.
- Discovery actions now match the selected task policy: reserve, request
  approval, submit, or no action.
- Review controls are per reservation and per submission, with approve, decline,
  release, request changes, reject, accept, partial payout, tip, and ban
  controls.
- API/MCP instructions separate worker REST, worker MCP, requester MCP, and
  credential setup placeholders.
- The static demo Playwright test covers theme selection, user selection, typed
  task creation, local state persistence, and clear state through the settings
  page.
- The screenshot capture helper targets exact page-tab labels and captures
  overview, discovery, create, review, API/MCP, settings, desktop theme, and
  mobile theme states.
- The Elm build helper rejects the recursive npm Elm wrapper when `ELM_BIN`
  points to it, preventing local builds from hanging and flooding Node warnings.

The `task/demo-game-like-personas` branch expanded the static demo into a
game-like mission board:

- The page labels became Command, Mission Board, Post Mission, Review Queue,
  Uplink, and Settings.
- The demo seed data grew from a few tasks to a larger mission set covering
  public and organization visibility, open submissions, reservations, requester
  approval, submitted work, changes requested, rejected work, accepted work,
  draft missions, funded missions, expired reservations, credit rewards,
  collectible rewards, and bundled rewards.
- The work-viewing surface changed from a table into mission lanes for
  Available, Reserved, Awaiting approval, Submitted, and Settled work.
- Mission cards now show rank, sector, policy, availability, and reward chips.
- Persona switching now changes the active persona, page, selected mission, and
  available actions.
- LocalStorage-backed state now includes activity feed entries, balances,
  collectible inventories, mission timelines, review drafts, local mission IDs,
  and mission state transitions.
- Requesters can locally draft, fund, open, and cancel missions.
- Implementors can locally reserve missions, request approval, submit payloads,
  and resubmit after changes are requested.
- Reviewers/requesters can locally approve or decline reservations, release
  reservations, request changes, reject with partial payout and tip, accept with
  payout and collectible transfer, and ban implementors from a mission.
- The Uplink page can simulate an agent run that creates an agent-labeled
  submission.
- Static demo Playwright coverage was expanded to verify persona switching,
  mission drafting persistence, and reserve-submit-accept transitions.
- Screenshot capture was updated for the new page labels and mission board
  screenshots.

The `task/demo-ui-polish-pass` branch polished the expanded static demo after
specialized review:

- The review queue became persona-scoped so implementors do not see or act on
  requester review work.
- Mission board and review actions now operate on the task shown in the current
  filtered view instead of stale global selection.
- Request-changes review decisions no longer pay partial payout or tip credits;
  payouts apply only to accepted and rejected decisions.
- Review inputs save without rerendering while the user is clicking a decision
  control, preventing lost review clicks.
- Settled, rejected, and changes-requested submissions now render as outcomes
  instead of still-active decision forms.
- Mission cards now show persona-specific next actions plus requester or
  assignee context, with wider lanes and clearer card hierarchy.
- Mission briefings now show the expected response schema in a readable block.
- Page tabs, persona buttons, theme buttons, mode buttons, and mission cards
  expose current or pressed state to assistive technology.
- Demo localStorage reads and writes are guarded, normalized, and bounded before
  stored state is merged into the seed demo state.
- Static demo Playwright coverage was expanded for persona-scoped review access
  and request-changes resubmission flow.

The `task/demo-ui-polish-pass` branch verification was performed:

- `node --check site/demo/app.js` passed. The
  `task/server-queues-revisions-parity` branch added queue, revision, parity,
  and raw-ID-audit work:

- Task listing gained a typed search filter and SQL matching on task title or
  ID.
- `/api/tasks` and `/api/teams/{team_id}/work` now reject malformed task-list
  pagination instead of silently falling back to default paging.
- Organization task queues and team work queues use server-side search and
  pagination from Elm.
- Organization task state filters are server-backed; team work-type filters
  continue to apply to the loaded page.
- The revision inbox gained a `Revise` shortcut that opens the task detail with
  the previous response JSON prefilled.
- Shared scenario parity covers organization queue search and team work queue
  search.
- HTTP E2E coverage was added for task-list search across user, organization,
  and team work scopes.
- Contract fixtures cover an organization-team task-list item response.
- The backendless demo mirrors organization-team task visibility, queue search,
  and queue pagination.
- The browser raw-ID audit found no remaining confirmed high-traffic raw-ID
  input; IDs remain in links, protocol/API/MCP examples, audit/metadata rows,
  and copyable integration snippets.

The `task/server-queues-revisions-parity` branch verification was performed:

- `ELM_BIN=/opt/homebrew/bin/elm deno task frontend:build` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build go test ./...`
  passed.
- `deno test --allow-read tests/deno` passed.
- `deno check tools/*.ts tests/**/*.ts` passed.
- `deno lint tools tests` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build make check-contracts`
  passed.
- `make check-format` passed.
- `deno run --allow-read tools/check_policy.ts` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build go vet ./...`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build go tool deadcode -test ./...`
  passed.
- `deno run -A npm:jscpd@5.0.11 site/demo internal cmd tools web/elm/src tests`
  passed.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys npm:@playwright/test@1.61.0 test -c tests/playwright/playwright.config.ts --no-deps --output=/Users/zardoz/projects/sharecrop/test-results tests/playwright/demo.spec.ts tests/playwright/mobile.spec.ts`
  passed against the already-running demo server.
- `go test -tags http_e2e ./tests/http_e2e` was attempted locally but did not
  run because `DATABASE_URL` was not set and the sandbox blocked `httptest`
  network binding.
- Local real-app Playwright was not run because PostgreSQL was not reachable at
  `localhost:15432`.

The `task/post81-dashboards-revisions-parity` branch added post-PR81 dashboard,
revision, parity, and onboarding work:

- Team work, organization task, requester task, and discovery lists gained
  loaded-list search/filter controls.
- Worker submission pages gained a revision inbox for submissions in
  `changes_requested`.
- Team/organization work and collectible load failures now surface
  section-specific messages instead of being hidden by unrelated successful
  loads.
- Shared scenario parity now asserts `submission_commented` notification task
  metadata.
- HTTP contract fixtures cover the `submission_commented` notification list
  shape.
- Browser tests cover the new list search controls, team/org dashboard controls,
  and revision inbox flow.
- [docs/onboarding.md](./docs/onboarding.md) was added and linked from README
  and hosted docs.
- The readiness review was updated for the new dashboard, revision, and
  onboarding surfaces.

The `task/post81-dashboards-revisions-parity` branch verification was performed:

- `ELM_BIN=/opt/homebrew/bin/elm deno task frontend:build` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build go test ./...`
  passed.
- `deno test --allow-read tests/deno` passed.
- `deno check tools/*.ts tests/**/*.ts` passed.
- `deno lint tools tests` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build make check-contracts`
  passed.
- `make check-format` passed.
- `deno run --allow-read tools/check_policy.ts` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build go vet ./...`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build go tool deadcode -test ./...`
  passed.
- `deno run -A npm:jscpd@5.0.11 site/demo internal cmd tools web/elm/src tests`
  passed.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys npm:@playwright/test@1.61.0 test -c tests/playwright/playwright.config.ts --no-deps --output=/Users/zardoz/projects/sharecrop/test-results tests/playwright/demo.spec.ts tests/playwright/mobile.spec.ts`
  passed against the already-running demo server.
- Local real-app Playwright was not run because PostgreSQL was not reachable at
  `localhost:15432`.

- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys npm:@playwright/test@1.61.0 test -c tests/playwright/playwright.config.ts tests/playwright/demo_static.spec.ts`
  passed.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys tools/capture_demo_screenshots.ts`
  captured desktop and mobile screenshots; screenshots reviewed included the
  Mission Board, Review Queue, Command desktop, and showcase mobile states.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys tools/audit_demo_ui.ts`
  passed for deployed and local demo pages.
- `make check-format` passed.
- `make check-ts` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `make test-deno` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-contracts check-dead-code lint vet`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui`
  passed.

The `task/demo-list-detail-navigation` branch changed the static demo discovery
model:

- The top-level demo labels became Dashboard, Tasks, Post Task, Reviews,
  Agent/API, and Settings.
- The Tasks page stopped using the board-like lane abstraction and now renders a
  scannable task list.
- Task rows show title, objective, rank, sector, policy, requester or assignee
  context, status, next action, reward chips, and row-level action buttons.
- Clicking a task title or Open task page navigates to a separate Task Detail
  page instead of replacing a detail pane on the list page.
- Task Detail owns the full briefing, response schema, action console, task log,
  back-to-list control, and API/MCP handoff link.
- Requester rows without pending review work no longer show a Review queue call
  to action.
- Demo Playwright coverage was updated for task-list navigation, task-detail
  actions, and renamed navigation labels.
- Screenshot capture was updated for task-list and task-detail desktop and
  mobile states.

The `task/demo-list-detail-navigation` branch verification was performed:

- `node --check site/demo/app.js` passed.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys npm:@playwright/test@1.61.0 test -c tests/playwright/playwright.config.ts tests/playwright/demo_static.spec.ts`
  passed.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys tools/capture_demo_screenshots.ts`
  captured screenshots; reviewed images included desktop task list, desktop task
  detail, mobile task detail, and mobile dashboard states.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys tools/audit_demo_ui.ts`
  passed for deployed and local demo pages.
- `make check-format` passed.
- `make check-ts` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `make test-deno` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-contracts check-dead-code lint vet`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui`
  passed.

The `task/demo-real-parity-fixtures` branch tightened backendless-demo parity
with the real app:

- The demo backend exposed a test hook with its route table and resolver.
- Deno route coverage compared demo `/api` routes with the real HTTP router and
  allowed only health, MCP, and root/static serving as real-only routes.
- Deno response-shape coverage exercised demo account lifecycle, user directory,
  task list/detail, collectible minting, create-time collectible reward escrow,
  and unknown-route behavior.
- The demo backend implemented account lifecycle routes, `/api/users`,
  email-backed organization/team member provisioning, profile/password/account
  responses, and clear 404 errors for unimplemented API routes.
- Demo task creation now honors `reward.collectible_ids` by escrowing selected
  collectibles and returning the held collectible count, matching the real
  create-time reward flow.
- Shared Playwright scenario constants were added for account lifecycle and
  selector-backed reward creation specs.

The `task/demo-real-parity-fixtures` branch verification was performed:

- `deno task check:ts` passed.
- `deno task lint` passed.
- `deno task test` passed.
- `ELM_BIN=/opt/homebrew/bin/elm deno task frontend:build` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build go test ./internal/http ./internal/auth ./internal/task ./internal/ledger ./internal/db`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 go test -tags http_e2e ./tests/http_e2e`
  passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys npm:@playwright/test@1.61.0 test -c tests/playwright/playwright.config.ts tests/playwright/demo.spec.ts tests/playwright/account.spec.ts tests/playwright/create-task-selectors.spec.ts`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build ELM_BIN=/opt/homebrew/bin/elm make check-format check-contracts check-policy check-ts lint vet test-deno frontend build`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build go tool deadcode -test ./...`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build go test ./...`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_BIN=/opt/homebrew/bin/elm make build`
  passed.

The `task/product-readiness-foundation` branch added a combined readiness
foundation:

- Operations docs now include a runbook with required configuration, migration
  procedure, backup/restore guidance, log-token delivery mode, admin operations
  status, and current multi-process limits.
- A systemd service template was added for one-process deployments.
- The database has operations-state foundation tables for audit events,
  rate-limit buckets, and MCP HTTP sessions.
- Platform admins can read `/api/admin/operations` for account-token delivery
  mode, secure-cookie mode, active MCP session count, active rate-limit buckets,
  and current runtime storage mode.
- Account verification and password reset token issue support
  `SHARECROP_ACCOUNT_TOKEN_DELIVERY=log`; local/test API-token mode remains the
  default.
- Account deactivation removes password credentials, revokes active
  refresh/account tokens, anonymizes the stored email, and keeps referenced user
  rows for task/submission/ledger history.
- The create-task user selector can query `/api/users?query=...`; Playwright
  coverage exercises the search path before selecting a recipient.
- HTTP contract fixtures were expanded for account-token sent responses, user
  directory responses, submission comments, and operations status.
- Submission comment posting now uses a real submit button.
- Product docs were updated to keep rewards scoped to Sharecrop credits and
  admin-minted Sharecrop collectibles only.

The `task/product-readiness-foundation` branch verification was performed:

- `deno task check:ts` passed.
- `deno task lint` passed.
- `deno task test` passed.
- `ELM_BIN=/opt/homebrew/bin/elm deno task frontend:build` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build go test ./internal/http ./internal/auth ./internal/app`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 go test -tags http_e2e ./tests/http_e2e`
  passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys npm:@playwright/test@1.61.0 test -c tests/playwright/playwright.config.ts tests/playwright/account.spec.ts tests/playwright/create-task-selectors.spec.ts`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build ELM_BIN=/opt/homebrew/bin/elm make check-format check-contracts check-policy check-ts lint vet test-deno frontend build`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build go test ./...`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build go tool deadcode -test ./...`
  passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.cache/go-build GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_BIN=/opt/homebrew/bin/elm make build`
  passed.

The `task/runtime-audit-team-dashboard` branch added runtime state, audit, and
team dashboard work:

- Production `serve` now wires Postgres-backed rate-limit buckets.
- Production `serve` now persists MCP HTTP session identity, TTL admission
  state, close state, and active counts in Postgres. Live SSE subscribers and
  replay buffers remain process-local and operations reports
  `postgres_session_process_stream`.
- Runtime construction fails loudly when explicit production runtime
  dependencies are missing.
- MCP raw response encoding no longer synthesizes a fallback response if JSON
  encoding fails; it now fails loudly.
- Audit events were added with a Postgres store and admin list endpoint.
- Audit writes now cover admin default collectible awards, account deactivation,
  organization member provisioning, organization member role updates,
  organization member deactivation, submission accept/request-changes/reject
  decisions, and task refunds.
- The Elm app gained an admin page that shows operations status and audit events
  for platform admins.
- Team detail pages now load `/api/teams/{team_id}/work`, show a review queue,
  and show team work.
- Task listing supports `scope=team`.
- Generated Elm contracts now include Admin operations/audit response types.
- The backendless demo implements team work and admin audit routes, serves the
  current compiled Elm bundle, and computes a `/demo/` base path explicitly.

The `task/runtime-audit-team-dashboard` branch verification was performed:

- `go test ./...` passed.
- `make check-format` passed.
- `make check-contracts` passed.
- `deno task check:policy` passed.
- `deno check tools/*.ts tests/**/*.ts` passed.
- `deno task lint` passed.
- `deno test --allow-read tests/deno` passed.
- `go vet ./...` passed.
- `go tool deadcode -test ./...` passed.
- `ELM_BIN=/opt/homebrew/bin/elm make build` passed.
- `deno task e2e:ui` passed with local Postgres.
- Local Playwright screenshot/overflow checks passed for admin desktop/mobile
  and team desktop/mobile.
- `DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations go test -tags integration ./tests/integration`
  passed.
- `DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 go test -tags http_e2e ./tests/http_e2e`
  passed.
# Shauth OpenID Connect sign-in

Sharecrop accepted Shauth as an additional browser identity provider while
preserving local password sign-in and first-party rotating sessions. The server
used discovery, authorization-code exchange with PKCE, nonce and state checks,
signature/audience validation, and an authenticated short-lived transaction
cookie. It persisted only the verified issuer/subject relationship, rejected
implicit linking to existing password accounts by email, and returned the
browser to the normal refresh-cookie session bootstrap. The ECS deployment
accepted the confidential-client coordinates from AWS Secrets Manager and
validated complete HTTPS configuration before startup. The database and WASI
bridge carried external identities through the same production path as native
hosting.
