# Single store implementation: retire `internal/wasmdemo`

## Goal

The browser demo should be **seeded with data but otherwise run the same
frontend and backend as production** — no bespoke reimplementation of the
domain layer. Today it already runs the real Elm frontend, the real
`internal/http` mux, and the real domain services; the *only* deviation is the
storage adapter: `internal/wasmdemo` hand-writes ~6,500 LOC of localStorage-
backed `Store` implementations (query logic, secondary indexes, ordering,
pagination) that duplicate `internal/db`'s Postgres stores.

We are eliminating that duplication so there is **one** store implementation,
parameterised only by the SQL engine underneath:

- **Production:** the stores over Postgres (`pgx`), exactly as today.
- **Browser demo:** the *same* stores over **SQLite compiled to wasm**,
  persisted in the browser.

`internal/wasmdemo` is then deleted in full.

## Why this is viable (de-risked empirically, 2026-07)

The store logic is already written once — as **277 raw SQL statements** across
24 files in `internal/db`, using `pgx/v5` + `$N` placeholders. Reusing that SQL
verbatim needs a SQLite that runs in the browser's wasm target.

- `modernc.org/sqlite` — **rejected**: its C-transpiled `libc` has no wasm port
  (`build constraints exclude all Go files` for both `js/wasm` and
  `wasip1/wasm`).
- `github.com/ncruces/go-sqlite3` — **works**: a `database/sql` driver that runs
  an embedded `sqlite3.wasm` via wazero, pure Go, in-process, no JS interop.
  - Builds for `js/wasm` **and** `wasip1/wasm` (~17 MB, embedded sqlite).
  - **Executes**: a create/insert/select round-trip ran through the wasip1
    build under a wazero host (wasm-in-wasm sqlite) and returned correct data.

## Scope of the current data layer

- 24 store files, 17 hold `*pgxpool.Pool` directly.
- **Transactions everywhere**: `store.pool.Begin(ctx)` with helper functions
  taking `pgx.Tx` (ledger double-entry, auth, org, task, submission,
  collectible, privacy, mcp-session, rate-limit, series). This is the largest
  structural surface.
- **Row locking**: `SELECT … FOR UPDATE` (ledger/task/collectible). SQLite is a
  single global writer, so locking is implicit — `FOR UPDATE` is stripped and
  correctness holds (concurrency differs, irrelevant to a single-user demo).
- **Dialect divergences (Postgres → SQLite):**
  - `now()` → `CURRENT_TIMESTAMP`
  - `$N::jsonb` casts → drop the cast (store JSON as `TEXT`)
  - `jsonb_agg` / `jsonb_build_object` → `json_group_array` / `json_object`
    (only `submission_store.go`)
  - `FOR UPDATE` → removed
  - DDL: `uuid`/`timestamptz`/`jsonb` → `TEXT`; `references … on delete` needs
    `PRAGMA foreign_keys = ON`
  - `$N` placeholders: **work natively** in ncruces sqlite (verified) — no
    rewriting needed.
  - **No** array / `= ANY` usage — nothing to port there.
- 34 Postgres DDL migrations in `migrations/`.

## Design

A minimal database-handle abstraction under `internal/db`, satisfied by two
adapters. The stores depend on the interface, never on `pgx` or `database/sql`
directly.

```go
type Querier interface {
    Exec(ctx, sql string, args ...any) (int64, error)      // rows affected
    Query(ctx, sql string, args ...any) (Rows, error)
    QueryRow(ctx, sql string, args ...any) Row
}
type Beginner interface { Querier; Begin(ctx) (Tx, error) }
type Tx interface { Querier; Commit(ctx) error; Rollback(ctx) error }
type Rows interface { Next() bool; Scan(...any) error; Close(); Err() error }
type Row  interface { Scan(...any) error }
```

- **pgx adapter** (prod): wraps `*pgxpool.Pool` / `pgx.Tx`. Identity SQL.
- **sqlite adapter** (browser): wraps `*sql.DB` / `*sql.Tx` over
  `ncruces/go-sqlite3`, applying the dialect rewrite at the statement boundary.

Every `pgx.Tx` in a helper signature becomes `db.Tx`. This is mechanical but
wide — the bulk of the porting effort.

`args ...any` collides with the policy check's ban on the bare word "any"; the
interface lives in a package whose doc explains the unavoidable `database/sql`
signature, or uses a named `type Arg = any` alias vetted against the checker.

## P0.5 browser de-risk — RESULTS (GO)

Validated in real headless Chrome (Playwright) with `ncruces/go-sqlite3`
v0.35.2 (which translates SQLite wasm to Go via wasm2go — native Go, not
interpreted):

- Cold page-load → sqlite up + schema + **5000 rows inserted (one tx)** + an
  indexed paginated query: **361 ms wall total**. WASM instantiate 11 ms; open
  49 ms; DDL 69 ms; insert-5000 159 ms; paginated query **5.3 ms**.
- Correct: 5000 rows, `$N` placeholders, index, `order by … desc limit/offset`,
  transactions all work in-browser.
- Size: 17.8 MB raw / **4.5 MB gzip** / **3.0 MB brotli** (sqlite + minimal
  main). A combined demo (mux + services + sqlite) is expected ~6–7 MB gzip.

