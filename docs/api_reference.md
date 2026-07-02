# HTTP API Reference

This reference lists the stable application routes used by the Elm UI, external HTTP clients, and shared scenario tests.

All protected routes require `Authorization: Bearer <access_token>` unless the route is explicitly public. Browser sessions also use the refresh-token cookie for `/api/auth/refresh`.

[docs/openapi.json](./openapi.json) is generated from the route registrations in `internal/http/server.go` (`make openapi`, checked in CI by `make check-openapi`) and is an accurate machine-readable method/path/operationId/bearer-auth inventory. Its per-route request/response bodies are generic JSON object placeholders, not typed schemas; this document remains the source for prose per-route request/response descriptions.

## Authentication

- `POST /api/auth/register`: create an account with `email` and `password`.
- `POST /api/auth/login`: exchange email/password for an access token.
- `POST /api/auth/refresh`: rotate a refresh-token cookie and issue a new access token.
- `POST /api/auth/logout`: revoke the current refresh token.
- `POST /api/auth/guest`: create a guest browser session.

## Account

- `POST /api/account/email-verification`: issue an email-verification token through the configured delivery mode.
- `POST /api/auth/email-verification/confirm`: confirm an issued email-verification token.
- `POST /api/auth/password-reset/request`: issue a password-reset token through the configured delivery mode.
- `POST /api/auth/password-reset/confirm`: reset a password with an issued reset token.
- `PATCH /api/account/password`: change the authenticated user's password.
- `PATCH /api/account/profile`: change the authenticated user's profile email.
- `DELETE /api/account`: deactivate the authenticated account.
- `POST /api/privacy-requests`: create an audited privacy request. Accepted `kind` values are `data_export` and `sensitive_field_deletion`; the response includes `kind`, `status`, `requested_by`, timestamps, `export_json`, `resolution_note`, and `redacted_field_count`.
- `GET /api/privacy-requests`: list the authenticated user's privacy requests.

## Tasks

- `POST /api/tasks`: create a draft task. The request may include
  `attachments`, a list of `{name, content_type, data_url}` entries. Allowed
  content types are PNG, JPEG, GIF, WebP, plain text, JSON, and PDF. Each decoded
  file must be under 500 KiB, and each request may include up to five
  attachments.
- `GET /api/tasks`: list tasks visible to the requester. Supports pagination and task-state filters.
- `GET /api/tasks/{task_id}`: read task detail.
- `POST /api/tasks/{task_id}/open`: open a draft task.
- `POST /api/tasks/{task_id}/cancel`: cancel an unfunded draft or open no-reward task.
- `POST /api/tasks/{task_id}/unpublish`: move an open task back to draft.
- `POST /api/tasks/{task_id}/funding`: fund a task from a user or organization balance.
- `POST /api/tasks/{task_id}/refund`: refund held credit or bundle escrow.
- `POST /api/tasks/{task_id}/collectible-reward`: fund a task with a collectible.
- `POST /api/tasks/{task_id}/collectible-refund`: refund held collectible escrow.
- `POST /api/tasks/{task_id}/capability-tokens`: create a scoped task capability token.

## Reservations And Submissions

- `POST /api/tasks/{task_id}/reservations`: reserve a user-assignee task or reserve/request approval for an organization-team assignee task.
- `GET /api/tasks/{task_id}/reservations`: list reservations for a task.
- `POST /api/tasks/{task_id}/reservations/{reservation_id}/approve`: approve a requested reservation.
- `POST /api/tasks/{task_id}/reservations/{reservation_id}/decline`: decline a requested reservation.
- `POST /api/tasks/{task_id}/reservations/{reservation_id}/cancel`: cancel a reservation as requester.
- `POST /api/tasks/{task_id}/submissions`: submit a JSON response. The request
  may include the same small `attachments` shape and limits as task creation.
- `GET /api/tasks/{task_id}/submissions`: list task submissions for an authorized reviewer.
- `GET /api/users/{user_id}/submissions`: list the authenticated user's own submissions. Supports `limit` and `offset`.
- `GET /api/submission-receipts/{receipt_token}`: read receipt status by receipt token.
- `POST /api/tasks/{task_id}/submissions/{submission_id}/accept`: accept a submission and settle reward/tips.
- `POST /api/tasks/{task_id}/submissions/{submission_id}/request-changes`: request changes and keep the task active.
- `POST /api/tasks/{task_id}/submissions/{submission_id}/reject`: reject a submission with a required note and optional partial/tip.

