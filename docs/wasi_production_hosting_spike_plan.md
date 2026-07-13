# WASI Production Hosting Spike — Plan

> **Status: completed / historical.** The spike and the full production cutover
> are done: WASI hosting is the production **default** (`SHARECROP_WASI_MODE=native`
> opts out), and `internal/wasmdemo` is deleted. Kept for history; for current
> state see [README.md](../README.md), [STATUS.md](../STATUS.md), and
> [deployment.md](./deployment.md).

## Goal

Replace `cmd/sharecrop`'s native production server with a single WASM
artifact compiled from the real backend (`internal/http` + the real domain
services in `internal/task`, `internal/org`, `internal/submission`, etc.,
backed by the real Postgres stores in `internal/db`), hosted under WASI.

The browser-demo half of the original duplication problem is already
solved without WASI: since PR 139 the browser demo runs the real
`internal/http` mux and real domain services compiled to `js/wasm`, over
browser-storage-backed stores (PR 138); `internal/wasmdemo` is no longer a
separate reimplementation. See
[wasm_demo_backend_spike.md](./wasm_demo_backend_spike.md). The remaining
rationale for this spike is WASI production hosting itself: one compiled
artifact for real, horizontally-scaled production deployment, with the
host environment (native Go host process embedding a WASM runtime)
supplying Postgres and networking.

This document plans a **spike** — de-risking the two questions raised in
chat before this became a real engineering effort: is this feasible at
all, and if so, roughly what does it cost. It is not an implementation
plan for the full effort; see "Non-goals" below.

## Confirmed direction (from chat, before this doc)

- Compile target: `GOOS=wasip1 GOARCH=wasm`, hosted via an embedded Go WASM
  runtime (`github.com/tetratelabs/wazero`, pure Go, no cgo).
- If direct in-guest Postgres access proves infeasible (it does — see
  "Verified findings" below), fall back to a **sidecar shape**: a native Go
  host process owns the real Postgres connection and the real HTTP
  listener, and the WASM guest is called for the actual request-handling
  logic. This is not a second OS process — the *host binary itself*, which
  embeds wazero and runs the guest module in-process, is the "sidecar."
- Top priority: **strongly prevent the sidecar/guest split from becoming
  two divergent implementations** — the exact problem this whole effort
  exists to eliminate. Needs both an architectural answer (below) and
  tooling/tests that make drift structurally hard, not just a promise to
  be careful.

## Deployment shapes this must support (clarified in chat, 2026-07-05)

The one compiled artifact has to work correctly in two genuinely different
deployment shapes, not just compile once and "mostly work" in whichever one
gets tested first:

1. **The browser demo** (already the real mux + browser-storage-backed
   stores compiled to `js/wasm` since PR 139) — single effective user,
   localStorage-backed, runs on the browser's main thread.
2. **Real production** — presumably **multiple replicas** (horizontally
   scaled, likely behind a load balancer), plus explicit attention to
   **threads or web workers if the implementation ever uses them**. Neither
   is used today: the demo runs WASM on the main thread with no Worker
   offload, and the verified-safe production bridge design (Phase 1, below)
   deliberately never shares one WASM instance across goroutines in the
   first place.

