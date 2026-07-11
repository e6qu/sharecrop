# Status

All work through pull request 141 is merged into `main`. PR 141 landed the
review-driven fixes, the two-section credit wallet (replacing the escrow
state machine), and Argon2id password hashing.

Implemented surface:

- Go HTTP API (`internal/http`) over Postgres-backed domain services, an Elm
  browser client, an MCP interface at `/mcp` (Streamable HTTP with SSE
  replay), scoped agent and organization-wide credentials, and a generated
  OpenAPI document (`docs/openapi.json`).
- The browser demo runs the real backend compiled to `js/wasm`. PR 138 added
  browser-storage-backed `Store` implementations for all 9 domain packages
  (`internal/wasmdemo/browserstore_*.go`). PR 139 cut the demo over to the
  real `internal/http` mux and the real domain services
  (`cmd/sharecrop-wasm/main_js_wasm.go`); `internal/wasmdemo` is no longer a
  separate request-handling reimplementation — it holds only the browser
  stores and the seed routine. Architecture details:
  [docs/wasm_demo_backend_spike.md](./docs/wasm_demo_backend_spike.md).
- PR 140 was an audit pass that fixed authorization checks, WASM demo
  parity gaps, and Elm client issues found by review.

Test status: PR CI runs format/contract/policy/type checks, Go unit and
integration tests, HTTP end-to-end tests, shared scenario parity against
both the real DB-backed backend and the compiled WASM demo, and Playwright
browser tests. All green on `main`.

Active task: `task/wallet-followup-and-boyscout` — a post-#141 follow-up
pass. It covers:

- Per-task funding state on the task DETAIL response: the response now
  reports `allocated_credits` and the individual `allocated_collectible_ids`
  currently held for the task (collectibles are non-fungible and tracked
  individually, not counted), read from `task_funds` /
  `task_fund_collectibles` on both backends. The browser gates the Refund
  button on live funding (no more Refund button on an unfunded declared
  reward), shows a per-task funding line, and surfaces the prominent fund
  callout for any unfunded draft (not just no-reward drafts).
- Boy-scout fixes from a fresh review: client validation for a collectible
  reward with nothing selected; Cancel now refetches the wallet; stale
  "escrowed reward" copy corrected to the wallet model; the raw-JSON submit
  editor seeds from the typed fields; password-reset fields are their own
  forms so Enter no longer triggers a login; review buttons get
  `type_ "button"`.
- Security hardening: `VerifyPassword`/`parseHashString` reject a
  zero-length salt or key so a malformed stored hash cannot act as a
  universal password.
- Data hygiene: refunding/cancelling a task no longer leaves the worker's
  reservation dangling (see `WHAT_WE_DID.md`).

Blocking issues: none. GitHub Pages `deploy-pages` occasionally fails
transiently after a merge and clears on retry; it is not caused by
repository code.
