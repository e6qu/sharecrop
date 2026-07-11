# Status

All work through pull request 146 is merged into `main`. PR 141 landed the
review-driven fixes, the two-section credit wallet, and Argon2id hashing;
PR 142 exposed per-task funding; PR 143 verified the deployed demo and
corrected the "two backends" framing; PR 144 landed the WASI hosting spike
Phase 2 (one store method bridged from a wasip1 guest to Postgres); PR 145
split the reward-return action into owner "Reclaim" vs worker "Refund" with
an info toggle; PR 146 landed WASI spike Phase 3 (codegen the audit-store
bridge with a CI drift gate + dual-run tests).

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

Active task: `task/wasi-spike-phase4` — Phase 4, the **final** phase of the
WASI production hosting spike (see
[docs/wasi_production_hosting_spike_plan.md](./docs/wasi_production_hosting_spike_plan.md)):
one real HTTP request end to end through the guest.
`cmd/sharecrop-wasi-http-host` runs a real `net/http.Server` that handles
each request by running a fresh wasip1 guest
(`cmd/sharecrop-wasi-http-guest`) over the Phase 3 `rpc` transport; the guest
runs the **real production mux** (`httpserver.New(...)`, the same one
`cmd/sharecrop serve` builds) through an `httptest` recorder. Request/response
marshaling is `internal/wasibridge/httpbridge`. The integration test
(`tests/integration/httpbridge_test.go`) asserts byte-identical responses
(status, Content-Type, body) vs the in-process mux for `GET /healthz` (200)
and an unknown `/api` route (404); a real `curl` confirms it. The route
touches no store, so no DB bridge is needed for this slice.

**The spike is complete** — all four phases verified. The follow-up is the
full implementation effort (its own workstream, not part of the spike):
bridge the remaining stores, wire real services in the guest, cover the full
route surface, weigh instance pooling, migrate `cmd/sharecrop`, and retire
`internal/wasmdemo`. Nothing about the native server or browser demo changes.

Blocking issues: none. GitHub Pages `deploy-pages` occasionally fails
transiently after a merge and clears on retry; it is not caused by
repository code.