The architecture already chosen for this effort (fresh WASM instance per
unit of work, exactly one driving goroutine per instance, all cross-request
state via real Postgres rather than in-process memory — see "Verified
findings" below) is inherently replica-safe *by construction*: each replica
is an independent OS process with no cross-replica in-memory state to keep
in sync, and anything that must be shared already goes through Postgres,
which is designed for concurrent multi-client access. The one already-known
gap this doesn't cover for free: `mcpHTTPSessionStore`'s in-memory
subscriber channels (see the deferred HTTP/2+3/100-session effort) would
need a pub/sub mechanism or sticky sessions to work correctly across
multiple replicas for MCP/SSE streaming specifically. Every other code path
(regular request/response against Postgres-backed state) is already
replica-agnostic with no extra work.

This isn't a new phase or deliverable — it's a constraint every later
architectural decision in this spike should be checked against, the same
way Phase 2 should be checked against "does this still work single-user in
a browser," not just "does this work against a real Postgres."

## Ecosystem research: is HTTP/networking really a WASI gap, and what should host it?

Researched with real web search (not just training-knowledge recall) after
the empirical findings below raised the question of whether this is a
Go-specific gap or an ecosystem-wide one, and what should run the host
process.

- **It's a Go-toolchain-specific gap, not an ecosystem-wide one.** WASI
  Preview 2 (`wasi:sockets` + `wasi:http`, both TCP/UDP client+server and a
  proper HTTP interface) is stable as of 2026 and is exactly what "no
  networking in wasip1" is missing. The gap is that **mainline Go only
  supports WASI Preview 1** (`GOOS=wasip1`) — Preview 2 / component-model
  support is tracked in [golang/go#65333](https://github.com/golang/go/issues/65333)
  but sits in the Backlog milestone with no active work and no comments
  since filing (checked directly). Preview 1 will not gain sockets
  retroactively — it's a different, frozen spec generation.
- **TinyGo (not mainline Go) already supports WASI Preview 2**, including
  `wasi:http`, and is production-proven via [Fermyon Spin](https://developer.fermyon.com/spin/go-components):
  Spin's Go SDK (`github.com/spinframework/spin-go-sdk`) gives a TinyGo
  component real outbound HTTP *and* — very relevant here — real outbound
  **Postgres** access via `spin-go-sdk/v2/pg`, using exactly the
  "host makes the real connection on the guest's behalf" pattern this
  plan already independently arrived at. This is strong external
  validation that the architecture in this doc is sound, not a
  from-scratch guess.
- **The likely disqualifying catch: TinyGo's standard library is
  incomplete in ways that matter a lot for this codebase.** TinyGo's
  `reflect` support is partial, and `encoding/json` — which depends on
  reflection — is reported to compile but panic at runtime for
  non-trivial cases. `internal/http` and every domain service in this repo
  uses `encoding/json` pervasively (every handler's
  `json.NewDecoder(r.Body).Decode(&request)` / `json.Marshal`). Compiling
  the *real* Sharecrop code through TinyGo would very likely require
  replacing `encoding/json` across the codebase with a non-reflective
  alternative — a large, invasive change working against the entire point
  of this effort (reuse the real code, don't fork it). This needs to be
  verified directly against this codebase (not just taken on secondhand
  reports) before ruling TinyGo out for real, but it's a serious enough
  signal to not treat Spin/TinyGo as the default path.
- **Net recommendation given the above**: stay on mainline Go
  (`GOOS=wasip1`) for full stdlib fidelity — already proven to compile and
  run this codebase's real logic correctly in Phase 0/1 — and keep the
  custom stdin/stdout-pipe bridge from Phase 1 for the networking Go's
  wasip1 target lacks, rather than switching to TinyGo/Spin purely to get
  networking "for free" at the cost of `encoding/json` compatibility. Spin
  remains valuable prior art to model the bridge design after, just not
  the runtime this effort should adopt directly.
- **Host runtime options (Node/Deno/Bun) for a JS-based host instead of a
  Go one**: none of the three have native WASI Preview 2 / component-model
  support yet. Node and Bun both support WASI Preview 1 (Bun's
  implementation is a fork of `wasi-js`, itself derived from `node-wasi`);
  Deno has partial Preview 1 support and an open, unstarted tracking issue
  for 0.2 ([denoland/deno#24289](https://github.com/denoland/deno/issues/24289)).
  A tool called `jco` (Bytecode Alliance) can transpile a Preview 2
  component to plain JS + wasm, which *could* run in any of these without
  native runtime support — worth knowing about, not needed for the
  mainline-Go/custom-bridge path this plan recommends, since Preview 2 was
  the reason to consider a JS host at all. If a JS host is ever revisited:
  Bun's own native `fetch`/Postgres client (`Bun.SQL` — Bun ships a
  first-party Postgres client) could serve as the host-side implementation
  behind a custom (non-WASI-standard) import bridge, the same shape as
  this plan's Go+`pgx` bridge — but this isn't needed if staying on the
  Go+wazero path.
- **Real production precedent for server-side Wasm generally** (grounding
  that this whole direction is a legitimate, not speculative, industry
  pattern): American Express built an internal FaaS platform on wasmCloud;
  Akamai's Fermyon-based edge platform reportedly handles 75M requests/sec
  across 4,000+ edge locations; Cloudflare Workers and Fastly Compute both
  run Wasm in production at scale. A 2026 industry survey cited server-side
  Wasm usage overtaking browser-only usage in production deployments for
  the first time.

## Verified findings (empirically tested in this session, not just reasoned about)

Environment: Go 1.26.3 darwin/arm64, `wazero` v1.12.0 (via
`go run github.com/tetratelabs/wazero/cmd/wazero@latest`).

1. **`GOOS=wasip1 GOARCH=wasm` builds successfully** with this toolchain,
   and `wazero run` executes the result. No tooling gap here.
2. **`crypto/rand.Read` and `time.Now()` work correctly** under wasip1 —
   backed by real WASI preview1 syscalls (`random_get`, `clock_time_get`).
   Confirmed by running a trivial program that reads random bytes and the
   clock; both returned real, differing values across runs.
3. **Outbound TCP (`net.Dial`) does not work — confirmed as a stub, not
   just "unsupported."** A guest dialing `127.0.0.1:<port with nothing
   listening>` and a guest dialing `127.0.0.1:<port with a real `nc -l`
   listener>` both returned the *identical* `connect: Connection refused`
   error. If this were reaching a real socket layer, the second case should
   have succeeded. It didn't — proving Go's wasip1 `net` package returns a
   canned failure for outbound dials regardless of what's actually
   listening on the host. **This directly blocks `pgx` (our Postgres
   driver) from working inside a wasip1 guest under any circumstance,
   confirming the concern raised in chat.**
4. **Inbound TCP (`net.Listen` + `http.Serve`) does not work either — and
   this was not previously known going into this spike.** A guest calling
   `net.Listen("tcp", ...)` then `http.Serve(...)` deadlocks immediately:
   `fatal error: all goroutines are asleep - deadlock!` inside
   `net.(*fakeNetFD).accept` / `net_fake.go` / `fd_fake.go` — Go's stdlib
   ships an entirely fake `net.Conn`/`net.Listener` implementation for
   `GOOS=wasip1`, not a real one backed by WASI syscalls, for *both*
   directions. `wazero run -listen host:port` (which pre-opens a real host
   socket and can hand it to a guest) does not change this: Go's own `net`
   package has no code path that knows how to use that pre-opened socket —
   a guest would need a non-stdlib, wazero-specific API to reach it, which
   nothing in `internal/http` or the wider Go standard library uses today.

**Net effect on the architecture**: it is not just Postgres access that
needs to move to the host side. The wasip1 guest cannot do *any* real
networking, inbound or outbound. This means the guest cannot be "the real
backend, compiled as-is, minus its DB layer" — it has to be a pure
request-handling function, structurally identical in shape to how
`internal/wasmdemo`'s browser demo already works today
(`sharecropHandleRequest(method, path, headers, body) → response`), except
compiled from the *real* `internal/http` + domain services instead of a
from-scratch reimplementation, and hosted by a native Go process instead of
a browser.

5. **A naive "reactor" bridge (host-imported functions callable repeatedly
   after one-time init, via `//go:wasmimport`/`//go:wasmexport`) is unsafe
   and was empirically caught corrupting state, not just theorized about.**
   The natural first design was: instantiate the guest once, let it block
   forever in `main()` (e.g. an infinite sleep loop) so it stays alive, and
   have the host call exported functions into it repeatedly whenever a
   request needs one. Building this required a background goroutine to
   drive `_start` (since `main()` never returns) while the host's main
   goroutine separately calls the exported function on the *same module
   instance*. This produced a wrong result (a corrupted sum instead of the
   correct one) and then a crash (`wasm error: out of bounds memory
   access`) in the guest's own runtime. **A single wazero module instance
   is not safe to drive from two host goroutines concurrently** — even
   with proper readiness synchronization (a host-imported `host_ready()`
   callback closing a channel), the guest's internal Go scheduler state
   gets corrupted the moment a second logical call path touches the same
   instance while the first is still "live" (parked in a sleep, not
   actually finished). This is a real, structural constraint of the
   embedding model, not a bug in the test code.
6. **The safe, verified alternative: one fresh module instance per
   call, driven by exactly one goroutine, with plain WASI stdin/stdout as
   the RPC transport — not `wasmimport`/`wasmexport` at all.** Compile the
   guest once (`wazero.Runtime.CompileModule`, cheap to reuse), then for
   each unit of work, instantiate a *fresh* module
   (`wazero.Runtime.InstantiateModule`) wired to a pair of `io.Pipe()`s as
   its stdin/stdout. Exactly one goroutine calls `InstantiateModule` and
   blocks until the guest's `main()` returns (this is the *only* goroutine
   ever touching that instance). A second goroutine pumps the pipes:
   reads length-prefixed (or newline-framed, for the spike) request
   frames the guest writes to stdout, does the real work (in the full
   design, a real Postgres query), and writes the response back to the
   guest's stdin, which the guest's blocked `Read` call picks up — all
   within that one instance's one `main()` execution. No shared instance
   state is ever touched by two goroutines at once; the pipes are the only
   thing bridged between two goroutines, and that is exactly what
   wazero's `WithStdin`/`WithStdout` are designed to do safely. **Verified
   working**: a guest reading an iteration count, then looping "write
   `DOUBLE n`, read the doubled result" `n` times, writing `RESULT <sum>`
   at the end, produced the mathematically correct result
   (`2 × (0+1+...+9999) = 99990000`, confirmed exact) with **~3.45
   microseconds per round trip** (10,000 calls in 34.5ms) — comfortably
   under the "low hundreds of microseconds" target from the original
   Phase 1 checkpoint below.
   **Module instantiation cost — now measured in Phase 2 (see finding #8).**
   The 10,000 round trips above happened inside a single instantiation, so
   they did not cover the cost of `InstantiateModule` itself. Phase 2's
   end-to-end test does one fresh instantiation per unit of work, so its
   per-call mean (~2.7ms idle, several ms under concurrent load) *is* the
   instance-per-request cost — dominated by instantiation, not by the
   store round trip. This is the number that decides whether "one fresh
   instance per HTTP request" is viable as-is or needs instance pooling.

7. **A real `internal/db` store method round-trips from a wasip1 guest to
   real Postgres and back, with the `DomainError` shape intact — verified
   in Phase 2, not reasoned about.** `internal/wasibridge` wires
   `AuthStore.FindCredentialByEmail` (the smallest real read) through the
   finding-#6 bridge by hand. A guest compiled to `GOOS=wasip1`
   (`cmd/sharecrop-wasi-spike-guest`, ~4.3 MB, and it *does* compile the
   real `internal/auth` including `golang.org/x/crypto/argon2` to wasip1)
   issues the lookup over its stdin/stdout; the host services it against a
   real local Postgres via the exact `db.NewAuthStore(pool)` value
   `cmd/sharecrop` uses. The integration test
   (`tests/integration/wasibridge_store_test.go`, `-tags integration`)
   proves all three result shapes: **found** matches a direct store call
   field-for-field (user id, email, password hash, status), **missing**
   returns `CredentialMissing`, and **rejected** returns
   `CredentialLookupRejected` whose `core.ErrorCode` survives as the
   canonical value — crossing the serialization boundary *twice* (host→guest,
   then the guest's report back to host). This resolves the "After Phase 2"
   decision point below: error-shape round-tripping is not lossy.
8. **Fresh-instance-per-unit-of-work costs low single-digit milliseconds,
   dominated by instantiation, not the DB round trip.** The Phase 2 test
   instantiates a fresh guest module for every lookup (the finding-#6 safe
   shape) and measures the full wall time: ~2.7ms mean on an idle machine,
   rising to several ms under concurrent build load. The local Postgres
   round trip inside that window is sub-millisecond, so `InstantiateModule`
   is the cost driver. **Implication for the full effort**: "one fresh WASM
   instance per HTTP request" adds ~2–3ms of floor latency per request.
   Whether that is acceptable as-is or motivates an instance-pool /
   snapshot strategy is a Phase 3+ decision, now backed by a real number
   rather than a guess. The bridge itself (serialization + pipe transport)
   is negligible next to instantiation, consistent with the 3.45µs/call
   from finding #6.

## Target architecture

```
┌─────────────────────────── native Go host process ───────────────────────────┐
│                                                                                │
│  net/http.Server (real, native)              pgxpool.Pool (real, native)      │
│    Accepts real HTTP connections               Real Postgres connection      │
│    ── on each request ──►                        ▲                          │
│         │                                        │ stdin/stdout pipe RPC     │
│         ▼                                        │ (one fresh instance,      │
│  ┌──────────────────────── wazero runtime ────────┴──one driving goroutine)──┐│
│  │  WASM guest (GOOS=wasip1, compiled from internal/http + domain            ││
│  │  services + internal/db's store *interfaces*, unmodified real code)       ││
│  │                                                                            ││
│  │  Guest writes a framed "store RPC" request to its stdout for every DB      ││
│  │  operation instead of driving pgx directly, and reads the response back   ││
│  │  from stdin; host dispatches straight into the SAME internal/db store     ││
│  │  implementations cmd/sharecrop uses.                                      ││
│  └──────────────────────────────────────────────────────────────────────────┘│
└────────────────────────────────────────────────────────────────────────────────┘
```

Key properties:
- **The host process is not a separate reimplementation** — it directly
  constructs and calls the exact same `internal/db.New*Store(pool)` values
  `cmd/sharecrop` already constructs today. The only new code is a thin
  transport shim on both sides of the host/guest boundary.
- **One fresh guest instance per unit of work** (in the full design, likely
  "per HTTP request," pooled/reused where wazero supports it for
  performance — a later-phase concern, not a spike concern), driven by
  exactly one goroutine end to end, with the stdin/stdout pipe pair as the
  only cross-goroutine bridge. This sidesteps the concurrency hazard in
  finding #5 by construction, rather than by careful discipline.

## Anti-drift strategy (the explicit priority from chat)

The single biggest risk is that "the RPC bridge" quietly becomes a second,
hand-maintained reimplementation of what `internal/db`/`internal/http`
already do — recreating the exact two-backends problem this effort exists
to solve, just one layer deeper. Three independent safeguards, not one:

1. **No new business logic anywhere in the bridge.** The host-side
   dispatcher reads a framed request off the guest's stdout pipe
   (`store name, method name, encoded args`), calls straight into the
   real `internal/db` store method, and writes the encoded result back to
   the guest's stdin. It must contain zero decision logic of its own — if
   a reviewer can find an `if`/`switch` in the bridge that isn't purely
   "which method do I call and how do I (de)serialize its arguments,"
   that's a defect.
2. **Codegen the bridge from the same interface definitions the stores
   already implement, the same way this repo already generates Elm
   contract types from `internal/contracts/definitions.go`
   (`go run ./cmd/sharecrop generate elm-contracts`) and OpenAPI from the
   live route table (`go run ./cmd/sharecrop generate openapi`).** A new
   `go run ./cmd/sharecrop generate wasi-bridge` step introspects the
   store interfaces (`internal/task.Store`, `internal/org.Store`, etc. —
   parsed via `go/ast`, mirroring `internal/openapi`'s existing extraction
   approach) and emits: (a) the guest-side RPC client stubs, (b) the
   host-side dispatcher. A `check-wasi-bridge` CI gate (mirroring
   `check-contracts`/`check-openapi` exactly) regenerates and diffs on
   every PR — a hand-edited bridge file that drifts from the real
   interfaces fails CI immediately, not at review time or in production.
3. **Dual-run the identical test suite against both access paths.** Every
   store-level test that exists today (or gets added) runs twice: once
   against `internal/db`'s stores directly (as today), and once against
   the same stores reached through the compiled guest + host bridge, with
   a real Postgres behind both. A behavioral difference between the two
   paths is a test failure, not a design review finding it later.

## Spike phases

Each phase has an explicit go/no-go checkpoint. Stop and report back
if a phase's checkpoint fails — do not push forward into more
implementation on a shaky foundation.

- **Phase 0 — toolchain viability. ✅ Done, in this session.**
  Confirm `GOOS=wasip1` builds and runs under wazero at all, and precisely
  characterize what does/doesn't work (networking, both directions;
  crypto/rand; clock). See "Verified findings" above.

- **Phase 1 — minimal round trip. ✅ Done, in this session.**
  Proved the stdin/stdout-pipe RPC shape (see finding #6 above): a wasip1
  guest, driven by one goroutine and bridged to the host via a pair of
  `io.Pipe()`s wired as its stdin/stdout, round-trips 10,000 framed
  request/response exchanges correctly (mathematically verified result)
  at ~3.45µs per call — well under the "low hundreds of microseconds"
  target. Also proved, by building it wrong first, that the naive
  concurrent-goroutine "reactor" shape is unsafe (finding #5) — ruled out,
  not just avoided by convention.
  *Checkpoint met*: round-trip works, correctly, at a latency that leaves
  ample headroom for real per-query overhead (serialization, actual
  Postgres round-trip time) on top.

- **Phase 2 — one real store method, end to end. ✅ Done, in this session.**
  `internal/wasibridge` wires `AuthStore.FindCredentialByEmail` through the
  Phase 1 bridge shape by hand (not codegen). A `GOOS=wasip1` guest
  (`cmd/sharecrop-wasi-spike-guest`) issues the lookup over stdin/stdout;
  the native host (`internal/wasibridge/host.go`, embedding wazero, plus the
  `cmd/sharecrop-wasi-spike` CLI) services it against a real Postgres via
  the same `db.NewAuthStore(pool)` the server uses. Fast unit tests cover
  the wire format (`internal/wasibridge/protocol_test.go`); an
  integration test drives the whole path against real Postgres
  (`tests/integration/wasibridge_store_test.go`). See findings #7 and #8.
  *Checkpoint met*: found/missing/rejected all round-trip correctly with
  real data; the rejected path preserves the `DomainError` code across two
  serialization crossings; instance-per-call cost measured (~2.7ms,
  instantiation-dominated). The bridge holds **no business logic** — the
  host dispatcher only parses the email argument, calls the real store, and
  serializes the result (anti-drift safeguard #1). Safeguards #2 (codegen)
  and #3 (dual-run for every method) arrive in Phase 3.

- **Phase 3 — codegen the bridge for one full store. ✅ Done, in this session.**
  Target store: `internal/audit.Store` (all 3 methods: `Record`, `Get`,
  `List`) - small but representative (typed ids, a string-wrapper enum,
  nested structs, three sealed-union filters, a slice, `time.Time`,
  `DomainError`). Generic transport `internal/wasibridge/rpc` (method-keyed
  call frames over the Phase 1 pipe shape) generalizes the Phase 2 auth
  spike, both now sharing `internal/wasibridge/wire` (framing) and
  `internal/wasibridge/domainwire` (the shared `DomainError` codec).
  `internal/wasibridge/auditbridge` holds hand-written, round-trip-tested
  per-type codecs (`codec.go`) plus a **generated** `bridge_gen.go`
  (`Dispatch` host router + `GuestStore` client) emitted by
  `internal/wasibridge/gen` via `go run ./cmd/sharecrop generate wasi-bridge`.
  The `check-wasi-bridge` CI gate regenerates and diffs (mirroring
  `check-contracts`/`check-openapi`).
  *Checkpoint met*: the generator reproduces the committed bridge exactly
  (safeguard #2); the dual-run integration test
  (`tests/integration/auditbridge_store_test.go`) exercises every method
  against real Postgres through both the direct-db path and the compiled
  `GOOS=wasip1` guest + host bridge, with matching results including the
  not-found rejection and the write path (safeguard #3); and the generator
  errors loudly on a Store method whose type has no registered codec
  (`gen_test.go`), so a new method can't silently ship an incomplete bridge.
  The generated `GuestStore` also carries a `var _ audit.Store = GuestStore{}`
  assertion, so adding a method to the interface without regenerating fails
  to compile. Anti-drift safeguard #1 (no business logic in the bridge)
  holds: `Dispatch` only decodes args, calls the real method, and encodes
  the result.

- **Phase 4 — one real HTTP request, fully end to end. ✅ Done, in this session.**
  `cmd/sharecrop-wasi-http-host` runs a real `net/http.Server` whose handler,
  per request, serializes the request (`internal/wasibridge/httpbridge`), runs
  a fresh wasip1 guest (`cmd/sharecrop-wasi-http-guest`) over the Phase 3 `rpc`
  transport, and writes the serialized response back. The guest runs the
  **real production routing table** - `httpserver.New(...)`, the exact mux
  `cmd/sharecrop serve` builds - through an `httptest` recorder (the same shape
  the browser demo uses). The route is `GET /healthz`, which touches no store,
  so no DB bridge is needed for this slice; a store-touching route would bind
  Phase 3 `GuestStore`s in place of the guest's nil services and its DB calls
  would RPC back over the same channel.
  *Checkpoint met*: the integration test
  (`tests/integration/httpbridge_test.go`) asserts the response through the
  compiled guest is **byte-identical** (status, `Content-Type`, body) to the
  same mux run in-process, for both `GET /healthz` (200) and an unknown
  `/api/...` route (404); a real `curl` against the host process returns
  `200 {"status":"ok"}` through the wasm guest.

**Everything past Phase 4 (broadening to the full route/store surface,
performance hardening, replacing `cmd/sharecrop` for real) is the actual
implementation effort this spike is scoping — not part of the spike
itself.**

## Implementation progress (post-spike)

The spike is done; this tracks the follow-up implementation effort as it lands.

- **Bridge codegen generalized to N stores.** `internal/wasibridge/gen` is now
  store-agnostic: each store is a `storeSpec` (naming its codecs), and
  `go run ./cmd/sharecrop generate wasi-bridge` regenerates every store in
  `gen.Targets()`. Shared core-type codecs (typed ids, page, time) moved to
  `internal/wasibridge/corewire` so bridges don't duplicate them.
  **ALL TEN domain stores are bridged** (`audit`, `notification`, `auth`,
  `agent`, `orgcred`, `assets`, `submission`, `ledger`, `org`, `task` - each in
  its own `*bridge` package), dual-run-verified against real Postgres. `agent`
  added nullable-pointer fields and a scope-set; `orgcred` reuses agent's
  `Label`/`ScopeSet`/`State` codecs (extracted into a shared
  `internal/wasibridge/agentwire` package); `assets` (collectibles) drove a
  generator enhancement to disambiguate a method with two arguments of the same
  type (`ListCollectiblesByOwner(string, string, ...)`), and covers command
  structs and collectible/id-slice result payloads; `submission` (submissions,
  attachments, validation outcomes, sensitive fields, and the comment thread)
  drove a second generator enhancement so a `[]LocalType` argument qualifies its
  element type (`CreateSubmission(..., []SensitiveField)`); `ledger` (the deepest
  unions - commands carrying credit/tip/collectible/ban selection unions, and
  accept/reject results carrying nested payout and tip outcome unions) needed no
  generator change and was dual-run-verified with a full fund -> accept ->
  refund flow; `org` (organizations, members, and teams, including the TeamOwner
  tagged union) drove a third generator enhancement - `extraImports`, for a
  method argument whose type lives in another package
  (`ProvisionMember(..., auth.EmailAddress, ...)`); `task` (the widest store -
  21 methods, the full Task with ~10 nested unions, series, reservations, and
  comments) needed no generator change, and its attachment codec was extracted
  into a shared `internal/wasibridge/attachmentwire` package (also adopted by
  `submission`). With every store bridged, the remaining work is the host wiring:
  weigh instance pooling against the ~2-3ms instance-per-request floor, then
  migrate `cmd/sharecrop serve` onto the WASI host. One generic
  guest (`cmd/sharecrop-wasi-store-guest`) routes every store by method prefix.
  `auth` (the largest - 13 methods, 10 result unions) exercised the pattern's
  edges: its opaque hash/token types round-trip through reconstruction
  constructors added to `internal/auth`
  (`RefreshTokenHashFromString`/`AccountTokenHashFromString`/
  `AccountTokenKindFromString`, mirroring the existing `ParsePasswordHash`),
  and shared core codecs grew to cover more id types plus plain strings and
  timestamps as method arguments (`corewire`). Remaining stores (ledger, task,
  org, submission, assets, orgcred) are the same pattern: add a spec +
  hand-written codecs + a dual-run test.
- **A real authenticated, store-touching route runs end to end through the
  guest** — the Phase 3 + Phase 4 pieces, combined. `GET /api/notifications`
  is served entirely by the wasip1 guest (`cmd/sharecrop-wasi-app-guest`,
  building the real mux via `internal/wasibridge/appmux` with a live
  notification service): the stateless access-token verifier checks the bearer
  token in-guest (no store — verification is signature + clock), and the
  notification read is bridged back to the host and hits real Postgres.
  `cmd/sharecrop-wasi-app-host` is the production-shaped `net/http.Server` for
  it (fresh guest per request, store calls dispatched to `internal/db` by
  method prefix, the token secret handed in via WASI env —
  `rpc.Host.WithGuestEnv`). The integration test
  (`tests/integration/approute_test.go`) asserts the guest's response is
  byte-identical to the same mux run in-process against the same store, and
  that it actually contains the seeded row. This is the key proof that the two
  halves compose; broadening to routes that need other stores just needs those
  stores bridged and wired into `appmux`. `GET /api/users` now does exactly
  that - it reads the auth store's directory through the guest via a live auth
  service (backed by the bridged auth `GuestStore`), byte-identical to native
  (`tests/integration/authroute_test.go`) - the first route to exercise a
  bridged store's *service*, not just stateless token verification. The
  host-side store routing is shared as `internal/wasibridge/storehost`.
- **The FULL production mux now runs in the guest.** With every store bridged,
  `internal/wasibridge/appmux` was expanded from the auth+notification slice to
  the complete domain-service graph (auth, notification, org, task, submission,
  ledger, agent, orgcred, assets, audit), wired in the same dependency order
  `cmd/sharecrop serve` uses - org and agent services feed the task service; the
  shared task store and the org service feed the submission service; no adapter
  types are needed because the services satisfy each other's cross-interfaces
  directly. The RuntimeState services with no dedicated domain store (rate
  limiters, MCP sessions, saved queue views, privacy, platform admins,
  moderation triage) keep their in-memory defaults; audit and notification run
  over the bridged stores. `appmux.New` now takes an `appmux.Stores` struct (ten
  store interfaces), so the guest passes bridge GuestStores and the tests pass
  real `internal/db` stores - the assembled mux is identical either way. A third
  route test (`GET /api/credits/balance`, backed by the ledger service) proves a
  service beyond auth/notification runs byte-identically through the guest and
  returns the real signup-grant balance.
- **Instance pooling now takes the guest-startup cost off the request hot path.**
  Finding #8's open question (is one fresh instance per HTTP request viable, or
  does it need pooling?) is resolved by pooling. The guest was a wasip1 *command*
  (ran `main()` once per unit of work and exited), so each request paid the
  ~2-3ms startup floor and command instances can't be reused; and finding #5
  proved a *shared* reactor instance corrupts state. The design that threads the
  needle: the guest's `main()` loops over units of work read from stdin as "work"
  frames (via `rpc.Serve`), staying alive between them, and the host keeps a pool
  of such instances (`rpc.Pool`) and checks one out per unit of work. Safety is
  unchanged from finding #6 - each instance is driven by exactly one goroutine
  and touched by no other: a per-instance `session` has a runner goroutine (owns
  the wazero instance) and a driver goroutine (owns both pipe ends, so the
  two-write framing never interleaves), and request goroutines reach the driver
  only over Go channels. Pooling adds no shared-instance concurrency; concurrency
  comes from having several sessions. `Host.Call` (fresh instance per call, used
  by the dual-run/route tests) and `Pool.Call` sit on the same `session`;
  `httpbridge.Handler` takes an `rpc.Caller` (either); the production app host
  pools by `SHARECROP_WASI_POOL_SIZE` (default GOMAXPROCS) and builds each mux
  once per instance, not per request. Two concurrency tests prove no cross-talk
  under load (144 store units through 4 reused instances; 16 concurrent HTTP
  requests through the pooled app host), each unit seeing only its own data.
- **The production cutover is DONE and WASI hosting is now the DEFAULT.**
  `cmd/sharecrop serve` runs production through the WASI guest embedded in the
  binary (`internal/wasiguest`, built by `make wasi-app-guest` as part of `make
  build`): it pools that guest (`rpc.Pool`, `SHARECROP_WASI_POOL_SIZE`/GOMAXPROCS),
  dispatches store calls to Postgres via `storehost`, routes `/api/`, `/mcp`,
  `/healthz` to the guest, and serves static assets + the SPA shell host-side. Set
  `SHARECROP_WASI_MODE=native` to run the in-process mux, or `SHARECROP_WASI_GUEST=
  <path>` to override the embedded guest; a binary built without the guest runs
  native, and a present-but-broken guest fails loudly. Verified by an integration test
  (both halves of the split) and a live smoke test of the real `serve` binary
  (`/healthz` 200 through the guest pool against Postgres, `/` 200 from host
  static). **Production can now run the same compiled WASM artifact as the browser
  demo - the deviation this whole plan set out to close is closed.** Non-blocking
  follow-ups: flip the WASI path to the default once proven in staging; embed or
  mount the static files into the guest if single-artifact static serving is
  wanted; retire `internal/wasmdemo`.

## Non-goals for this spike

- Full route/store coverage. Phases 2-4 intentionally pick the smallest
  possible slice.
- Performance tuning beyond the rough per-call latency sanity check in
  Phase 1.
- HTTP/2, HTTP/3, or the ~100-concurrent-MCP-SSE-session hardening —
  confirmed separately as sequenced *after* this effort, not part of it.
- Deleting or changing `cmd/sharecrop`, `internal/wasmdemo`, or
  `cmd/sharecrop-wasm` — nothing about the existing native server or
  browser demo changes during the spike.
- Deciding between the "byte-relay `net.Conn` shim" vs. "query-level RPC"
  sub-approaches mentioned as open questions in the original feasibility
  research — Phase 2 will surface which is actually less work once there's
  a real store method to try it against, rather than deciding on paper.

## Decision points

- **After Phase 1** (met — see above): the measured ~3.45µs per-call
  overhead leaves no reason to batch calls per-request rather than
  per-query; re-evaluate only if Phase 2's real Postgres round trip
  reveals overhead the toy test didn't (e.g. serialization cost for
  realistic row shapes, not just a single int).
- **After Phase 2** (met — see findings #7/#8): error-shape round-tripping
  is *not* lossy — a `DomainError`'s `core.ErrorCode` survives as the
  canonical value across two serialization crossings, and typed values
  (`UserID`, `EmailAddress`, `PasswordHash`) reconstruct via their existing
  parse constructors. The JSON-over-length-prefixed-frames serialization is
  sound to carry into Phase 3; the one refinement it needs there is codegen
  so the per-method (de)serialization is generated from the store
  interfaces rather than hand-written. Separately, finding #8's ~2–3ms
  instance-per-request floor is the cost signal to weigh a pooling strategy
  against — a performance decision, not a correctness blocker.
- **After Phase 4** (reached — go): every checkpoint held. Toolchain viable
  (Phase 0); the fresh-instance/one-goroutine/stdin-stdout-pipe transport is
  correct and cheap (Phase 1, ~3.45µs/call); a real store method round-trips
  with `DomainError` fidelity at ~2.7ms/instance (Phase 2); the bridge is
  generated from the interface, drift-gated, and dual-run-verified for a full
  store (Phase 3); and the real `internal/http` mux runs inside a wasip1 guest
  behind a native `net/http.Server` with byte-identical output (Phase 4). No
  checkpoint surfaced a reason to abandon the approach. **The spike is
  complete and the direction is confirmed.** The follow-up is the full
  implementation effort (its own phased plan): bridge the remaining store
  interfaces (extend the Phase 3 codecs + registry), wire the real services in
  the guest against those `GuestStore`s, cover the full route surface, weigh
  the ~2-3ms instance-per-request floor against an instance-pool strategy
  (Phase 2 finding #8), migrate `cmd/sharecrop` onto the hosted guest, and
  eventually retire `internal/wasmdemo` once the browser demo can run the same
  artifact. That effort is out of scope for this spike, which set out only to
  answer "is this feasible, and roughly what does it cost" - and it is, at a
  cost now measured rather than guessed.

## Effort framing

Deliberately not estimating in days/weeks yet. Phase 1's latency
checkpoint and Phase 2's error-shape checkpoint are the two things most
likely to blow up the estimate in either direction, and both are cheap to
find out (this is exactly why they're early phases, not late ones).
