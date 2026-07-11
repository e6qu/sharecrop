# Do Next

Current priority from
[docs/application_readiness_review.md](./docs/application_readiness_review.md):

1. Two large infrastructure efforts are wanted but explicitly deferred:

   - (a) WASI production hosting: replace `cmd/sharecrop`'s native
     `net/http` server with a WASI-hosted WASM binary compiled from the real
     backend (mainline Go, not TinyGo/Spin — TinyGo's `encoding/json`
     support is unreliable and this codebase uses `encoding/json`
     pervasively). The browser-demo half of the original duplication problem
     is already solved: since PR 139 the demo runs the real `internal/http`
     mux and real domain services compiled to `js/wasm` over
     browser-storage-backed stores (PR 138). The spike plan is
     [docs/wasi_production_hosting_spike_plan.md](./docs/wasi_production_hosting_spike_plan.md),
    **complete** — all four phases verified. Phase 2 bridged one auth method;
     Phase 3 added the generic transport (`internal/wasibridge/rpc`), shared
     framing/`DomainError` codecs (`.../wire`, `.../domainwire`), and codegen
     for a full store (`.../auditbridge` + `generate wasi-bridge` +
     `check-wasi-bridge` + dual-run); Phase 4 ran the real `internal/http` mux
     inside a wasip1 guest behind a native `net/http.Server`
     (`cmd/sharecrop-wasi-http-{host,guest}`, `.../httpbridge`) with
     byte-identical output for `GET /healthz`. The go/no-go is **go**; see the
     "After Phase 4" note in the plan.

     **The follow-up implementation effort has started.** The bridge codegen
     is now generalized to N stores (`gen` is store-agnostic via `storeSpec` +
     `gen.Targets()`; shared core codecs in `.../corewire`), and
     `internal/notification.Store` is bridged as the second store
     (`.../notificationbridge`, dual-run-verified), served by one generic guest
     (`cmd/sharecrop-wasi-store-guest`). **Next**: bridge the remaining stores
     the same way (a `storeSpec` + hand-written codecs + a dual-run test each) —
     `auth` (needed by almost every route, ~13 methods), then `ledger`, `task`,
     `org`, `submission`, `assets`, `orgcred`. After enough stores are bridged,
     wire the real domain services in the guest against those `GuestStore`s and
     prove a store-touching HTTP route end to end (tying Phase 3 + 4 together),
     then weigh the ~2-3ms instance-per-request floor against instance pooling,
     migrate `cmd/sharecrop` onto the hosted guest, and retire
     `internal/wasmdemo` once the browser demo can run the same artifact.
   - (b) Moving MCP/SSE to HTTP/2 by default (HTTP/3-ready) to support about
     100 concurrent streaming sessions, keeping HTTP/1.1 as an explicit,
     supported option for regular UI/API traffic.

2. Keep expanding shared scenario parity as new user-visible API surfaces
   are added, and keep running it against real APIs as behavior changes. The
   explicit-session runner accepts `--origin`, access-token input, and
   refresh-token input; the local real runner can register a scenario admin
   and grant platform-admin state through `DATABASE_URL` and `psql`.

3. Keep expanding generated/fixture-level HTTP contract coverage as the API
   surface grows.

4. Audit remaining raw-ID browser flows and replace high-traffic fields with
   selectors where directory data exists. No confirmed high-traffic raw-ID
   input remains after the latest audit in
   [docs/raw_id_browser_flow_audit.md](./docs/raw_id_browser_flow_audit.md).

5. Build a genuine production non-browser WASM host if a non-browser
   deployment target appears: persistent storage (file or database-backed)
   behind the same `storageHas`/`storageGet`/`storagePut` contract and a
   real clock. The reference host (`tools/wasm_runtime_loader.ts`) resolves
   identity from bearer tokens like every other host, but it is in-memory,
   fixed-clock, and its `nextID` counter yields sequential ids — a
   test/measurement host, not a production one. Keep re-running
   `deno task measure:wasm` as the WASM binary grows.

6. Do not add anonymous worker identity or provider email delivery unless
   the product direction changes. Registered-user submissions remain the
   model; account and organization setup stays admin/org-admin driven.

UI minors queue:

- Add `type_ "button"` to any remaining secondary buttons that move into
  forms; continue replacing raw-id fields as directory-backed selectors
  become available on more pages.

Recently finished (compressed; details in
[WHAT_WE_DID.md](./WHAT_WE_DID.md)):

- PRs 1 through 140 are merged into `main`. The numbered PR roadmap in
  [PLAN.md](./PLAN.md) and the later agreed parity roadmap (lifecycle
  basics, developer task types, task/series/submission comments; scheduling
  stayed agent-side by decision) are implemented.
- PR 140 audited and fixed authorization, WASM parity, and Elm client
  issues. PR 139 cut the browser WASM demo over to the real backend mux and
  real domain services. PR 138 added browser-storage-backed `Store`
  implementations for all 9 domain packages.
- PR 137 added the funded-before-open invariant to the demo, an Unpublish
  escape hatch for individual tasks, and wired real-backend scenario parity
  into CI.

Before starting, reread [AGENTS.md](./AGENTS.md) and update the continuity
files if task scope changes.
