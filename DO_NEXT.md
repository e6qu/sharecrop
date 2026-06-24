# Do Next

Prioritized queue:

1. Security follow-ups (see [BUGS.md](./BUGS.md)): add per-subject/per-IP rate limiting (HTTP 429) on MCP tool calls and the unauthenticated receipt/login/refresh endpoints; add a derived `idempotency_key` to the two `task_tip` ledger inserts for consistency with payout/refund entries.
3. Revisit collectible or inventory-based tips.
5. Redesign anonymous worker identity and payout.
6. Add user-issued or organization-issued tokens.
7. Add crypto reward metadata.
8. Move MCP HTTP sessions and SSE replay buffers out of process if multi-process deployment becomes a requirement. The in-memory store evicts idle sessions after a TTL but is still per-process.
9. Expand request and command contracts and HTTP contract fixture coverage.

Before starting, reread [AGENTS.md](./AGENTS.md) and update the continuity files if task scope changes.
