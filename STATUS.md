# Status

All work through pull request 147 is merged into `main`. PR 144/146/147 were
the WASI hosting spike (Phase 2: one store method bridged from a wasip1 guest;
Phase 3: codegen the audit-store bridge with a CI drift gate + dual-run;
Phase 4: the real `internal/http` mux running in a wasip1 guest behind a
native `net/http.Server`, byte-identical). PR 145 split the reward-return
action into owner "Reclaim" vs worker "Refund"; 141-143 landed the wallet,
Argon2id, per-task funding, and the deployed-demo verification.

Implemented surface:

- Go HTTP API (`internal/http`) over Postgres-backed domain services, an Elm
  browser client, an MCP interface at `/mcp` (Streamable HTTP with SSE
  replay), scoped agent and organization-wide credentials, and a generated
  OpenAPI document (`docs/openapi.json`).
- There is one backend: the `internal/http` mux plus the domain services
  (`internal/task`, `internal/ledger`, `internal/assets`, ...). It is shared
  source, not duplicated. The only per-deployment difference is the **storage
  adapter** each domain `Store` interface is bound to:
  - the server (`cmd/sharecrop serve`) binds the Postgres adapters in
    `internal/db` (pgx, SQL, `FOR UPDATE` locks — safe for concurrent,
    multi-replica access);
  - the browser demo (`cmd/sharecrop-wasm`, compiled to `js/wasm`) binds the
    browser key/value adapters in `internal/wasmdemo` (localStorage, single
    effective user).
  A browser cannot open a Postgres connection, so a browser-local storage
  adapter is unavoidable for the demo; the two adapters are kept behaviorally
  identical by the shared scenario-parity suite. The stated goal (tracked in
  `DO_NEXT.md`) is to run production on the *same compiled WASM artifact* as
  the demo via a WASI host (a native process embedding a WASM runtime that
  owns Postgres + networking and bridges storage to `internal/db`); today
  production still runs a native compile of the same source, which is the
  remaining deviation. Architecture details:
  [docs/wasm_demo_backend_spike.md](./docs/wasm_demo_backend_spike.md),
  [docs/wasi_production_hosting_spike_plan.md](./docs/wasi_production_hosting_spike_plan.md).
- PR 138 added the browser storage adapters; PR 139 cut the demo over to the
  real mux + domain services; PR 140 audited authorization, storage-adapter
  parity, and the Elm client.

Test status: PR CI runs format/contract/policy/type checks, Go unit and
integration tests, HTTP end-to-end tests, shared scenario parity against
both storage adapters (the Postgres store and the compiled-WASM browser
store), and Playwright browser tests. All green on `main`.

