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

Active task: `task/wasi-bridge-org` — bridge the `org` store (organizations,
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
