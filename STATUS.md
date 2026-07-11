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

Active task: `task/wasi-auth-route` — prove an auth-store-touching route end
to end through the guest. `appmux` now wires a **live auth service** (backed by
the bridged auth `GuestStore`) alongside the notification service, so
`GET /api/users` reads the auth store's directory through the guest,
byte-identical to native (`tests/integration/authroute_test.go`) - the first
route to exercise a bridged store's *service*, not just stateless token
verification. Host-side store routing (audit/auth/notification by method
prefix) is shared as `internal/wasibridge/storehost`, used by the app host and
the tests. Three stores bridged: audit, notification, auth. **Next**: bridge
the remaining stores (ledger, task, org, submission, assets, orgcred) and wire
their services/routes; then weigh the ~2-3ms instance-per-request floor
against instance pooling and migrate `cmd/sharecrop` onto the hosted guest.
Nothing about the native server or browser demo changes.

Blocking issues: none. GitHub Pages `deploy-pages` occasionally fails
transiently after a merge and clears on retry; it is not caused by
repository code.
