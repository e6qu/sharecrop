# Sharecrop User Stories

This document maps the current product surface to user-facing flows for the browser UI, demo site, HTTP API, and MCP interface.

## Demo Visitor

- As a visitor, I can open `/demo/` without a database-backed account so I can understand the product flows from seeded demo data.
- As a visitor, I can choose a demo user from a visible user selector so I can switch between requester, implementor, reviewer, and agent-operator perspectives.
- As a visitor, I can choose light mode, dark mode, and visual themes so I can evaluate the product tone.
- As a visitor, I can clear demo-local state from a visible widget so I can reset local choices and seeded workflow edits.
- As a visitor, I can open `/docs/` and read a task lifecycle, REST, and MCP quickstart.
- As a visitor, I can open the GitHub Pages root and see the main project landing page, with `/demo/` reserved for the interactive demo.

## Requester

- As a requester, I can create a task with title, description, response schema, visibility, participation policy, reservation expiry, and reward configuration.
- As a requester, I can set the reward to no reward, credits, collectibles, or a bundle of credits and collectibles.
- As a requester, I can fund task credit rewards from my balance or an organization balance when I have billing permission.
- As a requester, I can attach an eligible collectible to a task reward.
- As a requester, I can open a funded task for discovery.
- As a requester, I can make a task public or keep it scoped to a user, organization, team, organization users, or organization team.
- As a requester, I can require an exclusive reservation before work is submitted.
- As a requester, I can require approval before an implementor can work.
- As a requester, I can review reservation requests, approve one implementor, decline requests, or cancel an active reservation.
- As a requester, I can view submitted responses and validation errors.
- As a requester, I can accept a submission with a full or partial credit payout and an optional credit tip.
- As a requester, I can request changes without releasing the task to other implementors.
- As a requester, I can reject a submission with notes, optional partial credit payout, optional credit tip, and optional task-local implementor ban.
- As a requester, I can refund a task when its reward is still held.
- As a requester, I can tip a collectible when accepting a submission if the collectible is eligible for transfer.

## Implementor

- As an implementor, I can discover public tasks that are open and available to me.
- As an implementor, I can choose whether reserved tasks are included in discovery.
- As an implementor, I can view task instructions, response schema, reward, participation policy, and availability.
- As an implementor, I can reserve a task when the policy requires reservation.
- As an implementor, I can request approval when the policy requires requester approval.
- As an implementor, I can submit a response when I am eligible.
- As an implementor, I can revise work after changes are requested when the requester keeps my reservation active.
- As an implementor, I can see my task-local submission status, review notes, validation errors, response body, and submission comments.
- As an implementor, I can see whether a task pays credits, collectibles, both, or no reward.

## Organization Operator

- As an organization operator, I can create organizations and teams.
- As an organization operator, I can provision members with selected roles, update member roles, and deactivate members.
- As an organization operator with publisher permission, I can publish organization-owned tasks publicly.
- As an organization operator with reviewer permission, I can review organization task submissions through the browser and API.
- As an organization operator with billing permission, I can fund organization-owned task rewards from the organization credit account.

## Agent Operator

- As an agent operator, I can create scoped agent credentials.
- As an agent operator, I can copy an MCP client configuration for a local agent.
- As an agent operator, I can revoke credentials.
- As an agent operator, I can use HTTP or MCP instructions from each task page to reserve, inspect schema, submit responses, and review submissions when my credential has the required scopes.
- As an agent operator, I can use Streamable HTTP MCP sessions with initialize, session-bound tool calls, server-sent events, event replay, and session termination.

## Platform Reviewer

- As a platform reviewer, I can tell which workflows are implemented and which are placeholders in the demo.
- As a platform reviewer, I can compare the same workflow across corporate, rustic, blocky, and showcase themes in light and dark modes.
- As a platform reviewer, I can verify that API-backed UI flows still map to the HTTP and MCP contracts.

## Deferred Or Partial Stories

- Anonymous worker identity and payout are deferred; submissions currently require registered users.
- Organization-team reservation now has browser selectors, but full team submission and team-scoped worker dashboards still need more coverage.
- Raw IDs remain visible in protocol surfaces, links, audit/event metadata, and copyable API/MCP examples. No confirmed high-traffic user-entered raw-ID flow is currently listed.
- Rewards are intentionally limited to Sharecrop credits and admin-minted Sharecrop collectibles. User-issued tokens, organization-issued tokens, per-project tokens, crypto rewards, external wallets, and automated crypto payout are out of scope.
- MCP HTTP sessions and SSE replay buffers are in-memory and not shared across restarts or multiple app processes.
