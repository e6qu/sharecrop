# WASM Demo Backend Spike

The backendless demo currently uses `site/demo/backend.js`, an in-browser fake backend. Shared scenario parity tests now exist, so a WASM replacement can be evaluated against the same behavior instead of relying only on route/shape checks.

## Finding

A Go/WASM demo backend is viable only after the application services can run against explicit browser storage adapters. The current production server wires domain services through Postgres-backed stores, auth/session stores, rate-limit buckets, audit stores, notification stores, and MCP stores. A browser build cannot reuse pgx, migrations, process-local server wiring, or `net/http` handlers directly.

## Required Shape

- A `js/wasm` request adapter receives method, path, headers, and body from `fetch` interception.
- Domain/application services are constructed with browser storage adapters, not fake fallback stores.
- Browser storage adapters are explicit packages, likely backed by IndexedDB for persisted data and in-memory state only where the domain already treats state as process-local.
- WASM tests run the shared scenario parity suite against the request adapter.
- Missing browser storage features fail at build or request time with clear errors.

## Adoption Gates

Do not replace `site/demo/backend.js` until the WASM path can satisfy these gates:

1. It passes the shared scenario parity suite for task creation, selectors, comments, reservations, submission review, notifications, collectibles, and account-token flows.
2. It has no fallback behavior for unimplemented stores or handlers.
3. It has a deterministic reset path for demo data.
4. Bundle size and startup time are measured from a production build.
5. GitHub Pages deployment still passes the deployed routing check.

## Next Spike Step

Create a narrow WASM request adapter around one vertical slice: organization/team selectors plus task creation and task comments. If that slice requires broad store rewrites or hidden substitute behavior, keep the JavaScript demo backend and expand shared parity tests instead.
