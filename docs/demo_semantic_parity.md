# Backendless Demo Semantic Parity

`site/demo/backend.js` is a fake backend. It now has route and response-shape parity checks, a shared scenario runner used by Deno demo tests, and a CLI entry point for running the same scenario against a deployed or local real API. It still cannot prove full semantic parity with the Go API because it reimplements behavior in JavaScript.

## Current Checks

- `deno test --allow-read tests/deno` runs demo route/shape checks and the shared scenario parity suite against `site/demo/backend.js`.
- `deno task check:scenario-parity -- --origin <api-origin> --token <access-token>` runs the same shared scenario against a real API. The supplied token must be a platform admin token that can refresh auth, read operations status, issue account tokens, mint/transfer collectibles, create organizations, create teams, create tasks, create submissions, create task/submission comments, and register additional scenario actors.
- `deno task check:pages-routing -- --origin <pages-origin>` checks deployed GitHub Pages root, docs, demo entry paths, and demo assets after deployment.

## Options

1. Expand shared scenario tests that run the same scripted user flows against the Go HTTP API and the demo backend.
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

Keep expanding shared scenario parity before replacing the fake backend. The suite covers selector pagination/query behavior, team/organization queue search/type/sort behavior, org/team/task/comment creation, admin operations, account-token issue shape, collectible catalog/mint/transfer, submission creation/comments with sensitive-field response metadata, notification read shape, and a multi-actor reservation/submission-review/payout flow.

Use [wasm_demo_backend_spike.md](./wasm_demo_backend_spike.md) for the current WASM finding. Adopt WASM only if it can use explicit browser storage adapters without fallbacks or dead paths.
