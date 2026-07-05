# Status

The repository contains pull request 1 through pull request 134 work, merged
into `main`, plus the current
`task/task-visual-language-and-multiselect-filter` branch. PR 108's GitHub
Pages deployment failed three times in a row after merge for what looked
like a transient GitHub-side Pages backend issue (build/artifact steps
always succeeded; only `deploy-pages` failed or hung, with a different
symptom each time); most later PRs' deployments succeeded on the first try
with no code or workflow changes, confirming it was not a code problem —
though PR 127's deployment hit the same transient failure again and
cleared on a manual retry, so this class of flakiness is still occasionally
live, not fully resolved.

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
org-admin collectible awards) are complete.

Active task: `task/task-visual-language-and-multiselect-filter` covers a
follow-up batch of task UI requests, grounded in a design proposal the user
reviewed and approved first (color palette, WCAG contrast checks, and
mockups for each change, before any code was written):

- **Task state color-coding.** Task list rows (My tasks, Discover public
  tasks) now show state as a colored badge (`taskStateBadge`, reusing
  `Ui.badgeVariant`'s existing 4-tone system) instead of plain text. Added
  a 5th tone, `info` (blue), for the `Closed` state, which previously
  shared "neutral" with `Draft` and wasn't visually distinct. All 5
  tones checked against WCAG AA (6.49:1 to 9.45:1 contrast).
- **"Mine" highlighting.** A task the viewer created or is the active
  assignee on gets a blue left-accent border and a small "MINE" tag,
  in both the My-tasks and Discover-public-tasks lists.
- **Funding discoverability fix**, reproduced live before fixing: the
  "Fund this task" control existed but was one unstyled collapsed
  disclosure line, visually identical to unrelated sections like
  "API & MCP" and "Report task" — easy to miss entirely. A brand-new,
  unfunded draft task now shows an open-by-default blue callout
  ("Before you open this task...") instead of a collapsed disclosure;
  once funded, or for a task that already declared some reward, the
  original on-demand disclosure is used as before.
- **Required-field validation on the create-task form.** Title/description
  now get a muted-red border + inline error message when a submit attempt
  fails with them empty, clearing per-field as each is filled in. Found a
  real WCAG tension: a genuinely muted border can't hit the 3:1 non-text
  contrast target alone (neither do this app's existing borders), so color
  is paired with an icon + explicit text rather than relying on hue alone
  (WCAG 1.4.1).
- **Multi-select task-state filter**, replacing single-select buttons that
  only covered 3 of 5 states. Required a real backend change, not just a
  UI change: the server only supported one `state=` value
  (`task.StateEquals`); added `task.StateIn` plus a `tasks.state = some(@filter_states)`
  SQL clause (`internal/db/task_store.go`), and `internal/http/tasks.go`
  now reads repeated `state=` query params. Mirrored in
  `internal/wasmdemo`'s `ListTasks` (now takes `[]string`).

All changes covered by new tests (Go integration/http_e2e for the
reward-kind and multi-state backend changes; Playwright for the visual
changes) and a full local check-suite + Playwright run before commit.
