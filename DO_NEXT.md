# Do Next

Prioritized queue:

1. Continue decomposing on the HTTP side: `server.go` is now mostly task, submission, and reservation handlers that could split into their own files (organization, team, funding, user, series, and credits handlers already have theirs). The Elm `Main.elm` split is complete (`Types`/`View`/`Api`).
2. Revisit collectible or inventory-based tips.
5. Redesign anonymous worker identity and payout.
6. Add user-issued or organization-issued tokens.
7. Add crypto reward metadata.
8. Move MCP HTTP sessions and SSE replay buffers out of process if multi-process deployment becomes a requirement. The in-memory store evicts idle sessions after a TTL but is still per-process.
9. Expand request and command contracts and HTTP contract fixture coverage.

Before starting, reread [AGENTS.md](./AGENTS.md) and update the continuity files if task scope changes.
