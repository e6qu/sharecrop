# Do Next

Prioritized queue:

1. Open the requester ergonomics and task page instructions pull request from `task/requester-ergonomics-instructions`, then wait for it to merge before starting the next task branch.
2. Then add review outcomes:
   - Request changes with required notes.
   - Reject with or without partial reward.
   - Accept with partial/full reward and optional tips from current balance/inventory.
   - Ban an implementor from the same task.
3. Then replace single-kind reward handling with reward bundles containing credits, collectibles, both, or neither.
4. Then add MCP workflow tools and full Streamable HTTP SSE, including `GET /mcp`, `DELETE /mcp`, session enforcement, event IDs, and replay where practical.
5. Later: revisit anonymous workers, user/org-issued tokens, crypto reward metadata, and broader contract fixture coverage.

Before starting, reread [AGENTS.md](./AGENTS.md) and update the continuity files if task scope changes.
