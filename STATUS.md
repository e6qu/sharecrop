# Status

All work through pull request 140 is merged into `main`.

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

Active task: `task/journey-review-fixes` — a review-driven fix pass across
backend parity, Elm client UX, security, and documentation, ready for a
pull request. It covers:

- Backend correctness/parity: 16 findings fixed (transactional
  `ChangeTaskState`, refund/cancel and deactivation guards, and 11
  real-vs-WASM store divergences — reservation expiry/uniqueness,
  implementor bans, idempotency semantics, ledger pagination, series
  bookkeeping, team scope, unknown-task 404, strict pagination, fund audit
  events). See `WHAT_WE_DID.md`.
- Two-section wallet (replaces the escrow state machine): every user and
  organization credit account now has a **spendable** section
  (`sum(ledger_entries)`) and an **allocated** section
  (`sum(task_funds.credit_amount)`). Funding moves credits spendable ->
  allocated (a stateless `task_funds` row, no held/released/refunded state),
  so allocated credits cannot be double-spent. Finishing a task moves the
  funder's allocated credits to the worker's spendable balance; refunding or
  closing an un-awarded task returns them to the funder's spendable balance.
  Refunds are auto-granted for the task owner or the active reservation
  holder while the task is not yet awarded. Collectible rewards use the same
  stateless temp store (`task_fund_collectibles`) and keep the collectible's
  `escrowed` lifecycle state as the trade lock. Balance endpoints return
  `{spendable_credits, allocated_credits}`; the ledger kind `task_escrow`
  was renamed `task_fund`.
- Password hashing switched from PBKDF2 to Argon2id (OWASP first choice,
  m=19 MiB/t=2/p=1) via `golang.org/x/crypto/argon2`, matching what
  AGENTS.md/PLAN.md already documented.
- Security: password-reset/verification token delivery now defaults to `log`
  (fail closed) so production no longer returns tokens to anonymous callers;
  the demo opts into `api` explicitly. Password-reset no longer reveals
  whether an email is registered. Token-confirm endpoints are rate limited.
  User-directory search escapes LIKE metacharacters and caps query length.
  The demo shell's `<base href>` is rebuilt from safe path segments.
- Elm client: deep-link login now loads the target page; mid-session token
  refresh (with an explicit expiry logout); a schema-driven worker response
  form; a fix for the silent reward downgrade on create; disclosure panels
  no longer snap shut mid-edit; successes and failures are visually
  distinct; standalone-team create + browsable team pages; self-service
  privacy-request list; a deactivate-account confirmation step; and a batch
  of smaller fixes (trade/tip/funding gating, pagination end state, honest
  no-email copy, agent-scope reset).
- Demo seed: added a pending submission, an approval-required reservation
  request, an inbox notification, and a held collectible targeting `mara`
  so the single-actor demo can exercise the review/approval/collectible
  journeys.

Deferred follow-up: the Overview/organization pages now show the account's
allocated total, but the task DETAIL response still does not report a task's
live allocated amount, so the Refund button can still appear on an unfunded
declared-reward task (clicking it returns a clear "nothing to refund"
message). Exposing per-task funding state on the detail response to gate the
button is a smaller remaining follow-up. See `DO_NEXT.md`.

Blocking issues: none. GitHub Pages `deploy-pages` occasionally fails
transiently after a merge and clears on retry; it is not caused by
repository code.
