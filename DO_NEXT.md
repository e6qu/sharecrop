# Do Next

Prioritized queue:

1. Model organizations as real entities (deferred from the economy/orgs bundle to a focused PR): give users an `org` and tasks an `org_id` — a migration plus domain/RBAC rewrite — replacing the role-string approximation used for organization visibility and review scoping. Largest pending item; touches migrations, domain, contracts, and RBAC tests.
2. Decompose the `Main.elm` monolith (deferred from the same bundle): lift the interdependent `Model`/`Msg` types into a shared `Types` module first, then split view/update/command groups out without behavior change. On the HTTP side, `server.go` is now mostly task/submission/reservation handlers that could split next.
3. Smaller schema-designer follow-ups: array-length constraints, and normalizing field names to identifier-safe keys (the designer warns on duplicate/empty names but does not rewrite them).
4. Revisit collectible or inventory-based tips.
5. Redesign anonymous worker identity and payout.
6. Add user-issued or organization-issued tokens.
7. Add crypto reward metadata.
8. Move MCP HTTP sessions and SSE replay buffers out of process if multi-process deployment becomes a requirement. The in-memory store evicts idle sessions after a TTL but is still per-process.
9. Expand request and command contracts and HTTP contract fixture coverage.

Before starting, reread [AGENTS.md](./AGENTS.md) and update the continuity files if task scope changes.
