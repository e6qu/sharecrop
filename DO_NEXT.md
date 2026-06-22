# Do Next

Prioritized queue:

1. Open the reservation, approval, and discovery availability foundation pull request from `task/reservation-approval-foundation`, then wait for it to merge before starting the next task branch.
2. Next implementation after that merge: improve requester ergonomics and task page instructions:
   - Replace manual task-ID entry in funding and collectible-award forms with task selection from the user's task list.
   - Add participation-policy, assignee-scope, and reservation-expiry controls to task creation.
   - Add reservation/approval panels and worker reserve/apply/submit actions.
   - Show REST and MCP instructions on each task page.
3. Then add review outcomes:
   - Request changes with required notes.
   - Reject with or without partial reward.
   - Accept with partial/full reward and optional tips from current balance/inventory.
   - Ban an implementor from the same task.
4. Then replace single-kind reward handling with reward bundles containing credits, collectibles, both, or neither.
5. Then add MCP workflow tools and full Streamable HTTP SSE, including `GET /mcp`, `DELETE /mcp`, session enforcement, event IDs, and replay where practical.
6. Later: revisit anonymous workers, user/org-issued tokens, crypto reward metadata, and broader contract fixture coverage.

Before starting, reread [AGENTS.md](./AGENTS.md) and update the continuity files if task scope changes.
