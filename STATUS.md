# Status

All work through pull request 145 is merged into `main`. PR 141 landed the
review-driven fixes, the two-section credit wallet, and Argon2id hashing;
PR 142 exposed per-task funding; PR 143 verified the deployed demo and
corrected the "two backends" framing; PR 144 landed the WASI hosting spike
Phase 2 (`internal/wasibridge`: one real store method bridged from a wasip1
guest to Postgres); PR 145 split the reward-return action into owner
"Reclaim" vs worker "Refund" with an info toggle explaining each.

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

Active task: `task/wasi-spike-phase3` — Phase 3 of the WASI production
hosting spike (see
[docs/wasi_production_hosting_spike_plan.md](./docs/wasi_production_hosting_spike_plan.md)):
codegen the bridge for one full store, with a CI drift gate and dual-run
tests. The generic transport `internal/wasibridge/rpc` (method-keyed calls
over the Phase 1 pipe shape) now underpins both the Phase 2 auth spike and
Phase 3, sharing `internal/wasibridge/wire` (framing) and
`.../domainwire` (the shared `DomainError` codec).
`internal/wasibridge/auditbridge` bridges the whole `internal/audit.Store`
(Record/Get/List) with hand-written, tested codecs plus a **generated**
`bridge_gen.go` (`Dispatch` + `GuestStore`) emitted by
`go run ./cmd/sharecrop generate wasi-bridge`. The `check-wasi-bridge` gate
regenerates and diffs (like `check-openapi`). The dual-run integration test
(`tests/integration/auditbridge_store_test.go`) runs every method against
real Postgres through both the direct-db path and the compiled wasip1
guest + host bridge and asserts they match; the generator errors loudly on
an unregistered type (`gen_test.go`). Nothing about the native server or
browser demo changes. Next: Phase 4 (one real HTTP request end to end
through the guest).

Blocking issues: none. GitHub Pages `deploy-pages` occasionally fails
transiently after a merge and clears on retry; it is not caused by
repository code.
