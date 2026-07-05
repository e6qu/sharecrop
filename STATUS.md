# Status

The repository contains pull request 1 through pull request 136 work, merged
into `main`, plus the current
`task/unpublish-escape-hatch-and-scenario-parity-ci` branch. PR 108's
GitHub Pages deployment failed three times in a row after
merge for what looked like a transient GitHub-side Pages backend issue
(build/artifact steps always succeeded; only `deploy-pages` failed or hung,
with a different symptom each time); most later PRs' deployments succeeded
on the first try with no code or workflow changes, confirming it was not a
code problem — though PR 127's deployment hit the same transient failure
again and cleared on a manual retry, so this class of flakiness is still
occasionally live, not fully resolved.

The 5-phase RBAC + API-token effort (PRs 115-121), two clean-up passes
(PRs 122, 124), a docs refresh (PR 123), the WASI production-hosting
spike's plan + Phase 0/1 (PR 125), ecosystem research (PR 126), and
deployment-shape requirements (PR 132), a Go 1.26.4 upgrade (PR 127), a
strengthened "at most one open PR at a time" rule in `AGENTS.md` (PR 128),
the `site/demo/backend.js` deprecation (PR 129: replacement CI coverage;
PR 130: deletion), a fix for the demo (WASM) backend collapsing every
rejection to HTTP 500 plus a corrected fund-panel visibility gate
(PR 131), a batch of task-funding/creation UX fixes (PR 133: fund any
reward kind, open a task after creating it; PR 134: collectible-reward UI,
org-admin collectible awards), and a task visual-language pass (PR 135:
state color-coding, "mine" highlighting, a funding-discoverability
callout, required-field validation, a multi-select state filter) are
complete.

PR 136 fixed two reservation bugs the user hit live (reported as "I can't
cancel reservation" and "the reservation drawer is not visible when
toggled off"), confirmed by real browser reproduction rather than
assumption, plus a follow-up request to show the reward as its own badge
in task lists with small icons on all badges. See its `WHAT_WE_DID.md`
entry for the full writeup (a worker couldn't see or cancel their own
reservation; the "Reserve" button never went away after reserving; new
reward badges + state-badge icons).

Active task: `task/unpublish-escape-hatch-and-scenario-parity-ci` traces
the user's next report ("the task view still! still! does not have a
visible drawer to change funding of a task") to its actual root cause,
verified live against the deployed demo, not guessed at:

- **Root cause found**: "a credit/bundle reward must be funded before a
  task can open" is enforced only in `internal/db/task_store.go`'s
  `requireOpenableReward` (the Postgres store layer) — never in the
  shared `internal/task/service.go` domain layer. `internal/wasmdemo` is a
  wholly separate reimplementation of task logic with its own storage and
  its own state-transition checks, and it never got this invariant added,
  so a demo task could reach "open" with a declared-but-unescrowed
  reward. Confirmed empirically (curl against a live local server) that
  this invariant is missing on the demo and present on the real backend
  before writing any fix.
- **Fixed the missing invariant** in `internal/wasmdemo/request_handler.go`'s
  "open" action, mirroring the real backend's check exactly (credit/bundle
  needs escrow matching the declared amount; collectible/bundle needs at
  least one held collectible).
- **Added a real escape hatch**: `Task.Unpublish` (`open` → `draft`) already
  existed on both backends (`POST /api/tasks/{id}/unpublish`) but was never
  wired to a UI button for individual tasks (only for task series) — an
  owner had no way to move a task back to draft to fix its funding and
  reopen it. Added the button, the Msg/Api/View wiring, and a Playwright
  test covering the full recovery flow (declare reward → reject premature
  open → fund → open → unpublish → fund panel available again → reopen).
- **Closed the systemic gap the user pushed on directly**: the shared
  `tests/scenario_parity/scenario.ts` script (meant to prove the two
  backends behave identically) was, until now, only ever run against the
  WASM demo in CI (`check-wasm-scenario-parity`) — `check:scenario-parity`
  (the real-backend variant) had no CI job at all, so a new assertion
  added there could silently diverge from real-backend behavior with
  nothing automated to catch it. Wired it into `tools/run_db_checks.sh`
  (spins up a real DB-backed server, runs the shared scenario against it)
  and added Deno setup to the `db-checks` CI job. Verified the new
  assertion is real, not vacuous, by reverting the fix and confirming the
  check actually fails (200 instead of 409) before restoring it.
- Bigger open question, raised by the user directly and not yet resolved:
  whether to unify onto a single WASM-compiled backend (used for both the
  browser demo and a multi-replica production deployment) so this class of
  invariant-duplication bug becomes structurally impossible rather than
  something each new check has to catch. Asked a clarifying scope question
  with no response yet; proceeded with the concrete, verified, low-risk fix
  above in the meantime rather than guessing at scope for a multi-session
  rewrite. See `DO_NEXT.md`.
