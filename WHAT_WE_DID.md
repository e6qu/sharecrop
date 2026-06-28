# What We Did

`task/org-team-assignment` made organization-team assignment workable across the main interfaces:

- **Domain and permissions.** Task reservation now has an organization-team path in addition to the existing user path. The organization service exposes a team-membership check that verifies the team is organization-owned, belongs to the requested organization, and includes the acting user.
- **Submission eligibility.** Non-open tasks now treat an active organization-team reservation as satisfying eligibility for users who belong to the reserved team.
- **HTTP and MCP.** `POST /api/tasks/{task_id}/reservations` still accepts an empty body for user reservations, and now accepts `{"assignee_kind":"organization_team","organization_id":"...","team_id":"..."}` for organization-team reservations. `sharecrop.reserve_task` accepts the same optional assignee fields.
- **Browser and demo.** The task detail reservation card shows organization/team ID inputs for `organization_team` tasks and posts those IDs when reserving or requesting approval. The demo backend accepts team reservations and checks seeded team membership; generated browser bundles were refreshed.
- **Tests.** Unit tests cover organization-team reservation success and non-member rejection. A new HTTP e2e test covers organization team creation, member reservation, outsider rejection, and member submission. Local verification passed for `go test ./...` and `make frontend`; the new database-backed HTTP e2e test could not run in this environment because `DATABASE_URL` is not set.

`task/application-completeness-review` documented a project readiness review:

- Added [docs/application_readiness_review.md](./docs/application_readiness_review.md), comparing the implemented backend, Elm UI, HTTP API, MCP surface, tests, and user stories against the product thesis.
- The review found that the registered requester/worker task loop is implemented and tested, but the application is not yet ready for ordinary production use. The highest-priority gaps are organization-team assignment, organization reviewer browser parity, worker submission status/discussion, coherent reward creation, account lifecycle, operations, and product/API/MCP docs.
- Updated [DO_NEXT.md](./DO_NEXT.md) with the review-driven priority queue.
- Corrected stale user-story and bug/test-gap notes: collectible review tips and browser organization-member listing exist; organization-team assignee selection exists but is not workable end to end; browser organization roles and worker submission UX remain partial.
- Verification: `make check-policy` and `make test-deno` passed. A docs-only `deno fmt --check` probe was not treated as required because Markdown is outside the repository's `make check-format` target and would reflow old continuity text unrelated to this branch.

`task/bundle-refund-ui-parity` corrected the bundle-refund UX and a stale BUGS claim:

- Investigation found the "no one-shot bundle refund" risk recorded in BUGS was wrong: the credit `/refund` endpoint already calls `refundHeldCollectibleReward` inside its transaction, so it returns held credits AND collectibles together (covered by `TestBundleRefundReturnsCreditsAndCollectible`). The actual gap was UI-only — a bundle task rendered both "Refund credits" and "Refund collectible", and the collectible one 409'd on bundle.
- **Owner refund controls** now offer exactly one refund action per reward kind: credit → `/refund` ("Refund credits"), collectible → `/collectible-refund` ("Refund collectible"), bundle → `/refund` ("Refund reward", returns credits + collectible together). The dead bundle "Refund collectible" button is gone.
- **Demo parity.** `site/demo/backend.js` `/refund` now releases escrowed collectibles too (mirroring `refundHeldCollectibleReward`), so a demo bundle refund returns everything in one call; the cancel guard and collectible-refund routes already matched.
- **Tests.** A real-backend Playwright flow escrows credits + collectible on a bundle task, opens it, refunds via the "Refund reward" UI button, and asserts both balance restoration and the collectible returning to holdings. Stabilized `openTaskFromDiscovery` (network-idle + 15s balance wait) against intermittent shared-server load flakiness that had been timing out post-login navigation.
- Removed the stale BUGS entry.

`task/fix-cancel-escrow-guard` closed the active orphan-escrow-on-cancel bug surfaced in the previous PR:

- **Cancel now rejects while escrow is held.** The task store's `ChangeTaskState` gained a `requireNoHeldEscrow` guard on the cancellation path: it counts held credits (`task_escrows.state = 'held'`) and held collectibles (`task_collectible_rewards.state = 'held'`) and returns a 409 "refund the task's held escrow before cancelling" when either exists. Previously the `Cancel` state transition left held escrow stranded against a cancelled task with no return path. This is the documented "reject Cancel while escrow is held" option. http_e2e covers a funded task returning 409 on cancel and a subsequent successful refund. The browser already routes funded tasks to Refund, so the only behavioural change users see is the rare funded-draft Cancel now surfacing that 409 (with the Refund action alongside) instead of silently orphaning.
- **Demo parity.** The `site/demo/backend.js` cancel route now rejects when the task holds escrow (`escrow > 0` or any escrowed collectible), matching the real backend's precondition.
- **Playwright helper race fix.** `openTaskFromDiscovery` in `screens.spec.ts` intermittently timed out clicking `nav-discovery` because it ran before the post-login data load finished. It now waits for the balance to render before navigating; the helper and `loginViaUi` were retyped from ad-hoc structural types to the real Playwright `Page`.
- **Stale-doc cleanup.** Removed a BUGS entry claiming the demo deep-link 404s on GitHub Pages — that was already fixed by fragment (hash) routing in PR #61. Documented the residual bundle-refund gap (no one-shot refund for bundle rewards) as a known risk.

`task/ui-cancel-collectible-tip` exposed the task-lifecycle and review actions the HTTP API already supported but the browser lacked:

- **Cancel a task.** The owner controls gained a Cancel button. It is offered for draft tasks (any reward) and for open no-reward tasks. Reward-bearing OPEN tasks are ended via Refund instead: the backend's `Cancel` (task state transition) does not return held escrow, while `RefundTask` does — so gating Cancel away from funded tasks avoids orphaning escrow. (That Cancel-of-a-funded-task gap is recorded in BUGS as a backend-level risk.)
- **Collectible tip on accept.** The review form gained a "Tip a collectible" select populated from the requester's holdings; the selected collectible is sent as `tip_collectible_id`, which the backend's accept handler gifts to the worker via `GiftCollectible`. The selection is reset alongside the rest of the review form after each review action (extending the existing review-form reset).
- **Refund a collectible reward.** Owner controls offer "Refund collectible" for draft/open collectible- or bundle-reward tasks, wired to `POST /api/tasks/{id}/collectible-refund`.
- **Owner-controls gating rewrite.** The buttons are now built from a single `List.filterMap identity` over state/reward conditions, replacing the prior exclusive if/else (which showed only ever one of Open/Refund). Open is draft-only; Cancel/Refund-credits/Refund-collectible appear contextually.
- **Demo parity.** `site/demo/backend.js` implements `POST /api/tasks/:id/collectible-refund` (returns escrowed collectibles to the requester, resets the task reward kind) and honors `tip_collectible_id` on accept (gifts the collectible, sets `payout_kind`/`collectible_ids`/`worker_user_id`).
- **Tests.** Two real-backend Playwright flows: an owner cancels a no-reward task (asserts Cancel visible, Refund hidden, "cancelled" message); an owner tips a transferable collectible on accept (asserts the tipped collectible leaves their holdings). Backend collectible-tip and collectible-refund semantics were already covered by http_e2e (`TestCollectibleTipTransfersOnAccept`, `TestCollectibleRewardRefundReturnsToOwner`, `TestBundleRefundReturnsCreditsAndCollectible`).

`task/polish-bugfix-uiux-review` was a combined bug-sweep + UI/UX review pass driven by three parallel review agents (Go backend, demo-vs-real drift, Elm client), with the findings fixed in one branch:

- **Elm review-form state leak (HIGH).** The review form (note / partial credit / tip / ban) carried across task→task navigation, across discovery→detail, and from one submission to the next within a task — so rejecting submission A with ban=true silently re-banned submission B. Fixed by resetting the four fields in `enterPage` (TaskDetailPage), `DiscoveryViewClicked`, and the `ReviewActionReceived` Ok branch. Also added missing `enterPage` resets for `CollectiblesPage`, `CreateTaskPage`, and `FundingPage` (the same leak family: stale award/mint/create/fund messages and prefilled drafts reappeared on return).
- **Stale task detail after refund (HIGH).** `RefundTaskReceived` only refreshed the task list + ledger, leaving the detail card badge on open/funded next to a "Task refunded" note. It now refetches the detail (and reservations/submissions) via `refreshAfterAccept`.
- **Perpetual "Loading…" on bad deep-links (MED).** A forbidden or failed task detail wrote its error to `submitMessage`, which is hidden when `detail == Nothing`, so the page hung on "Loading task…" forever. Added a `detailError : Maybe String` field; the detail card now renders the error.
- **Token-mint errors surfaced (MED).** `TaskTokenMinted`/`UserTokenMinted` error branches returned `(model, Cmd.none)` — a no-op button. They now surface a friendly error.
- **Dead/no-op controls gated by state (MED).** Owner Open/Refund rendered for every task state (clicking them on an open/closed task was server-rejected); they now render only for states they can act on (Open: draft; Refund: draft/open). The worker submit form rendered on closed/cancelled/refunded tasks; it now renders only when the task is open. (Gating is on `state` only — `viewerAction` is viewer-independent in both the real backend and the demo, so it cannot express "this viewer has reserved".)
- **Go backend cleanup.** Deleted the dead `requireAdmin` helper (the single admin gate is inline in `collectibles.go`). `writeSeriesDetailStatus` now propagates a `ListSeriesComments` rejection through `writeDomainError` instead of swallowing it and returning an empty comment list. The receipt-status handler routes through `writeDomainError` (was a hardcoded 404). The `/mcp` body read uses `http.MaxBytesReader` so an oversized body returns 413 instead of being silently truncated into a misleading "not valid JSON-RPC" 400.
- **Demo fidelity (`site/demo/backend.js`).** Reject no longer closes the task (matches prod's `closeTask: false`) and now releases the rejected worker's reservation to `cancelled_by_requester`; reject/request-changes now require an open task; `payout_kind` reflects the actual payout (not the reward kind), `worker_user_id` is populated only when a payout matched, and request-changes reports `payout_kind: "none"`; refund returns the released `amount` (was 0); the ledger seed reconciles with the balance (1250, was off by 10); PATCH task-series no longer wipes the description when the field is omitted; awarded collectibles use `state: "awarded"` (was "minted").
- **Reviews done.** Go backend lens (authz/IDOR, ledger, input bounds, dead code — clean apart from the items fixed), demo drift lens (state machine, economy, enum/shape, seeds), Elm client lens (state leak, stale data, dead controls, decoder/error, a11y). A visual screenshot set was generated to `/tmp/sharecrop-review-screens` but the agent could not inspect images; the user should review those captures.
- **Deferred (recorded in BUGS/DO_NEXT):** Team/Series/User detail load-vs-error distinction (only TaskDetail was upgraded); demo `reservationChange`/reserve still skip the ownership + assignee-scope guards; `type_ "button"` on assorted secondary buttons and free-text-id → picker follow-ups.

Follow-up commit on the same branch tackled the deferred load-vs-error and demo-guard items:

- **Detail load-vs-error extended.** TeamDetail, SeriesDetail, and UserProfile now carry their own `*Error : Maybe String` field and render the error message on a failed/forbidden fetch instead of hanging on "Loading…". The `SeriesDetailReceived` handler was rewritten to a case (the `seriesRenameTitleFor`/`seriesRenameDescriptionFor` helpers became dead and were removed).
- **Demo reservation guards.** `site/demo/backend.js` `reserve` now rejects non-user-scoped tasks ("this task does not accept user reservations"), and `reservationChange` requires the task requester (`created_by === ME`, else 403) and only transitions reservations in `requested`/`active` states (else 409) — matching the real backend's `changeReservationByRequester` + store guard.
- **Client-side validation.** The submit form rejects empty or non-JSON input before posting; the fund form rejects non-positive amounts.

`task/backlog-cleanup` cleared bounded backlog deferrals and applied a UI/UX + QA boyscout review (one background review agent):

- **Admin-panel gating.** The auth response now carries a `role` ("admin"/"member", stamped from `SHARECROP_ADMIN_USER_IDS` in `writeAuthResponse`; contract field added as a string since the codebase bans `Bool` in contracts). The client stores `isAdmin` and **hides the "Admin: award" panel and the catalog Award buttons for non-admins** (the catalog stays browsable). The demo's auth role is `admin`, so the showcase keeps them.
- **Back-button regression (critical).** The task-detail Back button used a non-fragment href (`/tasks`), which after the hash-routing switch dumped users on Overview. Now `#/tasks` / `#/discovery`.
- **Go status codes.** `getTask` returned 403 for a *missing* task — now `writeDomainError` (real 404). A sweep replaced hardcoded `writeError(w, http.Status…, reason.Description())` with `writeDomainError(w, reason)` across the organization/series/collectible/org-credit handlers, fixing contradictory siblings (one list endpoint 403, its twin 500) and wrong 400s so each rejection maps to its correct status. Validated by the http_e2e status assertions.
- **Dead/no-op controls.** The "Submit a response" form no longer renders when the task failed to load (was a live form posting to an unreadable task); review controls (note/payout/tip/ban) render only when there are submissions (was a full review form above "No submissions").
- **Demo user-submissions** endpoint returns the user's real submissions (was always `[]`).
- **Deferred (noted):** id-picker dropdowns for free-text scope ids; org-reviewer review controls in the browser (needs a `viewer_action` "manage" value); per-page loading-vs-error states (a forbidden deep-link still shows a perpetual "Loading…"); plus the three large standalone initiatives (out-of-process Postgres session/rate-limiter store, anonymous-worker identity, crypto reward metadata) kept in DO_NEXT. The QA review found **no WCAG contrast failures** and confirmed full demo route/decoder parity.

`task/demo-fidelity` minimized the in-browser demo's "fakes" so `site/demo/backend.js` behaves like the real Go backend. A specialized agent compared all ~69 demo routes to the Go handlers/domain; the high-impact divergences are fixed:

- **Review state machine.** accept/reject/request-changes now act only on `submitted` work (409 otherwise); accept additionally requires an open task and rejects when a submission was already accepted — so double-clicking Accept or reviewing an `invalid`/seeded submission no longer replays payouts and drifts the balance.
- **Lifecycle + economy guards** matching prod's 409s: open (draft only), cancel (draft/open), unpublish (open only); refund (draft/open + must have escrow, and it returns to the funding wallet); funding (rejects re-funding an already-funded task — escrow is a single hold); reservation (open state, non-open policy, and the requester can't reserve their own task).
- **Create fidelity.** task-create honors the `owner` (so org-owned tasks store `owner_kind`/`owner_id`), the `visibility` scope id, the `assignee_scope`, and `reservation_expiry_hours` — previously all dropped, which made org-owned/org-scoped tasks look user-owned and miss their org page.
- **Series consistency.** add/reorder set `series_kind = "existing_series"` and remove resets to `"standalone"`; the seed values were aligned (was `"existing"`).
- **Seed nit.** the seeded "Golden Sickle" collectible is now `transferable_between_users`, matching its catalog template (was inconsistently non-transferable).
- **Intentional demo affordances kept** and documented in the file header: auto-login on `/api/auth/refresh`, the Reset button, and unvalidated tokens with a single seeded "you" (Mara).
- The 31-test Playwright suite still passes (no flow regressions). Deferred (low impact, noted): the user-submissions page stub and member/team provisioning showing a synthetic id rather than the typed email.