The deployed GitHub Pages demo (https://e6qu.github.io/sharecrop) was
verified end to end after the #142 merge: it boots to spendable 70 /
allocated 30, the ledger shows "Task funding", the funded task shows
"Allocated to this task: 30 credits" with the Refund button gated correctly,
inbox and collectibles are populated, and there are no console errors.

The WASI hosting **spike is complete** (all four phases; see
[docs/wasi_production_hosting_spike_plan.md](./docs/wasi_production_hosting_spike_plan.md)).
The follow-up **implementation effort** has started.

Active task: `task/wasi-bridge-privacy` — bridge the **privacy** RuntimeState
service (fourth and last codegen-friendly infra service; 6 methods, 3 result
unions). `RecordSensitiveFieldAccess` takes a `submission.Submission`
(→ `extraImports`), but the store reads only the submission's ID and each
sensitive field's Path, so the wire carries a minimal submission (documented).
`Resolve` takes two strings → generator disambiguation. `appmux.Stores` gained a
`Privacy` field; dual-run-verified (create/list/resolve/record-access/retention)
and route tests pass; full integration suite run locally (no contamination).
**Four of six infra services bridged.** Remaining: the two that don't fit the
codegen — **rate limiter** (`Allow(key) bool`, no ctx) and **MCP session
persistence** (multi-return tuples) — as hand-written bridges or host-side, then
the production cutover of `cmd/sharecrop serve`. Nothing about the native server
or browser demo changes.

Earlier: `task/wasi-bridge-moderationtriage` bridged the **moderation-triage**
RuntimeState service (third of six infra services). Its `RecordOpen` takes an
`audit.Event` (a third package → `extraImports`), but both the memory and db
stores read only the event's `ID` and `CreatedAt`, so the wire carries just those
two and rebuilds a minimal event (documented; the dual-run test would catch it if
`RecordOpen` grew to read more). `Update` takes two strings (state, note) →
generator disambiguation. `appmux.Stores` gained a `ModerationTriage` field;
dual-run-verified (record-open/list/update) and route tests pass. Three of six
infra services bridged (saved-queue-view, platform-admin, moderation-triage).
**Remaining**: privacy (codegen-friendly but references `submission.Submission`),
then the two non-codegen ones (rate limiter, MCP sessions), then the cutover.
Nothing about the native server or browser demo changes.

Earlier: `task/wasi-bridge-platformadmin` bridged the **platform-admin**
RuntimeState service (second of six infra services; codegen-friendly). Its `Grant`
takes two `core.UserID` args, which the generator's repeated-arg disambiguation
handles (`UserID`/`UserID2`). One wrinkle: the db store takes bootstrap admins, so
`storehost` now parses `SHARECROP_ADMIN_USER_IDS` (host-side config) to seed
`db.NewPlatformAdminStore`. `appmux.Stores` gained a `PlatformAdmins` field
overriding the in-memory default; dual-run-verified (grant/is-admin/list) and the
full-graph route tests pass with it wired in. **Two codegen-friendly infra
services remain (privacy, moderation-triage - each needs a shared cross-package
value codec: submission.Submission / audit.Event, likely extracted like
attachmentwire), then the two non-codegen ones (rate limiter, MCP sessions), then
the production cutover.** Nothing about the native server or browser demo changes.

Earlier: `task/wasi-bridge-savedqueueview` bridged the first of the six
RuntimeState infra services so the pooled mux shares one Postgres store instead
of a per-instance in-memory copy. `cmd/sharecrop serve` backs these six (rate
limiters, MCP sessions, saved queue views, privacy, platform admins, moderation
triage) with db stores; `appmux` had them all in-memory, which under pooling
means per-instance state — a regression. This bridges `SavedQueueViewService`
(the codegen now targets an `internal/http` interface, package `httpserver`; the
db store already implements it, so the bridge just carries it to the host).
`appmux.Stores` gained a `SavedQueueViews` field, overriding the in-memory
default; `savedqueueviewbridge` is dual-run-verified
(`tests/integration/savedqueueviewbridge_store_test.go`) and route-verified
(`GET /api/saved-queue-views` byte-identical through the guest). The route-test
harness was de-duplicated into shared `serveRouteBothWays` +
`assertBridgeMatchesNative` helpers. **Next**: bridge the other codegen-friendly
infra services (privacy, platform admin, moderation triage), then the two that
don't fit the codegen (rate limiter - no ctx/bool returns; MCP session
persistence - multi-return tuples) as hand-written bridges or host-side, then the
production cutover of `cmd/sharecrop serve`. Nothing about the native server or
browser demo changes.

Earlier: `task/wasi-instance-pool` added **instance pooling** for the WASI app
host. The guest was a wasip1 *command* (ran `main()` once per unit of work and
exited), so each HTTP request paid the ~2-3ms guest-startup floor. Command
instances can't be reused, and the Phase-1 findings proved a *shared* reactor
instance corrupts state. The viable design, now implemented: the guest's
`main()` **loops** over units of work read from stdin (via new `rpc.Serve`),
staying alive between them, and the host keeps a **pool** of such instances
(`rpc.Pool`), checking one out per unit of work. Each instance is still driven by
exactly one goroutine and touched by no other - a per-instance `session` has a
runner goroutine (owns the wazero instance) and a driver goroutine (owns both
pipe ends, so the two-write framing never interleaves); request goroutines reach
the driver only over Go channels. So pooling adds no shared-instance concurrency;
concurrency comes from having several sessions. The unit-of-work protocol moved
from argv to stdin "work" frames; `Host.Call` (fresh instance per call, used by
all the dual-run/route tests) and `Pool.Call` both sit on the same `session`.
`httpbridge.Handler` now takes an `rpc.Caller` (either), and the production
`cmd/sharecrop-wasi-app-host` uses a pool sized by `SHARECROP_WASI_POOL_SIZE`
(default GOMAXPROCS). Two new tests prove no cross-talk under load - 144
concurrent store units through 4 reused instances, and 16 concurrent HTTP
requests through the pooled app host - each seeing only its own data. **Next**:
move `cmd/sharecrop serve` itself onto the WASI host (the production cutover).
Nothing about the native server or browser demo changes.

