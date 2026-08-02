# HTTP API Reference

This reference lists the stable application routes used by the Elm UI, external HTTP clients, and shared scenario tests.

All protected routes require `Authorization: Bearer <access_token>` unless the route is explicitly public. Browser sessions also use the refresh-token cookie for `/api/auth/refresh`. When Shauth is configured, a user access token is accepted only alongside its active Sharecrop browser-session cookie; RP-Initiated, Front-Channel, and Back-Channel Logout therefore fail closed immediately instead of leaving the SPA usable until the access token expires. Agent and organization credentials remain independent non-browser credentials.

[docs/openapi.json](./openapi.json) is generated from the route registrations in `internal/http/server.go` (`make openapi`, checked in CI by `make check-openapi`) and is an accurate machine-readable method/path/operationId/bearer-auth inventory. Request/response body schemas are derived from the actual Go DTO struct each handler decodes/writes, resolved through `internal/openapi`'s `go/ast`-based analysis of `internal/http`; a route whose handler does not match one of the standard decode/write patterns (raw MCP JSON-RPC passthrough or `healthz`) gets a generic `{"type": "object"}` (or empty) placeholder rather than a guess. Response status codes are derived from the handler bodies as well: each operation lists the success statuses its handler actually writes (200/201/202/204, or a redirect), the 4xx statuses it can produce, and a `default` error response. All error responses reference the shared `ErrorResponse` component schema (`{error, code}`, with `code` enumerating the ten error codes). This document remains the source for prose per-route request/response descriptions where the generated schema is generic. The same document is browsable at `/docs/openapi.html` on the deployed GitHub Pages site, and served raw at `/docs/openapi.json`.

## Credential Coverage

Three bearer credential kinds reach the API. Their coverage differs:

| Credential | Prefix | Covers |
| --- | --- | --- |
| User access token | JWT | Every route in this document. User sessions carry no scope model. |
| Organization credential | `scrop_org_` | Routes widened to org parity: task listing/state changes/detail-adjacent flows that accept an org actor, reservation review, submission listing (`GET /api/tasks/{task_id}/submissions`, `submissions_read`) and submission review (accept/request-changes/reject, `submissions_review`) on the organization's own tasks, webhook management, the event feed (`GET /api/events`, `notifications_read`), MCP. Each call is checked against the scopes the credential was minted with. `scope=organization` task listings only; the org credential acts as the organization, not as a member. An org credential cannot pay tips or ban implementors on a review (those name a user), and its reviews record the system actor on the emitted events. |
| Personal agent credential | `scrop_agent_` | `GET /api/tasks` (public scope only, `tasks_read`), `GET /api/tasks/{task_id}` (`tasks_read`), `POST /api/tasks/{task_id}/submissions` (`submissions_write`), `POST /api/tasks/{task_id}/reservations` (`submissions_write`), `GET /api/events` (`notifications_read`), and the MCP endpoint (all tools, per scope). Other REST routes reject it. A task-scoped credential (auto-issued on reservation) is additionally bound to its task. Work-seeking (reserving tasks, submitting to tasks the owner has not already reserved, and daily spending caps) is additionally gated by the credential's work policy — see the work-policy endpoint below. |

## Authentication

