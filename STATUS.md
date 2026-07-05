# Status

The repository contains pull request 1 through pull request 133 work, merged
into `main`, plus the current `task/fund-any-reward-kind-and-open-on-create`
branch. PR 108's GitHub Pages deployment failed three times in a row after
merge for what looked like a transient GitHub-side Pages backend issue
(build/artifact steps always succeeded; only `deploy-pages` failed or hung,
with a different symptom each time); most later PRs' deployments succeeded
on the first try with no code or workflow changes, confirming it was not a
code problem — though PR 127's deployment hit the same transient failure
again and cleared on a manual retry, so this class of flakiness is still
occasionally live, not fully resolved.

The 5-phase RBAC + API-token effort (PRs 115-121), two clean-up passes
(PRs 122, 124), a docs refresh (PR 123), the WASI production-hosting spike's
plan + Phase 0/1 (PR 125), ecosystem research (PR 126), and deployment-shape
requirements (PR 132), a Go 1.26.4 upgrade (PR 127), a strengthened
"at most one open PR at a time" rule in `AGENTS.md` (PR 128), the
`site/demo/backend.js` deprecation (PR 129: replacement CI coverage; PR 130:
deletion), a fix for the demo (WASM) backend collapsing every rejection to
HTTP 500 plus a corrected fund-panel visibility gate (PR 131), and making a
draft task always fundable regardless of reward kind plus opening a task in
the UI after creating it (PR 133) are complete.

Active task: `task/fund-any-reward-kind-and-open-on-create` continues a
batch of related feature requests started in PR 133. Since that PR merged:

3. **A creator can add a collectible to an existing task's reward from the
   task detail page.** The backend route
   (`POST /api/tasks/{task_id}/collectible-reward`) and its Elm API wrapper
   already existed but were never wired to any UI trigger. Added a
   draft-only "Add a collectible to this task's reward" panel (reuses the
   Collectibles page's existing `AwardClicked`/`awardTaskId` plumbing, now
   synced to the viewed task the same way `fundTaskId` already was).
   Fixing this also surfaced that `internal/db/collectible_store.go`'s
   `FundCollectibleReward` never transitioned the task's `reward_kind`
   (none -> collectible, credit -> bundle) the way the credit-funding path
   now does — fixed the same way, with a new integration test. Separately
   fixed a real staleness bug this surfaced: neither the fund nor the
   award success handlers re-fetched the viewed task's own detail record,
   so the owner-controls buttons (e.g. the refund button appearing) stayed
   stale until a manual page reload — added `Api.refreshLedgerAndTaskDetail`
   and used it from both.
5. **An org admin can award an org-owned collectible to an active member**
   (backend + HTTP route + Elm UI all new — this one didn't already exist).
   New `assets.Service.AwardOrganizationCollectible` /
   `CollectibleStore.AwardOrganizationCollectible`, gated by
   `org.PermissionManageMembers` at the HTTP boundary
   (`POST /api/organizations/{organization_id}/collectibles/{id}/award`),
   verifying in the same transaction that the collectible actually belongs
   to that org and the recipient is an active member. New Elm UI on the
   organization detail page: a member picker plus an "Award to member"
   button per org-owned collectible. Covered by a new HTTP e2e test
   (permission-denied, non-member-recipient, and happy-path cases); not
   yet covered by a Playwright regression test since the shared Playwright
   server has no platform-admin bootstrapped (needed to get a collectible
   into org ownership in the first place) — verified manually instead by
   driving a real browser against a separately-started server with
   `SHARECROP_ADMIN_USER_IDS` set.

Confirmed already done, no work needed: **4. an admin can award
collectibles to any user** — `POST /api/collectibles/award` (platform-admin
gated, catalog-only) already exists, with a full Elm UI
("Admin: award a default collectible" on the Collectibles page, recipient
kind user/team/organization).

`internal/wasmdemo` does not yet implement item 5's new endpoint (org
collectible award) — the demo can add collectibles to a task's reward
(item 3) but not the org-admin-awards-to-member flow. Not addressed in this
branch.