Earlier: `task/wasi-appmux-full-graph` wired the **full production mux** into the
WASI app guest. `internal/wasibridge/appmux`
grew from the auth+notification slice to the complete domain-service graph (auth,
notification, org, task, submission, ledger, agent, orgcred, assets, audit),
built in the same dependency order `cmd/sharecrop serve` uses (org+agent feed
task; the shared task store + org service feed submission; no adapter types
needed - services satisfy each other's cross-interfaces directly). The
RuntimeState services without a dedicated store (rate limiters, MCP sessions,
saved queue views, privacy, platform admins, moderation triage) keep their
in-memory defaults; audit + notification run over bridged stores. `appmux.New`
now takes an `appmux.Stores` struct (ten store interfaces), so the guest
(`cmd/sharecrop-wasi-app-guest`) passes bridge GuestStores and the route tests
pass real db stores - identical mux either way. Three routes are dual-run-
verified byte-identical to native through the full-graph guest:
`GET /api/notifications`, `GET /api/users`, and now `GET /api/credits/balance`
(ledger service, returns the real signup-grant balance). **Next**: weigh
instance pooling vs the ~2-3ms fresh-instance-per-request floor, then move
`cmd/sharecrop serve` onto the WASI host. Nothing about the native server or
browser demo changes.

Earlier: `task/wasi-bridge-task` bridged the `task` store, the **last and
widest store** (21 methods: tasks, series, reservations, and task/series comment
threads). The `Task` model alone carries ~10 nested unions (owner, reward spec,
visibility, series placement, data payload, assignee, active assignee, and the
filter/scope unions). `internal/wasibridge/taskbridge` (codecs across codec.go /
models_codec.go / commands_codec.go / results_codec.go + generated
`bridge_gen.go`) is dual-run-verified against real Postgres
(`tests/integration/taskbridge_store_test.go`) across create/find/change-state/
list/reserve/comment/series/attach-to-series/series-comment. It needed **no
generator change**. `corewire` gained `TaskSeriesID`/`TaskReservationID`/
`SeriesCommentID`/`TaskCommentID` codecs. The attachment codec, previously
duplicated in `submissionbridge`, was extracted into a shared
`internal/wasibridge/attachmentwire` package (both bridges now use it); shared
task test builders + diff helpers live in `internal/task/tasktest`. The generic
store guest and `storehost` route `task.*` too. **ALL TEN stores are now
bridged: audit, notification, auth, agent, orgcred, assets, submission, ledger,
org, task.** The store-bridging phase is COMPLETE. **Next**: the host wiring -
weigh instance pooling against the ~2-3ms instance-per-request floor, then
migrate `cmd/sharecrop serve` onto the WASI host. Nothing about the native
server or browser demo changes.

Earlier: `task/wasi-bridge-org` bridged the `org` store (organizations,
members, and teams: 16 methods, including the `TeamOwner` tagged union -
organization-owned vs standalone user-owned teams). `internal/wasibridge/orgbridge`
(codecs + generated `bridge_gen.go`) is dual-run-verified against real Postgres
(`tests/integration/orgbridge_store_test.go`) across create-org / provision /
update-roles / deactivate / create-team / add-member and every read path. It
drove a third **generator enhancement**: `ProvisionMember(..., auth.EmailAddress,
...)` is the first method with an argument whose type lives in a *third* package,
so `storeSpec` gained an `extraImports` field and the generated file now imports
`auth` (backward-compatible - every other spec leaves `extraImports` nil, so
their generated files are unchanged). `corewire` gained `TeamID` and
`OrganizationMembershipID` codecs; no new reconstruction constructors were needed
(org value types round-trip through their existing validating constructors).
Shared test builders live in `internal/org/orgtest`. The generic store guest and
`storehost` route `org.*` too. **Nine stores bridged: audit, notification, auth,
agent, orgcred, assets, submission, ledger, org.** **Next**: the last store,
`task` (~20 methods); then weigh instance pooling and migrate `cmd/sharecrop`.
Nothing about the native server or browser demo changes.

Earlier: `task/wasi-bridge-ledger` bridged the `ledger` store (the store
with the deepest unions: fund/accept/request-changes/reject/refund commands and
balance/allocated/entries reads; accept and reject commands carry nested
credit/tip/collectible/ban *selection* unions, and their results carry nested
*payout* and *tip outcome* unions). `internal/wasibridge/ledgerbridge` (codecs +
generated `bridge_gen.go`) is dual-run-verified against real Postgres with a
full **fund -> accept -> refund** flow through the wasip1 guest
(`tests/integration/ledgerbridge_store_test.go`), plus exhaustive codec
round-trip tests for every payout/tip/selection variant. It needed **no
generator change** (it fit the existing framework). `corewire` gained
`LedgerEntryID` and `CreditAccountID` codecs; no new reconstruction constructors
were needed because every ledger value type round-trips through its existing
validating constructor (amounts are always positive, signed amounts non-zero,
keys non-empty). The generic store guest and `storehost` route `ledger.*` too.
**Eight stores bridged: audit, notification, auth, agent, orgcred, assets,
submission, ledger.** **Next**: bridge the last two (task ~20 methods, org ~15);
then weigh instance pooling and migrate `cmd/sharecrop`. Nothing about the
native server or browser demo changes.

Earlier: `task/wasi-bridge-submission` bridged the `submission` store
(the widest so far: submissions with attachments, validation outcomes, and
sensitive fields; the receipt-token lookup; per-task and per-submitter lists;
and the submission comment thread). `internal/wasibridge/submissionbridge`
(codecs + generated `bridge_gen.go`) is dual-run-verified against real Postgres
(`tests/integration/submissionbridge_store_test.go`). It drove a second
**generator enhancement**: `CreateSubmission(..., []SensitiveField)` takes a
slice of a package-local type, so `qualify` now qualifies the slice *element*
(`[]submission.SensitiveField`, not the meaningless `submission.[]SensitiveField`)
- backward-compatible (no existing method has a slice-of-local-type arg).
`corewire` gained `SubmissionID`/`SubmissionReceiptTokenID`/`SubmissionCommentID`
codecs; `submission` gained a `ReceiptTokenHashFromString` reconstruction
constructor (the opaque hash has no other from-string path); the submission and
comment comparison helpers live in `internal/submission/submissiontest`. The
generic store guest and `storehost` route `submission.*` too. **Seven stores
bridged: audit, notification, auth, agent, orgcred, assets, submission.**
**Next**: bridge the remaining three (ledger - the hardest, nested payout/tip/
selection unions; task ~20 methods; org ~15); then weigh instance pooling and
migrate `cmd/sharecrop`. Nothing about the native server or browser demo changes.

Earlier: `task/wasi-bridge-assets` bridged the `assets` store
(collectibles: create/list/list-by-owner/fund/refund/gift/award/task-held),
driving the repeated-same-type-argument generator enhancement (`Query`/`Query2`).

Earlier: `task/wasi-bridge-orgcred` bridged the `orgcred` store
(organization-wide credentials), and extract the agent value-type codecs
(`Label`/`ScopeSet`/`State` + the shared `CreateStoreResult`) into a new
`internal/wasibridge/agentwire` package so agentbridge and orgcredbridge don't
duplicate them (jscpd would block identical codecs on the same types). The agent
bridge was refactored onto `agentwire` (regenerated, no behavior change).
`internal/wasibridge/orgcredbridge` is dual-run-verified against real Postgres
(`tests/integration/orgcredbridge_store_test.go`); shared test-comparison
helpers live in `internal/agent/agenttest.SharedFieldsDiff`. `corewire` gained
`OrgCredentialID` + nullable-`*time.Time` codecs. **Five stores bridged: audit,
notification, auth, agent, orgcred.** **Next**: bridge the remaining stores
(submission, assets, then the big ones - ledger/task/org); then weigh instance
pooling and migrate `cmd/sharecrop`. Nothing about the native server or browser
demo changes.

Blocking issues: none. GitHub Pages `deploy-pages` occasionally fails
transiently after a merge and clears on retry; it is not caused by
repository code.