**Persistence:** `memdb` imports bytes (`Create`) but exposes no export. Path:
a small forked exportable memdb VFS (read the page buffer back out) → snapshot
to IndexedDB via `syscall/js`; or accept demo-resets-on-reload. Decided in the
demo-cutover phase (Px). Not a feasibility blocker.

## Pm dialect — de-risked concretely

Using `ncruces/go-sqlite3` with the default `_timefmt` (read = auto-detect,
write = RFC3339Nano):

- **Timestamps (the crux):** ncruces auto-decodes a text column into `time.Time`
  **iff the column's declared type is time-ish** (`timestamptz`, `timestamp`,
  `datetime`, `date`) — a `text` column does not decode. Roundtrip of a Go
  `time.Time` is exact. So the SQLite DDL **keeps `timestamptz`** (SQLite accepts
  arbitrary type names) rather than mapping it to `text`. `now()` translates to
  `strftime('%Y-%m-%dT%H:%M:%fZ','now')` so DB-side and Go-provided timestamps
  share one RFC3339 format and sort consistently.
- **DDL translation:** `uuid`/`jsonb`/`json` → `text`; keep `timestamptz`;
  `default now()` → `default (strftime('%Y-%m-%dT%H:%M:%fZ','now'))`; keep
  `references` / `check` / indexes (FK enforcement left off — pragma default).
- **Statement translation:** `now()` → the strftime form; strip `::text` /
  `::jsonb` casts; strip `for update` (SQLite is single-writer); `jsonb_agg` /
  `jsonb_build_object` → `json_group_array` / `json_object` (submissions only).
- Already handled by the shared handle: `$N` placeholders, `NamedArgs`
  (→ `sql.Named`), `ErrNoRows` (← `sql.ErrNoRows`).

## Phased program (one PR at a time, prod stays green throughout)

- **P0 — abstraction + pilot store.** Add the `Querier/Tx/Rows` interfaces + pgx
  adapter. Port one simple store (`notification`) to it. Prod behaviour
  unchanged; existing db-checks pass. No demo changes, no sqlite yet.
- **P0.5 — browser runtime de-risk.** Headless-Chrome (local Playwright) harness
  loading a `js/wasm` build that opens ncruces sqlite, runs a query, and
  **persists** across reload via ncruces' browser VFS (OPFS/IndexedDB). Confirms
  the two remaining unknowns: in-browser wazero execution + persistence. If this
  fails, stop and reconsider before porting further.
- **P1…Pn — port each remaining store** off `*pgxpool.Pool`/`pgx.Tx` to the
  abstraction (prod-only, mechanical, green each PR).
### Pm.2 store-by-store dialect validation (in progress)

SQLite-only round-trip tests (FK off, struct-literal construction in `package
db`) confirm the risky translations per store:

- **Verified:** notification (now/casts/returning/`$N`→`?N`/timestamp
  round-trip), audit (`NamedArgs` `@limit`/`@offset`/`@action`), saved-queue-view
  (`on conflict do update` + `excluded` + inline CHECK enforcement).
- **json aggregation:** `json_group_array(json_object(...) order by ...)` works
  (SQLite 3.44+), so `jsonb_agg`/`jsonb_build_object` translate cleanly.
- **Remaining dialect gaps (submission + a few stores), Pm.2b:**
  - `ilike` → `like` (SQLite LIKE is case-insensitive for ASCII) — 3 sites.
  - `X at time zone 'UTC'` → strip; `to_char(X, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`
    → `strftime('%Y-%m-%dT%H:%M:%SZ', X)` — 2 sites (submission sensitive
    fields).
  - `array_agg(x)` → `json_group_array(x)` (check scan target) — 4 sites.
  - `encode(content, 'base64')` — no SQLite builtin; register a custom `encode`
    scalar via ncruces `CreateFunction` in a connection init hook. This open+
    register helper must live in a separate ncruces-importing package (not
    `package db`) so production keeps linking zero ncruces packages.
  - Then a submission-store SQLite test covering the full aggregation read path.

- **Pm — sqlite adapter + dialect + sqlite migrations + dual-run gate.** Add the
  sqlite adapter and dialect translator; generate/maintain the SQLite migration
  set. A dual-run test (mirroring the existing bridge dual-run harness) asserts
  each store returns identical results on Postgres and SQLite. This is the
  dialect correctness gate.
- **Px — cut the demo over.** `cmd/sharecrop-wasm` builds the real `internal/db`
  stores over the sqlite adapter instead of `wasmdemo`; seed via the real
  services; persistence wired to the browser VFS.
- **Pz — delete `internal/wasmdemo`** (all `browserstore_*.go`, `browser_storage
  .go`; keep seed rewritten onto real stores). Fold in the deferred
  **verification-depth** work: full `audit.Event` / `submission.Submission`
  codecs via shared wire packages for the moderation-triage and privacy bridges.

## Open questions to resolve during P0/P0.5

- Which browser VFS (OPFS vs IndexedDB) does ncruces support under `js/wasm`,
  and what's the persistence API?
- In-browser perf of interpreted sqlite for the seed + typical demo flows
  (17 MB module; wazero interpreter, no JIT under wasm).