`task/demo-pages-routing` fixed the GitHub Pages demo URL/refresh problem cleanly (no fallback), and hardened logout:

- **Fragment (hash) routing.** The demo runs at `/sharecrop/demo/` but the client built root-absolute paths, so click-navigation left the base and hard-refresh/deep-links 404'd (Pages bounced to the site root). Rather than add a 404.html SPA fallback + base-path threading (a "needless fallback" that hides bugs), the router now keeps the whole route in the URL **fragment** (`#/...`). The path stays a real file, so hard-refresh and deep-links work with **no 404.html, no base-path config, and no Go SPA catch-all**. `pageFromUrl` reads `url.fragment`; every internal href is `#/...`; the two path-building `Nav.pushUrl` calls became `#/...`.
- **Explicit NotFoundPage.** The router's `_ -> OverviewPage` catch-all silently turned bad links into Overview. It's now an explicit `NotFoundPage` (root `""` → Overview; unknown → NotFound), so dead links are visible instead of masquerading.
- **Demo-only Reset button.** A new **required** `demo : Bool` flag (no implicit default) — `web/static/index.html` passes `false`, `site/demo/index.html` passes `true`. When true, the nav shows a "Reset demo" button; a `reloadDemo` port reloads the page, which re-seeds the in-browser backend and auto-logs-in. Shown only in the demo.
- **Real logout revokes the session.** Logout previously only cleared the cookie; the refresh token stayed valid server-side. Now `auth.Service.Logout` → store `RevokeRefreshFamily` revokes the whole token family (mirroring the reuse-detection revoke), and the handler reads the cookie + revokes before clearing it. Re-login is not blocked. http_e2e confirms a post-logout refresh is rejected and a subsequent login succeeds.
- **Tests.** Deep-link Playwright navigations switched to `/#/...`; added hash hard-refresh + NotFound + reset presence/absence tests and the logout-revoke http_e2e test.
- **Deferred to the next PR (your broader ask):** a demo-fidelity QA pass to minimize the `site/demo/backend.js` fakes so the in-browser demo behaves as close to a real deployment as possible.

`task/uiux-journey-review` was a thorough UI/UX + user-journey review round (two specialized subagents) with boyscout fixes, and it recorded a product decision:

- **Scheduling/recurrence descoped server-side** (decision 2026-06-25): recurring/scheduled task posting is a local-agent responsibility (a client cron/work-loop calling the existing MCP/API `create_task`/`open_task`/`fund_task`). No server scheduler/`task_schedules`/recurrence model. Recorded in `DO_NEXT.md` and the parity-roadmap memory so it isn't re-proposed.
- **No contrast failures** were found across the newest surfaces (collectibles gallery/award/trade, org/team holdings, submission comments, template/schema designer) — verified with computed WCAG ratios.
- **Bug fixes** (all from the review):
  - Re-funding a task was a silent no-op — `fundingRequestBody` hardcoded `idempotency_key = "fund:" ++ taskId`, so a second funding replayed. Now keyed per attempt (`fundNonce`), so adding escrow works while network retries stay idempotent.
  - Award feedback cross-contaminated: award-to-task and admin award-default shared `awardMessage`. Split into `awardMessage` / `awardDefaultMessage`.
  - `transferMessage`/`transferRecipientId` leaked across collectible detail pages — added a `CollectibleDetailPage` reset to `enterPage`.
  - The task-detail card stayed stale after accept/reject/request-changes — `refreshAfterAccept` now refetches the task detail + reservations.
  - Task-comment add now guards empty bodies and surfaces errors (was posting blank / swallowing failures), matching the submission/series threads.
  - The catalog Award and "Award to selected task" buttons are disabled until a recipient/task is chosen.
  - Exclusive chooser buttons (participation/visibility/assignee/owner/collectible-kind/policy/award-kind/state-filter) gained `aria-pressed`.
  - The admin-award 403 shows a friendly message plus an admin-only note on the panel.
  - Demo: removing a task from a series sets `series_id = ""` (not `null`), which previously blanked the strict-decoded task detail.
- **Deferred (flagged):** full `is_admin`-on-session gating to hide the admin panel for non-admins; org-reviewer review controls in the browser (today only the literal creator sees them); free-text id inputs → picker dropdowns; org/team holdings auto-refresh right after an award.

`task/submission-comments` added a private comment thread on each submission (PR 2 of the backlog sequence), mirroring the existing task-comment vertical end to end:

- **Domain/store/HTTP:** migration `000022_submission_comments`; `core.SubmissionCommentID`; `submission.AddSubmissionComment`/`ListSubmissionComments` with a visibility rule that permits only the submission's author (worker) or the owner of its task (requester); `GET`/`POST /api/submissions/{id}/comments`.
- **MCP:** `sharecrop.add_submission_comment` (submissions:write) and `sharecrop.list_submission_comments` (submissions:read).
- **Contracts:** `SubmissionCommentResponse` (id, submission_id, author_user_id, body, created_at) + `SubmissionCommentsResponse`.
- **Client/demo:** each submission row on the task detail has a "Comments" toggle opening its thread (list + add box); the demo backend serves the routes and seeds a comment.
- **Tests:** submission domain unit, http_e2e (owner + submitter post/list; unrelated user → 403), Playwright (owner comments on a worker's submission).
- **Deferred:** a dedicated worker-side submission view in the client — the backend + MCP already let the worker participate; only the owner-side UI shipped here.

`task/admin-collectible-ownership` finished the two collectible follow-ups — a real admin gate on awarding, and real org/team ownership (PR 1 of a sequence burning down the follow-up + roadmap backlog):

- **Platform-admin role:** the server reads `SHARECROP_ADMIN_USER_IDS` (comma-separated user ids) into an admin set (`parseAdminUserIDs`) and a `requireAdmin` helper. `POST /api/collectibles/award` now returns 401 (unauthenticated) / 403 (authenticated non-admin); only an admin can mint catalog copies. The demo award panel is unchanged (the demo has no real auth, so it stays the showcase).
- **Org/team ownership:** migration `000021` adds `owner_kind` to collectibles and drops the users foreign key on `owner_user_id` so it can hold any owner entity's uuid. `assets.Collectible` now has `OwnerKind string` + `OwnerID string`; the fund/tip/transfer flows require `OwnerKind == "user"`. Awarding accepts `recipient_kind` ∈ {user, team, organization}. New `GET /api/organizations/{id}/collectibles` and `GET /api/teams/{id}/collectibles` (service `ListByOwner`, store `ListCollectiblesByOwner`), surfaced as a "Collectibles" holdings section on the org and team detail pages. `CollectibleResponse` gained `owner_kind` (contracts regenerated); the demo seeds/award/mint set `owner_kind`.
- **Tests:** http_e2e covers the admin gate (non-admin 403), award to a user and to a team, the team holdings endpoint, and trade-back; assets unit + integration updated for the new owner fields; the store's list loop was extracted to a shared helper to satisfy copy-paste detection.
- Bootstrap note: the http_e2e admin test registers users on one server, then rebuilds with `SHARECROP_ADMIN_USER_IDS` set to the admin id (the shared test DB persists registrations).

`task/default-collectibles` added 25 hand-crafted pixel-art default collectibles with an admin award flow and user-to-user trading, in a single PR:

- **25-item catalog** (`internal/assets/catalog.go`): a farm/harvest-themed set where `kind` doubles as rarity — 15 badges (common), 5 editions (rare), 5 unique (legendary), all tradeable. Each carries an `art` slug.
- **Pixel sprites** (`web/elm/src/Sharecrop/Sprites.elm`): each of the 25 is a hand-authored CSS pixel-art icon (`pixel slug cell` → a grid of colored cells), themed to the arcade palette. Shown in the catalog gallery, on holdings rows, and large on the detail page.
- **Model/contracts:** collectibles gained an `art` field — migration `000020`, `assets.Collectible.Art`, `Mint(..., art)`, `CollectibleResponse.art`, plus generated `CollectibleCatalogEntry`/`CollectibleCatalogResponse` types.
- **Real backend endpoints:** `GET /api/collectibles/catalog`; `POST /api/collectibles/award` (mints a catalog copy owned by a user; team/org recipients are rejected with a demo-only note); `POST /api/collectibles/{id}/transfer` (user→user, reusing the policy-enforced `GiftCollectible`).
- **Demo (full showcase):** `backend.js` seeds the 25-entry catalog, implements award to user/team/org (owner_kind) and transfer; the Collectibles page has an admin award-recipient control and the gallery; awarding to yourself surfaces the copy in your holdings; trading moves it on. The transfer confirmation is rendered at the detail-card level so it persists after the traded item leaves your holdings.
- **Tests:** `assets` unit, `http_e2e` (catalog count, award, holdings, trade-back, team-award rejection), Playwright (`demo.spec` gallery + award + trade); the mobile overflow test now also covers the gallery.
- **Deferred (flagged in the PR):** real-DB org/team *ownership* of collectibles (the real backend keeps user ownership; org/team awarding is demonstrated in the demo) and a real system-admin role (award is an ungated faucet for now).

`task/create-template-menu` reframed task creation around templates and applied a batch of usability fixes from a specialized UI/UX review:

- Create-task: the "Task type" select became a **Template** menu — "Freeform (no template)" or a named template (Code review, Security review, Product review, UI/UX review, QA testing). Freeform shows the structured schema designer; selecting a template hides the designer, prefills the description + response schema, and shows a note explaining that. `CreateTaskTypeChanged` now clears `createSchemaFields` when applying a template (and resets to a freeform schema when switching back), which fixes the designer-vs-raw-JSON silent-clobber the review flagged.
- `createMessage` leak (review P0): the task-detail owner-controls card now has its own `taskActionMessage`, so "Created task X" no longer appears under owner controls and "Task opened/refunded" no longer appears under the create form.
- Reservation-expiry field (review P1): shown and validated only when participation is `reservation_required`/`approval_required` (shared `Labels.participationUsesReservation`), instead of always-on and always-blocking.
- Labels (review P2): `kindLabel` returns human ledger labels and the ledger renders signed, colored amounts; `collectibleKindLabel` returns "Unique"/"Edition"/"Badge"; a new `scopeLabel` names agent scopes, and the scope checkbox uses the shared styled checkbox via `onCheck`.
- Stale task detail (review P3): `enterPage` clears the previous task's detail/submissions/reservations/comments/token state for `TaskDetailPage`, so a task→task link no longer flashes the prior task's content.
- Generic reference-URL helper. Playwright covers the template menu (prefill + designer toggle); label/message assertions updated.
- Deferred from the review (noted, not done this PR): chooser buttons → radio/aria semantics, org-ID free-text → picker dropdowns, award-collectible disabled-state, a unified secret-display convention.

`task/round-fuzz-mobile-design` was a fuzz + mobile/UI-UX/contrast + demo-functionality review round (two specialized subagents) with a design-surface increase and boyscout fixes:

- Fuzz: added `FuzzAgentValueParsers` (internal/agent) over the credential value parsers — the scope enum, the label, and the secret, which base64-decodes untrusted input. Crash-free; the existing schema/value/token/parsePage/task-value targets were re-run.
- Design/edit surface: the structured response-schema designer gained `enum` (comma-separated allowed values) and `array` (item type) field kinds, alongside the scalar kinds. `SchemaFieldDraft` carries `itemKind`/`enumValues`; `encodeFieldSchema` emits the right schema per kind. The designer can now express the developer-template schemas (which use enum/array), not just flat scalar objects.
- Mobile/UI-UX (from the review): `Ui.fieldClass` gained `min-h-[44px]` so all inputs/selects meet the 44px tap target; the schema-row Required checkbox now uses the shared styled `checkboxClass` instead of a tiny raw input; `copyButton` is full-width on a phone; the designer helper text moved from slate-500 to slate-600 for contrast headroom. The mobile Playwright test now opens the task API/MCP panel and mints a token, and visits the profile agent-access card, asserting no horizontal overflow on those long-code-block surfaces. The contrast review found no WCAG failures.
- Usability: added a "Profile" nav link to the user's own page — last round's user-page token had no in-app navigation to reach it.
- Demo boyscout (from the demo audit): opening the demo from `file://` no longer renders `null/mcp` commands (both index.html files fall back to a placeholder origin); org-owned task funding debits the organization wallet (not the personal balance), and refunds/tips return to whichever wallet funded the task. The audit otherwise confirmed full route/decoder parity and that prior fixes (economy, seeds, validator) hold.

`task/user-page-token` put a personal agent token and MCP install commands on the user's own profile page:

- On `/users/{id}` when the viewer is that user (`userId == subjectId`), a "Your agent access" card mints a full-capability agent credential inline (scopes tasks_read/tasks_write/submissions_read/submissions_write/submissions_review) and shows the real token in a copyable code block with a Rotate button. The section is omitted entirely when viewing someone else's page, so the token is owner-only.
- Below the token, copyable MCP install/update commands with real values (no placeholders): `claude mcp add --transport http sharecrop <origin>/mcp --header "Authorization: Bearer <token>"` (Claude Code), an update form (`claude mcp remove sharecrop && claude mcp add ...`), and the `.mcp.json` config block (reusing `mcpConfig`, for Codex / Claude Desktop / generic clients). Copy buttons reuse the clipboard port from the previous PR. Reuses `mintTaskToken`'s pattern with a new `mintUserToken`/`UserTokenMinted`/`userAgentToken`.
- Playwright covers minting on your own page (token + install commands present and placeholder-free) and that the section is absent on another user's page.

`task/task-integration-panel` made the task's API/MCP instructions uniform, collapsible, and placeholder-free, with one token for both surfaces:

- A single agent token now drives both REST and MCP. New `requireWorkerSubject(r, scope)` (in `internal/http/server.go`) accepts either a user access token or an agent credential that holds the required scope, resolving to the credential's owning user (the same way MCP already treats an agent credential). It is applied to the worker REST endpoints the task instructions demonstrate: `GET /api/tasks/{id}` (tasks_read), reserve and submit (submissions_write). An http_e2e test proves an agent token works on those endpoints and that a token missing the scope is rejected.
- The task-detail "API & MCP" section is now a collapsible panel (collapsed by default). Inside, a "Create agent token" button mints an agent credential inline and shows the real, copyable token; below it, uniform REST and MCP entries each have a one-line description of what the command does, a Copy button, and real values (origin, task id, token) — no `<...>` placeholders. MCP is presented as the `.mcp.json` install snippet plus the JSON-RPC tool-call bodies (the client manages the session, so there is no manual `Mcp-Session-Id`); REST is the equivalent curl using the same token. Copy buttons use a new `copyToClipboard` Elm port (Main became a `port module`), wired in both `web/static/index.html` and `site/demo/index.html`. The old placeholder curl examples were removed.
- Playwright covers the collapsed-by-default panel, minting a real token, and that the rendered commands embed the token with no placeholders.

`task/mobile-demo-design` was a mobile-usability + demo-completeness + design-surface round (two specialized review subagents) plus a re-run fuzz pass:

- Demo full functionality: the in-browser fake backend now moves credits on review. Accept releases the held escrow, records a `task_payout`, refunds any unpaid remainder to the balance, and charges a tip — previously it closed the task without moving any credits, so the seeded reward economy had no payoff. Reject/request-changes honor partial credit + tip and refund the rest (request-changes leaves escrow held). Added the missing `POST /api/tasks/:id/unpublish` demo route. Seeded task-7 with a `task_type`, `reference_url`, and a comment so those recently-added surfaces are visible without creating a task.
- Mobile: added a Playwright mobile project (`mobile.spec.ts` at 375x667) asserting no horizontal overflow across every page; it caught genuine overflows (the collectibles policy/award buttons, long-ID rows). Fixes: responsive outer page padding (`p-4 sm:p-8`) and card padding; 44px-min tap targets on buttons via `Ui` button classes; action rows wrap with shrink-0/min-w-0; review buttons stack on mobile; long-ID rows get `min-w-0`/`break-words`; inline forms let the field grow/shrink (`fieldLabel` gains `grow min-w-0`); arcade buttons may wrap. The collectible transfer-policy labels became human-readable ("Transferable within organization" instead of `transferable_within_organization`), which also removed an unbreakable-token overflow.
- Design/edit surface: the create-task form gained a structured response-schema designer — add fields (name, type, required) and the schema JSON is generated for you, with the raw JSON kept as an advanced fallback; the field rows stack on mobile.
- Fuzzing: re-ran the suite to confirm the new MCP tools (covered by `FuzzHandleRaw`) and all value parsers stay crash-free.

`task/fuzz-journeys-uiux` was a fuzz + user-journey + UI/UX review round (two specialized subagents) with opportunistic boyscout fixes:

- Fuzzing: `internal/task/fuzz_test.go` `FuzzTaskValueParsers` exercises the value parsers added for templates/series/comments (task type, reference URL, comment body, series state/description) — no panics, and an accepted reference URL is always an absolute http(s) URL (confirming the real backend rejects `javascript:` and relative URLs).
- Journey/flow fixes: the demo `validateValue` understood only the designer/seed schema dialect (fields as a map, `items`, `required`), so tasks created from the new templates (canonical `fields` array, `item`, `presence`, `kind:"enum"`) had their submissions silently accepted; it now handles both dialects (and a real `enum` kind). The flagship seeded series `series-orchard` was published with a "tasks added here" comment but had zero member tasks; linked two seeded tasks (positions 1-2). A worker who submits an invalid response now sees the field-level `path: message` errors, not just "(invalid)".
- UI/UX fixes (contrast verified clean by the reviewer): comment authors render as links to their user page instead of a raw UUID; long reference URLs and comment bodies wrap (`break-all`/`break-words`) instead of overflowing the card; the task reference link opens in a new tab with `rel="noopener noreferrer"`; and arcade-theme anchor buttons (Open/View/Back) now get the chunky pixel-button treatment for consistency.

`task/dev-templates-comments` added pre-baked developer task types, a typed reference URL, and per-task comments:

- A task now carries a `task_type` (general, code_review, security_review, product_review, ui_ux_review, qa_testing) and an optional `reference_url` (validated as an absolute http(s) URL — the specific pull request or resource to work on). `migration 000019` adds both columns (with a CHECK on the type and an index), plus a `task_comments` table. New domain: `TaskType`, `ReferenceURL`, `TaskComment`, `core.TaskCommentID`, and viewer-gated `AddTaskComment`/`ListTaskComments`.
- The create-task form gained a task-type picker that prefills the description and response schema from a client-side template catalog (each developer type ships a description skeleton and a ready-made object schema), plus a reference-URL field; the task detail shows the type badge, a clickable reference link, and a comment thread. `create_task` (HTTP + MCP) accepts `task_type`/`reference_url`, task responses and `get_task` expose them, and new `add_task_comment`/`list_task_comments` MCP tools plus `GET/POST /api/tasks/{id}/comments` back the thread. Generated `TaskResponse` gained the two fields; new `TaskCommentResponse`.
- The demo fake backend stores the type/reference and serves the comment endpoints. Tests: an http_e2e round-trip (type + reference + bad-URL rejection + comment thread) and a Playwright test driving the code-review template, the PR link, and a task comment.

`task/series-first-class` promoted task series from a grouping label to a first-class managed domain:

- A series now carries a description and a `draft`/`published`/`closed` lifecycle (transitions publish/unpublish/close/reopen), supports a comment thread, owns a stable `/series/{id}` URL, and lets its creator add, remove, and reorder member tasks. Only the creator can edit a series; a draft series is private to its creator, and a task whose series is not published cannot be reserved or submitted to (enforced in the reserve and submission-eligibility store queries). Tasks gained an `Unpublish` transition (open -> draft) so a task can be pulled back to draft.
- Migration `000018` adds `description`/`state`/`updated_at` to `task_series`, a `series_comments` table, and an index on `tasks(series_id)`. New domain code: `SeriesState` + transitions, `SeriesDescription`, `CommentBody`, `SeriesComment`, `core.SeriesCommentID`, the creator-only service methods, and the store mutations (append at max-position+1, reorder by rewriting positions in a transaction, comment insert/list).
- New HTTP surface (`internal/http/series.go`): create/get/update, the four state transitions, add/remove/reorder member tasks, and the comment thread, plus `POST /api/tasks/{id}/unpublish`. The series detail response is `{series, tasks, comments}`. New MCP tools mirror the agent-relevant subset (`create_series`, `add_task_to_series`, `remove_task_from_series`, publish/unpublish/close/reopen, `add_series_comment`, `list_series_comments`, `unpublish_task`).
- Elm UI: a `/series` list page in the nav with a create form, and the `/series/{id}` detail page (previously orphaned and decoding the wrong shape) now shows the series, its ordered tasks linked to their detail pages, the comment thread, and creator-only controls (rename, the lifecycle buttons, add task by id, remove, reorder up/down) plus an add-comment box; each task detail links back to its series. The generated `TaskSeriesResponse` gained `description`/`state` and a new `SeriesCommentResponse` contract.
- The in-browser demo backend implements the whole series surface (seeded with a published "Orchard intake" series and a comment). Tests: an http_e2e lifecycle test (create -> add task -> publish -> comment, plus a creator-only 403), an MCP e2e test (create_series -> add_task_to_series -> publish -> comment), and a Playwright test that drives the series UI end to end.

`task/lifecycle-parity` (PR1 of a 4-PR roadmap from a full user-journey/gap review) completed the post-and-work-a-task lifecycle for both agents and humans:

- A four-surface gap review (MCP, HTTP API, Elm UI, domain model) found that an agent could not actually post a workable task: `create_task` never set a participation policy (so the task was un-reservable) and there was no MCP tool to open or fund it. Fixed: `create_task` sets `Participation` (new optional `participation_policy` arg, default `open`), `AssigneeScope` (user), and `ReservationTTL` (default); added `open_task` and `fund_task` MCP tools (`internal/mcp/tools.go`, `tool_calls.go`, `server.go`, with adapter methods in `internal/http/agent_mcp.go`). `list_tasks` gained an optional `state` filter, and `get_submission_status` now returns `review_note` so a worker sees reviewer feedback. A new http_e2e test (`TestMCPAgentCreatesFundsOpensWorkableTask`) drives create -> fund -> open and has a different worker submit, proving the task is genuinely workable.
- The human web UI previously hardcoded `response_schema_json` to `{"kind":"freeform"}` and the payload to `none` (`Api.elm`), so a person could not constrain submissions or embed task input. The create-task form now has a response-schema textarea (defaulting to freeform) and an optional task-input JSON textarea, wired through new model fields/messages. The task-detail "Task input" block now also renders the real backend's `json` payload kind (it previously only matched the demo's `inline`). A Playwright test authors a structured schema + payload and asserts the detail surfaces both.
- The MCP install docs named scopes that do not exist (`reservations_write`, `reviews_write`), so a copy-paste produced a 400; corrected to the real scope set and documented `fund_task`/`open_task` in the propose-work loop.