- `POST /api/auth/register`: create an account with `email`, `password`, and an optional `display_name` (absent or empty derives the display name from the email's local part).
- `POST /api/auth/login`: exchange email/password for an access token.
- `POST /api/auth/refresh`: rotate a refresh-token cookie and issue a new access token.
- Register, login, and refresh responses include `display_name` for user sessions; guest sessions report an empty `display_name`.
- Register, login, and refresh responses also include `email_verification_state` (`unverified` or `verified`) for user sessions, so the client can prompt "verify your email to receive your signup grant" without a profile read. A fresh registration is always `unverified`. Guest sessions report an empty value (guests have no email); the field is also empty when the enrichment read fails, so an empty value means "unknown", never "unverified".
- `GET /api/auth/shauth`: start Authorization Code Flow with PKCE against the configured Shauth issuer.
- `GET /api/auth/shauth/callback`: verify the Shauth response, retain the provider-signed session coordinates server-side, and establish the rotating Sharecrop refresh-token session.
- `POST /api/auth/logout`: revoke the current Sharecrop refresh-token family and return a provider-discovered RP-Initiated Logout URL. The browser navigates to that URL to end the shared Shauth session.
- `GET /auth/shauth/logout/complete`: ignore all request parameters and redirect only to Shauth's issuer-origin logout-completion endpoint so Shauth can consume its one-time correlation before returning to Sharecrop.
- `GET /auth/validation`: expose the authenticated Shauth username, email, role, and immutable Sharecrop release revision used by Shauth's browser acceptance checks.
- `POST /auth/shauth/logout`: end the Sharecrop and Shauth sessions from the validation page and navigate through the same global logout flow.
- `GET /api/auth/shauth/frontchannel-logout`: accept Shauth's issuer-bound Front-Channel Logout notification, revoke every local refresh-token family correlated by the provider session ID, and return an embeddable no-content document with an issuer-only `frame-ancestors` policy.
- `GET /api/auth/signed-out`: receive Shauth's final post-logout redirect, revoke any residual Sharecrop refresh-token family, clear the cookie, and show a static signed-out page without automatically starting a new login.
- `POST /api/auth/shauth/backchannel-logout`: accept a signed OpenID Connect `logout_token` from Shauth, validate its exact issuer, audience, signature, expiry, event, `iat`, `jti`, prohibited `nonce`, and either `sid` or `sub`, then atomically revoke the matching refresh-token families and record the token against replay.
- `POST /api/auth/guest`: create a guest browser session.

## Account

- `POST /api/account/email-verification`: issue an email-verification token through the configured delivery mode.
- `POST /api/auth/email-verification/confirm`: confirm an issued email-verification token. The response is `{"status": "verified"}`; it does not carry the new balance — read `GET /api/credits/balance` afterwards. The 100-credit signup grant lands inside the confirm transaction the first time the account becomes verified, exactly once: re-verifying with a freshly issued token succeeds again (idempotently) without granting again, while replaying an already-consumed token is rejected. Registration alone leaves a zero balance. An organization's signup grant is decided when the organization is created: a verified creator's new organization receives it, an unverified creator's organization gets an account with no grant (and no retroactive grant on later verification).
- `POST /api/auth/password-reset/request`: issue a password-reset token through the configured delivery mode.
- `POST /api/auth/password-reset/confirm`: reset a password with an issued reset token.
- `PATCH /api/account/password`: change the authenticated user's password.
- `GET /api/account/profile`: read the authenticated user's own profile: `id`, `email`, `display_name`, and `email_verification_state` (`unverified` or `verified`). The user directory (`GET /api/users`) does not expose other users' verification states.
- `PATCH /api/account/profile`: change the authenticated user's profile email.
- `PATCH /api/account/display-name`: replace the authenticated user's `display_name` (required, length-limited).
- `DELETE /api/account`: deactivate the authenticated account.
- `POST /api/privacy-requests`: create an audited privacy request. Accepted `kind` values are `data_export` and `sensitive_field_deletion`; the response includes `kind`, `status`, `requested_by`, timestamps, `export_json`, `resolution_note`, and `redacted_field_count`.
- `GET /api/privacy-requests`: list the authenticated user's privacy requests.
- `POST /api/moderation/reports`: report a task, submission, comment, user, organization, team, or collectible for moderation review. Requires `subject_kind`, `subject_id`, and a `reason` category (`spam`, `abuse`, `pii`, `policy`, `dispute`, or `other`); `details` is optional and length-limited. `dispute` with a `submission` subject is the structured path for a worker contesting a rejected submission review. Any authenticated user can report; triage is admin-only (see the admin routes below).

## Agent Credentials

- `POST /api/agent-credentials`: mint a personal, scoped agent credential. Accepts `label`, `scopes`, and an optional `expires_at` (RFC3339 timestamp; omit or send `""` for never-expiring). The response includes the plaintext secret exactly once. Every credential is minted `work_seeking_disabled`.
- `GET /api/agent-credentials`: list the authenticated user's own agent credentials (never returns secrets).
- `POST /api/agent-credentials/{credential_id}/revoke`: revoke one of the authenticated user's own agent credentials.

### Work Policy

Work-seeking is default-deny: a freshly minted credential cannot reserve tasks and cannot submit to a task its owner has not already reserved until the owner enables work-seeking with a budget. A worker call through a disabled credential answers 403 `permission_denied`. Spending (funding a task, tipping, sending credits over MCP) is asking for work rather than seeking it, so it is not gated by the work-seeking state — but a daily spend cap can only be configured on an enabled policy, and only capped spending is metered.

- `PUT /api/agent-credentials/{credential_id}/work-policy`: set the credential's work policy. Owner user session only; another user's session answers 404 `not_found` (the endpoint does not reveal whether the id exists), and a task-scoped worker credential answers 400 `invalid_argument` (it operates inside its reservation and never carries a policy of its own). Only an active credential qualifies (a revoked one answers 409). The request body:
  - `work_seeking` (required): `work_seeking_disabled` or `work_seeking_enabled` (`invalid_enum` otherwise). Disabling ignores every allowance field and clears the stored ones.
  - `max_tasks_per_day` (required when enabled, 1..10000): how many tasks the credential may take on per UTC calendar day. Reservations count, and so does a direct submission to a task the credential had not reserved.
  - `max_concurrent_reservations` (optional, 1..1000): cap on simultaneously active reservations established via the credential. Absent or 0 means uncapped.
  - `max_credits_per_day` (optional, positive): cap on credits spent via the credential per UTC day (task funding, tips on reviews, peer sends over MCP). Absent or 0 means uncapped.
  - `task_types` (optional, array of task types): restricts which task types the credential may reserve or submit to. Absent or empty allows every type; an unknown type answers `invalid_enum`.
  - `min_reward_credits` (optional, positive): reward floor — the credential is refused tasks whose credit reward is below it, so the agent does not burn its daily budget on work its human considers not worth taking. Absent or 0 means no floor.
  - `token_budget_tokens` and `token_budget_note` (optional): an **advisory** model-token budget. The server stores it and returns it; it is **never enforced** — the server does not meter model tokens. Agents read it to pace themselves. A note without a token count answers `invalid_argument`; the note is capped at 500 characters.
  - The response is the full credential object (below) with the stored policy.
- Credential responses (list, mint, revoke, and the work-policy `PUT`) carry:
  - `work_policy`: `{work_seeking, max_tasks_per_day, max_concurrent_reservations, max_credits_per_day, task_types, min_reward_credits, token_budget_tokens, token_budget_note}`. Absent allowances are reported as `0` / `[]` / `""` exactly as they are configured: 0 means unlimited/none, an empty `task_types` means every type. A disabled policy carries every allowance field as 0/empty.
  - `tasks_used_today`: daily-task-budget units consumed in the current UTC day.
  - `credits_spent_today`: credits charged against the daily spend cap in the current UTC day. Spending is metered only while a `max_credits_per_day` cap is configured; uncapped spending is not counted here.
  - `active_reservations`: reservations established via the credential that are still active.
- When a worker call would exceed an enabled budget (daily tasks, concurrent reservations, or the daily spend cap), the server answers 429 with the distinct `budget_exceeded` error code and a message naming the exhausted dimension. Daily windows are UTC calendar days and reset at 00:00 UTC; the concurrent-reservation cap frees up as reservations complete or are cancelled. 429 `rate_limited` (request-volume throttling) is a different condition with a different code.
- `POST /api/organizations/{organization_id}/credentials`: mint an organization-wide credential with full org-admin parity. Requires `PermissionManageMembers` on the organization. Same `label`/`scopes`/`expires_at` shape as personal credentials.
- `GET /api/organizations/{organization_id}/credentials`: list an organization's own org-wide credentials.
- `POST /api/organizations/{organization_id}/credentials/{credential_id}/revoke`: revoke an organization-wide credential.
- Reserving a task also auto-issues a task-scoped agent credential for the reserving user (reservations are active immediately; there is no approval step); the plaintext secret is returned exactly once in that reservation response's `issued_worker_credential` field.

## Tasks

- `POST /api/tasks`: create a draft task. The request may include
  `attachments`, a list of `{name, content_type, data_url}` entries. Allowed
  content types are PNG, JPEG, GIF, WebP, plain text, JSON, and PDF. Each decoded
  file must be under 500 KiB, and each request may include up to five
  attachments.
- `GET /api/tasks`: list tasks visible to the requester. `scope` is optional and defaults to `public`; the other values are `user`, `organization` (requires `organization_id`), and `team` (requires `team_id`). Filters: `state` (repeatable), `participation_policy`, `task_type`, `query`, `sort`, `created_after` (RFC3339; only tasks created strictly after it), `funded` (`reward_funded`, `reward_unfunded`, or `no_credit_reward`; absent applies no funding restriction), `include_reserved`, `limit`, `offset`. List items carry `creator_display_name` on every row, `holder_display_name` (the user holding the active reservation when the reservation is user-assigned; empty otherwise), `funded` (the same three-value funded state), and `pending_review_count` (submissions in state `submitted`), which is populated only on tasks the caller created and 0 on every other row. The response carries `total` alongside `next_offset`. Personal agent credentials holding `tasks_read` may call this endpoint for the public scope only; any other scope answers 403 `permission_denied`.
- `GET /api/tasks/{task_id}`: read task detail, including `creator_display_name`.
- `POST /api/tasks/{task_id}/open`: open a draft task.
- `POST /api/tasks/{task_id}/cancel`: cancel an unfunded draft or open no-reward task.
- `POST /api/tasks/{task_id}/unpublish`: move an open task back to draft.
- `POST /api/tasks/{task_id}/funding`: fund a task from a user or organization balance.
- `POST /api/tasks/{task_id}/refund`: return a credit or bundle task's allocated credits to the funder's spendable balance and cancel the task. Allowed for the task owner or the holder of the active reservation while the task is not yet awarded.
- `POST /api/tasks/{task_id}/collectible-reward`: fund a task with a collectible.
- `POST /api/tasks/{task_id}/collectible-refund`: return a task's held collectible reward to the funder and cancel the task.

## Reservations And Submissions

- `POST /api/tasks/{task_id}/reservations`: reserve a task. Reservations are active immediately for user and organization-team assignees alike — there is no approval gate, and the worker proceeds straight to submitting. The historical `requested` and `declined` reservation states remain only on rows written before the approval flow was removed.
- `GET /api/tasks/{task_id}/reservations`: list reservations for a task. Each reservation carries `holder_display_name`, the requesting worker's display name.
- `POST /api/tasks/{task_id}/reservations/{reservation_id}/cancel`: cancel a reservation as requester.
- `POST /api/tasks/{task_id}/submissions`: submit a JSON response. The request
  may include the same small `attachments` shape and limits as task creation.
- `GET /api/tasks/{task_id}/submissions`: list task submissions for an authorized reviewer — a user session, or an organization credential holding `submissions_read` on the organization's own tasks. Each submission carries `submitter_display_name`. Submission `state` values are `submitted`, `invalid`, `accepted`, `rejected`, `changes_requested`, and `superseded` (terminal: another submission's accept closed the task while this one was still awaiting review).
- `GET /api/users/{user_id}/submissions`: list the authenticated user's own submissions. Supports `limit` and `offset`.
- `GET /api/users`: search the user directory with `query`, `limit`, and `offset`; returns id/email/display_name/status entries for selector-backed flows.
- `GET /api/users/{user_id}`: read a user's profile: their `display_name` and the tasks they created that are visible to the caller.
- `GET /api/users/{user_id}/work`: list the tasks currently assigned to a user that are visible to the caller.
- `GET /api/submission-receipts/{receipt_token}`: read receipt status by receipt token.
- `POST /api/tasks/{task_id}/submissions/{submission_id}/accept`: accept a submission and settle reward/tips. Competing submissions still in state `submitted` move to the terminal `superseded` state, and each superseded submitter receives a `submission_superseded` event and notification.
- `POST /api/tasks/{task_id}/submissions/{submission_id}/request-changes`: request changes and keep the task active.
- `POST /api/tasks/{task_id}/submissions/{submission_id}/reject`: reject a submission with a required note and optional partial/tip.

