# Backendless Demo Semantic Parity

`site/demo/backend.js` is a fake backend. It now has route and response-shape parity checks, and selected user-flow semantics are covered by Deno tests. It still cannot prove full semantic parity with the Go API because it reimplements behavior in JavaScript.

## Options

1. Add shared scenario tests that run the same scripted user flows against the Go HTTP API and the demo backend.
   - Best next step.
   - Keeps the backendless demo on static hosting.
   - Catches behavior drift for task creation, reservations, submissions, reviews, notifications, collectibles, org/team flows, account tokens, and admin views.
   - Does not require replacing the demo backend immediately.

2. Compile pure Go domain/application services to WASM and expose a browser-side request handler.
   - Higher semantic parity for code paths moved behind storage interfaces.
   - Requires browser storage adapters for auth, tasks, submissions, ledger, assets, orgs, audit, notifications, MCP sessions, and rate limits.
   - Does not reuse PostgreSQL, pgx, net/http server wiring, process signals, or production migrations directly in the browser.
   - Build size, startup cost, JS interop, IndexedDB persistence, and deterministic test setup need a spike before adoption.

3. Host a real demo backend with a disposable database.
   - Highest semantic parity.
   - No longer backendless and not suitable for static-only hosting.
   - Requires provisioning, reset behavior, abuse controls, and operational ownership.

4. Generate more of the demo fake from contract definitions.
   - Good for route and JSON shape coverage.
   - Does not prove domain semantics unless paired with scenario tests.

## Recommendation

Build shared scenario parity tests first. They improve the current backendless demo and transfer directly to the backend-backed solution. After the core scenario suite exists, run a WASM spike focused on one vertical slice, such as task creation plus submission review plus notifications. Adopt WASM only if it can use explicit browser storage adapters without fallbacks or dead paths.