`task/fuzz-flows-contrast` added a fuzz target, fixed WCAG contrast/focus failures, and fixed a demo flow dead-end and example wording:

- Fuzzing: added `internal/http/fuzz_test.go` `FuzzParsePage` — arbitrary `?limit=&offset=` query strings must produce a `core.Page` whose limit stays in [1,200] and whose offset is never negative, so a malformed query can never reach SQL as an out-of-range LIMIT or a negative OFFSET, and it must never panic. Holds (the existing `FuzzHandleRaw` already drives the MCP tool-call argument decoders).
- Contrast review (computed real WCAG 2.1 ratios with the relative-luminance formula; confirmed that the only text on the bare green page background is the page-title h1 — everything else renders inside parchment cards): lightened the arcade theme's page green from `#6b8f3a` to `#b3cf86` so the dark heading ink clears AA (2.21:1 -> 4.79:1) and body ink reaches 9.17:1; added a visible keyboard focus outline (`:focus`/`:focus-visible`), which the theme entirely lacked (WCAG 2.4.7 failure for keyboard users); pinned arcade `::placeholder` to the muted ink (`opacity:1`), up from app.css's ~3:1 50%-of-text default. In the shipped app, moved the low-contrast "revoked" credential label from `text-slate-400` (2.56:1) to `text-slate-600` (7.58:1), and added a base `::placeholder` rule in `web/styles/input.css` pinning placeholders to slate-500 (4.76:1) instead of Tailwind v4's ~2.7:1 preflight default.
- Demo flow + example audit (two subagents — route-by-route flow functionality and example self-containment): the only functional gap was the task-series detail seed (`site/demo/backend.js`) missing `owner_kind` and `created_by`, which made `GET /api/task-series/:id` and the series list fail the strict decoder so the series page hung on "Loading series…"; seeded the contract fields. Reworded the review-extraction task whose "before the first colon" rule collided with each line's "Rating:" prefix (now "between the em dash and the next colon"). Every other Elm client route was verified to have a correctly-shaped fake-backend handler, and all seven seeded tasks remain solvable from their own embedded data.

`task/fuzz-and-polish` added fuzz tests over the untrusted-input parsers, fixed a JSON-encoding bug they found, and applied a demo usability review:

- Added Go native fuzz harnesses: `internal/schema/fuzz_test.go` (`FuzzParseSchemaJSON`, `FuzzParseValueJSON` with an encode/re-parse round-trip, and `FuzzValidate` driving parse + validate + sensitivity index + redact), `internal/auth/fuzz_test.go` (`FuzzVerifyAccessToken` — asserts the token verifier never panics and never accepts a token it did not itself sign, i.e. no HMAC forgery), and `internal/mcp/fuzz_test.go` (`FuzzHandleRaw` — the JSON-RPC transport against the in-memory fake services must never panic and must emit valid JSON or no response). The seed corpora run as ordinary regression tests under `go test`.
- The round-trip fuzz found a real bug: `schema.EncodeValueJSON` quoted strings with `strconv.Quote`, which produces Go literal syntax — byte `0x7f` becomes `\x7f`, which is invalid JSON. This function encodes the redacted submission source that is stored and returned to task owners, so a response containing such a byte round-tripped to invalid JSON. Switched `writeJSONString` to `encoding/json`. The crasher is checked in as a regression seed under `internal/schema/testdata`.
- Verified (did not assume) that the nested-union validator is not a DoS vector: exponential validation cost requires an exponential-size schema, which the HTTP body-size cap already prevents; a body-sized nested/wide union validates in microseconds-to-milliseconds. No artificial complexity budget was added.
- Demo usability review (two subagents — example correctness and flow dead-ends), all fixed in `site/demo/backend.js` (+ one Elm label change rebuilt into the demo bundle): added `POST /api/tasks/:id/refund` (the Refund owner control previously hit the catch-all `ok({})` and failed to decode the escrow shape) and `GET /api/organizations/:id/credits/balance` (the org page balance was stuck on "Loading…"); fixed the date-normalization task whose stated "day-first" rule contradicted its own month-first worked example; reworded a support-ticket example that was equally defensible as `bug` or `billing`; and relabeled the collectibles award flow ("Award a collectible to a task" + helper text + "Award to selected task" button) so the separate task picker and per-collectible button read as one two-step action. `demo.spec.ts` gained tests for the refund and org-balance flows.

`task/demo-on-real-elm` rebuilt the demo to run the real Elm client against an in-browser fake backend:

- Replaced the hand-built static demo with the actual compiled Elm client (`site/demo/elm.js` + `app.css`) served alongside `backend.js`, an `XMLHttpRequest` shim (elm/http uses XHR, not fetch) with a stateful in-memory store that answers every `/api/*` call. Seeded with realistic, specific agentic-work tasks — invoice line-item extraction, support-ticket classification, ledger fraud verification, a weather agent, field-note transcription, photo alt-text — each with real input payloads and strict response schemas. The demo is now the same code as the shipped client and cannot drift.
- Ported the pixel-art arcade theme onto the real client via `arcade.css` (loaded only by the demo): grassy backdrop, parchment dialog-box panels with hard offset shadows, blocky pressable buttons and nav, terminal-green schema blocks, and Press Start 2P / VT323 fonts, overriding the client's Tailwind utilities. The shipped app never loads it.
- Seeded two public tasks owned by other users (and one agent-submitted task) so the discover -> reserve -> submit worker loop and the reverse-MCP story are exercisable, not just the requester side.
- Ran a UI/UX + flow-correctness review of the rebuilt demo. It caught that the shim emitted enum values the real Elm decoders reject (`availability_kind` like `pending_approval`/`submitted`, a `funded` task state, a `task_funding` ledger kind, a `reservations_write` scope, a `cancelled` reservation state) — and because Elm's `Decode.list` is all-or-nothing, those blanked the My-tasks list, ledger, agents screen, and task detail. Fixed every enum to the decoder-valid set and corrected the post/fund/open and review flows. Replaced `demo_static.spec.ts` with `demo.spec.ts` (serves `site/demo` in-process; asserts boot, ledger + My-tasks populate, and detail schema render).
- Known limitation: hard-refresh / deep-link on a demo sub-route 404s on GitHub Pages because the path-routed Elm `Browser.application` builds root-absolute URLs under the `/sharecrop/demo/` base; in-app click navigation works. Recorded in BUGS.md.

`task/collectible-tips-arcade-mcp` added collectible tips, a pixel-art demo theme, MCP docs, and fixtures, with reviews:

- Collectible/inventory tips (real app + demo): added `assets.AllowsTip`, `assets.GiftCollectible` (service), and a `GiftCollectible` store transfer (lock the collectible, enforce ownership + minted state + transfer policy, update `owner_user_id`); the accept handler parses `tip_collectible_id`, settles credits, derives the worker from the payout, and gifts the collectible (a separate per-store transaction sequenced after the settle; idempotent on replay; uniform not-available error). An e2e test covers a successful tip and the policy refusal. The demo review console offers a "Tip a collectible" select that transfers from the reviewer's inventory to the worker on accept.
- Pixel-art "arcade" theme (now the demo default): a farm-RPG palette, chunky hard-outlined dialog-box panels with hard offset shadows, blocky pressable buttons, square pills, terminal-green schema blocks, and pixel fonts (Press Start 2P headings, VT323 body), inspired by Habitica / idle-clicker UIs. Scoped to `body[data-theme="arcade"]`; the other themes stay selectable.
- MCP docs: precise install steps (scoped agent token, `/mcp` client config, an initialize handshake) and the agent work loop as concrete tool calls — poll (`list_tasks`/`get_task`/`get_task_schema`), claim (`reserve_task`), submit (`submit_response`/`get_submission_status`), review (`accept_submission`/`reject_submission`/`request_submission_changes`, approve/decline reservation), and propose (`create_task`).
- Contract fixtures: pinned the wire JSON shape of six uncovered response DTOs (reservation, team, organization, organization member, task capability token, submission-created).
- Reviews + fixes: a security review of the new collectible-tip/rate-limit code surfaced only medium/low items, all addressed (within-org tip denied until org is modeled, idempotent gift, uniform tip error, accept/reject now rate-limited per subject). A UI review of the arcade theme drove fixes: dark-mode primary-button/active-nav contrast, button labels kept whole (whole buttons wrap), schema-block padding, a clear disabled-button state, more legible VT323 eyebrows, scrolling mobile tabs, and wrapping reward rows.
- Deferred: the out-of-process Postgres session/SSE/rate-limiter store (cross-process SSE replay needs `LISTEN/NOTIFY`); queued as DO_NEXT #1.

