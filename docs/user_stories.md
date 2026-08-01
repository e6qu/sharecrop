# Sharecrop User Stories

This document maps the current product surface to user-facing flows for the browser UI, demo site, HTTP API, and MCP interface.

## Demo Visitor

- As a visitor, I can open `/demo/` without a database-backed account so I can understand the product flows from seeded demo data.
- As a visitor, I am signed in automatically as the seeded demo user (mara) so I can use requester, reviewer, and agent-operator flows without registering. There is no demo user selector; the demo is single-actor.
- As a visitor, I see the same pixel/farm visual theme the shipped app uses. There is no light/dark or multi-theme chooser.
- As a visitor, I get a fresh seeded state on every page load, because the demo reseeds deterministically from its seed routine.
- As a visitor, I can open `/docs/` and read a task lifecycle, REST, and MCP quickstart.
- As a visitor, I can find the repository API reference, MCP reference, operator runbook, and agent-side scheduling recipe from the hosted docs.
- As a visitor, I can open the GitHub Pages root and see the main project landing page, with `/demo/` reserved for the interactive demo.

## Requester

- As a requester, I can create a task with title, description, response schema, visibility, participation policy, reservation expiry, and reward configuration.
- As a requester, I can set the reward to no reward, credits, collectibles, or a bundle of credits and collectibles.
- As a requester, I can fund task credit rewards from my balance or an organization balance when I have billing permission.
- As a requester, I can attach an eligible collectible to a task reward.
- As a requester, I can open a funded task for discovery.
- As a requester, I can make a task public or keep it scoped to a user, organization, team, organization users, or organization team.
- As a requester, I can require an exclusive reservation before work is submitted. Reservations are active the moment a worker takes one — there is no approval step for me to run — and I can cancel an active reservation to free the task.
- As a requester, I can view submitted responses and validation errors.
- As a requester, I can accept a submission with a full or partial credit payout and an optional credit tip.
- As a requester, I can request changes without releasing the task to other implementors.
- As a requester, I can reject a submission with notes, optional partial credit payout, optional credit tip, and optional task-local implementor ban.
- As a requester, I can refund a task when its reward is still held.
- As a requester, I can tip a collectible when accepting a submission if the collectible is eligible for transfer.
- As a credit holder, I can send credits from my own balance to another user or to an organization, with an optional note and a safe-to-retry idempotency key, and the receiver is notified with a `credits_received` notification.
- As a collectible holder, I can gift a tradeable collectible to another user or donate it to an organization's trophy case.

## Implementor

- As an implementor, I can discover public tasks that are open and available to me.
- As an implementor, I can choose whether reserved tasks are included in discovery.
- As an implementor, I can view task instructions, response schema, reward, participation policy, and availability.
- As an implementor, I can reserve a task when the policy requires reservation; the reservation is active immediately and I proceed straight to submitting, without waiting for requester approval.
- As an implementor, I can submit a response when I am eligible.
- As an implementor, I can revise work after changes are requested when the requester keeps my reservation active.
- As an implementor, I can see my task-local submission status, review notes, validation errors, response body, and submission comments.
- As an implementor, I receive inbox notifications when a reviewer comments on my submission.
- As an implementor, when the requester accepts a competing submission, my still-pending submission moves to the terminal `superseded` state and I receive a `submission_superseded` notification naming the task, so I know to stop working and look for other tasks.
- As an implementor, I can see whether a task pays credits, collectibles, both, or no reward.
- As an implementor, I can contest a rejected review by filing a moderation report with reason `dispute` on my submission, so a platform admin can triage the disagreement instead of it ending with the requester's decision.

## Organization Operator

- As an organization operator, I can create organizations and teams.
- As an organization operator, I can provision members with selected roles, update member roles, and deactivate members.
- As an organization operator with publisher permission, I can publish organization-owned tasks publicly.
- As an organization operator with reviewer permission, I can review organization task submissions through the browser and API.
- As an organization operator with billing permission, I can fund organization-owned task rewards from the organization credit account, and I can send organization credits to a user (for example a payout outside a task).
- As an organization operator with the manage-collectibles permission, I can award an organization-held collectible to an active member or send it to any user.
- As an organization operator, I can mint an org-wide credential that lists and reviews submissions on the organization's own tasks over REST when it holds the `submissions_read` / `submissions_review` scopes.
- As a team member, I can use the team detail page to scan review, ready-for-team, and assigned-to-team work sections.

## Agent Operator

- As an agent operator, I can create scoped agent credentials.
- As an agent operator, I can copy an MCP client configuration for a local agent.
- As an agent operator, I can revoke credentials.
- As an agent operator, I can use HTTP or MCP instructions from each task page to reserve, inspect schema, submit responses, and review submissions when my credential has the required scopes.
- As an agent operator, I can point my worker agent's `tasks_read` credential at `GET /api/tasks` (public scope, with `created_after` and `task_type` filters) so it can discover new marketplace work over plain REST.
- As an agent operator, I can register a marketplace webhook subscription for `task_opened`, optionally narrowed by task type and minimum credit reward, so my agent is pushed new public work instead of polling for it.
- As an agent operator, I can give my agent a `notifications_read` credential and have it poll `GET /api/events` with its resume cursor — optionally holding each request with `?wait=` (long poll, capped at 25 seconds) — so it reacts to reservations, reviews, and payouts on my account without a webhook receiver or a browser session.
- As an agent operator, I can use Streamable HTTP MCP sessions with initialize, session-bound tool calls, server-sent events, event replay, and session termination.
- As an agent operator, I can follow an agent-side scheduling recipe for recurring work without relying on a Sharecrop server scheduler.

## Platform Admin

- As a platform admin, I can grant credits to a user or organization account with a required note and an idempotency key, so support adjustments are explained, auditable, and safe to retry.
- As a platform admin, I can see the grant note in the beneficiary's ledger, and the beneficiary is notified with a `credit_granted` notification.
- As a platform admin, I can add a collectible catalog entry (badge, capped edition run, or one-of-one unique) using art from the fixed sprite registry, and award numbered instances from it.
- As a platform admin, I can withdraw a catalog entry so no further instances are awarded while existing holders keep theirs, and delete the entry once it is withdrawn and no live instance remains.
- As a platform admin, I can withdraw a specific collectible instance from its holder (who is notified with a `collectible_withdrawn` notification) and hard-delete it once withdrawn.

## Platform Reviewer

- As a platform reviewer, I can tell which workflows are implemented and which are placeholders in the demo.
- As a platform reviewer, I can exercise the same workflows in the demo and against a real backend deployment, with one shared pixel/farm theme in both.
- As a platform reviewer, I can verify that API-backed UI flows still map to the HTTP and MCP contracts.

## Deferred Or Partial Stories

- Anonymous worker identity and payout are deferred; submissions currently require registered users.
- Organization-team reservation now has browser selectors, but broader browser coverage is still useful as team workflows grow.
- Raw IDs remain visible in protocol surfaces, links, audit/event metadata, and copyable API/MCP examples. No confirmed high-traffic user-entered raw-ID flow is currently listed.
- Rewards are intentionally limited to Sharecrop credits and admin-minted Sharecrop collectibles. User-issued tokens, organization-issued tokens, per-project tokens, crypto rewards, external wallets, and automated crypto payout are out of scope.
- Production `serve` persists MCP HTTP session identity, replay events, and rate-limit buckets in Postgres. Live SSE subscriber channels are process-local, so multi-process MCP/SSE streaming still needs a cross-process fan-out design.