## Comments

- `GET /api/tasks/{task_id}/comments` and `POST /api/tasks/{task_id}/comments`: task thread.
- `GET /api/submissions/{submission_id}/comments` and `POST /api/submissions/{submission_id}/comments`: private submission thread for the submitter and authorized reviewer.
- `GET /api/task-series/{series_id}/comments` and `POST /api/task-series/{series_id}/comments`: series thread.

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

- `GET /api/credits/balance`: read authenticated-user credit balance.
- `GET /api/credits/ledger`: list authenticated-user ledger entries.
- `GET /api/organizations/{organization_id}/credits/balance`: read organization credit balance.
- `GET /api/collectibles`: list authenticated-user collectible holdings.
- `POST /api/collectibles`: mint a collectible owned by the authenticated user.
- `GET /api/collectibles/catalog`: list platform default collectibles.
- `POST /api/collectibles/award`: platform-admin award of a catalog collectible to a user, team, or organization.
- `POST /api/collectibles/{collectible_id}/transfer`: transfer a collectible to another user.
- `GET /api/organizations/{organization_id}/collectibles`: list organization collectible holdings.
- `GET /api/teams/{team_id}/collectibles`: list team collectible holdings.
- `GET /api/notifications`: list authenticated-user notifications.
- `POST /api/notifications/{notification_id}/read`: mark a notification read.
- `GET /api/admin/operations`: platform-admin runtime status.
- `GET /api/admin/platform-admins`: platform-admin configuration list.
- `POST /api/admin/platform-admins`: grant platform-admin access to a user with `user_id`.
- `POST /api/admin/platform-admins/{user_id}/revoke`: revoke a granted platform-admin role by lifecycle state.
- `GET /api/admin/audit-events`: platform-admin audit event list. Supports `action`, `subject_kind`, `subject_id`, `limit`, and `offset`.
- `GET /api/admin/moderation/reports`: platform-admin moderation report list. Supports `state`, `limit`, and `offset`.
- `POST /api/admin/moderation/reports/{report_id}/triage`: update moderation report `state` and `resolution_note`. Accepted states are `open`, `resolved`, and `dismissed`.
- `GET /api/admin/privacy-requests`: platform-admin privacy request queue.
- `POST /api/admin/privacy-requests/{privacy_request_id}/resolve`: resolve a queued privacy request with a `resolution_note`. Data-export requests store export JSON. Sensitive-field deletion requests mark delete-on-request sensitive-field metadata as redacted and record affected counts.
- `POST /api/admin/privacy-retention/run`: run delete-on-request sensitive-field retention and return the redacted-field count.

## Notes

- Pagination uses `limit` and `offset` where list handlers expose paging.
- Selector-backed browser flows use `query`, `limit`, and `offset` for users, organizations, standalone teams, and organization teams.
- Task list endpoints support `state`, `participation_policy`, `query`, `task_type`, `sort`, `limit`, and `offset` where the corresponding scope is exposed. Sort values are `newest`, `oldest`, `title_asc`, `title_desc`, `reward_desc`, and `reward_asc`.
- Submission responses include `sensitive_fields` metadata for indexed sensitive response paths. The metadata identifies path, category, retention, redaction policy, lifecycle state, and redaction time.
- Task detail and submission responses include `attachments` as
  `{name, content_type, size_bytes, data_url}`. Attachment bytes are stored
  inline for the small-file path; object storage is not implemented.
- Privacy requests are persisted. Requesters can list their own requests, and platform admins can list and resolve requests. Resolution stores basic export JSON for data-export requests or marks delete-on-request sensitive-field metadata as redacted. Platform admins can also run retention for active delete-on-request sensitive-field metadata. Core rows are not removed.
- Rewards are Sharecrop credits and admin-minted Sharecrop collectibles only. External wallets, crypto integrations, and per-project tokens are out of scope.