## Comments

- `GET /api/tasks/{task_id}/comments` and `POST /api/tasks/{task_id}/comments`: task thread.
- `GET /api/submissions/{submission_id}/comments` and `POST /api/submissions/{submission_id}/comments`: private submission thread for the submitter and authorized reviewer.
- `GET /api/task-series/{series_id}/comments` and `POST /api/task-series/{series_id}/comments`: series thread.
- Every comment response carries `author_display_name` alongside `author_user_id`.

## Events And Webhooks

The domain event stream is one wire shape shared by the live feed and webhook deliveries: `id`, `kind`, `actor_kind`, `actor_user_id`, `actor_display_name`, `occurred_at`, `cursor`, subject references (`task_id`, `task_title`, `submission_id`, `reservation_id`, `series_id`, `organization_id`, `collectible_id`), and `metadata_json`. `actor_display_name` and `task_title` are read-time enrichments carried by both the feed reads and webhook delivery bodies (the delivery claim resolves them in the same read); they are empty for system actors and for events without a task subject.

- `GET /api/events`: list the caller's visible events after an optional `after` cursor. Feed paging is cursor-based (`after` and `limit`); `offset` is not accepted. The response carries `next_cursor` to resume from. Callers are a user session, a personal agent credential holding `notifications_read` (the feed is the owning user's own event stream), or an organization credential holding `notifications_read` (the feed is the organization's own event stream: events whose subject organization is the credential's organization, the same rule an organization-owned recipient-audience webhook uses). The scope is `notifications_read` because the feed carries the same recipient-scoped facts the notification inbox is built from; a credential never sees more than its owner would.
- `GET /api/events?wait=N`: optional long poll for agents. `wait` is a whole number of seconds; when the page after `after` would be empty, the server holds the request until an event for the caller arrives or the wait elapses, then responds with the normal list shape (possibly empty). Waits above 25 seconds are capped at 25 (the API Gateway edge cuts requests at 30 seconds); malformed or negative values are rejected with `invalid_argument`. Under the WASI guest runtime the hold degrades to an immediate response with the same shape, so clients must treat an empty page as "poll again with the same `after`".
- `GET /api/events/stream`: the same feed as Server-Sent Events, for browser sessions only (agents and org credentials poll or long-poll `GET /api/events` instead). The `Last-Event-ID` header takes precedence over `?after` on reconnect. Connections end cleanly before the 30-second edge cap; clients reconnect with `Last-Event-ID`.
- `POST /api/webhook-subscriptions`: create a subscription with `url` (an absolute `https` URL; plain `http` receivers are rejected with `invalid_argument`), `kinds` (domain event kinds), and an optional `audience`:
  - `recipient` (the default): deliveries carry events addressed to the subscription owner.
  - `marketplace`: deliveries carry every public open `task_opened` event, regardless of recipients. Requires `kinds` to be exactly `["task_opened"]`. Optional narrowing filters: `filter_task_type` (one task type) and `filter_min_credit_reward` (a positive integer credit floor). The filter fields are rejected with `invalid_argument` on recipient subscriptions.
  The response returns the subscription plus the signing `secret` exactly once; listings never repeat it. Callers are a user session or an org credential holding `webhooks_manage`; an org credential must also hold the read scope matching every subscribed kind.