`task/ratelimit-tipkey-reviews` landed the security follow-ups and ran third-pass reviews:

- Added an in-memory token-bucket rate limiter (`internal/http/rate_limit.go`): per-client-IP on the unauthenticated login/refresh/receipt endpoints and per-agent-subject on MCP requests, returning HTTP 429 when exceeded. Idle buckets are evicted so keys cannot accumulate; client IP uses the direct peer (X-Forwarded-For is not trusted). Unit-tested.
- Gave the two `task_tip` ledger inserts derived idempotency keys (`:tip-debit` / `:tip-credit`), matching payout/refund entries, so the unique constraint would catch a double-tip if the task-lock ordering ever changed. This closes both lower-risk follow-ups that prior security reviews had recorded in BUGS.md.
- A third-pass security review (told what was already fixed and what was in flight) returned no new findings: the authz/RBAC and ledger lenses were clean and the single raw input finding did not survive synthesis's re-verification — the expected outcome at this maturity.
- A round-5 UI/UX review found a real logic bug and a product gap, both fixed: reject left a task open and re-submittable after releasing escrow (so it could never be paid) — reject now closes the task; and agent submissions were indistinguishable from human ones at the review surface — agent-originated reservations/submissions are now marked and rendered as "Sol Rivera · agent" with a "via MCP · scoped token" chip in place of the human track record. Also: a structured schema with no fields now warns and shows an empty state; dashboard cards have parity sub-notes; the Release button is hidden once a result is submitted; a credits/bundle reward warns on a non-positive amount; review panels size to their own content; and the docs MCP tool names match the Agent/API console.

`task/http-dtos-and-reviews` continued the HTTP split, landed the deferred UI minors, and ran second-pass reviews:

- Moved the HTTP request/response DTO struct declarations and the `writableResponse` interface out of `server.go` into `dtos.go` (package `httpserver`); `server.go` is about 906 lines. No behavior change.
- A second-pass security review (given the prior findings and accepted risks) found one new, real, high-severity IDOR: `changeReservationByRequester` checked ownership of the URL-path task, but `ChangeReservationState` matched the reservation by id only via an auto-commit `Exec`, checking the reservation-to-task binding only after the write. An actor owning any task could approve/decline/cancel a reservation belonging to another task (force-granting submission eligibility or denying a legitimate worker). Fixed by binding the `UPDATE` to `task_id` in the same statement, with an e2e test; the service-layer post-check is now defense-in-depth. No other new issues; the prior accepted risks remain in BUGS.md.
- Landed the deferred demo UI minors: unified the two neutral metadata-chip styles into one (toned reward/status chips keep their color); replaced the docs placeholder with a real quickstart (task lifecycle, MCP connect config, scoped/revocable tokens, REST/MCP tool reference); and added a per-persona lifetime track record (settled tasks + acceptance rate) shown as a trust signal on profiles, the reservation queue, and submission rows.
- A round-4 UI/UX review drove further fixes: "Run as Sol agent" now requires the Agent operator persona and an approved reservation for approval-policy tasks (it could previously inject a submission under Sol's identity from any persona and bypass the approval gate); task-list status renders as a colored pill everywhere (a shared `status-pill` base on `toneSpan`); funding failure shows an inline reason at the Fund control; the dashboard open-task count is scoped to the persona's visible tasks; the invoice timeline actor was corrected and reservation-state labels humanized.

`task/http-split-and-security` split `server.go` and applied security + UI/UX reviews:

- Split `internal/http/server.go` (about 2476 lines) into cohesive files in package `httpserver`: `tasks.go` (task + reservation handlers, task request decoders, task response converters), `submissions.go`, `reviews.go`, and `credits.go`. `server.go` retains the router, shared request/response types, and shared helpers (about 1186 lines). No behavior change; Go unit + http_e2e suites, vet, gofmt, copy-paste, and dead-code all pass.
- Ran a multi-agent security review of the Go backend (lenses: authz/RBAC/IDOR, input/injection, secrets/tokens/crypto, ledger/escrow integrity & concurrency, MCP/session/DoS). The authz/RBAC lens found no issues, and the synthesis verified-and-dropped false positives (receipt-token enumeration, tip replay/overdraft — both ruled out by a 256-bit random capability and a `FOR UPDATE` task lock). Applied the real findings: the refresh-token cookie is now `Secure` by default (env opt-out for local HTTP dev); MCP HTTP sessions are capped per agent subject and globally with a 429 on overflow (covered by a new test); and the submission response-value parser caps array items and object fields. Rate limiting and tip-entry idempotency keys were recorded in BUGS.md as accepted lower-risk follow-ups.
- Ran a multi-agent UI/UX/product review of the demo and fixed the majors: every open task is now escrow-backed (the collectible-only audit task carries credits, so "open = funded escrow" holds); Reject defaults to paying 0 while Accept defaults to the full reward, each shown on its button; "Run as Sol agent" is gated by the same claimability rules as the task page (it can no longer reopen a settled task or steal another worker's); Post Task shows only the reward inputs for the chosen reward kind; and the tip hint / worker response nudge copy were clarified. Three moderate minors (neutral-chip style unification, a real Docs page, a worker trust signal) were deferred to DO_NEXT.

`task/elm-split-schema-polish` finished the Main.elm decomposition, deepened the schema designer, and fixed the demo economy:

- Finished decomposing `Main.elm`: extracted `Sharecrop.View` (the view layer plus the `*SuccessLabel` strings) and `Sharecrop.Api` (the HTTP commands, request-body encoders, decoders, result extractors, and the `withSession`/`updateLoggedIn`/`*Command` update glue), and lifted the shared `pageToPath` and visibility helpers into `Sharecrop.Types`. `Main.elm` went from about 2944 lines to about 681 (init, update dispatch, routing, and wiring). No behavior change; 21 Playwright tests and the copy-paste/dead-code checks pass.
- Schema designer: list fields gained min/max item constraints, validated against worker submissions; field names are normalized to lowercase identifier-safe keys shown inline; the designer warns when min items exceed max items; and the decimal field type is validated as a real number.
- Demo economy: seed balances are now net of each requester's escrow (the persona balance is the total; committed escrow is carved out), so available + escrow reconciles and closing seeded tasks no longer mints credits on refund or settle. Review Settle/Tip became numeric inputs pre-filled with real defaults in a two-column grid; a settle is bounded by escrow and the tip by the requester's spendable balance, with over-limit settles refused via a warning rather than clamped; Accept/Reject surface the amount paid; escrow is shown on the worker-facing task page; declining an approval returns the task to a claimable state; and the Agent/API console and Reviews page select only tasks visible/actionable to the persona.
- A multi-agent UI/UX/user-journey/product review of the demo drove the economy/designer fixes. It found a real escrow-accounting blocker (seeded escrow was never debited from balances, so the dashboard double-counted and settles minted credits) and an org-task leak in the Agent/API console (the external operator could read org-only tasks). Both are fixed.

`task/demo-orgs-elm-split-polish` modeled demo organizations, decomposed Main.elm, and applied a specialized review:

- Demo organizations as entities: demo users now belong to an organization (the agent operator is external, with no org), and organization-visibility tasks carry an org id. `canSeeTask` and `canReviewTask` scope organization tasks to members of the owning org instead of a role-string check, so external users no longer see or review org-internal work. The org name surfaces in the topbar, the persona switcher, and the task badge.
- Main.elm decomposition: lifted the `Flags`/`Session`/`Page`/`LoggedInModel`/`TaskDetail`/`Model`/`Msg` type block out of `Main.elm` into a new `Sharecrop.Types` module (no behavior change), shrinking the monolith by ~230 lines and unblocking later view/command splits.
- Ran a specialized multi-agent UI/UX/user-journey/product review of the evolved demo and fixed what it found: a real escrow bug (settle netted escrow/payout against the reviewer clicking Accept rather than the task's requester — now resolves the requester explicitly); the Agent/API console leaked org-internal tasks to the external agent (now lists only the persona's visible tasks); submit validation accepted the empty prefilled template (now rejects empty required fields/arrays); the simulated agent run submitted an empty skeleton (now a realistic schema-filled payload); the reservation queue offered self-approval (now hidden, and the seeded org reservation belongs to a worker); org identity was invisible (now in the topbar + persona switcher); and the task badge row was all-green (metadata pills are now neutral so only status pills carry color). A few minor items (seed-escrow presentation, settle-input default legibility, over-payout clamping) are recorded in DO_NEXT.

`task/economy-orgs-and-polish` bundled a reward economy, schema validation, and team membership:

- Demo reward economy: Fund now checks the requester's available balance and moves the reward credits into a per-task escrow bucket; accept/reject settles from escrow (refunding any unused credits and netting the tip) and cancel refunds it. The dashboard shows "Credits available" and "Held in escrow", and seed escrow is derived from already-funded open tasks so balances stay consistent.
- Demo schema designer + validation: each text field can be marked required and given allowed values (enum); the designer warns on duplicate/empty field names and shows the constraints in the friendly summary. Worker submissions are validated against the schema on submit — missing required keys, wrong types, and out-of-enum values are reported inline and block the submission; a valid one is confirmed and stored.
- Demo polish: the comms/activity feed linkifies persona names to their profiles (matching the cross-linked app), and the unused difficulty field was removed from the seed data.
- Real app standalone-team membership: `POST /api/teams/{id}/members` (org.Service.AddTeamMember + store AddTeamMemberByEmail) lets a user-owned team's owner — or a manager of an owning organization — add members by email; the team page shows an add-member form to the owner and refreshes the roster. RBAC denies others (403). Covered by an e2e test and a Playwright flow.
- Deferred to focused follow-up PRs: modeling organizations as real entities (org id on users + tasks, a DB migration plus RBAC rewrite) and decomposing the `Main.elm` monolith (which needs the interdependent Model/Msg types lifted into shared modules first). Both are large refactors and were left out of this bundle to keep it clean and green; they are recorded in DO_NEXT.

`task/demo-cross-linking` made demo entities real links and acted on a specialized review:

- Converted task and user references into real anchors. Task rows use a stretched `#/tasks/{id}` link over the whole row and user names are `#/users/{id}` anchors, so they support left-click, open-in-new-tab, and right-click; `handleClick` returns early for anchors so the browser handles them. The dead kanban board/card code and CSS were removed (the flat list is the only task list).
- Added a per-row reserve control: Reserve / Request approval when claimable, the requester's context action (Fund / Open / Review queue) now carrying the row's task id, an "Open to submit/run agent" action for available tasks, or a muted non-interactive "Reserved" pill when already claimed.
- Ran a UI/UX + user-journey + product review (multi-agent) and applied its fixes: the per-row Review-queue button now opens that row's task; available rows always show a next action; `nextAction` blocks stealing another worker's active reservation on the detail page; rows top-align so status no longer floats; rows get cursor/hover/underline affordance; the dashboard hero states the reverse-MCP value proposition; Post Task explains what a credit is; stray "mission" copy and the RPG-style S/A/B/C difficulty badge were removed. Full reward escrow accounting and linkifying the activity feed were left for DO_NEXT.
- Replaced the bespoke copy-paste script with jscpd (the standard cross-language detector), pinned to 5.0.11 and tuned to 12 lines / 150 tokens (now also scanning `site/demo`). Added a `.pre-commit-config.yaml` that runs jscpd and `go tool deadcode`, and a CI `pre-commit` job that runs the hooks via the framework.
- Verified every GitHub Action and tool version against its registry/release feed and pinned each to the latest published more than a day old (checkout v7.0.0, setup-go v6.4.0, setup-python v6.2.0, setup-deno v2.0.4, deno v2.8.3, pages actions v6/v5/v5; playwright kept at 1.61.0 since 1.61.1 was under a day old).

`task/demo-stakeholder-review-polish` acted on a multi-stakeholder review (requester, worker, agent operator, org reviewer, first-time visitor, visual/UX, accessibility) of the demo and fixed what it surfaced:

- Payout: Accept now settles the full funded reward by default instead of a canned 18-credit partial that silently underpaid, and the requester is debited the payout plus tip (credits actually move). The canned review note and amounts no longer bleed across tasks.
- Submissions: the response box is stored per task and prefilled from that task's schema; the orchard seed submission was corrected to match its own schema; the agent-run submission is schema-shaped.
- Clarity: the dashboard hero states the product (request agentic tasks from people and their agents); mission/payload/reward-crate/uplink jargon is replaced with task/response/reward across copy, buttons, and timelines; schemas are rendered in plain language ("labels: list of text") next to the raw JSON in the designer, the briefing, and the review console; a worker sees the review note and their prior response when changes are requested.
- Agent/API console: one host and one token placeholder, REST and agent payloads generated from the task schema, worker-only scopes, and a policy-aware MCP workflow. An organization reviewer can no longer review public tasks they did not request.
- Visual: the dark-mode hard offset shadow was replaced with a soft shadow, a visible focus ring was added, status badges are color-coded by lifecycle/availability, the persona control reads as a role switcher, and the reservation-expiry field is hidden for open-submission tasks.

Earlier, \`task/demo-selfcontained-tasks-and-redesign\` made the demo tasks real and redesigned the demo:

- Reworked every demo seed task so it carries its own input material. Each task gained an `inputs` array of blocks (records table, list, text, or code) rendered as an "Input / materials" section in the briefing, and the objectives now reference that on-screen material. This fixes tasks that described inputs ("20 photos by URL", "the linked ledger") that did not exist anywhere — they are now completable from what is shown. The tasks are framed as reverse-MCP agentic requests: humans asking other people and their agents for a structured result.
- Added a receive-schema designer to the Post Task page: a requester writes free-form instructions and either keeps a free-form response or builds a structured one by adding named fields with types (text, whole number, decimal, list of text); the generated response schema is shown live.
- Redesigned the demo visuals: switched the default to the clean "showcase" theme, replaced the hard offset shadow with a soft shadow, increased the corner radius, lightened the heavy panel borders, and styled the new input tables/lists/code and schema designer. Replaced the full task briefing that was wedged onto the dashboard with a compact "continue where you left off" spotlight.

Earlier, `task/team-pages-and-module-split` finished the entity-page work and paid down the HTTP and browser monoliths:

- Added `GET /api/teams/{id}`, returning a team and its member roster, with a new `org.Service.GetTeam` that allows a viewer only when they own the team, belong to it, or (for an organization team) are a member of the owning organization, backed by store `FindTeam` and `ListTeamMembers`. A routed `/teams/{id}` page renders the team name, owner kind, and roster (each member linking to their profile); organization team rows link to it. An e2e test proves the roster is denied to unrelated users.
- Added an assignee-scope selector (user or organization team) to the create-task form, wiring the existing `assignee_scope` field instead of always assigning to a user. A browser test confirms a worker sees the organization-team assignee scope.
- Split the HTTP handler monolith: organization and team handlers moved to `internal/http/organizations.go`, funding and refund handlers to `internal/http/funding.go`, and the team-detail handler is in `internal/http/teams.go` (joining the earlier `users.go`, `series.go`, and `org_credits.go`). No behavior change; shared request and response types and writers stay in `server.go`.
- Split the Elm monolith: the pure enum, label, and format helpers moved from `Main.elm` into a new `Sharecrop.Labels` module, shrinking `Main.elm` by roughly 300 lines with no behavior change.

Earlier, `task/org-followups` added linkable, RBAC-aware pages for every entity a user can reach and finished the organization follow-ups:

- Rewrote the static demo seed tasks to be self-contained (concrete input, deliverable, and acceptance criteria) and de-jargoned the personas and areas, then added hash-routed demo pages including per-user profiles and an always-visible reset control.
- Gave the browser app a URL per entity: routed `/organizations/{id}`; a role-aware `/tasks/{id}` that shows owner controls (open, refund, review) to the task creator and worker controls (reserve, submit) to others, replacing the inline owner detail; `/users/{id}` profiles; `/users/{id}/work`; `/users/{id}/submissions`; `/collectibles/{id}`; and `/series/{id}`.
- Added `GET /api/organizations/{id}/members` (real membership-and-roles query, restricted to active members) with a member list in the organization page; `GET /api/users/{id}` (a user's public tasks via a public-only `CreatorListScope`); `GET /api/users/{id}/work` (public assignments via `AssigneeListScope`); and `GET /api/users/{id}/submissions` (the caller's own submissions only).
- Enforced and tested role-based access control on every new surface: a private task is denied through task detail and is absent from public discovery and from the owner's public profile; submissions are visible only to their submitter (others get 403); the member roster is visible only to members.
- Extended the create-task form with team and organization visibility scopes (a standalone team id is a valid scope) and let the funding form fund a task from organization credits via the existing org-funding endpoint.
- Moved the user-profile, work, and submissions handlers into `internal/http/users.go`.
- Verified all responses are real persisted data with no mocks, placeholders, or stubs in production code.

Earlier, `task/multi-page-routing` gave the browser app real per-section URLs and decomposed the single dashboard panel:

- The HTTP server now serves the single-page-application shell for every non-API route (`index` no longer 404s non-root paths), so deep links and refreshes load the app. Unmatched API paths still return 404.
- The Elm app routes each section to its own URL and page: `/` overview, `/tasks`, `/tasks/new`, `/tasks/{id}`, `/discovery`, `/funding`, `/agents`, `/collectibles`, `/organizations`. The navigation bar uses real `<a href>` links, and the one stacked dashboard was split into focused pages, with per-page data loading.
- The static demo gained an always-visible reset control in the top bar, in addition to the settings-page control.
- Added Playwright coverage that link navigation updates the URL and that deep-linking a page loads it, plus HTTP end-to-end coverage that deep routes serve the shell while unknown API paths 404.

Following the review branch, `task/teams-org-context-collectible-ui` picked up three deferred follow-ups:

- Added standalone (user-owned) teams. Team ownership is a tagged union over organization-owned and user-owned teams, with migration 000017, store create and list methods, `POST` and `GET /api/teams`, the owner exposed on the team contract, and e2e coverage. This is the clean redo of the standalone-teams attempt that was reverted on the review branch.
- Added organization context to the browser: an organization switcher that loads the organization credit balance, organization-scoped task list, and teams; team creation and member provisioning for the active organization; and organization-owned task creation through an owner chooser. Member listing and organization-funded tasks from the browser remain follow-ups because no member listing endpoint exists yet.
- Surfaced multiple collectible rewards in the browser: the reward label is pluralized, the escrowed collectible count shows on tasks even when collectibles are awarded ad hoc, and the task list refreshes after awarding.

A multi-area review of the product, browser UI, HTTP and MCP API, backend domain, data model, tests, and security produced a set of improvements landed on `task/full-review-improvements`:

- Pinned submission redaction behavior: unauthorized receipt-token holders receive redacted data, authorized requesters and organization reviewers receive unredacted data, and non-reviewers receive `403`. The earlier reported "list leak" was not a leak; the redaction model is for unauthorized viewers.
- Authorized submission accept, reject, and request-changes for organization members with the review-submissions permission, resolved inside the review transaction so authorization cannot drift from the write.
- Bounded the Sharecrop schema and value parsers by nesting depth.
- Added a request body size limit to the JSON HTTP endpoints.
- Added refresh-token-family reuse revocation backed by a `family_id` column.
- Added an idle-timeout eviction to the in-memory MCP HTTP session store.
- Kept the transactional acceptance checks in the database transaction rather than moving them to the service layer, and added a concurrency test proving at most one accepted submission per task. Moving the checks out would have reintroduced a time-of-check/time-of-use gap.
- Added multiple collectible rewards per task, transferring all held collectibles on acceptance and returning them on refund.
- Added `limit`/`offset` pagination to list endpoints through a `core.Page` value type.
- Added `state` and `participation_policy` filters to the task list and exposed the active reservation assignee on task list items, fixing an operator-precedence bug in the user-scope list query.
- Added browser task visibility controls (public, private, specific user), task-state guidance, a task-state filter, active-assignee display, an organizations panel, and improved checkbox and label accessibility and contrast through shared `Sharecrop.Ui` helpers.
- Extracted the auth HTTP handlers into `internal/http/auth_handlers.go`.
- Ran a minimalism review of the branch and applied the safe, behavior-preserving simplifications it confirmed.

Standalone (user-owned) teams, deeper organization context in the browser, and a fuller decomposition of `internal/http/server.go` and `web/elm/src/Main.elm` were left as follow-ups in [DO_NEXT.md](./DO_NEXT.md).

The project plan was written in [PLAN.md](./PLAN.md).

The agent workflow was documented in [AGENTS.md](./AGENTS.md).

The Claude pointer file was added in [CLAUDE.md](./CLAUDE.md).

The continuity-file policy was clarified:

- Continuity files were set to update before and after each task.
- [STATUS.md](./STATUS.md) was set to summarize implementation status precisely and factually.
- [WHAT_WE_DID.md](./WHAT_WE_DID.md) was set to remain append-oriented while allowing old or irrelevant parts to be compressed.
- [DO_NEXT.md](./DO_NEXT.md) was set to hold a prioritized queue.
- [BUGS.md](./BUGS.md) was set to include confirmed defects, test gaps, and open risks.
- pull request descriptions were set to be precise and timeless without reproducing code.

The remaining agent-practice questions were resolved:

- [STATUS.md](./STATUS.md) was set to stay short and cover current implemented surface, test status, active task, and blocking issues.
- Continuity updates were set to happen in the same task branch, with the final after-task update near the end of the branch.

The task workflow was updated to use one commit per task, with code, tests, and continuity-file updates included in that task commit.

Testing was set to happen throughout each task and again before finishing the task.

The task workflow was updated again so agents create one git commit at the end of each task by default.

The task workflow was updated so each task uses its own task branch and pull request.

The pull request workflow was constrained to one open pull request at a time.

New task branches were set to start from synced `origin/main` after the previous task pull request is merged.

user interface changes were set to require manual screenshot review when practical.

Playwright user interface tests were set to grow as the user interface matures and workflows stabilize.

The project repository and pull request 1 implementation defaults were recorded:

- GitHub project URL: `https://github.com/e6qu/sharecrop`.
- Canonical SSH remote: `git@github.com:e6qu/sharecrop.git`.
- Go module path: `github.com/e6qu/sharecrop`.
- Local development was set to use Docker Compose for Postgres.
- App config was set to use `DATABASE_URL`.
- The task runner was set to `make`.
- The frontend tool runner was changed to Deno.
- npm was excluded from the frontend toolchain.
- Elm and Tailwind were set to run through Deno-managed tooling or pinned local tooling without npm.
- The first test database strategy was set to one resettable test database per test run.
- The first migration command was set to `sharecrop migrate up`.
- The default app port was set to `18080`.
- The default local Postgres port was set to `15432`.
- Common development ports such as `3000`, `5432`, `8000`, and `8080` were avoided.

The MCP implementation direction was changed:

- No Go MCP library was selected.
- MCP protocol handling was set to be implemented locally from the official MCP specification.

Vitest was considered and not selected.

Deno's built-in test runner was selected for Deno tooling unless a TypeScript/Vite layer is introduced later.

pull request 1 added the project skeleton and build system:

- The Go module `github.com/e6qu/sharecrop` was created.
- The `cmd/sharecrop` binary entry point was added.
- A `net/http` server was added with `/healthz` and an embedded static app shell.
- Config loading was added for HTTP address, `DATABASE_URL`, and migrations directory.
- PostgreSQL access was isolated in `internal/db` with `pgx`.
- A plain SQL migration runner was added with `sharecrop migrate up`.
- An initial migration file was added.
- Docker Compose configuration was added for Postgres on local port `15432`.
- An Elm app shell was added.
- Tailwind was wired through Deno-managed tooling.
- Deno smoke tests were added.
- Go HTTP unit tests were added.
- HTTP end-to-end smoke tests were added behind the `http_e2e` build tag.
- Playwright user interface smoke tests were added.
- A manual screenshot helper was added.
- `make` commands were added for build, test, serve, migration, frontend, and user interface end-to-end.
- Generated local artifacts were excluded through `.gitignore`.

pull request 1 verification was performed:

- `GOCACHE=$PWD/.cache/go-build go test ./...` passed.
- `deno task test` passed.
- `deno task frontend:build` passed.
- `make build` passed.
- `deno task e2e:ui` passed earlier in the task.
- Manual screenshot review showed the app shell rendering the Sharecrop heading and skeleton text.

pull request 1 verification gaps were recorded:

- Docker Compose Postgres startup was not verified because the environment rejected Docker approval.
- `sharecrop migrate up` against live Postgres was not verified for the same reason.
- Final rerun of `deno task e2e:ui` was not performed because local-network/browser permissions had already been exhausted in this environment after an earlier successful run.
- `make build` with both `GOCACHE` and `GOMODCACHE` isolated inside the workspace could not fetch `pgx` because network access was restricted. The build had passed earlier with the existing module cache.

pull request 2 added core domain foundations and continuous integration quality gates:

- Core domain errors were added.
- Strong ID wrappers were added for users, tasks, and organizations.
- UUIDv7 generation and parsing were isolated behind `internal/core/id`.
- Lifecycle state parsing was added.
- Visibility scope variants and parsing were added.
- Per-type result variants were used instead of generic result types.
- continuous integration was added for formatting, TypeScript checks, policy checks, copy-paste detection, dead-code detection, Deno linting, Go vet, unit tests, frontend build, binary build, migrations, HTTP end-to-end, and user interface end-to-end.
- continuous integration was limited to pull requests targeting `main`, without direct `main` push runs or bare branch push runs.
- The Elm build tool was changed to require explicit `ELM_BIN`.
- Config loading was changed to require explicit environment variables instead of fallback values.
- Docker Compose was fixed for PostgreSQL 18 by mounting the volume at `/var/lib/postgresql`.

pull request 2 verification was performed:

- `make check-format` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `make check-ts` passed.
- `make lint` passed.
- `GOCACHE=$PWD/.cache/go-build make vet` passed.
- `GOCACHE=$PWD/.cache/go-build make test` passed.
- `GOCACHE=$PWD/.cache/go-build make check-dead-code` passed.
- `ELM_BIN=/opt/homebrew/bin/elm make frontend` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod make build` passed.
- `make test-http` passed with local listener permission.
- `deno task e2e:ui` passed with local browser permission.
- Manual screenshot review showed the app shell rendering without visible layout issues after pull request 2 changes.
- `docker compose up -d postgres` passed.
- `make migrate-up` passed against local Postgres.
- `docker compose down` passed.

pull request 2 verification gaps were recorded:

- Aggregate `make ci` was not run locally because the environment approval request timed out twice.

Pull request 3 added authentication, sessions, and guest identity:

- Guest and refresh-token identifiers were added to the core identifier set.
- Authentication value types were added for email addresses, password secrets, access-token secrets, refresh tokens, subjects, and session results.
- Password hashing was implemented with standard-library PBKDF2 and SHA-256 behind `internal/auth`.
- JSON Web Token access-token signing was implemented with standard-library HMAC SHA-256 behind `internal/auth`.
- Opaque refresh-token generation and hashing were added.
- The authentication service added registered user creation, login, guest subject creation, refresh-token rotation, and refresh-token reuse rejection.
- PostgreSQL tables were added for users, guest subjects, password credentials, and refresh tokens.
- The PostgreSQL authentication repository was added under `internal/db`.
- HTTP endpoints were added for registration, login, refresh, and guest session creation.
- Refresh tokens were returned as HttpOnly cookies.
- Config parsing was split into pure `ParseConfig` and the environment-reading `LoadConfig` shell.
- `SHARECROP_ACCESS_TOKEN_SECRET` was added as an explicit required environment variable.
- Dead-code detection was changed from `go run ...@latest` to a declared Go tool dependency invoked through `go tool deadcode`.

Pull request 3 verification was performed:

- `make check-format` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `make check-ts` passed.
- `make lint` passed.
- `GOCACHE=$PWD/.cache/go-build make vet` passed.
- `GOCACHE=$PWD/.cache/go-build make test` passed.
- `GOCACHE=$PWD/.cache/go-build make check-dead-code` passed.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build go test -run '^$' -tags http_e2e ./tests/http_e2e` passed.
- `ELM_BIN=/opt/homebrew/bin/elm make frontend` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed with a non-fatal global module stat-cache warning from the sandbox.

Pull request 3 verification gaps were recorded:

- Runtime HTTP end-to-end tests were not run locally because the environment rejected the required local listener and PostgreSQL approval after the usage limit was reached.
- Playwright browser tests were not rerun locally because the user interface was not changed and the environment could not grant further browser/listener approval.

Pull request 4 added organizations, teams, and provisioning:

- Team and organization membership identifiers were added to the core identifier set.
- Organization names, team names, organization membership statuses, organization roles, and organization permissions were added under `internal/org`.
- Organization public-publisher permission was modeled separately from reviewer and billing roles.
- Organization service methods were added for organization creation, organization listing, member provisioning, member deactivation, team creation, and team listing.
- Access-token verification was added to the authentication boundary.
- PostgreSQL tables were added for organizations, organization memberships, organization membership roles, teams, and team members.
- PostgreSQL organization repository code was added under `internal/db`.
- HTTP endpoints were added for organization creation, organization listing, organization member provisioning, organization member deactivation, organization team creation, and organization team listing.
- Organization HTTP endpoints required verified bearer access tokens and service-level permission checks.

Pull request 4 test strategy was evaluated:

- Domain constructors, enums, permissions, and service permission checks were covered by unit tests.
- HTTP handler mapping was covered with unit tests using typed test doubles.
- API and PostgreSQL behavior were covered by HTTP end-to-end tests using the real migration runner, repository, service, access tokens, and PostgreSQL.
- Browser user interface tests were not expanded because this task did not change browser user interface source.

Pull request 4 verification was performed:

- `make check-format` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `make check-ts` passed.
- `make lint` passed.
- `GOCACHE=$PWD/.cache/go-build make vet` passed.
- `GOCACHE=$PWD/.cache/go-build make test` passed.
- `GOCACHE=$PWD/.cache/go-build make check-dead-code` passed.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build go test -run '^$' -tags http_e2e ./tests/http_e2e` passed.
- `docker compose up -d postgres` passed.
- `SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make migrate-up` passed.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make test-http` passed.
- `docker compose down` passed.
- `ELM_BIN=/opt/homebrew/bin/elm make frontend` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed with a non-fatal global module stat-cache warning from the sandbox.

Pull request 5 added the Go-to-Elm contract generator:

- Go-owned contract definitions were added under `internal/contracts`.
- The contract model covered aliases, product types, enums, named type references, string type references, and list type references.
- Elm generation was added for modules, type aliases, enums, decoders, and encoders.
- Generated Elm modules were added under `web/elm/src/Sharecrop/Generated/`.
- First generated contracts covered auth responses, error responses, identifiers, organization responses, organization member responses, team responses, membership statuses, subject kinds, and organization roles.
- The `sharecrop generate elm-contracts` command was added.
- The Makefile gained `contracts` and `check-contracts` targets.
- Frontend builds were changed to generate contracts before compiling Elm.
- The handwritten Elm app consumed the generated `Sharecrop.Generated.Auth.SubjectKind` type directly.

Pull request 5 test strategy was evaluated:

- Generator unit tests checked generated auth output, deterministic output, and absence of weak generated Elm shapes such as `Bool` and `Dict`.
- `check-contracts` verified generated files were current and deterministic.
- Elm compilation verified generated modules worked with Elm 0.19.1.
- The handwritten Elm app imported a generated module to ensure generated contracts were usable from normal Elm code.
- Existing HTTP end-to-end tests remained the API behavior checks for this slice.
- Playwright and manual screenshot checks were run because Elm source changed.

Pull request 5 verification was performed:

- `make check-format` passed.
- `make check-contracts` passed with `GOCACHE=$PWD/.cache/go-build`.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `make check-ts` passed.
- `make lint` passed.
- `GOCACHE=$PWD/.cache/go-build make vet` passed.
- `GOCACHE=$PWD/.cache/go-build make test` passed.
- `GOCACHE=$PWD/.cache/go-build make check-dead-code` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make frontend` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed with a non-fatal global module stat-cache warning from the sandbox.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build go test -run '^$' -tags http_e2e ./tests/http_e2e` passed.
- `docker compose up -d postgres` passed.
- `SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make migrate-up` passed.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make test-http` passed.
- `ELM_BIN=/opt/homebrew/bin/elm SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build deno task e2e:ui` passed.
- Manual screenshot review passed for `/tmp/sharecrop-pr5-shell.png`.
- `docker compose down` passed.

Pull request 6 added the Sharecrop schema parser and validator:

- Local schema domain types were added under `internal/schema`.
- Schema kinds were added for object, array, string, integer, decimal string, enum, literal, union, and freeform schemas.
- Field presence was modeled as explicit `required` and `may_omit` values.
- Sensitivity categories, retention policies, and redaction policies were modeled as typed values.
- Schema JSON parsing converted boundary data into Sharecrop-owned schema types.
- Response payload JSON parsing converted payloads into Sharecrop-owned value types without using generic maps.
- Schema validation produced typed validation errors with field paths.

- Sensitive-field indexing and redaction were added for typed response values.

Pull request 7 added task series, tasks, visibility, and capability tokens:

- Task-series, task, task-visibility-scope, and task-capability-token migrations were added.
- Task-series and task-capability-token identifiers were added to the core identifier set.
- Task owner, task state, task series placement, task visibility, task payload, and task capability-token lifecycle types were added under `internal/task`.
- Opaque task capability-token generation and hashing were added without encoding task identifiers into token strings.
- The task service added task creation, opening, cancellation, listing, and capability-token creation.
- Organization-owned tasks required organization task-creation permission.
- Public organization tasks required organization public-publisher permission.
- Default task visibility mapped user-owned tasks to user visibility and organization-owned tasks to organization visibility.
- PostgreSQL task repository code was added under `internal/db`.
- HTTP task endpoints were added for creation, listing, opening, cancellation, and capability-token creation.
- Task response schemas were parsed with the local Sharecrop schema parser during task creation.
- Generated Elm contracts were extended with task identifiers, task enums, task list items, task lists, and task capability-token responses.

Pull request 7 test strategy was evaluated:

- Unit tests covered task state transitions, capability-token opacity, capability-token parsers, identifier round trips, and organization public-publishing permission behavior.
- HTTP unit tests covered task request parsing and default user visibility.
- HTTP end-to-end tests covered task creation, user-scoped listing, task opening, task cancellation, capability-token creation, and organization public-publishing permissions against PostgreSQL.
- Playwright and manual screenshot checks were run because generated Elm source changed.

Pull request 7 verification was performed:

- `GOCACHE=$PWD/.cache/go-build go test ./...` passed.
- `make check-format` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `deno task check:ts` passed.
- `deno task lint` passed.
- `GOCACHE=$PWD/.cache/go-build make vet` passed.
- `GOCACHE=$PWD/.cache/go-build make check-dead-code` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make frontend` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed with a non-fatal global module stat-cache warning from the sandbox.
- `docker compose up -d postgres` passed.
- `SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build go run ./cmd/sharecrop migrate up` passed.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build go test -tags http_e2e ./tests/http_e2e` passed.
- `ELM_BIN=/opt/homebrew/bin/elm SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build deno task e2e:ui` passed.
- Manual screenshot review passed for `/tmp/sharecrop-pr7-shell.png`.

Pull request 8 added submissions, anonymous access, and sensitive-field handling:

- Submission and submission receipt-token identifiers were added to the core identifier set.
- Submission tables were added for submissions, receipt tokens, validation errors, and sensitive-field index rows.
- Submission domain types were added for authenticated submitters, anonymous wallet submitters, submission states, validation outcomes, response JSON, wallet addresses, and receipt tokens.
- Opaque receipt-token generation and hashing were added.
- The submission service added authenticated submission, anonymous public submission, receipt lookup, and requester submission listing.
- Anonymous submissions were limited to public tasks.
- Anonymous submitters stored payout wallet addresses without linking them to user identifiers.
- Submitted response JSON was parsed and validated against the task response schema.
- Schema-invalid submissions were recorded with `invalid` state and validation-error rows.
- Sensitive submitted fields were indexed from the task response schema.
- Receipt lookup returned redacted response JSON for sensitive fields.
- PostgreSQL submission repository code was added under `internal/db`.
- HTTP endpoints were added for authenticated task submissions, anonymous public task submissions, receipt status, and requester submission listing.
- Generated Elm contracts were extended with submission identifiers, submission states, submitter kinds, validation-error responses, submission responses, submission lists, and submission-created responses.

Pull request 8 test strategy was evaluated:

- Unit tests covered anonymous/public submission permission, receipt-token creation, invalid submission recording, sensitive redaction for receipt lookup, and identifier round trips.
- HTTP unit tests covered authenticated submission request handling and receipt-token response shape.
- HTTP end-to-end tests were added for anonymous public submission, receipt redaction, invalid response recording, and requester submission listing.
- Browser user interface tests were not expanded because pull request 8 did not add visible submission screens.

Pull request 8 verification was performed:

- `GOCACHE=$PWD/.cache/go-build go test ./...` passed.
- `make check-format` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `deno task check:ts` passed.
- `deno task lint` passed.
- `GOCACHE=$PWD/.cache/go-build go vet ./...` passed.
- `GOCACHE=$PWD/.cache/go-build make check-dead-code` passed.
- `go run ./cmd/sharecrop generate elm-contracts` regenerated identical generated Elm contracts.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make frontend` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed.
- `docker compose up -d postgres` passed.
- `DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build go run ./cmd/sharecrop migrate up` applied the submission migration.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make test-http` passed, including anonymous submission, receipt redaction, invalid-response recording, and requester listing tests.
- `ELM_BIN=/opt/homebrew/bin/elm ... make e2e-ui` passed the app-shell Playwright smoke test.
- `deno task test` passed.
- `docker compose down` passed.
- Sensitive-field indexing located sensitive values in submitted payloads.
- Redaction replaced or removed sensitive fields according to schema policy.

Pull request 6 test strategy was evaluated:

- Parser tests covered typed parsing, unsupported schema kinds, freeform mode, union schemas, and enum rejection.
- Validator tests covered required field failures and valid object payloads.
- Sensitive-data tests covered sensitive path indexing, replacement redaction, and remove redaction.
- Existing HTTP end-to-end tests remained the API behavior checks for this slice because task and submission endpoints are not implemented yet.
- Browser user interface tests were not expanded because this task did not change browser user interface source.

Pull request 6 verification was performed:

- `make check-format` passed.
- `make check-contracts` passed with `GOCACHE=$PWD/.cache/go-build`.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `make check-ts` passed.
- `make lint` passed.
- `GOCACHE=$PWD/.cache/go-build make vet` passed.
- `GOCACHE=$PWD/.cache/go-build make test` passed.
- `GOCACHE=$PWD/.cache/go-build make check-dead-code` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make frontend` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed with a non-fatal global module stat-cache warning from the sandbox.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build go test -run '^$' -tags http_e2e ./tests/http_e2e` passed.
- `docker compose up -d postgres` passed.
- `SHARECROP_HTTP_ADDR=:18080 SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make migrate-up` passed.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make test-http` passed.
- `docker compose down` passed.

Pull request 9 added credits, ledger, escrow, first accepted submission, the credit ledger user interface, and an expanded test pyramid:

- Credit account and ledger entry identifiers were added to the core identifier set.
- Credit account, append-only ledger entry, and task escrow tables were added, along with a single-accepted partial unique index on submissions.
- Credit domain types were added under `internal/ledger` for positive credit amounts, signed ledger amounts, derived balances, ledger entry kinds, escrow states, idempotency keys, ledger entries, and task escrows.
- Balance derivation summed the signed amounts of an account's ledger entries.
- The ledger service added task funding, submission acceptance with payout, task refund, balance lookup, and ledger listing.
- The PostgreSQL ledger repository performed funding, acceptance, and refund inside row-locked transactions.
- Each new registered user received a credit account and a `signup_grant` of 100 credits inside the user-creation transaction.
- Task funding escrowed credits from the funder's account and required sufficient balance and a draft, owner-held task.
- Submission acceptance was transactional, closed the task, enforced a single accepted submission per task, and paid the accepted authenticated worker from the escrow.
- Task refund cancelled a funded task and returned escrowed credits to the funder.
- Fund, accept, and refund used idempotency keys so retries did not double-charge or double-pay.
- HTTP endpoints were added for credit balance, ledger listing, task funding, submission acceptance, and task refund.
- The contract generator gained an integer reference type, and a single-field record decoder was changed from `Decode.mapN` to `Decode.map`.
- Generated Elm contracts were extended with credit account and ledger entry identifiers, ledger entry kinds, escrow states, balance responses, ledger entry responses, ledger responses, and task escrow responses.
- The Elm app was changed from an app shell into an interactive client with register and login, a credit balance and ledger view, and a task funding form backed by the API.
- A Postgres-backed integration test tier was added under the `integration` build tag with a `make test-integration` target.
- continuous integration was split into parallel static, unit, build, integration, HTTP end-to-end, and Playwright jobs.

Pull request 9 test strategy was evaluated:

- Unit tests covered credit amount validation, signed amount parsing, ledger entry kind and escrow state parsing, idempotency key validation, balance derivation, and ledger service delegation.
- HTTP unit tests covered the credit balance endpoint and task funding request handling with typed test doubles.
- Integration tests covered the signup grant, funding, single-escrow enforcement, acceptance payout, idempotent acceptance, and refund against PostgreSQL.
- HTTP end-to-end tests covered the signup grant, the fund-open-submit-accept-payout flow, idempotent acceptance, single-accepted enforcement, refund, insufficient-credit funding, and no-reward acceptance.
- Playwright tests covered registering through the browser to see the signup grant balance and ledger entry, and funding a task through the browser.
- Manual screenshot review covered the logged-out shell and the logged-in credit dashboard.

Pull request 9 verification was performed:

- `GOCACHE=$PWD/.cache/go-build go test ./...` passed.
- `make check-format` passed.
- `make check-contracts` passed.
- `make check-policy` passed.
- `make check-ts` passed.
- `make check-copy-paste` passed.
- `GOCACHE=$PWD/.cache/go-build make check-dead-code` passed.
- `make lint` passed.
- `GOCACHE=$PWD/.cache/go-build make vet` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed.
- `docker compose up -d postgres` passed.
- `DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make migrate-up` applied the credits and ledger migration.
- `DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make test-integration` passed and was rerun to confirm idempotency safety against a persistent database.
- `SHARECROP_ACCESS_TOKEN_SECRET=... DATABASE_URL=... SHARECROP_MIGRATIONS_DIR=$PWD/migrations GOCACHE=$PWD/.cache/go-build make test-http` passed.
- `ELM_BIN=/opt/homebrew/bin/elm ... make e2e-ui` passed the app-shell, signup-grant, and browser-funding Playwright tests.
- Manual screenshot review passed for `/tmp/sharecrop-pr9-shell.png` and `/tmp/sharecrop-pr9-dashboard.png`.
- `docker compose down` was run after verification.

Pull request 10 added MCP, agent credentials, agent setup, and task discovery surfaces:

- Agent credential identifiers were added to the core identifier set.
- Agent credential and agent credential scope tables were added.
- Agent credential domain types were added under `internal/agent` for scopes, lifecycle state, labels, opaque secrets and hashes, scope sets, and scope checks.
- The agent credential service added scoped creation, verification, listing, and revocation, with PostgreSQL repository code that stored scopes in a child table.
- A local MCP JSON-RPC server was added under `internal/mcp`, implemented from the MCP specification without a Go MCP library, handling `initialize`, `ping`, `tools/list`, and `tools/call`.
- MCP tools were added for `sharecrop.list_tasks`, `sharecrop.get_task`, `sharecrop.get_task_schema`, `sharecrop.create_task`, `sharecrop.submit_response`, `sharecrop.get_submission_status`, `sharecrop.list_task_submissions`, and `sharecrop.accept_submission`, each gated by an agent scope and adapted over the existing task, submission, and ledger services.
- A task service `Get` method and a `GET /api/tasks/{task_id}` endpoint were added with a task view-permission check covering creators, public tasks, user visibility, and organization visibility.
- HTTP endpoints were added for agent credential creation, listing, and revocation, and a `POST /mcp` endpoint authenticated by agent credentials with per-tool scope enforcement.
- Generated Elm contracts were extended with the agent credential identifier, agent scopes, agent credential state, and agent credential responses.
- The browser app gained a task list panel with REST and MCP curl examples per task, and an agent setup panel for creating, viewing, and revoking scoped credentials with generated MCP client configuration and a one-time token.
- The Elm app was changed to accept an `origin` flag so the generated MCP configuration and curl examples use the live server origin.

Pull request 10 test strategy was evaluated:

- Unit tests covered agent scope parsing, scope-set de-duplication and checks, opaque secret round trips, label validation, and agent service create/verify/revoke.
- MCP unit tests covered initialize, tools/list, unknown methods, scope enforcement, tool dispatch, unknown tools, and domain rejections surfaced as tool errors.
- HTTP unit tests covered agent credential creation, unknown-scope rejection, and the MCP endpoint requiring an agent credential.
- Integration tests covered agent credential create, verify, list, and revoke against PostgreSQL.
- HTTP end-to-end tests covered the agent discover-submit-status-list-accept flow over MCP with a credit payout, MCP scope enforcement, revoked-credential rejection, and the single-task REST endpoint.
- Playwright tests covered creating an agent credential through the browser to see the token and MCP configuration, and listing the user's tasks with agent curl examples.
- Manual screenshot review covered the agent setup panel.

Pull request 10 verification was performed:

- `GOCACHE=$PWD/.cache/go-build go test ./...` passed.
- `make check-format`, `make check-contracts`, `make check-policy`, `make check-ts`, `make check-copy-paste`, `make check-dead-code`, `make lint`, and `make vet` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed.
- `docker compose up -d postgres` passed and the agent credentials migration applied.
- `make test-integration` passed and was idempotency-safe across reruns.
- `make test-http` passed, including the MCP and agent flows.
- `ELM_BIN=/opt/homebrew/bin/elm ... make e2e-ui` passed the app-shell, ledger, and agent Playwright tests.
- Manual screenshot review passed for `/tmp/sharecrop-pr10-agents.png`.

Pull request 11 added deferred backend gaps, MCP transports, and user interface polish with new screens:

- UUIDv7 generation was verified in code for version 7 and time ordering, and a parser-rejection test was added.
- HTTP contract fixture tests were added to pin the wire JSON shape of representative API responses.
- A task-series read API was added: `task` service `ListSeries` and `GetSeries` with a series view-permission check, PostgreSQL `ListSeries` and `FindSeries` repository code, `GET /api/task-series` and `GET /api/task-series/{id}` endpoints, and generated Elm task-series contracts.
- The MCP server gained `sharecrop.list_task_series` and `sharecrop.get_task_series` tools.
- The MCP server gained JSON-RPC batch handling through a shared `HandleRaw` entry point used by both transports.
- The MCP HTTP endpoint was hardened toward Streamable HTTP: a `Mcp-Session-Id` header on initialize, `Origin` validation for DNS-rebinding protection, a `405` response on `GET`, and a request body size limit.
- A stdio MCP transport was added through a `sharecrop mcp` command that authenticates with `SHARECROP_AGENT_TOKEN`, verifies the agent credential, and drives the same MCP server over newline-delimited JSON-RPC on stdin and stdout. This is the transport local agent clients launch.
- The transport surface was chosen from what Claude Code and Codex both implement as MCP clients: stdio and Streamable HTTP with a static bearer token. HTTP/1.1 and HTTP/2 are negotiated by the web server, and HTTP/3 and raw sockets were intentionally not added.
- A reusable shadcn-inspired Elm component module was added under `Sharecrop.Ui` with cards, buttons, inputs, badges, code blocks, and labels, and the app was refactored to use it.
- Browser page navigation was added with a public task discovery screen and a task detail screen that submits responses and lets task owners review and accept submissions.

Pull request 11 test strategy was evaluated:

- Unit tests covered UUIDv7 version and ordering, contract wire shapes, the series view-permission check, the MCP series tools, JSON-RPC batch and notification handling, and the stdio loop.
- Integration tests covered the task-series store list and find against PostgreSQL.
- HTTP end-to-end tests covered the series REST endpoints, the MCP series tools, MCP batch requests, the `GET` `405`, and the `Mcp-Session-Id` header.
- The stdio command was smoke-tested end to end against PostgreSQL by piping `initialize` and `tools/list` to `sharecrop mcp`.
- Playwright tests covered discovering a public task, submitting through the browser, and an owner reviewing and accepting the submission, while preserving the existing dashboard and agent-setup tests.
- Manual screenshot review covered the task detail screen.

Pull request 11 verification was performed:

- `make check-format`, `make check-contracts`, `make check-policy`, `make check-ts`, `make check-copy-paste`, `make check-dead-code`, `make lint`, and `make vet` passed.
- `GOCACHE=$PWD/.cache/go-build make test` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed.
- `docker compose up -d postgres` passed and the existing migrations applied.
- `make test-integration` passed and remained idempotency-safe across reruns.
- `make test-http` passed, including the series and MCP transport tests.
- `ELM_BIN=/opt/homebrew/bin/elm ... make e2e-ui` passed the app-shell, ledger, agent, and screens Playwright tests.
- `SHARECROP_AGENT_TOKEN=... go run ./cmd/sharecrop mcp` returned the initialize result and tool list over stdio.
- Manual screenshot review passed for `/tmp/sharecrop-pr11-detail.png`.

Pull request 12 narrowed the asset economy to platform-only rewards, removed anonymous workers, and added organization credit accounts and platform collectibles:

- Anonymous wallet-based submission was removed: the anonymous submitter domain type, wallet address value, public submission route and handler, anonymous columns, and anonymous tests were deleted, and a migration dropped the `submitter_kind` and `wallet_address` columns and made submissions registered-users-only.
- Organization credit accounts were added: a migration extended `credit_accounts` to support organization owners, organizations received a credit account and grant inside the organization-creation transaction, organization-owned tasks can be funded from the organization account behind the manage-billing permission, and an organization credit balance endpoint was added.
- The ledger funding logic for user and organization funding was unified behind a shared escrow-completion helper.
- A platform collectible model was added under `internal/assets` with collectible kinds, lifecycle states, names, and transfer-policy variants, plus a reward-payout policy check.
- The collectible service and PostgreSQL repository added minting, listing, collectible task reward escrow, and refund.
- The submission-acceptance flow was generalized so accepting a submission for a collectible-reward task transfers the collectible to the worker, reported as a collectible payout.
- HTTP endpoints were added for minting and listing collectibles, funding and refunding collectible rewards, and the organization credit balance.
- Generated Elm contracts were extended with the collectible identifier, collectible kinds, states, transfer policies, and collectible responses.
- The browser app gained a collectibles panel for minting, viewing holdings, and awarding a collectible to a task, and the submission request dropped the wallet address field.

Pull request 12 scope decisions were recorded:

- Rewards were kept entirely on-platform: Sharecrop credits are the platform token and platform collectibles are the non-fungible reward. User-issued tokens, organization-issued tokens, crypto rewards, and external wallets were intentionally excluded.
- Anonymous workers were deferred until the anonymous identity and payout model is decided.

Pull request 12 test strategy was evaluated:

- Unit tests covered collectible kind, state, and transfer-policy parsing, the reward-payout policy check, and collectible minting.
- HTTP unit tests covered the collectible response wire shapes through the existing handler doubles.
- Integration tests continued to cover the ledger and series stores against PostgreSQL.
- HTTP end-to-end tests covered organization credit account funding and balance, the collectible award-on-accept flow, collectible reward refund, the issuer-controlled policy denial, and the rewritten registered-user submission tests.
- Playwright tests covered minting a collectible and awarding it to a task through the browser, while preserving the existing dashboard, agent, discovery, and acceptance tests.
- Manual screenshot review covered the collectibles panel.

Pull request 12 verification was performed:

- `make check-format`, `make check-contracts`, `make check-policy`, `make check-ts`, `make check-copy-paste`, `make check-dead-code`, `make lint`, and `make vet` passed.
- `GOCACHE=$PWD/.cache/go-build make test` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make build` passed.
- `docker compose up -d postgres` passed and the credit account and collectible migrations applied on a fresh database.
- `make test-integration` and `make test-http` passed.
- `ELM_BIN=/opt/homebrew/bin/elm ... make e2e-ui` passed the app-shell, ledger, agent, screens, and collectible Playwright tests.
- Manual screenshot review passed for `/tmp/sharecrop-pr12-collectibles.png`.

Pull request 13 fixed reward, lifecycle, requester, contract, HTTP, MCP, and session issues found during review:

- Tasks gained an explicit reward specification for no-reward and credit-reward tasks, with response fields for reward kind and credit amount.
- Credit escrow funding now requires the task to declare a matching credit reward, and credit-reward tasks cannot be opened until matching escrow is held.
- Submission acceptance stores the accept idempotency key, same-key retries return the accepted outcome without paying twice, and different-key re-accepts are rejected.
- Submission creation now requires an open visible task, and requester submission listing allows the task creator or organization reviewers.
- Domain errors now distinguish missing resources, permission denials, conflicts, and invalid states so HTTP handlers can return `404`, `403`, and `409` where applicable.
- Organization and collectible funding endpoints use the shared domain HTTP status mapping.
- Generated Elm product decoders now support records larger than eight fields, and generated task and ledger contracts include task detail and accept-submission response shapes.
- MCP task creation requires reward arguments, tool output includes reward details, raw JSON-RPC handling responds to `id:null`, client response objects are ignored as server input, and `/mcp` validates `Accept` and `MCP-Protocol-Version`.
- Browser routing moved to `Browser.application` with dashboard, discovery, and task detail routes.
- Browser auth restores sessions through refresh cookies and clears the refresh cookie through `POST /api/auth/logout`.
- The dashboard gained task creation with optional credit rewards, funding prefill for newly created credit-reward tasks, open and refund controls, task detail viewing, submission detail review, and accept controls.
- Browser task rows and detail screens show reward labels.

Pull request 13 test strategy was evaluated:

- Unit tests covered reward parsing and service-level submission visibility/open-state behavior, organization reviewer submission listing, MCP raw handling, and logout cookie clearing.
- Integration tests covered credit reward funding, acceptance, idempotent re-accept, and refund persistence against PostgreSQL.
- HTTP end-to-end tests covered credit-reward funding and payout, no-reward acceptance, organization credit funding, collectible funding status mapping, task lifecycle status mapping, MCP reward-aware flows, and submission visibility/open-state behavior.
- Playwright tests covered browser task funding with declared rewards, task discovery, worker submission, owner review and acceptance, session switching through logout, and the existing dashboard, agent, ledger, and collectible workflows.
- Manual screenshot review covered the updated dashboard.

Pull request 13 verification was performed:

- `docker compose up -d postgres` passed.
- `GOCACHE=$PWD/.cache/go-build go test ./internal/org ./internal/task ./internal/submission ./internal/http ./internal/db ./internal/mcp` passed.
- `DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations GOCACHE=$PWD/.cache/go-build make test-integration` passed.
- `DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 GOCACHE=$PWD/.cache/go-build make test-http` passed.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=$PWD/.cache/go-build make frontend` passed.
- `ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 GOCACHE=$PWD/.cache/go-build make e2e-ui` passed.
- Manual screenshot review passed for `/tmp/sharecrop-dashboard.png`.
- `ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 GOCACHE=$PWD/.cache/go-build make ci` passed before the logout endpoint was added.
- After the logout endpoint was added, the equivalent final local check set passed in target groups: `make check-format check-contracts check-policy check-ts check-copy-paste check-dead-code lint vet test frontend build`, `make test-integration`, `make test-http`, and `make e2e-ui`.

The post-PR13 workflow plan was updated:

- Task rewards were planned as bundles that can contain credits, collectibles, both, or neither.
- Reservation-required and requester-approval-required task flows were planned.
- The default reservation expiry was set to 48 hours, with automatic release on expiry.
- Tasks were planned to allow exactly one active assignee: one user or one team.
- First implementation of team assignment was scoped to users and same-organization teams; public teams remain deferred until public-team modeling exists.
- Reserved tasks were planned to disappear from default discovery and reappear only when the viewer selects include-reserved, except for the active assignee and requester.
- Request changes was planned to require requester notes and keep the same assignee exclusive.
- Review outcomes were planned for accept, request changes, reject with optional partial reward, reject without reward, optional task-local implementor ban, and optional tips from requester balance or inventory.
- MCP work was planned to add workflow tools and full Streamable HTTP SSE with `GET /mcp`, `DELETE /mcp`, session enforcement, event IDs, and replay where practical.
- The next implementation sequence was recorded as PR 14 reservation/approval foundations, PR 15 requester ergonomics and task-page instructions, PR 16 review outcomes, PR 17 reward bundles, and PR 18 MCP workflow tools plus full SSE.

The reservation, approval, and discovery availability foundation branch added backend task assignment support:

- A task reservation identifier was added to the core identifier set.
- Task domain models gained participation policies, assignee scopes, reservation expiry, assignee variants, reservation lifecycle states, availability kinds, and viewer actions.
- Task creation commands and HTTP task creation requests gained participation policy, assignee scope, and reservation expiry values with defaults of open participation, user assignees, and 48 hours.
- PostgreSQL migrations added task participation fields, task reservations, and task-local implementor-ban storage.
- PostgreSQL task storage creates and reads participation fields, releases expired reservations, enforces one active reservation per task, rejects duplicate pending or active reservations by the same assignee, and gates submission eligibility to the active user reservation for reservation-required and approval-required tasks.
- Public task discovery hides actively reserved tasks from unrelated workers by default and shows them when `include_reserved=true`, while keeping them visible to the requester and active assignee.
- HTTP APIs were added for reserving a task, listing task reservations, approving a reservation, declining a reservation, and requester cancellation.
- Submission creation checks task reservation eligibility before validating and storing a response.
- Submission storage marks an active user reservation as submitted when that assignee submits.
- Generated Elm task contracts gained participation, assignee, availability, viewer-action, and reservation response types.
- Unit tests covered reservation service rules and submission eligibility rejection.
- HTTP end-to-end coverage was added for a reservation-required public task: reserve, unrelated submit rejection, default discovery hiding, include-reserved discovery, requester and active-assignee discovery visibility, and active-assignee submission.

The reservation foundation branch verification was performed:

- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-format check-policy check-ts check-copy-paste check-dead-code lint vet test frontend` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 go test -tags http_e2e ./tests/http_e2e -run TestReservationRequiredTaskDiscoveryAndSubmission` passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make test-integration` passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make test-http` passed with local Postgres access.

The requester ergonomics and task page instructions branch improved browser task workflows:

- The requester task list now includes tasks created by the requester even when those tasks are publicly visible.
- Browser task creation gained participation-policy controls and a reservation-expiry field.
- The funding form and collectible-award form now use task selectors sourced from the requester's task list instead of manual task identifier fields.
- Public discovery gained an include-reserved checkbox.
- Task detail pages gained reserve/request-approval actions, requester reservation review controls for approve, decline, and cancel, and task-specific REST and MCP examples.
- Generated static browser assets were rebuilt.

The requester ergonomics branch test coverage was updated:

- HTTP end-to-end coverage checks that a requester-created public task appears in that requester's task list.
- Playwright funding and collectible tests use the new task selectors.
- Playwright coverage was added for creating a reservation-required public task through the browser, opening it, reserving it as a worker, hiding it from another worker by default, and showing it with include-reserved.

The requester ergonomics branch verification was performed:

- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-format check-contracts check-policy check-ts check-copy-paste check-dead-code lint vet test frontend` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make test-integration` passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make test-http` passed with local Postgres access.
- `ELM_BIN=/opt/homebrew/bin/elm GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui` passed with local Postgres access.
- Manual screenshot review passed for `/tmp/sharecrop-requester-ergonomics.png`.

The review outcomes branch added requester review flows:

- A migration added the `changes_requested` submission state, stored review notes, reviewer metadata, review idempotency keys, and the `task_tip` ledger kind.
- Submission responses expose `review_note`.
- Acceptance supports full or partial credit payout and optional credit tips from the requester balance. Partial acceptance refunds withheld escrow to the funder.
- Request changes requires requester notes, stores the note, moves the submission to `changes_requested`, and reactivates a submitted user reservation for the same implementor.
- Rejection requires requester notes and supports optional partial credit payout from held escrow, optional credit tip from requester balance, and optional task-local implementor ban.
- Task-local implementor bans block direct open-task submissions as well as future reservations.
- HTTP endpoints were added for request changes and rejection, and the existing accept endpoint gained optional `payout_amount` and `tip_amount`.
- MCP tools were added for request changes and rejection, and the accept tool gained optional `payout_amount` and `tip_amount`.
- Browser task detail review controls now include review note, partial payout, tip, ban implementor, accept, request changes, and reject controls.
- Generated Elm contracts and static browser assets were rebuilt.

The review outcomes branch test coverage was updated:

- Ledger service tests cover rejection delegation and ban selection.
- Integration tests cover partial accept with tip, request-changes note storage and reservation reactivation, and reject with partial payout, tip, and implementor ban.
- HTTP contract fixture tests cover submission review notes, accept tips, and review responses.

The review outcomes branch verification was performed:

- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make test-integration` passed after the review outcome integration tests were added.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make test-http` passed after HTTP and MCP review endpoints were added.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod make check-format check-policy check-ts check-copy-paste lint vet frontend build` passed.
- `make check-dead-code` could not be rerun after final changes because the required network escalation for downloading `golang.org/x/tools` was rejected by the approval system.
- A final rerun of `make test-integration`, `make test-http`, and UI screenshot review could not be performed after the final frontend and handler refactor because escalation was rejected by the approval system.

The reward bundles branch added combined reward modeling:

- Task reward specs gained collectible-only and bundled credit-plus-collectible variants.
- A migration allowed `collectible` and `bundle` task reward kinds while keeping credit amounts required only for credit-bearing rewards.
- Task creation through HTTP and MCP accepts reward kinds `none`, `credit`, `collectible`, and `bundle`.
- Task list, detail, generated Elm contracts, MCP summaries, and MCP detail outputs expose `reward_collectible_count` alongside reward kind and credit amount.
- Opening a task now requires held credit escrow for credit-bearing rewards and a held collectible reward for collectible-bearing rewards.
- Credit funding can coexist with a collectible reward on bundled tasks.
- Accepting a bundled task pays the credit escrow and transfers the collectible in one accepted payout outcome.
- Same-key accept retries reconstruct bundled payout responses without paying twice.
- Refunding a bundled task through the credit refund endpoint returns both the held credits and the held collectible; the collectible-only refund endpoint rejects declared bundles so it cannot strand credit escrow.
- Browser reward labels show credits, collectibles, or both.

The reward bundles branch test coverage was updated:

- HTTP end-to-end coverage verifies that bundled tasks cannot open until both reward components are funded, acceptance pays both components, same-key accept retries remain idempotent, and bundled refunds return both credits and the collectible.
- HTTP end-to-end helper response shapes include reward kind, credit amount, and collectible count.

The reward bundles branch verification was performed:

- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make frontend` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 go test -tags http_e2e ./tests/http_e2e` passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 go test -tags integration ./tests/integration` passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-format check-policy check-ts check-copy-paste lint vet test-deno check-dead-code frontend` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui` passed with local Postgres access.
- `make check-contracts` regenerated the intended Elm contract changes and failed before commit because the generated files differed from `HEAD`; it should be rerun after the reward bundles commit.
- Manual screenshot review was skipped; Playwright UI coverage passed.

The MCP workflow and Streamable HTTP SSE branch added the remaining MCP workflow surface:

- MCP services and tools gained task reservation support: reserve/request approval, list task reservations, approve reservation, decline reservation, and cancel reservation.
- Reservation tool results return reservation identifiers, task identifiers, assignee kind and identifier, state, and requester identifier.
- Streamable HTTP MCP now stores initialized HTTP sessions and requires `Mcp-Session-Id` on later non-initialize POST requests.
- `GET /mcp` now serves `text/event-stream`, replays recent session response events after `Last-Event-ID`, stays open, and streams later POST responses to connected clients with event IDs.
- `DELETE /mcp` terminates the current session and later requests with that session ID fail.
- MCP sessions and recent response events are kept in the app process memory.
- Browser task detail MCP curl examples now show initialize first, then session-aware `submit_response` and `get_task_schema` tool calls.

The MCP workflow and Streamable HTTP SSE branch test coverage was updated:

- MCP unit tests cover the new reservation tool dispatch path.
- HTTP end-to-end MCP tests now initialize sessions, include `Mcp-Session-Id` on tool calls, cover reserve/list/approve reservation tools, cover SSE replay, cover live SSE delivery after a later POST, and cover session deletion.
- Existing MCP series tool HTTP tests now use initialized sessions.

The MCP workflow and Streamable HTTP SSE branch verification was performed:

- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 go test -tags http_e2e ./tests/http_e2e` passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 go test -tags integration ./tests/integration` passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui` passed with local Postgres access.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-format check-contracts check-policy check-ts check-copy-paste check-dead-code lint vet test-deno` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build` passed.
- Manual screenshot review was skipped; Playwright UI coverage passed.

The UI themes and GitHub Pages demo branch added static demo and documentation surfaces:

- [docs/user_stories.md](./docs/user_stories.md) was added to map demo visitor, requester, implementor, organization operator, agent operator, platform reviewer, and deferred stories.
- A GitHub Pages static site was added under `site/`.
- The Pages root serves the project landing page.
- `/demo/` serves an interactive static demo with localStorage-backed state.
- `/docs/` serves a documentation placeholder.
- The demo supports light and dark mode selection.
- The demo supports corporate, rustic, blocky, and showcase themes.
- The demo supports demo user selection for requester, implementor, organization reviewer, and agent operator perspectives.
- The demo includes mock Google, Apple, Microsoft, Facebook, and X.com sign-in buttons without implementing provider authentication.
- The demo includes a visible clear-state control.
- The demo maps requester creation, discovery, reservation, approval, submission, review, partial payout, tip, ban, REST instruction, and MCP instruction stories into one static workflow surface.
- A GitHub Actions Pages workflow was added to publish `site/` after pushes to `main` or manual dispatch.
- Playwright coverage was added for static demo theme switching, user switching, local state persistence, and state reset.
- A screenshot helper was added under `tools/` for repeatable demo screenshots.

The UI themes and GitHub Pages demo branch verification was performed:

- `make check-format` passed.
- `make check-ts` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `make test-deno` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make frontend` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-contracts check-dead-code lint vet` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui` passed with local Postgres access.
- Screenshot review was performed for `/tmp/sharecrop-screens/desktop-corporate-light.png`, `/tmp/sharecrop-screens/desktop-blocky-dark.png`, `/tmp/sharecrop-screens/mobile-rustic-light.png`, and `/tmp/sharecrop-screens/mobile-showcase-dark.png`.

The demo UI/UX repair branch refined the GitHub Pages demo:

- The demo was split into separate pages for overview, discovery, requester workflow, review queue, API/MCP instructions, and demo settings.
- The previous all-in-one scrolling demo was replaced with a focused app shell, page tabs, task tables, detail panels, and review panels.
- Demo login was moved into a discrete top-right account control that starts as Guest and opens a login panel.
- The login panel supports selecting a demo user and choosing mock Google, Apple, Microsoft, Facebook, and X.com provider buttons.
- The login panel closes after user selection, provider selection, or page navigation so it does not block other controls.
- Dark theme overrides were fixed for all visual themes.
- A demo audit helper was added to check deployed and local demo pages for console warnings, console errors, page errors, failed requests, and horizontal overflow.
- Playwright static demo coverage was updated for the separated navigation and login flow.

The demo UI/UX repair branch verification was performed:

- `make check-format` passed.
- `make check-ts` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `make test-deno` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-contracts check-dead-code lint vet` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui` passed with local Postgres access.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys tools/audit_demo_ui.ts` passed for deployed and local demo pages.
- Screenshot review was performed for `/tmp/sharecrop-screens/desktop-corporate-light.png` and `/tmp/sharecrop-screens/mobile-showcase-dark.png`.

The `task/demo-performance-flow-review` branch repaired the static demo after performance and flow review:

- The demo runtime was changed from per-render event binding to document-level delegated event handling.
- Text input now updates in-memory state and debounces localStorage writes instead of re-rendering the page on every keystroke.
- Locally created demo tasks, reservations, and submissions are bounded so localStorage state cannot grow without limit.
- localStorage quota failures clear the demo state and return the user to settings.
- The expensive sticky translucent top bar with backdrop blur was replaced with an opaque non-sticky top bar.
- The top-level demo pages were kept separate for overview, discovery, requester create, review, API/MCP instructions, and settings.
- Demo clear-state controls were moved out of the header and into settings.
- Requester task creation now shows title, description, reward, visibility, participation policy, and reservation expiry fields.
- Discovery actions now match the selected task policy: reserve, request approval, submit, or no action.
- Review controls are per reservation and per submission, with approve, decline, release, request changes, reject, accept, partial payout, tip, and ban controls.
- API/MCP instructions separate worker REST, worker MCP, requester MCP, and credential setup placeholders.
- The static demo Playwright test covers theme selection, user selection, typed task creation, local state persistence, and clear state through the settings page.
- The screenshot capture helper targets exact page-tab labels and captures overview, discovery, create, review, API/MCP, settings, desktop theme, and mobile theme states.
- The Elm build helper rejects the recursive npm Elm wrapper when `ELM_BIN` points to it, preventing local builds from hanging and flooding Node warnings.

The `task/demo-game-like-personas` branch expanded the static demo into a game-like mission board:

- The page labels became Command, Mission Board, Post Mission, Review Queue, Uplink, and Settings.
- The demo seed data grew from a few tasks to a larger mission set covering public and organization visibility, open submissions, reservations, requester approval, submitted work, changes requested, rejected work, accepted work, draft missions, funded missions, expired reservations, credit rewards, collectible rewards, and bundled rewards.
- The work-viewing surface changed from a table into mission lanes for Available, Reserved, Awaiting approval, Submitted, and Settled work.
- Mission cards now show rank, sector, policy, availability, and reward chips.
- Persona switching now changes the active persona, page, selected mission, and available actions.
- LocalStorage-backed state now includes activity feed entries, balances, collectible inventories, mission timelines, review drafts, local mission IDs, and mission state transitions.
- Requesters can locally draft, fund, open, and cancel missions.
- Implementors can locally reserve missions, request approval, submit payloads, and resubmit after changes are requested.
- Reviewers/requesters can locally approve or decline reservations, release reservations, request changes, reject with partial payout and tip, accept with payout and collectible transfer, and ban implementors from a mission.
- The Uplink page can simulate an agent run that creates an agent-labeled submission.
- Static demo Playwright coverage was expanded to verify persona switching, mission drafting persistence, and reserve-submit-accept transitions.
- Screenshot capture was updated for the new page labels and mission board screenshots.

The `task/demo-ui-polish-pass` branch polished the expanded static demo after specialized review:

- The review queue became persona-scoped so implementors do not see or act on requester review work.
- Mission board and review actions now operate on the task shown in the current filtered view instead of stale global selection.
- Request-changes review decisions no longer pay partial payout or tip credits; payouts apply only to accepted and rejected decisions.
- Review inputs save without rerendering while the user is clicking a decision control, preventing lost review clicks.
- Settled, rejected, and changes-requested submissions now render as outcomes instead of still-active decision forms.
- Mission cards now show persona-specific next actions plus requester or assignee context, with wider lanes and clearer card hierarchy.
- Mission briefings now show the expected response schema in a readable block.
- Page tabs, persona buttons, theme buttons, mode buttons, and mission cards expose current or pressed state to assistive technology.
- Demo localStorage reads and writes are guarded, normalized, and bounded before stored state is merged into the seed demo state.
- Static demo Playwright coverage was expanded for persona-scoped review access and request-changes resubmission flow.

The `task/demo-ui-polish-pass` branch verification was performed:

- `node --check site/demo/app.js` passed.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys npm:@playwright/test@1.61.0 test -c tests/playwright/playwright.config.ts tests/playwright/demo_static.spec.ts` passed.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys tools/capture_demo_screenshots.ts` captured desktop and mobile screenshots; screenshots reviewed included the Mission Board, Review Queue, Command desktop, and showcase mobile states.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys tools/audit_demo_ui.ts` passed for deployed and local demo pages.
- `make check-format` passed.
- `make check-ts` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `make test-deno` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-contracts check-dead-code lint vet` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui` passed.

The `task/demo-list-detail-navigation` branch changed the static demo discovery model:

- The top-level demo labels became Dashboard, Tasks, Post Task, Reviews, Agent/API, and Settings.
- The Tasks page stopped using the board-like lane abstraction and now renders a scannable task list.
- Task rows show title, objective, rank, sector, policy, requester or assignee context, status, next action, reward chips, and row-level action buttons.
- Clicking a task title or Open task page navigates to a separate Task Detail page instead of replacing a detail pane on the list page.
- Task Detail owns the full briefing, response schema, action console, task log, back-to-list control, and API/MCP handoff link.
- Requester rows without pending review work no longer show a Review queue call to action.
- Demo Playwright coverage was updated for task-list navigation, task-detail actions, and renamed navigation labels.
- Screenshot capture was updated for task-list and task-detail desktop and mobile states.

The `task/demo-list-detail-navigation` branch verification was performed:

- `node --check site/demo/app.js` passed.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys npm:@playwright/test@1.61.0 test -c tests/playwright/playwright.config.ts tests/playwright/demo_static.spec.ts` passed.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys tools/capture_demo_screenshots.ts` captured screenshots; reviewed images included desktop task list, desktop task detail, mobile task detail, and mobile dashboard states.
- `deno run --allow-env --allow-read --allow-write --allow-run --allow-net --allow-sys tools/audit_demo_ui.ts` passed for deployed and local demo pages.
- `make check-format` passed.
- `make check-ts` passed.
- `make check-policy` passed.
- `make check-copy-paste` passed.
- `make test-deno` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache go test ./...` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make check-contracts check-dead-code lint vet` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm make build` passed.
- `GOCACHE=/Users/zardoz/projects/sharecrop/.gocache GOMODCACHE=/Users/zardoz/projects/sharecrop/.cache/go-mod ELM_HOME=/Users/zardoz/projects/sharecrop/.elm ELM_BIN=/opt/homebrew/bin/elm DATABASE_URL=postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable SHARECROP_MIGRATIONS_DIR=/Users/zardoz/projects/sharecrop/migrations SHARECROP_ACCESS_TOKEN_SECRET=01234567890123456789012345678901 make e2e-ui` passed.
