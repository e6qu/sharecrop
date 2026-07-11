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

Active task: `task/wasi-bridge-multistore` — generalize the bridge codegen
beyond one store and prove it scales. `internal/wasibridge/gen` is now
store-agnostic (each store is a `storeSpec`; `generate wasi-bridge` iterates
`gen.Targets()`), shared core-type codecs (ids, page, time) moved to
`internal/wasibridge/corewire`, and `internal/notification.Store` is bridged
as the second store (`.../notificationbridge`), dual-run-verified against real
Postgres. One generic guest (`cmd/sharecrop-wasi-store-guest`) routes every
store by method prefix. The audit bridge was regenerated onto `corewire` with
no behavior change. Remaining stores (auth, ledger, task, org, submission,
assets, orgcred) follow the same recipe: a spec + hand-written codecs + a
dual-run test. Nothing about the native server or browser demo changes.

Blocking issues: none. GitHub Pages `deploy-pages` occasionally fails
transiently after a merge and clears on retry; it is not caused by
repository code.