- `GET /api/webhook-subscriptions`: list the caller's (or, with `organization_id`, an organization's) subscriptions. Subscription responses include `audience`, `filter_task_type`, and `filter_min_credit_reward` (empty / 0 when unset).
- `DELETE /api/webhook-subscriptions/{subscription_id}`: revoke a subscription.
- `GET /api/webhook-subscriptions/{subscription_id}/deliveries`: list a subscription's delivery attempts (`event_cursor`, `state`, `attempt_count`, `next_attempt_at`, `last_status`).

### Signature Verification

Every delivery is signed with the subscription's secret (`scrop_whsec_…`). Headers:

- `Sharecrop-Webhook-Id`: the delivery id.
- `Sharecrop-Webhook-Timestamp`: the send instant as a unix-seconds integer.
- `Sharecrop-Webhook-Signature`: `v1=` + hex(HMAC-SHA256(secret, `<timestamp>.<raw body>`)).

Verify by recomputing the HMAC over the timestamp header, a literal `.`, and the raw request body, then comparing in constant time:

```python
import hashlib, hmac

expected = "v1=" + hmac.new(secret.encode(), f"{timestamp}.".encode() + raw_body, hashlib.sha256).hexdigest()
valid = hmac.compare_digest(expected, signature_header)
```

Reject deliveries whose timestamp is far from your clock to bound replay, and deduplicate by `Sharecrop-Webhook-Id` (retries can legitimately repeat an id).

