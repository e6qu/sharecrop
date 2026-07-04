# WASI Production Hosting Spike — Plan

## Goal

Replace `cmd/sharecrop`'s native production server with a single WASM
artifact compiled from the real backend (`internal/http` + the real domain
services in `internal/task`, `internal/org`, `internal/submission`, etc.,
backed by the real Postgres stores in `internal/db`), so there is exactly
one implementation of Sharecrop's server-side logic — not the native
server plus a separate from-scratch reimplementation (`internal/wasmdemo`)
for the browser demo. The browser demo and production would both run the
same compiled artifact; only the host environment around it differs
(browser JS host vs. a native Go host process).

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
   **Not yet measured** (a real gap, not covered by the 3.45µs number
   above): *module instantiation* cost itself — the 10,000 round trips
   happened inside a single instantiation. If the production design is
   genuinely "one fresh instance per HTTP request," `InstantiateModule`'s
   own overhead (separate from per-call latency) determines whether that's
   viable versus needing instance pooling/reuse across requests. This is
   an explicit measurement to add in Phase 2 or 4, not yet done.

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

- **Phase 2 — one real store method, end to end.**
  Pick the smallest real store method (e.g. `internal/db`'s
  `AuthStore`-equivalent read), wire it through the Phase 1 bridge shape by
  hand (not yet generated) so a wasip1 guest can call it and get a real row
  back from a real local Postgres via the host. This proves the actual
  target, not just a toy string round trip.
  *Checkpoint*: one real query round-trips correctly with real data,
  including error cases (row not found, constraint violation) mapping
  back through the bridge without losing the original `DomainError`
  shape `internal/core` methods depend on.

- **Phase 3 — codegen the bridge for one full store.**
  Build the `generate wasi-bridge` tool against one complete store
  interface (not all 15), and add the `check-wasi-bridge` CI gate.
  *Checkpoint*: codegen output compiles, passes the dual-run test
  described above for every method on that one store, and CI catches an
  intentionally-introduced drift (e.g. hand-edit the generated file and
  confirm `check-wasi-bridge` fails).

- **Phase 4 — one real HTTP request, fully end to end.**
  Host process has a real `net/http.Server`; on a request, it marshals the
  parsed request into the guest (compiled from real `internal/http` +
  enough real domain-service code to handle one real route, e.g.
  `GET /healthz` or a simple read-only route), the guest's real handler
  logic runs, any DB access goes through the Phase 3 bridge, and the
  response comes back out through the host's `http.ResponseWriter`.
  *Checkpoint*: a `curl` against the host process gets a byte-identical
  response to what `cmd/sharecrop serve` returns for the same request
  against the same seeded Postgres data.

**Everything past Phase 4 (broadening to the full route/store surface,
performance hardening, replacing `cmd/sharecrop` for real) is the actual
implementation effort this spike is scoping — not part of the spike
itself.**

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
- **After Phase 2**: if error-shape round-tripping (`DomainError`, typed
  IDs, etc.) proves fragile or lossy across the serialization boundary,
  that's a signal the whole approach needs a different serialization
  strategy before scaling to more stores.
- **After Phase 4**: this is the real go/no-go point for committing to the
  full effort. If everything checks out, the follow-up is a proper phased
  implementation plan (comparable in structure to the RBAC effort's
  5-phase plan) covering full store/route coverage, migration off
  `cmd/sharecrop`, and retiring `internal/wasmdemo` once the browser demo
  can run the same guest artifact.

## Effort framing

Deliberately not estimating in days/weeks yet. Phase 1's latency
checkpoint and Phase 2's error-shape checkpoint are the two things most
likely to blow up the estimate in either direction, and both are cheap to
find out (this is exactly why they're early phases, not late ones).
