# Do Next

Prioritized queue:

1. Pull request 14: implement reservation, approval, and discovery availability foundations:
   - Participation policies: open submissions, reservation required, requester approval required.
   - Assignee scopes: one user or one same-organization team. Public-team support remains deferred until public teams exist.
   - Reservation states and 48-hour default expiry with automatic release.
   - One active assignee per task.
   - Task-local implementor bans.
   - Discovery availability and include-reserved filtering.
   - HTTP APIs and tests for reserve, request approval, approve, decline, cancel, list reservations, and reservation-gated submission.
2. Pull request 15: improve requester ergonomics and task page instructions:
   - Replace manual task-ID entry in funding and collectible-award forms with task selection from the user's task list.
   - Add participation-policy, assignee-scope, and reservation-expiry controls to task creation.
   - Add reservation/approval panels and worker reserve/apply/submit actions.
   - Show REST and MCP instructions on each task page.
3. Pull request 16: add review outcomes:
   - Request changes with required notes.
   - Reject with or without partial reward.
   - Accept with partial/full reward and optional tips from current balance/inventory.
   - Ban an implementor from the same task.
4. Pull request 17: replace single-kind reward handling with reward bundles containing credits, collectibles, both, or neither.
5. Pull request 18: add MCP workflow tools and full Streamable HTTP SSE, including `GET /mcp`, `DELETE /mcp`, session enforcement, event IDs, and replay where practical.
6. Later: revisit anonymous workers, user/org-issued tokens, crypto reward metadata, and broader contract fixture coverage.

Before starting, reread [AGENTS.md](./AGENTS.md) and update the continuity files if task scope changes.