A runnable reference receiver implementing this recipe — signature check, five-minute timestamp-skew rejection, and id dedupe — is [tools/webhook_receiver_sample.ts](../tools/webhook_receiver_sample.ts):

```
SHARECROP_WEBHOOK_SECRET=scrop_whsec_... deno run --allow-net --allow-env tools/webhook_receiver_sample.ts
```

## Task Series

- `GET /api/task-series` and `POST /api/task-series`: list/create task series.
- `GET /api/task-series/{series_id}`: read series detail.
- `PATCH /api/task-series/{series_id}`: update series title and description.
- `POST /api/task-series/{series_id}/publish`: publish a draft series.
- `POST /api/task-series/{series_id}/unpublish`: move a published series back to draft.
- `POST /api/task-series/{series_id}/close`: close a series.
- `POST /api/task-series/{series_id}/reopen`: reopen a closed series.
- `POST /api/task-series/{series_id}/tasks`: add a task to a series.
- `DELETE /api/task-series/{series_id}/tasks/{task_id}`: remove a task from a series.
- `POST /api/task-series/{series_id}/reorder`: reorder member tasks.

## Organizations And Teams

- `GET /api/organizations` and `POST /api/organizations`: list/create organizations.
- `GET /api/organizations/{organization_id}/members` and `POST /api/organizations/{organization_id}/members`: list/provision organization members.
- `PATCH /api/organizations/{organization_id}/members/{user_id}/roles`: update organization roles.
- `PATCH /api/organizations/{organization_id}/members/{user_id}/deactivate`: deactivate an organization member.
- `GET /api/organizations/{organization_id}/teams` and `POST /api/organizations/{organization_id}/teams`: list/create organization teams.
- `GET /api/teams` and `POST /api/teams`: list/create standalone teams.
- `GET /api/teams/{team_id}`: team detail.
- `POST /api/teams/{team_id}/members`: add a standalone-team member by email.
- `GET /api/teams/{team_id}/work`: list tasks visible or assigned to the team.
- `GET /api/saved-queue-views`: list authenticated-user saved queue views. Supports optional `scope`.
- `POST /api/saved-queue-views`: upsert an authenticated-user saved queue view. Accepted scopes are `team_work` and `organization_tasks`.

