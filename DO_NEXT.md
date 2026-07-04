# Do Next

Current priority from
[docs/application_readiness_review.md](./docs/application_readiness_review.md):

Active branch:

1. `task/mcp-orgs-teams-collectibles-notifications-users` is in progress —
   Phase 4b of a 5-phase, explicitly-planned RBAC/API-token effort (full plan
   agreed with the user; each phase ships as its own PR). Phase 1 (PR 115)
   laid the credential-model foundation; Phase 2 (PR 116) added
   organization-wide tokens; Phase 3 (PR 117) centralized org-token
   authorization; Phase 4a (PR 118) wired `auth.Subject`/`OrgSubject` through
   the whole MCP transport stack. **This phase (4b)** adds the ~30 new MCP
   tools the Phase 4 research found missing: organizations/teams (13 tools),
   org-wide credential self-management (3 tools), collectibles (8 tools,
   excluding the admin-gated `award_collectible` — deferred to 4c),
   notifications (2 tools), users/profile (4 tools). Per the "MCP tracks
   REST, doesn't exceed it" principle from 4a, each tool's actor-kind was
   decided by checking its specific REST handler, not just what the domain
   layer permits — only `get_team`/`add_team_member` ended up org-token
   capable, matching REST's `requireUserOrOrgSubject`-gated handlers exactly.
   Split `internal/mcp` into 5 new per-domain file pairs
   (`tools_orgs.go`/`tool_calls_orgs.go`, etc.) instead of growing the
   existing monolithic files further. Verified with 9 new `http_e2e` tests
   plus manual curl verification against the real server. See
   `STATUS.md`/`WHAT_WE_DID.md` for the full writeup. Remaining: Phase 4c
   (admin-gated MCP tools — `award_collectible`, moderation, privacy,
   platform-admin, audit — plus the double-check mechanism REST's
   `requireAdminSubject` already has but MCP dispatch doesn't), Phase 5
   (Elm UI).

2. `task/task-detail-reorder-profile-links-uiux` (PR 114, merged) refined
   the task detail and profile pages for usability at the user's explicit
   direction: Report task is now a collapsed disclosure; reservation status
   moved to the top of the task detail page, above role-specific controls.
   That reorder surfaced a real gap — task owners previously had no way to
   see or act on a pending reservation request through the browser at all
   (Approve/Decline existed in code but were never reachable by the one role
   allowed to click them). Fixed, plus a new test since that flow had zero
   prior coverage. Also scoped reservation/submission action buttons to who's
   actually entitled to click them (previously any worker saw every
   reservation's buttons, including other people's). People now link to
   their profiles wherever their user ID appears (reservation holder,
   submitter, task creator, notification actor, admin user ID). On profile
   pages, "Submissions" now only shows on your own profile (the API 403s for
   anyone else), and "Public work" is relabeled "Currently working on" with
   richer per-task info.

3. `task/merge-tasks-nav-uiux-polish` (PR 113, merged) consolidated the nav
   further at the user's explicit direction: New task, Discovery, and the
   whole Work menu (Submissions, Series) are gone as separate nav
   destinations — Tasks is now a hub (My tasks + Discover public tasks always
   expanded, My submissions + Series collapsed, "+ New task" button at the
   top). Inbox moved into the Account menu. Nav is now just Overview/Tasks
   flat plus Manage/Account menus (down from 8 items). Added inline task
   funding (collapsed "Fund this task" panel on the task detail page, plus a
   row-level "Fund" button), reusing the standalone Funding page's exact Msg
   plumbing. Found and fixed two real pre-existing bugs this refactor
   uncovered (submissions-pagination and series-refresh handlers were keyed
   off the current page being an exact standalone route, so they silently
   no-op'd from the new hub).

4. `task/navbar-dropdown-menu-more-seed-tasks` (PR 112, merged) followed up
   on PR 111's nav grouping (still 15 buttons across 3 rows) with real
   dropdown menus (Work/Manage/Account), collapsing to one row. A native
   `<details>`/`<summary>` first attempt had two real bugs the Playwright
   suite caught: Elm silently drops inline `onclick` attributes, so closing
   the menu on navigation needed real Elm state instead
   (`openNavMenu`/`ToggleNavMenu`, reset in `enterPage` on every route
   change) rather than a native-only approach. Also expanded the WASM demo's
   seeded tasks from 6 to 14.

5. `task/ui-navbar-declutter-a11y-seed` (PR 111, merged) was a deliberately
   large bundled UI/UX pass (one PR by explicit request, not the usual
   one-task-per-branch split): a grouped `<nav>` navbar replacing the old flat
   14-button row (with a fixed Profile active-state bug and a new Submissions
   nav entry), further `Ui.disclosure`-based decluttering on Create Task,
   Collectibles, account settings, user submissions, and task series pages, an
   accessibility pass (per-page `<h1>`, `aria-hidden` decorative sprites,
   color-differentiated status badges, a real focus-visible ring on base-theme
   text inputs), a richer WASM demo seed (team, more org members, funded task,
   pending submission/reservation/notification, awarded collectibles — without
   touching the balance/catalog-count invariants existing Playwright specs
   depend on), and several small opportunistic fixes found along the way
   (notably: the demo skin's nav-highlight CSS was unconditionally overriding
   every nav link to look the same, so the active-page highlight was invisible
   app-wide, not just for Profile). See `STATUS.md`/`WHAT_WE_DID.md` for the
   full breakdown.

6. `task/task-series-wasm-support` (PR 110, merged) continued hand-testing the
   browser demo: found that `/api/task-series` (list, create) and
   `/api/task-series/{id}` (detail) were entirely unclassified in the WASM demo
   (a 404) — the whole Task Series feature had zero WASM demo support, not just
   a single missing route. Implemented `StoredTaskSeries` storage and a new
   `TaskSeriesHandler` covering create/list/detail, matching real backend
   response shapes exactly (including that create returns the full detail
   wrapper, not a bare series object, found by hitting a second decode error
   after fixing the first 404). Series lifecycle transitions, series-task
   membership, and series comments are explicitly out of scope and remain a
   known gap (see item 5 below and `BUGS.md`).

   Note: PR 108's GitHub Pages deployment failed three times in a row after
   merge (10-minute timeout, then two different deploy-pages errors on reruns),
   but PR 109's deployment succeeded on the first try with no code or workflow
   changes — confirming it was a transient GitHub-side issue that has since
   cleared, not something to keep investigating.

Next recommended work:

1. Keep expanding shared scenario parity as new user-visible API surfaces are
   added. The current suite covers selectors, collectible
   mint/transfer/create-time refund, comments, notifications with task metadata,
   small task/submission attachments, team/organization queue search/type/sort,
   persisted saved queue views, organization reviewer acceptance,
   sensitive-field response metadata/redaction state, privacy
   request/audit/resolution/retention shape, moderation report/admin-list/audit
   triage shape, platform-admin grant/revoke/audit shape, admin audit,
   personal-ledger, organization-ledger, notification, and user-submission
   pagination, and multi-actor reservation/submission acceptance.
2. Keep running shared scenario parity against real APIs as behavior changes.
   The explicit-session runner accepts `--origin`, access-token input, and
   refresh-token input. The local real runner can register a scenario admin and
   grant platform-admin state through `DATABASE_URL` and `psql`.
3. Keep expanding generated/fixture-level HTTP contract coverage as the API
   surface grows.
4. Audit remaining raw-ID browser flows and replace high-traffic fields with
   selectors where directory data exists. No confirmed high-traffic raw-ID input
   remains after the latest audit in
   [docs/raw_id_browser_flow_audit.md](./docs/raw_id_browser_flow_audit.md).
5. Add enough explicit host-backed stores and request handlers for the Go/WASM
   backend target to run the shared scenario parity suite without fallback
   stores. The deployed browser demo is the first host, but WASM is also a
   production backend execution target. User, account-token, agent-credential,
   platform-admin, audit-event, collectible, privacy-request, moderation-triage,
   saved-queue-view, task, task series (create/list/detail only), attachment,
   notification, organization, organization-member, team, comment, reservation,
   submission, and ledger storage/handler slices now exist. The Go `js/wasm`
   command can be built, loaded, explicitly configured with host adapters, and
   used for the shared scenario parity suite. The demo defaults to compiled WASM
   artifacts and configured browser host functions. Two confirmed remaining
   gaps: team-membership storage (`GET /api/teams/{team_id}` always returns an
   empty member list), and task series lifecycle/membership/comments
   (publish/unpublish/close/reopen, adding/removing/reordering tasks in a
   series, and series comments).
6. Add provider email delivery only if the product direction changes; current
   account/org setup stays admin-driven.
7. Build a genuine production non-browser WASM host: persistent storage (file or
   database-backed) behind the same `storageHas`/`storageGet`/`storagePut`
   contract, a real clock, verified-session actor resolution instead of the
   reference host's `setActor` test hook, and cryptographically random
   IDs/secrets instead of the reference host's sequential counter. The
   non-browser host contract is documented and the reference/test implementation
   exists (`tools/wasm_runtime_loader.ts`); a deployable production
   implementation does not yet exist because there is no concrete non-browser
   deployment target for it yet. Keep re-running `deno task measure:wasm` as the
   WASM binary grows.

Recently finished:

1. PR 109 was merged into `main`; its GitHub Pages deployment succeeded on the
   first try, confirming PR 108's three prior deployment failures were a
   transient GitHub-side issue that has since cleared.
1. The `task/team-detail-404-fix-and-declutter` branch fixed a fourth real
   broken flow found by hand-testing the Go/WASM demo (the team detail page
   404ing for every team), and decluttered the team detail page's team-work
   filter panel with `Ui.disclosure`.
1. PR 108 was merged into `main`. Its GitHub Pages deployment failed three times
   in a row (see the note under "Active branch" above); the merge and code are
   fine, only the deploy pipeline is affected.
1. The `task/org-detail-declutter-and-audit-fix` branch fixed a third real
   broken flow found by hand-testing the Go/WASM demo (the organization detail
   page's "Organization audit" section 404ing, plus a copy-paste bug that stored
   that error under the wrong heading), and decluttered the organization detail
   page and the Collectibles page's admin-only picker with `Ui.disclosure`.
1. PR 107 was merged into `main`.
1. The `task/ui-ux-declutter-and-profile-fix` branch fixed two real broken flows
   found by hand-testing the Go/WASM demo in a browser (the profile page's
   `GET /api/users/{user_id}` and `GET /api/users/{user_id}/work`), introduced
   the `Ui.disclosure` collapsible-section component and applied it to the Admin
   page/Tasks/Discovery filters/Create Task's advanced fields, and fixed a
   virtual-DOM node-reuse bug by keying the page root view by route.
1. PR 106 was merged into `main`.
1. The `task/openapi-schema-field-access` branch closed the one confirmed
   generator gap from the prior branch: `internal/openapi` resolves a two-value
   type assertion (`name, ok := x.(Type)`) and a field-access expression
   (`response.value`), including through untagged internal result-union structs,
   so `createModerationReport`'s response resolves to a typed schema too. 99/106
   responses resolve (up from 98/106); 39/61 request bodies, unchanged.
1. PR 105 was merged into `main`.
1. The `task/openapi-typed-schemas` branch added typed per-route
   request/response JSON schemas to `docs/openapi.json`, derived directly from
   the Go DTO struct each `internal/http` handler actually decodes/writes
   (resolved through `go/ast`, not a hand-authored mapping). 98/106 responses
   and 39/61 request bodies resolved to a typed schema; the next branch closed
   the one remaining confirmed gap.
1. PR 104 was merged into `main`.
1. The `task/landing-docs-visual-redesign` branch designed and implemented real
   CSS for `site/index.html` and `site/docs/index.html` (previously unstyled in
   production) and fixed the broken `demo/styles.css` link, using a new
   hand-authored `site/marketing.css`. `site/docs/openapi.html` was restyled to
   match.
1. PR 103 was merged into `main`.
1. The `task/openapi-pages-subpage` branch published the generated OpenAPI
   document as a browsable page on the deployed GitHub Pages site
   (`site/docs/openapi.html`, backed by a committed `site/docs/openapi.json`
   copy) and wired `make check-openapi` into PR CI. It found (but left unfixed,
   as out of scope) that `site/index.html` and `site/docs/index.html` render
   unstyled in production; the next branch fixed that.
1. PR 102 was merged into `main`.
1. The `task/generated-openapi-reference` branch added `internal/openapi`
   (`make openapi` / `docs/openapi.json`), a generator that parses
   `internal/http`'s route registrations and local call graph with `go/ast` to
   produce an accurate method/path/operationId/bearer-auth inventory, closing
   the "no generated OpenAPI reference" documentation gap. It did not wire
   `make check-openapi` into `.github/workflows/ci.yml`, only the Makefile; the
   next branch fixed that gap.
1. PR 101 was merged into `main`.
1. The `task/wasm-nonbrowser-host-measurement` branch added
   `deno task measure:wasm` (artifact size, startup time, host-process memory,
   request latency) and documented the non-browser host adapter reference
   (`tools/wasm_runtime_loader.ts`) plus the gap between that reference host and
   a genuine production non-browser host.
1. PR 100 was merged into `main`.
1. The `task/wasm-default-demo-shared-parity` branch made the compiled Go/WASM
   backend the default static-demo backend, expanded explicit WASM behavior
   slices for collectibles, account tokens, agent credentials, admin operations,
   privacy resolution/redaction, and moderation projection writes, and refreshed
   docs/continuity for the Go/WASM demo default.
1. PR 99 was merged into `main`.
1. The `task/wasm-browser-host-full-parity-gates` branch added an opt-in browser
   WASM host path, Go WASM demo artifact build, Pages workflow integration,
   expanded WASM dispatch/scenario coverage for privacy requests, saved queue
   views, organizations, teams, task interactions, reservations, submissions,
   ledger, and continuity updates.
1. PR 98 was merged into `main`.
1. The `task/wasm-host-adapters-scenario-parity` branch wired explicit host
   configuration into the Go `js/wasm` command, ran request execution through
   the configured WASM host for task/comment/reservation/submission/ledger
   slices, updated the WASM scenario runner to exercise that path, recorded
   remaining WASM behavior slices, and verified deployed Pages routing.
1. PR 97 was merged into `main`.
1. The `task/wasm-submission-parity-host-adapters` branch added explicit
   host-backed Go/WASM backend slices for submissions, comments, reservations,
   and ledger surfaces; a Go `js/wasm` command; a Deno WASM runner that loads a
   compiled Go `.wasm` backend and checks required exports; and host adapter
   shape documentation.
1. PR 96 was merged into `main`.
1. The `task/wasm-org-team-parity-contracts-rawid` working tree added
   no-fallback WASM demo organization, organization-member, organization-team,
   and standalone-team storage/handlers, organization/member/team route
   classification, shared scenario parity for organization member lifecycle
   shape, org/team contract fixtures, backendless-demo deactivation response
   parity, DB-backed browser assertions for member role/deactivation controls,
   and a raw-ID/WASM/demo parity doc refresh. It also fixed the real
   organization member list so managers can see deactivated non-removed members
   while permissions still require active membership.
1. PR 95 was merged into `main`.

1. The real-parity-wasm-submission-contracts-rawid-attachments branch added a
   local real API shared scenario parity runner that registers a scenario admin,
   grants platform-admin state through `DATABASE_URL` and `psql`, and runs the
   same shared suite against a real local API.
1. The branch updated the explicit real scenario runner to carry refresh-token
   cookies, accept refresh-token file input, and include response error context
   in status failures.
1. The branch tightened shared scenario parity to satisfy the real task-create
   and submission lifecycle contract, then added adjacent one-row pagination
   checks for personal ledger, organization ledger, and notifications.
1. The branch added a no-fallback WASM demo notification browser store,
   notification route classification, and list/mark-read request handlers with
   actor-scoped validation.
1. The branch expanded HTTP fixture coverage for task owner, visibility,
   placement, and payload request subshapes.
1. The branch refreshed the raw-ID audit and WASM/demo parity docs.
1. The branch added DB-backed Playwright coverage for rejected attachment type,
   oversized file, and five-file limit UI guardrails.

1. The real-parity-wasm-contracts-pagination-hardening branch hardened the real
   shared scenario parity runner with `/healthz` probing, `--token-file`, and
   contextual invalid-JSON errors.
1. The branch added a no-fallback WASM demo task browser store, task
   create/detail route classification, and task create/detail request handler
   with explicit actor, ID-source, task, and attachment validation.
1. The branch added a five-attachment request limit across the Go API,
   backendless demo, browser guards, and WASM attachment storage.
1. The branch expanded HTTP contract fixtures for standalone attachment request
   and response shapes.
1. The branch added browser and backendless-demo pagination for personal ledger,
   organization ledger, and inbox notification lists.
1. The branch added DB-backed Playwright coverage for creating a task with a
   small attachment through the real backend.
1. The branch fixed backendless-demo task payload kind drift from `inline` to
   `json` and removed the dead browser display branch for `inline`.
1. The branch refreshed API/readiness/demo-parity/WASM/continuity docs and
   passed deployed GitHub Pages routing verification.

1. The parity-contract-wasm-pagination-uploads branch added small task and
   submission attachments under 500 KiB, inline Postgres attachment storage,
   generated Elm attachment contracts, browser upload controls and attachment
   links, backendless-demo upload validation, shared scenario parity for task
   and submission attachments, user-submission pagination, explicit WASM demo
   attachment storage, a raw-ID audit refresh, and deployed Pages routing
   verification.
1. The branch also fixed a backendless-demo default-visibility response mismatch
   found by the new browser upload flow.

1. The postmerge-db-parity-wasm-pagination-coverage branch cleaned post-PR-91
   continuity, added admin pagination controls for audit events,
   platform-admins, privacy requests, and moderation reports, fixed
   backendless-demo pagination for those list routes, expanded shared scenario
   parity for admin audit pagination, added data-export response fixture
   coverage, added a demo-only Playwright config, and hardened DB-backed
   Playwright registration failure messages.
1. The branch added explicit WASM demo saved-queue-view browser storage, route
   classification, and request handlers with fail-loud validation and no
   fallback stores.
1. The branch refreshed raw-ID, readiness, demo-parity, WASM-spike, status,
   bugs, and next-task docs. DB-backed checks and DB-backed Playwright screens
   passed against isolated local PostgreSQL 15.

1. The parity-wasm-dashboard-revision-polish branch added shared scenario parity
   for persisted saved queue views, request fixture coverage for privacy
   resolution and saved queue view commands, a raw-ID browser-flow audit
   refresh, no-fallback WASM privacy-request storage and request handlers,
   saved-view/demo status parity fixes, queue/revision count headings, broader
   demo/mobile Playwright coverage, and continuity updates.

1. The db-admin-wasm-parity-hardening branch ran database-backed checks, fixed
   revoked platform-admin authorization, added focused integration tests for
   platform-admin lifecycle, moderation triage, privacy retention, and
   sensitive-field access events, expanded demo admin Playwright coverage, added
   a no-fallback WASM moderation-triage request handler, expanded shared
   scenario parity for admin audit events, and refreshed raw-ID/readiness/API
   docs.

1. The admin-moderation-retention-wasm branch added platform-admin lifecycle
   configuration, shared admin authorization gates, moderation report
   triage/filtering/subject links, privacy retention execution, sensitive-field
   access event recording, expanded contracts and scenario parity, backendless
   demo parity, explicit WASM moderation-triage browser storage, admin UI
   controls, and a blank-select value fix for selector-backed flows. Hard
   deletes remained prohibited.

1. The moderation-parity-contract-wasm branch added task reporting, admin
   moderation report listing, moderation audit projection, generated Moderation
   Elm contracts, HTTP wire-shape fixtures, shared scenario parity for
   moderation report/admin-list/audit shape, backendless demo parity, focused
   Playwright demo coverage, a raw-ID browser-flow audit, and a narrow
   `internal/wasmdemo` request-adapter spike. Hard deletes remained prohibited.

1. The privacy-ops-demo-wasm-parity branch added admin browser resolution for
   privacy requests, richer persisted data-export JSON, sensitive-field
   redaction state/counts/events, generated contract and HTTP fixture updates,
   shared scenario parity for privacy resolution and redaction effects,
   backendless demo parity, focused Playwright demo coverage, saved-view label
   polish, demo CSS build copying, readiness/API/WASM docs refresh, and WASM
   compile-check findings. Hard deletes remained prohibited.

1. The persisted-ops-privacy-lifecycle branch added persisted saved queue views,
   organization ledger and org-scoped audit dashboard panels, persisted privacy
   request listing/resolution with export JSON and sensitive-field redaction
   state updates, standalone-team task assignees, persisted MCP SSE polling
   fan-out groundwork, backendless demo parity, generated contracts, and
   continuity updates. Hard deletes remained prohibited.

1. The org-ops-queues-privacy branch added in-session saved views for team work
   and organization task queues, a loaded-data organization operations
   dashboard, a worker submission revision timeline, audited privacy request
   creation, generated Privacy Elm contracts, HTTP fixture/unit coverage, shared
   scenario parity for privacy request/audit shape, backendless demo parity,
   browser assertions, and docs updates.

1. The queue-revisions-ops-privacy branch added task-list `task_type` and `sort`
   filters, browser queue controls, audit-event filters, submission
   sensitive-field metadata in responses, visible worker submission response and
   privacy summaries, shared scenario parity for queue type/sort and sensitive
   metadata, backendless demo parity, and docs updates.

1. The server-queues/revisions/parity branch added task-list search filters,
   server-side search/pagination for organization task and team work queues, a
   revision-inbox shortcut that opens the task with the prior response
   prefilled, stricter task-list pagination errors, shared scenario queue-search
   coverage, HTTP E2E queue-search coverage, and backendless demo parity for
   organization-team queue search.

1. PR 82 was merged.

1. The post-PR81 dashboards/revisions/parity branch added loaded-list
   search/filter controls for team work, organization tasks, requester tasks,
   and discovery; added a worker revision inbox; surfaced dashboard load
   failures in section-specific messages; tightened submission-comment
   notification metadata parity; added browser coverage for the new flows; and
   added [docs/onboarding.md](./docs/onboarding.md).

1. PR 81 CI passed, including `db-checks` and Playwright.

1. The readiness-dashboard-docs-parity branch updated post-PR80 continuity,
   confirmed PR 80 CI passed with `db-checks` and Playwright, repaired stale
   readiness-review gaps, added team work dashboard sections, submission-comment
   notifications, inbox task links, task/discovery pagination controls,
   API/MCP/scheduling docs, and backendless demo thread-access parity.

1. PR 80 CI passed, including `db-checks` and Playwright.

1. The parity-contract-discussion-polish branch confirmed PR 79 CI passed with
   `db-checks`, cleaned stale readiness/user-story notes, and recorded that
   current raw-ID browser exposure is limited to protocol
   surfaces/links/audit/API examples unless a new high-traffic flow is found.
1. The branch expanded shared scenario parity with organization reviewer
   acceptance of an organization-owned task funded from the organization
   balance.
1. The branch tightened backendless demo review permissions for
   organization-owned tasks, initialized balances for newly created demo
   organizations, and added a submission-comment request wire-shape fixture.
1. The branch made the browser auto-open the submission discussion after a
   worker submits and after a reviewer action succeeds, with focused Playwright
   assertions added for that behavior.

1. The runtime-parity/reward-hardening branch added `make db-checks`, a PR CI
   database-backed runtime job, deployed Pages routing verification,
   organization-scoped collectibles, in-transaction collectible tips,
   deletion-semantics documentation, collectible contract fixture expansion, and
   shared scenario parity for create-time collectible refund.
1. The branch removed the separate accept-then-gift collectible tip window by
   settling collectible tips inside `LedgerStore.AcceptSubmission`.
1. The branch re-enabled `transferable_within_organization` tips with an
   explicit collectible organization scope and active-membership checks.
1. The branch kept rewards limited to Sharecrop credits and admin-minted
   Sharecrop collectibles.
1. PR 79 CI passed, including `db-checks`.

1. The multi-actor parity branch added scenario clients that carry distinct
   actor tokens and a shared scenario for approval-required reservation, owner
   approval, worker submission, owner acceptance with payout/tip, worker
   balance, and owner/worker notifications.
1. The branch made the backendless demo token-aware for local demo bearer
   tokens, so protected demo routes fail on missing/unknown tokens instead of
   silently acting as the seeded user.
1. The branch replaced the series add-task raw task-ID field with the existing
   task selector.
1. The branch added HTTP wire-shape fixtures for organization,
   organization-member, team-member, task-series-list, collectible, and
   collectibles wrappers.
1. The branch recorded the current WASM decision: keep the JavaScript demo
   backend until explicit browser storage adapters can satisfy the documented
   adoption gates without fallbacks.

1. The expanded-parity branch added admin operations, account-token issue shape,
   collectible catalog/mint/transfer, submission creation/comments, and
   notification read shape to the shared scenario parity suite.
1. The branch ran
   `deno task check:pages-routing -- --origin https://e6qu.github.io/sharecrop`
   successfully against the deployed GitHub Pages site.
1. The branch added selector-backed user/team/organization controls for admin
   default-collectible award recipients and collectible transfer recipients.
1. The branch expanded HTTP wire-shape fixtures for health/error/empty
   responses, ledger list, teams/tasks/reservations/submissions wrappers, full
   task response, agent credential request/created/list responses, and user
   profile response.
1. The scenario-parity/selectors/contracts branch added a shared scenario runner
   used by Deno demo tests and by `tools/run_scenario_parity.ts` for real API
   checks with an explicit origin/token.
1. The branch added paginated/typeahead selectors for users, organizations,
   standalone teams, and organization teams, with matching Go API query support
   and backendless demo query/pagination behavior.
1. The branch expanded HTTP wire-shape fixtures for request/command contracts
   and newer response surfaces including series, task comments, team detail,
   collectible catalog, and account-token responses.
1. The branch added `tools/check_pages_routing.ts` plus a Deno task for
   post-deploy GitHub Pages root/docs/demo/asset checks.
1. The branch documented the WASM demo backend spike in
   [docs/wasm_demo_backend_spike.md](./docs/wasm_demo_backend_spike.md),
   requiring explicit browser storage adapters and no fallback behavior before
   adoption.
1. The runtime-notifications branch added a persisted notification inbox,
   generated notification contracts, browser Inbox page, demo notification
   routes, HTTP contract fixtures, and domain/store tests.
1. The runtime-notifications branch added direct integration tests for audit
   event listing, notification lifecycle, Postgres rate-limit buckets, MCP HTTP
   session counts, and persisted MCP replay events. These tests require
   `DATABASE_URL`.
1. The runtime-notifications branch persisted MCP HTTP replay events in
   Postgres. Live SSE subscriber channels remain process-local.
1. The runtime-notifications branch added
   [docs/demo_semantic_parity.md](./docs/demo_semantic_parity.md), recommending
   shared scenario parity tests before any Go/WASM demo backend spike.
1. The combined runtime/audit/team-dashboard branch wired Postgres-backed
   rate-limit buckets, persisted MCP HTTP session identity for production
   `serve`, admin audit writes/viewing, team work dashboards, generated Admin
   Elm contracts, and demo base-path/bundle parity.
1. The branch kept email provider delivery and anonymous worker identity out of
   scope. Account and organization setup remains admin/org-admin driven.
1. The branch removed a response-encoding fallback in MCP raw response handling;
   response encoding failures now fail loudly.

Earlier finished:

1. The combined product-readiness branch added an operations runbook, systemd
   service template, operations-state schema foundation, and admin operations
   status endpoint.
2. The browser user selector can query `/api/users?query=...`; selector
   Playwright coverage now exercises the search path.
3. Account verification/reset token issue supports
   `SHARECROP_ACCOUNT_TOKEN_DELIVERY=log`, while local/test API-token mode
   remains available.
4. HTTP contract fixture coverage now includes account-token sent responses,
   user directory responses, submission comments, and operations status.
5. Account deletion semantics are deactivation plus password credential removal,
   refresh/account-token revocation, and email anonymization.
6. Submission comment posting uses a real form submit button, so click and Enter
   behavior match.
7. Reward docs now state the current product boundary: Sharecrop credits and
   admin-minted Sharecrop collectibles only.

- The backendless demo route surface is checked against the real HTTP router.
  Intentional real-only skips are health, MCP, and root/static serving routes.
- The backendless demo has Deno contract coverage for account lifecycle, user
  directory, task list/detail, and create-time collectible reward response
  shapes.
- The demo implements account lifecycle routes, `/api/users`,
  profile/password/account responses, clearer 404s for unknown demo API routes,
  email-backed org/team member provisioning, and create-time collectible reward
  escrow.
- Shared Playwright scenario constants cover account lifecycle and
  selector-backed reward creation flows.
- Real backend coverage for the carried-over behaviors is confirmed by existing
  HTTP E2E and targeted Playwright account/selector specs.

Recently finished:

- The combined follow-up branch added account lifecycle endpoints/UI,
  authenticated user directory, selector-backed user/team task creation
  controls, create-time collectible escrow for collectible/bundle rewards, and
  real-app Playwright coverage.
- The earlier combined follow-up branch added organization role management,
  organization reviewer browser parity, task-local worker submissions,
  selector-backed organization flows, reward-kind creation, demo parity, and
  documentation updates.
- Organization-team assignment now has reservation/request-approval and
  submission eligibility through HTTP, MCP, selector-backed browser controls,
  and demo behavior.

Previously agreed multi-PR roadmap (humans + agents get full lifecycle/feature
parity):

- PR1 (done): lifecycle/parity basics — MCP `open_task`/`fund_task` +
  participation in `create_task` + `list_tasks` state filter + `review_note` in
  `get_submission_status`; human UI authors a response schema + payload; docs
  scope names fixed.
- Series (done, `task/series-first-class`): task series promoted to a
  first-class domain — description + draft/published/closed lifecycle,
  creator-only add/remove/reorder of member tasks, a series comment thread, a
  stable URL, and draft-gating of member-task execution. This delivered
  series-level commenting; task-level and submission-level comments (the
  original PR3 scope) are still open below.
- PR2 (done, `task/dev-templates-comments`): pre-baked developer task types
  (code_review/security_review/product_review/ui_ux_review/qa_testing) with a
  template catalog in the create UI, a typed `reference_url` (the PR/resource to
  work on), and a per-task comment thread (HTTP + MCP + UI). This also delivered
  the task-level half of PR3's comments; submission-level comments remain.
- PR2 (superseded) was: pre-baked developer task TYPES — a task type/category
  and a catalog of templates (code review, security review, UI/UX review,
  product review, QA/testing) each with a description skeleton, a typed PR/URL
  field, and a ready-made response schema; surfaced in the create UI and
  `create_task`. (Net-new: type field across DB -> contract -> Elm -> MCP, plus
  a templates catalog.)
- PR3 (mostly done): comments now exist on SERIES and TASKS. Remaining: a
  comment thread on a SUBMISSION so requester and worker can discuss a specific
  submission (two-way, not just the one-way `review_note`). Reuse the
  `task_comments` shape.
- PR4 (descoped — do NOT propose a server-side scheduler): scheduling /
  recurrence is intentionally NOT a server feature. Decision (2026-06-25):
  recurring/scheduled task posting is the responsibility of a **local agent** (a
  client running a cron/work-loop that calls the existing MCP/API
  `create_task` + `open_task` + `fund_task` on its own cadence). The Sharecrop
  server stays request/response with no background job runner, `task_schedules`
  table, or recurrence model. If scheduling resurfaces, the answer is an
  agent-side recipe/example (e.g. a cron + MCP snippet in the docs), not a
  server scheduler.

Smaller parity follow-ups noticed during the review (fold into the PR they fit,
or a cleanup PR): the human accept flow cannot send a collectible tip though the
API supports it [DONE in task/ui-cancel-collectible-tip]; no UI to cancel a task
[DONE], create/list standalone teams, deactivate an org member, or refund a
collectible reward [DONE]; no small file/image attachment support [DONE in
task/parity-contract-wasm-pagination-uploads].

Polish follow-ups from `task/polish-bugfix-uiux-review`:

- UI minors: add `type_ "button"` to any remaining secondary buttons that move
  into forms; continue replacing raw-id fields as directory-backed selectors
  become available on more pages.

Other queued work:

1. Do not add anonymous worker identity unless the product direction changes.
   Registered-user submissions are the current model.
2. Do not add provider email delivery yet. Account and organization setup stays
   admin/org-admin driven.

Before starting, reread [AGENTS.md](./AGENTS.md) and update the continuity files
if task scope changes.
