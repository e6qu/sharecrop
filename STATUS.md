# Status

The repository contains pull request 1 through pull request 135 work, merged
into `main`, plus the current `task/reservation-fixes-and-reward-badges`
branch. PR 108's GitHub Pages deployment failed three times in a row after
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

Active task: `task/reservation-fixes-and-reward-badges` fixes two reservation
bugs the user hit live (reported as "I can't cancel reservation" and "the
reservation drawer is not visible when toggled off"), confirmed by real
browser reproduction rather than assumption, plus a follow-up request to
show the reward as its own badge in task lists with small icons on all
badges:

- **A worker could never see or cancel their own reservation.**
  `task.Service.ListReservations` (`internal/task/service.go`) rejected
  anyone but the task owner outright, so a worker who reserved a task got
  an empty reservation list back — their own reservation (and its Cancel
  button) simply never rendered. Widened it: the owner still sees every
  reservation; anyone else sees only their own.
- **Even once visible, Cancel 403'd.** `CancelReservation` shared its
  permission check with `ApproveReservation`/`DeclineReservation`
  (owner-only), but cancelling is meant to be available to the reservation's
  own holder too, not just the owner force-cancelling it. Gave
  `CancelReservation` its own permission path: owner, or the actor who
  requested that specific reservation.
- **The "Reserve" button never went away after reserving.** The task
  detail page's `viewer_action` (`internal/http/tasks.go`) was computed
  purely from the task's own state/policy, never checking whether the
  viewer already held a reservation — so the Reserve/Request-approval
  button stayed forever, inviting a pointless second (server-rejected)
  attempt. `taskToResponseForActor` now overrides it to `wait` once the
  actor already has a requested/active reservation on the task.
- **Reward badges + state icons.** The reward now renders as its own
  small badge (new "reward"/purple tone, 7.39:1 contrast, `◆` icon) next
  to the state badge in task list rows, instead of muted trailing text.
  Each of the 5 state badges also got a small decorative icon
  (`aria-hidden`, since the badge's own text already names the state per
  WCAG 1.4.1): `●` open, `○` draft, `✓` closed, `✕` cancelled, `⏳` expired.

All changes covered by new tests (a Go http_e2e case for the
list/viewer_action fixes; Playwright for the cancel flow and the reward
badge) and a full local check-suite + Playwright run before commit. Found
via live reproduction with real browser screenshots per the user's request
("take screenshots to debug and fix"), not guessed at.