## Collectibles, Ledger, Notifications, Admin

- `GET /api/credits/balance`: read the authenticated user's credit balance. The account has two sections: `spendable_credits` (credits that can be spent or used to fund tasks) and `allocated_credits` (credits currently locked to funded tasks).
- `GET /api/credits/ledger`: list authenticated-user ledger entries, newest first. Each entry carries `id`, `kind`, `amount`, `task_id` (empty when the entry is not tied to a task), and `note` (the stored note, for example the required explanation on a platform-admin credit grant or the message on a peer credit send; empty for entry kinds without one).
- `POST /api/credits/transfers`: send credits to another account as a `peer_transfer` double entry. Request: `source_kind` (`self`, or `organization` with `source_organization_id` — the caller needs the organization's billing permission), `target_kind` (`user` or `organization`), `target_id`, `amount` (positive), an optional `note` (stored on both ledger rows), and `idempotency_key`. The response returns the sender-side `entry_id` and `amount`; a replayed key returns the original entry without moving credits again. Self-sends and organization-to-organization sends are rejected with `invalid_argument`, as is an amount above the spendable balance. Every sending account — user sessions included — is limited to 500 credits of peer transfers per UTC calendar day (an anti-abuse velocity ceiling, not an agent budget); crossing it answers 429 `budget_exceeded` and resets at 00:00 UTC. Platform-admin grants use a different path and are exempt. The receiver (or the receiving organization's owner/admin/billing members) gets a `credits_received` notification.
- `GET /api/organizations/{organization_id}/credits/balance`: read the organization credit balance, with the same `spendable_credits` and `allocated_credits` sections.
- `GET /api/organizations/{organization_id}/credits/ledger`: list organization ledger entries. Requires billing permission on the organization.
- `GET /api/organizations/{organization_id}/audit-events`: list audit events whose subject is the organization. Requires membership-management permission on the organization. Supports `limit` and `offset`.
- `GET /api/collectibles`: list authenticated-user collectible holdings. Collectible responses carry provenance: `catalog_slug` (the catalog entry a catalog-awarded instance came from; empty for custom mints), `edition_number` (the mint sequence number for edition instances; 0 when unnumbered), `issuer_display_name` (the minting or awarding user, resolved on list reads), and `owner_display_name` (the current owner's display label — user display name, organization name, or team name per `owner_kind`, resolved on list reads; empty on mutation responses). The `state` value `withdrawn` marks instances an admin removed from circulation.
- `POST /api/collectibles`: mint a collectible owned by the authenticated user. `transfer_policy` is optional and defaults to freely tradeable (`transferable_between_users`).
- `GET /api/collectibles/catalog`: list every collectible catalog entry with its lifecycle `state` (`available` or `withdrawn`), `max_editions` (1 for uniques, the run size for editions, 0 for uncapped badges), `minted_count` (live, non-withdrawn instances), `live_owner_count` (distinct owners of the live instances), and `owner_display_name` (for `unique` entries, the display label of the live instance's holder; empty for non-unique entries and unminted slots). Withdrawn entries stay listed but can no longer be awarded.
- `POST /api/collectibles/award`: platform-admin award of a catalog collectible to a user, team, or organization. Unique entries allow one live instance; edition entries are numbered against `max_editions` and refuse awards past the cap; withdrawn entries refuse awards.
- `POST /api/collectibles/{collectible_id}/transfer`: transfer a collectible. `target_kind` is optional: `user` (the default) moves it to another user; `organization` donates it to an organization's trophy case. `recipient_id` is the matching user or organization id. The transfer policy is enforced in the store transaction.
- `GET /api/organizations/{id}/collectibles`: list organization collectible holdings.
- `POST /api/organizations/{organization_id}/collectibles/{id}/award`: award one of the organization's held collectibles to an active member (`recipient_id`). Requires the `manage_collectibles` permission on the organization.
- `POST /api/organizations/{organization_id}/collectibles/{collectible_id}/transfer`: send one of the organization's held collectibles to any user (`recipient_id`), member or not. The acting member's `manage_collectibles` permission is verified inside the transfer transaction.
- `GET /api/teams/{id}/collectibles`: list team collectible holdings.
- `GET /api/notifications`: list authenticated-user notifications. Supports `state=unread`, `limit`, and `offset`. Each notification carries `actor_display_name` (empty for system actors) and `subject_title` (the subject task's title when the subject is a task, or the submission's task title when the subject is a submission; empty otherwise).
- `POST /api/notifications/{notification_id}/read`: mark a notification read.
- `GET /api/admin/operations`: platform-admin runtime status.
- `GET /api/admin/operations/counters`: platform-admin operations counters: `outbox_recorded_backlog` and `outbox_dispatch_failed` (domain events awaiting dispatch / retired after exhausting attempts), `webhook_deliveries_pending` and `webhook_deliveries_dead`, `oldest_pending_webhook_age_seconds` (0 means no pending delivery; a pending delivery younger than one second also reports 0), and the current UTC day's totals: `signup_grants_today`, `peer_transfers_today`, `peer_transfer_credits_today`, and `budget_refusals_today` (refused work-budget consumptions across all dimensions). The read model aggregates directly over the host's database; on runtimes without one (the in-memory dev default) the endpoint answers 503 `unavailable`. Under WASI hosting the host serves this route natively.
- `GET /api/admin/platform-admins`: platform-admin configuration list.
- `POST /api/admin/platform-admins`: grant platform-admin access to a user with `user_id`.
- `POST /api/admin/platform-admins/{user_id}/revoke`: revoke a granted platform-admin role by lifecycle state.
- `GET /api/admin/audit-events`: platform-admin audit event list. Supports `action`, `subject_kind`, `subject_id`, `limit`, and `offset`.
- `POST /api/admin/credits/grants`: platform-admin manual credit grant. Request: `target_kind` (`user` or `organization`), `target_id`, `amount` (positive), `note` (required explanation, stored on the ledger entry), `idempotency_key`. The response returns `entry_id` and `amount`. Replaying the same idempotency key returns the original entry without double-crediting; a non-admin caller receives 403 `permission_denied`. The beneficiaries (the granted user, or the organization's owner/admin/billing members) receive a `credit_granted` notification.
- `POST /api/admin/collectible-catalog`: platform-admin catalog entry creation. Request: `slug`, `name`, `kind` (`unique`, `edition`, or `badge`), `transfer_policy`, `art` (a sprite slug from the fixed art registry), and `max_editions` (required as 1 for `unique`, a positive run size for `edition`, and absent/0 for uncapped `badge`).
- `POST /api/admin/collectible-catalog/{slug}/withdraw`: mark a catalog entry no longer awardable. Existing instances are unaffected; withdrawing an already-withdrawn entry answers 409 `conflict`.
- `POST /api/admin/collectible-catalog/{slug}/release`: return a withdrawn catalog entry to `available` so it can be awarded again. An entry that is not withdrawn answers 409 `conflict`.
- `DELETE /api/admin/collectible-catalog/{slug}`: delete a withdrawn catalog entry that no instance references — live or withdrawn. A still-available entry or one with remaining instances (including withdrawn ones) answers 409 `conflict`; delete the withdrawn instances first.
- `POST /api/admin/collectibles/{collectible_id}/withdraw`: withdraw a catalog-minted instance from its holder (state `withdrawn`). The former holder receives a `collectible_withdrawn` notification. Escrowed instances (held as a task reward) are refused.
- `POST /api/admin/collectibles/{collectible_id}/release`: release a withdrawn instance back into its holder's inventory (state `minted`, owner unchanged). The holder receives a `collectible_released` notification. A non-withdrawn instance answers 409 `conflict`, as does releasing a `unique` whose live slot was re-minted while this instance was withdrawn.
- `DELETE /api/admin/collectibles/{collectible_id}`: hard-delete a withdrawn instance. Every other state answers 409 `conflict`.
- `GET /api/admin/moderation/reports`: platform-admin moderation report list. Supports `state`, `limit`, and `offset`.
- `POST /api/admin/moderation/reports/{report_id}/triage`: update moderation report `state` and `resolution_note`. Accepted states are `open`, `resolved`, and `dismissed`.
- `GET /api/admin/privacy-requests`: platform-admin privacy request queue.
- `POST /api/admin/privacy-requests/{privacy_request_id}/resolve`: resolve a queued privacy request with a `resolution_note`. Data-export requests store export JSON. Sensitive-field deletion requests mark delete-on-request sensitive-field metadata as redacted and record affected counts.
- `POST /api/admin/privacy-retention/run`: run delete-on-request sensitive-field retention and return the redacted-field count.

## Notes

- Pagination uses `limit` and `offset` where list handlers expose paging. `next_offset` is the offset of the next page, or 0 on the last page.
- The pager-backed list responses for tasks (including a user's work and a team's work), notifications, the user and organization credits ledgers, task submissions, a user's own submissions, webhook deliveries, and the admin moderation report list also carry `total`: the count of every row matching the filter, ignoring `limit`/`offset`. `total` is additive; `next_offset` semantics are unchanged.
- Idempotent mutations (task funding, submission accept/request-changes/reject, task refund, admin credit grants, peer credit sends) treat a replayed `idempotency_key` as a replay: the response returns the original result, and no duplicate domain events, notifications, or webhook deliveries are recorded.
- Error responses share one shape: `{"error": "<description>", "code": "<error code>"}` with the ten codes enumerated in the generated OpenAPI `ErrorResponse` component schema.
- Selector-backed browser flows use `query`, `limit`, and `offset` for users, organizations, standalone teams, and organization teams.
- Task list endpoints support `state`, `participation_policy`, `query`, `task_type`, `sort`, `created_after`, `funded`, `limit`, and `offset` where the corresponding scope is exposed. Sort values are `newest`, `oldest`, `title_asc`, `title_desc`, `reward_desc`, and `reward_asc`.
- The 16 task types (used by task creation, list filtering, marketplace webhook filters, and work-policy `task_types` restrictions) are: `general`, `code_review`, `security_review`, `product_review`, `ui_ux_review`, `qa_testing`, `document_review`, `documentation_writing`, `diagram_writing`, `planning`, `research`, `data_extraction`, `troubleshooting`, `code_analysis`, `architecture_review`, and `threat_analysis`.
- Submission responses include `sensitive_fields` metadata for indexed sensitive response paths. The metadata identifies path, category, retention, redaction policy, lifecycle state, and redaction time.
- Task detail and submission responses include `attachments` as
  `{name, content_type, size_bytes, data_url}`. Attachment bytes are stored
  inline for the small-file path; object storage is not implemented.
- Privacy requests are persisted. Requesters can list their own requests, and platform admins can list and resolve requests. Resolution stores basic export JSON for data-export requests or marks delete-on-request sensitive-field metadata as redacted. Platform admins can also run retention for active delete-on-request sensitive-field metadata. Core rows are not removed.
- Rewards are Sharecrop credits and admin-minted Sharecrop collectibles only. External wallets, crypto integrations, and per-project tokens are out of scope.
